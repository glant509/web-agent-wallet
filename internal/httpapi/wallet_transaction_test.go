package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPrepareEVMTransactionRejectsInvalidInput(t *testing.T) {
	handler := New(nil, nil)
	request := httptest.NewRequest(http.MethodPost, "/v1/wallet/evm/prepare", strings.NewReader(`{"chain_id":"ethereum"}`))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestBroadcastEVMTransactionRejectsUnsignedPayload(t *testing.T) {
	handler := New(nil, nil)
	request := httptest.NewRequest(http.MethodPost, "/v1/wallet/evm/broadcast", strings.NewReader(`{"chain_id":"ethereum","raw_transaction":"not-signed"}`))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestEVMTransactionHistoryRejectsInvalidAddress(t *testing.T) {
	handler := New(nil, nil)
	request := httptest.NewRequest(http.MethodPost, "/v1/wallet/evm/history", strings.NewReader(`{"chain_id":"ethereum","address":"bad"}`))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", recorder.Code, recorder.Body.String())
	}
}
