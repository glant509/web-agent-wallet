import * as Accounts from "../wallet-accounts/index";

type AnyRecord = Record<string, any>;

interface WalletAccountControllerOptions {
  state: AnyRecord;
  isWalletUnlocked: () => boolean;
  maskAddress: (address: string) => string;
  deriveAddress: (seed: unknown, chainId: string, accountIndex: number) => string;
  randomBytes: (length: number) => Uint8Array;
  bytesToBase64: (bytes: Uint8Array) => string;
  saveVault: (vault: AnyRecord) => Promise<void>;
  onLocked: () => void;
  onChanged: () => void;
  document?: Document;
  crypto?: Crypto;
}

function requiredElement<T extends HTMLElement>(doc: Document, id: string): T {
  const element = doc.getElementById(id);
  if (!element) throw new Error(`missing wallet account element: ${id}`);
  return element as T;
}

export function createWalletAccountController(options: WalletAccountControllerOptions) {
  const doc = options.document || document;
  const cryptoAPI = options.crypto || crypto;
  const dialog = requiredElement(doc, "wallet-account-dialog");
  const closeButton = requiredElement<HTMLButtonElement>(doc, "wallet-account-close");
  const list = requiredElement(doc, "wallet-account-list");
  const addButton = requiredElement<HTMLButtonElement>(doc, "wallet-account-add");
  const renameForm = requiredElement<HTMLFormElement>(doc, "wallet-account-rename");
  const nameInput = requiredElement<HTMLInputElement>(doc, "wallet-account-name");
  const renameCancel = requiredElement<HTMLButtonElement>(doc, "wallet-account-rename-cancel");
  const error = requiredElement(doc, "wallet-account-error");
  const keyrings = requiredElement(doc, "wallet-keyrings");
  let renameIndex = -1;

  function normalize(keyring: unknown): AnyRecord {
    return Accounts.normalizeKeyring(keyring);
  }

  function createDefault(): AnyRecord {
    return Accounts.createDefaultKeyring();
  }

  function getAccounts(): AnyRecord[] {
    options.state.wallet.keyring = normalize(options.state.wallet.keyring);
    return options.state.wallet.keyring.accounts;
  }

  function getSelectedIndex(): number {
    return normalize(options.state.wallet.keyring).selectedAccount;
  }

  function getSelected(): AnyRecord {
    const accounts = getAccounts();
    return accounts.find((account) => account.index === getSelectedIndex()) || accounts[0];
  }

  function accountPath(chainId: string, accountIndex: number): string {
    return Accounts.accountPath(chainId, accountIndex);
  }

  function renderList(): void {
    list.replaceChildren();
    const selectedIndex = getSelectedIndex();
    getAccounts().forEach((account) => {
      const button = doc.createElement("button");
      button.type = "button";
      button.className = `wallet-account-row${account.index === selectedIndex ? " active" : ""}`;
      button.dataset.walletAccountIndex = String(account.index);
      button.setAttribute("aria-label", `${account.name}，账户 ${account.index}，点击切换并修改名称`);
      const copy = doc.createElement("span");
      copy.className = "wallet-account-copy";
      const name = doc.createElement("strong");
      name.textContent = `${account.name} · #${account.index}`;
      const detail = doc.createElement("span");
      try {
        detail.textContent = `${options.maskAddress(options.deriveAddress(options.state.wallet.seed, options.state.selectedChainId, account.index))} · ${accountPath(options.state.selectedChainId, account.index)}`;
      } catch {
        detail.textContent = `地址派生失败 · ${accountPath(options.state.selectedChainId, account.index)}`;
      }
      copy.append(name, detail);
      const stateLabel = doc.createElement("em");
      stateLabel.textContent = account.index === selectedIndex ? "当前" : "切换 / 改名";
      button.append(copy, stateLabel);
      list.appendChild(button);
    });
  }

  function renderKeyrings(keyring: unknown): void {
    keyrings.replaceChildren();
    normalize(keyring).networks.forEach((network: AnyRecord) => {
      const item = doc.createElement("span");
      item.className = "wallet-keyring";
      item.textContent = `${network.label} · ${network.path}`;
      keyrings.appendChild(item);
    });
    keyrings.hidden = false;
  }

  function closeRename(): void {
    renameIndex = -1;
    nameInput.value = "";
    renameForm.hidden = true;
  }

  function openRename(accountIndex: number): void {
    const account = getAccounts().find((item) => item.index === accountIndex);
    if (!account) return;
    renameIndex = accountIndex;
    nameInput.value = account.name;
    renameForm.hidden = false;
    error.textContent = "";
    nameInput.focus();
    nameInput.select();
  }

  function open(): void {
    if (!options.isWalletUnlocked()) {
      if (options.state.wallet.vault) options.onLocked();
      return;
    }
    error.textContent = "";
    closeRename();
    renderList();
    dialog.classList.add("visible");
    dialog.setAttribute("aria-hidden", "false");
  }

  function close(): void {
    dialog.classList.remove("visible");
    dialog.setAttribute("aria-hidden", "true");
    error.textContent = "";
    closeRename();
  }

  async function persist(): Promise<void> {
    if (!options.isWalletUnlocked() || !options.state.wallet.vault || !options.state.wallet.masterKey) throw new Error("钱包尚未解锁。");
    const iv = Uint8Array.from(options.randomBytes(12));
    const now = new Date().toISOString();
    const plaintext = new TextEncoder().encode(JSON.stringify({
      mnemonic: options.state.wallet.mnemonic,
      keyring: normalize(options.state.wallet.keyring),
      createdAt: options.state.wallet.vault.createdAt || now,
      updatedAt: now
    }));
    const ciphertext = await cryptoAPI.subtle.encrypt({ name: "AES-GCM", iv }, options.state.wallet.masterKey, plaintext);
    const updatedVault = {
      ...options.state.wallet.vault,
      iv: options.bytesToBase64(iv),
      ciphertext: options.bytesToBase64(new Uint8Array(ciphertext)),
      updatedAt: now
    };
    await options.saveVault(updatedVault);
    options.state.wallet.vault = updatedVault;
  }

  async function select(accountIndex: number, editName: boolean): Promise<void> {
    const current = normalize(options.state.wallet.keyring);
    if (!current.accounts.some((account: AnyRecord) => account.index === accountIndex)) return;
    if (current.selectedAccount !== accountIndex) {
      options.state.wallet.keyring = normalize({ ...current, account: accountIndex, selectedAccount: accountIndex });
      try {
        await persist();
        options.state.wallet.addresses = {};
        options.state.assetBalances.items = [];
        options.state.assetBalances.error = "";
        options.state.assetBalances.updatedAt = "";
        renderKeyrings(options.state.wallet.keyring);
        renderList();
        options.onChanged();
      } catch {
        options.state.wallet.keyring = current;
        error.textContent = "切换账户失败，请稍后重试。";
        return;
      }
    }
    if (editName) openRename(accountIndex);
  }

  async function add(): Promise<void> {
    const current = normalize(options.state.wallet.keyring);
    const nextIndex = current.accounts.reduce((largest: number, account: AnyRecord) => Math.max(largest, account.index), -1) + 1;
    if (nextIndex > 99) {
      error.textContent = "最多可以添加 100 个账户。";
      return;
    }
    options.state.wallet.keyring = normalize({
      ...current, account: nextIndex, selectedAccount: nextIndex,
      accounts: [...current.accounts, { index: nextIndex, name: `账户 ${nextIndex}` }]
    });
    addButton.disabled = true;
    try {
      await persist();
      options.state.wallet.addresses = {};
      options.state.assetBalances.items = [];
      options.state.assetBalances.error = "";
      options.state.assetBalances.updatedAt = "";
      renderKeyrings(options.state.wallet.keyring);
      renderList();
      openRename(nextIndex);
      options.onChanged();
    } catch {
      options.state.wallet.keyring = current;
      error.textContent = "添加账户失败，请稍后重试。";
    } finally {
      addButton.disabled = false;
    }
  }

  async function saveName(): Promise<void> {
    const name = nameInput.value.trim();
    if (!name) {
      error.textContent = "账户名称不能为空。";
      return;
    }
    const current = normalize(options.state.wallet.keyring);
    if (!current.accounts.some((account: AnyRecord) => account.index === renameIndex)) {
      error.textContent = "未找到需要修改的账户。";
      return;
    }
    options.state.wallet.keyring = normalize({
      ...current,
      accounts: current.accounts.map((account: AnyRecord) => account.index === renameIndex ? { ...account, name: name.slice(0, 24) } : account)
    });
    try {
      await persist();
      renderList();
      closeRename();
      options.onChanged();
    } catch {
      options.state.wallet.keyring = current;
      error.textContent = "保存账户名称失败，请稍后重试。";
    }
  }

  function bind(): void {
    closeButton.addEventListener("click", close);
    dialog.addEventListener("click", (event) => { if (event.target === dialog) close(); });
    list.addEventListener("click", (event) => {
      const button = (event.target as Element).closest<HTMLElement>("[data-wallet-account-index]");
      const index = Number(button?.dataset.walletAccountIndex);
      if (Number.isSafeInteger(index) && index >= 0) void select(index, true);
    });
    addButton.addEventListener("click", () => void add());
    renameForm.addEventListener("submit", (event) => { event.preventDefault(); void saveName(); });
    renameCancel.addEventListener("click", closeRename);
    doc.addEventListener("keydown", (event) => {
      if (event.key === "Escape" && dialog.classList.contains("visible")) close();
    });
  }

  return Object.freeze({ accountPath, bind, close, createDefault, getAccounts, getSelected, getSelectedIndex, normalize, open, persist, renderKeyrings, renderList });
}
