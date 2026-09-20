import * as WalletCore from "./wallet-core/index";
import { createAPIClient } from "./api-client/index";
import * as UIComponents from "./ui-components/index";
import * as Biometrics from "./biometric-auth/index";
import * as WalletAccounts from "./wallet-accounts/index";
import * as WalletHistory from "./wallet-history/index";
import * as Transactions from "./transactions/index";
import * as AppState from "./app-state/index";
import * as Market from "./market/index";
import * as WalletHistoryController from "./controllers/wallet-history";
import * as QRCode from "./qr-code/index";
import * as AssetTransferController from "./controllers/asset-transfer";
import * as AssetBalanceController from "./controllers/asset-balance";
import * as WalletAccountController from "./controllers/wallet-accounts";
import * as WalletBiometricController from "./controllers/wallet-biometrics";
import { createWalletVaultStore } from "./wallet-vault-store/index";
import * as WalletSessionController from "./controllers/wallet-session";
import * as WalletSecretExportController from "./controllers/wallet-secret-export";
import * as WalletVaultController from "./controllers/wallet-vault";

declare global {
  interface Window {
    AgentWalletCore: typeof WalletCore;
    AgentWalletAPI: ReturnType<typeof createAPIClient>;
    AgentWalletUI: typeof UIComponents;
    AgentWalletBiometrics: typeof Biometrics;
    AgentWalletAccounts: typeof WalletAccounts;
    AgentWalletHistory: typeof WalletHistory;
    AgentWalletTransactions: typeof Transactions;
    AgentWalletState: typeof AppState;
    AgentWalletMarket: typeof Market;
    AgentWalletHistoryController: typeof WalletHistoryController;
    AgentWalletQRCode: typeof QRCode;
    AgentWalletAssetTransferController: typeof AssetTransferController;
    AgentWalletAssetBalanceController: typeof AssetBalanceController;
    AgentWalletAccountController: typeof WalletAccountController;
    AgentWalletBiometricController: typeof WalletBiometricController;
    AgentWalletVaultStore: ReturnType<typeof createWalletVaultStore>;
    AgentWalletSessionController: typeof WalletSessionController;
    AgentWalletSecretExportController: typeof WalletSecretExportController;
    AgentWalletVaultController: typeof WalletVaultController;
  }
}

window.AgentWalletCore = Object.freeze(WalletCore);
window.AgentWalletAPI = createAPIClient();
window.AgentWalletUI = Object.freeze(UIComponents);
window.AgentWalletBiometrics = Object.freeze(Biometrics);
window.AgentWalletAccounts = Object.freeze(WalletAccounts);
window.AgentWalletHistory = Object.freeze(WalletHistory);
window.AgentWalletTransactions = Object.freeze(Transactions);
window.AgentWalletState = Object.freeze(AppState);
window.AgentWalletMarket = Object.freeze(Market);
window.AgentWalletHistoryController = Object.freeze(WalletHistoryController);
window.AgentWalletQRCode = Object.freeze(QRCode);
window.AgentWalletAssetTransferController = Object.freeze(AssetTransferController);
window.AgentWalletAssetBalanceController = Object.freeze(AssetBalanceController);
window.AgentWalletAccountController = Object.freeze(WalletAccountController);
window.AgentWalletBiometricController = Object.freeze(WalletBiometricController);
window.AgentWalletVaultStore = createWalletVaultStore();
window.AgentWalletSessionController = Object.freeze(WalletSessionController);
window.AgentWalletSecretExportController = Object.freeze(WalletSecretExportController);
window.AgentWalletVaultController = Object.freeze(WalletVaultController);
