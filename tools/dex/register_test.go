package dex

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"web3-service-agent/internal/config"
	"web3-service-agent/internal/marketdata"
	"web3-service-agent/internal/tool"
)

func TestRegisterLoadsDefinitionFromFile(t *testing.T) {
	registry := tool.NewRegistry()
	if err := Register(registry); err != nil {
		t.Fatalf("register dex tool: %v", err)
	}

	definitions := registry.Definitions()
	if len(definitions) != 2 {
		t.Fatalf("expected 2 tool definitions, got %d", len(definitions))
	}
	if definitions[0].Name != dexTokenTradeDashboardToolName {
		t.Fatalf("unexpected first tool name: %q", definitions[0].Name)
	}
	if definitions[1].Name != dexTokenTradeKlinesToolName {
		t.Fatalf("unexpected second tool name: %q", definitions[1].Name)
	}
}

func TestHandleTokenTradeDashboardByToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/search":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"coins":[{"id":"bitcoin","name":"Bitcoin","symbol":"btc","api_symbol":"btc","market_cap_rank":1}]}`))
		case "/coins/markets":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"asset_id":"bitcoin","id":"bitcoin","symbol":"btc","name":"Bitcoin","current_price":118888.12,"price_change_percentage_24h":3.21,"high_24h":120100,"low_24h":116400,"total_volume":51000000000,"market_cap":2300000000000,"fully_diluted_valuation":2500000000000,"circulating_supply":19700000,"total_supply":21000000,"market_cap_rank":1}]`))
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

	result, err := handleTokenTradeDashboard(context.Background(), tool.Call{
		Arguments: []byte(`{"token":"btc","vs_currency":"usd"}`),
	})
	if err != nil {
		t.Fatalf("trade dashboard by token: %v", err)
	}

	for _, marker := range []string{"Trade dashboard for Bitcoin (BTC):", "24h_change_pct: +3.21%", "24h_high_usd: 120100.00", "market_cap_rank: 1"} {
		if !strings.Contains(result.Content, marker) {
			t.Fatalf("missing %s in result: %q", marker, result.Content)
		}
	}
}

func TestHandleTokenTradeDashboardRequiresSelector(t *testing.T) {
	_, err := handleTokenTradeDashboard(context.Background(), tool.Call{Arguments: []byte(`{}`)})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestHandleTokenTradeKlinesByAssetID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/coins/markets":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"id":"bitcoin","symbol":"btc","name":"Bitcoin","current_price":118888.12,"price_change_percentage_24h":3.21,"high_24h":120100,"low_24h":116400,"total_volume":51000000000,"market_cap":2300000000000,"fully_diluted_valuation":2500000000000,"circulating_supply":19700000,"total_supply":21000000,"market_cap_rank":1}]`))
		case "/coins/bitcoin/market_chart":
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

	result, err := handleTokenTradeKlines(context.Background(), tool.Call{
		Arguments: []byte(`{"asset_id":"bitcoin","interval":"15m","limit":8}`),
	})
	if err != nil {
		t.Fatalf("trade klines by asset id: %v", err)
	}

	for _, marker := range []string{"Trade klines for Bitcoin (BTC):", "- interval: 15m", "2024-07-17T00:00:00Z open=100.0000 high=102.0000 low=100.0000 close=102.0000", "2024-07-17T00:30:00Z open=105.0000 high=105.0000 low=105.0000 close=105.0000"} {
		if !strings.Contains(result.Content, marker) {
			t.Fatalf("missing %s in result: %q", marker, result.Content)
		}
	}
}
