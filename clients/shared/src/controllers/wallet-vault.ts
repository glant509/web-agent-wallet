type AnyRecord = Record<string, any>;
type WalletMode = "unlock" | "create" | "backup" | "choice" | "import" | "change-password" | "export-mnemonic" | "export-private-key" | "export-display";

interface WalletVaultControllerOptions {
  state: AnyRecord;
  vaultStore: { load: () => Promise<AnyRecord | null> };
  unlockVault: (vault: AnyRecord, password: string) => Promise<{ masterKey: CryptoKey; payload: AnyRecord }>;
  createVault: (password: string, mnemonic: string, keyring: AnyRecord) => Promise<{ vault: AnyRecord; masterKey: CryptoKey }>;
  deriveSeed: (mnemonic: string) => Promise<Uint8Array>;
  normalizeKeyring: (keyring: unknown) => AnyRecord;
  createDefaultKeyring: () => AnyRecord;
  generateMnemonic: () => Promise<string>;
  parseMnemonicWords: (text: string) => Promise<string[]>;
  isValidPassword: (password: string) => boolean;
  isUnlocked: () => boolean;
  startSession: () => void;
  refreshBiometrics: () => Promise<void>;
  renderBiometrics: () => void;
  resetBiometricChoice: () => void;
  shouldEnableBiometrics: () => boolean;
  biometricStatus: () => AnyRecord;
  enableBiometrics: (password: string) => Promise<void>;
  disableBiometrics: () => Promise<void>;
  resetSecretExport: () => void;
  exportMnemonic: (password: string) => Promise<void>;
  exportPrivateKey: (password: string) => Promise<void>;
  copyText: (text: string) => Promise<void>;
  renderVault: (errorMessage?: string) => void;
  onPendingAction: (action: string) => Promise<void> | void;
  document?: Document;
  window?: Window;
}

function requiredElement<T extends HTMLElement>(doc: Document, id: string): T {
  const element = doc.getElementById(id);
  if (!element) throw new Error(`missing wallet vault element: ${id}`);
  return element as T;
}

