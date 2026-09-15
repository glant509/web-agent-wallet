package wallet

import (
	"bytes"
	"context"
	crand "crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/big"
	"net/http"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"web3-service-agent/internal/config"
	"web3-service-agent/internal/marketdata"
	"web3-service-agent/internal/tool"
)

const (
	walletChainTokenBalancesToolName = "wallet_chain_token_balances"
	walletMultiChainBalancesToolName = "wallet_multichain_balances"
	walletAssetPortfolioToolName     = "wallet_asset_portfolio"
	solanaTokenProgramID             = "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA"
)

var walletHTTPClient = &http.Client{Timeout: 15 * time.Second}
var chainRPCConfig = struct {
	mu   sync.RWMutex
	urls map[string][]string
}{}

type ChainBalancesRequest struct {
	Address        string   `json:"address"`
	ChainID        string   `json:"chain_id"`
	TokenAddresses []string `json:"token_addresses"`
}

type MultiChainBalancesRequest struct {
	Addresses map[string]string `json:"addresses"`
	ChainIDs  []string          `json:"chain_ids"`
}

type AssetPortfolioRequest struct {
	SelectedChainID string            `json:"selected_chain_id"`
	Addresses       map[string]string `json:"addresses"`
}

type TokenBalance struct {
	ChainID      string  `json:"chain_id"`
	TokenAddress string  `json:"token_address,omitempty"`
	Name         string  `json:"name"`
	Symbol       string  `json:"symbol"`
	Decimals     int     `json:"decimals"`
	IsNative     bool    `json:"is_native"`
	Balance      string  `json:"balance"`
	BalanceRaw   string  `json:"balance_raw"`
	PriceUSD     float64 `json:"price_usd"`
	ValueUSD     float64 `json:"value_usd"`
}

type ChainBalances struct {
	ChainID    string         `json:"chain_id"`
	Address    string         `json:"address"`
	Items      []TokenBalance `json:"items"`
	TotalUSD   float64        `json:"total_usd"`
	NativeName string         `json:"native_name"`
}

type MultiChainBalances struct {
	Chains   []ChainBalances `json:"chains"`
	TotalUSD float64         `json:"total_usd"`
}

type AssetPortfolio struct {
	Selected ChainBalances      `json:"selected"`
	Wallet   MultiChainBalances `json:"wallet"`
}

type trackedToken struct {
	Address  string
	Name     string
	Symbol   string
	Decimals int
	AssetID  string
}

type evmChain struct {
	ID            string
	NativeName    string
	NativeSymbol  string
	NativeAssetID string
	RPCURL        string
	Tokens        []trackedToken
}

type solanaChain struct {
	ID           string
	NativeName   string
	NativeSymbol string
	RPCURL       string
	Tokens       []trackedToken
}

