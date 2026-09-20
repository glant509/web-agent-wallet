export function maskAddress(address: string, leading = 4, trailing = 4): string {
  const value = String(address || "");
  if (!value) return "";
  if (value.startsWith("0x") && value.length > 2 + leading + trailing) {
    return `0x${value.slice(2, 2 + leading)}***${value.slice(-trailing)}`;
  }
  if (value.length <= leading + trailing + 3) return value;
  return `${value.slice(0, leading)}***${value.slice(-trailing)}`;
}

export function formatUSD(value: unknown): string {
  const amount = Number(value);
  if (!Number.isFinite(amount)) return "$0.00";
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: "USD",
    minimumFractionDigits: 2,
    maximumFractionDigits: amount >= 1 ? 2 : 6
  }).format(amount);
}

export function setVisible(element: HTMLElement | null, visible: boolean): void {
  if (element) element.hidden = !visible;
}

export function setBusy(button: HTMLButtonElement | null, busy: boolean, busyLabel?: string): () => void {
  if (!button) return () => undefined;
  const previousText = button.textContent || "";
  const previousDisabled = button.disabled;
  button.disabled = busy;
  if (busy && busyLabel) button.textContent = busyLabel;
  return () => {
    button.disabled = previousDisabled;
    button.textContent = previousText;
  };
}

export function createLabeledValueRow(label: string, value: string, className: string): HTMLDivElement {
  const row = document.createElement("div");
  row.className = className;
  const name = document.createElement("span");
  name.textContent = label;
  const detail = document.createElement("strong");
  detail.textContent = value;
  row.append(name, detail);
  return row;
}
