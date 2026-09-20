export interface Candle {
  timestamp: string;
  open: number;
  high: number;
  low: number;
  close: number;
}

export function parseCandleLine(line: string): Candle | null {
  const match = line.match(/^- ([^ ]+) open=([0-9.]+) high=([0-9.]+) low=([0-9.]+) close=([0-9.]+)/);
  if (!match) return null;
  return { timestamp: match[1], open: Number(match[2]), high: Number(match[3]), low: Number(match[4]), close: Number(match[5]) };
}

export function parseObservation(text: unknown) {
  if (typeof text !== "string" || !text.includes("Candles:")) return null;
  const lines = text.split("\n").map((line) => line.trim()).filter(Boolean);
  const candlesIndex = lines.indexOf("Candles:");
  if (candlesIndex <= 0) return null;
  const meta: Record<string, string> = {};
  lines.slice(1, candlesIndex).forEach((line) => {
    if (!line.startsWith("- ")) return;
    const separator = line.indexOf(":");
    if (separator <= 0) return;
    meta[line.slice(2, separator).trim()] = line.slice(separator + 1).trim();
  });
  const candles = lines.slice(candlesIndex + 1).map(parseCandleLine).filter((candle): candle is Candle => Boolean(candle));
  if (candles.length === 0) return null;
  return {
    title: lines[0].replace(/:$/, ""), mode: meta.mode || "",
    vsCurrency: (meta.vs_currency || "usd").toUpperCase(), interval: meta.interval || "",
    totalCandles: Number(meta.total_candles || candles.length),
    candlesReturned: Number(meta.candles_returned || candles.length),
    assetID: meta.asset_id || "", tokenAddress: meta.token_address || "", chain: meta.chain || "",
    note: meta.note || "", source: meta.source || "", candles
  };
}

export function calculateChange(candles: Candle[]) {
  const first = candles[0];
  const last = candles[candles.length - 1];
  if (!first || !last || first.open === 0) return { className: "flat", label: "0.00%", percent: 0 };
  const percent = ((last.close - first.open) / first.open) * 100;
  return {
    className: percent > 0 ? "positive" : percent < 0 ? "negative" : "flat",
    label: `${percent > 0 ? "+" : ""}${percent.toFixed(2)}%`,
    percent
  };
}

export function formatChartPrice(value: number): string {
  if (value >= 1000) return value.toFixed(2);
  if (value >= 1) return value.toFixed(4);
  if (value >= 0.01) return value.toFixed(6);
  return value.toFixed(8);
}

export function formatChartAxisPrice(value: number): string {
  if (value >= 1000) return value.toFixed(0);
  if (value >= 1) return value.toFixed(2);
  return value.toFixed(4);
}

export function stats(candles: Candle[]) {
  if (candles.length === 0) return [];
  return [
    { label: "H", value: formatChartPrice(Math.max(...candles.map((candle) => candle.high))) },
    { label: "L", value: formatChartPrice(Math.min(...candles.map((candle) => candle.low))) },
    { label: "O", value: formatChartPrice(candles[0].open) },
    { label: "C", value: formatChartPrice(candles[candles.length - 1].close) }
  ];
}

export function formatTimeLabel(timestamp: string): string {
  const date = new Date(timestamp);
  if (Number.isNaN(date.getTime())) return timestamp;
  return `${String(date.getUTCMonth() + 1).padStart(2, "0")}/${String(date.getUTCDate()).padStart(2, "0")} ${String(date.getUTCHours()).padStart(2, "0")}:${String(date.getUTCMinutes()).padStart(2, "0")}`;
}

export function toChartY(value: number, minPrice: number, range: number, top: number, height: number): number {
  return top + ((minPrice + range - value) / range) * height;
}

export function formatUSDPrice(value: unknown): string {
  const amount = Number(value || 0);
  if (amount >= 1000) return "$" + amount.toLocaleString(undefined, { maximumFractionDigits: 2, minimumFractionDigits: 2 });
  if (amount >= 1) return "$" + amount.toFixed(2);
  if (amount >= 0.01) return "$" + amount.toFixed(4);
  return "$" + amount.toFixed(6);
}

export function formatPercent(value: unknown): string {
  const amount = Number(value || 0);
  return `${amount > 0 ? "+" : ""}${amount.toFixed(2)}%`;
}

export function formatCompact(value: unknown, currency = false): string {
  const amount = Number(value || 0);
  if (!amount) return currency ? "$0.00" : "0";
  return (currency ? "$" : "") + amount.toLocaleString(undefined, { notation: "compact", maximumFractionDigits: 2 });
}

export function formatUpdatedTime(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return `${String(date.getHours()).padStart(2, "0")}:${String(date.getMinutes()).padStart(2, "0")}:${String(date.getSeconds()).padStart(2, "0")}`;
}
