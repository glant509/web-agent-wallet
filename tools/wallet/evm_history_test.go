package wallet

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetchEVMTransactionHistoryIncludesIncomingAndOutgoing(t *testing.T) {
	explorer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		action := r.URL.Query().Get("action")
		var result any = []any{}
		if action == "txlist" {
			result = []map[string]string{
				{
					"timeStamp": "1789516800", "hash": "0x" + strings.Repeat("a", 64),
					"from": "0x2222222222222222222222222222222222222222", "to": "0x1111111111111111111111111111111111111111",
					"value": "500000000000000", "gasUsed": "21000", "gasPrice": "1000000000", "isError": "0", "transactionIndex": "1",
				},
				{
					"timeStamp": "1789516700", "hash": "0x" + strings.Repeat("b", 64),
					"from": "0x1111111111111111111111111111111111111111", "to": "0x3333333333333333333333333333333333333333",
					"value": "1000000000000000", "gasUsed": "21000", "gasPrice": "1000000000", "isError": "0", "transactionIndex": "2",
				},
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "1", "message": "OK", "result": result})
	}))
	defer explorer.Close()

	previous := evmHistoryExplorerURLs["ethereum"]
	evmHistoryExplorerURLs["ethereum"] = explorer.URL
	t.Cleanup(func() { evmHistoryExplorerURLs["ethereum"] = previous })

	items, err := FetchEVMTransactionHistory(context.Background(), EVMTransactionHistoryRequest{
		ChainID: "ethereum",
		Address: "0x1111111111111111111111111111111111111111",
	})
	if err != nil {
		t.Fatalf("fetch history: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 history items, got %d: %+v", len(items), items)
	}
	if items[0].Direction != "receive" || items[0].Amount != "0.0005" {
		t.Fatalf("unexpected incoming item: %+v", items[0])
	}
	if items[1].Direction != "send" || items[1].Amount != "0.001" || items[1].Fee != "0.000021" {
		t.Fatalf("unexpected outgoing item: %+v", items[1])
	}
}

func TestFetchEVMTransactionHistoryRejectsInvalidAddress(t *testing.T) {
	_, err := FetchEVMTransactionHistory(context.Background(), EVMTransactionHistoryRequest{ChainID: "ethereum", Address: "bad"})
	if err == nil {
		t.Fatal("expected invalid address error")
	}
}

func TestFetchEVMTransactionHistoryTreatsEmptySupportedChainResultsAsSuccess(t *testing.T) {
	explorer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		message := "No transactions found"
		if r.URL.Query().Get("action") == "tokentx" {
			message = "No token transfers found"
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":  "0",
			"message": message,
			"result":  []any{},
		})
	}))
	defer explorer.Close()

	for _, chainID := range []string{"ethereum", "base", "arbitrum", "optimism", "polygon", "avalanche"} {
		t.Run(chainID, func(t *testing.T) {
			previous := evmHistoryExplorerURLs[chainID]
			evmHistoryExplorerURLs[chainID] = explorer.URL
			t.Cleanup(func() { evmHistoryExplorerURLs[chainID] = previous })

			items, err := FetchEVMTransactionHistory(context.Background(), EVMTransactionHistoryRequest{
				ChainID: chainID,
				Address: "0x1111111111111111111111111111111111111111",
			})
			if err != nil {
				t.Fatalf("empty %s history should not fail: %v", chainID, err)
			}
			if len(items) != 0 {
				t.Fatalf("expected empty %s history, got %d items: %+v", chainID, len(items), items)
			}
		})
	}
}

func TestFetchBNBTransactionHistoryHandlesEmptyResults(t *testing.T) {
	explorer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Entity string `json:"entity"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"entity": payload.Entity, "count": 0, "data": []any{}})
	}))
	defer explorer.Close()
	previous := bnbHistoryAPIURL
	bnbHistoryAPIURL = explorer.URL
	t.Cleanup(func() { bnbHistoryAPIURL = previous })
	items, err := FetchEVMTransactionHistory(context.Background(), EVMTransactionHistoryRequest{
		ChainID: "bsc", Address: "0x1111111111111111111111111111111111111111",
	})
	if err != nil || len(items) != 0 {
		t.Fatalf("expected empty BNB history, got items=%+v err=%v", items, err)
	}
}

func TestFetchBNBTransactionHistoryIncludesNativeAndTokenTransfers(t *testing.T) {
	explorer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Entity string `json:"entity"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		var data any = []map[string]any{{
			"hash": "0x" + strings.Repeat("a", 64), "fromAddress": "0x2222222222222222222222222222222222222222",
			"toAddress": "0x1111111111111111111111111111111111111111", "value": "0.5",
			"gasUsed": "21000", "gasPrice": "1000000000", "timestamp": "2026-09-21T10:00:00.000Z",
			"status": true, "txIndex": 1,
		}}
		if payload.Entity == "token_transfers" {
			data = []map[string]any{{
				"txHash": "0x" + strings.Repeat("b", 64), "logIndex": 2,
				"fromAddress": "0x1111111111111111111111111111111111111111",
				"toAddress":   "0x3333333333333333333333333333333333333333",
				"value":       "1500000000000000000", "tokenAddress": "0x55d398326f99059fF775485246999027B3197955",
				"timestamp": "2026-09-21T11:00:00.000Z",
			}}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"entity": payload.Entity, "count": 1, "data": data})
	}))
	defer explorer.Close()
	previous := bnbHistoryAPIURL
	bnbHistoryAPIURL = explorer.URL
	t.Cleanup(func() { bnbHistoryAPIURL = previous })
	items, err := FetchEVMTransactionHistory(context.Background(), EVMTransactionHistoryRequest{
		ChainID: "bsc", Address: "0x1111111111111111111111111111111111111111",
	})
	if err != nil {
		t.Fatalf("fetch BNB history: %v", err)
	}
	if len(items) != 2 || items[0].Direction != "send" || items[0].TokenSymbol != "USDT" || items[0].Amount != "1.5" {
		t.Fatalf("unexpected BNB token transfer: %+v", items)
	}
	if items[1].Direction != "receive" || items[1].TokenSymbol != "BNB" || items[1].Amount != "0.5" {
		t.Fatalf("unexpected BNB native transfer: %+v", items[1])
	}
}

func TestFetchBNBTransactionHistoryDoesNotMistakeExplorerFailureForEmptyHistory(t *testing.T) {
	explorer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer explorer.Close()
	previous := bnbHistoryAPIURL
	bnbHistoryAPIURL = explorer.URL
	t.Cleanup(func() { bnbHistoryAPIURL = previous })
	items, err := FetchEVMTransactionHistory(context.Background(), EVMTransactionHistoryRequest{
		ChainID: "bsc", Address: "0x1111111111111111111111111111111111111111",
	})
	if err == nil || !strings.Contains(err.Error(), "429") || items != nil {
		t.Fatalf("expected BNB explorer rate-limit error, got items=%+v err=%v", items, err)
	}
}
