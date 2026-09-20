import * as history from "../wallet-history/index";

interface ChainOption {
  id: string;
  label: string;
}

interface WalletHistoryState {
  items: history.WalletHistoryItem[];
  chainId: string;
  type: string;
  loading: boolean;
  error: string;
  requestToken: number;
  traceId: string;
}

interface ControllerState {
  selectedChainId: string;
  wallet: { vault: unknown; pendingAction: string };
  walletHistory: WalletHistoryState;
}

interface WalletHistoryControllerOptions {
  state: ControllerState;
  chains: ChainOption[];
  supportedChainIds: string[];
  maskAddress: (address: string) => string;
  isWalletUnlocked: () => boolean;
  getSelectedAddress: () => string;
  ensureAddress: (chainId: string) => Promise<string>;
  normalizeChainId: (chainId: string) => string;
  request: (resource: string, options: RequestInit, traceId: string) => Promise<Response>;
  createTraceId: () => string;
  onLocked: (hasVault: boolean) => void;
  onBeforeOpen: () => void;
  document?: Document;
  window?: Window;
}

interface BroadcastRecord {
  hash: string;
  chainId: string;
  accountIndex: number;
  accountName: string;
  from: string;
  to: string;
  tokenSymbol: string;
  amount: string;
  fee: string;
  feeSymbol: string;
}

