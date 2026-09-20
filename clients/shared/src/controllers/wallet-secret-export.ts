type AnyRecord = Record<string, any>;

interface WalletSecretExportOptions {
  state: AnyRecord;
  openUnlock: (mode: "export-mnemonic" | "export-private-key") => void;
  closeUnlock: () => void;
  base64ToBytes: (value: string) => Uint8Array;
  deriveVaultKey: (password: string, salt: Uint8Array, iterations: number) => Promise<CryptoKey>;
  deriveSeed: (mnemonic: string) => Promise<Uint8Array>;
  normalizeKeyring: (keyring: unknown) => AnyRecord;
  getSelectedChain: () => { label: string };
  derivePrivateKey: (seed: Uint8Array, chainId: string, accountIndex: number) => Promise<AnyRecord>;
  deriveAddress: (seed: Uint8Array, chainId: string, accountIndex: number) => string;
  renderMnemonic: (mnemonic: string) => void;
  copyText: (value: string) => Promise<void>;
  document?: Document;
  window?: Window;
  revealTimeoutMs?: number;
}

function requiredElement<T extends HTMLElement>(doc: Document, id: string): T {
  const element = doc.getElementById(id);
  if (!element) throw new Error(`missing secret export element: ${id}`);
  return element as T;
}

export function createWalletSecretExportController(options: WalletSecretExportOptions) {
  const doc = options.document || document;
  const browserWindow = options.window || window;
  const revealTimeoutMs = options.revealTimeoutMs || 60000;
  const createStep = requiredElement(doc, "wallet-create-step");
  const choiceStep = requiredElement(doc, "wallet-choice-step");
  const importStep = requiredElement(doc, "wallet-import-step");
  const recovery = requiredElement(doc, "wallet-recovery");
  const recoveryConfirmRow = requiredElement(doc, "wallet-recovery-confirm-row");
  const recoveryCopy = requiredElement<HTMLButtonElement>(doc, "wallet-recovery-copy");
  const recoveryTitle = requiredElement(doc, "wallet-recovery-title");
  const recoveryDescription = requiredElement(doc, "wallet-recovery-description");
  const privateKeyExport = requiredElement(doc, "wallet-private-key-export");
  const privateKeyCopy = requiredElement<HTMLButtonElement>(doc, "wallet-private-key-copy");
  const privateKeyChain = requiredElement(doc, "wallet-private-key-chain");
  const privateKeyAddress = requiredElement(doc, "wallet-private-key-address");
  const privateKeyPath = requiredElement(doc, "wallet-private-key-path");
  const privateKeyFormat = requiredElement(doc, "wallet-private-key-format");
  const privateKeyValue = requiredElement(doc, "wallet-private-key-value");
  const lockCopy = requiredElement(doc, "wallet-lock-copy");
  const lockCancel = requiredElement<HTMLButtonElement>(doc, "wallet-lock-cancel");
  const lockSubmit = requiredElement<HTMLButtonElement>(doc, "wallet-lock-submit");
  const lockError = requiredElement(doc, "wallet-lock-error");
  const password = requiredElement<HTMLInputElement>(doc, "wallet-password");
  let revealTimer = 0;
  let copyTimer = 0;

  function requestMnemonic(): void {
    if (options.state.wallet.vault) options.openUnlock("export-mnemonic");
  }

  function requestPrivateKey(): void {
    if (options.state.wallet.vault) options.openUnlock("export-private-key");
  }

  async function decryptPayload(passwordValue: string): Promise<AnyRecord> {
    const vault = options.state.wallet.vault;
    if (!vault) throw new Error("未找到本地钱包保险库。");
    const salt = options.base64ToBytes(vault.salt);
    const exportKey = await options.deriveVaultKey(passwordValue, salt, vault.kdfParams.iterations);
    const iv = new Uint8Array(options.base64ToBytes(vault.iv));
    const ciphertext = new Uint8Array(options.base64ToBytes(vault.ciphertext));
    const plaintext = await crypto.subtle.decrypt(
      { name: "AES-GCM", iv },
      exportKey,
      ciphertext
    );
    const payload = JSON.parse(new TextDecoder().decode(plaintext));
    if (!payload || typeof payload.mnemonic !== "string") throw new Error("钱包保险库内容无效。");
    return payload;
  }

  function scheduleClose(): void {
    browserWindow.clearTimeout(revealTimer);
    revealTimer = browserWindow.setTimeout(() => {
      if (options.state.wallet.mode === "export-display") options.closeUnlock();
    }, revealTimeoutMs);
  }

  async function exportMnemonic(passwordValue: string): Promise<void> {
    if (!options.state.wallet.vault) {
      lockError.textContent = "未找到本地钱包保险库。";
      return;
    }
    lockSubmit.disabled = true;
    try {
      const payload = await decryptPayload(passwordValue);
      options.state.wallet.pendingMnemonic = payload.mnemonic;
      options.state.wallet.mode = "export-display";
      options.renderMnemonic(payload.mnemonic);
      createStep.hidden = true;
      choiceStep.hidden = true;
      importStep.hidden = true;
      privateKeyExport.hidden = true;
      recovery.hidden = false;
      recoveryConfirmRow.hidden = true;
      recoveryCopy.hidden = false;
      recoveryTitle.textContent = "导出恢复短语";
      recoveryDescription.textContent = "请在安全环境中离线备份。助记词将在关闭窗口、切换后台或 60 秒后从页面清除。";
      lockCopy.textContent = "任何获得助记词的人都可以控制钱包资产，请勿截图、联网保存或发送给他人。";
      lockCancel.textContent = "关闭";
      lockSubmit.textContent = "完成";
      lockError.textContent = "";
      password.value = "";
      scheduleClose();
    } catch {
      lockError.textContent = "无法导出助记词，请检查钱包密码。";
    } finally {
      lockSubmit.disabled = false;
    }
  }

  async function exportPrivateKey(passwordValue: string): Promise<void> {
    if (!options.state.wallet.vault) {
      lockError.textContent = "未找到本地钱包保险库。";
      return;
    }
    lockSubmit.disabled = true;
    let exportSeed: Uint8Array | null = null;
    try {
      const payload = await decryptPayload(passwordValue);
      exportSeed = await options.deriveSeed(payload.mnemonic);
      const keyring = options.normalizeKeyring(payload.keyring);
      const account = keyring.accounts.find((item: AnyRecord) => item.index === keyring.selectedAccount) || keyring.accounts[0];
      const exported = await options.derivePrivateKey(exportSeed, options.state.selectedChainId, account.index);
      const address = options.deriveAddress(exportSeed, options.state.selectedChainId, account.index);
      if (!address || !exported || typeof exported.privateKey !== "string" || typeof exported.path !== "string") {
        throw new Error("私钥派生结果无效。");
      }

      options.state.wallet.mode = "export-display";
      createStep.hidden = true;
      choiceStep.hidden = true;
      importStep.hidden = true;
      recovery.hidden = true;
      privateKeyExport.hidden = false;
      privateKeyChain.textContent = `${options.getSelectedChain().label} · ${account.name} (#${account.index})`;
      privateKeyAddress.textContent = address;
      privateKeyPath.textContent = exported.path;
      privateKeyFormat.textContent = exported.format || "hex";
      privateKeyValue.textContent = exported.privateKey;
      lockCopy.textContent = "私钥只在本设备临时显示。任何获得它的人都能控制当前地址，请勿截图、联网保存或发送给他人。";
      lockCancel.textContent = "关闭";
      lockSubmit.textContent = "完成";
      lockError.textContent = "";
      password.value = "";
      scheduleClose();
    } catch (caught) {
      lockError.textContent = caught instanceof Error && caught.message === "当前页面缺少本地私钥派生能力。"
        ? caught.message
        : "无法导出当前地址私钥，请检查钱包密码。";
    } finally {
      if (exportSeed) exportSeed.fill(0);
      lockSubmit.disabled = false;
    }
  }

  async function copyPrivateKey(): Promise<void> {
    const value = privateKeyValue.textContent.trim();
    if (!value) {
      lockError.textContent = "当前没有可复制的私钥。";
      return;
    }
    try {
      await options.copyText(value);
      lockError.textContent = "";
      privateKeyCopy.textContent = "copied";
    } catch {
      lockError.textContent = "复制私钥失败。";
      privateKeyCopy.textContent = "失败";
    }
    browserWindow.clearTimeout(copyTimer);
    copyTimer = browserWindow.setTimeout(() => {
      privateKeyCopy.textContent = "copy";
    }, 1600);
  }

  function reset(): void {
    browserWindow.clearTimeout(revealTimer);
    revealTimer = 0;
    browserWindow.clearTimeout(copyTimer);
    copyTimer = 0;
    privateKeyExport.hidden = true;
    privateKeyChain.textContent = "";
    privateKeyAddress.textContent = "";
    privateKeyPath.textContent = "";
    privateKeyFormat.textContent = "";
    privateKeyValue.textContent = "";
    privateKeyCopy.textContent = "copy";
  }

  function bind(): void {
    privateKeyCopy.addEventListener("click", () => void copyPrivateKey());
  }

  return Object.freeze({ bind, copyPrivateKey, exportMnemonic, exportPrivateKey, requestMnemonic, requestPrivateKey, reset });
}
