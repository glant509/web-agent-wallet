package market

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"web3-service-agent/internal/marketdata"
	"web3-service-agent/internal/tool"
)

const (
	marketTokenKlinesToolName   = "market_token_klines"
	marketTokenOverviewToolName = "market_token_overview"
)

var (
	dexScreenerTokenURL = "https://api.dexscreener.com/latest/dex/tokens"
	marketHTTPClient    = &http.Client{Timeout: 15 * time.Second}
)

//go:embed market_token_klines.json
var marketTokenKlinesDefinition []byte

//go:embed market_token_overview.json
var marketTokenOverviewDefinition []byte

type marketOverviewRequest struct {
	Token        string `json:"token"`
	TokenAddress string `json:"token_address"`
	Chain        string `json:"chain"`
	VSCurrency   string `json:"vs_currency"`
}

type dexPairsResponse struct {
	Pairs []dexPair `json:"pairs"`
}

type dexPair struct {
	ChainID     string       `json:"chainId"`
	DexID       string       `json:"dexId"`
	URL         string       `json:"url"`
	PairAddress string       `json:"pairAddress"`
	BaseToken   dexTokenInfo `json:"baseToken"`
	PriceUSD    string       `json:"priceUsd"`
	PriceNative string       `json:"priceNative"`
	Liquidity   dexLiquidity `json:"liquidity"`
	FDV         float64      `json:"fdv"`
	MarketCap   float64      `json:"marketCap"`
}

type dexTokenInfo struct {
	Address string `json:"address"`
	Name    string `json:"name"`
	Symbol  string `json:"symbol"`
}

type dexLiquidity struct {
	USD float64 `json:"usd"`
}

func Register(registry *tool.Registry) error {
	registrations := []struct {
		name    string
		content []byte
		handler tool.Handler
	}{
		{name: "market_token_klines.json", content: marketTokenKlinesDefinition, handler: handleMarketTokenKlines},
		{name: "market_token_overview.json", content: marketTokenOverviewDefinition, handler: handleMarketTokenOverview},
	}

	for _, item := range registrations {
		definition, err := tool.ParseDefinition(item.name, item.content)
		if err != nil {
			return err
		}
		if err := registry.Register(definition, item.handler); err != nil {
			return err
		}
	}

	return nil
}

func handleMarketTokenOverview(ctx context.Context, call tool.Call) (tool.Result, error) {
	var request marketOverviewRequest
	if err := json.Unmarshal(call.Arguments, &request); err != nil {
		return tool.Result{}, fmt.Errorf("decode market overview arguments: %w", err)
	}

	vsCurrency := strings.ToLower(strings.TrimSpace(request.VSCurrency))
	if vsCurrency == "" {
		vsCurrency = "usd"
	}

	token := strings.TrimSpace(request.Token)
	tokenAddress := strings.TrimSpace(request.TokenAddress)
	chain := normalizeChainSlug(request.Chain)

	switch {
	case tokenAddress != "":
		content, err := fetchDexScreenerOverview(ctx, marketHTTPClient, chain, tokenAddress)
		if err != nil {
			return tool.Result{}, err
		}
		return tool.Result{Content: content}, nil
	case token != "":
		content, err := fetchMarketOverview(ctx, token, vsCurrency)
		if err != nil {
			return tool.Result{}, err
		}
		return tool.Result{Content: content}, nil
	default:
		return tool.Result{}, fmt.Errorf("either token or token_address is required")
	}
}

func fetchMarketOverview(ctx context.Context, token, vsCurrency string) (string, error) {
	match, err := marketdata.Default().SearchCoin(ctx, token)
	if err != nil {
		return "", err
	}

	market, err := marketdata.Default().FetchMarket(ctx, match.ID, vsCurrency)
	if err != nil {
		return "", err
	}

	return renderCoinGeckoOverview(market, vsCurrency, marketdata.Default().ProviderName()), nil
}

func renderCoinGeckoOverview(market marketdata.MarketAsset, vsCurrency, source string) string {
	lines := []string{
		fmt.Sprintf("Market overview for %s (%s):", market.Name, strings.ToUpper(market.Symbol)),
		fmt.Sprintf("- asset_id: %s", market.ID),
		fmt.Sprintf("- price_%s: %s", vsCurrency, formatPrice(market.CurrentPrice)),
		fmt.Sprintf("- market_cap_%s: %s", vsCurrency, formatAmount(market.MarketCap)),
		fmt.Sprintf("- 24h_volume_%s: %s", vsCurrency, formatAmount(market.TotalVolume)),
		fmt.Sprintf("- circulating_supply: %s", formatAmount(market.CirculatingSupply)),
	}

	if market.TotalSupply != nil && *market.TotalSupply > 0 {
		lines = append(lines, fmt.Sprintf("- total_supply: %s", formatAmount(*market.TotalSupply)))
	} else {
		lines = append(lines, "- total_supply: unavailable")
	}
	if market.FullyDilutedValuation > 0 {
		lines = append(lines, fmt.Sprintf("- fdv_%s: %s", vsCurrency, formatAmount(market.FullyDilutedValuation)))
	}
	if market.MarketCapRank > 0 {
		lines = append(lines, fmt.Sprintf("- market_cap_rank: %d", market.MarketCapRank))
	}
	lines = append(lines, fmt.Sprintf("- source: %s", source))

	return strings.Join(lines, "\n")
}