var supportedEVMChains = map[string]evmChain{
	"ethereum": {
		ID:            "ethereum",
		NativeName:    "Ether",
		NativeSymbol:  "ETH",
		NativeAssetID: "ethereum",
		Tokens: []trackedToken{
			{Address: "0xdAC17F958D2ee523a2206206994597C13D831ec7", Name: "Tether USD", Symbol: "USDT", Decimals: 6, AssetID: "tether"},
			{Address: "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48", Name: "USD Coin", Symbol: "USDC", Decimals: 6, AssetID: "usd-coin"},
			{Address: "0x2260FAC5E5542a773Aa44fBCfeDf7C193bc2C599", Name: "Wrapped BTC", Symbol: "WBTC", Decimals: 8, AssetID: "wrapped-bitcoin"},
			{Address: "0x6B175474E89094C44Da98b954EedeAC495271d0F", Name: "Dai", Symbol: "DAI", Decimals: 18, AssetID: "dai"},
			{Address: "0x514910771AF9Ca656af840dff83E8264EcF986CA", Name: "Chainlink", Symbol: "LINK", Decimals: 18, AssetID: "chainlink"},
		},
	},
	"base": {
		ID:            "base",
		NativeName:    "Ether",
		NativeSymbol:  "ETH",
		NativeAssetID: "ethereum",
		Tokens: []trackedToken{
			{Address: "0x833589fCD6EDB6E08f4c7C32D4f71b54bdA02913", Name: "USD Coin", Symbol: "USDC", Decimals: 6, AssetID: "usd-coin"},
			{Address: "0x4200000000000000000000000000000000000006", Name: "Wrapped Ether", Symbol: "WETH", Decimals: 18, AssetID: "ethereum"},
			{Address: "0xcbb7c0000ab88b473b1f5afd9ef808440eed33bf", Name: "Coinbase Wrapped BTC", Symbol: "cbBTC", Decimals: 8, AssetID: "wrapped-bitcoin"},
			{Address: "0x50c5725949A6F0c72E6C4a641F24049A917DB0Cb", Name: "Dai", Symbol: "DAI", Decimals: 18, AssetID: "dai"},
			{Address: "0xfde4C96c8593536E31F229EA8f37b2ADa2699bb2", Name: "Tether USD", Symbol: "USDT", Decimals: 6, AssetID: "tether"},
		},
	},
	"arbitrum": {
		ID:            "arbitrum",
		NativeName:    "Ether",
		NativeSymbol:  "ETH",
		NativeAssetID: "ethereum",
		Tokens: []trackedToken{
			{Address: "0xFd086bC7CD5C481DCC9C85ebe478A1C0b69FCbb9", Name: "Tether USD", Symbol: "USDT", Decimals: 6, AssetID: "tether"},
			{Address: "0xaf88d065e77c8cC2239327C5EDb3A432268e5831", Name: "USD Coin", Symbol: "USDC", Decimals: 6, AssetID: "usd-coin"},
			{Address: "0x2f2a2543B76A4166549F7AaB2e75Bef0aefC5B0f", Name: "Wrapped BTC", Symbol: "WBTC", Decimals: 8, AssetID: "wrapped-bitcoin"},
			{Address: "0xda10009cbd5d07dd0cecc66161fc93d7c9000da1", Name: "Dai", Symbol: "DAI", Decimals: 18, AssetID: "dai"},
			{Address: "0xf97f4df75117a78c1A5a0DBb814Af92458539FB4", Name: "Chainlink", Symbol: "LINK", Decimals: 18, AssetID: "chainlink"},
		},
	},
	"optimism": {
		ID:            "optimism",
		NativeName:    "Ether",
		NativeSymbol:  "ETH",
		NativeAssetID: "ethereum",
		Tokens: []trackedToken{
			{Address: "0x94b008aA00579c1307B0EF2C499AD98a8ce58e58", Name: "Tether USD", Symbol: "USDT", Decimals: 6, AssetID: "tether"},
			{Address: "0x0b2c639c533813f4aa9d7837caf62653d097ff85", Name: "USD Coin", Symbol: "USDC", Decimals: 6, AssetID: "usd-coin"},
			{Address: "0x68f180fcCe6836688e9084f035309E29Bf0A2095", Name: "Wrapped BTC", Symbol: "WBTC", Decimals: 8, AssetID: "wrapped-bitcoin"},
			{Address: "0xDA10009cBd5D07dd0CeCc66161FC93D7c9000da1", Name: "Dai", Symbol: "DAI", Decimals: 18, AssetID: "dai"},
			{Address: "0x4200000000000000000000000000000000000042", Name: "Optimism", Symbol: "OP", Decimals: 18, AssetID: "optimism"},
		},
	},
	"bsc": {
		ID:            "bsc",
		NativeName:    "BNB",
		NativeSymbol:  "BNB",
		NativeAssetID: "binancecoin",
		Tokens: []trackedToken{
			{Address: "0x55d398326f99059fF775485246999027B3197955", Name: "Tether USD", Symbol: "USDT", Decimals: 18, AssetID: "tether"},
			{Address: "0x8ac76a51cc950d9822d68b83fe1ad97b32cd580d", Name: "USD Coin", Symbol: "USDC", Decimals: 18, AssetID: "usd-coin"},
			{Address: "0x7130d2A12B9BCbFAe4f2634d864A1Ee1Ce3Ead9c", Name: "BTCB", Symbol: "BTCB", Decimals: 18, AssetID: "bitcoin"},
			{Address: "0x1AF3F329e8BE154074D8769D1FFa4eE058B1DBc3", Name: "Dai", Symbol: "DAI", Decimals: 18, AssetID: "dai"},
			{Address: "0xbb4CdB9CBd36B01bD1cBaEBF2De08d9173bc095c", Name: "Wrapped BNB", Symbol: "WBNB", Decimals: 18, AssetID: "binancecoin"},
		},
	},
	"polygon": {
		ID:            "polygon",
		NativeName:    "Polygon",
		NativeSymbol:  "POL",
		NativeAssetID: "matic-network",
		Tokens: []trackedToken{
			{Address: "0xc2132D05D31c914a87C6611C10748AEb04B58e8F", Name: "Tether USD", Symbol: "USDT", Decimals: 6, AssetID: "tether"},
			{Address: "0x3c499c542cef5e3811e1192ce70d8cc03d5c3359", Name: "USD Coin", Symbol: "USDC", Decimals: 6, AssetID: "usd-coin"},
			{Address: "0x1BFD67037B42Cf73acF2047067bd4F2C47D9BfD6", Name: "Wrapped BTC", Symbol: "WBTC", Decimals: 8, AssetID: "wrapped-bitcoin"},
			{Address: "0x8f3Cf7ad23Cd3CaDbD9735AFf958023239c6A063", Name: "Dai", Symbol: "DAI", Decimals: 18, AssetID: "dai"},
			{Address: "0x7ceB23fD6bC0adD59E62ac25578270cFf1b9f619", Name: "Wrapped Ether", Symbol: "WETH", Decimals: 18, AssetID: "ethereum"},
		},
	},
	"avalanche": {
		ID:            "avalanche",
		NativeName:    "Avalanche",
		NativeSymbol:  "AVAX",
		NativeAssetID: "avalanche-2",
		Tokens: []trackedToken{
			{Address: "0x9702230A8Ea53601f5Cd2dc00fDBC13d4Df4A8c7", Name: "Tether USD", Symbol: "USDT", Decimals: 6, AssetID: "tether"},
			{Address: "0xB97EF9Ef8734C71904D8002F8b6Bc66Dd9c48a6E", Name: "USD Coin", Symbol: "USDC", Decimals: 6, AssetID: "usd-coin"},
			{Address: "0x50b7545627a5162F82A992c33b87aDc75187B218", Name: "Wrapped BTC", Symbol: "WBTC.e", Decimals: 8, AssetID: "wrapped-bitcoin"},
			{Address: "0xd586E7F844cEa2F87f50152665BCbc2C279D8d70", Name: "Dai", Symbol: "DAI.e", Decimals: 18, AssetID: "dai"},
			{Address: "0x49D5c2BdFfac6CE2BFdB6640F4F80f226bc10bAB", Name: "Wrapped Ether", Symbol: "WETH.e", Decimals: 18, AssetID: "ethereum"},
		},
	},
}

