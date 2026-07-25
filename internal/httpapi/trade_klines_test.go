package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"web3-service-agent/internal/config"
	"web3-service-agent/internal/marketdata"
)

func TestHandleTradeKlines(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/coins/markets":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"id":"bitcoin","symbol":"btc","name":"Bitcoin","current_price":118888.12,"price_change_percentage_24h":3.21,"high_24h":120100,"low_24h":116400,"total_volume":51000000000,"market_cap":2300000000000,"fully_diluted_valuation":2500000000000,"circulating_supply":19700000,"total_supply":21000000,"market_cap_rank":1}]`))
		case "/coins/bitcoin/market_chart":
			if got := r.URL.Query().Get("days"); got != "1" {
				t.Fatalf("unexpected days: %q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"prices":[[1721174400000,100],[1721175000000,102],[1721175600000,101],[1721176200000,105]]}`))
		default:
			http.NotFound(w, r)
		}
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
	request := httptest.NewRequest(http.MethodGet, "/v1/trade/klines?id=bitcoin&interval=15m&limit=8", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	body := recorder.Body.String()
	for _, marker := range []string{`"source":"CoinGecko"`, `"asset_id":"bitcoin"`, `"interval":"15m"`, `"close":105`} {
		if !strings.Contains(body, marker) {
			t.Fatalf("expected %s in body: %s", marker, body)
		}
	}
}
