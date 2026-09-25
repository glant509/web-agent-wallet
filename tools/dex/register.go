package dex

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"web3-service-agent/internal/marketdata"
	"web3-service-agent/internal/tool"
)

const dexTokenTradeDashboardToolName = "dex_token_trade_dashboard"

//go:embed dex_token_trade_dashboard.json
var dexTokenTradeDashboardDefinition []byte

//go:embed dex_token_trade_klines.json
var dexTokenTradeKlinesDefinition []byte

type TradeDashboardRequest struct {
	Token      string `json:"token"`
	AssetID    string `json:"asset_id"`
	VSCurrency string `json:"vs_currency"`
}

type TradeDashboard struct {
	AssetID                  string   `json:"id"`
	Name                     string   `json:"name"`
	Symbol                   string   `json:"symbol"`
	VSCurrency               string   `json:"vs_currency"`
	CurrentPrice             float64  `json:"current_price"`
	PriceChangePercentage24h float64  `json:"price_change_percentage_24h"`
	High24h                  float64  `json:"high_24h"`
	Low24h                   float64  `json:"low_24h"`
	TotalVolume              float64  `json:"total_volume"`
	MarketCap                float64  `json:"market_cap"`
	FullyDilutedValuation    float64  `json:"fully_diluted_valuation"`
	CirculatingSupply        float64  `json:"circulating_supply"`
	TotalSupply              *float64 `json:"total_supply"`
	MarketCapRank            int      `json:"market_cap_rank"`
}

func Register(registry *tool.Registry) error {
	registrations := []struct {
		name    string
		content []byte
		handler tool.Handler
	}{
		{name: "dex_token_trade_dashboard.json", content: dexTokenTradeDashboardDefinition, handler: handleTokenTradeDashboard},
		{name: "dex_token_trade_klines.json", content: dexTokenTradeKlinesDefinition, handler: handleTokenTradeKlines},
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

func handleTokenTradeDashboard(ctx context.Context, call tool.Call) (tool.Result, error) {
	var request TradeDashboardRequest
	if err := json.Unmarshal(call.Arguments, &request); err != nil {
		return tool.Result{}, fmt.Errorf("decode trade dashboard arguments: %w", err)
	}

	dashboard, err := FetchTradeDashboard(ctx, request)
	if err != nil {
		return tool.Result{}, err
	}

	return tool.Result{Content: renderTradeDashboard(dashboard)}, nil
}

func FetchTradeDashboard(ctx context.Context, request TradeDashboardRequest) (TradeDashboard, error) {
	vsCurrency := strings.ToLower(strings.TrimSpace(request.VSCurrency))
	if vsCurrency == "" {
		vsCurrency = "usd"
	}

	assetID := strings.TrimSpace(request.AssetID)
	if assetID == "" {
		token := strings.TrimSpace(request.Token)
		if token == "" {
			return TradeDashboard{}, fmt.Errorf("either token or asset_id is required")
		}

		match, err := marketdata.Default().SearchCoin(ctx, token)
		if err != nil {
			return TradeDashboard{}, err
		}
		assetID = match.ID
	}

	return fetchTradeDashboardMarket(ctx, assetID, vsCurrency)
}

func fetchTradeDashboardMarket(ctx context.Context, assetID, vsCurrency string) (TradeDashboard, error) {
	market, err := marketdata.Default().FetchMarket(ctx, assetID, vsCurrency)
	if err != nil {
		return TradeDashboard{}, err
	}

	return TradeDashboard{
		AssetID:                  market.ID,
		Name:                     market.Name,
		Symbol:                   market.Symbol,
		VSCurrency:               vsCurrency,
		CurrentPrice:             market.CurrentPrice,
		PriceChangePercentage24h: market.PriceChange24h,
		High24h:                  market.High24h,
		Low24h:                   market.Low24h,
		TotalVolume:              market.TotalVolume,
		MarketCap:                market.MarketCap,
		FullyDilutedValuation:    market.FullyDilutedValuation,
		CirculatingSupply:        market.CirculatingSupply,
		TotalSupply:              market.TotalSupply,
		MarketCapRank:            market.MarketCapRank,
	}, nil
}

func renderTradeDashboard(dashboard TradeDashboard) string {
	lines := []string{
		fmt.Sprintf("Trade dashboard for %s (%s):", dashboard.Name, strings.ToUpper(dashboard.Symbol)),
		fmt.Sprintf("- asset_id: %s", dashboard.AssetID),
		fmt.Sprintf("- price_%s: %s", dashboard.VSCurrency, formatPrice(dashboard.CurrentPrice)),
		fmt.Sprintf("- 24h_change_pct: %s", formatPercent(dashboard.PriceChangePercentage24h)),
		fmt.Sprintf("- 24h_high_%s: %s", dashboard.VSCurrency, formatPrice(dashboard.High24h)),
		fmt.Sprintf("- 24h_low_%s: %s", dashboard.VSCurrency, formatPrice(dashboard.Low24h)),
		fmt.Sprintf("- 24h_volume_%s: %s", dashboard.VSCurrency, formatAmount(dashboard.TotalVolume)),
		fmt.Sprintf("- market_cap_%s: %s", dashboard.VSCurrency, formatAmount(dashboard.MarketCap)),
		fmt.Sprintf("- circulating_supply: %s", formatAmount(dashboard.CirculatingSupply)),
	}

	if dashboard.TotalSupply != nil && *dashboard.TotalSupply > 0 {
		lines = append(lines, fmt.Sprintf("- total_supply: %s", formatAmount(*dashboard.TotalSupply)))
	} else {
		lines = append(lines, "- total_supply: unavailable")
	}
	if dashboard.FullyDilutedValuation > 0 {
		lines = append(lines, fmt.Sprintf("- fdv_%s: %s", dashboard.VSCurrency, formatAmount(dashboard.FullyDilutedValuation)))
	}
	if dashboard.MarketCapRank > 0 {
		lines = append(lines, fmt.Sprintf("- market_cap_rank: %d", dashboard.MarketCapRank))
	}
	lines = append(lines, fmt.Sprintf("- source: %s", marketdata.Default().ProviderName()))

	return strings.Join(lines, "\n")
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

func formatPercent(value float64) string {
	sign := ""
	if value > 0 {
		sign = "+"
	}
	return sign + strconv.FormatFloat(value, 'f', 2, 64) + "%"
}
