import * as QRCode from "../qr-code/index";
import * as Transactions from "../transactions/index";
import { createLabeledValueRow } from "../ui-components/index";

type AnyRecord = Record<string, any>;

interface ChainInfo {
  id: string;
  label: string;
  short: string;
}

interface AssetTransferOptions {
  state: AnyRecord;
  tokenCatalog: Record<string, string[][]>;
  supportedChainIds: string[];
  getCurrentBalanceItems: () => AnyRecord[];
  getSelectedChain: () => ChainInfo;
  isWalletUnlocked: () => boolean;
  ensureSelectedAddress: () => Promise<string>;
  getSelectedAddress: () => string;
  getSelectedAccount: () => { index: number; name: string };
  getSelectedAccountIndex: () => number;
  request: (resource: string, options: RequestInit, traceId: string) => Promise<Response>;
  createTraceId: () => string;
  copyText: (value: string) => Promise<void>;
  signerAddress: (seed: unknown, accountIndex: number) => string;
  signTransaction: (seed: unknown, accountIndex: number, transaction: AnyRecord) => Promise<string>;
  onLocked: (mode: "send" | "receive", hasVault: boolean) => void;
  onBeforeOpen: () => void;
  onBroadcast: (plan: AnyRecord, hash: string, recipient: string) => void;
  onComplete: () => void;
  document?: Document;
  window?: Window;
}

function requiredElement<T extends HTMLElement>(doc: Document, id: string): T {
  const element = doc.getElementById(id);
  if (!element) throw new Error(`missing asset transfer element: ${id}`);
  return element as T;
}

