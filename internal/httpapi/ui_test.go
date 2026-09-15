package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRootServesMobileUI(t *testing.T) {
	handler := New(nil, nil)

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if got := recorder.Header().Get("Content-Type"); !strings.Contains(got, "text/html") {
		t.Fatalf("unexpected content type: %q", got)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, "web3-service-agent") {
		t.Fatalf("unexpected body: %q", body)
	}
	if !strings.Contains(body, "tabbar") {
		t.Fatalf("expected mobile UI markup in body")
	}
	for _, marker := range []string{`data-tab="home"`, `data-tab="market"`, `data-tab="trade"`, `data-tab="pay"`, `data-tab="asset"`} {
		if !strings.Contains(body, marker) {
			t.Fatalf("expected %s in body", marker)
		}
	}
	for _, marker := range []string{"parseKlineObservation", "createKlineChartSVG", "createKlineChart(", "bindKlineInteractions", "renderKlineDetail", "kline-detail-panel", "kline-tooltip", "buildKlineTrendPolylinePoints", "wallet-lock", "wallet-password", "wallet-reset-button", "wallet-recovery-copy", "wallet-choice-step", "wallet-import-paste", "wallet-generate-choice", "parseMnemonicWords", "confirmWalletReset", "wallet-export-mnemonic-button", "requestWalletMnemonicExport", "exportWalletMnemonic", "验证并显示", "wallet-account-menu-button", "wallet-account-dialog", "createBIP44Networks", "addWalletAccount", "selectWalletAccount", "saveWalletAccountName", "persistWalletKeyring", "copyTextToClipboard", "copySelectedChainAddress", "chain-selector-button", "chain-selector-menu", "wallet-header-address", "wallet-header-copy", "wallet_derivation.js", "evm_signer.js", "renderChainSelector", "renderWalletHeader", "getSelectedChainAddress", "maskAddress", "asset-action-card", `data-asset-action="send"`, `data-asset-action="receive"`, "asset-action-dialog", "asset-send-max", "prepareAssetSendReview", "renderAssetTransactionReview", "confirmAndBroadcastAssetTransaction", "/v1/wallet/evm/prepare", "/v1/wallet/evm/broadcast", "renderAssetTokenPicker", "selectAssetActionToken", "createQRCodeMatrix", "asset-balance-card", "asset-balance-total", "refreshAssetBalances", "buildSelectedBalanceAddressMap", "WalletVaultStore", "web3_wallet", "AES-256-GCM", "PBKDF2-SHA-256", "generateBIP39Mnemonic", "deriveBIP39Seed", "createDefaultBIP44Keyring", "m/44'/60'/", "m/44'/501'/", "walletSessionTimeoutMs = 30 * 60 * 1000", "30 分钟无操作", "kline-card", "market-spotlight", "page-status", "market-top-list", "ensureMarketTopList", "trade-dashboard-content", "openTradeDashboard", "trade-kline-panel", "refreshTradeKlines", "15m candles · refresh 60s", "}, 60000);", "bottom: calc(84px + env(safe-area-inset-bottom));", "max-height: calc(100dvh - 48px - env(safe-area-inset-top) - env(safe-area-inset-bottom));"} {
		if !strings.Contains(body, marker) {
			t.Fatalf("expected %s in body", marker)
		}
	}
	for _, marker := range []string{`data-asset-action="history"`, "wallet-history-dialog", "wallet-history-chain-trigger", "wallet-history-type-trigger", "wallet-history-menu-option", "wallet-history-list", "openWalletHistory", "refreshWalletTransactionHistory", "/v1/wallet/evm/history", "renderWalletHistory", "recordBroadcastTransaction", "web3_wallet_transaction_history_v1", "去浏览器查看"} {
		if !strings.Contains(body, marker) {
			t.Fatalf("expected transaction history marker %s in body", marker)
		}
	}
}

func TestServesBIP39Wordlist(t *testing.T) {
	handler := New(nil, nil)
	request := httptest.NewRequest(http.MethodGet, "/wallet/bip39-english", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if got := recorder.Header().Get("Content-Type"); !strings.Contains(got, "text/plain") {
		t.Fatalf("unexpected content type: %q", got)
	}
	if lines := strings.Count(strings.TrimSpace(recorder.Body.String()), "\n") + 1; lines != 2048 {
		t.Fatalf("expected 2048 BIP39 words, got %d", lines)
	}
}

func TestServesWalletDerivationScript(t *testing.T) {
	handler := New(nil, nil)
	request := httptest.NewRequest(http.MethodGet, "/ui/wallet_derivation.js", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if got := recorder.Header().Get("Content-Type"); !strings.Contains(got, "javascript") && !strings.Contains(got, "text/plain") {
		t.Fatalf("unexpected content type: %q", got)
	}
	body := recorder.Body.String()
	for _, marker := range []string{"WalletAddressDerivation", "Invalid BIP44 account index", "deriveAddress:lc"} {
		if !strings.Contains(body, marker) {
			t.Fatalf("expected derivation bundle marker %q", marker)
		}
	}
}

func TestServesEVMSignerScript(t *testing.T) {
	handler := New(nil, nil)
	request := httptest.NewRequest(http.MethodGet, "/ui/evm_signer.js", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if got := recorder.Header().Get("Content-Type"); !strings.Contains(got, "javascript") && !strings.Contains(got, "text/plain") {
		t.Fatalf("unexpected content type: %q", got)
	}
	if body := recorder.Body.String(); !strings.Contains(body, "WalletEVMSigner") || !strings.Contains(body, "signTransaction") {
		t.Fatalf("expected EVM signer bundle")
	}
}