const storageKey = "web3_wallet_transaction_history_v1";
const typeOptions = [
  { value: "all", label: "全部类型" },
  { value: "send", label: "发送" },
  { value: "receive", label: "接收" },
  { value: "swap", label: "交易" },
  { value: "bridge", label: "跨链" }
];
const explorerURLs: Record<string, string> = {
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

function requiredElement<T extends HTMLElement>(doc: Document, id: string): T {
  const element = doc.getElementById(id);
  if (!element) throw new Error(`missing wallet history element: ${id}`);
  return element as T;
}

export function createWalletHistoryController(options: WalletHistoryControllerOptions) {
  const doc = options.document || document;
  const browserWindow = options.window || window;
  const dialog = requiredElement(doc, "wallet-history-dialog");
  const back = requiredElement<HTMLButtonElement>(doc, "wallet-history-back");
  const chainFilter = requiredElement(doc, "wallet-history-chain-filter");
  const chainTrigger = requiredElement<HTMLButtonElement>(doc, "wallet-history-chain-trigger");
  const chainLabel = requiredElement(doc, "wallet-history-chain-label");
  const chainMenu = requiredElement<HTMLElement>(doc, "wallet-history-chain-menu");
  const typeFilter = requiredElement(doc, "wallet-history-type-filter");
  const typeTrigger = requiredElement<HTMLButtonElement>(doc, "wallet-history-type-trigger");
  const typeLabelElement = requiredElement(doc, "wallet-history-type-label");
  const typeMenu = requiredElement<HTMLElement>(doc, "wallet-history-type-menu");
  const list = requiredElement(doc, "wallet-history-list");
  const empty = requiredElement(doc, "wallet-history-empty");
  const loading = requiredElement(doc, "wallet-history-loading");
  const error = requiredElement(doc, "wallet-history-error");
  const explorer = requiredElement<HTMLButtonElement>(doc, "wallet-history-explorer");

  function load(): history.WalletHistoryItem[] {
    try {
      const raw = browserWindow.localStorage.getItem(storageKey);
      const parsed: unknown = raw ? JSON.parse(raw) : [];
      return Array.isArray(parsed)
        ? parsed.filter((item) => item && typeof item === "object" && typeof (item as { hash?: unknown }).hash === "string")
          .map(history.normalizeItem).slice(0, 200)
        : [];
    } catch {
      return [];
    }
  }

  function persist(): void {
    try {
      browserWindow.localStorage.setItem(storageKey, JSON.stringify(options.state.walletHistory.items.slice(0, 200)));
    } catch {
      // History is a convenience cache; broadcasting must never depend on it.
    }
  }

  function recordBroadcast(record: BroadcastRecord): void {
    const item = history.normalizeItem({
      id: `${record.hash}:local`, hash: record.hash, chainId: record.chainId,
      type: "send", direction: "send", timestamp: new Date().toISOString(),
      accountIndex: record.accountIndex, accountName: record.accountName,
      from: record.from, to: record.to, tokenSymbol: record.tokenSymbol,
      amount: record.amount, fee: record.fee, feeSymbol: record.feeSymbol, status: "broadcast"
    });
    options.state.walletHistory.items = [
      item,
      ...options.state.walletHistory.items.filter((existing) => existing.hash !== record.hash)
    ].slice(0, 200);
    persist();
  }

  function explorerURL(chainId: string, kind: "tx" | "address", value: string): string {
    const base = explorerURLs[chainId];
    if (!base || !value) return "";
    if (kind === "tx") return `${base}/tx/${encodeURIComponent(value)}`;
    return `${base}/${chainId === "solana" ? "account" : "address"}/${encodeURIComponent(value)}`;
  }

  function openExternal(url: string): void {
    if (!url) return;
    const opened = browserWindow.open(url, "_blank", "noopener,noreferrer");
    if (opened) opened.opener = null;
  }

  function explorerTarget(): string {
    const walletHistory = options.state.walletHistory;
    const chainId = walletHistory.chainId === "all"
      ? options.state.selectedChainId
      : (walletHistory.chainId || options.state.selectedChainId);
    let address = options.isWalletUnlocked() && chainId === options.state.selectedChainId
      ? options.getSelectedAddress()
      : "";
    if (!address) {
      const record = walletHistory.items.find((item) => item.chainId === chainId);
      address = record ? (record.from || record.to || "") : "";
    }
    return explorerURL(chainId, "address", address);
  }

  function renderFilterMenu(menu: HTMLElement, kind: string, entries: Array<{ value: string; label: string }>, selected: string): void {
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

  function initializeFilters(): void {
    const walletHistory = options.state.walletHistory;
    renderFilterMenu(chainMenu, "chain", [
      { value: "all", label: "全部网络" },
      ...options.chains.map((chain) => ({ value: chain.id, label: chain.label }))
    ], walletHistory.chainId || options.state.selectedChainId);
    renderFilterMenu(typeMenu, "type", typeOptions, walletHistory.type || "all");
    const selectedChain = options.chains.find((chain) => chain.id === walletHistory.chainId);
    chainLabel.textContent = walletHistory.chainId === "all" ? "全部网络" : (selectedChain?.label || "Ethereum");
    typeLabelElement.textContent = typeOptions.find((entry) => entry.value === walletHistory.type)?.label || "全部类型";
  }

  function closeMenus(): void {
    [[chainFilter, chainTrigger, chainMenu], [typeFilter, typeTrigger, typeMenu]].forEach(([filter, trigger, menu]) => {
      filter.classList.remove("open");
      trigger.setAttribute("aria-expanded", "false");
      (menu as HTMLElement).hidden = true;
    });
  }

  function toggleMenu(kind: "chain" | "type"): void {
    const target = kind === "chain"
      ? [chainFilter, chainTrigger, chainMenu]
      : [typeFilter, typeTrigger, typeMenu];
    const shouldOpen = (target[2] as HTMLElement).hidden;
    closeMenus();
    if (shouldOpen) {
      target[0].classList.add("open");
      target[1].setAttribute("aria-expanded", "true");
      (target[2] as HTMLElement).hidden = false;
    }
  }

  function createItem(item: history.WalletHistoryItem): HTMLButtonElement {
    const button = doc.createElement("button");
    button.type = "button";
    button.className = "wallet-history-item";
    button.dataset.walletHistoryHash = item.hash;
    button.dataset.walletHistoryChain = item.chainId;
    button.setAttribute("aria-label", `${history.typeLabel(item.type)} ${item.amount} ${item.tokenSymbol}`);
    const icon = doc.createElement("span");
    icon.className = "wallet-history-token-icon";
    icon.textContent = String(item.tokenSymbol || "TX").slice(0, 5);
    const copy = doc.createElement("span");
    copy.className = "wallet-history-copy";
    const title = doc.createElement("strong");
    title.textContent = history.typeLabel(item.type);
    const detail = doc.createElement("span");
    const incoming = item.direction === "receive";
    detail.textContent = `${incoming ? "来自 " : "至 "}${options.maskAddress((incoming ? item.from : item.to) || "地址未知")}`;
    copy.append(title, detail);
    const value = doc.createElement("span");
    value.className = "wallet-history-value";
    const amount = doc.createElement("strong");
    amount.className = incoming ? "incoming" : "";
    amount.textContent = `${incoming ? "+" : "-"}${item.amount} ${item.tokenSymbol}`;
    const fee = doc.createElement("span");
    fee.textContent = incoming
      ? (item.status === "broadcast" ? "已广播" : "")
      : (item.fee ? `手续费 ${item.fee} ${item.feeSymbol}` : (item.status === "broadcast" ? "已广播" : ""));
    value.append(amount, fee);
    button.append(icon, copy, value);
    return button;
  }

  function render(): void {
    const walletHistory = options.state.walletHistory;
    const items = history.filter(walletHistory.items, { chainId: walletHistory.chainId, type: walletHistory.type });
    list.replaceChildren();
    loading.hidden = !walletHistory.loading;
    error.hidden = !walletHistory.error;
    error.textContent = walletHistory.error;
    empty.hidden = walletHistory.loading || items.length > 0;
    const groups = new Map<string, history.WalletHistoryItem[]>();
    items.forEach((item) => {
      const date = history.dateLabel(item.timestamp);
      groups.set(date, [...(groups.get(date) || []), item]);
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

  async function refresh(): Promise<void> {
    if (!options.isWalletUnlocked()) return;
    const walletHistory = options.state.walletHistory;
    const requestToken = ++walletHistory.requestToken;
    walletHistory.loading = true;
    walletHistory.error = "";
    render();
    const requested = walletHistory.chainId === "all" ? options.supportedChainIds : [walletHistory.chainId];
    const chainIds = requested.filter((chainId) => options.supportedChainIds.includes(chainId));
    if (chainIds.length === 0) {
      walletHistory.loading = false;
      walletHistory.error = "当前网络暂不支持在钱包内同步交易历史，请使用下方区块浏览器查看。";
      render();
      return;
    }
    const results = await Promise.allSettled(chainIds.map(async (chainId) => {
      const normalizedChainId = options.normalizeChainId(chainId);
      const address = await options.ensureAddress(normalizedChainId);
      if (!address) throw new Error("无法派生当前账户地址");
      const response = await options.request("/v1/wallet/evm/history", {
        method: "POST", headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ chain_id: normalizedChainId, address })
      }, walletHistory.traceId);
      const payload = await response.json().catch(() => ({})) as { error?: string; items?: unknown[] };
      if (!response.ok) throw new Error(payload.error || "链上历史查询失败");
      return Array.isArray(payload.items) ? payload.items : [];
    }));
    if (requestToken !== walletHistory.requestToken) return;
    const remoteItems = results.flatMap((result) => result.status === "fulfilled" ? result.value : []);
    walletHistory.items = history.merge(remoteItems, load());
    const failures = results
      .map((result, index) => ({ result, chainId: chainIds[index] }))
      .filter((entry): entry is { result: PromiseRejectedResult; chainId: string } => entry.result.status === "rejected");
    failures.forEach(({ result, chainId }) => {
      console.warn("[wallet-history] chain sync failed", {
        traceId: walletHistory.traceId,
        chainId,
        reason: result.reason instanceof Error ? result.reason.message : String(result.reason || "unknown error")
      });
    });
    walletHistory.error = failures.length === results.length
      ? "暂时无法同步链上历史，已显示本地记录。请稍后重试或前往区块浏览器查看。"
      : (failures.length ? "部分网络同步失败，其余链上记录已显示。" : "");
    walletHistory.loading = false;
    render();
  }

  function open(): void {
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
    render();
    dialog.classList.add("visible");
    dialog.setAttribute("aria-hidden", "false");
    void refresh();
  }

  function close(): void {
    options.state.walletHistory.requestToken += 1;
    options.state.walletHistory.loading = false;
    closeMenus();
    dialog.classList.remove("visible");
    dialog.setAttribute("aria-hidden", "true");
  }

  function bind(): void {
    back.addEventListener("click", close);
    chainTrigger.addEventListener("click", () => toggleMenu("chain"));
    typeTrigger.addEventListener("click", () => toggleMenu("type"));
    dialog.addEventListener("click", (event) => {
      if (event.target === dialog) return close();
      const target = event.target as Element;
      const option = target.closest<HTMLElement>("[data-history-filter-kind]");
      if (option) {
        const kind = option.dataset.historyFilterKind;
        const value = option.dataset.historyFilterValue || "all";
        if (kind === "chain") options.state.walletHistory.chainId = value;
        if (kind === "type") options.state.walletHistory.type = value;
        closeMenus();
        initializeFilters();
        render();
        if (kind === "chain") void refresh();
        return;
      }
      if (!chainFilter.contains(target) && !typeFilter.contains(target)) closeMenus();
    });
    list.addEventListener("click", (event) => {
      const item = (event.target as Element).closest<HTMLElement>("[data-wallet-history-hash]");
      if (item) openExternal(explorerURL(item.dataset.walletHistoryChain || "", "tx", item.dataset.walletHistoryHash || ""));
    });
    explorer.addEventListener("click", () => openExternal(explorerTarget()));
    doc.addEventListener("keydown", (event) => {
      if (event.key !== "Escape" || !dialog.classList.contains("visible")) return;
      if (!chainMenu.hidden || !typeMenu.hidden) closeMenus(); else close();
    });
  }

  return Object.freeze({ bind, close, load, open, persist, recordBroadcast, refresh, render });
}
