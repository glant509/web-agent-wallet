package wallet

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

var evmHistoryExplorerURLs = map[string]string{
	"ethereum": "https://eth.blockscout.com",
	"base":     "https://base.blockscout.com",
	"arbitrum": "https://arbitrum.blockscout.com",
	"optimism": "https://optimism.blockscout.com",
	"polygon":  "https://polygon.blockscout.com",
}

type EVMTransactionHistoryRequest struct {
	ChainID string `json:"chain_id"`
	Address string `json:"address"`
}

type EVMTransactionHistoryItem struct {
	ID          string `json:"id"`
	Hash        string `json:"hash"`
	ChainID     string `json:"chain_id"`
	Type        string `json:"type"`
	Direction   string `json:"direction"`
	Timestamp   string `json:"timestamp"`
	From        string `json:"from"`
	To          string `json:"to"`
	TokenSymbol string `json:"token_symbol"`
	Amount      string `json:"amount"`
	Fee         string `json:"fee,omitempty"`
	FeeSymbol   string `json:"fee_symbol,omitempty"`
	Status      string `json:"status"`
}

type blockscoutAccountResponse struct {
	Status  string          `json:"status"`
	Message string          `json:"message"`
	Result  json.RawMessage `json:"result"`
}

type blockscoutTransaction struct {
	TimeStamp        string `json:"timeStamp"`
	Hash             string `json:"hash"`
	From             string `json:"from"`
	To               string `json:"to"`
	Value            string `json:"value"`
	GasUsed          string `json:"gasUsed"`
	GasPrice         string `json:"gasPrice"`
	IsError          string `json:"isError"`
	TransactionIndex string `json:"transactionIndex"`
	TokenSymbol      string `json:"tokenSymbol"`
	TokenDecimal     string `json:"tokenDecimal"`
}

func FetchEVMTransactionHistory(ctx context.Context, request EVMTransactionHistoryRequest) ([]EVMTransactionHistoryItem, error) {
	chain, err := resolveEVMChain(request.ChainID)
	if err != nil {
		return nil, err
	}
	address := normalizeEVMAddress(request.Address)
	if !evmAddressPattern.MatchString(address) {
		return nil, fmt.Errorf("invalid wallet address")
	}
	baseURL := evmHistoryExplorerURLs[chain.ID]
	if baseURL == "" {
		return nil, fmt.Errorf("transaction history is not supported on %s", chain.ID)
	}

	nativeTransactions, err := fetchBlockscoutTransactions(ctx, baseURL, address, "txlist")
	if err != nil {
		return nil, fmt.Errorf("load native transaction history: %w", err)
	}
	tokenTransactions, err := fetchBlockscoutTransactions(ctx, baseURL, address, "tokentx")
	if err != nil {
		return nil, fmt.Errorf("load token transaction history: %w", err)
	}

	items := make([]EVMTransactionHistoryItem, 0, len(nativeTransactions)+len(tokenTransactions))
	for _, transaction := range nativeTransactions {
		value, ok := decimalBigInt(transaction.Value)
		if !ok || value.Sign() <= 0 {
			continue
		}
		items = append(items, historyItemFromBlockscout(chain, address, transaction, chain.NativeSymbol, 18, "native"))
	}
	for _, transaction := range tokenTransactions {
		decimals, parseErr := strconv.Atoi(transaction.TokenDecimal)
		if parseErr != nil || decimals < 0 || decimals > 255 {
			continue
		}
		value, ok := decimalBigInt(transaction.Value)
		if !ok || value.Sign() <= 0 {
			continue
		}
		symbol := strings.TrimSpace(transaction.TokenSymbol)
		if symbol == "" {
			symbol = "TOKEN"
		}
		items = append(items, historyItemFromBlockscout(chain, address, transaction, symbol, decimals, "token"))
	}

	sort.SliceStable(items, func(left, right int) bool { return items[left].Timestamp > items[right].Timestamp })
	if len(items) > 200 {
		items = items[:200]
	}
	return items, nil
}

func fetchBlockscoutTransactions(ctx context.Context, baseURL, address, action string) ([]blockscoutTransaction, error) {
	endpoint, err := url.Parse(strings.TrimRight(baseURL, "/") + "/api")
	if err != nil {
		return nil, err
	}
	query := endpoint.Query()
	query.Set("module", "account")
	query.Set("action", action)
	query.Set("address", address)
	query.Set("page", "1")
	query.Set("offset", "10")
	query.Set("sort", "desc")
	endpoint.RawQuery = query.Encode()

	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, err
	}
	response, err := walletHTTPClient.Do(httpRequest)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("explorer returned %s", response.Status)
	}
	var payload blockscoutAccountResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode explorer response: %w", err)
	}
	if payload.Status != "1" {
		message := strings.ToLower(payload.Message + " " + string(payload.Result))
		if isEmptyBlockscoutHistory(message) {
			return []blockscoutTransaction{}, nil
		}
		return nil, fmt.Errorf("explorer error: %s", strings.TrimSpace(payload.Message))
	}
	var transactions []blockscoutTransaction
	if err := json.Unmarshal(payload.Result, &transactions); err != nil {
		return nil, fmt.Errorf("decode transaction list: %w", err)
	}
	return transactions, nil
}

func isEmptyBlockscoutHistory(message string) bool {
	for _, marker := range []string{"no transaction", "no token transfer", "no record"} {
		if strings.Contains(message, marker) {
			return true
		}
	}
	return false
}

func historyItemFromBlockscout(chain evmChain, address string, transaction blockscoutTransaction, symbol string, decimals int, assetKind string) EVMTransactionHistoryItem {
	direction := "receive"
	if strings.EqualFold(transaction.From, address) {
		direction = "send"
	}
	value, _ := decimalBigInt(transaction.Value)
	fee := big.NewInt(0)
	gasUsed, gasUsedOK := decimalBigInt(transaction.GasUsed)
	gasPrice, gasPriceOK := decimalBigInt(transaction.GasPrice)
	if gasUsedOK && gasPriceOK {
		fee.Mul(gasUsed, gasPrice)
	}
	timestamp := time.Unix(0, 0).UTC()
	if seconds, err := strconv.ParseInt(transaction.TimeStamp, 10, 64); err == nil {
		timestamp = time.Unix(seconds, 0).UTC()
	}
	status := "confirmed"
	if transaction.IsError == "1" {
		status = "failed"
	}
	return EVMTransactionHistoryItem{
		ID:          transaction.Hash + ":" + assetKind + ":" + direction + ":" + transaction.TransactionIndex,
		Hash:        transaction.Hash,
		ChainID:     chain.ID,
		Type:        direction,
		Direction:   direction,
		Timestamp:   timestamp.Format(time.RFC3339),
		From:        transaction.From,
		To:          transaction.To,
		TokenSymbol: symbol,
		Amount:      formatExactUnits(value, decimals, decimals),
		Fee:         formatExactUnits(fee, 18, 8),
		FeeSymbol:   chain.NativeSymbol,
		Status:      status,
	}
}

func decimalBigInt(value string) (*big.Int, bool) {
	parsed, ok := new(big.Int).SetString(strings.TrimSpace(value), 10)
	return parsed, ok
}
