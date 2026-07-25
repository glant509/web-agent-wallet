package httpapi

import (
	"net/http"
	"strconv"
	"strings"

	"web3-service-agent/internal/marketdata"
	dextool "web3-service-agent/tools/dex"
)

type tradeKlinesResponse struct {
	Item      dextool.TradeKlineSeries `json:"item"`
	Source    string                   `json:"source"`
	UpdatedAt string                   `json:"updated_at"`
}

func (s *Server) handleTradeKlines(w http.ResponseWriter, r *http.Request) {
	limit := 32
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
	}

	request := dextool.TradeKlinesRequest{
		AssetID:    strings.TrimSpace(r.URL.Query().Get("id")),
		Token:      strings.TrimSpace(r.URL.Query().Get("token")),
		VSCurrency: strings.TrimSpace(r.URL.Query().Get("vs_currency")),
		Interval:   strings.TrimSpace(r.URL.Query().Get("interval")),
		Limit:      limit,
		Days:       1,
	}

	item, err := dextool.FetchTradeKlines(r.Context(), request)
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}

	writeJSON(w, http.StatusOK, tradeKlinesResponse{
		Item:      item,
		Source:    marketdata.Default().ProviderName(),
		UpdatedAt: nowRFC3339(),
	})
}
