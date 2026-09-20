export interface WalletVaultRecord {
  id: string;
  [key: string]: unknown;
}

interface PlatformWindow extends Window {
  AgentWalletPlatform?: {
    secureVault: {
      isPlatformBacked: () => boolean;
      load: () => Promise<WalletVaultRecord | null>;
      save: (vault: WalletVaultRecord) => Promise<void>;
      remove: () => Promise<void>;
    };
  };
}

export function createWalletVaultStore(browserWindow: PlatformWindow = window as PlatformWindow) {
  const databaseName = "web3_wallet";
  const storeName = "vault";

  function platformVault() {
    const vault = browserWindow.AgentWalletPlatform?.secureVault;
    return vault?.isPlatformBacked() ? vault : null;
  }

  async function open(): Promise<IDBDatabase> {
    if (!("indexedDB" in browserWindow) || !browserWindow.crypto?.subtle) {
      throw new Error("当前浏览器不支持本地加密钱包。");
    }
    return new Promise((resolve, reject) => {
      const request = browserWindow.indexedDB.open(databaseName, 1);
      request.onupgradeneeded = () => {
        const database = request.result;
        if (!database.objectStoreNames.contains(storeName)) database.createObjectStore(storeName, { keyPath: "id" });
      };
      request.onsuccess = () => resolve(request.result);
      request.onerror = () => reject(request.error || new Error("无法打开本地钱包保险库。"));
    });
  }

  async function load(): Promise<WalletVaultRecord | null> {
    const nativeVault = platformVault();
    if (nativeVault) return nativeVault.load();
    const database = await open();
    return new Promise((resolve, reject) => {
      const request = database.transaction(storeName, "readonly").objectStore(storeName).get("primary");
      request.onsuccess = () => resolve((request.result as WalletVaultRecord | undefined) || null);
      request.onerror = () => reject(request.error || new Error("读取钱包保险库失败。"));
    });
  }

  async function save(vault: WalletVaultRecord): Promise<void> {
    const nativeVault = platformVault();
    if (nativeVault) return nativeVault.save(vault);
    const database = await open();
    return new Promise((resolve, reject) => {
      const transaction = database.transaction(storeName, "readwrite");
      transaction.objectStore(storeName).put(vault);
      transaction.oncomplete = () => resolve();
      transaction.onerror = () => reject(transaction.error || new Error("保存钱包保险库失败。"));
      transaction.onabort = () => reject(transaction.error || new Error("保存钱包保险库失败。"));
    });
  }

  async function remove(): Promise<void> {
    const nativeVault = platformVault();
    if (nativeVault) return nativeVault.remove();
    const database = await open();
    return new Promise((resolve, reject) => {
      const transaction = database.transaction(storeName, "readwrite");
      transaction.objectStore(storeName).delete("primary");
      transaction.oncomplete = () => resolve();
      transaction.onerror = () => reject(transaction.error || new Error("删除钱包保险库失败。"));
      transaction.onabort = () => reject(transaction.error || new Error("删除钱包保险库失败。"));
    });
  }

  return Object.freeze({ load, open, remove, save });
}
