package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"web3-service-agent/internal/config"
	"web3-service-agent/internal/marketdata"
)

func TestHandleMarketTop(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/coins/markets" {
			t.Fatalf("unexpected path: %q", r.URL.Path)
		}
		if got := r.URL.Query().Get("vs_currency"); got != "usd" {
			t.Fatalf("unexpected vs_currency: %q", got)
		}
		if got := r.URL.Query().Get("order"); got != "market_cap_desc" {
			t.Fatalf("unexpected order: %q", got)
		}
		if got := r.URL.Query().Get("per_page"); got != "10" {
			t.Fatalf("unexpected per_page: %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"name":"Bitcoin","symbol":"btc","current_price":118888.12,"price_change_percentage_24h":3.21,"market_cap_rank":1},
			{"name":"Ethereum","symbol":"eth","current_price":3888.56,"price_change_percentage_24h":2.45,"market_cap_rank":2}
		]`))
	}))
	defer server.Close()

	originalService := marketdata.Default()
	service, err := marketdata.NewService(config.MarketProvider{Used: true, Name: "coingecko", BaseURL: server.URL})
	if err != nil {
		t.Fatalf("create marketdata service: %v", err)
	}
	marketdata.SetDefaultForTest(service)
	defer func() {
		marketdata.SetDefaultForTest(originalService)
	}()

	handler := New(nil, nil)
	request := httptest.NewRequest(http.MethodGet, "/v1/market/top", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	body := recorder.Body.String()
	for _, marker := range []string{`"source":"CoinGecko"`, `"name":"Bitcoin"`, `"current_price":118888.12`, `"price_change_percentage_24h":3.21`} {
		if !strings.Contains(body, marker) {
			t.Fatalf("expected %s in body: %s", marker, body)
		}
	}
}
