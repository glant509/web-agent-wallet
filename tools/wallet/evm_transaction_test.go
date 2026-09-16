package wallet

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPrepareEVMNativeMaxTransaction(t *testing.T) {
	rpc := newEVMTransactionRPCServer(t)
	defer rpc.Close()
	setTestChainRPC(t, "ethereum", rpc.URL)

	plan, err := PrepareEVMTransaction(context.Background(), EVMTransactionRequest{
		ChainID: "ethereum",
		From:    "0x1111111111111111111111111111111111111111",
		To:      "0x2222222222222222222222222222222222222222",
		SendMax: true,
	})
	if err != nil {
		t.Fatalf("prepare native max transaction: %v", err)
	}
	if !plan.IsNative || plan.TransactionType != 2 {
		t.Fatalf("unexpected transaction plan: %+v", plan)
	}
	if plan.GasLimit != "0x5e56" {
		t.Fatalf("expected padded gas limit 0x5e56, got %s", plan.GasLimit)
	}
	if plan.EstimatedFeeRaw != "72450000000000" {
		t.Fatalf("unexpected fee: %s", plan.EstimatedFeeRaw)
	}
	if plan.AmountRaw != "999927550000000000" || plan.Value != "0xde074cf12e06c00" {
		t.Fatalf("unexpected max amount: raw=%s value=%s", plan.AmountRaw, plan.Value)
	}
}

func TestPrepareEVMTokenTransactionSeparatesNativeFee(t *testing.T) {
	rpc := newEVMTransactionRPCServer(t)
	defer rpc.Close()
	setTestChainRPC(t, "ethereum", rpc.URL)

	plan, err := PrepareEVMTransaction(context.Background(), EVMTransactionRequest{
		ChainID:      "ethereum",
		From:         "0x1111111111111111111111111111111111111111",
		To:           "0x2222222222222222222222222222222222222222",
		TokenAddress: "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",
		Amount:       "12.5",
	})
	if err != nil {
		t.Fatalf("prepare token transaction: %v", err)
	}
	if plan.IsNative || plan.TokenSymbol != "USDC" || plan.TokenDecimals != 6 {
		t.Fatalf("unexpected token plan: %+v", plan)
	}
	if plan.AmountRaw != "12500000" || plan.Value != "0x0" {
		t.Fatalf("unexpected token amount/value: %+v", plan)
	}
	if plan.To != "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48" {
		t.Fatalf("expected token contract as transaction recipient, got %s", plan.To)
	}
	if !strings.HasPrefix(plan.Data, "0xa9059cbb") || !strings.Contains(plan.Data, strings.Repeat("0", 24)+"2222222222222222222222222222222222222222") {
		t.Fatalf("unexpected ERC-20 transfer calldata: %s", plan.Data)
	}
}

func TestBroadcastEVMTransaction(t *testing.T) {
	rpc := newEVMTransactionRPCServer(t)
	defer rpc.Close()
	setTestChainRPC(t, "ethereum", rpc.URL)

	hash, err := BroadcastEVMTransaction(context.Background(), "ethereum", "0x02abcdef")
	if err != nil {
		t.Fatalf("broadcast transaction: %v", err)
	}
	if hash != "0x"+strings.Repeat("a", 64) {
		t.Fatalf("unexpected transaction hash: %s", hash)
	}
}

func TestParseDecimalUnits(t *testing.T) {
	value, err := parseDecimalUnits("12.3456", 6)
	if err != nil || value.String() != "12345600" {
		t.Fatalf("unexpected parsed value %v, err=%v", value, err)
	}
	for _, invalid := range []string{"", "0", "-1", "1.0000001", "1e3"} {
		if _, err := parseDecimalUnits(invalid, 6); err == nil {
			t.Fatalf("expected %q to be rejected", invalid)
		}
	}
}

func newEVMTransactionRPCServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Method string `json:"method"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode rpc request: %v", err)
		}
		var result any
		switch request.Method {
		case "eth_chainId":
			result = "0x1"
		case "eth_getTransactionCount":
			result = "0x7"
		case "eth_getBalance":
			result = "0xde0b6b3a7640000"
		case "eth_estimateGas":
			result = "0x5208"
		case "eth_gasPrice":
			result = "0x77359400"
		case "eth_getBlockByNumber":
			result = map[string]string{"baseFeePerGas": "0x3b9aca00"}
		case "eth_maxPriorityFeePerGas":
			result = "0x3b9aca00"
		case "eth_call":
			result = "0x5f5e100"
		case "eth_sendRawTransaction":
			result = "0x" + strings.Repeat("a", 64)
		default:
			t.Fatalf("unexpected rpc method %q", request.Method)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": 1, "result": result})
	}))
}

func setTestChainRPC(t *testing.T, chainID, rpcURL string) {
	t.Helper()
	chainRPCConfig.mu.Lock()
	previous := chainRPCConfig.urls
	chainRPCConfig.urls = map[string][]string{chainID: {rpcURL}}
	chainRPCConfig.mu.Unlock()
	t.Cleanup(func() {
		chainRPCConfig.mu.Lock()
		chainRPCConfig.urls = previous
		chainRPCConfig.mu.Unlock()
	})
}
