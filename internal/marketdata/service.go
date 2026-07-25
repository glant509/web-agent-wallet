package marketdata

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"web3-service-agent/internal/config"
)

const defaultCoinGeckoBaseURL = "https://api.coingecko.com/api/v3"

type CoinSearchItem struct {
	ID            string
	Name          string
	Symbol        string
	APISymbol     string
	MarketCapRank int
}

type MarketAsset struct {
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
	PriceChange24h        float64  `json:"price_change_percentage_24h"`
	High24h               float64  `json:"high_24h"`
	Low24h                float64  `json:"low_24h"`
}

type Service interface {
	ProviderName() string
	SearchCoin(ctx context.Context, token string) (CoinSearchItem, error)
	FetchMarket(ctx context.Context, assetID, vsCurrency string) (MarketAsset, error)
	FetchMarketChart(ctx context.Context, assetID, vsCurrency string, days int) ([][]float64, error)
	FetchContractMarketChart(ctx context.Context, chain, tokenAddress, vsCurrency string, days int) ([][]float64, error)
	FetchContractMarketChartRange(ctx context.Context, chain, tokenAddress, vsCurrency string, fromTimestamp, toTimestamp int64) ([][]float64, error)
	FetchMarketChartRange(ctx context.Context, assetID, vsCurrency string, fromTimestamp, toTimestamp int64) ([][]float64, error)
	FetchTopMarkets(ctx context.Context, vsCurrency string, limit int) ([]MarketAsset, error)
}

var (
	defaultServiceMu sync.RWMutex
	defaultService   Service = newCoinGeckoService(config.MarketProvider{
		Used:    true,
		Name:    "coingecko",
		BaseURL: defaultCoinGeckoBaseURL,
	})
)

func Configure(cfg *config.Config) error {
	if cfg == nil {
		return fmt.Errorf("marketdata config is required")
	}

	provider, err := cfg.ActiveMarketProvider()
	if err != nil {
		return err
	}

	service, err := NewService(*provider)
	if err != nil {
		return err
	}

	defaultServiceMu.Lock()
	defaultService = service
	defaultServiceMu.Unlock()
	return nil
}

func Default() Service {
	defaultServiceMu.RLock()
	defer defaultServiceMu.RUnlock()
	return defaultService
}

func SetDefaultForTest(service Service) {
	defaultServiceMu.Lock()
	defaultService = service
	defaultServiceMu.Unlock()
}

func NewService(provider config.MarketProvider) (Service, error) {
	name := strings.ToLower(strings.TrimSpace(provider.Name))
	switch name {
	case "", "coingecko":
		if strings.TrimSpace(provider.BaseURL) == "" {
			provider.BaseURL = defaultCoinGeckoBaseURL
		}
		return newCoinGeckoService(provider), nil
	default:
		return nil, fmt.Errorf("unsupported market provider: %s", provider.Name)
	}
}
