package dex

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

const dexTokenTradeKlinesToolName = "dex_token_trade_klines"

type TradeKlinesRequest struct {
	Token      string `json:"token"`
	AssetID    string `json:"asset_id"`
	VSCurrency string `json:"vs_currency"`
	Interval   string `json:"interval"`
	Limit      int    `json:"limit"`
	Days       int    `json:"days"`
}

type TradeKlineCandle struct {
	Timestamp string  `json:"timestamp"`
	Open      float64 `json:"open"`
	High      float64 `json:"high"`
	Low       float64 `json:"low"`
	Close     float64 `json:"close"`
}

type TradeKlineSeries struct {
	AssetID    string             `json:"asset_id"`
	Name       string             `json:"name"`
	Symbol     string             `json:"symbol"`
	VSCurrency string             `json:"vs_currency"`
	Interval   string             `json:"interval"`
	Candles    []TradeKlineCandle `json:"candles"`
}

func handleTokenTradeKlines(ctx context.Context, call tool.Call) (tool.Result, error) {
	var request TradeKlinesRequest
	if err := json.Unmarshal(call.Arguments, &request); err != nil {
		return tool.Result{}, fmt.Errorf("decode trade klines arguments: %w", err)
	}

	series, err := FetchTradeKlines(ctx, request)
	if err != nil {
		return tool.Result{}, err
	}

	return tool.Result{Content: renderTradeKlines(series)}, nil
}

func FetchTradeKlines(ctx context.Context, request TradeKlinesRequest) (TradeKlineSeries, error) {
	vsCurrency := strings.ToLower(strings.TrimSpace(request.VSCurrency))
	if vsCurrency == "" {
		vsCurrency = "usd"
	}

	interval, err := parseTradeKlineInterval(request.Interval)
	if err != nil {
		return TradeKlineSeries{}, err
	}

	days := request.Days
	if days == 0 {
		days = 1
	}
	if days < 1 || days > 365 {
		return TradeKlineSeries{}, fmt.Errorf("days must be between 1 and 365")
	}

	limit := request.Limit
	switch {
	case limit == 0:
		limit = 32
	case limit < 0:
		return TradeKlineSeries{}, fmt.Errorf("limit must be positive")
	case limit > 100:
		return TradeKlineSeries{}, fmt.Errorf("limit must be 100 or less")
	}

	dashboard, err := FetchTradeDashboard(ctx, TradeDashboardRequest{
		Token:      request.Token,
		AssetID:    request.AssetID,
		VSCurrency: vsCurrency,
	})
	if err != nil {
		return TradeKlineSeries{}, err
	}

	prices, err := marketdata.Default().FetchMarketChart(ctx, dashboard.AssetID, vsCurrency, days)
	if err != nil {
		return TradeKlineSeries{}, err
	}

	candles, err := aggregateTradeCandles(prices, interval)
	if err != nil {
		return TradeKlineSeries{}, err
	}
	if len(candles) == 0 {
		return TradeKlineSeries{}, fmt.Errorf("no kline data returned")
	}
	if limit < len(candles) {
		candles = candles[len(candles)-limit:]
	}

	return TradeKlineSeries{
		AssetID:    dashboard.AssetID,
		Name:       dashboard.Name,
		Symbol:     dashboard.Symbol,
		VSCurrency: vsCurrency,
		Interval:   formatTradeKlineInterval(interval),
		Candles:    candles,
	}, nil
}

func parseTradeKlineInterval(raw string) (time.Duration, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		value = "15m"
	}

	switch value {
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

func aggregateTradeCandles(prices [][]float64, interval time.Duration) ([]TradeKlineCandle, error) {
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

	candles := make([]TradeKlineCandle, 0, len(points))
	for _, point := range points {
		bucketStart := point.timestamp.Truncate(interval)
		if len(candles) == 0 || candles[len(candles)-1].Timestamp != bucketStart.Format(time.RFC3339) {
			candles = append(candles, TradeKlineCandle{
				Timestamp: bucketStart.Format(time.RFC3339),
				Open:      point.price,
				High:      point.price,
				Low:       point.price,
				Close:     point.price,
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

func renderTradeKlines(series TradeKlineSeries) string {
	lines := []string{
		fmt.Sprintf("Trade klines for %s (%s):", series.Name, strings.ToUpper(series.Symbol)),
		fmt.Sprintf("- asset_id: %s", series.AssetID),
		fmt.Sprintf("- vs_currency: %s", series.VSCurrency),
		fmt.Sprintf("- interval: %s", series.Interval),
		fmt.Sprintf("- candles_returned: %d", len(series.Candles)),
		fmt.Sprintf("- source: %s", marketdata.Default().ProviderName()),
		"Candles:",
	}

	for _, candle := range series.Candles {
		lines = append(lines, fmt.Sprintf(
			"- %s open=%s high=%s low=%s close=%s",
			candle.Timestamp,
			formatPrice(candle.Open),
			formatPrice(candle.High),
			formatPrice(candle.Low),
			formatPrice(candle.Close),
		))
	}

	return strings.Join(lines, "\n")
}

func formatTradeKlineInterval(interval time.Duration) string {
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
