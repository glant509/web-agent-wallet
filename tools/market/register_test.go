package market

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"web3-service-agent/internal/config"
	"web3-service-agent/internal/marketdata"
	"web3-service-agent/internal/tool"
)

func TestRegisterLoadsDefinitionFromFile(t *testing.T) {
	registry := tool.NewRegistry()
	if err := Register(registry); err != nil {
		t.Fatalf("register market tool: %v", err)
	}

	definitions := registry.Definitions()
	if len(definitions) != 2 {
		t.Fatalf("expected 2 tool definitions, got %d", len(definitions))
	}
	if definitions[0].Name != marketTokenKlinesToolName {
		t.Fatalf("unexpected first tool name: %q", definitions[0].Name)
	}
	if definitions[1].Name != marketTokenOverviewToolName {
		t.Fatalf("unexpected second tool name: %q", definitions[1].Name)
	}
}

func TestHandleMarketTokenKlinesByToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/search":
			if got := r.URL.Query().Get("query"); got != "btc" {
				t.Fatalf("unexpected search query: %q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"coins":[{"id":"bitcoin","name":"Bitcoin","symbol":"btc","api_symbol":"btc","market_cap_rank":1}]}`))
		case "/coins/bitcoin/market_chart":
			if got := r.URL.Query().Get("vs_currency"); got != "usd" {
				t.Fatalf("unexpected vs_currency query: %q", got)
			}
			if got := r.URL.Query().Get("days"); got != "2" {
				t.Fatalf("unexpected days query: %q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"prices":[
					[1721174400000,100],
					[1721175600000,110],
					[1721177700000,90],
					[1721178600000,105],
					[1721180100000,120]
				]
			}`))
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

	result, err := handleMarketTokenKlines(context.Background(), tool.Call{
		Arguments: []byte(`{"token":"btc","days":2,"interval":"1h","limit":10}`),
	})
	if err != nil {
		t.Fatalf("market token klines by token: %v", err)
	}

	if !strings.Contains(result.Content, "Klines for Bitcoin (BTC):") {
		t.Fatalf("unexpected result: %q", result.Content)
	}
	if !strings.Contains(result.Content, "- interval: 1h") {
		t.Fatalf("missing interval in result: %q", result.Content)
	}
	if !strings.Contains(result.Content, "2024-07-17T00:00:00Z open=100.0000 high=110.0000 low=90.0000 close=90.0000") {
		t.Fatalf("missing first candle in result: %q", result.Content)
	}
	if !strings.Contains(result.Content, "2024-07-17T01:00:00Z open=105.0000 high=120.0000 low=105.0000 close=120.0000") {
		t.Fatalf("missing second candle in result: %q", result.Content)
	}
}

func TestHandleMarketTokenKlinesByAddressHistorical(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/coins/base/contract/0xabc/market_chart/range" {
			t.Fatalf("unexpected path: %q", r.URL.Path)
		}
		if got := r.URL.Query().Get("vs_currency"); got != "usd" {
			t.Fatalf("unexpected vs_currency query: %q", got)
		}
		if got := r.URL.Query().Get("from"); got != "1721174400" {
			t.Fatalf("unexpected from query: %q", got)
		}
		if got := r.URL.Query().Get("to"); got != "1721179800" {
			t.Fatalf("unexpected to query: %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"prices":[
				[1721174400000,1.00],
				[1721175000000,1.20],
				[1721176200000,1.10],
				[1721177100000,1.30],
				[1721178000000,1.15]
			]
		}`))
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

	result, err := handleMarketTokenKlines(context.Background(), tool.Call{
		Arguments: []byte(`{"chain":"base","token_address":"0xabc","from_timestamp":1721174400,"to_timestamp":1721179800,"interval":"30m","limit":2}`),
	})
	if err != nil {
		t.Fatalf("market token historical klines by address: %v", err)
	}

	if !strings.Contains(result.Content, "Klines for 0xabc on base:") {
		t.Fatalf("unexpected result: %q", result.Content)
	}
	if !strings.Contains(result.Content, "- mode: historical") {
		t.Fatalf("missing historical mode in result: %q", result.Content)
	}
	if !strings.Contains(result.Content, "- note: showing latest 2 candles from requested range") {
		t.Fatalf("missing truncation note in result: %q", result.Content)
	}
	if !strings.Contains(result.Content, "2024-07-17T00:30:00Z open=1.1000 high=1.3000 low=1.1000 close=1.3000") {
		t.Fatalf("missing second candle in result: %q", result.Content)
	}
	if !strings.Contains(result.Content, "2024-07-17T01:00:00Z open=1.1500 high=1.1500 low=1.1500 close=1.1500") {
		t.Fatalf("missing third candle in result: %q", result.Content)
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
		case "/coins/markets":
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

	originalService := marketdata.Default()
	service, err := marketdata.NewService(config.MarketProvider{Used: true, Name: "coingecko", BaseURL: server.URL})
	if err != nil {
		t.Fatalf("create marketdata service: %v", err)
	}
	marketdata.SetDefaultForTest(service)
	defer func() {
		marketdata.SetDefaultForTest(originalService)
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

func TestHandleMarketTokenKlinesRequiresChainForAddress(t *testing.T) {
	_, err := handleMarketTokenKlines(context.Background(), tool.Call{
		Arguments: []byte(`{"token_address":"0xabc"}`),
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestHandleMarketTokenKlinesRequiresCompleteHistoricalRange(t *testing.T) {
	_, err := handleMarketTokenKlines(context.Background(), tool.Call{
		Arguments: []byte(`{"token":"btc","from_timestamp":1721174400}`),
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}
