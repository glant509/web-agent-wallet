package wallet

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
	"web3-service-agent/internal/tool"
)

type stubMarketService struct {
	prices map[string]float64
}

func (s stubMarketService) ProviderName() string { return "stub" }
func (s stubMarketService) SearchCoin(context.Context, string) (marketdata.CoinSearchItem, error) {
	return marketdata.CoinSearchItem{}, nil
}
func (s stubMarketService) FetchMarket(_ context.Context, assetID, _ string) (marketdata.MarketAsset, error) {
	return marketdata.MarketAsset{ID: assetID, CurrentPrice: s.prices[assetID]}, nil
}
func (s stubMarketService) FetchMarketChart(context.Context, string, string, int) ([][]float64, error) {
	return nil, nil
}
func (s stubMarketService) FetchContractMarketChart(context.Context, string, string, string, int) ([][]float64, error) {
	return nil, nil
}
func (s stubMarketService) FetchContractMarketChartRange(context.Context, string, string, string, int64, int64) ([][]float64, error) {
	return nil, nil
}
func (s stubMarketService) FetchMarketChartRange(context.Context, string, string, int64, int64) ([][]float64, error) {
	return nil, nil
}
func (s stubMarketService) FetchTopMarkets(context.Context, string, int) ([]marketdata.MarketAsset, error) {
	return nil, nil
}

type failingMarketService struct{}

func (failingMarketService) ProviderName() string { return "failing" }
func (failingMarketService) SearchCoin(context.Context, string) (marketdata.CoinSearchItem, error) {
	return marketdata.CoinSearchItem{}, nil
}
func (failingMarketService) FetchMarket(context.Context, string, string) (marketdata.MarketAsset, error) {
	return marketdata.MarketAsset{}, errors.New("price lookup should not be called")
}
func (failingMarketService) FetchMarketChart(context.Context, string, string, int) ([][]float64, error) {
	return nil, nil
}
func (failingMarketService) FetchContractMarketChart(context.Context, string, string, string, int) ([][]float64, error) {
	return nil, nil
}
func (failingMarketService) FetchContractMarketChartRange(context.Context, string, string, string, int64, int64) ([][]float64, error) {
	return nil, nil
}
func (failingMarketService) FetchMarketChartRange(context.Context, string, string, int64, int64) ([][]float64, error) {
	return nil, nil
}
func (failingMarketService) FetchTopMarkets(context.Context, string, int) ([]marketdata.MarketAsset, error) {
	return nil, nil
}

func TestRegisterLoadsDefinitionsFromFiles(t *testing.T) {
	registry := tool.NewRegistry()
	if err := Register(registry); err != nil {
		t.Fatalf("register wallet tools: %v", err)
	}

	definitions := registry.Definitions()
	if len(definitions) != 3 {
		t.Fatalf("expected 3 tool definitions, got %d", len(definitions))
	}
	if definitions[0].Name != walletAssetPortfolioToolName {
		t.Fatalf("unexpected first tool name: %q", definitions[0].Name)
	}
	if definitions[1].Name != walletChainTokenBalancesToolName {
		t.Fatalf("unexpected second tool name: %q", definitions[1].Name)
	}
	if definitions[2].Name != walletMultiChainBalancesToolName {
		t.Fatalf("unexpected third tool name: %q", definitions[2].Name)
	}
}

