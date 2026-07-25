package httpapi

import (
	"net/http"
	"strings"

	"web3-service-agent/internal/marketdata"
	dextool "web3-service-agent/tools/dex"
)

type tradeTokenResponse struct {
	Item      dextool.TradeDashboard `json:"item"`
	Source    string                 `json:"source"`
	UpdatedAt string                 `json:"updated_at"`
}

func (s *Server) handleTradeToken(w http.ResponseWriter, r *http.Request) {
	request := dextool.TradeDashboardRequest{
		AssetID:    strings.TrimSpace(r.URL.Query().Get("id")),
		Token:      strings.TrimSpace(r.URL.Query().Get("token")),
		VSCurrency: strings.TrimSpace(r.URL.Query().Get("vs_currency")),
	}

	item, err := dextool.FetchTradeDashboard(r.Context(), request)
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}

	writeJSON(w, http.StatusOK, tradeTokenResponse{
		Item:      item,
		Source:    marketdata.Default().ProviderName(),
		UpdatedAt: nowRFC3339(),
	})
}
