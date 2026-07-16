package market

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"web3-service-agent/internal/tool"
)

const marketTokenOverviewToolName = "market_token_overview"

var (
	coingeckoSearchURL  = "https://api.coingecko.com/api/v3/search"
	coingeckoMarketsURL = "https://api.coingecko.com/api/v3/coins/markets"
	dexScreenerTokenURL = "https://api.dexscreener.com/latest/dex/tokens"
	marketHTTPClient    = &http.Client{Timeout: 15 * time.Second}
)

type marketOverviewRequest struct {
	Token        string `json:"token"`
	TokenAddress string `json:"token_address"`
	Chain        string `json:"chain"`
	VSCurrency   string `json:"vs_currency"`
}

type coinSearchResponse struct {
	Coins []coinSearchItem `json:"coins"`
}

type coinSearchItem struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Symbol        string `json:"symbol"`
	APISymbol     string `json:"api_symbol"`
	MarketCapRank int    `json:"market_cap_rank"`
}

type coinMarket struct {
	ID                    string   `json:"id"`
	Symbol                string   `json:"symbol"`
	Name                  string   `json:"name"`
	CurrentPrice          float64  `json:"current_price"`
	MarketCap             float64  `json:"market_cap"`
	TotalVolume           float64  `json:"total_volume"`
	CirculatingSupply     float64  `json:"circulating_supply"`
	TotalSupply           *float64 `json:"total_supply"`
	FullyDilutedValuation float64  `json:"fully_diluted_valuation"`
	MarketCapRank         int      `json:"market_cap_rank"`
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
	definition, err := tool.LoadDefinition(definitionPath())
	if err != nil {
		return err
	}
	return registry.Register(definition, handleMarketTokenOverview)
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
		content, err := fetchCoinGeckoOverview(ctx, marketHTTPClient, token, vsCurrency)
		if err != nil {
			return tool.Result{}, err
		}
		return tool.Result{Content: content}, nil
	default:
		return tool.Result{}, fmt.Errorf("either token or token_address is required")
	}
}

func fetchCoinGeckoOverview(ctx context.Context, client *http.Client, token, vsCurrency string) (string, error) {
	match, err := searchCoin(ctx, client, token)
	if err != nil {
		return "", err
	}

	market, err := fetchCoinMarket(ctx, client, match.ID, vsCurrency)
	if err != nil {
		return "", err
	}

	return renderCoinGeckoOverview(market, vsCurrency), nil
}

func searchCoin(ctx context.Context, client *http.Client, token string) (coinSearchItem, error) {
	endpoint, err := url.Parse(coingeckoSearchURL)
	if err != nil {
		return coinSearchItem{}, fmt.Errorf("parse coingecko search url: %w", err)
	}

	query := endpoint.Query()
	query.Set("query", token)
	endpoint.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return coinSearchItem{}, fmt.Errorf("build coingecko search request: %w", err)
	}
	request.Header.Set("Accept", "application/json")

	response, err := client.Do(request)
	if err != nil {
		return coinSearchItem{}, fmt.Errorf("request coingecko search: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 8*1024))
		return coinSearchItem{}, fmt.Errorf("coingecko search returned %s: %s", response.Status, strings.TrimSpace(string(body)))
	}

	var payload coinSearchResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return coinSearchItem{}, fmt.Errorf("decode coingecko search response: %w", err)
	}

	match, ok := bestCoinSearchMatch(token, payload.Coins)
	if !ok {
		return coinSearchItem{}, fmt.Errorf("no coin match found for %q", token)
	}
	return match, nil
}

func bestCoinSearchMatch(query string, coins []coinSearchItem) (coinSearchItem, bool) {
	if len(coins) == 0 {
		return coinSearchItem{}, false
	}

	target := normalizeTokenQuery(query)
	candidates := make([]coinSearchItem, len(coins))
	copy(candidates, coins)

	sort.SliceStable(candidates, func(i, j int) bool {
		leftPriority, leftRank := coinMatchScore(target, candidates[i])
		rightPriority, rightRank := coinMatchScore(target, candidates[j])
		if leftPriority != rightPriority {
			return leftPriority < rightPriority
		}
		if leftRank != rightRank {
			return leftRank < rightRank
		}
		return strings.ToLower(candidates[i].ID) < strings.ToLower(candidates[j].ID)
	})

	return candidates[0], true
}

func coinMatchScore(target string, coin coinSearchItem) (int, int) {
	switch {
	case normalizeTokenQuery(coin.ID) == target:
		return 0, marketRankValue(coin.MarketCapRank)
	case normalizeTokenQuery(coin.Symbol) == target || normalizeTokenQuery(coin.APISymbol) == target:
		return 1, marketRankValue(coin.MarketCapRank)
	case normalizeTokenQuery(coin.Name) == target:
		return 2, marketRankValue(coin.MarketCapRank)
	case strings.Contains(normalizeTokenQuery(coin.ID), target):
		return 3, marketRankValue(coin.MarketCapRank)
	case strings.Contains(normalizeTokenQuery(coin.Name), target):
		return 4, marketRankValue(coin.MarketCapRank)
	default:
		return 5, marketRankValue(coin.MarketCapRank)
	}
}

func marketRankValue(rank int) int {
	if rank <= 0 {
		return int(^uint(0) >> 1)
	}
	return rank
}

func fetchCoinMarket(ctx context.Context, client *http.Client, coinID, vsCurrency string) (coinMarket, error) {
	endpoint, err := url.Parse(coingeckoMarketsURL)
	if err != nil {
		return coinMarket{}, fmt.Errorf("parse coingecko markets url: %w", err)
	}

	query := endpoint.Query()
	query.Set("ids", coinID)
	query.Set("vs_currency", vsCurrency)
	endpoint.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return coinMarket{}, fmt.Errorf("build coingecko market request: %w", err)
	}
	request.Header.Set("Accept", "application/json")

	response, err := client.Do(request)
	if err != nil {
		return coinMarket{}, fmt.Errorf("request coingecko market data: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 8*1024))
		return coinMarket{}, fmt.Errorf("coingecko market data returned %s: %s", response.Status, strings.TrimSpace(string(body)))
	}

	var payload []coinMarket
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return coinMarket{}, fmt.Errorf("decode coingecko market response: %w", err)
	}
	if len(payload) == 0 {
		return coinMarket{}, fmt.Errorf("coingecko market data missing asset %q", coinID)
	}
	return payload[0], nil
}

func renderCoinGeckoOverview(market coinMarket, vsCurrency string) string {
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
	lines = append(lines, "- source: CoinGecko")

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

func normalizeTokenQuery(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
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

func definitionPath() string {
	_, currentFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(currentFile), "market_token_overview.json")
}
