package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"web3-service-agent/internal/config"
	"web3-service-agent/internal/marketdata"
	wallettool "web3-service-agent/tools/wallet"
)

type assetPortfolioFailingMarketService struct{}

func (assetPortfolioFailingMarketService) ProviderName() string { return "failing" }
func (assetPortfolioFailingMarketService) SearchCoin(context.Context, string) (marketdata.CoinSearchItem, error) {
	return marketdata.CoinSearchItem{}, nil
}
func (assetPortfolioFailingMarketService) FetchMarket(context.Context, string, string) (marketdata.MarketAsset, error) {
	return marketdata.MarketAsset{}, errors.New("price lookup should not be called")
}
func (assetPortfolioFailingMarketService) FetchMarketChart(context.Context, string, string, int) ([][]float64, error) {
	return nil, nil
}
func (assetPortfolioFailingMarketService) FetchContractMarketChart(context.Context, string, string, string, int) ([][]float64, error) {
	return nil, nil
}
func (assetPortfolioFailingMarketService) FetchContractMarketChartRange(context.Context, string, string, string, int64, int64) ([][]float64, error) {
	return nil, nil
}
func (assetPortfolioFailingMarketService) FetchMarketChartRange(context.Context, string, string, int64, int64) ([][]float64, error) {
	return nil, nil
}
func (assetPortfolioFailingMarketService) FetchTopMarkets(context.Context, string, int) ([]marketdata.MarketAsset, error) {
	return nil, nil
}

func TestHandleAssetPortfolioRequiresBodyFields(t *testing.T) {
	handler := New(nil, nil)
	request := httptest.NewRequest(http.MethodPost, "/v1/asset/portfolio", strings.NewReader(`{"addresses":{}}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "selected_chain_id is required") {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}

func TestHandleAssetPortfolioReturnsBalances(t *testing.T) {
	originalService := marketdata.Default()
	marketdata.SetDefaultForTest(assetPortfolioFailingMarketService{})
	defer marketdata.SetDefaultForTest(originalService)

	rpcServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Method string            `json:"method"`
			Params []json.RawMessage `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode rpc request: %v", err)
		}

		result := "0x0"
		switch payload.Method {
		case "eth_getBalance":
			result = "0xde0b6b3a7640000"
		case "eth_call":
			var call map[string]string
			if err := json.Unmarshal(payload.Params[0], &call); err != nil {
				t.Fatalf("decode eth_call params: %v", err)
			}
			if strings.EqualFold(call["to"], "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48") {
				result = "0x75bcd15"
			}
		default:
			t.Fatalf("unexpected rpc method: %s", payload.Method)
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      1,
			"result":  result,
		})
	}))
	defer rpcServer.Close()

	wallettool.Configure(&config.Config{
		Chains: []*config.Chain{
			{
				Used:    true,
				Name:    "Ethereum",
				ChainID: "ethereum",
				Type:    "evm",
				Urls:    []string{rpcServer.URL},
			},
		},
	})
	defer wallettool.Configure(nil)

	handler := New(nil, nil)
	request := httptest.NewRequest(http.MethodPost, "/v1/asset/portfolio", strings.NewReader(`{"selected_chain_id":"1","addresses":{"eth":"1111111111111111111111111111111111111111"}}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	body := recorder.Body.String()
	for _, marker := range []string{`"chain_id":"ethereum"`, `"is_native":true`, `"wallet":{"chains":[`, `"updated_at":"`, `"price_usd":0`, `"value_usd":0`} {
		if !strings.Contains(body, marker) {
			t.Fatalf("expected %s in body: %s", marker, body)
		}
	}
}

func TestHandleAssetPortfolioReturnsSolanaBalances(t *testing.T) {
	originalService := marketdata.Default()
	marketdata.SetDefaultForTest(assetPortfolioFailingMarketService{})
	defer marketdata.SetDefaultForTest(originalService)

	rpcServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Method string `json:"method"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode rpc request: %v", err)
		}

		var result any
		switch payload.Method {
		case "getBalance":
			result = map[string]any{"value": 250000000}
		case "getTokenAccountsByOwner":
			result = map[string]any{
				"value": []any{
					map[string]any{
						"account": map[string]any{
							"data": map[string]any{
								"parsed": map[string]any{
									"info": map[string]any{
										"mint": "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v",
										"tokenAmount": map[string]any{
											"amount":         "4200000",
											"uiAmountString": "4.2",
											"decimals":       6,
										},
									},
								},
							},
						},
					},
				},
			}
		default:
			t.Fatalf("unexpected rpc method: %s", payload.Method)
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      1,
			"result":  result,
		})
	}))
	defer rpcServer.Close()

	wallettool.Configure(&config.Config{
		Chains: []*config.Chain{
			{
				Used:    true,
				Name:    "Solana",
				ChainID: "solana",
				Type:    "solana",
				Urls:    []string{rpcServer.URL},
			},
		},
	})
	defer wallettool.Configure(nil)

	handler := New(nil, nil)
	request := httptest.NewRequest(http.MethodPost, "/v1/asset/portfolio", strings.NewReader(`{"selected_chain_id":"sol","addresses":{"solana":"So11111111111111111111111111111111111111112"}}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	body := recorder.Body.String()
	for _, marker := range []string{`"chain_id":"solana"`, `"symbol":"SOL"`, `"symbol":"USDC"`, `"price_usd":0`, `"value_usd":0`} {
		if !strings.Contains(body, marker) {
			t.Fatalf("expected %s in body: %s", marker, body)
		}
	}
}
