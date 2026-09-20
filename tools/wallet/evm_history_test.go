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

	for _, chainID := range []string{"ethereum", "base", "arbitrum", "optimism", "polygon"} {
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

func TestFetchEVMTransactionHistoryReportsUnsupportedBuiltInChains(t *testing.T) {
	for _, chainID := range []string{"bsc", "avalanche"} {
		t.Run(chainID, func(t *testing.T) {
			items, err := FetchEVMTransactionHistory(context.Background(), EVMTransactionHistoryRequest{
				ChainID: chainID,
				Address: "0x1111111111111111111111111111111111111111",
			})
			if err == nil || !strings.Contains(err.Error(), "transaction history is not supported") {
				t.Fatalf("expected unsupported history error for %s, got items=%+v err=%v", chainID, items, err)
			}
		})
	}
}