var supportedSolanaChains = map[string]solanaChain{
	"solana": {
		ID:           "solana",
		NativeName:   "Solana",
		NativeSymbol: "SOL",
		Tokens: []trackedToken{
			{Address: "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", Name: "USD Coin", Symbol: "USDC", Decimals: 6},
			{Address: "Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB", Name: "Tether USD", Symbol: "USDT", Decimals: 6},
			{Address: "So11111111111111111111111111111111111111112", Name: "Wrapped SOL", Symbol: "wSOL", Decimals: 9},
		},
	},
}

func Register(registry *tool.Registry) error {
	registrations := []struct {
		path    string
		handler tool.Handler
	}{
		{path: chainBalancesDefinitionPath(), handler: handleChainTokenBalances},
		{path: multiChainBalancesDefinitionPath(), handler: handleMultiChainBalances},
		{path: assetPortfolioDefinitionPath(), handler: handleAssetPortfolio},
	}

	for _, item := range registrations {
		definition, err := tool.LoadDefinition(item.path)
		if err != nil {
			return err
		}
		if err := registry.Register(definition, item.handler); err != nil {
			return err
		}
	}
	return nil
}

func Configure(cfg *config.Config) {
	next := make(map[string][]string)
	if cfg != nil {
		for _, chain := range cfg.Chains {
			if chain == nil || !chain.Used {
				continue
			}
			key := NormalizeBalanceChainID(firstNonEmpty(chain.ChainID, chain.Name))
			if key == "" {
				continue
			}
			urls := compactConfigURLs(chain.Urls)
			if len(urls) == 0 {
				continue
			}
			next[key] = urls
		}
	}
	chainRPCConfig.mu.Lock()
	chainRPCConfig.urls = next
	chainRPCConfig.mu.Unlock()
}

func handleChainTokenBalances(ctx context.Context, call tool.Call) (tool.Result, error) {
	var request ChainBalancesRequest
	if err := json.Unmarshal(call.Arguments, &request); err != nil {
		return tool.Result{}, fmt.Errorf("decode wallet chain balance arguments: %w", err)
	}
	result, err := FetchChainBalances(ctx, request)
	if err != nil {
		return tool.Result{}, err
	}
	return tool.Result{Content: renderChainBalances(result)}, nil
}

func handleMultiChainBalances(ctx context.Context, call tool.Call) (tool.Result, error) {
	var request MultiChainBalancesRequest
	if err := json.Unmarshal(call.Arguments, &request); err != nil {
		return tool.Result{}, fmt.Errorf("decode wallet multi-chain balance arguments: %w", err)
	}
	result, err := FetchMultiChainBalances(ctx, request)
	if err != nil {
		return tool.Result{}, err
	}
	return tool.Result{Content: renderMultiChainBalances(result)}, nil
}

func handleAssetPortfolio(ctx context.Context, call tool.Call) (tool.Result, error) {
	var request AssetPortfolioRequest
	if err := json.Unmarshal(call.Arguments, &request); err != nil {
		return tool.Result{}, fmt.Errorf("decode wallet asset portfolio arguments: %w", err)
	}
	result, err := FetchAssetPortfolio(ctx, request)
	if err != nil {
		return tool.Result{}, err
	}
	return tool.Result{Content: renderAssetPortfolio(result)}, nil
}

func FetchChainBalances(ctx context.Context, request ChainBalancesRequest) (ChainBalances, error) {
	return fetchChainBalances(ctx, request, true)
}