export function createWalletVaultController(options: WalletVaultControllerOptions) {
  const doc = options.document || document;
  const browserWindow = options.window || window;
  const modal = requiredElement(doc, "wallet-lock");
  const form = requiredElement<HTMLFormElement>(doc, "wallet-lock-form");
  const password = requiredElement<HTMLInputElement>(doc, "wallet-password");
  const passwordConfirm = requiredElement<HTMLInputElement>(doc, "wallet-password-confirm");
  const passwordConfirmField = requiredElement(doc, "wallet-password-confirm-field");
  const newPassword = requiredElement<HTMLInputElement>(doc, "wallet-new-password");
  const newPasswordField = requiredElement(doc, "wallet-new-password-field");
  const passwordLabel = requiredElement(doc, "wallet-password-label");
  const error = requiredElement(doc, "wallet-lock-error");
  const copy = requiredElement(doc, "wallet-lock-copy");
  const cancel = requiredElement<HTMLButtonElement>(doc, "wallet-lock-cancel");
  const submit = requiredElement<HTMLButtonElement>(doc, "wallet-lock-submit");
  const createStep = requiredElement(doc, "wallet-create-step");
  const choiceStep = requiredElement(doc, "wallet-choice-step");
  const generateChoice = requiredElement<HTMLButtonElement>(doc, "wallet-generate-choice");
  const importChoice = requiredElement<HTMLButtonElement>(doc, "wallet-import-choice");
  const recovery = requiredElement(doc, "wallet-recovery");
  const mnemonic = requiredElement(doc, "wallet-mnemonic");
  const recoveryConfirm = requiredElement<HTMLInputElement>(doc, "wallet-recovery-confirm");
  const recoveryConfirmRow = requiredElement(doc, "wallet-recovery-confirm-row");
  const recoveryCopy = requiredElement<HTMLButtonElement>(doc, "wallet-recovery-copy");
  const recoveryTitle = requiredElement(doc, "wallet-recovery-title");
  const recoveryDescription = requiredElement(doc, "wallet-recovery-description");
  const importStep = requiredElement(doc, "wallet-import-step");
  const importText = requiredElement<HTMLTextAreaElement>(doc, "wallet-import-text");
  const importFill = requiredElement<HTMLButtonElement>(doc, "wallet-import-fill");
  const importMnemonic = requiredElement(doc, "wallet-import-mnemonic");
  const importConfirm = requiredElement<HTMLInputElement>(doc, "wallet-import-confirm");
  const accessButton = requiredElement<HTMLButtonElement>(doc, "wallet-access-button");
  const changePasswordButton = requiredElement<HTMLButtonElement>(doc, "wallet-change-password-button");
  let recoveryCopyTimer = 0;

  function renderRecoveryPhrase(value: string): void {
    mnemonic.replaceChildren();
    value.split(" ").forEach((word, index) => {
      const item = doc.createElement("span");
      item.textContent = `${String(index + 1).padStart(2, "0")}. ${word}`;
      mnemonic.appendChild(item);
    });
  }

  function renderImportInputs(words: string[]): void {
    importMnemonic.replaceChildren();
    for (let index = 0; index < 12; index += 1) {
      const input = doc.createElement("input");
      input.type = "text";
      input.autocomplete = "off";
      input.autocapitalize = "off";
      input.spellcheck = false;
      input.placeholder = String(index + 1).padStart(2, "0");
      input.value = words[index] || "";
      importMnemonic.appendChild(input);
    }
  }

  function open(mode: WalletMode): void {
    options.state.wallet.mode = mode;
    modal.classList.add("visible");
    modal.setAttribute("aria-hidden", "false");
    error.textContent = "";
    password.value = "";
    passwordConfirm.value = "";
    newPassword.value = "";
    recoveryConfirm.checked = false;
    recoveryConfirmRow.hidden = false;
    recoveryTitle.textContent = "备份恢复短语";
    recoveryDescription.textContent = "请离线抄写这 12 个单词。它只显示这一次；确认完成后即可创建新钱包。";
    recovery.hidden = true;
    options.resetSecretExport();
    importStep.hidden = true;
    createStep.hidden = false;
    choiceStep.hidden = true;
    importConfirm.checked = false;
    importText.value = "";
    recoveryCopy.textContent = "copy";
    recoveryCopy.hidden = true;
    cancel.textContent = "取消";
    submit.hidden = false;
    mnemonic.replaceChildren();
    importMnemonic.replaceChildren();
    password.disabled = false;
    passwordConfirm.disabled = false;
    newPassword.disabled = false;
    options.resetBiometricChoice();

    const creating = mode === "create" || mode === "backup";
    const changingPassword = mode === "change-password";
    const exportingSecret = mode === "export-mnemonic" || mode === "export-private-key";
    passwordConfirmField.hidden = !creating && !changingPassword;
    newPasswordField.hidden = !changingPassword;
    passwordLabel.textContent = creating ? "设置钱包密码" : changingPassword ? "当前钱包密码" : "钱包密码";
    passwordConfirm.placeholder = changingPassword ? "再次输入新钱包密码" : "再次输入钱包密码";
    copy.textContent = exportingSecret
      ? "为保护资产，请再次验证当前钱包密码。验证成功后敏感密钥只在本设备临时显示。"
      : changingPassword
      ? "验证当前密码后，将使用新的随机盐和 IV 重新加密本地钱包保险库。"
      : creating
      ? "创建后，恢复短语只会在此设备显示一次。保险库仅保存 AES-GCM 加密数据到本机安全存储。"
      : "输入本地钱包密码以派生密钥并解密保险库。密码和恢复短语不会发送到服务器。";
    submit.textContent = exportingSecret ? "验证并显示" : changingPassword ? "确认更换" : creating ? "下一步" : "解锁";
    options.renderBiometrics();
    password.focus();
  }

  function close(clearPendingAction = true): void {
    browserWindow.clearTimeout(recoveryCopyTimer);
    recoveryCopyTimer = 0;
    options.resetSecretExport();
    modal.classList.remove("visible");
    modal.setAttribute("aria-hidden", "true");
    error.textContent = "";
    password.value = "";
    passwordConfirm.value = "";
    newPassword.value = "";
    password.disabled = false;
    passwordConfirm.disabled = false;
    newPassword.disabled = false;
    newPasswordField.hidden = true;
    options.resetBiometricChoice();
    recoveryCopy.textContent = "copy";
    recoveryCopy.hidden = true;
    recoveryConfirmRow.hidden = false;
    recoveryTitle.textContent = "备份恢复短语";
    recoveryDescription.textContent = "请离线抄写这 12 个单词。它只显示这一次；确认完成后即可创建新钱包。";
    cancel.textContent = "取消";
    submit.hidden = false;
    createStep.hidden = false;
    choiceStep.hidden = true;
    importStep.hidden = true;
    importConfirm.checked = false;
    importText.value = "";
    mnemonic.replaceChildren();
    recovery.hidden = true;
    importMnemonic.replaceChildren();
    options.state.wallet.pendingPassword = "";
    options.state.wallet.pendingMnemonic = null;
    if (clearPendingAction) options.state.wallet.pendingAction = "";
  }

  async function initialize(): Promise<void> {
    try {
      options.state.wallet.vault = await options.vaultStore.load();
      await options.refreshBiometrics();
      options.renderVault();
    } catch (caught) {
      options.renderVault(caught instanceof Error ? caught.message : "钱包保险库初始化失败");
    }
  }

  async function prepareAccess(): Promise<void> {
    options.state.wallet.pendingAction = "";
    if (options.isUnlocked()) {
      options.renderVault();
      return;
    }
    try {
      options.state.wallet.vault = await options.vaultStore.load();
      await options.refreshBiometrics();
    } catch (caught) {
      options.renderVault(caught instanceof Error ? caught.message : "无法读取本地钱包保险库");
      return;
    }
    open(options.state.wallet.vault ? "unlock" : "create");
  }

  async function unlock(passwordValue: string): Promise<void> {
    if (!options.state.wallet.vault) {
      error.textContent = "未找到本地钱包保险库。";
      return;
    }
    submit.disabled = true;
    try {
      const { masterKey, payload } = await options.unlockVault(options.state.wallet.vault, passwordValue);
      options.state.wallet.masterKey = masterKey;
      options.state.wallet.mnemonic = payload.mnemonic;
      options.state.wallet.seed = await options.deriveSeed(payload.mnemonic);
      options.state.wallet.addresses = {};
      options.state.wallet.keyring = options.normalizeKeyring(payload.keyring);
      let biometricWarning = "";
      const biometric = options.biometricStatus();
      if (options.shouldEnableBiometrics() && biometric.available && !biometric.enabled) {
        try {
          await options.enableBiometrics(passwordValue);
          await options.refreshBiometrics();
        } catch (caught) {
          biometricWarning = caught instanceof Error ? caught.message : "生物识别快捷解锁启用失败。";
        }
      }
      const pendingAction = options.state.wallet.pendingAction;
      close(false);
      options.startSession();
      options.renderVault();
      await options.onPendingAction(pendingAction);
      if (biometricWarning) browserWindow.alert(`钱包已解锁，但${biometricWarning}`);
    } catch {
      error.textContent = "无法解锁钱包，请检查密码。";
    } finally {
      submit.disabled = false;
      password.value = "";
    }
  }

  async function changePassword(currentPassword: string, nextPassword: string): Promise<void> {
    if (!options.state.wallet.vault) {
      error.textContent = "未找到本地钱包保险库。";
      return;
    }
    submit.disabled = true;
    try {
      const { payload } = await options.unlockVault(options.state.wallet.vault, currentPassword);
      const updated = await options.createVault(nextPassword, payload.mnemonic, payload.keyring);
      options.state.wallet.vault = updated.vault;
      options.state.wallet.masterKey = updated.masterKey;
      options.state.wallet.mnemonic = payload.mnemonic;
      options.state.wallet.seed = await options.deriveSeed(payload.mnemonic);
      options.state.wallet.addresses = {};
      options.state.wallet.keyring = options.normalizeKeyring(payload.keyring);
      await options.disableBiometrics().catch(() => undefined);
      await options.refreshBiometrics();
      close();
      options.startSession();
      options.renderVault();
      browserWindow.alert("钱包密码已更换。下次解锁请使用新密码。");
    } catch {
      error.textContent = "无法更换密码，请检查当前密码。";
    } finally {
      submit.disabled = false;
    }
  }

  async function finalizeCreation(mnemonicValue: string): Promise<void> {
    const pendingPassword = options.state.wallet.pendingPassword;
    if (!pendingPassword) throw new Error("钱包密码状态已失效，请重新开始创建。");
    submit.disabled = true;
    try {
      const keyring = options.createDefaultKeyring();
      const created = await options.createVault(pendingPassword, mnemonicValue, keyring);
      options.state.wallet.vault = created.vault;
      options.state.wallet.masterKey = created.masterKey;
      options.state.wallet.mnemonic = mnemonicValue;
      options.state.wallet.seed = await options.deriveSeed(mnemonicValue);
      options.state.wallet.addresses = {};
      options.state.wallet.keyring = keyring;
      options.state.wallet.pendingMnemonic = null;
      options.state.wallet.pendingPassword = "";
      close();
      options.startSession();
      options.renderVault();
    } finally {
      submit.disabled = false;
    }
  }

  async function readImportedMnemonic(): Promise<string> {
    const words = Array.from(importMnemonic.querySelectorAll("input"))
      .map((input) => input.value.trim().toLowerCase())
      .filter(Boolean);
    if (words.length !== 12) throw new Error("请完整填写 12 个助记词。");
    const normalized = await options.parseMnemonicWords(words.join(" "));
    options.state.wallet.pendingMnemonic = normalized.join(" ");
    return options.state.wallet.pendingMnemonic;
  }

  async function submitCurrentMode(): Promise<void> {
    const passwordValue = password.value;
    const mode = options.state.wallet.mode as WalletMode;
    if (["unlock", "export-mnemonic", "export-private-key"].includes(mode) && !options.isValidPassword(passwordValue)) {
      error.textContent = "密码必须是 8-16 位，并且仅包含字母、数字和 !@#$%^&*.";
      return;
    }
    if (mode === "unlock") return unlock(passwordValue);
    if (mode === "export-mnemonic") return options.exportMnemonic(passwordValue);
    if (mode === "export-private-key") return options.exportPrivateKey(passwordValue);
    if (mode === "export-display") return close();
    if (mode === "change-password") {
      const nextPassword = newPassword.value;
      if (!options.isValidPassword(passwordValue)) error.textContent = "当前密码格式不正确。";
      else if (!options.isValidPassword(nextPassword)) error.textContent = "新密码必须是 8-16 位，并且仅包含字母、数字和 !@#$%^&*.";
      else if (nextPassword === passwordValue) error.textContent = "新密码不能与当前密码相同。";
      else if (passwordConfirm.value !== nextPassword) error.textContent = "两次输入的新密码不一致。";
      else await changePassword(passwordValue, nextPassword);
      return;
    }
    if (mode === "create") {
      if (!options.isValidPassword(passwordValue)) error.textContent = "密码必须是 8-16 位，并且仅包含字母、数字和 !@#$%^&*.";
      else if (passwordConfirm.value !== passwordValue) error.textContent = "两次输入的钱包密码不一致。";
      else {
        options.state.wallet.pendingPassword = passwordValue;
        options.state.wallet.mode = "choice";
        password.value = "";
        passwordConfirm.value = "";
        error.textContent = "";
        createStep.hidden = true;
        choiceStep.hidden = false;
        submit.hidden = true;
        copy.textContent = "请选择生成新的助记词，或者导入已有的 12 个助记词。";
      }
      return;
    }
    if (mode === "backup") {
      if (!recoveryConfirm.checked || !options.state.wallet.pendingMnemonic) {
        error.textContent = "请确认已离线备份恢复短语。";
        return;
      }
      try { await finalizeCreation(options.state.wallet.pendingMnemonic); }
      catch (caught) { error.textContent = caught instanceof Error ? caught.message : "钱包保险库创建失败"; }
      return;
    }
    if (mode === "import") {
      if (!importConfirm.checked) {
        error.textContent = "请确认你理解恢复短语的保管风险。";
        return;
      }
      try { await finalizeCreation(await readImportedMnemonic()); }
      catch (caught) { error.textContent = caught instanceof Error ? caught.message : "导入助记词失败"; }
    }
  }

  async function startGeneratedMnemonic(): Promise<void> {
    try {
      options.state.wallet.pendingMnemonic = await options.generateMnemonic();
      renderRecoveryPhrase(options.state.wallet.pendingMnemonic);
      options.state.wallet.mode = "backup";
      choiceStep.hidden = true;
      recovery.hidden = false;
      importStep.hidden = true;
      recoveryCopy.hidden = false;
      submit.hidden = false;
      submit.textContent = "确认备份并创建";
      error.textContent = "";
      copy.textContent = "请立即离线备份这 12 个助记词。密码框已隐藏，接下来只需确认备份并创建钱包。";
    } catch (caught) {
      error.textContent = caught instanceof Error ? caught.message : "无法生成恢复短语";
    }
  }

  function startImportMnemonic(): void {
    options.state.wallet.mode = "import";
    options.state.wallet.pendingMnemonic = null;
    choiceStep.hidden = true;
    recovery.hidden = true;
    importStep.hidden = false;
    submit.hidden = false;
    submit.textContent = "确认导入并创建";
    error.textContent = "";
    copy.textContent = "粘贴或逐个填写已有的 12 个助记词。去掉首尾换行后会按空格切分并自动填充到下方输入框。";
    renderImportInputs(Array(12).fill(""));
    importText.focus();
  }

  async function hydrateImportedMnemonic(): Promise<void> {
    try {
      renderImportInputs(await options.parseMnemonicWords(importText.value));
      error.textContent = "";
    } catch (caught) {
      error.textContent = caught instanceof Error ? caught.message : "导入助记词失败";
    }
  }

  async function copyRecoveryPhrase(): Promise<void> {
    if (!options.state.wallet.pendingMnemonic) {
      error.textContent = "请先生成恢复短语。";
      return;
    }
    try {
      await options.copyText(options.state.wallet.pendingMnemonic.split(" ").join(" "));
      error.textContent = "";
      recoveryCopy.textContent = "copied";
      browserWindow.clearTimeout(recoveryCopyTimer);
      recoveryCopyTimer = browserWindow.setTimeout(() => { recoveryCopy.textContent = "copy"; }, 1600);
    } catch (caught) {
      error.textContent = caught instanceof Error ? caught.message : "复制恢复短语失败。";
    }
  }

  function bind(): void {
    form.addEventListener("submit", (event) => { event.preventDefault(); void submitCurrentMode(); });
    cancel.addEventListener("click", () => close());
    password.addEventListener("input", () => { error.textContent = ""; });
    password.addEventListener("keydown", (event) => {
      if (event.key === "Enter") { event.preventDefault(); void submitCurrentMode(); }
      if (event.key === "Escape") close();
    });
    accessButton.addEventListener("click", () => void prepareAccess());
    changePasswordButton.addEventListener("click", () => open("change-password"));
    generateChoice.addEventListener("click", () => void startGeneratedMnemonic());
    importChoice.addEventListener("click", startImportMnemonic);
    importFill.addEventListener("click", () => void hydrateImportedMnemonic());
    recoveryCopy.addEventListener("click", () => void copyRecoveryPhrase());
  }

  return Object.freeze({
    bind, changePassword, close, copyRecoveryPhrase, finalizeCreation, hydrateImportedMnemonic,
    initialize, open, prepareAccess, readImportedMnemonic, renderImportInputs, renderRecoveryPhrase,
    startGeneratedMnemonic, startImportMnemonic, submit: submitCurrentMode, unlock
  });
}
