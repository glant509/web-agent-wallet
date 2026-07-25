package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"web3-service-agent/internal/marketdata"
)

type marketTopItem struct {
	ID                       string  `json:"id"`
	Name                     string  `json:"name"`
	Symbol                   string  `json:"symbol"`
	CurrentPrice             float64 `json:"current_price"`
	PriceChangePercentage24h float64 `json:"price_change_percentage_24h"`
	MarketCapRank            int     `json:"market_cap_rank"`
}

type marketTopResponse struct {
	Items     []marketTopItem `json:"items"`
	Source    string          `json:"source"`
	UpdatedAt string          `json:"updated_at"`
}

func (s *Server) handleMarketTop(w http.ResponseWriter, r *http.Request) {
	items, err := fetchTopMarketItems(r.Context())
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}

	writeJSON(w, http.StatusOK, marketTopResponse{
		Items:     items,
		Source:    marketdata.Default().ProviderName(),
		UpdatedAt: nowRFC3339(),
	})
}

func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func fetchTopMarketItems(ctx context.Context) ([]marketTopItem, error) {
	markets, err := marketdata.Default().FetchTopMarkets(ctx, "usd", 10)
	if err != nil {
		return nil, fmt.Errorf("request top market data: %w", err)
	}

	items := make([]marketTopItem, 0, len(markets))
	for _, market := range markets {
		items = append(items, marketTopItem{
			ID:                       market.ID,
			Name:                     market.Name,
			Symbol:                   market.Symbol,
			CurrentPrice:             market.CurrentPrice,
			PriceChangePercentage24h: market.PriceChange24h,
			MarketCapRank:            market.MarketCapRank,
		})
	}
	return items, nil
}
