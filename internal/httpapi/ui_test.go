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
	for _, marker := range []string{"parseKlineObservation", "createKlineChartSVG", "createKlineChart(", "bindKlineInteractions", "renderKlineDetail", "kline-detail-panel", "kline-tooltip", "buildKlineTrendPolylinePoints", "wallet-lock", "wallet-password", "wallet-reset-button", "wallet-recovery-copy", "wallet-choice-step", "wallet-import-paste", "wallet-generate-choice", "parseMnemonicWords", "confirmWalletReset", "copyTextToClipboard", "WalletVaultStore", "web3_wallet", "AES-256-GCM", "PBKDF2-SHA-256", "generateBIP39Mnemonic", "deriveBIP39Seed", "createDefaultBIP44Keyring", "m/44'/60'/0'/0/0", "m/44'/501'/0'/0'", "walletSessionTimeoutMs", "kline-card", "market-spotlight", "page-status", "market-top-list", "ensureMarketTopList", "trade-dashboard-content", "openTradeDashboard", "trade-kline-panel", "refreshTradeKlines", "15m candles · refresh 60s", "}, 60000);", "bottom: calc(84px + env(safe-area-inset-bottom));", "max-height: calc(100dvh - 48px - env(safe-area-inset-top) - env(safe-area-inset-bottom));"} {
		if !strings.Contains(body, marker) {
			t.Fatalf("expected %s in body", marker)
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
