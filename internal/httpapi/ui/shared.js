"use strict";
(() => {
  var __defProp = Object.defineProperty;
  var __export = (target, all) => {
    for (var name in all)
      __defProp(target, name, { get: all[name], enumerable: true });
  };

  // clients/shared/src/wallet-core/index.ts
  var wallet_core_exports = {};
  __export(wallet_core_exports, {
    base64ToBytes: () => base64ToBytes,
    bytesToBase64: () => bytesToBase64,
    createVault: () => createVault,
    deriveBIP39Seed: () => deriveBIP39Seed,
    deriveVaultKey: () => deriveVaultKey,
    isValidPassword: () => isValidPassword,
    randomBytes: () => randomBytes,
    randomHex: () => randomHex,
    unlockVault: () => unlockVault
  });
  function isValidPassword(password) {
    return /^[A-Za-z0-9!@#$%^&*.]{8,16}$/.test(password);
  }
  function randomBytes(length) {
    return crypto.getRandomValues(new Uint8Array(length));
  }
  function randomHex(byteLength) {
    return Array.from(randomBytes(byteLength), (value) => value.toString(16).padStart(2, "0")).join("");
  }
  function bytesToBase64(bytes) {
    let binary = "";
    bytes.forEach((value) => {
      binary += String.fromCharCode(value);
    });
    return btoa(binary);
  }
  function base64ToBytes(value) {
    const binary = atob(value);
    const bytes = new Uint8Array(binary.length);
    for (let index = 0; index < binary.length; index += 1) {
      bytes[index] = binary.charCodeAt(index);
    }
    return bytes;
  }
  async function deriveVaultKey(password, salt, iterations) {
    const passwordBytes = new TextEncoder().encode(password);
    try {
      const material = await crypto.subtle.importKey("raw", passwordBytes, "PBKDF2", false, ["deriveKey"]);
      return await crypto.subtle.deriveKey(
        { name: "PBKDF2", hash: "SHA-256", salt, iterations },
        material,
        { name: "AES-GCM", length: 256 },
        false,
        ["encrypt", "decrypt"]
      );
    } finally {
      passwordBytes.fill(0);
    }
  }
  async function deriveBIP39Seed(mnemonic, passphrase = "") {
    const mnemonicBytes = new TextEncoder().encode(mnemonic.normalize("NFKD"));
    const salt = new TextEncoder().encode(("mnemonic" + passphrase).normalize("NFKD"));
    try {
      const material = await crypto.subtle.importKey("raw", mnemonicBytes, "PBKDF2", false, ["deriveBits"]);
      const bits = await crypto.subtle.deriveBits(
        { name: "PBKDF2", hash: "SHA-512", salt, iterations: 2048 },
        material,
        512
      );
      return new Uint8Array(bits);
    } finally {
      mnemonicBytes.fill(0);
      salt.fill(0);
    }
  }
  async function createVault(password, mnemonic, keyring, iterations = 6e5) {
    const salt = randomBytes(16);
    const iv = randomBytes(12);
    const masterKey = await deriveVaultKey(password, salt, iterations);
    const plaintext = new TextEncoder().encode(JSON.stringify({ mnemonic, keyring, createdAt: (/* @__PURE__ */ new Date()).toISOString() }));
    try {
      const encrypted = await crypto.subtle.encrypt({ name: "AES-GCM", iv }, masterKey, plaintext);
      const createdAt = (/* @__PURE__ */ new Date()).toISOString();
      return {
        masterKey,
        vault: {
          id: "primary",
          version: 1,
          cipher: "AES-256-GCM",
          kdf: "PBKDF2-SHA-256",
          kdfParams: { iterations, hash: "SHA-256" },
          salt: bytesToBase64(salt),
          iv: bytesToBase64(iv),
          ciphertext: bytesToBase64(new Uint8Array(encrypted)),
          createdAt
        }
      };
    } finally {
      plaintext.fill(0);
      salt.fill(0);
      iv.fill(0);
    }
  }
  async function unlockVault(vault, password) {
    const salt = base64ToBytes(vault.salt);
    const iv = base64ToBytes(vault.iv);
    const ciphertext = base64ToBytes(vault.ciphertext);
    try {
      const masterKey = await deriveVaultKey(password, salt, vault.kdfParams.iterations);
      const decrypted = await crypto.subtle.decrypt({ name: "AES-GCM", iv }, masterKey, ciphertext);
      const plaintext = new Uint8Array(decrypted);
      try {
        const payload = JSON.parse(new TextDecoder().decode(plaintext));
        if (!payload || typeof payload.mnemonic !== "string") {
          throw new Error("Invalid wallet vault payload");
        }
        return { masterKey, payload };
      } finally {
        plaintext.fill(0);
      }
    } finally {
      salt.fill(0);
      iv.fill(0);
      ciphertext.fill(0);
    }
  }

  // clients/shared/src/api-client/index.ts
  var APIError = class extends Error {
    constructor(status2, message, body) {
      super(message);
      this.status = status2;
      this.body = body;
      this.name = "APIError";
    }
  };
  var AgentWalletAPIClient = class {
    constructor(resolveURL = (resource) => resource) {
      this.resolveURL = resolveURL;
    }
    createTrace(traceId = "") {
      return {
        traceId: /^[0-9a-f]{32}$/i.test(traceId) ? traceId.toLowerCase() : randomHex(16),
        spanId: randomHex(8)
      };
    }
    request(resource, options = {}, traceId = "") {
      const trace = this.createTrace(traceId);
      const headers = new Headers(options.headers || {});
      headers.set("traceparent", `00-${trace.traceId}-${trace.spanId}-01`);
      headers.set("X-Trace-ID", trace.traceId);
      headers.set("X-Span-ID", trace.spanId);
      return fetch(this.resolveURL(resource), { ...options, headers });
    }
    async json(resource, options = {}, traceId = "") {
      const response = await this.request(resource, options, traceId);
      const body = await response.json().catch(() => null);
      if (!response.ok) {
        const message = body && typeof body === "object" && "error" in body ? String(body.error) : `Request failed with status ${response.status}`;
        throw new APIError(response.status, message, body);
      }
      return body;
    }
  };
  function createAPIClient() {
    return new AgentWalletAPIClient((resource) => {
      const platform = window.AgentWalletPlatform;
      return platform ? platform.resolveAPIURL(resource) : resource;
    });
  }

  // clients/shared/src/ui-components/index.ts
  var ui_components_exports = {};
  __export(ui_components_exports, {
    createLabeledValueRow: () => createLabeledValueRow,
    formatUSD: () => formatUSD,
    maskAddress: () => maskAddress,
    setBusy: () => setBusy,
    setVisible: () => setVisible
  });
  function maskAddress(address, leading = 4, trailing = 4) {
    const value = String(address || "");
    if (!value) return "";
    if (value.startsWith("0x") && value.length > 2 + leading + trailing) {
      return `0x${value.slice(2, 2 + leading)}***${value.slice(-trailing)}`;
    }
    if (value.length <= leading + trailing + 3) return value;
    return `${value.slice(0, leading)}***${value.slice(-trailing)}`;
  }
  function formatUSD(value) {
    const amount = Number(value);
    if (!Number.isFinite(amount)) return "$0.00";
    return new Intl.NumberFormat("en-US", {
      style: "currency",
      currency: "USD",
      minimumFractionDigits: 2,
      maximumFractionDigits: amount >= 1 ? 2 : 6
    }).format(amount);
  }
  function setVisible(element, visible) {
    if (element) element.hidden = !visible;
  }
  function setBusy(button, busy, busyLabel) {
    if (!button) return () => void 0;
    const previousText = button.textContent || "";
    const previousDisabled = button.disabled;
    button.disabled = busy;
    if (busy && busyLabel) button.textContent = busyLabel;
    return () => {
      button.disabled = previousDisabled;
      button.textContent = previousText;
    };
  }
  function createLabeledValueRow(label, value, className) {
    const row = document.createElement("div");
    row.className = className;
    const name = document.createElement("span");
    name.textContent = label;
    const detail = document.createElement("strong");
    detail.textContent = value;
    row.append(name, detail);
    return row;
  }

  // clients/shared/src/biometric-auth/index.ts
  var biometric_auth_exports = {};
  __export(biometric_auth_exports, {
    authenticate: () => authenticate,
    disable: () => disable,
    enable: () => enable,
    status: () => status
  });
  function plugin() {
    const capacitor = window.Capacitor;
    return capacitor?.Plugins?.WalletBiometrics || null;
  }
  function labelFor(type) {
    if (type === "face") return "Face ID";
    if (type === "fingerprint") return "\u6307\u7EB9/Touch ID";
    if (type === "iris") return "\u8679\u819C\u8BC6\u522B";
    return "\u751F\u7269\u8BC6\u522B";
  }
  async function status() {
    const native = plugin();
    if (!native) return { available: false, enrolled: false, enabled: false, biometryType: "none", label: "\u751F\u7269\u8BC6\u522B" };
    const result = await native.status();
    const biometryType = result.biometryType || "none";
    return {
      available: Boolean(result.available),
      enrolled: Boolean(result.enrolled),
      enabled: Boolean(result.enabled),
      biometryType,
      label: result.label || labelFor(biometryType)
    };
  }
  async function enable(credential, reason = "\u542F\u7528\u94B1\u5305\u5FEB\u6377\u89E3\u9501") {
    if (!credential) throw new Error("Credential is required");
    const native = plugin();
    if (!native) throw new Error("Biometric authentication is unavailable");
    await native.saveCredential({ credential, reason });
  }
  async function authenticate(reason = "\u9A8C\u8BC1\u8EAB\u4EFD\u4EE5\u89E3\u9501\u94B1\u5305") {
    const native = plugin();
    if (!native) throw new Error("Biometric authentication is unavailable");
    const result = await native.authenticate({ reason });
    if (!result.credential) throw new Error("Biometric credential is unavailable");
    return result.credential;
  }
  async function disable() {
    const native = plugin();
    if (native) await native.removeCredential();
  }

  // clients/shared/src/wallet-accounts/index.ts
  var wallet_accounts_exports = {};
  __export(wallet_accounts_exports, {
    accountPath: () => accountPath,
    createDefaultKeyring: () => createDefaultKeyring,
    createNetworks: () => createNetworks,
    normalizeKeyring: () => normalizeKeyring
  });
  function createNetworks(accountIndex) {
    const account = Number.isSafeInteger(accountIndex) && accountIndex >= 0 ? accountIndex : 0;
    const evmPath = `m/44'/60'/${account}'/0/0`;
    return [
      { id: "bitcoin", label: "Bitcoin", coinType: 0, path: `m/44'/0'/${account}'/0/0`, curve: "secp256k1" },
      { id: "ethereum", label: "Ethereum", coinType: 60, path: evmPath, curve: "secp256k1" },
      { id: "base", label: "Base", coinType: 60, path: evmPath, curve: "secp256k1" },
      { id: "arbitrum", label: "Arbitrum", coinType: 60, path: evmPath, curve: "secp256k1" },
      { id: "optimism", label: "Optimism", coinType: 60, path: evmPath, curve: "secp256k1" },
      { id: "bnb", label: "BNB Chain", coinType: 60, path: evmPath, curve: "secp256k1" },
      { id: "polygon", label: "Polygon", coinType: 60, path: evmPath, curve: "secp256k1" },
      { id: "avalanche", label: "Avalanche", coinType: 60, path: evmPath, curve: "secp256k1" },
      { id: "solana", label: "Solana", coinType: 501, path: `m/44'/501'/${account}'/0'`, curve: "ed25519" },
      { id: "sui", label: "Sui", coinType: 784, path: `m/44'/784'/${account}'/0'/0'`, curve: "ed25519" }
    ];
  }
  function createDefaultKeyring() {
    return {
      standard: "BIP44",
      account: 0,
      selectedAccount: 0,
      index: 0,
      accounts: [{ index: 0, name: "\u8D26\u6237 0" }],
      networks: createNetworks(0)
    };
  }
  function normalizeKeyring(input) {
    const candidate = input && typeof input === "object" ? input : {};
    const source = candidate.standard === "BIP44" ? candidate : {};
    const sourceAccounts = Array.isArray(source.accounts) ? source.accounts : [];
    const accounts = [];
    const seen = /* @__PURE__ */ new Set();
    sourceAccounts.forEach((entry) => {
      const account = entry && typeof entry === "object" ? entry : {};
      const index = Number(account.index);
      if (!Number.isSafeInteger(index) || index < 0 || index > 99 || seen.has(index)) return;
      seen.add(index);
      const rawName = typeof account.name === "string" ? account.name.trim() : "";
      accounts.push({ index, name: (rawName || `\u8D26\u6237 ${index}`).slice(0, 24) });
    });
    if (accounts.length === 0) {
      const legacyAccount = Number(source.account);
      const index = Number.isSafeInteger(legacyAccount) && legacyAccount >= 0 && legacyAccount <= 99 ? legacyAccount : 0;
      accounts.push({ index, name: `\u8D26\u6237 ${index}` });
    }
    accounts.sort((left, right) => left.index - right.index);
    const requested = Number(source.selectedAccount ?? source.account);
    const selectedAccount = accounts.some((account) => account.index === requested) ? requested : accounts[0].index;
    return {
      ...source,
      standard: "BIP44",
      account: selectedAccount,
      selectedAccount,
      index: 0,
      accounts,
      networks: createNetworks(selectedAccount)
    };
  }
  function accountPath(chainId, accountIndex) {
    return createNetworks(accountIndex).find((network) => network.id === chainId)?.path || `BIP44 account ${accountIndex}`;
  }

  // clients/shared/src/wallet-history/index.ts
  var wallet_history_exports = {};
  __export(wallet_history_exports, {
    dateLabel: () => dateLabel,
    filter: () => filter,
    merge: () => merge,
    normalizeItem: () => normalizeItem,
    recordKey: () => recordKey,
    typeLabel: () => typeLabel
  });
  function normalizeItem(input) {
    const item = input && typeof input === "object" ? input : {};
    return {
      id: String(item.id || item.hash || ""),
      hash: String(item.hash || ""),
      chainId: String(item.chain_id || item.chainId || "") === "bsc" ? "bnb" : String(item.chain_id || item.chainId || ""),
      type: String(item.type || ""),
      direction: String(item.direction || ""),
      timestamp: String(item.timestamp || ""),
      from: String(item.from || ""),
      to: String(item.to || ""),
      tokenSymbol: String(item.token_symbol || item.tokenSymbol || ""),
      amount: String(item.amount || "0"),
      fee: String(item.fee || ""),
      feeSymbol: String(item.fee_symbol || item.feeSymbol || ""),
      status: String(item.status || "confirmed")
    };
  }
  function recordKey(item) {
    return [item.chainId, item.hash, item.direction, item.tokenSymbol, item.amount].join(":").toLowerCase();
  }
  function merge(remoteItems, localItems, limit = 200) {
    const merged = /* @__PURE__ */ new Map();
    [...remoteItems, ...localItems].map(normalizeItem).forEach((item) => {
      const key = recordKey(item);
      if (item.hash && !merged.has(key)) merged.set(key, item);
    });
    return Array.from(merged.values()).slice(0, limit);
  }
  function typeLabel(type) {
    return { send: "\u53D1\u9001", receive: "\u63A5\u6536", swap: "\u4EA4\u6613", bridge: "\u8DE8\u94FE" }[type] || "\u4EA4\u6613";
  }
  function dateLabel(timestamp) {
    const date = new Date(timestamp);
    if (Number.isNaN(date.getTime())) return "\u65E5\u671F\u672A\u77E5";
    return `${date.getFullYear()}/${String(date.getMonth() + 1).padStart(2, "0")}/${String(date.getDate()).padStart(2, "0")}`;
  }
  function filter(items, filters) {
    const chainId = filters.chainId || "all";
    const type = filters.type || "all";
    return items.map(normalizeItem).filter((item) => chainId === "all" || item.chainId === chainId).filter((item) => type === "all" || item.type === type).sort((left, right) => new Date(right.timestamp).getTime() - new Date(left.timestamp).getTime());
  }

  // clients/shared/src/transactions/index.ts
  var transactions_exports = {};
  __export(transactions_exports, {
    forSigning: () => forSigning,
    isValidAmount: () => isValidAmount,
    isValidRecipient: () => isValidRecipient
  });
  function isValidRecipient(chainId, address) {
    const value = String(address || "").trim();
    if (!value) return false;
    if (["ethereum", "base", "arbitrum", "optimism", "bnb", "polygon", "avalanche"].includes(chainId)) {
      return /^0x[0-9a-fA-F]{40}$/.test(value);
    }
    if (chainId === "bitcoin") return /^(bc1[ac-hj-np-z02-9]{11,71}|[13][a-km-zA-HJ-NP-Z1-9]{25,34})$/.test(value);
    if (chainId === "solana") return /^[1-9A-HJ-NP-Za-km-z]{32,44}$/.test(value);
    if (chainId === "sui") return /^0x[0-9a-fA-F]{64}$/.test(value);
    return value.length >= 20;
  }
  function isValidAmount(amount) {
    const value = String(amount || "").trim();
    return /^\d+(\.\d+)?$/.test(value) && Number(value) > 0;
  }
  function forSigning(plan) {
    const transaction = {
      type: plan.transaction_type,
      chainId: plan.network_chain_id,
      nonce: plan.nonce,
      to: plan.to,
      value: plan.value,
      data: plan.data,
      gasLimit: plan.gas_limit
    };
    if (plan.transaction_type === 2) {
      transaction.maxFeePerGas = plan.max_fee_per_gas;
      transaction.maxPriorityFeePerGas = plan.max_priority_fee_per_gas;
    } else {
      transaction.gasPrice = plan.gas_price;
    }
    return transaction;
  }

  // clients/shared/src/app-state/index.ts
  var app_state_exports = {};
  __export(app_state_exports, {
    createInitialState: () => createInitialState
  });
  function createInitialState(sessionId, messagesByTab) {
    return {
      activeTab: "home",
      busyTab: "",
      selectedChainId: "ethereum",
      sessionId,
      wallet: {
        vault: null,
        masterKey: null,
        mnemonic: null,
        seed: null,
        addresses: {},
        keyring: null,
        mode: "",
        pendingAction: "",
        pendingPassword: "",
        pendingMnemonic: null,
        idleTimer: 0,
        lastActivity: 0,
        biometric: { available: false, enrolled: false, enabled: false, biometryType: "none", label: "\u751F\u7269\u8BC6\u522B" }
      },
      messagesByTab,
      marketTop: { items: [], loading: false, error: "", loadedAt: 0 },
      tradeDashboard: { item: null, selectedAssetId: "", selectedTokenName: "", loading: false, error: "" },
      tradeKlines: { item: null, loading: false, error: "", updatedAt: "", refreshTimer: 0 },
      assetBalances: { loading: false, error: "", totalUSD: 0, items: [], selectedChainID: "", updatedAt: "", requestToken: 0 },
      assetAction: { mode: "", token: null, plan: null, sendMax: false, busy: false, traceId: "" },
      walletHistory: { items: [], chainId: "", type: "all", loading: false, error: "", requestToken: 0, traceId: "" }
    };
  }

  // clients/shared/src/market/index.ts
  var market_exports = {};
  __export(market_exports, {
    calculateChange: () => calculateChange,
    formatChartAxisPrice: () => formatChartAxisPrice,
    formatChartPrice: () => formatChartPrice,
    formatCompact: () => formatCompact,
    formatPercent: () => formatPercent,
    formatTimeLabel: () => formatTimeLabel,
    formatUSDPrice: () => formatUSDPrice,
    formatUpdatedTime: () => formatUpdatedTime,
    parseCandleLine: () => parseCandleLine,
    parseObservation: () => parseObservation,
    stats: () => stats,
    toChartY: () => toChartY
  });
  function parseCandleLine(line) {
    const match = line.match(/^- ([^ ]+) open=([0-9.]+) high=([0-9.]+) low=([0-9.]+) close=([0-9.]+)/);
    if (!match) return null;
    return { timestamp: match[1], open: Number(match[2]), high: Number(match[3]), low: Number(match[4]), close: Number(match[5]) };
  }
  function parseObservation(text) {
    if (typeof text !== "string" || !text.includes("Candles:")) return null;
    const lines = text.split("\n").map((line) => line.trim()).filter(Boolean);
    const candlesIndex = lines.indexOf("Candles:");
    if (candlesIndex <= 0) return null;
    const meta = {};
    lines.slice(1, candlesIndex).forEach((line) => {
      if (!line.startsWith("- ")) return;
      const separator = line.indexOf(":");
      if (separator <= 0) return;
      meta[line.slice(2, separator).trim()] = line.slice(separator + 1).trim();
    });
    const candles = lines.slice(candlesIndex + 1).map(parseCandleLine).filter((candle) => Boolean(candle));
    if (candles.length === 0) return null;
    return {
      title: lines[0].replace(/:$/, ""),
      mode: meta.mode || "",
      vsCurrency: (meta.vs_currency || "usd").toUpperCase(),
      interval: meta.interval || "",
      totalCandles: Number(meta.total_candles || candles.length),
      candlesReturned: Number(meta.candles_returned || candles.length),
      assetID: meta.asset_id || "",
      tokenAddress: meta.token_address || "",
      chain: meta.chain || "",
      note: meta.note || "",
      source: meta.source || "",
      candles
    };
  }
  function calculateChange(candles) {
    const first = candles[0];
    const last = candles[candles.length - 1];
    if (!first || !last || first.open === 0) return { className: "flat", label: "0.00%", percent: 0 };
    const percent = (last.close - first.open) / first.open * 100;
    return {
      className: percent > 0 ? "positive" : percent < 0 ? "negative" : "flat",
      label: `${percent > 0 ? "+" : ""}${percent.toFixed(2)}%`,
      percent
    };
  }
  function formatChartPrice(value) {
    if (value >= 1e3) return value.toFixed(2);
    if (value >= 1) return value.toFixed(4);
    if (value >= 0.01) return value.toFixed(6);
    return value.toFixed(8);
  }
  function formatChartAxisPrice(value) {
    if (value >= 1e3) return value.toFixed(0);
    if (value >= 1) return value.toFixed(2);
    return value.toFixed(4);
  }
  function stats(candles) {
    if (candles.length === 0) return [];
    return [
      { label: "H", value: formatChartPrice(Math.max(...candles.map((candle) => candle.high))) },
      { label: "L", value: formatChartPrice(Math.min(...candles.map((candle) => candle.low))) },
      { label: "O", value: formatChartPrice(candles[0].open) },
      { label: "C", value: formatChartPrice(candles[candles.length - 1].close) }
    ];
  }
  function formatTimeLabel(timestamp) {
    const date = new Date(timestamp);
    if (Number.isNaN(date.getTime())) return timestamp;
    return `${String(date.getUTCMonth() + 1).padStart(2, "0")}/${String(date.getUTCDate()).padStart(2, "0")} ${String(date.getUTCHours()).padStart(2, "0")}:${String(date.getUTCMinutes()).padStart(2, "0")}`;
  }
  function toChartY(value, minPrice, range, top, height) {
    return top + (minPrice + range - value) / range * height;
  }
  function formatUSDPrice(value) {
    const amount = Number(value || 0);
    if (amount >= 1e3) return "$" + amount.toLocaleString(void 0, { maximumFractionDigits: 2, minimumFractionDigits: 2 });
    if (amount >= 1) return "$" + amount.toFixed(2);
    if (amount >= 0.01) return "$" + amount.toFixed(4);
    return "$" + amount.toFixed(6);
  }
  function formatPercent(value) {
    const amount = Number(value || 0);
    return `${amount > 0 ? "+" : ""}${amount.toFixed(2)}%`;
  }
  function formatCompact(value, currency = false) {
    const amount = Number(value || 0);
    if (!amount) return currency ? "$0.00" : "0";
    return (currency ? "$" : "") + amount.toLocaleString(void 0, { notation: "compact", maximumFractionDigits: 2 });
  }
  function formatUpdatedTime(value) {
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) return value;
    return `${String(date.getHours()).padStart(2, "0")}:${String(date.getMinutes()).padStart(2, "0")}:${String(date.getSeconds()).padStart(2, "0")}`;
  }

  // clients/shared/src/controllers/wallet-history.ts
  var wallet_history_exports2 = {};
  __export(wallet_history_exports2, {
    createWalletHistoryController: () => createWalletHistoryController
  });
  var storageKey = "web3_wallet_transaction_history_v1";
  var typeOptions = [
    { value: "all", label: "\u5168\u90E8\u7C7B\u578B" },
    { value: "send", label: "\u53D1\u9001" },
    { value: "receive", label: "\u63A5\u6536" },
    { value: "swap", label: "\u4EA4\u6613" },
    { value: "bridge", label: "\u8DE8\u94FE" }
  ];
  var explorerURLs = {
    ethereum: "https://etherscan.io",
    base: "https://basescan.org",
    arbitrum: "https://arbiscan.io",
    optimism: "https://explorer.optimism.io",
    bnb: "https://bscscan.com",
    polygon: "https://polygonscan.com",
    avalanche: "https://snowtrace.io",
    solana: "https://solscan.io",
    bitcoin: "https://mempool.space",
    sui: "https://suiscan.xyz/mainnet"
  };
  function requiredElement(doc, id) {
    const element = doc.getElementById(id);
    if (!element) throw new Error(`missing wallet history element: ${id}`);
    return element;
  }
  function createWalletHistoryController(options) {
    const doc = options.document || document;
    const browserWindow = options.window || window;
    const dialog = requiredElement(doc, "wallet-history-dialog");
    const back = requiredElement(doc, "wallet-history-back");
    const chainFilter = requiredElement(doc, "wallet-history-chain-filter");
    const chainTrigger = requiredElement(doc, "wallet-history-chain-trigger");
    const chainLabel = requiredElement(doc, "wallet-history-chain-label");
    const chainMenu = requiredElement(doc, "wallet-history-chain-menu");
    const typeFilter = requiredElement(doc, "wallet-history-type-filter");
    const typeTrigger = requiredElement(doc, "wallet-history-type-trigger");
    const typeLabelElement = requiredElement(doc, "wallet-history-type-label");
    const typeMenu = requiredElement(doc, "wallet-history-type-menu");
    const list = requiredElement(doc, "wallet-history-list");
    const empty = requiredElement(doc, "wallet-history-empty");
    const loading = requiredElement(doc, "wallet-history-loading");
    const error = requiredElement(doc, "wallet-history-error");
    const explorer = requiredElement(doc, "wallet-history-explorer");
    function load() {
      try {
        const raw = browserWindow.localStorage.getItem(storageKey);
        const parsed = raw ? JSON.parse(raw) : [];
        return Array.isArray(parsed) ? parsed.filter((item) => item && typeof item === "object" && typeof item.hash === "string").map(normalizeItem).slice(0, 200) : [];
      } catch {
        return [];
      }
    }
    function persist() {
      try {
        browserWindow.localStorage.setItem(storageKey, JSON.stringify(options.state.walletHistory.items.slice(0, 200)));
      } catch {
      }
    }
    function recordBroadcast(record) {
      const item = normalizeItem({
        id: `${record.hash}:local`,
        hash: record.hash,
        chainId: record.chainId,
        type: "send",
        direction: "send",
        timestamp: (/* @__PURE__ */ new Date()).toISOString(),
        accountIndex: record.accountIndex,
        accountName: record.accountName,
        from: record.from,
        to: record.to,
        tokenSymbol: record.tokenSymbol,
        amount: record.amount,
        fee: record.fee,
        feeSymbol: record.feeSymbol,
        status: "broadcast"
      });
      options.state.walletHistory.items = [
        item,
        ...options.state.walletHistory.items.filter((existing) => existing.hash !== record.hash)
      ].slice(0, 200);
      persist();
    }
    function explorerURL(chainId, kind, value) {
      const base = explorerURLs[chainId];
      if (!base || !value) return "";
      if (kind === "tx") return `${base}/tx/${encodeURIComponent(value)}`;
      return `${base}/${chainId === "solana" ? "account" : "address"}/${encodeURIComponent(value)}`;
    }
    function openExternal(url) {
      if (!url) return;
      const opened = browserWindow.open(url, "_blank", "noopener,noreferrer");
      if (opened) opened.opener = null;
    }
    function explorerTarget() {
      const walletHistory = options.state.walletHistory;
      const chainId = walletHistory.chainId === "all" ? options.state.selectedChainId : walletHistory.chainId || options.state.selectedChainId;
      let address = options.isWalletUnlocked() && chainId === options.state.selectedChainId ? options.getSelectedAddress() : "";
      if (!address) {
        const record = walletHistory.items.find((item) => item.chainId === chainId);
        address = record ? record.from || record.to || "" : "";
      }
      return explorerURL(chainId, "address", address);
    }
    function renderFilterMenu(menu, kind, entries, selected) {
      menu.replaceChildren();
      entries.forEach((entry) => {
        const button = doc.createElement("button");
        button.type = "button";
        button.className = `wallet-history-menu-option${entry.value === selected ? " active" : ""}`;
        button.dataset.historyFilterKind = kind;
        button.dataset.historyFilterValue = entry.value;
        button.setAttribute("role", "option");
        button.setAttribute("aria-selected", entry.value === selected ? "true" : "false");
        button.textContent = entry.label;
        menu.appendChild(button);
      });
    }
    function initializeFilters() {
      const walletHistory = options.state.walletHistory;
      renderFilterMenu(chainMenu, "chain", [
        { value: "all", label: "\u5168\u90E8\u7F51\u7EDC" },
        ...options.chains.map((chain) => ({ value: chain.id, label: chain.label }))
      ], walletHistory.chainId || options.state.selectedChainId);
      renderFilterMenu(typeMenu, "type", typeOptions, walletHistory.type || "all");
      const selectedChain = options.chains.find((chain) => chain.id === walletHistory.chainId);
      chainLabel.textContent = walletHistory.chainId === "all" ? "\u5168\u90E8\u7F51\u7EDC" : selectedChain?.label || "Ethereum";
      typeLabelElement.textContent = typeOptions.find((entry) => entry.value === walletHistory.type)?.label || "\u5168\u90E8\u7C7B\u578B";
    }
    function closeMenus() {
      [[chainFilter, chainTrigger, chainMenu], [typeFilter, typeTrigger, typeMenu]].forEach(([filter2, trigger, menu]) => {
        filter2.classList.remove("open");
        trigger.setAttribute("aria-expanded", "false");
        menu.hidden = true;
      });
    }
    function toggleMenu(kind) {
      const target = kind === "chain" ? [chainFilter, chainTrigger, chainMenu] : [typeFilter, typeTrigger, typeMenu];
      const shouldOpen = target[2].hidden;
      closeMenus();
      if (shouldOpen) {
        target[0].classList.add("open");
        target[1].setAttribute("aria-expanded", "true");
        target[2].hidden = false;
      }
    }
    function createItem(item) {
      const button = doc.createElement("button");
      button.type = "button";
      button.className = "wallet-history-item";
      button.dataset.walletHistoryHash = item.hash;
      button.dataset.walletHistoryChain = item.chainId;
      button.setAttribute("aria-label", `${typeLabel(item.type)} ${item.amount} ${item.tokenSymbol}`);
      const icon = doc.createElement("span");
      icon.className = "wallet-history-token-icon";
      icon.textContent = String(item.tokenSymbol || "TX").slice(0, 5);
      const copy = doc.createElement("span");
      copy.className = "wallet-history-copy";
      const title = doc.createElement("strong");
      title.textContent = typeLabel(item.type);
      const detail = doc.createElement("span");
      const incoming = item.direction === "receive";
      detail.textContent = `${incoming ? "\u6765\u81EA " : "\u81F3 "}${options.maskAddress((incoming ? item.from : item.to) || "\u5730\u5740\u672A\u77E5")}`;
      copy.append(title, detail);
      const value = doc.createElement("span");
      value.className = "wallet-history-value";
      const amount = doc.createElement("strong");
      amount.className = incoming ? "incoming" : "";
      amount.textContent = `${incoming ? "+" : "-"}${item.amount} ${item.tokenSymbol}`;
      const fee = doc.createElement("span");
      fee.textContent = incoming ? item.status === "broadcast" ? "\u5DF2\u5E7F\u64AD" : "" : item.fee ? `\u624B\u7EED\u8D39 ${item.fee} ${item.feeSymbol}` : item.status === "broadcast" ? "\u5DF2\u5E7F\u64AD" : "";
      value.append(amount, fee);
      button.append(icon, copy, value);
      return button;
    }
    function render2() {
      const walletHistory = options.state.walletHistory;
      const items = filter(walletHistory.items, { chainId: walletHistory.chainId, type: walletHistory.type });
      list.replaceChildren();
      loading.hidden = !walletHistory.loading;
      error.hidden = !walletHistory.error;
      error.textContent = walletHistory.error;
      empty.hidden = walletHistory.loading || items.length > 0;
      const groups = /* @__PURE__ */ new Map();
      items.forEach((item) => {
        const date = dateLabel(item.timestamp);
        groups.set(date, [...groups.get(date) || [], item]);
      });
      groups.forEach((transactions, date) => {
        const group = doc.createElement("section");
        group.className = "wallet-history-group";
        const heading = doc.createElement("h3");
        heading.className = "wallet-history-date";
        heading.textContent = date;
        group.append(heading, ...transactions.map(createItem));
        list.appendChild(group);
      });
      explorer.disabled = !explorerTarget();
    }
    async function refresh() {
      if (!options.isWalletUnlocked()) return;
      const walletHistory = options.state.walletHistory;
      const requestToken = ++walletHistory.requestToken;
      walletHistory.loading = true;
      walletHistory.error = "";
      render2();
      const requested = walletHistory.chainId === "all" ? options.supportedChainIds : [walletHistory.chainId];
      const chainIds = requested.filter((chainId) => options.supportedChainIds.includes(chainId));
      if (chainIds.length === 0) {
        walletHistory.loading = false;
        walletHistory.error = "\u5F53\u524D\u7F51\u7EDC\u6682\u4E0D\u652F\u6301\u5728\u94B1\u5305\u5185\u540C\u6B65\u4EA4\u6613\u5386\u53F2\uFF0C\u8BF7\u4F7F\u7528\u4E0B\u65B9\u533A\u5757\u6D4F\u89C8\u5668\u67E5\u770B\u3002";
        render2();
        return;
      }
      const results = await Promise.allSettled(chainIds.map(async (chainId) => {
        const normalizedChainId = options.normalizeChainId(chainId);
        const address = await options.ensureAddress(normalizedChainId);
        if (!address) throw new Error("\u65E0\u6CD5\u6D3E\u751F\u5F53\u524D\u8D26\u6237\u5730\u5740");
        const response = await options.request("/v1/wallet/evm/history", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ chain_id: normalizedChainId, address })
        }, walletHistory.traceId);
        const payload = await response.json().catch(() => ({}));
        if (!response.ok) throw new Error(payload.error || "\u94FE\u4E0A\u5386\u53F2\u67E5\u8BE2\u5931\u8D25");
        return Array.isArray(payload.items) ? payload.items : [];
      }));
      if (requestToken !== walletHistory.requestToken) return;
      const remoteItems = results.flatMap((result) => result.status === "fulfilled" ? result.value : []);
      walletHistory.items = merge(remoteItems, load());
      const failures = results.map((result, index) => ({ result, chainId: chainIds[index] })).filter((entry) => entry.result.status === "rejected");
      failures.forEach(({ result, chainId }) => {
        console.warn("[wallet-history] chain sync failed", {
          traceId: walletHistory.traceId,
          chainId,
          reason: result.reason instanceof Error ? result.reason.message : String(result.reason || "unknown error")
        });
      });
      walletHistory.error = failures.length === results.length ? "\u6682\u65F6\u65E0\u6CD5\u540C\u6B65\u94FE\u4E0A\u5386\u53F2\uFF0C\u5DF2\u663E\u793A\u672C\u5730\u8BB0\u5F55\u3002\u8BF7\u7A0D\u540E\u91CD\u8BD5\u6216\u524D\u5F80\u533A\u5757\u6D4F\u89C8\u5668\u67E5\u770B\u3002" : failures.length ? "\u90E8\u5206\u7F51\u7EDC\u540C\u6B65\u5931\u8D25\uFF0C\u5176\u4F59\u94FE\u4E0A\u8BB0\u5F55\u5DF2\u663E\u793A\u3002" : "";
      walletHistory.loading = false;
      render2();
    }
    function open() {
      if (!options.isWalletUnlocked()) {
        options.onLocked(Boolean(options.state.wallet.vault));
        return;
      }
      options.onBeforeOpen();
      const walletHistory = options.state.walletHistory;
      walletHistory.items = load();
      walletHistory.chainId = options.state.selectedChainId;
      walletHistory.type = "all";
      walletHistory.error = "";
      walletHistory.traceId = options.createTraceId();
      initializeFilters();
      render2();
      dialog.classList.add("visible");
      dialog.setAttribute("aria-hidden", "false");
      void refresh();
    }
    function close() {
      options.state.walletHistory.requestToken += 1;
      options.state.walletHistory.loading = false;
      closeMenus();
      dialog.classList.remove("visible");
      dialog.setAttribute("aria-hidden", "true");
    }
    function bind() {
      back.addEventListener("click", close);
      chainTrigger.addEventListener("click", () => toggleMenu("chain"));
      typeTrigger.addEventListener("click", () => toggleMenu("type"));
      dialog.addEventListener("click", (event) => {
        if (event.target === dialog) return close();
        const target = event.target;
        const option = target.closest("[data-history-filter-kind]");
        if (option) {
          const kind = option.dataset.historyFilterKind;
          const value = option.dataset.historyFilterValue || "all";
          if (kind === "chain") options.state.walletHistory.chainId = value;
          if (kind === "type") options.state.walletHistory.type = value;
          closeMenus();
          initializeFilters();
          render2();
          if (kind === "chain") void refresh();
          return;
        }
        if (!chainFilter.contains(target) && !typeFilter.contains(target)) closeMenus();
      });
      list.addEventListener("click", (event) => {
        const item = event.target.closest("[data-wallet-history-hash]");
        if (item) openExternal(explorerURL(item.dataset.walletHistoryChain || "", "tx", item.dataset.walletHistoryHash || ""));
      });
      explorer.addEventListener("click", () => openExternal(explorerTarget()));
      doc.addEventListener("keydown", (event) => {
        if (event.key !== "Escape" || !dialog.classList.contains("visible")) return;
        if (!chainMenu.hidden || !typeMenu.hidden) closeMenus();
        else close();
      });
    }
    return Object.freeze({ bind, close, load, open, persist, recordBroadcast, refresh, render: render2 });
  }

  // clients/shared/src/qr-code/index.ts
  var qr_code_exports = {};
  __export(qr_code_exports, {
    createQRCodeMatrix: () => createQRCodeMatrix,
    render: () => render
  });
  function createQRCodeMatrix(value) {
    const version = 5;
    const size = version * 4 + 17;
    const dataCodewords = 108;
    const errorCodewords = 26;
    const bytes = Array.from(new TextEncoder().encode(value));
    if (bytes.length > 106) throw new Error("\u63A5\u6536\u5730\u5740\u8FC7\u957F\uFF0C\u65E0\u6CD5\u751F\u6210\u4E8C\u7EF4\u7801\u3002");
    const bits = [];
    const appendBits = (number, length) => {
      for (let bit = length - 1; bit >= 0; bit -= 1) bits.push(number >>> bit & 1);
    };
    appendBits(4, 4);
    appendBits(bytes.length, 8);
    bytes.forEach((byte) => appendBits(byte, 8));
    appendBits(0, Math.min(4, dataCodewords * 8 - bits.length));
    while (bits.length % 8 !== 0) bits.push(0);
    let pad = 0;
    while (bits.length < dataCodewords * 8) {
      appendBits(pad % 2 === 0 ? 236 : 17, 8);
      pad += 1;
    }
    const data = [];
    for (let offset = 0; offset < bits.length; offset += 8) {
      let byte = 0;
      for (let bit = 0; bit < 8; bit += 1) byte = byte << 1 | bits[offset + bit];
      data.push(byte);
    }
    const exponent = new Array(512);
    const logarithm = new Array(256).fill(0);
    let valueAt = 1;
    for (let index = 0; index < 255; index += 1) {
      exponent[index] = valueAt;
      logarithm[valueAt] = index;
      valueAt <<= 1;
      if (valueAt & 256) valueAt ^= 285;
    }
    for (let index = 255; index < 512; index += 1) exponent[index] = exponent[index - 255];
    const multiply = (left, right) => left === 0 || right === 0 ? 0 : exponent[logarithm[left] + logarithm[right]];
    let generator = [1];
    for (let degree = 0; degree < errorCodewords; degree += 1) {
      const next = new Array(generator.length + 1).fill(0);
      generator.forEach((coefficient, index) => {
        next[index] ^= coefficient;
        next[index + 1] ^= multiply(coefficient, exponent[degree]);
      });
      generator = next;
    }
    const remainder = new Array(errorCodewords).fill(0);
    data.forEach((byte) => {
      const factor = byte ^ remainder[0];
      remainder.shift();
      remainder.push(0);
      for (let index = 0; index < errorCodewords; index += 1) remainder[index] ^= multiply(generator[index + 1], factor);
    });
    const codewordBits = [];
    data.concat(remainder).forEach((byte) => {
      for (let bit = 7; bit >= 0; bit -= 1) codewordBits.push(byte >>> bit & 1);
    });
    const matrix = Array.from({ length: size }, () => new Array(size).fill(false));
    const functions = Array.from({ length: size }, () => new Array(size).fill(false));
    const setFunction = (x, y, dark) => {
      if (x >= 0 && y >= 0 && x < size && y < size) {
        matrix[y][x] = dark;
        functions[y][x] = true;
      }
    };
    const drawFinder = (left, top) => {
      for (let dy = -1; dy <= 7; dy += 1) {
        for (let dx = -1; dx <= 7; dx += 1) {
          const inside = dx >= 0 && dx <= 6 && dy >= 0 && dy <= 6;
          setFunction(left + dx, top + dy, inside && (dx === 0 || dx === 6 || dy === 0 || dy === 6 || dx >= 2 && dx <= 4 && dy >= 2 && dy <= 4));
        }
      }
    };
    drawFinder(0, 0);
    drawFinder(size - 7, 0);
    drawFinder(0, size - 7);
    for (let index = 8; index < size - 8; index += 1) {
      if (!functions[6][index]) setFunction(index, 6, index % 2 === 0);
      if (!functions[index][6]) setFunction(6, index, index % 2 === 0);
    }
    for (let dy = -2; dy <= 2; dy += 1) {
      for (let dx = -2; dx <= 2; dx += 1) setFunction(30 + dx, 30 + dy, Math.max(Math.abs(dx), Math.abs(dy)) !== 1);
    }
    for (let index = 0; index < 15; index += 1) {
      const verticalY = index < 6 ? index : index < 8 ? index + 1 : size - 15 + index;
      const horizontalX = index < 8 ? size - index - 1 : index === 8 ? 7 : 14 - index;
      setFunction(8, verticalY, false);
      setFunction(horizontalX, 8, false);
    }
    setFunction(8, size - 8, true);
    let bitIndex = 0;
    let upward = true;
    for (let right = size - 1; right >= 1; right -= 2) {
      if (right === 6) right -= 1;
      for (let step = 0; step < size; step += 1) {
        const y = upward ? size - 1 - step : step;
        for (let offset = 0; offset < 2; offset += 1) {
          const x = right - offset;
          if (functions[y][x]) continue;
          const raw = bitIndex < codewordBits.length ? codewordBits[bitIndex] === 1 : false;
          matrix[y][x] = raw !== ((x + y) % 2 === 0);
          bitIndex += 1;
        }
      }
      upward = !upward;
    }
    const formatData = 8;
    let remainderBits = formatData << 10;
    for (let bit = 14; bit >= 10; bit -= 1) if (remainderBits >>> bit & 1) remainderBits ^= 1335 << bit - 10;
    const formatBits = (formatData << 10 | remainderBits) ^ 21522;
    for (let index = 0; index < 15; index += 1) {
      const dark = (formatBits >>> index & 1) === 1;
      const verticalY = index < 6 ? index : index < 8 ? index + 1 : size - 15 + index;
      const horizontalX = index < 8 ? size - index - 1 : index === 8 ? 7 : 14 - index;
      matrix[verticalY][8] = dark;
      matrix[8][horizontalX] = dark;
    }
    matrix[size - 8][8] = true;
    return matrix;
  }
  function render(container, value, doc = document) {
    const matrix = createQRCodeMatrix(value);
    const quietZone = 4;
    const dimension = matrix.length + quietZone * 2;
    const svg = doc.createElementNS("http://www.w3.org/2000/svg", "svg");
    svg.setAttribute("viewBox", `0 0 ${dimension} ${dimension}`);
    svg.setAttribute("role", "img");
    svg.setAttribute("aria-label", "\u63A5\u6536\u5730\u5740\u4E8C\u7EF4\u7801");
    svg.setAttribute("shape-rendering", "crispEdges");
    const background = doc.createElementNS("http://www.w3.org/2000/svg", "rect");
    background.setAttribute("width", String(dimension));
    background.setAttribute("height", String(dimension));
    background.setAttribute("fill", "#ffffff");
    svg.appendChild(background);
    const path = doc.createElementNS("http://www.w3.org/2000/svg", "path");
    let commands = "";
    matrix.forEach((row, y) => row.forEach((dark, x) => {
      if (dark) commands += `M${x + quietZone} ${y + quietZone}h1v1h-1z`;
    }));
    path.setAttribute("d", commands);
    path.setAttribute("fill", "#020617");
    svg.appendChild(path);
    container.replaceChildren(svg);
  }

  // clients/shared/src/controllers/asset-transfer.ts
  var asset_transfer_exports = {};
  __export(asset_transfer_exports, {
    createAssetTransferController: () => createAssetTransferController
  });
  function requiredElement2(doc, id) {
    const element = doc.getElementById(id);
    if (!element) throw new Error(`missing asset transfer element: ${id}`);
    return element;
  }
  function createAssetTransferController(options) {
    const doc = options.document || document;
    const browserWindow = options.window || window;
    const views = requiredElement2(doc, "views");
    const dialog = requiredElement2(doc, "asset-action-dialog");
    const closeButton = requiredElement2(doc, "asset-action-close");
    const title = requiredElement2(doc, "asset-action-title");
    const description = requiredElement2(doc, "asset-action-description");
    const tokenPicker = requiredElement2(doc, "asset-token-picker");
    const tokenList = requiredElement2(doc, "asset-token-list");
    const sendPage = requiredElement2(doc, "asset-send-page");
    const receivePage = requiredElement2(doc, "asset-receive-page");
    const sendSelectedToken = requiredElement2(doc, "asset-send-selected-token");
    const receiveSelectedToken = requiredElement2(doc, "asset-receive-selected-token");
    const sendChangeToken = requiredElement2(doc, "asset-send-change-token");
    const receiveChangeToken = requiredElement2(doc, "asset-receive-change-token");
    const sendForm = requiredElement2(doc, "asset-send-form");
    const sendAddress = requiredElement2(doc, "asset-send-address");
    const sendAmount = requiredElement2(doc, "asset-send-amount");
    const sendMax = requiredElement2(doc, "asset-send-max");
    const sendNext = requiredElement2(doc, "asset-send-next");
    const sendFeePreview = requiredElement2(doc, "asset-send-fee-preview");
    const receiveQR = requiredElement2(doc, "asset-receive-qr");
    const receiveAddress = requiredElement2(doc, "asset-receive-address");
    const receiveCopy = requiredElement2(doc, "asset-receive-copy");
    const actionError = requiredElement2(doc, "asset-action-error");
    const receiveError = requiredElement2(doc, "asset-receive-error");
    const review = requiredElement2(doc, "asset-transaction-review");
    const rows = requiredElement2(doc, "asset-transaction-rows");
    const reviewBack = requiredElement2(doc, "asset-transaction-back");
    const confirm = requiredElement2(doc, "asset-transaction-confirm");
    const transactionError = requiredElement2(doc, "asset-transaction-error");
    const success = requiredElement2(doc, "asset-transaction-success");
    const transactionHash = requiredElement2(doc, "asset-transaction-hash");
    let returnTimer = 0;
    function tokens() {
      const balanceItems = options.getCurrentBalanceItems();
      if (balanceItems.length) {
        return balanceItems.map((item) => ({
          symbol: String(item.symbol || "TOKEN"),
          name: String(item.name || item.symbol || "Token"),
          balance: String(item.balance || "0"),
          balanceRaw: String(item.balance_raw || "0"),
          tokenAddress: String(item.token_address || ""),
          decimals: Number.isSafeInteger(Number(item.decimals)) ? Number(item.decimals) : 18,
          isNative: Boolean(item.is_native)
        }));
      }
      return (options.tokenCatalog[options.state.selectedChainId] || []).map(([symbol, name], index) => ({
        symbol,
        name,
        balance: "--",
        balanceRaw: "0",
        tokenAddress: "",
        decimals: 18,
        isNative: index === 0
      }));
    }
    function renderTokenPicker() {
      const chain = options.getSelectedChain();
      title.textContent = options.state.assetAction.mode === "receive" ? "Receive" : "Send";
      description.textContent = `\u9009\u62E9 ${chain.label} \u652F\u6301\u7684 Token`;
      tokenList.replaceChildren();
      tokens().forEach((token, index) => {
        const button = doc.createElement("button");
        button.type = "button";
        button.className = "asset-token-option";
        button.dataset.assetTokenIndex = String(index);
        const unavailable = options.state.assetAction.mode === "send" && !token.isNative && !token.tokenAddress;
        button.disabled = unavailable;
        const icon = doc.createElement("span");
        icon.className = "asset-token-symbol";
        icon.textContent = token.symbol.slice(0, 5);
        const copy = doc.createElement("span");
        copy.className = "asset-token-copy";
        const symbol = doc.createElement("strong");
        symbol.textContent = `${token.symbol}${token.isNative ? " \xB7 Native" : ""}`;
        const name = doc.createElement("span");
        name.textContent = token.name;
        copy.append(symbol, name);
        const balance = doc.createElement("em");
        balance.textContent = unavailable ? "\u4F59\u989D\u52A0\u8F7D\u540E\u53EF\u7528" : token.balance === "--" ? "\u9009\u62E9" : token.balance;
        button.append(icon, copy, balance);
        tokenList.appendChild(button);
      });
      tokenPicker.hidden = false;
      sendPage.hidden = true;
      receivePage.hidden = true;
    }
    function open(mode) {
      if (mode !== "send" && mode !== "receive") return;
      if (!options.isWalletUnlocked()) {
        options.onLocked(mode, Boolean(options.state.wallet.vault));
        return;
      }
      options.onBeforeOpen();
      browserWindow.clearTimeout(returnTimer);
      returnTimer = 0;
      Object.assign(options.state.assetAction, { mode, token: null, plan: null, sendMax: false, busy: false, traceId: options.createTraceId() });
      actionError.textContent = "";
      receiveError.textContent = "";
      sendAddress.value = "";
      sendAmount.value = "";
      receiveQR.replaceChildren();
      receiveAddress.textContent = "";
      receiveCopy.textContent = "\u590D\u5236\u63A5\u6536\u5730\u5740";
      sendNext.disabled = false;
      sendNext.textContent = "\u4E0B\u4E00\u6B65";
      sendFeePreview.textContent = "\u624B\u7EED\u8D39\u5C06\u5728\u4E0B\u4E00\u6B65\u901A\u8FC7\u5F53\u524D\u7F51\u7EDC\u5B9E\u65F6\u9884\u4F30\u3002";
      review.hidden = true;
      success.hidden = true;
      renderTokenPicker();
      dialog.classList.add("visible");
      dialog.setAttribute("aria-hidden", "false");
    }
    function close() {
      browserWindow.clearTimeout(returnTimer);
      returnTimer = 0;
      dialog.classList.remove("visible");
      dialog.setAttribute("aria-hidden", "true");
      Object.assign(options.state.assetAction, { mode: "", token: null, plan: null, sendMax: false, busy: false });
      actionError.textContent = "";
      receiveError.textContent = "";
      sendPage.hidden = true;
      receivePage.hidden = true;
      receiveQR.replaceChildren();
      rows.replaceChildren();
    }
    async function selectToken(index) {
      const token = tokens()[index];
      if (!token) return;
      options.state.assetAction.token = token;
      tokenPicker.hidden = true;
      actionError.textContent = "";
      receiveError.textContent = "";
      const isReceive = options.state.assetAction.mode === "receive";
      sendPage.hidden = isReceive;
      receivePage.hidden = !isReceive;
      sendSelectedToken.textContent = `${token.symbol} \xB7 ${options.getSelectedChain().label}`;
      receiveSelectedToken.textContent = `${token.symbol} \xB7 ${options.getSelectedChain().label}`;
      sendMax.hidden = !token.isNative;
      options.state.assetAction.sendMax = false;
      options.state.assetAction.plan = null;
      sendFeePreview.textContent = token.isNative ? "\u70B9\u51FB\u6700\u5927\u503C\u4F1A\u81EA\u52A8\u9884\u7559\u5B9E\u65F6\u9884\u4F30\u7684\u6700\u9AD8\u7F51\u7EDC\u624B\u7EED\u8D39\u3002" : `Token \u8F6C\u8D26\u624B\u7EED\u8D39\u5C06\u4F7F\u7528 ${options.getSelectedChain().short} \u5355\u72EC\u652F\u4ED8\u3002`;
      title.textContent = `${isReceive ? "Receive" : "Send"} ${token.symbol}`;
      description.textContent = isReceive ? "\u626B\u63CF\u4E8C\u7EF4\u7801\u6216\u590D\u5236\u5730\u5740\u63A5\u6536\u8D44\u4EA7" : "\u586B\u5199\u63A5\u6536\u5730\u5740\u4E0E\u8F6C\u8D26\u6570\u91CF";
      if (isReceive) {
        const address = await options.ensureSelectedAddress();
        if (!address) {
          receiveError.textContent = "\u5F53\u524D\u94FE\u63A5\u6536\u5730\u5740\u6D3E\u751F\u5931\u8D25\u3002";
          return;
        }
        receiveAddress.textContent = address;
        render(receiveQR, address, doc);
      } else if (!options.supportedChainIds.includes(options.state.selectedChainId)) {
        sendForm.hidden = true;
        actionError.textContent = "\u5F53\u524D\u94FE\u5C1A\u672A\u914D\u7F6E\u5B89\u5168\u7684\u4EA4\u6613\u6784\u5EFA\u4E0E\u5E7F\u64AD\u80FD\u529B\u3002\u73B0\u9636\u6BB5 Send \u652F\u6301 Ethereum\u3001Base\u3001Arbitrum\u3001Optimism\u3001BNB\u3001Polygon \u548C Avalanche\u3002";
      } else {
        sendForm.hidden = false;
        browserWindow.requestAnimationFrame(() => sendAddress.focus());
      }
    }
    function validate() {
      if (!isValidRecipient(options.state.selectedChainId, sendAddress.value.trim())) {
        actionError.textContent = `\u8BF7\u8F93\u5165\u6709\u6548\u7684 ${options.getSelectedChain().label} \u63A5\u6536\u5730\u5740\u3002`;
        return false;
      }
      if (!isValidAmount(sendAmount.value.trim())) {
        actionError.textContent = "\u8BF7\u8F93\u5165\u5927\u4E8E 0 \u7684\u8F6C\u8D26\u6570\u91CF\u3002";
        return false;
      }
      return true;
    }
    async function requestPlan(sendMaximum) {
      const token = options.state.assetAction.token;
      if (!token || !options.supportedChainIds.includes(options.state.selectedChainId)) throw new Error("\u5F53\u524D\u94FE\u6682\u4E0D\u652F\u6301\u53D1\u9001\u3002 ");
      const from = await options.ensureSelectedAddress();
      if (!from) throw new Error("\u53D1\u9001\u5730\u5740\u6D3E\u751F\u5931\u8D25\u3002 ");
      let recipient = sendAddress.value.trim();
      if (sendMaximum && !isValidRecipient(options.state.selectedChainId, recipient)) recipient = from;
      const response = await options.request("/v1/wallet/evm/prepare", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          chain_id: options.state.selectedChainId,
          from,
          to: recipient,
          token_address: token.isNative ? "" : token.tokenAddress,
          amount: sendMaximum ? "" : sendAmount.value.trim(),
          send_max: sendMaximum
        })
      }, options.state.assetAction.traceId);
      const payload = await response.json();
      if (!response.ok) throw new Error(payload.error || "\u4EA4\u6613\u624B\u7EED\u8D39\u9884\u4F30\u5931\u8D25\u3002 ");
      return payload;
    }
    async function applyMaximum() {
      if (!options.state.assetAction.token?.isNative || options.state.assetAction.busy) return;
      options.state.assetAction.busy = true;
      sendMax.disabled = true;
      sendNext.disabled = true;
      actionError.textContent = "";
      sendFeePreview.textContent = "\u6B63\u5728\u6839\u636E\u5F53\u524D\u7F51\u7EDC\u8BA1\u7B97\u53EF\u8F6C\u51FA\u7684\u6700\u5927\u503C...";
      try {
        const plan = await requestPlan(true);
        options.state.assetAction.plan = plan;
        options.state.assetAction.sendMax = true;
        sendAmount.value = plan.amount;
        sendFeePreview.textContent = `\u5DF2\u9884\u7559\u6700\u9AD8\u7EA6 ${plan.estimated_fee} ${plan.native_symbol} \u7F51\u7EDC\u624B\u7EED\u8D39\u3002`;
      } catch (caught) {
        options.state.assetAction.sendMax = false;
        options.state.assetAction.plan = null;
        sendFeePreview.textContent = "\u65E0\u6CD5\u8BA1\u7B97\u6700\u5927\u503C\u3002";
        actionError.textContent = caught instanceof Error ? caught.message : "\u6700\u5927\u503C\u8BA1\u7B97\u5931\u8D25\u3002";
      } finally {
        options.state.assetAction.busy = false;
        sendMax.disabled = false;
        sendNext.disabled = false;
      }
    }
    function renderReview(plan) {
      const account = options.getSelectedAccount();
      rows.replaceChildren(
        createLabeledValueRow("\u5F53\u524D\u7F51\u7EDC", `${options.getSelectedChain().label} \xB7 Chain ID ${BigInt(plan.network_chain_id)}`, "asset-transaction-row"),
        createLabeledValueRow("\u53D1\u9001\u8D44\u4EA7", `${plan.amount} ${plan.token_symbol}`, "asset-transaction-row"),
        createLabeledValueRow("\u9884\u8BA1\u624B\u7EED\u8D39", `\u6700\u9AD8\u7EA6 ${plan.estimated_fee} ${plan.native_symbol}`, "asset-transaction-row"),
        createLabeledValueRow("\u8D26\u6237\u7F16\u53F7", `${account.name} \xB7 #${account.index}`, "asset-transaction-row"),
        createLabeledValueRow("\u53D1\u9001\u5730\u5740", options.getSelectedAddress(), "asset-transaction-row"),
        createLabeledValueRow("\u6536\u6B3E\u5730\u5740", sendAddress.value.trim(), "asset-transaction-row")
      );
      sendPage.hidden = true;
      success.hidden = true;
      review.hidden = false;
      transactionError.textContent = "";
      title.textContent = "\u786E\u8BA4\u4EA4\u6613";
      description.textContent = "\u8BF7\u4ED4\u7EC6\u6838\u5BF9\u7F51\u7EDC\u3001\u8D44\u4EA7\u3001\u624B\u7EED\u8D39\u548C\u5730\u5740";
    }
    async function prepareReview() {
      if (!validate() || options.state.assetAction.busy) return;
      options.state.assetAction.busy = true;
      sendNext.disabled = true;
      sendNext.textContent = "\u9884\u4F30\u624B\u7EED\u8D39...";
      actionError.textContent = "";
      try {
        const plan = await requestPlan(options.state.assetAction.sendMax);
        options.state.assetAction.plan = plan;
        sendAmount.value = plan.amount;
        renderReview(plan);
      } catch (caught) {
        options.state.assetAction.plan = null;
        actionError.textContent = caught instanceof Error ? caught.message : "\u4EA4\u6613\u51C6\u5907\u5931\u8D25\u3002";
      } finally {
        options.state.assetAction.busy = false;
        sendNext.disabled = false;
        sendNext.textContent = "\u4E0B\u4E00\u6B65";
      }
    }
    async function confirmAndBroadcast() {
      const plan = options.state.assetAction.plan;
      if (!plan || options.state.assetAction.busy || !options.isWalletUnlocked()) return;
      options.state.assetAction.busy = true;
      confirm.disabled = true;
      confirm.textContent = "\u672C\u5730\u7B7E\u540D\u5E76\u5E7F\u64AD\u4E2D...";
      transactionError.textContent = "";
      try {
        const accountIndex = options.getSelectedAccountIndex();
        const expectedAddress = options.getSelectedAddress().toLowerCase();
        if (expectedAddress !== options.signerAddress(options.state.wallet.seed, accountIndex).toLowerCase()) {
          throw new Error("\u7B7E\u540D\u8D26\u6237\u4E0E\u5F53\u524D\u53D1\u9001\u5730\u5740\u4E0D\u4E00\u81F4\uFF0C\u4EA4\u6613\u5DF2\u53D6\u6D88\u3002 ");
        }
        let rawTransaction = await options.signTransaction(options.state.wallet.seed, accountIndex, forSigning(plan));
        const response = await options.request("/v1/wallet/evm/broadcast", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ chain_id: plan.chain_id, raw_transaction: rawTransaction })
        }, options.state.assetAction.traceId);
        rawTransaction = "";
        const payload = await response.json();
        if (!response.ok) throw new Error(payload.error || "\u4EA4\u6613\u5E7F\u64AD\u5931\u8D25\u3002 ");
        if (payload.transaction_hash) options.onBroadcast(plan, payload.transaction_hash, sendAddress.value.trim());
        review.hidden = true;
        success.hidden = false;
        transactionHash.textContent = payload.transaction_hash || "\u5DF2\u63D0\u4EA4";
        title.textContent = "\u4EA4\u6613\u5DF2\u5E7F\u64AD";
        description.textContent = `${options.getSelectedChain().label} \u7F51\u7EDC\u5DF2\u63A5\u6536\u7B7E\u540D\u4EA4\u6613`;
        returnTimer = browserWindow.setTimeout(() => {
          returnTimer = 0;
          close();
          options.onComplete();
        }, 2200);
      } catch (caught) {
        transactionError.textContent = caught instanceof Error ? caught.message : "\u4EA4\u6613\u7B7E\u540D\u6216\u5E7F\u64AD\u5931\u8D25\u3002";
      } finally {
        options.state.assetAction.busy = false;
        confirm.disabled = false;
        confirm.textContent = "\u786E\u8BA4\u5E76\u5E7F\u64AD";
      }
    }
    async function copyReceiveAddress() {
      const address = receiveAddress.textContent?.trim() || "";
      if (!address) return;
      try {
        await options.copyText(address);
        receiveCopy.textContent = "\u5DF2\u590D\u5236";
      } catch {
        receiveCopy.textContent = "\u590D\u5236\u5931\u8D25";
      }
      browserWindow.setTimeout(() => {
        receiveCopy.textContent = "\u590D\u5236\u63A5\u6536\u5730\u5740";
      }, 1600);
    }
    function bind() {
      views.addEventListener("click", (event) => {
        const button = event.target.closest("[data-asset-action]");
        if (!button) return;
        const action = button.dataset.assetAction || "";
        if (action === "send" || action === "receive") open(action);
      });
      closeButton.addEventListener("click", close);
      dialog.addEventListener("click", (event) => {
        if (event.target === dialog) close();
      });
      tokenList.addEventListener("click", (event) => {
        const button = event.target.closest("[data-asset-token-index]");
        const index = Number(button?.dataset.assetTokenIndex);
        if (Number.isSafeInteger(index) && index >= 0) void selectToken(index);
      });
      sendChangeToken.addEventListener("click", () => {
        options.state.assetAction.token = null;
        actionError.textContent = "";
        renderTokenPicker();
      });
      receiveChangeToken.addEventListener("click", () => {
        options.state.assetAction.token = null;
        receiveError.textContent = "";
        receiveQR.replaceChildren();
        receiveAddress.textContent = "";
        renderTokenPicker();
      });
      sendForm.addEventListener("submit", (event) => {
        event.preventDefault();
        void prepareReview();
      });
      sendMax.addEventListener("click", () => void applyMaximum());
      sendAmount.addEventListener("input", () => {
        options.state.assetAction.sendMax = false;
        options.state.assetAction.plan = null;
        actionError.textContent = "";
      });
      sendAddress.addEventListener("input", () => {
        options.state.assetAction.plan = null;
        actionError.textContent = "";
      });
      confirm.addEventListener("click", () => void confirmAndBroadcast());
      reviewBack.addEventListener("click", () => {
        if (options.state.assetAction.busy) return;
        review.hidden = true;
        sendPage.hidden = false;
        title.textContent = `Send ${options.state.assetAction.token?.symbol || "Token"}`;
        description.textContent = "\u586B\u5199\u63A5\u6536\u5730\u5740\u4E0E\u8F6C\u8D26\u6570\u91CF";
      });
      receiveCopy.addEventListener("click", () => void copyReceiveAddress());
      doc.addEventListener("keydown", (event) => {
        if (event.key === "Escape" && dialog.classList.contains("visible")) close();
      });
    }
    return Object.freeze({ bind, close, open, renderTokenPicker });
  }

  // clients/shared/src/controllers/asset-balance.ts
  var asset_balance_exports = {};
  __export(asset_balance_exports, {
    createAssetBalanceController: () => createAssetBalanceController,
    derivationChainId: () => derivationChainId,
    normalizeChainId: () => normalizeChainId
  });
  function requiredElement3(doc, id) {
    const element = doc.getElementById(id);
    if (!element) throw new Error(`missing asset balance element: ${id}`);
    return element;
  }
  function normalizeChainId(chainId) {
    return chainId === "bnb" ? "bsc" : chainId;
  }
  function derivationChainId(chainId) {
    return chainId === "bsc" ? "bnb" : chainId;
  }
  function createAssetBalanceController(options) {
    const doc = options.document || document;
    const status2 = requiredElement3(doc, "asset-balance-status");
    const total = requiredElement3(doc, "asset-balance-total");
    const list = requiredElement3(doc, "asset-balance-list");
    const chainBadge = requiredElement3(doc, "asset-balance-chain-badge");
    function currentItems() {
      if (options.state.assetBalances.selectedChainID !== normalizeChainId(options.state.selectedChainId)) return [];
      return Array.isArray(options.state.assetBalances.items) ? options.state.assetBalances.items : [];
    }
    function summary(items) {
      return items.length ? `\u5DF2\u52A0\u8F7D ${items.length} \u9879\u4F59\u989D` : "--";
    }
    function render2() {
      chainBadge.textContent = options.getSelectedChain().short;
      if (!options.isWalletUnlocked()) {
        status2.textContent = "\u89E3\u9501\u94B1\u5305\u540E\u52A0\u8F7D\u4F59\u989D";
        total.textContent = "--";
        list.replaceChildren();
        return;
      }
      if (!options.supportedChainIds.includes(normalizeChainId(options.state.selectedChainId))) {
        status2.textContent = "\u5F53\u524D\u4F59\u989D\u770B\u677F\u6682\u652F\u6301 ETH\u3001Base\u3001Arbitrum\u3001OP\u3001BNB\u3001Polygon\u3001Solana\u3001Avalanche";
        total.textContent = summary(currentItems());
        list.replaceChildren();
        return;
      }
      if (options.state.assetBalances.loading) {
        status2.textContent = "\u4F59\u989D\u52A0\u8F7D\u4E2D...";
        total.textContent = summary(currentItems());
        list.replaceChildren();
        return;
      }
      if (options.state.assetBalances.error) {
        status2.textContent = options.state.assetBalances.error;
        total.textContent = "--";
        list.replaceChildren();
        return;
      }
      status2.textContent = options.state.assetBalances.updatedAt ? `\u5DF2\u66F4\u65B0 \xB7 ${new Date(options.state.assetBalances.updatedAt).toLocaleTimeString()}` : "\u5DF2\u52A0\u8F7D";
      const items = currentItems();
      total.textContent = summary(items);
      list.replaceChildren();
      items.forEach((item) => {
        const row = doc.createElement("div");
        row.className = "asset-balance-item";
        const token = doc.createElement("div");
        token.className = "asset-balance-token";
        const tokenTitle = doc.createElement("strong");
        tokenTitle.textContent = `${item.symbol}${item.is_native ? " \xB7 Native" : ""}`;
        const tokenName = doc.createElement("span");
        tokenName.textContent = item.name;
        token.append(tokenTitle, tokenName);
        const values = doc.createElement("div");
        values.className = "asset-balance-values";
        const balance = doc.createElement("strong");
        balance.textContent = item.balance;
        const raw = doc.createElement("span");
        raw.textContent = `Raw: ${String(item.balance_raw || "0")}`;
        values.append(balance, raw);
        row.append(token, values);
        list.appendChild(row);
      });
    }
    async function ensureAddress(chainId) {
      if (!options.isWalletUnlocked()) return "";
      const normalized = normalizeChainId(chainId);
      const cached = options.state.wallet.addresses[normalized];
      if (typeof cached === "string" && cached) return cached;
      const derived = options.deriveAddress(derivationChainId(normalized));
      options.state.wallet.addresses[normalized] = derived;
      return derived;
    }
    function clear(chainId = "") {
      const requestToken = options.state.assetBalances.requestToken;
      options.state.assetBalances = {
        loading: false,
        error: "",
        totalUSD: 0,
        items: [],
        selectedChainID: chainId,
        updatedAt: "",
        requestToken
      };
    }
    async function refresh() {
      render2();
      if (!options.isWalletUnlocked()) {
        clear();
        render2();
        return;
      }
      const selectedChainID = normalizeChainId(options.state.selectedChainId);
      if (!options.supportedChainIds.includes(selectedChainID)) {
        clear(selectedChainID);
        render2();
        return;
      }
      const requestToken = ++options.state.assetBalances.requestToken;
      Object.assign(options.state.assetBalances, {
        loading: true,
        error: "",
        items: [],
        selectedChainID,
        updatedAt: ""
      });
      render2();
      try {
        const address = await ensureAddress(selectedChainID);
        if (!address) throw new Error("\u5F53\u524D\u94FE\u5730\u5740\u6D3E\u751F\u5931\u8D25");
        const response = await options.request("/v1/asset/portfolio", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ selected_chain_id: selectedChainID, addresses: { [selectedChainID]: address } })
        });
        const payload = await response.json();
        if (!response.ok) throw new Error(payload.error || "\u4F59\u989D\u67E5\u8BE2\u5931\u8D25");
        if (requestToken !== options.state.assetBalances.requestToken || normalizeChainId(options.state.selectedChainId) !== selectedChainID) return;
        const selected = payload && typeof payload.selected === "object" ? payload.selected : {};
        const walletChains = Array.isArray(payload?.wallet?.chains) ? payload.wallet.chains : [];
        const fallback = walletChains.find((item) => item?.chain_id === selectedChainID) || walletChains[0] || {};
        Object.assign(options.state.assetBalances, {
          loading: false,
          error: "",
          totalUSD: 0,
          items: Array.isArray(selected.items) ? selected.items : Array.isArray(fallback.items) ? fallback.items : [],
          selectedChainID: selected.chain_id || fallback.chain_id || selectedChainID,
          updatedAt: payload.updated_at || ""
        });
      } catch (caught) {
        if (requestToken !== options.state.assetBalances.requestToken || normalizeChainId(options.state.selectedChainId) !== selectedChainID) return;
        Object.assign(options.state.assetBalances, {
          loading: false,
          error: caught instanceof Error ? caught.message : "\u4F59\u989D\u67E5\u8BE2\u5931\u8D25",
          totalUSD: 0,
          items: [],
          selectedChainID,
          updatedAt: ""
        });
      }
      render2();
    }
    function selectChain() {
      clear(normalizeChainId(options.state.selectedChainId));
      void refresh();
    }
    return Object.freeze({ currentItems, ensureAddress, normalizeChainId, refresh, render: render2, selectChain });
  }

  // clients/shared/src/controllers/wallet-accounts.ts
  var wallet_accounts_exports2 = {};
  __export(wallet_accounts_exports2, {
    createWalletAccountController: () => createWalletAccountController
  });
  function requiredElement4(doc, id) {
    const element = doc.getElementById(id);
    if (!element) throw new Error(`missing wallet account element: ${id}`);
    return element;
  }
  function createWalletAccountController(options) {
    const doc = options.document || document;
    const cryptoAPI = options.crypto || crypto;
    const dialog = requiredElement4(doc, "wallet-account-dialog");
    const closeButton = requiredElement4(doc, "wallet-account-close");
    const list = requiredElement4(doc, "wallet-account-list");
    const addButton = requiredElement4(doc, "wallet-account-add");
    const renameForm = requiredElement4(doc, "wallet-account-rename");
    const nameInput = requiredElement4(doc, "wallet-account-name");
    const renameCancel = requiredElement4(doc, "wallet-account-rename-cancel");
    const error = requiredElement4(doc, "wallet-account-error");
    const keyrings = requiredElement4(doc, "wallet-keyrings");
    let renameIndex = -1;
    function normalize(keyring) {
      return normalizeKeyring(keyring);
    }
    function createDefault() {
      return createDefaultKeyring();
    }
    function getAccounts() {
      options.state.wallet.keyring = normalize(options.state.wallet.keyring);
      return options.state.wallet.keyring.accounts;
    }
    function getSelectedIndex() {
      return normalize(options.state.wallet.keyring).selectedAccount;
    }
    function getSelected() {
      const accounts = getAccounts();
      return accounts.find((account) => account.index === getSelectedIndex()) || accounts[0];
    }
    function accountPath2(chainId, accountIndex) {
      return accountPath(chainId, accountIndex);
    }
    function renderList() {
      list.replaceChildren();
      const selectedIndex = getSelectedIndex();
      getAccounts().forEach((account) => {
        const button = doc.createElement("button");
        button.type = "button";
        button.className = `wallet-account-row${account.index === selectedIndex ? " active" : ""}`;
        button.dataset.walletAccountIndex = String(account.index);
        button.setAttribute("aria-label", `${account.name}\uFF0C\u8D26\u6237 ${account.index}\uFF0C\u70B9\u51FB\u5207\u6362\u5E76\u4FEE\u6539\u540D\u79F0`);
        const copy = doc.createElement("span");
        copy.className = "wallet-account-copy";
        const name = doc.createElement("strong");
        name.textContent = `${account.name} \xB7 #${account.index}`;
        const detail = doc.createElement("span");
        try {
          detail.textContent = `${options.maskAddress(options.deriveAddress(options.state.wallet.seed, options.state.selectedChainId, account.index))} \xB7 ${accountPath2(options.state.selectedChainId, account.index)}`;
        } catch {
          detail.textContent = `\u5730\u5740\u6D3E\u751F\u5931\u8D25 \xB7 ${accountPath2(options.state.selectedChainId, account.index)}`;
        }
        copy.append(name, detail);
        const stateLabel = doc.createElement("em");
        stateLabel.textContent = account.index === selectedIndex ? "\u5F53\u524D" : "\u5207\u6362 / \u6539\u540D";
        button.append(copy, stateLabel);
        list.appendChild(button);
      });
    }
    function renderKeyrings(keyring) {
      keyrings.replaceChildren();
      normalize(keyring).networks.forEach((network) => {
        const item = doc.createElement("span");
        item.className = "wallet-keyring";
        item.textContent = `${network.label} \xB7 ${network.path}`;
        keyrings.appendChild(item);
      });
      keyrings.hidden = false;
    }
    function closeRename() {
      renameIndex = -1;
      nameInput.value = "";
      renameForm.hidden = true;
    }
    function openRename(accountIndex) {
      const account = getAccounts().find((item) => item.index === accountIndex);
      if (!account) return;
      renameIndex = accountIndex;
      nameInput.value = account.name;
      renameForm.hidden = false;
      error.textContent = "";
      nameInput.focus();
      nameInput.select();
    }
    function open() {
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
    function close() {
      dialog.classList.remove("visible");
      dialog.setAttribute("aria-hidden", "true");
      error.textContent = "";
      closeRename();
    }
    async function persist() {
      if (!options.isWalletUnlocked() || !options.state.wallet.vault || !options.state.wallet.masterKey) throw new Error("\u94B1\u5305\u5C1A\u672A\u89E3\u9501\u3002");
      const iv = Uint8Array.from(options.randomBytes(12));
      const now = (/* @__PURE__ */ new Date()).toISOString();
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
    async function select(accountIndex, editName) {
      const current = normalize(options.state.wallet.keyring);
      if (!current.accounts.some((account) => account.index === accountIndex)) return;
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
          error.textContent = "\u5207\u6362\u8D26\u6237\u5931\u8D25\uFF0C\u8BF7\u7A0D\u540E\u91CD\u8BD5\u3002";
          return;
        }
      }
      if (editName) openRename(accountIndex);
    }
    async function add() {
      const current = normalize(options.state.wallet.keyring);
      const nextIndex = current.accounts.reduce((largest, account) => Math.max(largest, account.index), -1) + 1;
      if (nextIndex > 99) {
        error.textContent = "\u6700\u591A\u53EF\u4EE5\u6DFB\u52A0 100 \u4E2A\u8D26\u6237\u3002";
        return;
      }
      options.state.wallet.keyring = normalize({
        ...current,
        account: nextIndex,
        selectedAccount: nextIndex,
        accounts: [...current.accounts, { index: nextIndex, name: `\u8D26\u6237 ${nextIndex}` }]
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
        error.textContent = "\u6DFB\u52A0\u8D26\u6237\u5931\u8D25\uFF0C\u8BF7\u7A0D\u540E\u91CD\u8BD5\u3002";
      } finally {
        addButton.disabled = false;
      }
    }
    async function saveName() {
      const name = nameInput.value.trim();
      if (!name) {
        error.textContent = "\u8D26\u6237\u540D\u79F0\u4E0D\u80FD\u4E3A\u7A7A\u3002";
        return;
      }
      const current = normalize(options.state.wallet.keyring);
      if (!current.accounts.some((account) => account.index === renameIndex)) {
        error.textContent = "\u672A\u627E\u5230\u9700\u8981\u4FEE\u6539\u7684\u8D26\u6237\u3002";
        return;
      }
      options.state.wallet.keyring = normalize({
        ...current,
        accounts: current.accounts.map((account) => account.index === renameIndex ? { ...account, name: name.slice(0, 24) } : account)
      });
      try {
        await persist();
        renderList();
        closeRename();
        options.onChanged();
      } catch {
        options.state.wallet.keyring = current;
        error.textContent = "\u4FDD\u5B58\u8D26\u6237\u540D\u79F0\u5931\u8D25\uFF0C\u8BF7\u7A0D\u540E\u91CD\u8BD5\u3002";
      }
    }
    function bind() {
      closeButton.addEventListener("click", close);
      dialog.addEventListener("click", (event) => {
        if (event.target === dialog) close();
      });
      list.addEventListener("click", (event) => {
        const button = event.target.closest("[data-wallet-account-index]");
        const index = Number(button?.dataset.walletAccountIndex);
        if (Number.isSafeInteger(index) && index >= 0) void select(index, true);
      });
      addButton.addEventListener("click", () => void add());
      renameForm.addEventListener("submit", (event) => {
        event.preventDefault();
        void saveName();
      });
      renameCancel.addEventListener("click", closeRename);
      doc.addEventListener("keydown", (event) => {
        if (event.key === "Escape" && dialog.classList.contains("visible")) close();
      });
    }
    return Object.freeze({ accountPath: accountPath2, bind, close, createDefault, getAccounts, getSelected, getSelectedIndex, normalize, open, persist, renderKeyrings, renderList });
  }

  // clients/shared/src/controllers/wallet-biometrics.ts
  var wallet_biometrics_exports = {};
  __export(wallet_biometrics_exports, {
    createWalletBiometricController: () => createWalletBiometricController
  });
  function requiredElement5(doc, id) {
    const element = doc.getElementById(id);
    if (!element) throw new Error(`missing biometric element: ${id}`);
    return element;
  }
  function createWalletBiometricController(options) {
    const doc = options.document || document;
    const browserWindow = options.window || window;
    const unlockButton = requiredElement5(doc, "wallet-biometric-unlock");
    const disableButton = requiredElement5(doc, "wallet-biometric-button");
    const enableRow = requiredElement5(doc, "wallet-biometric-enable-row");
    const enableCheckbox = requiredElement5(doc, "wallet-biometric-enable");
    const enableLabel = requiredElement5(doc, "wallet-biometric-enable-label");
    const error = requiredElement5(doc, "wallet-lock-error");
    async function refresh() {
      try {
        options.state.wallet.biometric = await options.biometrics.status();
      } catch {
        options.state.wallet.biometric = { available: false, enrolled: false, enabled: false, biometryType: "none", label: "\u751F\u7269\u8BC6\u522B" };
      }
      render2();
    }
    function render2() {
      const biometric = options.state.wallet.biometric;
      const label = biometric.label || "\u751F\u7269\u8BC6\u522B";
      const unlocking = options.state.wallet.mode === "unlock" && Boolean(options.state.wallet.vault);
      unlockButton.hidden = !(unlocking && biometric.available && biometric.enabled);
      unlockButton.textContent = `\u4F7F\u7528 ${label} \u89E3\u9501`;
      enableRow.hidden = !(unlocking && biometric.available && !biometric.enabled);
      enableLabel.textContent = `\u672C\u6B21\u89E3\u9501\u540E\u542F\u7528 ${label} \u5FEB\u6377\u89E3\u9501`;
      disableButton.hidden = !(options.isWalletUnlocked() && biometric.enabled);
      disableButton.textContent = `\u5173\u95ED ${label}`;
    }
    async function unlock() {
      error.textContent = "";
      unlockButton.disabled = true;
      try {
        await options.unlockWithCredential(await options.biometrics.authenticate());
      } catch (caught) {
        error.textContent = caught instanceof Error ? caught.message : "\u751F\u7269\u8BC6\u522B\u9A8C\u8BC1\u5931\u8D25\u3002";
        await refresh();
      } finally {
        unlockButton.disabled = false;
      }
    }
    async function disable2() {
      if (!browserWindow.confirm("\u5173\u95ED\u751F\u7269\u8BC6\u522B\u5FEB\u6377\u89E3\u9501\uFF1F\u4E4B\u540E\u4ECD\u53EF\u4F7F\u7528\u94B1\u5305\u5BC6\u7801\u89E3\u9501\u3002")) return;
      try {
        await options.biometrics.disable();
        await refresh();
      } catch (caught) {
        browserWindow.alert(caught instanceof Error ? caught.message : "\u5173\u95ED\u751F\u7269\u8BC6\u522B\u5FEB\u6377\u89E3\u9501\u5931\u8D25\u3002");
      }
    }
    function shouldEnableAfterUnlock() {
      return !enableRow.hidden && enableCheckbox.checked;
    }
    function resetEnableChoice() {
      enableCheckbox.checked = false;
      enableRow.hidden = true;
      unlockButton.hidden = true;
    }
    function bind() {
      unlockButton.addEventListener("click", () => void unlock());
      disableButton.addEventListener("click", () => void disable2());
    }
    return Object.freeze({ bind, disable: disable2, refresh, render: render2, resetEnableChoice, shouldEnableAfterUnlock, unlock });
  }

  // clients/shared/src/wallet-vault-store/index.ts
  function createWalletVaultStore(browserWindow = window) {
    const databaseName = "web3_wallet";
    const storeName = "vault";
    function platformVault() {
      const vault = browserWindow.AgentWalletPlatform?.secureVault;
      return vault?.isPlatformBacked() ? vault : null;
    }
    async function open() {
      if (!("indexedDB" in browserWindow) || !browserWindow.crypto?.subtle) {
        throw new Error("\u5F53\u524D\u6D4F\u89C8\u5668\u4E0D\u652F\u6301\u672C\u5730\u52A0\u5BC6\u94B1\u5305\u3002");
      }
      return new Promise((resolve, reject) => {
        const request = browserWindow.indexedDB.open(databaseName, 1);
        request.onupgradeneeded = () => {
          const database = request.result;
          if (!database.objectStoreNames.contains(storeName)) database.createObjectStore(storeName, { keyPath: "id" });
        };
        request.onsuccess = () => resolve(request.result);
        request.onerror = () => reject(request.error || new Error("\u65E0\u6CD5\u6253\u5F00\u672C\u5730\u94B1\u5305\u4FDD\u9669\u5E93\u3002"));
      });
    }
    async function load() {
      const nativeVault = platformVault();
      if (nativeVault) return nativeVault.load();
      const database = await open();
      return new Promise((resolve, reject) => {
        const request = database.transaction(storeName, "readonly").objectStore(storeName).get("primary");
        request.onsuccess = () => resolve(request.result || null);
        request.onerror = () => reject(request.error || new Error("\u8BFB\u53D6\u94B1\u5305\u4FDD\u9669\u5E93\u5931\u8D25\u3002"));
      });
    }
    async function save(vault) {
      const nativeVault = platformVault();
      if (nativeVault) return nativeVault.save(vault);
      const database = await open();
      return new Promise((resolve, reject) => {
        const transaction = database.transaction(storeName, "readwrite");
        transaction.objectStore(storeName).put(vault);
        transaction.oncomplete = () => resolve();
        transaction.onerror = () => reject(transaction.error || new Error("\u4FDD\u5B58\u94B1\u5305\u4FDD\u9669\u5E93\u5931\u8D25\u3002"));
        transaction.onabort = () => reject(transaction.error || new Error("\u4FDD\u5B58\u94B1\u5305\u4FDD\u9669\u5E93\u5931\u8D25\u3002"));
      });
    }
    async function remove() {
      const nativeVault = platformVault();
      if (nativeVault) return nativeVault.remove();
      const database = await open();
      return new Promise((resolve, reject) => {
        const transaction = database.transaction(storeName, "readwrite");
        transaction.objectStore(storeName).delete("primary");
        transaction.oncomplete = () => resolve();
        transaction.onerror = () => reject(transaction.error || new Error("\u5220\u9664\u94B1\u5305\u4FDD\u9669\u5E93\u5931\u8D25\u3002"));
        transaction.onabort = () => reject(transaction.error || new Error("\u5220\u9664\u94B1\u5305\u4FDD\u9669\u5E93\u5931\u8D25\u3002"));
      });
    }
    return Object.freeze({ load, open, remove, save });
  }

  // clients/shared/src/controllers/wallet-session.ts
  var wallet_session_exports = {};
  __export(wallet_session_exports, {
    createWalletSessionController: () => createWalletSessionController
  });
  function createWalletSessionController(options) {
    const doc = options.document || document;
    const browserWindow = options.window || window;
    function isUnlocked() {
      return Boolean(options.state.wallet.masterKey && typeof options.state.wallet.mnemonic === "string");
    }
    function lock() {
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
    function refresh() {
      if (!isUnlocked()) return;
      options.state.wallet.lastActivity = Date.now();
      browserWindow.clearTimeout(options.state.wallet.idleTimer);
      options.state.wallet.idleTimer = browserWindow.setTimeout(lock, options.timeoutMs);
    }
    function start() {
      refresh();
    }
    function bind() {
      ["pointerdown", "keydown", "touchstart"].forEach((eventName) => {
        doc.addEventListener(eventName, refresh, { passive: true });
      });
      doc.addEventListener("visibilitychange", () => {
        if (doc.hidden) lock();
      });
    }
    return Object.freeze({ bind, isUnlocked, lock, refresh, start });
  }

  // clients/shared/src/controllers/wallet-secret-export.ts
  var wallet_secret_export_exports = {};
  __export(wallet_secret_export_exports, {
    createWalletSecretExportController: () => createWalletSecretExportController
  });
  function requiredElement6(doc, id) {
    const element = doc.getElementById(id);
    if (!element) throw new Error(`missing secret export element: ${id}`);
    return element;
  }
  function createWalletSecretExportController(options) {
    const doc = options.document || document;
    const browserWindow = options.window || window;
    const revealTimeoutMs = options.revealTimeoutMs || 6e4;
    const createStep = requiredElement6(doc, "wallet-create-step");
    const choiceStep = requiredElement6(doc, "wallet-choice-step");
    const importStep = requiredElement6(doc, "wallet-import-step");
    const recovery = requiredElement6(doc, "wallet-recovery");
    const recoveryConfirmRow = requiredElement6(doc, "wallet-recovery-confirm-row");
    const recoveryCopy = requiredElement6(doc, "wallet-recovery-copy");
    const recoveryTitle = requiredElement6(doc, "wallet-recovery-title");
    const recoveryDescription = requiredElement6(doc, "wallet-recovery-description");
    const privateKeyExport = requiredElement6(doc, "wallet-private-key-export");
    const privateKeyCopy = requiredElement6(doc, "wallet-private-key-copy");
    const privateKeyChain = requiredElement6(doc, "wallet-private-key-chain");
    const privateKeyAddress = requiredElement6(doc, "wallet-private-key-address");
    const privateKeyPath = requiredElement6(doc, "wallet-private-key-path");
    const privateKeyFormat = requiredElement6(doc, "wallet-private-key-format");
    const privateKeyValue = requiredElement6(doc, "wallet-private-key-value");
    const lockCopy = requiredElement6(doc, "wallet-lock-copy");
    const lockCancel = requiredElement6(doc, "wallet-lock-cancel");
    const lockSubmit = requiredElement6(doc, "wallet-lock-submit");
    const lockError = requiredElement6(doc, "wallet-lock-error");
    const password = requiredElement6(doc, "wallet-password");
    let revealTimer = 0;
    let copyTimer = 0;
    function requestMnemonic() {
      if (options.state.wallet.vault) options.openUnlock("export-mnemonic");
    }
    function requestPrivateKey() {
      if (options.state.wallet.vault) options.openUnlock("export-private-key");
    }
    async function decryptPayload(passwordValue) {
      const vault = options.state.wallet.vault;
      if (!vault) throw new Error("\u672A\u627E\u5230\u672C\u5730\u94B1\u5305\u4FDD\u9669\u5E93\u3002");
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
      if (!payload || typeof payload.mnemonic !== "string") throw new Error("\u94B1\u5305\u4FDD\u9669\u5E93\u5185\u5BB9\u65E0\u6548\u3002");
      return payload;
    }
    function scheduleClose() {
      browserWindow.clearTimeout(revealTimer);
      revealTimer = browserWindow.setTimeout(() => {
        if (options.state.wallet.mode === "export-display") options.closeUnlock();
      }, revealTimeoutMs);
    }
    async function exportMnemonic(passwordValue) {
      if (!options.state.wallet.vault) {
        lockError.textContent = "\u672A\u627E\u5230\u672C\u5730\u94B1\u5305\u4FDD\u9669\u5E93\u3002";
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
        recoveryTitle.textContent = "\u5BFC\u51FA\u6062\u590D\u77ED\u8BED";
        recoveryDescription.textContent = "\u8BF7\u5728\u5B89\u5168\u73AF\u5883\u4E2D\u79BB\u7EBF\u5907\u4EFD\u3002\u52A9\u8BB0\u8BCD\u5C06\u5728\u5173\u95ED\u7A97\u53E3\u3001\u5207\u6362\u540E\u53F0\u6216 60 \u79D2\u540E\u4ECE\u9875\u9762\u6E05\u9664\u3002";
        lockCopy.textContent = "\u4EFB\u4F55\u83B7\u5F97\u52A9\u8BB0\u8BCD\u7684\u4EBA\u90FD\u53EF\u4EE5\u63A7\u5236\u94B1\u5305\u8D44\u4EA7\uFF0C\u8BF7\u52FF\u622A\u56FE\u3001\u8054\u7F51\u4FDD\u5B58\u6216\u53D1\u9001\u7ED9\u4ED6\u4EBA\u3002";
        lockCancel.textContent = "\u5173\u95ED";
        lockSubmit.textContent = "\u5B8C\u6210";
        lockError.textContent = "";
        password.value = "";
        scheduleClose();
      } catch {
        lockError.textContent = "\u65E0\u6CD5\u5BFC\u51FA\u52A9\u8BB0\u8BCD\uFF0C\u8BF7\u68C0\u67E5\u94B1\u5305\u5BC6\u7801\u3002";
      } finally {
        lockSubmit.disabled = false;
      }
    }
    async function exportPrivateKey(passwordValue) {
      if (!options.state.wallet.vault) {
        lockError.textContent = "\u672A\u627E\u5230\u672C\u5730\u94B1\u5305\u4FDD\u9669\u5E93\u3002";
        return;
      }
      lockSubmit.disabled = true;
      let exportSeed = null;
      try {
        const payload = await decryptPayload(passwordValue);
        exportSeed = await options.deriveSeed(payload.mnemonic);
        const keyring = options.normalizeKeyring(payload.keyring);
        const account = keyring.accounts.find((item) => item.index === keyring.selectedAccount) || keyring.accounts[0];
        const exported = await options.derivePrivateKey(exportSeed, options.state.selectedChainId, account.index);
        const address = options.deriveAddress(exportSeed, options.state.selectedChainId, account.index);
        if (!address || !exported || typeof exported.privateKey !== "string" || typeof exported.path !== "string") {
          throw new Error("\u79C1\u94A5\u6D3E\u751F\u7ED3\u679C\u65E0\u6548\u3002");
        }
        options.state.wallet.mode = "export-display";
        createStep.hidden = true;
        choiceStep.hidden = true;
        importStep.hidden = true;
        recovery.hidden = true;
        privateKeyExport.hidden = false;
        privateKeyChain.textContent = `${options.getSelectedChain().label} \xB7 ${account.name} (#${account.index})`;
        privateKeyAddress.textContent = address;
        privateKeyPath.textContent = exported.path;
        privateKeyFormat.textContent = exported.format || "hex";
        privateKeyValue.textContent = exported.privateKey;
        lockCopy.textContent = "\u79C1\u94A5\u53EA\u5728\u672C\u8BBE\u5907\u4E34\u65F6\u663E\u793A\u3002\u4EFB\u4F55\u83B7\u5F97\u5B83\u7684\u4EBA\u90FD\u80FD\u63A7\u5236\u5F53\u524D\u5730\u5740\uFF0C\u8BF7\u52FF\u622A\u56FE\u3001\u8054\u7F51\u4FDD\u5B58\u6216\u53D1\u9001\u7ED9\u4ED6\u4EBA\u3002";
        lockCancel.textContent = "\u5173\u95ED";
        lockSubmit.textContent = "\u5B8C\u6210";
        lockError.textContent = "";
        password.value = "";
        scheduleClose();
      } catch (caught) {
        lockError.textContent = caught instanceof Error && caught.message === "\u5F53\u524D\u9875\u9762\u7F3A\u5C11\u672C\u5730\u79C1\u94A5\u6D3E\u751F\u80FD\u529B\u3002" ? caught.message : "\u65E0\u6CD5\u5BFC\u51FA\u5F53\u524D\u5730\u5740\u79C1\u94A5\uFF0C\u8BF7\u68C0\u67E5\u94B1\u5305\u5BC6\u7801\u3002";
      } finally {
        if (exportSeed) exportSeed.fill(0);
        lockSubmit.disabled = false;
      }
    }
    async function copyPrivateKey() {
      const value = privateKeyValue.textContent.trim();
      if (!value) {
        lockError.textContent = "\u5F53\u524D\u6CA1\u6709\u53EF\u590D\u5236\u7684\u79C1\u94A5\u3002";
        return;
      }
      try {
        await options.copyText(value);
        lockError.textContent = "";
        privateKeyCopy.textContent = "copied";
      } catch {
        lockError.textContent = "\u590D\u5236\u79C1\u94A5\u5931\u8D25\u3002";
        privateKeyCopy.textContent = "\u5931\u8D25";
      }
      browserWindow.clearTimeout(copyTimer);
      copyTimer = browserWindow.setTimeout(() => {
        privateKeyCopy.textContent = "copy";
      }, 1600);
    }
    function reset() {
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
    function bind() {
      privateKeyCopy.addEventListener("click", () => void copyPrivateKey());
    }
    return Object.freeze({ bind, copyPrivateKey, exportMnemonic, exportPrivateKey, requestMnemonic, requestPrivateKey, reset });
  }

  // clients/shared/src/controllers/wallet-vault.ts
  var wallet_vault_exports = {};
  __export(wallet_vault_exports, {
    createWalletVaultController: () => createWalletVaultController
  });
  function requiredElement7(doc, id) {
    const element = doc.getElementById(id);
    if (!element) throw new Error(`missing wallet vault element: ${id}`);
    return element;
  }
  function createWalletVaultController(options) {
    const doc = options.document || document;
    const browserWindow = options.window || window;
    const modal = requiredElement7(doc, "wallet-lock");
    const form = requiredElement7(doc, "wallet-lock-form");
    const password = requiredElement7(doc, "wallet-password");
    const passwordConfirm = requiredElement7(doc, "wallet-password-confirm");
    const passwordConfirmField = requiredElement7(doc, "wallet-password-confirm-field");
    const newPassword = requiredElement7(doc, "wallet-new-password");
    const newPasswordField = requiredElement7(doc, "wallet-new-password-field");
    const passwordLabel = requiredElement7(doc, "wallet-password-label");
    const error = requiredElement7(doc, "wallet-lock-error");
    const copy = requiredElement7(doc, "wallet-lock-copy");
    const cancel = requiredElement7(doc, "wallet-lock-cancel");
    const submit = requiredElement7(doc, "wallet-lock-submit");
    const createStep = requiredElement7(doc, "wallet-create-step");
    const choiceStep = requiredElement7(doc, "wallet-choice-step");
    const generateChoice = requiredElement7(doc, "wallet-generate-choice");
    const importChoice = requiredElement7(doc, "wallet-import-choice");
    const recovery = requiredElement7(doc, "wallet-recovery");
    const mnemonic = requiredElement7(doc, "wallet-mnemonic");
    const recoveryConfirm = requiredElement7(doc, "wallet-recovery-confirm");
    const recoveryConfirmRow = requiredElement7(doc, "wallet-recovery-confirm-row");
    const recoveryCopy = requiredElement7(doc, "wallet-recovery-copy");
    const recoveryTitle = requiredElement7(doc, "wallet-recovery-title");
    const recoveryDescription = requiredElement7(doc, "wallet-recovery-description");
    const importStep = requiredElement7(doc, "wallet-import-step");
    const importText = requiredElement7(doc, "wallet-import-text");
    const importFill = requiredElement7(doc, "wallet-import-fill");
    const importMnemonic = requiredElement7(doc, "wallet-import-mnemonic");
    const importConfirm = requiredElement7(doc, "wallet-import-confirm");
    const accessButton = requiredElement7(doc, "wallet-access-button");
    const changePasswordButton = requiredElement7(doc, "wallet-change-password-button");
    let recoveryCopyTimer = 0;
    function renderRecoveryPhrase(value) {
      mnemonic.replaceChildren();
      value.split(" ").forEach((word, index) => {
        const item = doc.createElement("span");
        item.textContent = `${String(index + 1).padStart(2, "0")}. ${word}`;
        mnemonic.appendChild(item);
      });
    }
    function renderImportInputs(words) {
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
    function open(mode) {
      options.state.wallet.mode = mode;
      modal.classList.add("visible");
      modal.setAttribute("aria-hidden", "false");
      error.textContent = "";
      password.value = "";
      passwordConfirm.value = "";
      newPassword.value = "";
      recoveryConfirm.checked = false;
      recoveryConfirmRow.hidden = false;
      recoveryTitle.textContent = "\u5907\u4EFD\u6062\u590D\u77ED\u8BED";
      recoveryDescription.textContent = "\u8BF7\u79BB\u7EBF\u6284\u5199\u8FD9 12 \u4E2A\u5355\u8BCD\u3002\u5B83\u53EA\u663E\u793A\u8FD9\u4E00\u6B21\uFF1B\u786E\u8BA4\u5B8C\u6210\u540E\u5373\u53EF\u521B\u5EFA\u65B0\u94B1\u5305\u3002";
      recovery.hidden = true;
      options.resetSecretExport();
      importStep.hidden = true;
      createStep.hidden = false;
      choiceStep.hidden = true;
      importConfirm.checked = false;
      importText.value = "";
      recoveryCopy.textContent = "copy";
      recoveryCopy.hidden = true;
      cancel.textContent = "\u53D6\u6D88";
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
      passwordLabel.textContent = creating ? "\u8BBE\u7F6E\u94B1\u5305\u5BC6\u7801" : changingPassword ? "\u5F53\u524D\u94B1\u5305\u5BC6\u7801" : "\u94B1\u5305\u5BC6\u7801";
      passwordConfirm.placeholder = changingPassword ? "\u518D\u6B21\u8F93\u5165\u65B0\u94B1\u5305\u5BC6\u7801" : "\u518D\u6B21\u8F93\u5165\u94B1\u5305\u5BC6\u7801";
      copy.textContent = exportingSecret ? "\u4E3A\u4FDD\u62A4\u8D44\u4EA7\uFF0C\u8BF7\u518D\u6B21\u9A8C\u8BC1\u5F53\u524D\u94B1\u5305\u5BC6\u7801\u3002\u9A8C\u8BC1\u6210\u529F\u540E\u654F\u611F\u5BC6\u94A5\u53EA\u5728\u672C\u8BBE\u5907\u4E34\u65F6\u663E\u793A\u3002" : changingPassword ? "\u9A8C\u8BC1\u5F53\u524D\u5BC6\u7801\u540E\uFF0C\u5C06\u4F7F\u7528\u65B0\u7684\u968F\u673A\u76D0\u548C IV \u91CD\u65B0\u52A0\u5BC6\u672C\u5730\u94B1\u5305\u4FDD\u9669\u5E93\u3002" : creating ? "\u521B\u5EFA\u540E\uFF0C\u6062\u590D\u77ED\u8BED\u53EA\u4F1A\u5728\u6B64\u8BBE\u5907\u663E\u793A\u4E00\u6B21\u3002\u4FDD\u9669\u5E93\u4EC5\u4FDD\u5B58 AES-GCM \u52A0\u5BC6\u6570\u636E\u5230\u672C\u673A\u5B89\u5168\u5B58\u50A8\u3002" : "\u8F93\u5165\u672C\u5730\u94B1\u5305\u5BC6\u7801\u4EE5\u6D3E\u751F\u5BC6\u94A5\u5E76\u89E3\u5BC6\u4FDD\u9669\u5E93\u3002\u5BC6\u7801\u548C\u6062\u590D\u77ED\u8BED\u4E0D\u4F1A\u53D1\u9001\u5230\u670D\u52A1\u5668\u3002";
      submit.textContent = exportingSecret ? "\u9A8C\u8BC1\u5E76\u663E\u793A" : changingPassword ? "\u786E\u8BA4\u66F4\u6362" : creating ? "\u4E0B\u4E00\u6B65" : "\u89E3\u9501";
      options.renderBiometrics();
      password.focus();
    }
    function close(clearPendingAction = true) {
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
      recoveryTitle.textContent = "\u5907\u4EFD\u6062\u590D\u77ED\u8BED";
      recoveryDescription.textContent = "\u8BF7\u79BB\u7EBF\u6284\u5199\u8FD9 12 \u4E2A\u5355\u8BCD\u3002\u5B83\u53EA\u663E\u793A\u8FD9\u4E00\u6B21\uFF1B\u786E\u8BA4\u5B8C\u6210\u540E\u5373\u53EF\u521B\u5EFA\u65B0\u94B1\u5305\u3002";
      cancel.textContent = "\u53D6\u6D88";
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
    async function initialize() {
      try {
        options.state.wallet.vault = await options.vaultStore.load();
        await options.refreshBiometrics();
        options.renderVault();
      } catch (caught) {
        options.renderVault(caught instanceof Error ? caught.message : "\u94B1\u5305\u4FDD\u9669\u5E93\u521D\u59CB\u5316\u5931\u8D25");
      }
    }
    async function prepareAccess() {
      options.state.wallet.pendingAction = "";
      if (options.isUnlocked()) {
        options.renderVault();
        return;
      }
      try {
        options.state.wallet.vault = await options.vaultStore.load();
        await options.refreshBiometrics();
      } catch (caught) {
        options.renderVault(caught instanceof Error ? caught.message : "\u65E0\u6CD5\u8BFB\u53D6\u672C\u5730\u94B1\u5305\u4FDD\u9669\u5E93");
        return;
      }
      open(options.state.wallet.vault ? "unlock" : "create");
    }
    async function unlock(passwordValue) {
      if (!options.state.wallet.vault) {
        error.textContent = "\u672A\u627E\u5230\u672C\u5730\u94B1\u5305\u4FDD\u9669\u5E93\u3002";
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
            biometricWarning = caught instanceof Error ? caught.message : "\u751F\u7269\u8BC6\u522B\u5FEB\u6377\u89E3\u9501\u542F\u7528\u5931\u8D25\u3002";
          }
        }
        const pendingAction = options.state.wallet.pendingAction;
        close(false);
        options.startSession();
        options.renderVault();
        await options.onPendingAction(pendingAction);
        if (biometricWarning) browserWindow.alert(`\u94B1\u5305\u5DF2\u89E3\u9501\uFF0C\u4F46${biometricWarning}`);
      } catch {
        error.textContent = "\u65E0\u6CD5\u89E3\u9501\u94B1\u5305\uFF0C\u8BF7\u68C0\u67E5\u5BC6\u7801\u3002";
      } finally {
        submit.disabled = false;
        password.value = "";
      }
    }
    async function changePassword(currentPassword, nextPassword) {
      if (!options.state.wallet.vault) {
        error.textContent = "\u672A\u627E\u5230\u672C\u5730\u94B1\u5305\u4FDD\u9669\u5E93\u3002";
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
        await options.disableBiometrics().catch(() => void 0);
        await options.refreshBiometrics();
        close();
        options.startSession();
        options.renderVault();
        browserWindow.alert("\u94B1\u5305\u5BC6\u7801\u5DF2\u66F4\u6362\u3002\u4E0B\u6B21\u89E3\u9501\u8BF7\u4F7F\u7528\u65B0\u5BC6\u7801\u3002");
      } catch {
        error.textContent = "\u65E0\u6CD5\u66F4\u6362\u5BC6\u7801\uFF0C\u8BF7\u68C0\u67E5\u5F53\u524D\u5BC6\u7801\u3002";
      } finally {
        submit.disabled = false;
      }
    }
    async function finalizeCreation(mnemonicValue) {
      const pendingPassword = options.state.wallet.pendingPassword;
      if (!pendingPassword) throw new Error("\u94B1\u5305\u5BC6\u7801\u72B6\u6001\u5DF2\u5931\u6548\uFF0C\u8BF7\u91CD\u65B0\u5F00\u59CB\u521B\u5EFA\u3002");
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
    async function readImportedMnemonic() {
      const words = Array.from(importMnemonic.querySelectorAll("input")).map((input) => input.value.trim().toLowerCase()).filter(Boolean);
      if (words.length !== 12) throw new Error("\u8BF7\u5B8C\u6574\u586B\u5199 12 \u4E2A\u52A9\u8BB0\u8BCD\u3002");
      const normalized = await options.parseMnemonicWords(words.join(" "));
      options.state.wallet.pendingMnemonic = normalized.join(" ");
      return options.state.wallet.pendingMnemonic;
    }
    async function submitCurrentMode() {
      const passwordValue = password.value;
      const mode = options.state.wallet.mode;
      if (["unlock", "export-mnemonic", "export-private-key"].includes(mode) && !options.isValidPassword(passwordValue)) {
        error.textContent = "\u5BC6\u7801\u5FC5\u987B\u662F 8-16 \u4F4D\uFF0C\u5E76\u4E14\u4EC5\u5305\u542B\u5B57\u6BCD\u3001\u6570\u5B57\u548C !@#$%^&*.";
        return;
      }
      if (mode === "unlock") return unlock(passwordValue);
      if (mode === "export-mnemonic") return options.exportMnemonic(passwordValue);
      if (mode === "export-private-key") return options.exportPrivateKey(passwordValue);
      if (mode === "export-display") return close();
      if (mode === "change-password") {
        const nextPassword = newPassword.value;
        if (!options.isValidPassword(passwordValue)) error.textContent = "\u5F53\u524D\u5BC6\u7801\u683C\u5F0F\u4E0D\u6B63\u786E\u3002";
        else if (!options.isValidPassword(nextPassword)) error.textContent = "\u65B0\u5BC6\u7801\u5FC5\u987B\u662F 8-16 \u4F4D\uFF0C\u5E76\u4E14\u4EC5\u5305\u542B\u5B57\u6BCD\u3001\u6570\u5B57\u548C !@#$%^&*.";
        else if (nextPassword === passwordValue) error.textContent = "\u65B0\u5BC6\u7801\u4E0D\u80FD\u4E0E\u5F53\u524D\u5BC6\u7801\u76F8\u540C\u3002";
        else if (passwordConfirm.value !== nextPassword) error.textContent = "\u4E24\u6B21\u8F93\u5165\u7684\u65B0\u5BC6\u7801\u4E0D\u4E00\u81F4\u3002";
        else await changePassword(passwordValue, nextPassword);
        return;
      }
      if (mode === "create") {
        if (!options.isValidPassword(passwordValue)) error.textContent = "\u5BC6\u7801\u5FC5\u987B\u662F 8-16 \u4F4D\uFF0C\u5E76\u4E14\u4EC5\u5305\u542B\u5B57\u6BCD\u3001\u6570\u5B57\u548C !@#$%^&*.";
        else if (passwordConfirm.value !== passwordValue) error.textContent = "\u4E24\u6B21\u8F93\u5165\u7684\u94B1\u5305\u5BC6\u7801\u4E0D\u4E00\u81F4\u3002";
        else {
          options.state.wallet.pendingPassword = passwordValue;
          options.state.wallet.mode = "choice";
          password.value = "";
          passwordConfirm.value = "";
          error.textContent = "";
          createStep.hidden = true;
          choiceStep.hidden = false;
          submit.hidden = true;
          copy.textContent = "\u8BF7\u9009\u62E9\u751F\u6210\u65B0\u7684\u52A9\u8BB0\u8BCD\uFF0C\u6216\u8005\u5BFC\u5165\u5DF2\u6709\u7684 12 \u4E2A\u52A9\u8BB0\u8BCD\u3002";
        }
        return;
      }
      if (mode === "backup") {
        if (!recoveryConfirm.checked || !options.state.wallet.pendingMnemonic) {
          error.textContent = "\u8BF7\u786E\u8BA4\u5DF2\u79BB\u7EBF\u5907\u4EFD\u6062\u590D\u77ED\u8BED\u3002";
          return;
        }
        try {
          await finalizeCreation(options.state.wallet.pendingMnemonic);
        } catch (caught) {
          error.textContent = caught instanceof Error ? caught.message : "\u94B1\u5305\u4FDD\u9669\u5E93\u521B\u5EFA\u5931\u8D25";
        }
        return;
      }
      if (mode === "import") {
        if (!importConfirm.checked) {
          error.textContent = "\u8BF7\u786E\u8BA4\u4F60\u7406\u89E3\u6062\u590D\u77ED\u8BED\u7684\u4FDD\u7BA1\u98CE\u9669\u3002";
          return;
        }
        try {
          await finalizeCreation(await readImportedMnemonic());
        } catch (caught) {
          error.textContent = caught instanceof Error ? caught.message : "\u5BFC\u5165\u52A9\u8BB0\u8BCD\u5931\u8D25";
        }
      }
    }
    async function startGeneratedMnemonic() {
      try {
        options.state.wallet.pendingMnemonic = await options.generateMnemonic();
        renderRecoveryPhrase(options.state.wallet.pendingMnemonic);
        options.state.wallet.mode = "backup";
        choiceStep.hidden = true;
        recovery.hidden = false;
        importStep.hidden = true;
        recoveryCopy.hidden = false;
        submit.hidden = false;
        submit.textContent = "\u786E\u8BA4\u5907\u4EFD\u5E76\u521B\u5EFA";
        error.textContent = "";
        copy.textContent = "\u8BF7\u7ACB\u5373\u79BB\u7EBF\u5907\u4EFD\u8FD9 12 \u4E2A\u52A9\u8BB0\u8BCD\u3002\u5BC6\u7801\u6846\u5DF2\u9690\u85CF\uFF0C\u63A5\u4E0B\u6765\u53EA\u9700\u786E\u8BA4\u5907\u4EFD\u5E76\u521B\u5EFA\u94B1\u5305\u3002";
      } catch (caught) {
        error.textContent = caught instanceof Error ? caught.message : "\u65E0\u6CD5\u751F\u6210\u6062\u590D\u77ED\u8BED";
      }
    }
    function startImportMnemonic() {
      options.state.wallet.mode = "import";
      options.state.wallet.pendingMnemonic = null;
      choiceStep.hidden = true;
      recovery.hidden = true;
      importStep.hidden = false;
      submit.hidden = false;
      submit.textContent = "\u786E\u8BA4\u5BFC\u5165\u5E76\u521B\u5EFA";
      error.textContent = "";
      copy.textContent = "\u7C98\u8D34\u6216\u9010\u4E2A\u586B\u5199\u5DF2\u6709\u7684 12 \u4E2A\u52A9\u8BB0\u8BCD\u3002\u53BB\u6389\u9996\u5C3E\u6362\u884C\u540E\u4F1A\u6309\u7A7A\u683C\u5207\u5206\u5E76\u81EA\u52A8\u586B\u5145\u5230\u4E0B\u65B9\u8F93\u5165\u6846\u3002";
      renderImportInputs(Array(12).fill(""));
      importText.focus();
    }
    async function hydrateImportedMnemonic() {
      try {
        renderImportInputs(await options.parseMnemonicWords(importText.value));
        error.textContent = "";
      } catch (caught) {
        error.textContent = caught instanceof Error ? caught.message : "\u5BFC\u5165\u52A9\u8BB0\u8BCD\u5931\u8D25";
      }
    }
    async function copyRecoveryPhrase() {
      if (!options.state.wallet.pendingMnemonic) {
        error.textContent = "\u8BF7\u5148\u751F\u6210\u6062\u590D\u77ED\u8BED\u3002";
        return;
      }
      try {
        await options.copyText(options.state.wallet.pendingMnemonic.split(" ").join(" "));
        error.textContent = "";
        recoveryCopy.textContent = "copied";
        browserWindow.clearTimeout(recoveryCopyTimer);
        recoveryCopyTimer = browserWindow.setTimeout(() => {
          recoveryCopy.textContent = "copy";
        }, 1600);
      } catch (caught) {
        error.textContent = caught instanceof Error ? caught.message : "\u590D\u5236\u6062\u590D\u77ED\u8BED\u5931\u8D25\u3002";
      }
    }
    function bind() {
      form.addEventListener("submit", (event) => {
        event.preventDefault();
        void submitCurrentMode();
      });
      cancel.addEventListener("click", () => close());
      password.addEventListener("input", () => {
        error.textContent = "";
      });
      password.addEventListener("keydown", (event) => {
        if (event.key === "Enter") {
          event.preventDefault();
          void submitCurrentMode();
        }
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
      bind,
      changePassword,
      close,
      copyRecoveryPhrase,
      finalizeCreation,
      hydrateImportedMnemonic,
      initialize,
      open,
      prepareAccess,
      readImportedMnemonic,
      renderImportInputs,
      renderRecoveryPhrase,
      startGeneratedMnemonic,
      startImportMnemonic,
      submit: submitCurrentMode,
      unlock
    });
  }

  // clients/shared/src/global.ts
  window.AgentWalletCore = Object.freeze(wallet_core_exports);
  window.AgentWalletAPI = createAPIClient();
  window.AgentWalletUI = Object.freeze(ui_components_exports);
  window.AgentWalletBiometrics = Object.freeze(biometric_auth_exports);
  window.AgentWalletAccounts = Object.freeze(wallet_accounts_exports);
  window.AgentWalletHistory = Object.freeze(wallet_history_exports);
  window.AgentWalletTransactions = Object.freeze(transactions_exports);
  window.AgentWalletState = Object.freeze(app_state_exports);
  window.AgentWalletMarket = Object.freeze(market_exports);
  window.AgentWalletHistoryController = Object.freeze(wallet_history_exports2);
  window.AgentWalletQRCode = Object.freeze(qr_code_exports);
  window.AgentWalletAssetTransferController = Object.freeze(asset_transfer_exports);
  window.AgentWalletAssetBalanceController = Object.freeze(asset_balance_exports);
  window.AgentWalletAccountController = Object.freeze(wallet_accounts_exports2);
  window.AgentWalletBiometricController = Object.freeze(wallet_biometrics_exports);
  window.AgentWalletVaultStore = createWalletVaultStore();
  window.AgentWalletSessionController = Object.freeze(wallet_session_exports);
  window.AgentWalletSecretExportController = Object.freeze(wallet_secret_export_exports);
  window.AgentWalletVaultController = Object.freeze(wallet_vault_exports);
})();
