(function initializeAgentWalletPlatform(global) {
  "use strict";

  const vaultStorageKey = "agent-wallet-encrypted-vault";
  const apiBaseStorageKey = "agent-wallet-api-base-url";

  function capacitorPlatform() {
    const capacitor = global.Capacitor;
    if (!capacitor || typeof capacitor.getPlatform !== "function") {
      return "";
    }
    return capacitor.getPlatform();
  }

  function isChromeExtension() {
    return Boolean(global.chrome && global.chrome.runtime && global.chrome.runtime.id);
  }

  function platform() {
    const nativePlatform = capacitorPlatform();
    if (nativePlatform === "ios" || nativePlatform === "android") {
      return nativePlatform;
    }
    if (isChromeExtension()) {
      return "chrome-extension";
    }
    return "web";
  }

  function normalizeBaseURL(value) {
    return String(value || "").trim().replace(/\/+$/, "");
  }

  function configuredAPIBaseURL() {
    const runtimeValue = global.AGENT_WALLET_CONFIG && global.AGENT_WALLET_CONFIG.apiBaseURL;
    const storedValue = global.localStorage ? global.localStorage.getItem(apiBaseStorageKey) : "";
    const meta = global.document && global.document.querySelector('meta[name="agent-wallet-api-base-url"]');
    return normalizeBaseURL(runtimeValue || storedValue || (meta && meta.content));
  }

  function defaultAPIBaseURL() {
    const currentPlatform = platform();
    if (currentPlatform === "android") {
      return "http://10.0.2.2:8081";
    }
    if (
      currentPlatform === "ios" ||
      currentPlatform === "chrome-extension" ||
      (currentPlatform === "web" && global.location && global.location.protocol === "file:")
    ) {
      return "http://localhost:8081";
    }
    return "";
  }

  function apiBaseURL() {
    return configuredAPIBaseURL() || defaultAPIBaseURL();
  }

  function resolveAPIURL(resource) {
    if (typeof resource !== "string" || !resource.startsWith("/")) {
      return resource;
    }
    if (resource === "/wallet/bip39-english" && platform() !== "web") {
      return "./bip39_english.txt";
    }
    return apiBaseURL() + resource;
  }

  function assetBaseURL() {
    return platform() === "web" && global.location.protocol !== "file:" ? "/ui/" : "./";
  }

  function setAPIBaseURL(value) {
    const normalized = normalizeBaseURL(value);
    if (global.localStorage) {
      if (normalized) {
        global.localStorage.setItem(apiBaseStorageKey, normalized);
      } else {
        global.localStorage.removeItem(apiBaseStorageKey);
      }
    }
    return normalized;
  }

  function nativeVaultPlugin() {
    const plugins = global.Capacitor && global.Capacitor.Plugins;
    return plugins && plugins.SecureWalletVault ? plugins.SecureWalletVault : null;
  }

  const secureVault = {
    isPlatformBacked() {
      return Boolean(nativeVaultPlugin() || isChromeExtension());
    },

    async load() {
      const plugin = nativeVaultPlugin();
      if (plugin) {
        const result = await plugin.load();
        return result && result.value ? JSON.parse(result.value) : null;
      }
      if (isChromeExtension()) {
        const result = await global.chrome.storage.local.get(vaultStorageKey);
        return result[vaultStorageKey] || null;
      }
      return null;
    },

    async save(vault) {
      const plugin = nativeVaultPlugin();
      if (plugin) {
        await plugin.save({ value: JSON.stringify(vault) });
        return true;
      }
      if (isChromeExtension()) {
        await global.chrome.storage.local.set({ [vaultStorageKey]: vault });
        return true;
      }
      return false;
    },

    async remove() {
      const plugin = nativeVaultPlugin();
      if (plugin) {
        await plugin.remove();
        return true;
      }
      if (isChromeExtension()) {
        await global.chrome.storage.local.remove(vaultStorageKey);
        return true;
      }
      return false;
    }
  };

  global.AgentWalletPlatform = Object.freeze({
    platform,
    apiBaseURL,
    assetBaseURL,
    resolveAPIURL,
    setAPIBaseURL,
    secureVault
  });
})(window);
