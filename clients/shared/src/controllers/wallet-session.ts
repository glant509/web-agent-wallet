type AnyRecord = Record<string, any>;

interface WalletSessionOptions {
  state: AnyRecord;
  timeoutMs: number;
  beforeLock: () => void;
  afterLock: () => void;
  document?: Document;
  window?: Window;
}

export function createWalletSessionController(options: WalletSessionOptions) {
  const doc = options.document || document;
  const browserWindow = options.window || window;

  function isUnlocked(): boolean {
    return Boolean(options.state.wallet.masterKey && typeof options.state.wallet.mnemonic === "string");
  }

  function lock(): void {
    options.beforeLock();
    browserWindow.clearTimeout(options.state.wallet.idleTimer);
    options.state.wallet.idleTimer = 0;
    options.state.wallet.lastActivity = 0;
    options.state.wallet.pendingAction = "";
    options.state.wallet.pendingPassword = "";
    options.state.wallet.addresses = {};
    options.state.wallet.masterKey = null;
    options.state.wallet.mnemonic = null;
    if (options.state.wallet.seed instanceof Uint8Array) options.state.wallet.seed.fill(0);
    options.state.wallet.seed = null;
    options.state.wallet.keyring = null;
    options.state.wallet.pendingMnemonic = null;
    options.afterLock();
  }

  function refresh(): void {
    if (!isUnlocked()) return;
    options.state.wallet.lastActivity = Date.now();
    browserWindow.clearTimeout(options.state.wallet.idleTimer);
    options.state.wallet.idleTimer = browserWindow.setTimeout(lock, options.timeoutMs);
  }

  function start(): void {
    refresh();
  }

  function bind(): void {
    ["pointerdown", "keydown", "touchstart"].forEach((eventName) => {
      doc.addEventListener(eventName, refresh, { passive: true });
    });
    doc.addEventListener("visibilitychange", () => {
      if (doc.hidden) lock();
    });
  }

  return Object.freeze({ bind, isUnlocked, lock, refresh, start });
}
