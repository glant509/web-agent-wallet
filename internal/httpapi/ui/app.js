const chatTabs = ["home"];
    const state = window.AgentWalletState.createInitialState(
      window.sessionStorage.getItem("agent-session-id") || "",
      loadMessagesByTab()
    );
    const WalletVaultStore = window.AgentWalletVaultStore;

    function randomTraceHex(byteLength) {
      return window.AgentWalletCore.randomHex(byteLength);
    }

    function createBrowserTraceId() {
      return randomTraceHex(16);
    }

    function tracedFetch(resource, options = {}, traceId = "") {
      return window.AgentWalletAPI.request(resource, options, traceId);
    }

    const views = Array.from(document.querySelectorAll(".view"));
    const tabs = Array.from(document.querySelectorAll(".tabbar button"));
    const appShellEl = document.getElementById("app-shell");
    const viewsEl = document.getElementById("views");
    const composerWrapEl = document.getElementById("composer-wrap");
    const formEl = document.getElementById("chat-form");
    const inputEl = document.getElementById("chat-input");
    const sendButton = document.getElementById("send-button");
    const aiAvailabilityEl = document.getElementById("home-ai-availability");
    let aiChatEnabled = false;
    const pageTitleEl = document.getElementById("page-title");
    const pageSubtitleEl = document.getElementById("page-subtitle");
    const pageStatusEl = document.getElementById("page-status");
    const walletHeaderAddressEl = document.getElementById("wallet-header-address");
    const walletHeaderCopyEl = document.getElementById("wallet-header-copy");
    const walletAccountMenuButtonEl = document.getElementById("wallet-account-menu-button");
    const walletAccountMenuLabelEl = document.getElementById("wallet-account-menu-label");
    const chainSelectorButtonEl = document.getElementById("chain-selector-button");
    const chainSelectorLogoEl = document.getElementById("chain-selector-logo");
    const chainSelectorCodeEl = document.getElementById("chain-selector-code");
    const chainSelectorMenuEl = document.getElementById("chain-selector-menu");
    const marketTopListEl = document.getElementById("market-top-list");
    const marketBoardStatusEl = document.getElementById("market-board-status");
    const tradeDashboardStatusEl = document.getElementById("trade-dashboard-status");
    const tradeDashboardContentEl = document.getElementById("trade-dashboard-content");
    const walletVaultCardEl = document.getElementById("wallet-vault-card");
    const walletLockErrorEl = document.getElementById("wallet-lock-error");
    const walletVaultCopyEl = document.getElementById("wallet-vault-copy");
    const walletVaultStateEl = document.getElementById("wallet-vault-state");
    const walletAddressEl = document.getElementById("wallet-address");
    const walletKeyringsEl = document.getElementById("wallet-keyrings");
    const walletAccessButtonEl = document.getElementById("wallet-access-button");
    const walletLockButtonEl = document.getElementById("wallet-lock-button");
    const walletChangePasswordButtonEl = document.getElementById("wallet-change-password-button");
    const walletResetButtonEl = document.getElementById("wallet-reset-button");
    const walletExportKeyButtonEl = document.getElementById("wallet-export-key-button");
    const walletExportKeyMenuEl = document.getElementById("wallet-export-key-menu");
    const walletSessionTimeoutMs = 30 * 60 * 1000;
    const walletKDFIterations = 600000;
    let bip39EnglishWords = null;
    const supportedChains = [
      { id: "ethereum", label: "Ethereum", short: "ETH", logo: "Ξ", logoClass: "chain-ethereum", networkId: "ethereum" },
      { id: "base", label: "Base", short: "BASE", logo: "B", logoClass: "chain-base", networkId: "base" },
      { id: "arbitrum", label: "Arbitrum", short: "ARB", logo: "A", logoClass: "chain-arbitrum", networkId: "" },
      { id: "optimism", label: "Optimism", short: "OP", logo: "OP", logoClass: "chain-optimism", networkId: "" },
      { id: "bnb", label: "BNB Chain", short: "BNB", logo: "◎", logoClass: "chain-bnb", networkId: "" },
      { id: "polygon", label: "Polygon", short: "POL", logo: "P", logoClass: "chain-polygon", networkId: "" },
      { id: "solana", label: "Solana", short: "SOL", logo: "S", logoClass: "chain-solana", networkId: "solana" },
      { id: "bitcoin", label: "Bitcoin", short: "BTC", logo: "₿", logoClass: "chain-bitcoin", networkId: "bitcoin" },
      { id: "avalanche", label: "Avalanche", short: "AVAX", logo: "A", logoClass: "chain-avalanche", networkId: "" },
      { id: "sui", label: "Sui", short: "SUI", logo: "SUI", logoClass: "chain-sui", networkId: "sui" }
    ];
    const assetActionTokenCatalog = {
      ethereum: [["ETH", "Ether"], ["USDT", "Tether USD"], ["USDC", "USD Coin"], ["WBTC", "Wrapped BTC"], ["DAI", "Dai"], ["LINK", "Chainlink"]],
      base: [["ETH", "Ether"], ["USDC", "USD Coin"], ["WETH", "Wrapped Ether"], ["cbBTC", "Coinbase Wrapped BTC"], ["DAI", "Dai"], ["USDT", "Tether USD"]],
      arbitrum: [["ETH", "Ether"], ["USDT", "Tether USD"], ["USDC", "USD Coin"], ["WBTC", "Wrapped BTC"], ["DAI", "Dai"], ["LINK", "Chainlink"]],
      optimism: [["ETH", "Ether"], ["USDT", "Tether USD"], ["USDC", "USD Coin"], ["WBTC", "Wrapped BTC"], ["DAI", "Dai"], ["OP", "Optimism"]],
      bnb: [["BNB", "BNB"], ["USDT", "Tether USD"], ["USDC", "USD Coin"], ["BTCB", "BTCB"], ["DAI", "Dai"], ["WBNB", "Wrapped BNB"]],
      polygon: [["POL", "Polygon"], ["USDT", "Tether USD"], ["USDC", "USD Coin"], ["WBTC", "Wrapped BTC"], ["DAI", "Dai"], ["WETH", "Wrapped Ether"]],
      avalanche: [["AVAX", "Avalanche"], ["USDT", "Tether USD"], ["USDC", "USD Coin"], ["WBTC.e", "Wrapped BTC"], ["DAI.e", "Dai"], ["WETH.e", "Wrapped Ether"]],
      solana: [["SOL", "Solana"], ["USDC", "USD Coin"], ["USDT", "Tether USD"], ["wSOL", "Wrapped SOL"]],
      bitcoin: [["BTC", "Bitcoin"]],
      sui: [["SUI", "Sui"]]
    };

    const pageMeta = {
      home: {
        title: "home",
        subtitle: "移动端 H5 总入口与通用问答",
        status: "AI copilot online",
        placeholder: "在 home 里直接提问，例如：这个应用能做什么",
        context: "首页总入口、导航、通用问答、功能说明",
        welcome: "你好，这里是 home 页面。你可以直接在首页发起通用问答，我会帮你介绍功能入口、页面分工和整体能力。"
      },
      market: {
        title: "market",
        subtitle: "市场行情与热点问答",
        status: "Charts and signals",
        placeholder: "问点市场相关的，例如：帮我展示 BTC 最近 7 天的 K 线图",
        context: "市场、价格、热点代币、行情分析",
        welcome: "你好，这里是 market 页面。你可以像和 ChatGPT 对话一样，连续问我行情、价格、市场摘要、热点代币，或者直接让我展示 K 线图。"
      },
      trade: {
        title: "trade",
        subtitle: "交易路线与执行问答",
        status: "Route planning ready",
        placeholder: "问点交易相关的，例如：ETH 换 USDC 怎么走",
        context: "交易、swap、quote、route、bridge、执行风险",
        welcome: "你好，这里是 trade 页面。你可以连续问我换币、交易步骤、路由、滑点和跨链相关问题。"
      },
      pay: {
        title: "pay",
        subtitle: "支付与转账场景问答",
        status: "Fast transfer flow",
        placeholder: "问点支付相关的，例如：链上转账前要检查什么",
        context: "支付、收款、转账、确认信息、支付流程",
        welcome: "你好，这里是 pay 页面。你可以连续问我支付、转账、收款和链上确认流程问题。"
      },
      asset: {
        title: "asset",
        subtitle: "资产与持仓统一查看",
        status: "Portfolio overview",
        placeholder: "问点资产相关的，例如：资产页要展示哪些指标",
        context: "资产、余额、持仓、收益、钱包概览",
        welcome: "你好，这里是 asset 页面。你可以连续问我资产页设计、持仓展示、收益指标和钱包概览问题。"
      }
    };

    const walletBiometricController = window.AgentWalletBiometricController.createWalletBiometricController({
      state,
      biometrics: window.AgentWalletBiometrics,
      isWalletUnlocked,
      unlockWithCredential: unlockWallet
    });

    const walletSecretExportController = window.AgentWalletSecretExportController.createWalletSecretExportController({
      state,
      openUnlock: openWalletUnlock,
      closeUnlock: closeWalletUnlock,
      base64ToBytes,
      deriveVaultKey,
      deriveSeed: deriveBIP39Seed,
      normalizeKeyring: normalizeBIP44Keyring,
      getSelectedChain,
      renderMnemonic: renderWalletRecoveryPhrase,
      copyText: copyTextToClipboard,
      async derivePrivateKey(seed, chainId, accountIndex) {
        if (!window.WalletEVMSigner || typeof window.WalletEVMSigner.derivePrivateKey !== "function") {
          throw new Error("当前页面缺少本地私钥派生能力。");
        }
        return window.WalletEVMSigner.derivePrivateKey(seed, chainId, accountIndex);
      },
      deriveAddress(seed, chainId, accountIndex) {
        if (!window.WalletAddressDerivation || typeof window.WalletAddressDerivation.deriveAddress !== "function") {
          throw new Error("当前页面缺少本地私钥派生能力。");
        }
        return window.WalletAddressDerivation.deriveAddress(seed, chainId, accountIndex);
      }
    });

    const walletAccountController = window.AgentWalletAccountController.createWalletAccountController({
      state,
      isWalletUnlocked,
      maskAddress,
      deriveAddress(seed, chainId, accountIndex) {
        if (!window.WalletAddressDerivation || typeof window.WalletAddressDerivation.deriveAddress !== "function") {
          throw new Error("地址派生能力不可用。");
        }
        return window.WalletAddressDerivation.deriveAddress(seed, chainId, accountIndex);
      },
      randomBytes,
      bytesToBase64,
      saveVault(vault) {
        return WalletVaultStore.save(vault);
      },
      onLocked() {
        state.wallet.pendingAction = "accounts";
        openWalletUnlock("unlock");
        walletLockErrorEl.textContent = "请先解锁钱包，再查看和切换账户。";
      },
      onChanged() {
        renderWalletHeader();
        void assetBalanceController.refresh();
      }
    });

    const assetBalanceController = window.AgentWalletAssetBalanceController.createAssetBalanceController({
      state,
      supportedChainIds: ["ethereum", "base", "arbitrum", "optimism", "bsc", "polygon", "solana", "avalanche"],
      getSelectedChain,
      isWalletUnlocked,
      deriveAddress: deriveAddressForChain,
      request: tracedFetch
    });

    const assetTransferController = window.AgentWalletAssetTransferController.createAssetTransferController({
      state,
      tokenCatalog: assetActionTokenCatalog,
      supportedChainIds: ["ethereum", "base", "arbitrum", "optimism", "bnb", "polygon", "avalanche"],
      getCurrentBalanceItems: assetBalanceController.currentItems,
      getSelectedChain,
      isWalletUnlocked,
      ensureSelectedAddress: ensureSelectedChainAddress,
      getSelectedAddress: getSelectedChainAddress,
      getSelectedAccount: walletAccountController.getSelected,
      getSelectedAccountIndex: walletAccountController.getSelectedIndex,
      request: tracedFetch,
      createTraceId: createBrowserTraceId,
      copyText: copyTextToClipboard,
      signerAddress(seed, accountIndex) {
        if (!window.WalletEVMSigner || typeof window.WalletEVMSigner.getAddress !== "function") {
          throw new Error("本地交易签名模块不可用。");
        }
        return window.WalletEVMSigner.getAddress(seed, accountIndex);
      },
      signTransaction(seed, accountIndex, transaction) {
        if (!window.WalletEVMSigner || typeof window.WalletEVMSigner.signTransaction !== "function") {
          throw new Error("本地交易签名模块不可用。");
        }
        return window.WalletEVMSigner.signTransaction(seed, accountIndex, transaction);
      },
      onLocked(mode, hasVault) {
        if (hasVault) {
          state.wallet.pendingAction = "asset-action:" + mode;
          openWalletUnlock("unlock");
          walletLockErrorEl.textContent = "请先解锁钱包，再使用 " + (mode === "send" ? "Send" : "Receive") + "。";
        } else {
          openWalletUnlock("create");
          walletLockErrorEl.textContent = "请先创建钱包，再使用资产操作。";
        }
      },
      onBeforeOpen: walletAccountController.close,
      onBroadcast(plan, hash, recipient) {
        const account = walletAccountController.getSelected();
        walletHistoryController.recordBroadcast({
          hash,
          chainId: state.selectedChainId,
          accountIndex: account.index,
          accountName: account.name,
          from: getSelectedChainAddress(),
          to: recipient,
          tokenSymbol: plan.token_symbol,
          amount: plan.amount,
          fee: plan.estimated_fee,
          feeSymbol: plan.native_symbol
        });
      },
      onComplete() {
        switchTab("asset");
        void assetBalanceController.refresh();
      }
    });

    const walletHistoryController = window.AgentWalletHistoryController.createWalletHistoryController({
      state,
      chains: supportedChains,
      supportedChainIds: ["ethereum", "base", "arbitrum", "optimism", "bnb", "polygon", "avalanche"],
      maskAddress,
      isWalletUnlocked,
      getSelectedAddress: getSelectedChainAddress,
      ensureAddress: assetBalanceController.ensureAddress,
      normalizeChainId: assetBalanceController.normalizeChainId,
      request: tracedFetch,
      createTraceId: createBrowserTraceId,
      onLocked(hasVault) {
        if (hasVault) {
          state.wallet.pendingAction = "wallet-history";
          openWalletUnlock("unlock");
          walletLockErrorEl.textContent = "请先解锁钱包，再同步当前账户的链上交易记录。";
        } else {
          openWalletUnlock("create");
          walletLockErrorEl.textContent = "请先创建钱包，再查看交易历史。";
        }
      },
      onBeforeOpen() {
        assetTransferController.close();
        walletAccountController.close();
      }
    });

    const walletSessionController = window.AgentWalletSessionController.createWalletSessionController({
      state,
      timeoutMs: walletSessionTimeoutMs,
      beforeLock() {
        walletAccountController.close();
        assetTransferController.close();
        if (state.wallet.mode === "export-display") closeWalletUnlock();
      },
      afterLock() {
        renderWalletVault();
        if (state.activeTab === "asset") syncPageStatus();
      }
    });

    const walletVaultController = window.AgentWalletVaultController.createWalletVaultController({
      state,
      vaultStore: WalletVaultStore,
      unlockVault: window.AgentWalletCore.unlockVault,
      createVault: createWalletVault,
      deriveSeed: deriveBIP39Seed,
      normalizeKeyring: normalizeBIP44Keyring,
      createDefaultKeyring: createDefaultBIP44Keyring,
      generateMnemonic: generateBIP39Mnemonic,
      parseMnemonicWords,
      isValidPassword: isValidWalletPassword,
      isUnlocked: isWalletUnlocked,
      startSession: startWalletSession,
      refreshBiometrics: refreshBiometricStatus,
      renderBiometrics: renderBiometricControls,
      resetBiometricChoice: walletBiometricController.resetEnableChoice,
      shouldEnableBiometrics: walletBiometricController.shouldEnableAfterUnlock,
      biometricStatus: () => state.wallet.biometric,
      enableBiometrics: window.AgentWalletBiometrics.enable,
      disableBiometrics: window.AgentWalletBiometrics.disable,
      resetSecretExport: walletSecretExportController.reset,
      exportMnemonic: walletSecretExportController.exportMnemonic,
      exportPrivateKey: walletSecretExportController.exportPrivateKey,
      copyText: copyTextToClipboard,
      renderVault: renderWalletVault,
      async onPendingAction(pendingAction) {
        if (pendingAction === "reset") {
          state.wallet.pendingAction = "";
          await confirmWalletReset();
        } else if (pendingAction === "accounts") {
          state.wallet.pendingAction = "";
          openWalletAccountDialog();
        } else if (pendingAction === "wallet-history") {
          state.wallet.pendingAction = "";
          walletHistoryController.open();
        } else if (pendingAction.startsWith("asset-action:")) {
          state.wallet.pendingAction = "";
          assetTransferController.open(pendingAction.slice("asset-action:".length));
        }
      }
    });

    state.walletHistory.items = walletHistoryController.load();
    ensureDefaultMessages();
    renderChainSelector();
    renderWalletHeader();
    assetBalanceController.render();
    walletBiometricController.bind();
    walletSecretExportController.bind();
    walletAccountController.bind();
    walletSessionController.bind();
    walletVaultController.bind();
    assetTransferController.bind();
    walletHistoryController.bind();
    bindEvents();
    switchTab("home");
    void loadAIAvailability();
    queueMicrotask(() => void initializeWallet());

    async function loadAIAvailability() {
      let message = "AI 聊天未启用。请在服务端配置 service.aiEnabled=true 并重启服务。";
      try {
        const response = await tracedFetch("/v1/config", { cache: "no-store" });
        if (!response.ok) {
          throw new Error("无法读取服务配置");
        }
        const config = await response.json();
        aiChatEnabled = config.ai_enabled === true;
      } catch (_) {
        aiChatEnabled = false;
        message = "无法确认 AI 聊天状态，请稍后刷新页面。";
      }
      aiAvailabilityEl.hidden = aiChatEnabled;
      aiAvailabilityEl.textContent = message;
      document.querySelectorAll(".home-ai-chat").forEach((element) => {
        element.hidden = !aiChatEnabled;
      });
      switchTab(state.activeTab);
    }

    function ensureDefaultMessages() {
      chatTabs.forEach((tab) => {
        if (!Array.isArray(state.messagesByTab[tab]) || state.messagesByTab[tab].length === 0) {
          state.messagesByTab[tab] = [{
            role: "assistant",
            content: pageMeta[tab].welcome,
            toolExecutions: []
          }];
        }
      });
      persistMessagesByTab();
    }

    function getSelectedChain() {
      return supportedChains.find((chain) => chain.id === state.selectedChainId) || supportedChains[0];
    }

    function renderChainSelector() {
      const chain = getSelectedChain();
      chainSelectorLogoEl.textContent = chain.logo;
      chainSelectorLogoEl.className = "wallet-chain-logo " + chain.logoClass;
      chainSelectorCodeEl.textContent = chain.short;
      Array.from(chainSelectorMenuEl.querySelectorAll("[data-chain-id]")).forEach((option) => {
        option.classList.toggle("active", option.dataset.chainId === chain.id);
      });
    }

    function renderWalletHeader() {
      const address = getSelectedChainAddress();
      if (!isWalletUnlocked()) {
        walletAccountMenuButtonEl.hidden = !state.wallet.vault;
        walletAccountMenuLabelEl.textContent = "账户 ▾";
        walletHeaderAddressEl.textContent = "点击 asset 创建或解锁钱包";
        walletHeaderCopyEl.disabled = true;
        walletHeaderCopyEl.textContent = "copy";
        return;
      }
      const selectedAccount = getSelectedWalletAccount();
      walletAccountMenuButtonEl.hidden = false;
      walletAccountMenuLabelEl.textContent = selectedAccount.name + " ▾";
      if (address) {
        walletHeaderAddressEl.textContent = maskAddress(address);
        walletHeaderCopyEl.disabled = false;
        walletHeaderCopyEl.textContent = "copy";
        return;
      }
      walletHeaderAddressEl.textContent = "地址派生中...";
      walletHeaderCopyEl.disabled = true;
      void ensureSelectedChainAddress();
    }

    function getSelectedChainAddress() {
      if (!state.wallet.addresses || typeof state.wallet.addresses !== "object") {
        return "";
      }
      const address = state.wallet.addresses[state.selectedChainId];
      return typeof address === "string" ? address : "";
    }

    function maskAddress(address) {
      return window.AgentWalletUI.maskAddress(address);
    }

    async function ensureSelectedChainAddress() {
      if (!isWalletUnlocked()) {
        return "";
      }
      const cachedAddress = getSelectedChainAddress();
      if (cachedAddress) {
        return cachedAddress;
      }
      try {
        const address = deriveAddressForChain(state.selectedChainId);
        state.wallet.addresses[state.selectedChainId] = address;
        if (state.selectedChainId) {
          walletHeaderAddressEl.textContent = maskAddress(address);
        }
        walletHeaderCopyEl.disabled = false;
        return address;
      } catch (_) {
        walletHeaderAddressEl.textContent = "地址派生失败";
        walletHeaderCopyEl.disabled = true;
        return "";
      }
    }

    async function copySelectedChainAddress() {
      let address = getSelectedChainAddress();
      if (!address) {
        walletHeaderCopyEl.textContent = "...";
        address = await ensureSelectedChainAddress();
      }
      if (!address) {
        walletHeaderCopyEl.textContent = "失败";
        window.setTimeout(() => {
          walletHeaderCopyEl.textContent = "copy";
        }, 1600);
        return;
      }
      try {
        await copyTextToClipboard(address);
        walletHeaderCopyEl.textContent = "已复制";
        window.setTimeout(() => {
          walletHeaderCopyEl.textContent = "copy";
        }, 1600);
      } catch (_) {
        walletHeaderCopyEl.textContent = "失败";
        window.setTimeout(() => {
          walletHeaderCopyEl.textContent = "copy";
        }, 1600);
      }
    }

    function deriveAddressForChain(chainId) {
      if (
        !window.WalletAddressDerivation ||
        typeof window.WalletAddressDerivation.deriveAddress !== "function" ||
        !(state.wallet.seed instanceof Uint8Array)
      ) {
        throw new Error("地址派生能力不可用。");
      }
      return window.WalletAddressDerivation.deriveAddress(state.wallet.seed, chainId, getSelectedWalletAccountIndex());
    }

    function syncPageStatus() {
      const meta = pageMeta[state.activeTab];
      if (!pageStatusEl || !meta) {
        return;
      }
      pageStatusEl.textContent = state.activeTab === "asset" && !isWalletUnlocked()
        ? "Wallet locked"
        : state.activeTab === "home" && !aiChatEnabled
          ? "AI chat unavailable"
          : (meta.status || "");
    }

    function bindEvents() {
      tabs.forEach((tabButton) => {
        tabButton.addEventListener("click", () => switchTab(tabButton.dataset.tab));
      });

      viewsEl.addEventListener("click", (event) => {
        const entryButton = event.target.closest("[data-open-tab]");
        if (entryButton) {
          switchTab(entryButton.dataset.openTab);
          return;
        }

        const marketTokenButton = event.target.closest("[data-market-asset-id]");
        if (marketTokenButton) {
          const assetId = marketTokenButton.dataset.marketAssetId || "";
          const tokenName = marketTokenButton.dataset.marketName || "";
          if (assetId) {
            openTradeDashboard(assetId, tokenName);
          }
          return;
        }

        const promptButton = event.target.closest("[data-prompt]");
        if (promptButton && aiChatEnabled && chatTabs.includes(state.activeTab)) {
          inputEl.value = promptButton.dataset.prompt || "";
          autoResize();
          inputEl.focus();
        }
      });

      formEl.addEventListener("submit", async (event) => {
        event.preventDefault();
        if (!aiChatEnabled || !chatTabs.includes(state.activeTab)) {
          return;
        }

        const text = inputEl.value.trim();
        if (!text || state.busyTab) {
          return;
        }

        inputEl.value = "";
        autoResize();
        await sendMessage(state.activeTab, text);
      });

      inputEl.addEventListener("input", autoResize);
      inputEl.addEventListener("keydown", (event) => {
        if (event.key === "Enter" && !event.shiftKey) {
          event.preventDefault();
          formEl.requestSubmit();
        }
      });

      walletLockButtonEl.addEventListener("click", () => lockWallet());
      walletResetButtonEl.addEventListener("click", () => {
        void requestWalletReset();
      });
      walletExportKeyButtonEl.addEventListener("click", (event) => {
        event.stopPropagation();
        const expanded = walletExportKeyButtonEl.getAttribute("aria-expanded") === "true";
        walletExportKeyButtonEl.setAttribute("aria-expanded", String(!expanded));
        walletExportKeyMenuEl.hidden = expanded;
        chainSelectorMenuEl.hidden = true;
        chainSelectorButtonEl.setAttribute("aria-expanded", "false");
      });
      walletExportKeyMenuEl.addEventListener("click", (event) => {
        const option = event.target.closest("[data-wallet-key-export]");
        if (!option) {
          return;
        }
        walletExportKeyMenuEl.hidden = true;
        walletExportKeyButtonEl.setAttribute("aria-expanded", "false");
        if (option.dataset.walletKeyExport === "private-key") {
          void requestWalletPrivateKeyExport();
        } else {
          void requestWalletMnemonicExport();
        }
      });
      viewsEl.addEventListener("click", (event) => {
        const actionButton = event.target.closest("[data-asset-action]");
        if (actionButton && actionButton.dataset.assetAction === "history") {
          walletHistoryController.open();
        }
      });
      chainSelectorButtonEl.addEventListener("click", (event) => {
        event.stopPropagation();
        const expanded = chainSelectorButtonEl.getAttribute("aria-expanded") === "true";
        chainSelectorButtonEl.setAttribute("aria-expanded", String(!expanded));
        chainSelectorMenuEl.hidden = expanded;
        walletExportKeyMenuEl.hidden = true;
        walletExportKeyButtonEl.setAttribute("aria-expanded", "false");
      });
      chainSelectorMenuEl.addEventListener("click", (event) => {
        const option = event.target.closest("[data-chain-id]");
        if (!option) {
          return;
        }
        state.selectedChainId = option.dataset.chainId || "ethereum";
        assetTransferController.close();
        chainSelectorMenuEl.hidden = true;
        chainSelectorButtonEl.setAttribute("aria-expanded", "false");
        renderChainSelector();
        renderWalletHeader();
        if (state.activeTab === "asset") {
          assetBalanceController.selectChain();
        }
      });
      walletHeaderCopyEl.addEventListener("click", () => {
        void copySelectedChainAddress();
      });
      walletAccountMenuButtonEl.addEventListener("click", () => {
        openWalletAccountDialog();
      });
      document.addEventListener("click", (event) => {
        if (!chainSelectorMenuEl.contains(event.target) && !chainSelectorButtonEl.contains(event.target)) {
          chainSelectorMenuEl.hidden = true;
          chainSelectorButtonEl.setAttribute("aria-expanded", "false");
        }
        if (!walletExportKeyMenuEl.contains(event.target) && !walletExportKeyButtonEl.contains(event.target)) {
          walletExportKeyMenuEl.hidden = true;
          walletExportKeyButtonEl.setAttribute("aria-expanded", "false");
        }
      });
    }

    function switchTab(tab) {
      state.activeTab = tab;
      if (tab !== "asset") {
        closeWalletAccountDialog();
        assetTransferController.close();
        walletHistoryController.close();
      }
      if (appShellEl) {
        appShellEl.dataset.activeTab = tab;
      }

      views.forEach((view) => {
        view.classList.toggle("active", view.dataset.view === tab);
      });
      tabs.forEach((button) => {
        button.classList.toggle("active", button.dataset.tab === tab);
      });

      const meta = pageMeta[tab];
      pageTitleEl.textContent = meta.title;
      pageSubtitleEl.textContent = meta.subtitle;
      const isChatTab = chatTabs.includes(tab);
      const showComposer = tab === "home" && aiChatEnabled;
      composerWrapEl.classList.toggle("hidden", !showComposer);
      inputEl.disabled = !showComposer;
      inputEl.placeholder = meta.placeholder || "";
      sendButton.disabled = !showComposer || Boolean(state.busyTab);
      syncPageStatus();

      if (tab === "market") {
        renderMarketTopList();
        void ensureMarketTopList();
      }
      if (tab === "trade") {
        renderTradeDashboard();
        syncTradeKlinePolling();
      } else {
        stopTradeKlinePolling();
      }

      if (isChatTab && aiChatEnabled) {
        renderThread(tab);
        autoResize();
        requestAnimationFrame(() => {
          scrollConversationToBottom(tab);
          inputEl.focus();
        });
      }

      if (tab === "asset") {
        void prepareWalletAccess();
        void assetBalanceController.refresh();
      }
    }

    async function initializeWallet() {
      await walletVaultController.initialize();
    }

    async function prepareWalletAccess() {
      await walletVaultController.prepareAccess();
    }

    function openWalletUnlock(mode) {
      walletVaultController.open(mode);
    }

    function closeWalletUnlock(clearPendingAction = true) {
      walletVaultController.close(clearPendingAction);
    }

    async function handleWalletSubmit() {
      await walletVaultController.submit();
    }

    async function unlockWallet(password) {
      await walletVaultController.unlock(password);
    }

    async function changeWalletPassword(currentPassword, newPassword) {
      await walletVaultController.changePassword(currentPassword, newPassword);
    }

    async function requestWalletMnemonicExport() {
      walletSecretExportController.requestMnemonic();
    }

    async function requestWalletPrivateKeyExport() {
      walletSecretExportController.requestPrivateKey();
    }

    async function exportWalletMnemonic(password) {
      await walletSecretExportController.exportMnemonic(password);
    }

    async function exportWalletPrivateKey(password) {
      await walletSecretExportController.exportPrivateKey(password);
    }

    function isValidWalletPassword(password) {
      return window.AgentWalletCore.isValidPassword(password);
    }

    function isWalletUnlocked() {
      return walletSessionController.isUnlocked();
    }

    function startWalletSession() {
      walletSessionController.start();
    }

    function refreshWalletSession() {
      walletSessionController.refresh();
    }

    function lockWallet() {
      walletSessionController.lock();
    }

    function renderWalletVault(errorMessage) {
      if (!walletVaultCopyEl) {
        return;
      }

      if (errorMessage) {
        walletVaultCardEl.hidden = false;
        walletVaultCopyEl.textContent = errorMessage;
        walletVaultStateEl.textContent = "UNAVAILABLE";
        walletVaultStateEl.classList.add("locked");
        walletAddressEl.textContent = "请使用受支持的现代浏览器并启用 IndexedDB。";
        walletAccessButtonEl.disabled = true;
        walletLockButtonEl.hidden = true;
        walletBiometricController.render();
        walletChangePasswordButtonEl.hidden = true;
        walletResetButtonEl.hidden = true;
        walletExportKeyButtonEl.hidden = true;
        walletExportKeyMenuEl.hidden = true;
        walletExportKeyButtonEl.setAttribute("aria-expanded", "false");
        renderWalletHeader();
        assetBalanceController.render();
        syncPageStatus();
        return;
      }

      walletAccessButtonEl.disabled = false;
      if (isWalletUnlocked()) {
        walletVaultCardEl.hidden = true;
        walletVaultCopyEl.textContent = "钱包已解锁。助记词仅存在于当前页面内存，会在 30 分钟无操作、关闭页面或手动锁定时清除。";
        walletVaultStateEl.textContent = "UNLOCKED";
        walletVaultStateEl.classList.remove("locked");
        walletAddressEl.textContent = "已解锁 · BIP39 助记词已恢复为 seed，BIP44 keyring 将按不同链的标准路径进行派生。";
        renderBIP44Keyrings(state.wallet.keyring);
        walletAccessButtonEl.textContent = "已解锁";
        walletAccessButtonEl.disabled = true;
        walletLockButtonEl.hidden = false;
        renderBiometricControls();
        walletChangePasswordButtonEl.hidden = false;
        walletResetButtonEl.hidden = false;
        walletExportKeyButtonEl.hidden = false;
        renderWalletHeader();
        syncPageStatus();
        if (state.activeTab === "asset") {
          void assetBalanceController.refresh();
        } else {
          assetBalanceController.render();
        }
        return;
      }

      const hasVault = Boolean(state.wallet.vault);
      walletVaultCardEl.hidden = false;
      walletKeyringsEl.hidden = true;
      walletKeyringsEl.replaceChildren();
      walletVaultCopyEl.textContent = hasVault
        ? "检测到本设备的加密钱包保险库。输入密码即可在浏览器本地解锁。"
        : "本设备尚未创建钱包。创建会生成 BIP39 12 词恢复短语，并配置 BIP44 多链 keyring。";
      walletVaultStateEl.textContent = hasVault ? "LOCKED" : "NEW";
      walletVaultStateEl.classList.toggle("locked", hasVault);
      walletAddressEl.textContent = hasVault
        ? "已锁定：加密恢复短语保存在浏览器 IndexedDB，服务器无法读取。"
        : "创建钱包后，保险库只保存经过 AES-256-GCM 加密的恢复短语。";
      walletAccessButtonEl.textContent = hasVault ? "解锁钱包" : "创建钱包";
      walletLockButtonEl.hidden = true;
      walletBiometricController.render();
      walletChangePasswordButtonEl.hidden = !hasVault;
      walletResetButtonEl.hidden = !hasVault;
      walletExportKeyButtonEl.hidden = !hasVault;
      walletExportKeyMenuEl.hidden = true;
      walletExportKeyButtonEl.setAttribute("aria-expanded", "false");
      renderWalletHeader();
      assetBalanceController.render();
      syncPageStatus();
    }

    async function refreshBiometricStatus() {
      await walletBiometricController.refresh();
    }

    function renderBiometricControls() {
      walletBiometricController.render();
    }
    function renderWalletRecoveryPhrase(mnemonic) {
      walletVaultController.renderRecoveryPhrase(mnemonic);
    }

    function renderWalletImportInputs(words) {
      walletVaultController.renderImportInputs(words);
    }

    async function startGeneratedMnemonicFlow() {
      await walletVaultController.startGeneratedMnemonic();
    }

    function startImportMnemonicFlow() {
      walletVaultController.startImportMnemonic();
    }

    async function copyWalletRecoveryPhrase() {
      await walletVaultController.copyRecoveryPhrase();
    }

    async function copyWalletPrivateKey() {
      await walletSecretExportController.copyPrivateKey();
    }

    async function hydrateImportedMnemonicInputs() {
      await walletVaultController.hydrateImportedMnemonic();
    }

    async function requestWalletReset() {
      if (!state.wallet.vault && !isWalletUnlocked()) {
        return;
      }
      if (!isWalletUnlocked()) {
        state.wallet.pendingAction = "reset";
        openWalletUnlock("unlock");
        walletLockErrorEl.textContent = "请先解锁钱包，再执行重置。";
        return;
      }
      await confirmWalletReset();
    }

    async function confirmWalletReset() {
      if (!window.confirm("您正在重置钱包，确认继续吗？")) {
        return;
      }
      try {
        await WalletVaultStore.remove();
        await window.AgentWalletBiometrics.disable().catch(() => undefined);
        await refreshBiometricStatus();
        closeWalletUnlock();
        lockWallet();
        state.wallet.vault = null;
        renderWalletVault();
        openWalletUnlock("create");
      } catch (error) {
        renderWalletVault(error instanceof Error ? error.message : "重置钱包失败。");
      }
    }

    async function finalizeWalletCreation(mnemonic) {
      await walletVaultController.finalizeCreation(mnemonic);
    }

    async function readImportedMnemonic() {
      return walletVaultController.readImportedMnemonic();
    }

    function createBIP44Networks(accountIndex) {
      return window.AgentWalletAccounts.createNetworks(accountIndex);
    }

    function createDefaultBIP44Keyring() {
      return walletAccountController.createDefault();
    }

    function normalizeBIP44Keyring(keyring) {
      return walletAccountController.normalize(keyring);
    }

    function getSelectedWalletAccountIndex() {
      return walletAccountController.getSelectedIndex();
    }

    function getSelectedWalletAccount() {
      return walletAccountController.getSelected();
    }

    function openWalletAccountDialog() {
      walletAccountController.open();
    }

    function closeWalletAccountDialog() {
      walletAccountController.close();
    }

    function renderBIP44Keyrings(keyring) {
      walletAccountController.renderKeyrings(keyring);
    }
    async function createWalletVault(password, mnemonic, keyring) {
      const { vault, masterKey } = await window.AgentWalletCore.createVault(
        password,
        mnemonic,
        normalizeBIP44Keyring(keyring),
        walletKDFIterations
      );
      await WalletVaultStore.save(vault);
      return { vault, masterKey };
    }

    async function deriveBIP39Seed(mnemonic, passphrase) {
      return window.AgentWalletCore.deriveBIP39Seed(mnemonic, passphrase || "");
    }

    async function deriveVaultKey(password, salt, iterations) {
      return window.AgentWalletCore.deriveVaultKey(password, salt, iterations);
    }

    async function generateBIP39Mnemonic() {
      const wordlist = await loadBIP39EnglishWordlist();
      const entropy = randomBytes(16);
      const digest = new Uint8Array(await crypto.subtle.digest("SHA-256", entropy));
      let bits = bytesToBits(entropy) + bytesToBits(digest).slice(0, 4);
      const words = [];
      for (let index = 0; index < 12; index += 1) {
        const wordIndex = parseInt(bits.slice(index * 11, (index + 1) * 11), 2);
        words.push(wordlist[wordIndex]);
      }
      bits = "";
      entropy.fill(0);
      digest.fill(0);
      return words.join(" ");
    }

    async function loadBIP39EnglishWordlist() {
      if (Array.isArray(bip39EnglishWords)) {
        return bip39EnglishWords;
      }
      const response = await tracedFetch("/wallet/bip39-english", { cache: "force-cache" });
      if (!response.ok) {
        throw new Error("无法加载 BIP39 词表。");
      }
      const words = (await response.text()).trim().split("\n").map((word) => word.trim()).filter(Boolean);
      if (words.length !== 2048) {
        throw new Error("BIP39 词表无效。");
      }
      bip39EnglishWords = words;
      return words;
    }

    function randomBytes(length) {
      return window.AgentWalletCore.randomBytes(length);
    }

    function bytesToBits(bytes) {
      return Array.from(bytes).map((byte) => byte.toString(2).padStart(8, "0")).join("");
    }

    function bytesToBase64(bytes) {
      return window.AgentWalletCore.bytesToBase64(bytes);
    }

    async function copyTextToClipboard(text) {
      if (navigator.clipboard && window.isSecureContext) {
        await navigator.clipboard.writeText(text);
        return;
      }

      const helper = document.createElement("textarea");
      helper.value = text;
      helper.setAttribute("readonly", "readonly");
      helper.style.position = "absolute";
      helper.style.left = "-9999px";
      document.body.appendChild(helper);
      helper.select();
      helper.setSelectionRange(0, helper.value.length);
      const copied = document.execCommand("copy");
      helper.remove();
      if (!copied) {
        throw new Error("当前浏览器无法复制恢复短语。");
      }
    }

    async function parseMnemonicWords(text) {
      const words = text
        .trim()
        .replace(/\s+/g, " ")
        .split(" ")
        .map((word) => word.trim().toLowerCase())
        .filter(Boolean);
      if (words.length !== 12) {
        throw new Error("助记词必须正好包含 12 个单词。");
      }
      const wordlist = await loadBIP39EnglishWordlist();
      if (words.some((word) => !wordlist.includes(word))) {
        throw new Error("存在不在 BIP39 词表中的助记词，请检查拼写。");
      }
      return words;
    }

    function base64ToBytes(value) {
      return window.AgentWalletCore.base64ToBytes(value);
    }

    function getThreadEl(tab) {
      return document.getElementById(tab + "-thread");
    }

    function getViewEl(tab) {
      return document.querySelector('.view[data-view="' + tab + '"]');
    }

    function renderThread(tab) {
      const threadEl = getThreadEl(tab);
      if (!threadEl) {
        return;
      }

      threadEl.innerHTML = "";
      const messages = state.messagesByTab[tab] || [];
      messages.forEach((message) => {
        const row = document.createElement("div");
        row.className = "message " + message.role + (message.pending ? " pending" : "");

        const avatar = document.createElement("div");
        avatar.className = "avatar";
        avatar.textContent = message.role === "user" ? "你" : message.role === "system" ? "!" : "AI";

        const bubble = document.createElement("div");
        bubble.className = "bubble";
        renderRichContent(bubble, message.content, { compact: false });

        row.appendChild(avatar);
        row.appendChild(bubble);

        if (Array.isArray(message.toolExecutions) && message.toolExecutions.length > 0) {
          const toolCalls = document.createElement("div");
          toolCalls.className = "tool-calls";
          message.toolExecutions.forEach((toolExecution) => {
            const item = document.createElement("div");
            item.className = "tool-call";
            renderToolExecution(item, toolExecution);
            toolCalls.appendChild(item);
          });
          bubble.appendChild(toolCalls);
        }

        threadEl.appendChild(row);
      });

      persistMessagesByTab();
      requestAnimationFrame(() => scrollConversationToBottom(tab));
    }

    async function ensureMarketTopList() {
      const now = Date.now();
      if (state.marketTop.loading) {
        return;
      }
      if (state.marketTop.items.length > 0 && now - state.marketTop.loadedAt < 60000) {
        return;
      }

      state.marketTop.loading = true;
      state.marketTop.error = "";
      renderMarketTopList();

      try {
        const response = await tracedFetch("/v1/market/top");
        const payload = await response.json();
        if (!response.ok) {
          throw new Error(payload.error || "行情加载失败");
        }

        state.marketTop.items = Array.isArray(payload.items) ? payload.items : [];
        state.marketTop.loadedAt = Date.now();
      } catch (error) {
        state.marketTop.error = error instanceof Error ? error.message : "行情加载失败";
      } finally {
        state.marketTop.loading = false;
        renderMarketTopList();
      }
    }

    function renderMarketTopList() {
      if (!marketTopListEl || !marketBoardStatusEl) {
        return;
      }

      marketTopListEl.innerHTML = "";

      if (state.marketTop.loading) {
        marketBoardStatusEl.textContent = "正在加载市值前十行情...";
        return;
      }

      if (state.marketTop.error) {
        marketBoardStatusEl.textContent = state.marketTop.error;
        return;
      }

      if (!Array.isArray(state.marketTop.items) || state.marketTop.items.length === 0) {
        marketBoardStatusEl.textContent = "暂无行情数据";
        return;
      }

      marketBoardStatusEl.textContent = "按市值排序 · 实时拉取";
      state.marketTop.items.forEach((item, index) => {
        marketTopListEl.appendChild(createMarketTopRow(item, index));
      });
    }

    function createMarketTopRow(item, index) {
      const row = document.createElement("button");
      row.className = "market-top-item";
      row.type = "button";
      row.dataset.marketAssetId = item.id || "";
      row.dataset.marketName = item.name || "";

      const rank = document.createElement("div");
      rank.className = "market-rank";
      rank.textContent = String(item.market_cap_rank || index + 1);
      row.appendChild(rank);

      const name = document.createElement("div");
      name.className = "market-token-name";
      const title = document.createElement("strong");
      title.textContent = item.name || "Unknown";
      const symbol = document.createElement("span");
      symbol.textContent = item.symbol || "";
      name.appendChild(title);
      name.appendChild(symbol);
      row.appendChild(name);

      const price = document.createElement("div");
      price.className = "market-token-price";
      price.textContent = formatUSDPrice(item.current_price);
      row.appendChild(price);

      const change = document.createElement("div");
      const delta = Number(item.price_change_percentage_24h || 0);
      change.className = "market-token-change " + (delta > 0 ? "positive" : delta < 0 ? "negative" : "flat");
      change.textContent = formatPercent(delta);
      row.appendChild(change);

      return row;
    }

    async function openTradeDashboard(assetId, tokenName) {
      state.tradeDashboard.selectedAssetId = assetId;
      state.tradeDashboard.selectedTokenName = tokenName || "";
      switchTab("trade");
      state.tradeDashboard.loading = true;
      state.tradeDashboard.error = "";
      state.tradeDashboard.item = null;
      state.tradeKlines.item = null;
      state.tradeKlines.error = "";
      renderTradeDashboard();

      try {
        const response = await tracedFetch("/v1/trade/token?id=" + encodeURIComponent(assetId));
        const payload = await response.json();
        if (!response.ok) {
          throw new Error(payload.error || "交易数据加载失败");
        }

        state.tradeDashboard.item = payload.item || null;
        if (!state.tradeDashboard.item && tokenName) {
          throw new Error("没有拿到 " + tokenName + " 的交易数据");
        }
      } catch (error) {
        state.tradeDashboard.error = error instanceof Error ? error.message : "交易数据加载失败";
      } finally {
        state.tradeDashboard.loading = false;
        renderTradeDashboard();
        syncTradeKlinePolling();
      }
    }

    function renderTradeDashboard() {
      if (!tradeDashboardStatusEl || !tradeDashboardContentEl) {
        return;
      }

      tradeDashboardContentEl.innerHTML = "";

      if (state.tradeDashboard.loading) {
        tradeDashboardStatusEl.textContent = "正在加载交易面板...";
        tradeDashboardContentEl.className = "trade-dashboard-empty";
        tradeDashboardContentEl.textContent = "正在同步价格、24h 波动、交易量和供应量数据...";
        return;
      }

      if (state.tradeDashboard.error) {
        tradeDashboardStatusEl.textContent = state.tradeDashboard.error;
        tradeDashboardContentEl.className = "trade-dashboard-empty";
        tradeDashboardContentEl.textContent = "请从 market 榜单重新选择一个代币，或稍后再试。";
        return;
      }

      if (!state.tradeDashboard.item) {
        tradeDashboardStatusEl.textContent = "从 market 榜单点击代币后查看";
        tradeDashboardContentEl.className = "trade-dashboard-empty";
        tradeDashboardContentEl.textContent = "选择一个代币后，这里会展示当前价格、24h 涨跌、24h 高低点、成交量、市值和供应量等交易数据。";
        return;
      }

      tradeDashboardStatusEl.textContent = "交易数据已更新 · CoinGecko";
      tradeDashboardContentEl.className = "";
      tradeDashboardContentEl.appendChild(createTradeDashboardCard(state.tradeDashboard.item));
    }

    function syncTradeKlinePolling() {
      if (state.activeTab !== "trade" || !state.tradeDashboard.selectedAssetId) {
        stopTradeKlinePolling();
        return;
      }

      if (!state.tradeKlines.refreshTimer) {
        state.tradeKlines.refreshTimer = window.setInterval(() => {
          void refreshTradeKlines();
        }, 60000);
      }

      void refreshTradeKlines();
    }

    function stopTradeKlinePolling() {
      if (state.tradeKlines.refreshTimer) {
        window.clearInterval(state.tradeKlines.refreshTimer);
        state.tradeKlines.refreshTimer = 0;
      }
    }

    async function refreshTradeKlines() {
      if (!state.tradeDashboard.selectedAssetId || state.tradeKlines.loading) {
        return;
      }

      state.tradeKlines.loading = true;
      state.tradeKlines.error = "";
      renderTradeDashboard();

      try {
        const response = await tracedFetch("/v1/trade/klines?id=" + encodeURIComponent(state.tradeDashboard.selectedAssetId) + "&interval=15m&limit=32");
        const payload = await response.json();
        if (!response.ok) {
          throw new Error(payload.error || "K 线加载失败");
        }

        state.tradeKlines.item = payload.item || null;
        state.tradeKlines.updatedAt = payload.updated_at || "";
      } catch (error) {
        state.tradeKlines.error = error instanceof Error ? error.message : "K 线加载失败";
      } finally {
        state.tradeKlines.loading = false;
        if (state.activeTab === "trade") {
          renderTradeDashboard();
        }
      }
    }

    function createTradeDashboardCard(item) {
      const wrapper = document.createElement("div");
      wrapper.className = "trade-dashboard-card";

      const hero = document.createElement("div");
      hero.className = "trade-hero";

      const title = document.createElement("div");
      title.className = "trade-token-title";
      const tokenName = document.createElement("strong");
      tokenName.textContent = item.name || "Unknown";
      const tokenMeta = document.createElement("span");
      tokenMeta.textContent = (item.symbol || "").toUpperCase() + (item.id ? " · " + item.id : "");
      title.appendChild(tokenName);
      title.appendChild(tokenMeta);
      hero.appendChild(title);

      const price = document.createElement("div");
      price.className = "trade-price";
      const priceValue = document.createElement("div");
      priceValue.className = "trade-price-value";
      priceValue.textContent = formatUSDPrice(item.current_price);
      const priceChange = document.createElement("div");
      const delta = Number(item.price_change_percentage_24h || 0);
      priceChange.className = "trade-price-change " + (delta > 0 ? "positive" : delta < 0 ? "negative" : "flat");
      priceChange.textContent = formatPercent(delta) + " today";
      price.appendChild(priceValue);
      price.appendChild(priceChange);
      hero.appendChild(price);
      wrapper.appendChild(hero);

      const statsGrid = document.createElement("div");
      statsGrid.className = "trade-stats-grid";
      [
        ["24h High", formatUSDPrice(item.high_24h)],
        ["24h Low", formatUSDPrice(item.low_24h)],
        ["24h Volume", formatCompactCurrency(item.total_volume)],
        ["Market Cap", formatCompactCurrency(item.market_cap)],
        ["FDV", formatCompactCurrency(item.fully_diluted_valuation)],
        ["Circulating", formatCompactAmount(item.circulating_supply)],
        ["Total Supply", item.total_supply ? formatCompactAmount(item.total_supply) : "unavailable"],
        ["Rank", item.market_cap_rank ? "#" + item.market_cap_rank : "n/a"]
      ].forEach((entry) => {
        const stat = document.createElement("div");
        stat.className = "trade-stat-card";
        const label = document.createElement("label");
        label.textContent = entry[0];
        const value = document.createElement("strong");
        value.textContent = entry[1];
        stat.appendChild(label);
        stat.appendChild(value);
        statsGrid.appendChild(stat);
      });
      wrapper.appendChild(statsGrid);

      const insights = document.createElement("div");
      insights.className = "trade-insight-row";
      [
        "spot-ready",
        delta >= 0 ? "positive momentum" : "pullback watch",
        "liquidity overview",
        "trade context"
      ].forEach((text) => {
        const pill = document.createElement("span");
        pill.className = "trade-insight-pill";
        pill.textContent = text;
        insights.appendChild(pill);
      });
      wrapper.appendChild(insights);

      wrapper.appendChild(createTradeKlinePanel());

      return wrapper;
    }

    function createTradeKlinePanel() {
      const panel = document.createElement("div");
      panel.className = "trade-kline-panel";

      const head = document.createElement("div");
      head.className = "trade-kline-head";
      const title = document.createElement("strong");
      title.textContent = "实时 K 线";
      head.appendChild(title);
      const meta = document.createElement("div");
      meta.className = "trade-kline-meta";
      meta.textContent = "15m candles · refresh 60s";
      head.appendChild(meta);
      panel.appendChild(head);

      const status = document.createElement("div");
      status.className = "trade-kline-status";
      if (state.tradeKlines.loading && !state.tradeKlines.item) {
        status.textContent = "正在加载 15m K 线...";
      } else if (state.tradeKlines.error) {
        status.textContent = state.tradeKlines.error;
      } else if (state.tradeKlines.updatedAt) {
        status.textContent = "最近更新 " + formatUpdatedTime(state.tradeKlines.updatedAt);
      } else {
        status.textContent = "等待 K 线数据...";
      }
      panel.appendChild(status);

      if (state.tradeKlines.item && Array.isArray(state.tradeKlines.item.candles) && state.tradeKlines.item.candles.length > 0) {
        const chartWrap = document.createElement("div");
        chartWrap.className = "kline-chart-wrap";
        chartWrap.appendChild(createKlineChart(state.tradeKlines.item.candles));
        panel.appendChild(chartWrap);
      }

      return panel;
    }

    function renderRichContent(container, text, options) {
      const chart = parseKlineObservation(text);
      if (!chart) {
        const copy = document.createElement("div");
        copy.className = options && options.compact ? "tool-copy" : "message-copy";
        copy.textContent = text;
        container.appendChild(copy);
        return;
      }

      container.appendChild(createKlineCard(chart, options || {}));
    }

    function renderToolExecution(container, toolExecution) {
      const chart = parseKlineObservation(toolExecution.observation);
      if (!chart) {
        const copy = document.createElement("div");
        copy.className = "tool-copy";
        copy.textContent = toolExecution.name + ": " + toolExecution.observation;
        container.appendChild(copy);
        return;
      }

      container.classList.add("kline-tool-call");
      container.appendChild(createKlineCard(chart, {
        compact: true,
        toolName: toolExecution.name
      }));
    }

    function parseKlineObservation(text) {
      return window.AgentWalletMarket.parseObservation(text);
    }

    function parseKlineCandleLine(line) {
      return window.AgentWalletMarket.parseCandleLine(line);
    }

    function createKlineCard(chart, options) {
      const card = document.createElement("div");
      card.className = "kline-card";

      const header = document.createElement("div");
      header.className = "kline-card-header";

      const titleBlock = document.createElement("div");
      const titleEl = document.createElement("div");
      titleEl.className = "kline-title";
      titleEl.textContent = chart.title;
      titleBlock.appendChild(titleEl);

      const subtitleEl = document.createElement("div");
      subtitleEl.className = "kline-subtitle";
      subtitleEl.textContent = buildKlineSubtitle(chart, options || {});
      titleBlock.appendChild(subtitleEl);
      header.appendChild(titleBlock);

      const priceBlock = document.createElement("div");
      priceBlock.className = "kline-price-block";
      const lastPriceEl = document.createElement("div");
      lastPriceEl.className = "kline-last-price";
      lastPriceEl.textContent = formatChartPrice(chart.candles[chart.candles.length - 1].close);
      priceBlock.appendChild(lastPriceEl);

      const change = calculateKlineChange(chart.candles);
      const changeEl = document.createElement("div");
      changeEl.className = "kline-change " + change.className;
      changeEl.textContent = change.label;
      priceBlock.appendChild(changeEl);
      header.appendChild(priceBlock);
      card.appendChild(header);

      const chipRow = document.createElement("div");
      chipRow.className = "kline-chip-row";
      buildKlineChips(chart, options || {}).forEach((chipText) => {
        const chip = document.createElement("span");
        chip.className = "kline-chip";
        chip.textContent = chipText;
        chipRow.appendChild(chip);
      });
      if (chipRow.childNodes.length > 0) {
        card.appendChild(chipRow);
      }

      const statsRow = document.createElement("div");
      statsRow.className = "kline-stats";
      buildKlineStats(chart.candles).forEach((item) => {
        const stat = document.createElement("span");
        stat.className = "kline-stat";
        const label = document.createElement("strong");
        label.textContent = item.label;
        stat.appendChild(label);
        stat.appendChild(document.createTextNode(item.value));
        statsRow.appendChild(stat);
      });
      card.appendChild(statsRow);

      const chartWrap = document.createElement("div");
      chartWrap.className = "kline-chart-wrap";
      chartWrap.appendChild(createKlineChart(chart.candles));
      card.appendChild(chartWrap);

      if (chart.note) {
        const note = document.createElement("div");
        note.className = "kline-note";
        note.textContent = chart.note;
        card.appendChild(note);
      }

      return card;
    }

    function buildKlineSubtitle(chart, options) {
      const parts = [];
      if (options.toolName) {
        parts.push(options.toolName);
      }
      if (chart.interval) {
        parts.push(chart.interval);
      }
      if (chart.vsCurrency) {
        parts.push(chart.vsCurrency);
      }
      if (chart.candlesReturned) {
        parts.push(chart.candlesReturned + " candles");
      }
      return parts.join(" · ");
    }

    function buildKlineChips(chart, options) {
      const chips = [];
      if (chart.mode) {
        chips.push(chart.mode);
      }
      if (chart.chain) {
        chips.push(chart.chain);
      }
      if (chart.source) {
        chips.push(chart.source);
      }
      if (!options.compact && chart.assetID) {
        chips.push(chart.assetID);
      }
      return chips;
    }

    function buildKlineStats(candles) {
      return window.AgentWalletMarket.stats(candles);
    }

    function calculateKlineChange(candles) {
      return window.AgentWalletMarket.calculateChange(candles);
    }

    let klineChartCounter = 0;

    function createKlineChart(candles) {
      const fragment = document.createDocumentFragment();

      const inner = document.createElement("div");
      inner.className = "kline-chart-inner";

      const svg = createKlineChartSVG(candles);
      inner.appendChild(svg);

      const tooltip = document.createElement("div");
      tooltip.className = "kline-tooltip";
      inner.appendChild(tooltip);

      fragment.appendChild(inner);

      const detail = document.createElement("div");
      detail.className = "kline-detail-panel";
      renderKlineDetail(detail, null, candles);
      fragment.appendChild(detail);

      bindKlineInteractions(svg, tooltip, detail, candles);

      return fragment;
    }

    function bindKlineInteractions(svg, tooltip, detail, candles) {
      const layout = svg.__klineLayout;
      if (!layout) {
        return;
      }

      const findIndex = (event) => {
        const rect = svg.getBoundingClientRect();
        if (!rect.width) {
          return -1;
        }
        const source = event.touches && event.touches[0] ? event.touches[0] : event;
        const localX = ((source.clientX - rect.left) / rect.width) * 360;
        let bestIndex = 0;
        let bestDelta = Infinity;
        layout.xPositions.forEach((x, index) => {
          const delta = Math.abs(x - localX);
          if (delta < bestDelta) {
            bestDelta = delta;
            bestIndex = index;
          }
        });
        return bestIndex;
      };

      const showCrosshair = (index) => {
        if (index < 0 || index >= candles.length) {
          return;
        }
        const candle = candles[index];
        const x = layout.xPositions[index];
        const y = layout.closeYs[index];
        layout.crosshairV.setAttribute("x1", String(x));
        layout.crosshairV.setAttribute("x2", String(x));
        layout.crosshairV.setAttribute("y1", String(layout.padding.top));
        layout.crosshairV.setAttribute("y2", String(layout.padding.top + layout.plotHeight));
        layout.crosshairV.setAttribute("visibility", "visible");
        layout.crosshairDot.setAttribute("cx", String(x));
        layout.crosshairDot.setAttribute("cy", String(y));
        layout.crosshairDot.setAttribute("visibility", "visible");

        tooltip.classList.add("visible");
        tooltip.style.left = ((x / 360) * 100).toFixed(2) + "%";
        tooltip.textContent = formatChartPrice(candle.close);
      };

      const hideCrosshair = () => {
        layout.crosshairV.setAttribute("visibility", "hidden");
        layout.crosshairDot.setAttribute("visibility", "hidden");
        tooltip.classList.remove("visible");
      };

      svg.addEventListener("mousemove", (event) => {
        const index = findIndex(event);
        showCrosshair(index);
      });
      svg.addEventListener("mouseleave", hideCrosshair);
      svg.addEventListener("click", (event) => {
        const index = findIndex(event);
        renderKlineDetail(detail, index, candles);
        showCrosshair(index);
      });
      svg.addEventListener("touchstart", (event) => {
        const index = findIndex(event);
        showCrosshair(index);
        renderKlineDetail(detail, index, candles);
        if (event.cancelable) {
          event.preventDefault();
        }
      }, { passive: false });
      svg.addEventListener("touchmove", (event) => {
        const index = findIndex(event);
        showCrosshair(index);
        if (event.cancelable) {
          event.preventDefault();
        }
      }, { passive: false });
      svg.addEventListener("touchend", hideCrosshair);
    }

    function renderKlineDetail(container, index, candles) {
      container.innerHTML = "";
      if (index === null || index === undefined || index < 0 || index >= candles.length) {
        const empty = document.createElement("div");
        empty.className = "kline-detail-empty";
        empty.textContent = "点击 K 线查看四价与涨跌幅";
        container.appendChild(empty);
        return;
      }

      const candle = candles[index];
      const prev = candles[index - 1];
      const base = prev ? prev.close : candle.open;
      const delta = base === 0 ? 0 : ((candle.close - base) / base) * 100;
      const bodyDelta = candle.open === 0 ? 0 : ((candle.close - candle.open) / candle.open) * 100;
      const sign = delta > 0 ? "+" : "";
      const trend = delta > 0 ? "positive" : delta < 0 ? "negative" : "flat";
      const bodyTrend = bodyDelta > 0 ? "positive" : bodyDelta < 0 ? "negative" : "flat";

      const time = document.createElement("div");
      time.className = "kline-detail-time";
      time.textContent = formatKlineTimeLabel(candle.timestamp);
      container.appendChild(time);

      const entries = [
        { label: "Open", value: formatChartPrice(candle.open), variant: "" },
        { label: "High", value: formatChartPrice(candle.high), variant: "positive" },
        { label: "Low", value: formatChartPrice(candle.low), variant: "negative" },
        { label: "Close", value: formatChartPrice(candle.close), variant: bodyTrend },
        { label: "Change %", value: sign + delta.toFixed(2) + "%", variant: trend }
      ];

      entries.forEach((entry) => {
        const item = document.createElement("div");
        item.className = "kline-detail-item" + (entry.variant ? " " + entry.variant : "");
        const label = document.createElement("label");
        label.textContent = entry.label;
        const value = document.createElement("strong");
        value.textContent = entry.value;
        item.appendChild(label);
        item.appendChild(value);
        container.appendChild(item);
      });
    }

    function createKlineChartSVG(candles) {
      const svgNS = "http://www.w3.org/2000/svg";
      const svg = document.createElementNS(svgNS, "svg");
      svg.setAttribute("class", "kline-chart");
      svg.setAttribute("viewBox", "0 0 360 220");
      svg.setAttribute("preserveAspectRatio", "none");
      svg.setAttribute("role", "img");
      svg.setAttribute("aria-label", "Token candlestick chart");

      const padding = { top: 14, right: 46, bottom: 26, left: 12 };
      const plotWidth = 360 - padding.left - padding.right;
      const plotHeight = 220 - padding.top - padding.bottom;
      const highs = candles.map((candle) => candle.high);
      const lows = candles.map((candle) => candle.low);
      const maxPrice = Math.max.apply(null, highs);
      const minPrice = Math.min.apply(null, lows);
      const priceRange = maxPrice - minPrice || Math.max(maxPrice * 0.01, 1);
      const paddedRange = priceRange * 0.08;
      const domainMin = minPrice - paddedRange;
      const domainMax = maxPrice + paddedRange;
      const range = domainMax - domainMin;

      const xPositions = candles.map((_, index) => (
        padding.left + (candles.length === 1 ? plotWidth / 2 : (plotWidth / (candles.length - 1)) * index)
      ));
      const closeYs = candles.map((candle) => toChartY(candle.close, domainMin, range, padding.top, plotHeight));

      const candleWidth = Math.max(Math.min((plotWidth / Math.max(candles.length, 1)) * 0.6, 14), 4);

      klineChartCounter += 1;
      const gradientId = "kline-area-" + klineChartCounter;

      const defs = document.createElementNS(svgNS, "defs");
      const gradient = document.createElementNS(svgNS, "linearGradient");
      gradient.setAttribute("id", gradientId);
      gradient.setAttribute("x1", "0");
      gradient.setAttribute("y1", "0");
      gradient.setAttribute("x2", "0");
      gradient.setAttribute("y2", "1");
      const stopTop = document.createElementNS(svgNS, "stop");
      stopTop.setAttribute("offset", "0%");
      stopTop.setAttribute("stop-color", "#60a5fa");
      stopTop.setAttribute("stop-opacity", "0.34");
      const stopBottom = document.createElementNS(svgNS, "stop");
      stopBottom.setAttribute("offset", "100%");
      stopBottom.setAttribute("stop-color", "#60a5fa");
      stopBottom.setAttribute("stop-opacity", "0");
      gradient.appendChild(stopTop);
      gradient.appendChild(stopBottom);
      defs.appendChild(gradient);
      svg.appendChild(defs);

      for (let index = 0; index < 5; index += 1) {
        const y = padding.top + (plotHeight / 4) * index;
        const line = document.createElementNS(svgNS, "line");
        line.setAttribute("x1", String(padding.left));
        line.setAttribute("x2", String(padding.left + plotWidth));
        line.setAttribute("y1", String(y));
        line.setAttribute("y2", String(y));
        line.setAttribute("stroke", "rgba(148, 163, 184, 0.12)");
        line.setAttribute("stroke-width", "1");
        line.setAttribute("stroke-dasharray", "2 5");
        svg.appendChild(line);

        const value = domainMax - (range / 4) * index;
        const label = document.createElementNS(svgNS, "text");
        label.setAttribute("x", String(padding.left + plotWidth + 6));
        label.setAttribute("y", String(y + 3.6));
        label.setAttribute("fill", "rgba(148, 163, 184, 0.86)");
        label.setAttribute("font-size", "10");
        label.textContent = formatChartAxisPrice(value);
        svg.appendChild(label);
      }

      const trendPoints = xPositions.map((x, index) => ({ x, y: closeYs[index] }));
      svg.appendChild(createKlineTrendArea(svgNS, trendPoints, padding.top + plotHeight, gradientId));

      candles.forEach((candle, index) => {
        const x = xPositions[index];
        const openY = toChartY(candle.open, domainMin, range, padding.top, plotHeight);
        const closeY = toChartY(candle.close, domainMin, range, padding.top, plotHeight);
        const highY = toChartY(candle.high, domainMin, range, padding.top, plotHeight);
        const lowY = toChartY(candle.low, domainMin, range, padding.top, plotHeight);
        const isUp = candle.close >= candle.open;
        const color = isUp ? "#34d399" : "#f87171";

        const wick = document.createElementNS(svgNS, "line");
        wick.setAttribute("x1", String(x));
        wick.setAttribute("x2", String(x));
        wick.setAttribute("y1", String(highY));
        wick.setAttribute("y2", String(lowY));
        wick.setAttribute("stroke", color);
        wick.setAttribute("stroke-width", "1.1");
        wick.setAttribute("stroke-linecap", "round");
        wick.setAttribute("stroke-opacity", "0.9");
        svg.appendChild(wick);

        const body = document.createElementNS(svgNS, "rect");
        body.setAttribute("x", String(x - candleWidth / 2));
        body.setAttribute("y", String(Math.min(openY, closeY)));
        body.setAttribute("width", String(candleWidth));
        body.setAttribute("height", String(Math.max(Math.abs(closeY - openY), 1.4)));
        body.setAttribute("rx", "1.2");
        body.setAttribute("fill", color);
        body.setAttribute("fill-opacity", isUp ? "0.96" : "0.9");
        svg.appendChild(body);
      });

      buildKlineXAxisLabels(candles, xPositions).forEach((tick) => {
        const text = document.createElementNS(svgNS, "text");
        text.setAttribute("x", String(tick.x));
        text.setAttribute("y", "212");
        text.setAttribute("text-anchor", tick.anchor);
        text.setAttribute("fill", "rgba(148, 163, 184, 0.86)");
        text.setAttribute("font-size", "10");
        text.textContent = tick.label;
        svg.appendChild(text);
      });

      const crosshairV = document.createElementNS(svgNS, "line");
      crosshairV.setAttribute("stroke", "rgba(148, 163, 184, 0.65)");
      crosshairV.setAttribute("stroke-width", "1");
      crosshairV.setAttribute("stroke-dasharray", "3 3");
      crosshairV.setAttribute("visibility", "hidden");
      crosshairV.setAttribute("pointer-events", "none");
      svg.appendChild(crosshairV);

      const crosshairDot = document.createElementNS(svgNS, "circle");
      crosshairDot.setAttribute("r", "3.4");
      crosshairDot.setAttribute("fill", "#60a5fa");
      crosshairDot.setAttribute("stroke", "#0f172a");
      crosshairDot.setAttribute("stroke-width", "1.4");
      crosshairDot.setAttribute("visibility", "hidden");
      crosshairDot.setAttribute("pointer-events", "none");
      svg.appendChild(crosshairDot);

      svg.__klineLayout = {
        padding,
        plotWidth,
        plotHeight,
        xPositions,
        closeYs,
        crosshairV,
        crosshairDot
      };

      return svg;
    }

    function createKlineTrendArea(svgNS, points, baselineY, gradientId) {
      const polygon = document.createElementNS(svgNS, "polygon");
      const areaPoints = points
        .map((point) => point.x.toFixed(2) + "," + point.y.toFixed(2))
        .concat([
          points[points.length - 1].x.toFixed(2) + "," + baselineY.toFixed(2),
          points[0].x.toFixed(2) + "," + baselineY.toFixed(2)
        ])
        .join(" ");
      polygon.setAttribute("points", areaPoints);
      polygon.setAttribute("fill", gradientId ? "url(#" + gradientId + ")" : "rgba(96, 165, 250, 0.12)");
      polygon.setAttribute("pointer-events", "none");
      return polygon;
    }

    function buildKlineTrendPolylinePoints(points) {
      return points.map((point) => point.x.toFixed(2) + "," + point.y.toFixed(2)).join(" ");
    }

    function buildKlineXAxisLabels(candles, xPositions) {
      const indices = Array.from(new Set([0, Math.floor((candles.length - 1) / 2), candles.length - 1]));
      return indices.map((index) => {
        const label = formatKlineTimeLabel(candles[index].timestamp);
        const position = candles.length === 1 || !xPositions
          ? 180
          : xPositions[index];
        return {
          x: position,
          label,
          anchor: index === 0 ? "start" : index === candles.length - 1 ? "end" : "middle"
        };
      });
    }

    function formatKlineTimeLabel(timestamp) {
      return window.AgentWalletMarket.formatTimeLabel(timestamp);
    }

    function toChartY(value, minPrice, range, top, height) {
      return window.AgentWalletMarket.toChartY(value, minPrice, range, top, height);
    }

    function formatChartPrice(value) {
      return window.AgentWalletMarket.formatChartPrice(value);
    }

    function formatChartAxisPrice(value) {
      return window.AgentWalletMarket.formatChartAxisPrice(value);
    }

    function formatUSDPrice(value) {
      return window.AgentWalletMarket.formatUSDPrice(value);
    }

    function formatPercent(value) {
      return window.AgentWalletMarket.formatPercent(value);
    }

    function formatCompactCurrency(value) {
      return window.AgentWalletMarket.formatCompact(value, true);
    }

    function formatCompactAmount(value) {
      return window.AgentWalletMarket.formatCompact(value, false);
    }

    function formatUpdatedTime(value) {
      return window.AgentWalletMarket.formatUpdatedTime(value);
    }

    function scrollThreadToBottom(tab) {
      const threadEl = getThreadEl(tab);
      if (!threadEl) {
        return;
      }
      threadEl.scrollTop = threadEl.scrollHeight;
    }

    function scrollConversationToBottom(tab) {
      scrollThreadToBottom(tab);

      const viewEl = getViewEl(tab);
      if (!viewEl) {
        return;
      }

      viewEl.scrollTop = viewEl.scrollHeight;
    }

    function autoResize() {
      inputEl.style.height = "auto";
      inputEl.style.height = Math.min(inputEl.scrollHeight, 144) + "px";
    }

    async function ensureSession(traceId) {
      if (state.sessionId) {
        return state.sessionId;
      }

      const response = await tracedFetch("/v1/sessions", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({})
      }, traceId);
      if (!response.ok) {
        throw new Error("创建会话失败");
      }

      const payload = await response.json();
      state.sessionId = payload.session_id || "";
      if (!state.sessionId) {
        throw new Error("服务端没有返回 session_id");
      }

      window.sessionStorage.setItem("agent-session-id", state.sessionId);
      return state.sessionId;
    }

    async function sendMessage(tab, text) {
      if (!aiChatEnabled) {
        return;
      }
      state.busyTab = tab;
      sendButton.disabled = true;

      const messages = state.messagesByTab[tab] || [];
      const userMessage = { role: "user", content: text, toolExecutions: [] };
      const pendingMessage = {
        role: "assistant",
        content: "正在思考中",
        pending: true,
        toolExecutions: []
      };

      messages.push(userMessage, pendingMessage);
      state.messagesByTab[tab] = messages;
      renderThread(tab);

      try {
        const traceId = createBrowserTraceId();
        const sessionId = await ensureSession(traceId);
        const response = await tracedFetch("/v1/agent/runs", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            session_id: sessionId,
            input: buildAgentInput(tab, text)
          })
        }, traceId);

        const payload = await response.json();
        if (!response.ok) {
          throw new Error(payload.error || "请求失败");
        }

        pendingMessage.pending = false;
        pendingMessage.content = payload.message || "已完成，但模型没有返回消息。";
        pendingMessage.toolExecutions = Array.isArray(payload.tool_executions) ? payload.tool_executions : [];
      } catch (error) {
        pendingMessage.role = "system";
        pendingMessage.pending = false;
        pendingMessage.content = error instanceof Error ? error.message : "请求失败";
      } finally {
        state.busyTab = "";
        if (aiChatEnabled && chatTabs.includes(state.activeTab)) {
          sendButton.disabled = false;
        }
        renderThread(tab);
        if (state.activeTab === tab) {
          inputEl.focus();
        }
      }
    }

    function buildAgentInput(tab, text) {
      const meta = pageMeta[tab];
      return "你现在在 " + tab + " 页面，请结合「" + meta.context + "」这个业务场景，用简洁中文回答用户。\n\n用户问题：" + text;
    }

    function loadMessagesByTab() {
      try {
        const raw = window.sessionStorage.getItem("agent-chat-history-by-tab");
        if (!raw) {
          return {};
        }
        const parsed = JSON.parse(raw);
        return parsed && typeof parsed === "object" ? parsed : {};
      } catch (_) {
        return {};
      }
    }

    function persistMessagesByTab() {
      const payload = {};
      chatTabs.forEach((tab) => {
        payload[tab] = (state.messagesByTab[tab] || []).filter((message) => !message.pending);
      });
      window.sessionStorage.setItem("agent-chat-history-by-tab", JSON.stringify(payload));
    }
  