func fetchChainBalances(ctx context.Context, request ChainBalancesRequest, includePrices bool) (ChainBalances, error) {
	chain, err := resolveEVMChain(request.ChainID)
	if err != nil {
		return ChainBalances{}, err
	}
	chain.RPCURL, err = resolveChainRPCURL(chain.ID)
	if err != nil {
		return ChainBalances{}, err
	}
	address := normalizeEVMAddress(request.Address)
	if address == "" {
		return ChainBalances{}, fmt.Errorf("address is required")
	}

	priceCache := map[string]float64{}
	items := make([]TokenBalance, 0, 1+len(chain.Tokens))

	nativeBalance, err := fetchNativeBalance(ctx, chain, address, priceCache, includePrices)
	if err != nil {
		return ChainBalances{}, err
	}
	items = append(items, nativeBalance)

	if len(request.TokenAddresses) > 0 {
		for _, tokenAddress := range request.TokenAddresses {
			item, err := fetchCustomTokenBalance(ctx, chain, address, tokenAddress, priceCache, includePrices)
			if err != nil {
				return ChainBalances{}, err
			}
			items = append(items, item)
		}
	} else {
		for _, token := range chain.Tokens {
			item, err := fetchTrackedTokenBalance(ctx, chain, address, token, priceCache, includePrices)
			if err != nil {
				return ChainBalances{}, err
			}
			items = append(items, item)
		}
	}

	total := 0.0
	for _, item := range items {
		total += item.ValueUSD
	}
	return ChainBalances{
		ChainID:    chain.ID,
		Address:    address,
		Items:      items,
		TotalUSD:   total,
		NativeName: chain.NativeName,
	}, nil
}

func FetchMultiChainBalances(ctx context.Context, request MultiChainBalancesRequest) (MultiChainBalances, error) {
	portfolio, err := collectPortfolio(ctx, request.Addresses, request.ChainIDs, true)
	if err != nil {
		return MultiChainBalances{}, err
	}
	return portfolio.wallet, nil
}

func FetchAssetPortfolio(ctx context.Context, request AssetPortfolioRequest) (AssetPortfolio, error) {
	selectedChainID := NormalizeBalanceChainID(request.SelectedChainID)
	if selectedChainID == "" {
		return AssetPortfolio{}, fmt.Errorf("selected_chain_id is required")
	}
	selectedAddress := selectedAddressForChain(request.Addresses, selectedChainID)
	if selectedAddress == "" {
		return AssetPortfolio{}, fmt.Errorf("address for selected_chain_id is required")
	}

	selected, err := fetchSelectedChainBalances(ctx, selectedChainID, selectedAddress)
	if err != nil {
		return AssetPortfolio{}, err
	}

	return AssetPortfolio{
		Selected: selected,
		Wallet: MultiChainBalances{
			Chains:   []ChainBalances{selected},
			TotalUSD: selected.TotalUSD,
		},
	}, nil
}

func fetchSelectedChainBalances(ctx context.Context, chainID, address string) (ChainBalances, error) {
	if _, ok := supportedSolanaChains[NormalizeBalanceChainID(chainID)]; ok {
		return fetchSolanaChainBalances(ctx, chainID, address)
	}
	return fetchChainBalances(ctx, ChainBalancesRequest{
		Address: address,
		ChainID: chainID,
	}, false)
}

func renderChainBalances(result ChainBalances) string {
	lines := []string{
		fmt.Sprintf("Wallet balances on %s:", result.ChainID),
		fmt.Sprintf("- address: %s", result.Address),
		fmt.Sprintf("- total_usd: %.2f", result.TotalUSD),
	}
	for _, item := range result.Items {
		label := item.Symbol
		if item.IsNative {
			label += " (native)"
		}
		lines = append(lines, fmt.Sprintf("- %s: %s (%s USD %.2f)", label, item.Balance, item.BalanceRaw, item.ValueUSD))
	}
	return strings.Join(lines, "\n")
}

func renderMultiChainBalances(result MultiChainBalances) string {
	lines := []string{fmt.Sprintf("Wallet total balance: %.2f USD", result.TotalUSD)}
	for _, chain := range result.Chains {
		lines = append(lines, fmt.Sprintf("- %s: %.2f USD", chain.ChainID, chain.TotalUSD))
	}
	return strings.Join(lines, "\n")
}

func renderAssetPortfolio(result AssetPortfolio) string {
	lines := []string{
		fmt.Sprintf("Selected chain: %s", result.Selected.ChainID),
		fmt.Sprintf("- address: %s", result.Selected.Address),
		fmt.Sprintf("- selected_assets: %d", len(result.Selected.Items)),
		fmt.Sprintf("- wallet_chains: %d", len(result.Wallet.Chains)),
	}
	for _, item := range result.Selected.Items {
		label := item.Symbol
		if item.IsNative {
			label += " (native)"
		}
		lines = append(lines, fmt.Sprintf("- %s: %s (raw %s)", label, item.Balance, item.BalanceRaw))
	}
	return strings.Join(lines, "\n")
}

