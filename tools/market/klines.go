package market

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"web3-service-agent/internal/marketdata"
	"web3-service-agent/internal/tool"
)

const defaultKlineLimit = 24

type marketKlinesRequest struct {
	Token         string `json:"token"`
	TokenAddress  string `json:"token_address"`
	Chain         string `json:"chain"`
	VSCurrency    string `json:"vs_currency"`
	Days          int    `json:"days"`
	FromTimestamp int64  `json:"from_timestamp"`
	ToTimestamp   int64  `json:"to_timestamp"`
	Interval      string `json:"interval"`
	Limit         int    `json:"limit"`
}

type marketKlineAsset struct {
	AssetID      string
	Name         string
	Symbol       string
	TokenAddress string
	Chain        string
}

type marketKlineCandle struct {
	Start time.Time
	Open  float64
	High  float64
	Low   float64
	Close float64
}

func handleMarketTokenKlines(ctx context.Context, call tool.Call) (tool.Result, error) {
	var request marketKlinesRequest
	if err := json.Unmarshal(call.Arguments, &request); err != nil {
		return tool.Result{}, fmt.Errorf("decode market klines arguments: %w", err)
	}

	vsCurrency := strings.ToLower(strings.TrimSpace(request.VSCurrency))
	if vsCurrency == "" {
		vsCurrency = "usd"
	}

	token := strings.TrimSpace(request.Token)
	tokenAddress := strings.TrimSpace(request.TokenAddress)
	chain := normalizeChainSlug(request.Chain)
	if token == "" && tokenAddress == "" {
		return tool.Result{}, fmt.Errorf("either token or token_address is required")
	}
	if tokenAddress != "" && chain == "" {
		return tool.Result{}, fmt.Errorf("chain is required when token_address is provided")
	}

	historical, err := hasHistoricalRange(request.FromTimestamp, request.ToTimestamp)
	if err != nil {
		return tool.Result{}, err
	}

	days := request.Days
	if !historical {
		if days == 0 {
			days = 1
		}
		if days < 1 || days > 365 {
			return tool.Result{}, fmt.Errorf("days must be between 1 and 365")
		}
	}

	limit := request.Limit
	switch {
	case limit == 0:
		limit = defaultKlineLimit
	case limit < 0:
		return tool.Result{}, fmt.Errorf("limit must be positive")
	case limit > 100:
		return tool.Result{}, fmt.Errorf("limit must be 100 or less")
	}

	interval, err := resolveKlineInterval(request.Interval, request.FromTimestamp, request.ToTimestamp, days, historical)
	if err != nil {
		return tool.Result{}, err
	}

	var (
		asset  marketKlineAsset
		prices [][]float64
	)

	switch {
	case tokenAddress != "":
		asset = marketKlineAsset{
			TokenAddress: tokenAddress,
			Chain:        chain,
		}

		if historical {
			prices, err = marketdata.Default().FetchContractMarketChartRange(ctx, chain, tokenAddress, vsCurrency, request.FromTimestamp, request.ToTimestamp)
		} else {
			prices, err = marketdata.Default().FetchContractMarketChart(ctx, chain, tokenAddress, vsCurrency, days)
		}
		if err != nil {
			return tool.Result{}, err
		}
	default:
		match, err := marketdata.Default().SearchCoin(ctx, token)
		if err != nil {
			return tool.Result{}, err
		}

		asset = marketKlineAsset{
			AssetID: match.ID,
			Name:    match.Name,
			Symbol:  match.Symbol,
		}

		if historical {
			prices, err = marketdata.Default().FetchMarketChartRange(ctx, match.ID, vsCurrency, request.FromTimestamp, request.ToTimestamp)
		} else {
			prices, err = marketdata.Default().FetchMarketChart(ctx, match.ID, vsCurrency, days)
		}
		if err != nil {
			return tool.Result{}, err
		}
	}

	candles, err := aggregatePriceCandles(prices, interval)
	if err != nil {
		return tool.Result{}, err
	}
	if len(candles) == 0 {
		return tool.Result{}, fmt.Errorf("no kline data returned")
	}

	return tool.Result{
		Content: renderMarketKlines(asset, vsCurrency, interval, candles, limit, historical),
	}, nil
}

func hasHistoricalRange(fromTimestamp, toTimestamp int64) (bool, error) {
	switch {
	case fromTimestamp == 0 && toTimestamp == 0:
		return false, nil
	case fromTimestamp == 0 || toTimestamp == 0:
		return false, fmt.Errorf("from_timestamp and to_timestamp must be provided together")
	case toTimestamp <= fromTimestamp:
		return false, fmt.Errorf("to_timestamp must be greater than from_timestamp")
	default:
		return true, nil
	}
}

func resolveKlineInterval(raw string, fromTimestamp, toTimestamp int64, days int, historical bool) (time.Duration, error) {
	if strings.TrimSpace(raw) != "" {
		return parseKlineInterval(raw)
	}

	if historical {
		return defaultKlineInterval(time.Unix(toTimestamp, 0).Sub(time.Unix(fromTimestamp, 0))), nil
	}

	return defaultKlineInterval(time.Duration(days) * 24 * time.Hour), nil
}

