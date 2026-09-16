package httpapi

import (
	"context"
	"net/http"
	"strings"
	"time"

	wallettool "web3-service-agent/tools/wallet"
)

type evmBroadcastRequest struct {
	ChainID        string `json:"chain_id"`
	RawTransaction string `json:"raw_transaction"`
}

func (s *Server) handlePrepareEVMTransaction(w http.ResponseWriter, r *http.Request) {
	var request wallettool.EVMTransactionRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	plan, err := wallettool.PrepareEVMTransaction(ctx, request)
	if err != nil {
		status := http.StatusBadGateway
		message := err.Error()
		if strings.Contains(message, "invalid") || strings.Contains(message, "required") || strings.Contains(message, "not supported") || strings.Contains(message, "insufficient") || strings.Contains(message, "not enough") || strings.Contains(message, "decimal places") {
			status = http.StatusBadRequest
		}
		writeError(w, status, err)
		return
	}
	writeJSON(w, http.StatusOK, plan)
}

func (s *Server) handleBroadcastEVMTransaction(w http.ResponseWriter, r *http.Request) {
	var request evmBroadcastRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	hash, err := wallettool.BroadcastEVMTransaction(ctx, request.ChainID, request.RawTransaction)
	if err != nil {
		status := http.StatusBadGateway
		if strings.Contains(err.Error(), "invalid") || strings.Contains(err.Error(), "not supported") {
			status = http.StatusBadRequest
		}
		writeError(w, status, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"transaction_hash": hash})
}

func (s *Server) handleEVMTransactionHistory(w http.ResponseWriter, r *http.Request) {
	var request wallettool.EVMTransactionHistoryRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	items, err := wallettool.FetchEVMTransactionHistory(ctx, request)
	if err != nil {
		status := http.StatusBadGateway
		if strings.Contains(err.Error(), "invalid") || strings.Contains(err.Error(), "not supported") {
			status = http.StatusBadRequest
		}
		writeError(w, status, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}