func fetchDexScreenerOverview(ctx context.Context, client *http.Client, chain, tokenAddress string) (string, error) {
	endpoint := strings.TrimRight(dexScreenerTokenURL, "/") + "/" + url.PathEscape(tokenAddress)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("build dexscreener request: %w", err)
	}
	request.Header.Set("Accept", "application/json")

	response, err := client.Do(request)
	if err != nil {
		return "", fmt.Errorf("request dexscreener market data: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 8*1024))
		return "", fmt.Errorf("dexscreener market data returned %s: %s", response.Status, strings.TrimSpace(string(body)))
	}

	var payload dexPairsResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("decode dexscreener response: %w", err)
	}

	pair, err := selectBestDexPair(payload.Pairs, chain)
	if err != nil {
		return "", err
	}

	return renderDexScreenerOverview(pair), nil
}

func selectBestDexPair(pairs []dexPair, chain string) (dexPair, error) {
	filtered := make([]dexPair, 0, len(pairs))
	for _, pair := range pairs {
		if chain == "" || normalizeChainSlug(pair.ChainID) == chain {
			filtered = append(filtered, pair)
		}
	}
	if len(filtered) == 0 {
		if chain != "" {
			return dexPair{}, fmt.Errorf("no market pair found for chain %q", chain)
		}
		return dexPair{}, fmt.Errorf("no market pair data returned")
	}

	best := filtered[0]
	for _, pair := range filtered[1:] {
		if pair.Liquidity.USD > best.Liquidity.USD {
			best = pair
		}
	}
	return best, nil
}

func renderDexScreenerOverview(pair dexPair) string {
	lines := []string{
		fmt.Sprintf("Market overview for %s (%s) on %s:", pair.BaseToken.Name, strings.ToUpper(pair.BaseToken.Symbol), pair.ChainID),
		fmt.Sprintf("- token_address: %s", pair.BaseToken.Address),
		fmt.Sprintf("- pair_address: %s", pair.PairAddress),
		fmt.Sprintf("- dex: %s", pair.DexID),
	}

	if pair.PriceUSD != "" {
		lines = append(lines, fmt.Sprintf("- price_usd: %s", pair.PriceUSD))
	}
	if pair.PriceNative != "" {
		lines = append(lines, fmt.Sprintf("- price_native: %s", pair.PriceNative))
	}
	if pair.Liquidity.USD > 0 {
		lines = append(lines, fmt.Sprintf("- liquidity_usd: %s", formatAmount(pair.Liquidity.USD)))
	}
	if pair.MarketCap > 0 {
		lines = append(lines, fmt.Sprintf("- market_cap_usd: %s", formatAmount(pair.MarketCap)))
	}
	if pair.FDV > 0 {
		lines = append(lines, fmt.Sprintf("- fdv_usd: %s", formatAmount(pair.FDV)))
	}
	lines = append(lines, "- total_supply: unavailable from DexScreener token endpoint")
	if pair.URL != "" {
		lines = append(lines, fmt.Sprintf("- pair_url: %s", pair.URL))
	}
	lines = append(lines, "- source: DexScreener")

	return strings.Join(lines, "\n")
}

func normalizeChainSlug(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.ReplaceAll(normalized, "_", "-")
	normalized = strings.ReplaceAll(normalized, " ", "-")

	switch normalized {
	case "":
		return ""
	case "eth", "ethereum-mainnet":
		return "ethereum"
	case "bsc", "bnb", "binance-smart-chain":
		return "bsc"
	case "arb", "arbitrum-one":
		return "arbitrum"
	case "matic", "polygon-pos", "polygon-mainnet":
		return "polygon"
	case "avax", "avalanche-c-chain", "avax-c":
		return "avalanche"
	default:
		return normalized
	}
}

func formatAmount(value float64) string {
	return strconv.FormatFloat(value, 'f', 2, 64)
}

func formatPrice(value float64) string {
	switch {
	case value >= 1000:
		return strconv.FormatFloat(value, 'f', 2, 64)
	case value >= 1:
		return strconv.FormatFloat(value, 'f', 4, 64)
	case value >= 0.01:
		return strconv.FormatFloat(value, 'f', 6, 64)
	default:
		return strconv.FormatFloat(value, 'f', 10, 64)
	}
}