func fetchNativeBalance(ctx context.Context, chain evmChain, address string, priceCache map[string]float64, includePrices bool) (TokenBalance, error) {
	raw, err := evmRPC(ctx, chain.RPCURL, "eth_getBalance", []any{address, "latest"})
	if err != nil {
		return TokenBalance{}, fmt.Errorf("fetch native balance on %s: %w", chain.ID, err)
	}
	value, err := hexToBigInt(raw)
	if err != nil {
		return TokenBalance{}, err
	}
	price := 0.0
	valueUSD := 0.0
	if includePrices {
		price, err = priceForAsset(ctx, chain.NativeAssetID, priceCache)
		if err != nil {
			return TokenBalance{}, err
		}
		valueUSD = floatFromUnits(value, 18) * price
	}
	return TokenBalance{
		ChainID:    chain.ID,
		Name:       chain.NativeName,
		Symbol:     chain.NativeSymbol,
		Decimals:   18,
		IsNative:   true,
		Balance:    formatUnits(value, 18),
		BalanceRaw: value.String(),
		PriceUSD:   price,
		ValueUSD:   valueUSD,
	}, nil
}

func fetchTrackedTokenBalance(ctx context.Context, chain evmChain, address string, token trackedToken, priceCache map[string]float64, includePrices bool) (TokenBalance, error) {
	raw, err := fetchERC20Balance(ctx, chain.RPCURL, token.Address, address)
	if err != nil {
		return TokenBalance{}, fmt.Errorf("fetch %s balance on %s: %w", token.Symbol, chain.ID, err)
	}
	value, err := hexToBigInt(raw)
	if err != nil {
		return TokenBalance{}, err
	}
	price := 0.0
	valueUSD := 0.0
	if includePrices {
		price, err = priceForAsset(ctx, token.AssetID, priceCache)
		if err != nil {
			return TokenBalance{}, err
		}
		valueUSD = floatFromUnits(value, token.Decimals) * price
	}
	return TokenBalance{
		ChainID:      chain.ID,
		TokenAddress: token.Address,
		Name:         token.Name,
		Symbol:       token.Symbol,
		Decimals:     token.Decimals,
		Balance:      formatUnits(value, token.Decimals),
		BalanceRaw:   value.String(),
		PriceUSD:     price,
		ValueUSD:     valueUSD,
	}, nil
}

func fetchCustomTokenBalance(ctx context.Context, chain evmChain, address, tokenAddress string, priceCache map[string]float64, includePrices bool) (TokenBalance, error) {
	normalizedAddress := normalizeEVMAddress(tokenAddress)
	if normalizedAddress == "" {
		return TokenBalance{}, fmt.Errorf("token_address is required")
	}
	for _, token := range chain.Tokens {
		if strings.EqualFold(token.Address, normalizedAddress) {
			return fetchTrackedTokenBalance(ctx, chain, address, token, priceCache, includePrices)
		}
	}

	raw, err := fetchERC20Balance(ctx, chain.RPCURL, normalizedAddress, address)
	if err != nil {
		return TokenBalance{}, err
	}
	value, err := hexToBigInt(raw)
	if err != nil {
		return TokenBalance{}, err
	}
	decimals, err := fetchERC20Decimals(ctx, chain.RPCURL, normalizedAddress)
	if err != nil {
		return TokenBalance{}, err
	}
	symbol, _ := fetchERC20Symbol(ctx, chain.RPCURL, normalizedAddress)
	if symbol == "" {
		symbol = "TOKEN"
	}
	return TokenBalance{
		ChainID:      chain.ID,
		TokenAddress: normalizedAddress,
		Name:         symbol,
		Symbol:       symbol,
		Decimals:     decimals,
		Balance:      formatUnits(value, decimals),
		BalanceRaw:   value.String(),
	}, nil
}

func fetchERC20Balance(ctx context.Context, rpcURL, tokenAddress, walletAddress string) (string, error) {
	data := "0x70a08231" + leftPadHex(strings.TrimPrefix(strings.ToLower(walletAddress), "0x"), 64)
	return evmCall(ctx, rpcURL, tokenAddress, data)
}

func fetchERC20Decimals(ctx context.Context, rpcURL, tokenAddress string) (int, error) {
	raw, err := evmCall(ctx, rpcURL, tokenAddress, "0x313ce567")
	if err != nil {
		return 0, err
	}
	value, err := hexToBigInt(raw)
	if err != nil {
		return 0, err
	}
	return int(value.Int64()), nil
}

func fetchERC20Symbol(ctx context.Context, rpcURL, tokenAddress string) (string, error) {
	raw, err := evmCall(ctx, rpcURL, tokenAddress, "0x95d89b41")
	if err != nil {
		return "", err
	}
	return decodeABIString(raw), nil
}

func evmCall(ctx context.Context, rpcURL, to, data string) (string, error) {
	return evmRPC(ctx, rpcURL, "eth_call", []any{
		map[string]string{"to": to, "data": data},
		"latest",
	})
}

