package market

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"web3-service-agent/internal/tool"
)

func TestRegisterLoadsDefinitionFromFile(t *testing.T) {
	registry := tool.NewRegistry()
	if err := Register(registry); err != nil {
		t.Fatalf("register market tool: %v", err)
	}

	definitions := registry.Definitions()
	if len(definitions) != 1 {
		t.Fatalf("expected 1 tool definition, got %d", len(definitions))
	}
	if definitions[0].Name != marketTokenOverviewToolName {
		t.Fatalf("unexpected tool name: %q", definitions[0].Name)
	}
}

func TestHandleMarketTokenOverviewByToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/search":
			if got := r.URL.Query().Get("query"); got != "btc" {
				t.Fatalf("unexpected search query: %q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"coins":[{"id":"bitcoin","name":"Bitcoin","symbol":"btc","api_symbol":"btc","market_cap_rank":1}]}`))
		case "/markets":
			if got := r.URL.Query().Get("ids"); got != "bitcoin" {
				t.Fatalf("unexpected ids query: %q", got)
			}
			if got := r.URL.Query().Get("vs_currency"); got != "usd" {
				t.Fatalf("unexpected vs_currency query: %q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"id":"bitcoin","symbol":"btc","name":"Bitcoin","current_price":118888.12,"market_cap":2300000000000,"total_volume":51000000000,"circulating_supply":19700000,"total_supply":21000000,"fully_diluted_valuation":2500000000000,"market_cap_rank":1}]`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	originalSearchURL := coingeckoSearchURL
	originalMarketsURL := coingeckoMarketsURL
	originalClient := marketHTTPClient
	coingeckoSearchURL = server.URL + "/search"
	coingeckoMarketsURL = server.URL + "/markets"
	marketHTTPClient = &http.Client{Timeout: 5 * time.Second}
	defer func() {
		coingeckoSearchURL = originalSearchURL
		coingeckoMarketsURL = originalMarketsURL
		marketHTTPClient = originalClient
	}()

	result, err := handleMarketTokenOverview(context.Background(), tool.Call{
		Arguments: []byte(`{"token":"btc","vs_currency":"usd"}`),
	})
	if err != nil {
		t.Fatalf("market token overview by token: %v", err)
	}

	if !strings.Contains(result.Content, "Market overview for Bitcoin (BTC):") {
		t.Fatalf("unexpected result: %q", result.Content)
	}
	if !strings.Contains(result.Content, "price_usd: 118888.12") {
		t.Fatalf("missing price in result: %q", result.Content)
	}
	if !strings.Contains(result.Content, "total_supply: 21000000.00") {
		t.Fatalf("missing total supply in result: %q", result.Content)
	}
}

func TestHandleMarketTokenOverviewByAddress(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/latest/dex/tokens/So11111111111111111111111111111111111111112" {
			t.Fatalf("unexpected path: %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"pairs":[
				{
					"chainId":"ethereum",
					"dexId":"uniswap",
					"url":"https://dex.example/eth",
					"pairAddress":"0xpair-eth",
					"baseToken":{"address":"0xeth","name":"Wrapped Example","symbol":"WEX"},
					"priceUsd":"1.10",
					"priceNative":"0.0004",
					"liquidity":{"usd":1500},
					"fdv":1000000,
					"marketCap":900000
				},
				{
					"chainId":"solana",
					"dexId":"raydium",
					"url":"https://dex.example/sol",
					"pairAddress":"SoPair",
					"baseToken":{"address":"So11111111111111111111111111111111111111112","name":"Wrapped SOL","symbol":"SOL"},
					"priceUsd":"180.12",
					"priceNative":"1.0000",
					"liquidity":{"usd":520000},
					"fdv":95000000,
					"marketCap":87000000
				}
			]
		}`))
	}))
	defer server.Close()

	originalDexURL := dexScreenerTokenURL
	originalClient := marketHTTPClient
	dexScreenerTokenURL = server.URL + "/latest/dex/tokens"
	marketHTTPClient = &http.Client{Timeout: 5 * time.Second}
	defer func() {
		dexScreenerTokenURL = originalDexURL
		marketHTTPClient = originalClient
	}()

	result, err := handleMarketTokenOverview(context.Background(), tool.Call{
		Arguments: []byte(`{"chain":"solana","token_address":"So11111111111111111111111111111111111111112"}`),
	})
	if err != nil {
		t.Fatalf("market token overview by address: %v", err)
	}

	if !strings.Contains(result.Content, "Market overview for Wrapped SOL (SOL) on solana:") {
		t.Fatalf("unexpected result: %q", result.Content)
	}
	if !strings.Contains(result.Content, "liquidity_usd: 520000.00") {
		t.Fatalf("missing liquidity in result: %q", result.Content)
	}
	if !strings.Contains(result.Content, "source: DexScreener") {
		t.Fatalf("missing source in result: %q", result.Content)
	}
}

func TestHandleMarketTokenOverviewRequiresSelector(t *testing.T) {
	_, err := handleMarketTokenOverview(context.Background(), tool.Call{Arguments: []byte(`{}`)})
	if err == nil {
		t.Fatal("expected validation error")
	}
}
