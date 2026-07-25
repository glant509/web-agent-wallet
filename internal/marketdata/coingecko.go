package marketdata

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"web3-service-agent/internal/config"
)

type coinGeckoService struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

type coinGeckoSearchResponse struct {
	Coins []CoinSearchItem `json:"coins"`
}

type coinGeckoChartResponse struct {
	Prices [][]float64 `json:"prices"`
}

func newCoinGeckoService(provider config.MarketProvider) Service {
	baseURL := strings.TrimRight(strings.TrimSpace(provider.BaseURL), "/")
	if baseURL == "" {
		baseURL = defaultCoinGeckoBaseURL
	}

	return &coinGeckoService{
		baseURL: baseURL,
		apiKey:  strings.TrimSpace(provider.APIKey),
		client:  &http.Client{Timeout: 15 * time.Second},
	}
}

func (s *coinGeckoService) ProviderName() string {
	return "CoinGecko"
}

func (s *coinGeckoService) SearchCoin(ctx context.Context, token string) (CoinSearchItem, error) {
	endpoint, err := url.Parse(s.baseURL + "/search")
	if err != nil {
		return CoinSearchItem{}, fmt.Errorf("parse coingecko search url: %w", err)
	}
	query := endpoint.Query()
	query.Set("query", token)
	endpoint.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return CoinSearchItem{}, fmt.Errorf("build coingecko search request: %w", err)
	}
	s.applyHeaders(request)

	response, err := s.client.Do(request)
	if err != nil {
		return CoinSearchItem{}, fmt.Errorf("request coingecko search: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 8*1024))
		return CoinSearchItem{}, fmt.Errorf("coingecko search returned %s: %s", response.Status, strings.TrimSpace(string(body)))
	}

	var payload coinGeckoSearchResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return CoinSearchItem{}, fmt.Errorf("decode coingecko search response: %w", err)
	}

	match, ok := bestCoinSearchMatch(token, payload.Coins)
	if !ok {
		return CoinSearchItem{}, fmt.Errorf("no coin match found for %q", token)
	}
	return match, nil
}

func (s *coinGeckoService) FetchMarket(ctx context.Context, assetID, vsCurrency string) (MarketAsset, error) {
	items, err := s.fetchMarkets(ctx, map[string]string{
		"ids":                     assetID,
		"vs_currency":             vsCurrency,
		"price_change_percentage": "24h",
	})
	if err != nil {
		return MarketAsset{}, err
	}
	if len(items) == 0 {
		return MarketAsset{}, fmt.Errorf("coingecko market data missing asset %q", assetID)
	}
	return items[0], nil
}

func (s *coinGeckoService) FetchTopMarkets(ctx context.Context, vsCurrency string, limit int) ([]MarketAsset, error) {
	if limit <= 0 {
		limit = 10
	}
	return s.fetchMarkets(ctx, map[string]string{
		"vs_currency":             vsCurrency,
		"order":                   "market_cap_desc",
		"per_page":                strconv.Itoa(limit),
		"page":                    "1",
		"sparkline":               "false",
		"price_change_percentage": "24h",
	})
}

func (s *coinGeckoService) fetchMarkets(ctx context.Context, params map[string]string) ([]MarketAsset, error) {
	endpoint, err := url.Parse(s.baseURL + "/coins/markets")
	if err != nil {
		return nil, fmt.Errorf("parse coingecko markets url: %w", err)
	}
	query := endpoint.Query()
	for key, value := range params {
		query.Set(key, value)
	}
	endpoint.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("build coingecko market request: %w", err)
	}
	s.applyHeaders(request)

	response, err := s.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("request coingecko market data: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 8*1024))
		return nil, fmt.Errorf("coingecko market data returned %s: %s", response.Status, strings.TrimSpace(string(body)))
	}

	var payload []MarketAsset
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode coingecko market response: %w", err)
	}
	return payload, nil
}

func (s *coinGeckoService) FetchMarketChart(ctx context.Context, assetID, vsCurrency string, days int) ([][]float64, error) {
	return s.fetchChart(ctx, s.baseURL+"/coins/"+url.PathEscape(assetID)+"/market_chart", map[string]string{
		"vs_currency": vsCurrency,
		"days":        strconv.Itoa(days),
	}, fmt.Sprintf("coingecko market chart for %q", assetID))
}

func (s *coinGeckoService) FetchMarketChartRange(ctx context.Context, assetID, vsCurrency string, fromTimestamp, toTimestamp int64) ([][]float64, error) {
	return s.fetchChart(ctx, s.baseURL+"/coins/"+url.PathEscape(assetID)+"/market_chart/range", map[string]string{
		"vs_currency": vsCurrency,
		"from":        strconv.FormatInt(fromTimestamp, 10),
		"to":          strconv.FormatInt(toTimestamp, 10),
	}, fmt.Sprintf("coingecko historical market chart for %q", assetID))
}

func (s *coinGeckoService) FetchContractMarketChart(ctx context.Context, chain, tokenAddress, vsCurrency string, days int) ([][]float64, error) {
	return s.fetchChart(ctx, s.baseURL+"/coins/"+url.PathEscape(chain)+"/contract/"+url.PathEscape(tokenAddress)+"/market_chart", map[string]string{
		"vs_currency": vsCurrency,
		"days":        strconv.Itoa(days),
	}, fmt.Sprintf("coingecko contract market chart for %q on %q", tokenAddress, chain))
}

func (s *coinGeckoService) FetchContractMarketChartRange(ctx context.Context, chain, tokenAddress, vsCurrency string, fromTimestamp, toTimestamp int64) ([][]float64, error) {
	return s.fetchChart(ctx, s.baseURL+"/coins/"+url.PathEscape(chain)+"/contract/"+url.PathEscape(tokenAddress)+"/market_chart/range", map[string]string{
		"vs_currency": vsCurrency,
		"from":        strconv.FormatInt(fromTimestamp, 10),
		"to":          strconv.FormatInt(toTimestamp, 10),
	}, fmt.Sprintf("coingecko contract historical market chart for %q on %q", tokenAddress, chain))
}

func (s *coinGeckoService) fetchChart(ctx context.Context, endpoint string, params map[string]string, label string) ([][]float64, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build %s request: %w", label, err)
	}
	s.applyHeaders(request)
	query := request.URL.Query()
	for key, value := range params {
		query.Set(key, value)
	}
	request.URL.RawQuery = query.Encode()

	response, err := s.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("request %s: %w", label, err)
	}
	defer response.Body.Close()

	if response.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 8*1024))
		return nil, fmt.Errorf("%s returned %s: %s", label, response.Status, strings.TrimSpace(string(body)))
	}

	var payload coinGeckoChartResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode %s response: %w", label, err)
	}
	return payload.Prices, nil
}

func (s *coinGeckoService) applyHeaders(request *http.Request) {
	request.Header.Set("Accept", "application/json")
	if s.apiKey != "" {
		request.Header.Set("x-cg-demo-api-key", s.apiKey)
		request.Header.Set("x-cg-pro-api-key", s.apiKey)
	}
}

func bestCoinSearchMatch(query string, coins []CoinSearchItem) (CoinSearchItem, bool) {
	if len(coins) == 0 {
		return CoinSearchItem{}, false
	}

	target := normalizeTokenQuery(query)
	candidates := make([]CoinSearchItem, len(coins))
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

func coinMatchScore(target string, coin CoinSearchItem) (int, int) {
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

func normalizeTokenQuery(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
