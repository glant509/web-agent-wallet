package httpapi

import (
	"net/http"

	wallettool "web3-service-agent/tools/wallet"
)

type assetPortfolioRequest struct {
	wallettool.AssetPortfolioRequest
}

type assetPortfolioResponse struct {
	wallettool.AssetPortfolio
	Updated string `json:"updated_at"`
}

func (s *Server) handleAssetPortfolio(w http.ResponseWriter, r *http.Request) {
	var request assetPortfolioRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	result, err := wallettool.FetchAssetPortfolio(r.Context(), request.AssetPortfolioRequest)
	if err != nil {
		switch err.Error() {
		case "selected_chain_id is required", "address for selected_chain_id is required", "addresses are required":
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeError(w, http.StatusBadGateway, err)
		return
	}

	writeJSON(w, http.StatusOK, assetPortfolioResponse{
		AssetPortfolio: result,
		Updated:        nowRFC3339(),
	})
}
