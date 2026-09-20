export function createInitialState(sessionId: string, messagesByTab: Record<string, unknown>) {
  return {
    activeTab: "home",
    busyTab: "",
    selectedChainId: "ethereum",
    sessionId,
    wallet: {
      vault: null, masterKey: null, mnemonic: null, seed: null, addresses: {}, keyring: null,
      mode: "", pendingAction: "", pendingPassword: "", pendingMnemonic: null,
      idleTimer: 0, lastActivity: 0,
      biometric: { available: false, enrolled: false, enabled: false, biometryType: "none", label: "生物识别" }
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