func TestFetchChainBalances(t *testing.T) {
	t.Setenv("TZ", "UTC")

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
			result = "0xde0b6b3a7640000" // 1 ETH
		case "eth_call":
			var call map[string]string
			if err := json.Unmarshal(payload.Params[0], &call); err != nil {
				t.Fatalf("decode eth_call params: %v", err)
			}
			switch strings.ToLower(call["to"]) {
			case strings.ToLower("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"):
				result = "0x75bcd15" // 123456789 -> 123.456789 USDC
			default:
				result = "0x0"
			}
		default:
			t.Fatalf("unexpected rpc method: %s", payload.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      1,
			"result":  result,
		})
	}))
	defer rpcServer.Close()

	originalService := marketdata.Default()
	marketdata.SetDefaultForTest(stubMarketService{
		prices: map[string]float64{
			"ethereum":        3200,
			"usd-coin":        1,
			"tether":          1,
			"wrapped-bitcoin": 60000,
			"dai":             1,
			"chainlink":       18,
		},
	})
	defer marketdata.SetDefaultForTest(originalService)
	Configure(&config.Config{
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
	defer Configure(nil)

	result, err := FetchChainBalances(context.Background(), ChainBalancesRequest{
		ChainID: "ethereum",
		Address: "0x1111111111111111111111111111111111111111",
	})
	if err != nil {
		t.Fatalf("fetch chain balances: %v", err)
	}
	if result.ChainID != "ethereum" {
		t.Fatalf("unexpected chain id: %s", result.ChainID)
	}
	if len(result.Items) != 6 {
		t.Fatalf("expected native plus 5 tokens, got %d", len(result.Items))
	}
	if result.TotalUSD <= 0 {
		t.Fatalf("expected positive total usd, got %f", result.TotalUSD)
	}
}

func TestFetchMultiChainBalances(t *testing.T) {
	originalService := marketdata.Default()
	marketdata.SetDefaultForTest(stubMarketService{
		prices: map[string]float64{"ethereum": 3000},
	})
	defer marketdata.SetDefaultForTest(originalService)

	rpcServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      1,
			"result":  "0xde0b6b3a7640000",
		})
	}))
	defer rpcServer.Close()
	Configure(&config.Config{
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
	defer Configure(nil)

	originalChain := supportedEVMChains["ethereum"]
	chain := originalChain
	chain.Tokens = nil
	supportedEVMChains["ethereum"] = chain
	defer func() {
		supportedEVMChains["ethereum"] = originalChain
	}()

	result, err := FetchMultiChainBalances(context.Background(), MultiChainBalancesRequest{
		Addresses: map[string]string{"ethereum": "0x1111111111111111111111111111111111111111"},
		ChainIDs:  []string{"ethereum"},
	})
	if err != nil {
		t.Fatalf("fetch multi-chain balances: %v", err)
	}
	if len(result.Chains) != 1 {
		t.Fatalf("expected 1 chain, got %d", len(result.Chains))
	}
	if result.TotalUSD <= 0 {
		t.Fatalf("expected positive total usd, got %f", result.TotalUSD)
	}
}

func TestFetchAssetPortfolioNormalizesChainIDs(t *testing.T) {
	originalService := marketdata.Default()
	marketdata.SetDefaultForTest(failingMarketService{})
	defer marketdata.SetDefaultForTest(originalService)

	rpcServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      1,
			"result":  "0xde0b6b3a7640000",
		})
	}))
	defer rpcServer.Close()

	Configure(&config.Config{
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
	defer Configure(nil)

	originalChain := supportedEVMChains["ethereum"]
	chain := originalChain
	chain.Tokens = nil
	supportedEVMChains["ethereum"] = chain
	defer func() {
		supportedEVMChains["ethereum"] = originalChain
	}()

	result, err := FetchAssetPortfolio(context.Background(), AssetPortfolioRequest{
		SelectedChainID: "1",
		Addresses: map[string]string{
			"ETH": "1111111111111111111111111111111111111111",
		},
	})
	if err != nil {
		t.Fatalf("fetch asset portfolio: %v", err)
	}
	if result.Selected.ChainID != "ethereum" {
		t.Fatalf("unexpected selected chain id: %s", result.Selected.ChainID)
	}
	if result.Selected.Address != "0x1111111111111111111111111111111111111111" {
		t.Fatalf("unexpected selected address: %s", result.Selected.Address)
	}
	if len(result.Wallet.Chains) != 1 {
		t.Fatalf("expected 1 wallet chain, got %d", len(result.Wallet.Chains))
	}
	if result.Selected.TotalUSD != 0 || result.Wallet.TotalUSD != 0 {
		t.Fatalf("expected no usd totals in asset portfolio, got selected=%f wallet=%f", result.Selected.TotalUSD, result.Wallet.TotalUSD)
	}
	if result.Selected.Items[0].PriceUSD != 0 || result.Selected.Items[0].ValueUSD != 0 {
		t.Fatalf("expected no usd pricing in asset portfolio item, got price=%f value=%f", result.Selected.Items[0].PriceUSD, result.Selected.Items[0].ValueUSD)
	}
}

func TestFetchAssetPortfolioOnlyQueriesSelectedChain(t *testing.T) {
	originalService := marketdata.Default()
	marketdata.SetDefaultForTest(failingMarketService{})
	defer marketdata.SetDefaultForTest(originalService)

	rpcCalls := 0
	rpcServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rpcCalls++
		_ = json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      1,
			"result":  "0x0",
		})
	}))
	defer rpcServer.Close()

	Configure(&config.Config{
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
	defer Configure(nil)

	originalEthereum := supportedEVMChains["ethereum"]
	ethereum := originalEthereum
	ethereum.Tokens = nil
	supportedEVMChains["ethereum"] = ethereum
	defer func() {
		supportedEVMChains["ethereum"] = originalEthereum
	}()

	result, err := FetchAssetPortfolio(context.Background(), AssetPortfolioRequest{
		SelectedChainID: "ethereum",
		Addresses: map[string]string{
			"ethereum": "0x1111111111111111111111111111111111111111",
			"base":     "0x2222222222222222222222222222222222222222",
		},
	})
	if err != nil {
		t.Fatalf("fetch asset portfolio: %v", err)
	}
	if result.Selected.ChainID != "ethereum" {
		t.Fatalf("unexpected selected chain: %s", result.Selected.ChainID)
	}
	if len(result.Wallet.Chains) != 1 {
		t.Fatalf("expected only selected chain in wallet, got %d", len(result.Wallet.Chains))
	}
	if rpcCalls != 1 {
		t.Fatalf("expected 1 rpc call for selected chain only, got %d", rpcCalls)
	}
}

func TestFetchAssetPortfolioSupportsSolana(t *testing.T) {
	originalService := marketdata.Default()
	marketdata.SetDefaultForTest(failingMarketService{})
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
			result = map[string]any{"value": 1500000000}
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
											"amount":         "1234500",
											"uiAmountString": "1.2345",
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

	Configure(&config.Config{
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
	defer Configure(nil)

	result, err := FetchAssetPortfolio(context.Background(), AssetPortfolioRequest{
		SelectedChainID: "sol",
		Addresses: map[string]string{
			"solana": "So11111111111111111111111111111111111111112",
		},
	})
	if err != nil {
		t.Fatalf("fetch solana asset portfolio: %v", err)
	}
	if result.Selected.ChainID != "solana" {
		t.Fatalf("unexpected selected chain: %s", result.Selected.ChainID)
	}
	if result.Selected.Address != "So11111111111111111111111111111111111111112" {
		t.Fatalf("unexpected address: %s", result.Selected.Address)
	}
	if len(result.Selected.Items) != 4 {
		t.Fatalf("expected native plus 3 tracked tokens, got %d", len(result.Selected.Items))
	}
	if result.Selected.Items[0].Symbol != "SOL" {
		t.Fatalf("expected native SOL first, got %s", result.Selected.Items[0].Symbol)
	}
	if result.Selected.Items[1].Symbol != "USDC" || result.Selected.Items[1].BalanceRaw != "1234500" {
		t.Fatalf("unexpected tracked token balance: %+v", result.Selected.Items[1])
	}
}

func TestConfigureOverridesRPCURL(t *testing.T) {
	originalService := marketdata.Default()
	marketdata.SetDefaultForTest(stubMarketService{
		prices: map[string]float64{"ethereum": 3000},
	})
	defer marketdata.SetDefaultForTest(originalService)

	rpcServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      1,
			"result":  "0xde0b6b3a7640000",
		})
	}))
	defer rpcServer.Close()

	Configure(&config.Config{
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
	defer Configure(nil)

	originalChain := supportedEVMChains["ethereum"]
	chain := originalChain
	chain.Tokens = nil
	supportedEVMChains["ethereum"] = chain
	defer func() {
		supportedEVMChains["ethereum"] = originalChain
	}()

	result, err := FetchChainBalances(context.Background(), ChainBalancesRequest{
		ChainID: "ethereum",
		Address: "0x1111111111111111111111111111111111111111",
	})
	if err != nil {
		t.Fatalf("fetch chain balances with configured rpc: %v", err)
	}
	if result.TotalUSD <= 0 {
		t.Fatalf("expected positive total usd, got %f", result.TotalUSD)
	}
}

func TestFetchChainBalancesRequiresConfiguredRPC(t *testing.T) {
	Configure(nil)

	_, err := FetchChainBalances(context.Background(), ChainBalancesRequest{
		ChainID: "ethereum",
		Address: "0x1111111111111111111111111111111111111111",
	})
	if err == nil || !strings.Contains(err.Error(), "no configured rpc url") {
		t.Fatalf("expected missing configured rpc url error, got %v", err)
	}
}
