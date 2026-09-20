type AnyRecord = Record<string, any>;

interface ChainInfo {
  short: string;
}

interface AssetBalanceOptions {
  state: AnyRecord;
  supportedChainIds: string[];
  getSelectedChain: () => ChainInfo;
  isWalletUnlocked: () => boolean;
  deriveAddress: (chainId: string) => string;
  request: (resource: string, options?: RequestInit, traceId?: string) => Promise<Response>;
  document?: Document;
}

function requiredElement(doc: Document, id: string): HTMLElement {
  const element = doc.getElementById(id);
  if (!element) throw new Error(`missing asset balance element: ${id}`);
  return element;
}

export function normalizeChainId(chainId: string): string {
  return chainId === "bnb" ? "bsc" : chainId;
}

export function derivationChainId(chainId: string): string {
  return chainId === "bsc" ? "bnb" : chainId;
}

export function createAssetBalanceController(options: AssetBalanceOptions) {
  const doc = options.document || document;
  const status = requiredElement(doc, "asset-balance-status");
  const total = requiredElement(doc, "asset-balance-total");
  const list = requiredElement(doc, "asset-balance-list");
  const chainBadge = requiredElement(doc, "asset-balance-chain-badge");

  function currentItems(): AnyRecord[] {
    if (options.state.assetBalances.selectedChainID !== normalizeChainId(options.state.selectedChainId)) return [];
    return Array.isArray(options.state.assetBalances.items) ? options.state.assetBalances.items : [];
  }

  function summary(items: AnyRecord[]): string {
    return items.length ? `已加载 ${items.length} 项余额` : "--";
  }

  function render(): void {
    chainBadge.textContent = options.getSelectedChain().short;
    if (!options.isWalletUnlocked()) {
      status.textContent = "解锁钱包后加载余额";
      total.textContent = "--";
      list.replaceChildren();
      return;
    }
    if (!options.supportedChainIds.includes(normalizeChainId(options.state.selectedChainId))) {
      status.textContent = "当前余额看板暂支持 ETH、Base、Arbitrum、OP、BNB、Polygon、Solana、Avalanche";
      total.textContent = summary(currentItems());
      list.replaceChildren();
      return;
    }
    if (options.state.assetBalances.loading) {
      status.textContent = "余额加载中...";
      total.textContent = summary(currentItems());
      list.replaceChildren();
      return;
    }
    if (options.state.assetBalances.error) {
      status.textContent = options.state.assetBalances.error;
      total.textContent = "--";
      list.replaceChildren();
      return;
    }
    status.textContent = options.state.assetBalances.updatedAt
      ? `已更新 · ${new Date(options.state.assetBalances.updatedAt).toLocaleTimeString()}`
      : "已加载";
    const items = currentItems();
    total.textContent = summary(items);
    list.replaceChildren();
    items.forEach((item) => {
      const row = doc.createElement("div");
      row.className = "asset-balance-item";
      const token = doc.createElement("div");
      token.className = "asset-balance-token";
      const tokenTitle = doc.createElement("strong");
      tokenTitle.textContent = `${item.symbol}${item.is_native ? " · Native" : ""}`;
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

  async function ensureAddress(chainId: string): Promise<string> {
    if (!options.isWalletUnlocked()) return "";
    const normalized = normalizeChainId(chainId);
    const cached = options.state.wallet.addresses[normalized];
    if (typeof cached === "string" && cached) return cached;
    const derived = options.deriveAddress(derivationChainId(normalized));
    options.state.wallet.addresses[normalized] = derived;
    return derived;
  }

  function clear(chainId = ""): void {
    const requestToken = options.state.assetBalances.requestToken;
    options.state.assetBalances = {
      loading: false, error: "", totalUSD: 0, items: [],
      selectedChainID: chainId, updatedAt: "", requestToken
    };
  }

  async function refresh(): Promise<void> {
    render();
    if (!options.isWalletUnlocked()) {
      clear();
      render();
      return;
    }
    const selectedChainID = normalizeChainId(options.state.selectedChainId);
    if (!options.supportedChainIds.includes(selectedChainID)) {
      clear(selectedChainID);
      render();
      return;
    }
    const requestToken = ++options.state.assetBalances.requestToken;
    Object.assign(options.state.assetBalances, {
      loading: true, error: "", items: [], selectedChainID, updatedAt: ""
    });
    render();
    try {
      const address = await ensureAddress(selectedChainID);
      if (!address) throw new Error("当前链地址派生失败");
      const response = await options.request("/v1/asset/portfolio", {
        method: "POST", headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ selected_chain_id: selectedChainID, addresses: { [selectedChainID]: address } })
      });
      const payload = await response.json() as AnyRecord;
      if (!response.ok) throw new Error(payload.error || "余额查询失败");
      if (requestToken !== options.state.assetBalances.requestToken || normalizeChainId(options.state.selectedChainId) !== selectedChainID) return;
      const selected = payload && typeof payload.selected === "object" ? payload.selected : {};
      const walletChains = Array.isArray(payload?.wallet?.chains) ? payload.wallet.chains : [];
      const fallback = walletChains.find((item: AnyRecord) => item?.chain_id === selectedChainID) || walletChains[0] || {};
      Object.assign(options.state.assetBalances, {
        loading: false, error: "", totalUSD: 0,
        items: Array.isArray(selected.items) ? selected.items : (Array.isArray(fallback.items) ? fallback.items : []),
        selectedChainID: selected.chain_id || fallback.chain_id || selectedChainID,
        updatedAt: payload.updated_at || ""
      });
    } catch (caught) {
      if (requestToken !== options.state.assetBalances.requestToken || normalizeChainId(options.state.selectedChainId) !== selectedChainID) return;
      Object.assign(options.state.assetBalances, {
        loading: false, error: caught instanceof Error ? caught.message : "余额查询失败",
        totalUSD: 0, items: [], selectedChainID, updatedAt: ""
      });
    }
    render();
  }

  function selectChain(): void {
    clear(normalizeChainId(options.state.selectedChainId));
    void refresh();
  }

  return Object.freeze({ currentItems, ensureAddress, normalizeChainId, refresh, render, selectChain });
}