func parseKlineInterval(raw string) (time.Duration, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "5m":
		return 5 * time.Minute, nil
	case "15m":
		return 15 * time.Minute, nil
	case "30m":
		return 30 * time.Minute, nil
	case "1h":
		return time.Hour, nil
	case "4h":
		return 4 * time.Hour, nil
	case "1d":
		return 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("unsupported interval %q; use one of 5m, 15m, 30m, 1h, 4h, 1d", raw)
	}
}

func defaultKlineInterval(rangeDuration time.Duration) time.Duration {
	switch {
	case rangeDuration <= 24*time.Hour:
		return 15 * time.Minute
	case rangeDuration <= 7*24*time.Hour:
		return time.Hour
	case rangeDuration <= 30*24*time.Hour:
		return 4 * time.Hour
	default:
		return 24 * time.Hour
	}
}

func aggregatePriceCandles(prices [][]float64, interval time.Duration) ([]marketKlineCandle, error) {
	if interval <= 0 {
		return nil, fmt.Errorf("kline interval must be positive")
	}
	if len(prices) == 0 {
		return nil, fmt.Errorf("no price data returned")
	}

	type pricePoint struct {
		timestamp time.Time
		price     float64
	}

	points := make([]pricePoint, 0, len(prices))
	for _, item := range prices {
		if len(item) < 2 {
			return nil, fmt.Errorf("invalid price data point returned")
		}
		points = append(points, pricePoint{
			timestamp: time.UnixMilli(int64(item[0])).UTC(),
			price:     item[1],
		})
	}

	sort.Slice(points, func(i, j int) bool {
		return points[i].timestamp.Before(points[j].timestamp)
	})

	candles := make([]marketKlineCandle, 0, len(points))
	for _, point := range points {
		bucketStart := point.timestamp.Truncate(interval)
		if len(candles) == 0 || !candles[len(candles)-1].Start.Equal(bucketStart) {
			candles = append(candles, marketKlineCandle{
				Start: bucketStart,
				Open:  point.price,
				High:  point.price,
				Low:   point.price,
				Close: point.price,
			})
			continue
		}

		candle := &candles[len(candles)-1]
		if point.price > candle.High {
			candle.High = point.price
		}
		if point.price < candle.Low {
			candle.Low = point.price
		}
		candle.Close = point.price
	}

	return candles, nil
}

func renderMarketKlines(asset marketKlineAsset, vsCurrency string, interval time.Duration, candles []marketKlineCandle, limit int, historical bool) string {
	totalCandles := len(candles)
	if limit > totalCandles {
		limit = totalCandles
	}
	displayedCandles := candles[totalCandles-limit:]

	title := "Klines for token:"
	switch {
	case asset.Name != "":
		title = fmt.Sprintf("Klines for %s (%s):", asset.Name, strings.ToUpper(asset.Symbol))
	case asset.TokenAddress != "":
		title = fmt.Sprintf("Klines for %s on %s:", asset.TokenAddress, asset.Chain)
	}

	mode := "recent"
	if historical {
		mode = "historical"
	}

	lines := []string{
		title,
		fmt.Sprintf("- mode: %s", mode),
		fmt.Sprintf("- vs_currency: %s", vsCurrency),
		fmt.Sprintf("- interval: %s", formatKlineInterval(interval)),
		fmt.Sprintf("- total_candles: %d", totalCandles),
		fmt.Sprintf("- candles_returned: %d", len(displayedCandles)),
	}

	if asset.AssetID != "" {
		lines = append(lines, fmt.Sprintf("- asset_id: %s", asset.AssetID))
	}
	if asset.TokenAddress != "" {
		lines = append(lines, fmt.Sprintf("- token_address: %s", asset.TokenAddress))
	}
	if asset.Chain != "" {
		lines = append(lines, fmt.Sprintf("- chain: %s", asset.Chain))
	}
	if totalCandles > len(displayedCandles) {
		lines = append(lines, fmt.Sprintf("- note: showing latest %d candles from requested range", len(displayedCandles)))
	}
	lines = append(lines, fmt.Sprintf("- source: %s", marketdata.Default().ProviderName()), "Candles:")

	for _, candle := range displayedCandles {
		lines = append(lines, fmt.Sprintf(
			"- %s open=%s high=%s low=%s close=%s",
			candle.Start.Format(time.RFC3339),
			formatPrice(candle.Open),
			formatPrice(candle.High),
			formatPrice(candle.Low),
			formatPrice(candle.Close),
		))
	}

	return strings.Join(lines, "\n")
}

func formatKlineInterval(interval time.Duration) string {
	switch interval {
	case 5 * time.Minute:
		return "5m"
	case 15 * time.Minute:
		return "15m"
	case 30 * time.Minute:
		return "30m"
	case time.Hour:
		return "1h"
	case 4 * time.Hour:
		return "4h"
	case 24 * time.Hour:
		return "1d"
	default:
		return interval.String()
	}
}