export function createAssetTransferController(options: AssetTransferOptions) {
  const doc = options.document || document;
  const browserWindow = options.window || window;
  const views = requiredElement(doc, "views");
  const dialog = requiredElement(doc, "asset-action-dialog");
  const closeButton = requiredElement<HTMLButtonElement>(doc, "asset-action-close");
  const title = requiredElement(doc, "asset-action-title");
  const description = requiredElement(doc, "asset-action-description");
  const tokenPicker = requiredElement(doc, "asset-token-picker");
  const tokenList = requiredElement(doc, "asset-token-list");
  const sendPage = requiredElement(doc, "asset-send-page");
  const receivePage = requiredElement(doc, "asset-receive-page");
  const sendSelectedToken = requiredElement(doc, "asset-send-selected-token");
  const receiveSelectedToken = requiredElement(doc, "asset-receive-selected-token");
  const sendChangeToken = requiredElement<HTMLButtonElement>(doc, "asset-send-change-token");
  const receiveChangeToken = requiredElement<HTMLButtonElement>(doc, "asset-receive-change-token");
  const sendForm = requiredElement<HTMLFormElement>(doc, "asset-send-form");
  const sendAddress = requiredElement<HTMLInputElement>(doc, "asset-send-address");
  const sendAmount = requiredElement<HTMLInputElement>(doc, "asset-send-amount");
  const sendMax = requiredElement<HTMLButtonElement>(doc, "asset-send-max");
  const sendNext = requiredElement<HTMLButtonElement>(doc, "asset-send-next");
  const sendFeePreview = requiredElement(doc, "asset-send-fee-preview");
  const receiveQR = requiredElement(doc, "asset-receive-qr");
  const receiveAddress = requiredElement(doc, "asset-receive-address");
  const receiveCopy = requiredElement<HTMLButtonElement>(doc, "asset-receive-copy");
  const actionError = requiredElement(doc, "asset-action-error");
  const receiveError = requiredElement(doc, "asset-receive-error");
  const review = requiredElement(doc, "asset-transaction-review");
  const rows = requiredElement(doc, "asset-transaction-rows");
  const reviewBack = requiredElement<HTMLButtonElement>(doc, "asset-transaction-back");
  const confirm = requiredElement<HTMLButtonElement>(doc, "asset-transaction-confirm");
  const transactionError = requiredElement(doc, "asset-transaction-error");
  const success = requiredElement(doc, "asset-transaction-success");
  const transactionHash = requiredElement(doc, "asset-transaction-hash");
  let returnTimer = 0;

  function tokens(): AnyRecord[] {
    const balanceItems = options.getCurrentBalanceItems();
    if (balanceItems.length) {
      return balanceItems.map((item) => ({
        symbol: String(item.symbol || "TOKEN"), name: String(item.name || item.symbol || "Token"),
        balance: String(item.balance || "0"), balanceRaw: String(item.balance_raw || "0"),
        tokenAddress: String(item.token_address || ""), decimals: Number.isSafeInteger(Number(item.decimals)) ? Number(item.decimals) : 18,
        isNative: Boolean(item.is_native)
      }));
    }
    return (options.tokenCatalog[options.state.selectedChainId] || []).map(([symbol, name], index) => ({
      symbol, name, balance: "--", balanceRaw: "0", tokenAddress: "", decimals: 18, isNative: index === 0
    }));
  }

  function renderTokenPicker(): void {
    const chain = options.getSelectedChain();
    title.textContent = options.state.assetAction.mode === "receive" ? "Receive" : "Send";
    description.textContent = `选择 ${chain.label} 支持的 Token`;
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
      symbol.textContent = `${token.symbol}${token.isNative ? " · Native" : ""}`;
      const name = doc.createElement("span");
      name.textContent = token.name;
      copy.append(symbol, name);
      const balance = doc.createElement("em");
      balance.textContent = unavailable ? "余额加载后可用" : (token.balance === "--" ? "选择" : token.balance);
      button.append(icon, copy, balance);
      tokenList.appendChild(button);
    });
    tokenPicker.hidden = false;
    sendPage.hidden = true;
    receivePage.hidden = true;
  }

  function open(mode: string): void {
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
    receiveCopy.textContent = "复制接收地址";
    sendNext.disabled = false;
    sendNext.textContent = "下一步";
    sendFeePreview.textContent = "手续费将在下一步通过当前网络实时预估。";
    review.hidden = true;
    success.hidden = true;
    renderTokenPicker();
    dialog.classList.add("visible");
    dialog.setAttribute("aria-hidden", "false");
  }

  function close(): void {
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

  async function selectToken(index: number): Promise<void> {
    const token = tokens()[index];
    if (!token) return;
    options.state.assetAction.token = token;
    tokenPicker.hidden = true;
    actionError.textContent = "";
    receiveError.textContent = "";
    const isReceive = options.state.assetAction.mode === "receive";
    sendPage.hidden = isReceive;
    receivePage.hidden = !isReceive;
    sendSelectedToken.textContent = `${token.symbol} · ${options.getSelectedChain().label}`;
    receiveSelectedToken.textContent = `${token.symbol} · ${options.getSelectedChain().label}`;
    sendMax.hidden = !token.isNative;
    options.state.assetAction.sendMax = false;
    options.state.assetAction.plan = null;
    sendFeePreview.textContent = token.isNative
      ? "点击最大值会自动预留实时预估的最高网络手续费。"
      : `Token 转账手续费将使用 ${options.getSelectedChain().short} 单独支付。`;
    title.textContent = `${isReceive ? "Receive" : "Send"} ${token.symbol}`;
    description.textContent = isReceive ? "扫描二维码或复制地址接收资产" : "填写接收地址与转账数量";
    if (isReceive) {
      const address = await options.ensureSelectedAddress();
      if (!address) {
        receiveError.textContent = "当前链接收地址派生失败。";
        return;
      }
      receiveAddress.textContent = address;
      QRCode.render(receiveQR, address, doc);
    } else if (!options.supportedChainIds.includes(options.state.selectedChainId)) {
      sendForm.hidden = true;
      actionError.textContent = "当前链尚未配置安全的交易构建与广播能力。现阶段 Send 支持 Ethereum、Base、Arbitrum、Optimism、BNB、Polygon 和 Avalanche。";
    } else {
      sendForm.hidden = false;
      browserWindow.requestAnimationFrame(() => sendAddress.focus());
    }
  }

  function validate(): boolean {
    if (!Transactions.isValidRecipient(options.state.selectedChainId, sendAddress.value.trim())) {
      actionError.textContent = `请输入有效的 ${options.getSelectedChain().label} 接收地址。`;
      return false;
    }
    if (!Transactions.isValidAmount(sendAmount.value.trim())) {
      actionError.textContent = "请输入大于 0 的转账数量。";
      return false;
    }
    return true;
  }

  async function requestPlan(sendMaximum: boolean): Promise<AnyRecord> {
    const token = options.state.assetAction.token;
    if (!token || !options.supportedChainIds.includes(options.state.selectedChainId)) throw new Error("当前链暂不支持发送。 ");
    const from = await options.ensureSelectedAddress();
    if (!from) throw new Error("发送地址派生失败。 ");
    let recipient = sendAddress.value.trim();
    if (sendMaximum && !Transactions.isValidRecipient(options.state.selectedChainId, recipient)) recipient = from;
    const response = await options.request("/v1/wallet/evm/prepare", {
      method: "POST", headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ chain_id: options.state.selectedChainId, from, to: recipient,
        token_address: token.isNative ? "" : token.tokenAddress,
        amount: sendMaximum ? "" : sendAmount.value.trim(), send_max: sendMaximum })
    }, options.state.assetAction.traceId);
    const payload = await response.json() as AnyRecord;
    if (!response.ok) throw new Error(payload.error || "交易手续费预估失败。 ");
    return payload;
  }

  async function applyMaximum(): Promise<void> {
    if (!options.state.assetAction.token?.isNative || options.state.assetAction.busy) return;
    options.state.assetAction.busy = true;
    sendMax.disabled = true;
    sendNext.disabled = true;
    actionError.textContent = "";
    sendFeePreview.textContent = "正在根据当前网络计算可转出的最大值...";
    try {
      const plan = await requestPlan(true);
      options.state.assetAction.plan = plan;
      options.state.assetAction.sendMax = true;
      sendAmount.value = plan.amount;
      sendFeePreview.textContent = `已预留最高约 ${plan.estimated_fee} ${plan.native_symbol} 网络手续费。`;
    } catch (caught) {
      options.state.assetAction.sendMax = false;
      options.state.assetAction.plan = null;
      sendFeePreview.textContent = "无法计算最大值。";
      actionError.textContent = caught instanceof Error ? caught.message : "最大值计算失败。";
    } finally {
      options.state.assetAction.busy = false;
      sendMax.disabled = false;
      sendNext.disabled = false;
    }
  }

  function renderReview(plan: AnyRecord): void {
    const account = options.getSelectedAccount();
    rows.replaceChildren(
      createLabeledValueRow("当前网络", `${options.getSelectedChain().label} · Chain ID ${BigInt(plan.network_chain_id)}`, "asset-transaction-row"),
      createLabeledValueRow("发送资产", `${plan.amount} ${plan.token_symbol}`, "asset-transaction-row"),
      createLabeledValueRow("预计手续费", `最高约 ${plan.estimated_fee} ${plan.native_symbol}`, "asset-transaction-row"),
      createLabeledValueRow("账户编号", `${account.name} · #${account.index}`, "asset-transaction-row"),
      createLabeledValueRow("发送地址", options.getSelectedAddress(), "asset-transaction-row"),
      createLabeledValueRow("收款地址", sendAddress.value.trim(), "asset-transaction-row")
    );
    sendPage.hidden = true;
    success.hidden = true;
    review.hidden = false;
    transactionError.textContent = "";
    title.textContent = "确认交易";
    description.textContent = "请仔细核对网络、资产、手续费和地址";
  }

  async function prepareReview(): Promise<void> {
    if (!validate() || options.state.assetAction.busy) return;
    options.state.assetAction.busy = true;
    sendNext.disabled = true;
    sendNext.textContent = "预估手续费...";
    actionError.textContent = "";
    try {
      const plan = await requestPlan(options.state.assetAction.sendMax);
      options.state.assetAction.plan = plan;
      sendAmount.value = plan.amount;
      renderReview(plan);
    } catch (caught) {
      options.state.assetAction.plan = null;
      actionError.textContent = caught instanceof Error ? caught.message : "交易准备失败。";
    } finally {
      options.state.assetAction.busy = false;
      sendNext.disabled = false;
      sendNext.textContent = "下一步";
    }
  }

  async function confirmAndBroadcast(): Promise<void> {
    const plan = options.state.assetAction.plan;
    if (!plan || options.state.assetAction.busy || !options.isWalletUnlocked()) return;
    options.state.assetAction.busy = true;
    confirm.disabled = true;
    confirm.textContent = "本地签名并广播中...";
    transactionError.textContent = "";
    try {
      const accountIndex = options.getSelectedAccountIndex();
      const expectedAddress = options.getSelectedAddress().toLowerCase();
      if (expectedAddress !== options.signerAddress(options.state.wallet.seed, accountIndex).toLowerCase()) {
        throw new Error("签名账户与当前发送地址不一致，交易已取消。 ");
      }
      let rawTransaction = await options.signTransaction(options.state.wallet.seed, accountIndex, Transactions.forSigning(plan));
      const response = await options.request("/v1/wallet/evm/broadcast", {
        method: "POST", headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ chain_id: plan.chain_id, raw_transaction: rawTransaction })
      }, options.state.assetAction.traceId);
      rawTransaction = "";
      const payload = await response.json() as AnyRecord;
      if (!response.ok) throw new Error(payload.error || "交易广播失败。 ");
      if (payload.transaction_hash) options.onBroadcast(plan, payload.transaction_hash, sendAddress.value.trim());
      review.hidden = true;
      success.hidden = false;
      transactionHash.textContent = payload.transaction_hash || "已提交";
      title.textContent = "交易已广播";
      description.textContent = `${options.getSelectedChain().label} 网络已接收签名交易`;
      returnTimer = browserWindow.setTimeout(() => {
        returnTimer = 0;
        close();
        options.onComplete();
      }, 2200);
    } catch (caught) {
      transactionError.textContent = caught instanceof Error ? caught.message : "交易签名或广播失败。";
    } finally {
      options.state.assetAction.busy = false;
      confirm.disabled = false;
      confirm.textContent = "确认并广播";
    }
  }

  async function copyReceiveAddress(): Promise<void> {
    const address = receiveAddress.textContent?.trim() || "";
    if (!address) return;
    try {
      await options.copyText(address);
      receiveCopy.textContent = "已复制";
    } catch {
      receiveCopy.textContent = "复制失败";
    }
    browserWindow.setTimeout(() => { receiveCopy.textContent = "复制接收地址"; }, 1600);
  }

  function bind(): void {
    views.addEventListener("click", (event) => {
      const button = (event.target as Element).closest<HTMLElement>("[data-asset-action]");
      if (!button) return;
      const action = button.dataset.assetAction || "";
      if (action === "send" || action === "receive") open(action);
    });
    closeButton.addEventListener("click", close);
    dialog.addEventListener("click", (event) => { if (event.target === dialog) close(); });
    tokenList.addEventListener("click", (event) => {
      const button = (event.target as Element).closest<HTMLElement>("[data-asset-token-index]");
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
    sendForm.addEventListener("submit", (event) => { event.preventDefault(); void prepareReview(); });
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
      description.textContent = "填写接收地址与转账数量";
    });
    receiveCopy.addEventListener("click", () => void copyReceiveAddress());
    doc.addEventListener("keydown", (event) => {
      if (event.key === "Escape" && dialog.classList.contains("visible")) close();
    });
  }

  return Object.freeze({ bind, close, open, renderTokenPicker });
}