func evmRPC(ctx context.Context, rpcURL, method string, params []any) (string, error) {
	payload, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  method,
		"params":  params,
	})
	if err != nil {
		return "", fmt.Errorf("marshal rpc payload: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, rpcURL, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("build rpc request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := walletHTTPClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("request rpc: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 8*1024))
		return "", fmt.Errorf("rpc returned %s: %s", response.Status, strings.TrimSpace(string(body)))
	}

	var rpcResponse struct {
		Result string `json:"result"`
		Error  *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(response.Body).Decode(&rpcResponse); err != nil {
		return "", fmt.Errorf("decode rpc response: %w", err)
	}
	if rpcResponse.Error != nil {
		return "", fmt.Errorf("rpc error: %s", rpcResponse.Error.Message)
	}
	return rpcResponse.Result, nil
}

func priceForAsset(ctx context.Context, assetID string, priceCache map[string]float64) (float64, error) {
	if assetID == "" {
		return 0, nil
	}
	if cached, ok := priceCache[assetID]; ok {
		return cached, nil
	}
	market, err := marketdata.Default().FetchMarket(ctx, assetID, "usd")
	if err != nil {
		return 0, fmt.Errorf("fetch market price for %s: %w", assetID, err)
	}
	priceCache[assetID] = market.CurrentPrice
	return market.CurrentPrice, nil
}

func decodeABIString(value string) string {
	trimmed := strings.TrimPrefix(value, "0x")
	if len(trimmed) == 64 {
		return strings.TrimRight(string(bytes.TrimRight(mustHexDecode(trimmed), "\x00")), "\x00")
	}
	if len(trimmed) >= 192 {
		lengthValue, ok := new(big.Int).SetString(trimmed[64:128], 16)
		if !ok {
			return ""
		}
		length := int(lengthValue.Int64()) * 2
		if length <= 0 || len(trimmed) < 128+length {
			return ""
		}
		return string(mustHexDecode(trimmed[128 : 128+length]))
	}
	return ""
}

func hexToBigInt(value string) (*big.Int, error) {
	trimmed := strings.TrimPrefix(strings.TrimSpace(value), "0x")
	if trimmed == "" {
		return big.NewInt(0), nil
	}
	number, ok := new(big.Int).SetString(trimmed, 16)
	if !ok {
		return nil, fmt.Errorf("invalid hex quantity %q", value)
	}
	return number, nil
}

func floatFromUnits(value *big.Int, decimals int) float64 {
	if value == nil {
		return 0
	}
	numerator := new(big.Float).SetInt(value)
	denominator := new(big.Float).SetFloat64(math.Pow10(decimals))
	result, _ := new(big.Float).Quo(numerator, denominator).Float64()
	return result
}

func formatUnits(value *big.Int, decimals int) string {
	number := floatFromUnits(value, decimals)
	switch {
	case number >= 1000:
		return strconv.FormatFloat(number, 'f', 2, 64)
	case number >= 1:
		return strconv.FormatFloat(number, 'f', 4, 64)
	default:
		return strconv.FormatFloat(number, 'f', 8, 64)
	}
}

func mustHexDecode(value string) []byte {
	decoded := make([]byte, len(value)/2)
	for index := 0; index < len(value); index += 2 {
		chunk := value[index : index+2]
		parsed, _ := strconv.ParseUint(chunk, 16, 8)
		decoded[index/2] = byte(parsed)
	}
	return decoded
}

func leftPadHex(value string, length int) string {
	if len(value) >= length {
		return value
	}
	return strings.Repeat("0", length-len(value)) + value
}

func normalizeEVMAddress(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if !strings.HasPrefix(trimmed, "0x") {
		trimmed = "0x" + trimmed
	}
	return trimmed
}

func resolveEVMChain(value string) (evmChain, error) {
	normalized := NormalizeBalanceChainID(value)
	chain, ok := supportedEVMChains[normalized]
	if !ok {
		return evmChain{}, fmt.Errorf("unsupported balance chain %q", value)
	}
	return chain, nil
}

func resolveSolanaChain(value string) (solanaChain, error) {
	normalized := NormalizeBalanceChainID(value)
	chain, ok := supportedSolanaChains[normalized]
	if !ok {
		return solanaChain{}, fmt.Errorf("unsupported balance chain %q", value)
	}
	return chain, nil
}

func supportedBalanceChainIDs() []string {
	ids := make([]string, 0, len(supportedEVMChains))
	for id := range supportedEVMChains {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	return ids
}

type collectedPortfolio struct {
	byChainID map[string]ChainBalances
	wallet    MultiChainBalances
}

func collectPortfolio(ctx context.Context, addresses map[string]string, chainIDs []string, includePrices bool) (collectedPortfolio, error) {
	normalizedAddresses := normalizeAddressMap(addresses)
	if len(normalizedAddresses) == 0 {
		return collectedPortfolio{}, fmt.Errorf("addresses are required")
	}

	targetChainIDs := normalizedChainIDs(chainIDs)
	if len(targetChainIDs) == 0 {
		targetChainIDs = supportedBalanceChainIDs()
	}

	chains := make([]ChainBalances, 0, len(targetChainIDs))
	byChainID := make(map[string]ChainBalances, len(targetChainIDs))
	total := 0.0
	for _, chainID := range targetChainIDs {
		address := normalizedAddresses[chainID]
		if address == "" {
			continue
		}
		chainBalance, err := fetchChainBalances(ctx, ChainBalancesRequest{
			Address: address,
			ChainID: chainID,
		}, includePrices)
		if err != nil {
			return collectedPortfolio{}, err
		}
		chains = append(chains, chainBalance)
		byChainID[chainID] = chainBalance
		total += chainBalance.TotalUSD
	}

	return collectedPortfolio{
		byChainID: byChainID,
		wallet: MultiChainBalances{
			Chains:   chains,
			TotalUSD: total,
		},
	}, nil
}

func NormalizeBalanceChainID(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "1", "eth", "ethereum-mainnet":
		return "ethereum"
	case "8453":
		return "base"
	case "42161", "arb":
		return "arbitrum"
	case "10", "op":
		return "optimism"
	case "56", "bnb", "binance-smart-chain":
		return "bsc"
	case "137", "matic", "polygon-pos":
		return "polygon"
	case "43114", "avax", "avalanche-c":
		return "avalanche"
	case "sol", "solana-mainnet":
		return "solana"
	default:
		return normalized
	}
}

func normalizeAddressMap(addresses map[string]string) map[string]string {
	normalized := make(map[string]string, len(addresses))
	for chainID, address := range addresses {
		key := NormalizeBalanceChainID(chainID)
		value := normalizeBalanceAddress(key, address)
		if key == "" || value == "" {
			continue
		}
		normalized[key] = value
	}
	return normalized
}

func normalizeBalanceAddress(chainID, address string) string {
	switch NormalizeBalanceChainID(chainID) {
	case "solana":
		return strings.TrimSpace(address)
	default:
		return normalizeEVMAddress(address)
	}
}

func selectedAddressForChain(addresses map[string]string, chainID string) string {
	return normalizeAddressMap(addresses)[NormalizeBalanceChainID(chainID)]
}

func fetchSolanaChainBalances(ctx context.Context, chainID, address string) (ChainBalances, error) {
	chain, err := resolveSolanaChain(chainID)
	if err != nil {
		return ChainBalances{}, err
	}
	chain.RPCURL, err = resolveChainRPCURL(chain.ID)
	if err != nil {
		return ChainBalances{}, err
	}
	address = normalizeBalanceAddress(chain.ID, address)
	if address == "" {
		return ChainBalances{}, fmt.Errorf("address is required")
	}

	nativeItem, err := fetchSolanaNativeBalance(ctx, chain, address)
	if err != nil {
		return ChainBalances{}, err
	}
	tokenBalances, err := fetchSolanaTrackedTokenBalances(ctx, chain, address)
	if err != nil {
		return ChainBalances{}, err
	}

	items := make([]TokenBalance, 0, 1+len(tokenBalances))
	items = append(items, nativeItem)
	items = append(items, tokenBalances...)

	return ChainBalances{
		ChainID:    chain.ID,
		Address:    address,
		Items:      items,
		TotalUSD:   0,
		NativeName: chain.NativeName,
	}, nil
}

func fetchSolanaNativeBalance(ctx context.Context, chain solanaChain, address string) (TokenBalance, error) {
	var result struct {
		Value uint64 `json:"value"`
	}
	if err := solanaRPC(ctx, chain.RPCURL, "getBalance", []any{address}, &result); err != nil {
		return TokenBalance{}, fmt.Errorf("fetch native balance on %s: %w", chain.ID, err)
	}

	value := new(big.Int).SetUint64(result.Value)
	return TokenBalance{
		ChainID:    chain.ID,
		Name:       chain.NativeName,
		Symbol:     chain.NativeSymbol,
		Decimals:   9,
		IsNative:   true,
		Balance:    formatUnits(value, 9),
		BalanceRaw: value.String(),
	}, nil
}

func fetchSolanaTrackedTokenBalances(ctx context.Context, chain solanaChain, address string) ([]TokenBalance, error) {
	var result struct {
		Value []struct {
			Account struct {
				Data struct {
					Parsed struct {
						Info struct {
							Mint        string `json:"mint"`
							TokenAmount struct {
								Amount         string `json:"amount"`
								UIAmountString string `json:"uiAmountString"`
								Decimals       int    `json:"decimals"`
							} `json:"tokenAmount"`
						} `json:"info"`
					} `json:"parsed"`
				} `json:"data"`
			} `json:"account"`
		} `json:"value"`
	}
	params := []any{
		address,
		map[string]string{"programId": solanaTokenProgramID},
		map[string]string{"encoding": "jsonParsed"},
	}
	if err := solanaRPC(ctx, chain.RPCURL, "getTokenAccountsByOwner", params, &result); err != nil {
		return nil, fmt.Errorf("fetch token accounts on %s: %w", chain.ID, err)
	}

	amountsByMint := make(map[string]*big.Int, len(result.Value))
	decimalsByMint := make(map[string]int, len(result.Value))
	for _, item := range result.Value {
		info := item.Account.Data.Parsed.Info
		if strings.TrimSpace(info.Mint) == "" {
			continue
		}
		amount, ok := new(big.Int).SetString(strings.TrimSpace(info.TokenAmount.Amount), 10)
		if !ok || amount.Sign() == 0 {
			continue
		}
		mint := strings.TrimSpace(info.Mint)
		if current, exists := amountsByMint[mint]; exists {
			current.Add(current, amount)
		} else {
			amountsByMint[mint] = amount
		}
		decimalsByMint[mint] = info.TokenAmount.Decimals
	}

	items := make([]TokenBalance, 0, len(chain.Tokens))
	for _, token := range chain.Tokens {
		value := amountsByMint[token.Address]
		if value == nil {
			value = big.NewInt(0)
		}
		decimals := token.Decimals
		if current, ok := decimalsByMint[token.Address]; ok {
			decimals = current
		}
		items = append(items, TokenBalance{
			ChainID:      chain.ID,
			TokenAddress: token.Address,
			Name:         token.Name,
			Symbol:       token.Symbol,
			Decimals:     decimals,
			Balance:      formatUnits(value, decimals),
			BalanceRaw:   value.String(),
		})
		delete(amountsByMint, token.Address)
		delete(decimalsByMint, token.Address)
	}

	unknownMints := make([]string, 0, len(amountsByMint))
	for mint := range amountsByMint {
		unknownMints = append(unknownMints, mint)
	}
	slices.Sort(unknownMints)
	for _, mint := range unknownMints {
		value := amountsByMint[mint]
		decimals := decimalsByMint[mint]
		symbol := shortenAddress(mint)
		items = append(items, TokenBalance{
			ChainID:      chain.ID,
			TokenAddress: mint,
			Name:         symbol,
			Symbol:       symbol,
			Decimals:     decimals,
			Balance:      formatUnits(value, decimals),
			BalanceRaw:   value.String(),
		})
	}

	return items, nil
}

func solanaRPC(ctx context.Context, rpcURL, method string, params []any, target any) error {
	payload, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  method,
		"params":  params,
	})
	if err != nil {
		return fmt.Errorf("marshal rpc payload: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, rpcURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("build rpc request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := walletHTTPClient.Do(request)
	if err != nil {
		return fmt.Errorf("request rpc: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 8*1024))
		return fmt.Errorf("rpc returned %s: %s", response.Status, strings.TrimSpace(string(body)))
	}

	var rpcResponse struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(response.Body).Decode(&rpcResponse); err != nil {
		return fmt.Errorf("decode rpc response: %w", err)
	}
	if rpcResponse.Error != nil {
		return fmt.Errorf("rpc error: %s", rpcResponse.Error.Message)
	}
	if target == nil {
		return nil
	}
	if err := json.Unmarshal(rpcResponse.Result, target); err != nil {
		return fmt.Errorf("decode rpc result: %w", err)
	}
	return nil
}

func shortenAddress(value string) string {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) <= 8 {
		return trimmed
	}
	return trimmed[:4] + "..." + trimmed[len(trimmed)-4:]
}

func normalizedChainIDs(chainIDs []string) []string {
	seen := make(map[string]struct{}, len(chainIDs))
	normalized := make([]string, 0, len(chainIDs))
	for _, chainID := range chainIDs {
		key := NormalizeBalanceChainID(chainID)
		if key == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, key)
	}
	return normalized
}

func resolveChainRPCURL(chainID string) (string, error) {
	chainRPCConfig.mu.RLock()
	configured := append([]string(nil), chainRPCConfig.urls[NormalizeBalanceChainID(chainID)]...)
	chainRPCConfig.mu.RUnlock()
	if len(configured) == 0 {
		return "", fmt.Errorf("no configured rpc url for chain %q", chainID)
	}
	if len(configured) == 1 {
		return configured[0], nil
	}
	index, err := crand.Int(crand.Reader, big.NewInt(int64(len(configured))))
	if err != nil {
		return "", fmt.Errorf("choose rpc url for chain %q: %w", chainID, err)
	}
	return configured[index.Int64()], nil
}

func compactConfigURLs(urls []string) []string {
	compacted := make([]string, 0, len(urls))
	for _, url := range urls {
		if trimmed := strings.TrimSpace(url); trimmed != "" {
			compacted = append(compacted, trimmed)
		}
	}
	return compacted
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func chainBalancesDefinitionPath() string {
	_, currentFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(currentFile), "wallet_chain_token_balances.json")
}

func multiChainBalancesDefinitionPath() string {
	_, currentFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(currentFile), "wallet_multichain_balances.json")
}

func assetPortfolioDefinitionPath() string {
	_, currentFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(currentFile), "wallet_asset_portfolio.json")
}
