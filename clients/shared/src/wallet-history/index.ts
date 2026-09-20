export interface WalletHistoryItem {
  id: string;
  hash: string;
  chainId: string;
  type: string;
  direction: string;
  timestamp: string;
  from: string;
  to: string;
  tokenSymbol: string;
  amount: string;
  fee: string;
  feeSymbol: string;
  status: string;
  [key: string]: unknown;
}

export function normalizeItem(input: unknown): WalletHistoryItem {
  const item = input && typeof input === "object" ? input as Record<string, unknown> : {};
  return {
    id: String(item.id || item.hash || ""),
    hash: String(item.hash || ""),
    chainId: String(item.chain_id || item.chainId || ""),
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

export function recordKey(item: WalletHistoryItem): string {
  return [item.chainId, item.hash, item.direction, item.tokenSymbol, item.amount].join(":").toLowerCase();
}

export function merge(remoteItems: unknown[], localItems: unknown[], limit = 200): WalletHistoryItem[] {
  const merged = new Map<string, WalletHistoryItem>();
  [...remoteItems, ...localItems].map(normalizeItem).forEach((item) => {
    const key = recordKey(item);
    if (item.hash && !merged.has(key)) merged.set(key, item);
  });
  return Array.from(merged.values()).slice(0, limit);
}

export function typeLabel(type: string): string {
  return ({ send: "发送", receive: "接收", swap: "交易", bridge: "跨链" } as Record<string, string>)[type] || "交易";
}

export function dateLabel(timestamp: string): string {
  const date = new Date(timestamp);
  if (Number.isNaN(date.getTime())) return "日期未知";
  return `${date.getFullYear()}/${String(date.getMonth() + 1).padStart(2, "0")}/${String(date.getDate()).padStart(2, "0")}`;
}

export function filter(
  items: unknown[],
  filters: { chainId?: string; type?: string }
): WalletHistoryItem[] {
  const chainId = filters.chainId || "all";
  const type = filters.type || "all";
  return items.map(normalizeItem)
    .filter((item) => chainId === "all" || item.chainId === chainId)
    .filter((item) => type === "all" || item.type === type)
    .sort((left, right) => new Date(right.timestamp).getTime() - new Date(left.timestamp).getTime());
}
