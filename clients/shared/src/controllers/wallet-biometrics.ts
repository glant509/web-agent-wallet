type AnyRecord = Record<string, any>;

interface WalletBiometricOptions {
  state: AnyRecord;
  biometrics: {
    status: () => Promise<AnyRecord>;
    authenticate: () => Promise<string>;
    disable: () => Promise<void>;
  };
  isWalletUnlocked: () => boolean;
  unlockWithCredential: (credential: string) => Promise<void>;
  document?: Document;
  window?: Window;
}

function requiredElement<T extends HTMLElement>(doc: Document, id: string): T {
  const element = doc.getElementById(id);
  if (!element) throw new Error(`missing biometric element: ${id}`);
  return element as T;
}

export function createWalletBiometricController(options: WalletBiometricOptions) {
  const doc = options.document || document;
  const browserWindow = options.window || window;
  const unlockButton = requiredElement<HTMLButtonElement>(doc, "wallet-biometric-unlock");
  const disableButton = requiredElement<HTMLButtonElement>(doc, "wallet-biometric-button");
  const enableRow = requiredElement(doc, "wallet-biometric-enable-row");
  const enableCheckbox = requiredElement<HTMLInputElement>(doc, "wallet-biometric-enable");
  const enableLabel = requiredElement(doc, "wallet-biometric-enable-label");
  const error = requiredElement(doc, "wallet-lock-error");

  async function refresh(): Promise<void> {
    try {
      options.state.wallet.biometric = await options.biometrics.status();
    } catch {
      options.state.wallet.biometric = { available: false, enrolled: false, enabled: false, biometryType: "none", label: "生物识别" };
    }
    render();
  }

  function render(): void {
    const biometric = options.state.wallet.biometric;
    const label = biometric.label || "生物识别";
    const unlocking = options.state.wallet.mode === "unlock" && Boolean(options.state.wallet.vault);
    unlockButton.hidden = !(unlocking && biometric.available && biometric.enabled);
    unlockButton.textContent = `使用 ${label} 解锁`;
    enableRow.hidden = !(unlocking && biometric.available && !biometric.enabled);
    enableLabel.textContent = `本次解锁后启用 ${label} 快捷解锁`;
    disableButton.hidden = !(options.isWalletUnlocked() && biometric.enabled);
    disableButton.textContent = `关闭 ${label}`;
  }

  async function unlock(): Promise<void> {
    error.textContent = "";
    unlockButton.disabled = true;
    try {
      await options.unlockWithCredential(await options.biometrics.authenticate());
    } catch (caught) {
      error.textContent = caught instanceof Error ? caught.message : "生物识别验证失败。";
      await refresh();
    } finally {
      unlockButton.disabled = false;
    }
  }

  async function disable(): Promise<void> {
    if (!browserWindow.confirm("关闭生物识别快捷解锁？之后仍可使用钱包密码解锁。")) return;
    try {
      await options.biometrics.disable();
      await refresh();
    } catch (caught) {
      browserWindow.alert(caught instanceof Error ? caught.message : "关闭生物识别快捷解锁失败。");
    }
  }

  function shouldEnableAfterUnlock(): boolean {
    return !enableRow.hidden && enableCheckbox.checked;
  }

  function resetEnableChoice(): void {
    enableCheckbox.checked = false;
    enableRow.hidden = true;
    unlockButton.hidden = true;
  }

  function bind(): void {
    unlockButton.addEventListener("click", () => void unlock());
    disableButton.addEventListener("click", () => void disable());
  }

  return Object.freeze({ bind, disable, refresh, render, resetEnableChoice, shouldEnableAfterUnlock, unlock });
}
