package wallet

import (
	"bytes"
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
	"ethereum":  "https://eth.blockscout.com",
	"base":      "https://base.blockscout.com",
	"arbitrum":  "https://arbitrum.blockscout.com",
	"optimism":  "https://optimism.blockscout.com",
	"polygon":   "https://polygon.blockscout.com",
	"avalanche": "https://api.routescan.io/v2/network/mainnet/evm/43114/etherscan",
}

var bnbHistoryAPIURL = "https://bnbscan.com/api/v1/query"

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

type bnbscanTransaction struct {
	Hash         string      `json:"hash"`
	TxHash       string      `json:"txHash"`
	From         string      `json:"fromAddress"`
	To           string      `json:"toAddress"`
	Value        string      `json:"value"`
	GasUsed      string      `json:"gasUsed"`
	GasPrice     string      `json:"gasPrice"`
	Timestamp    string      `json:"timestamp"`
	TokenAddress string      `json:"tokenAddress"`
	Status       bool        `json:"status"`
	TxIndex      json.Number `json:"txIndex"`
	LogIndex     json.Number `json:"logIndex"`
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
	if chain.ID == "bsc" {
		return fetchBNBTransactionHistory(ctx, chain, address)
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

func fetchBNBTransactionHistory(ctx context.Context, chain evmChain, address string) ([]EVMTransactionHistoryItem, error) {
	nativeTransactions, err := fetchBNBScanTransactions(ctx, address, "transactions")
	if err != nil {
		return nil, fmt.Errorf("load native transaction history: %w", err)
	}
	tokenTransfers, err := fetchBNBScanTransactions(ctx, address, "token_transfers")
	if err != nil {
		return nil, fmt.Errorf("load token transaction history: %w", err)
	}

	items := make([]EVMTransactionHistoryItem, 0, len(nativeTransactions)+len(tokenTransfers))
	for _, transaction := range nativeTransactions {
		value, ok := decimalUnitsToInteger(transaction.Value, 18)
		if !ok || value.Sign() <= 0 {
			continue
		}
		items = append(items, bnbHistoryItem(chain, address, transaction, value, chain.NativeSymbol, 18, "native"))
	}
	tokenMetadata := make(map[string]trackedToken, len(chain.Tokens))
	for _, token := range chain.Tokens {
		tokenMetadata[strings.ToLower(token.Address)] = token
	}
	var rpcURL string
	for _, transaction := range tokenTransfers {
		value, ok := decimalBigInt(transaction.Value)
		if !ok || value.Sign() <= 0 || !evmAddressPattern.MatchString(transaction.TokenAddress) {
			continue
		}
		token, found := tokenMetadata[strings.ToLower(transaction.TokenAddress)]
		if !found {
			if rpcURL == "" {
				rpcURL, err = resolveChainRPCURL(chain.ID)
				if err != nil {
					return nil, fmt.Errorf("resolve token metadata RPC: %w", err)
				}
			}
			decimals, decimalsErr := fetchERC20Decimals(ctx, rpcURL, transaction.TokenAddress)
			symbol, symbolErr := fetchERC20Symbol(ctx, rpcURL, transaction.TokenAddress)
			if decimalsErr != nil || symbolErr != nil || decimals < 0 || decimals > 255 || symbol == "" {
				return nil, fmt.Errorf("load token metadata for %s", transaction.TokenAddress)
			}
			token = trackedToken{Address: transaction.TokenAddress, Symbol: symbol, Decimals: decimals}
			tokenMetadata[strings.ToLower(transaction.TokenAddress)] = token
		}
		items = append(items, bnbHistoryItem(chain, address, transaction, value, token.Symbol, token.Decimals, "token"))
	}
	sort.SliceStable(items, func(left, right int) bool { return items[left].Timestamp > items[right].Timestamp })
	if len(items) > 200 {
		items = items[:200]
	}
	return items, nil
}

func fetchBNBScanTransactions(ctx context.Context, address, entity string) ([]bnbscanTransaction, error) {
	payload, err := json.Marshal(map[string]any{
		"entity":  entity,
		"filter":  map[string]string{"address": address},
		"limit":   10,
		"orderBy": "desc",
	})
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, bnbHistoryAPIURL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := walletHTTPClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("explorer returned %s", response.Status)
	}
	var result struct {
		Entity string               `json:"entity"`
		Data   []bnbscanTransaction `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode explorer response: %w", err)
	}
	if result.Entity != entity || result.Data == nil {
		return nil, fmt.Errorf("explorer returned an invalid %s response", entity)
	}
	return result.Data, nil
}

func decimalUnitsToInteger(value string, decimals int) (*big.Int, bool) {
	amount, ok := new(big.Rat).SetString(strings.TrimSpace(value))
	if !ok || amount.Sign() < 0 {
		return nil, false
	}
	amount.Mul(amount, new(big.Rat).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)))
	if !amount.IsInt() {
		return nil, false
	}
	return amount.Num(), true
}

func bnbHistoryItem(chain evmChain, address string, transaction bnbscanTransaction, value *big.Int, symbol string, decimals int, assetKind string) EVMTransactionHistoryItem {
	hash := transaction.Hash
	index := transaction.TxIndex.String()
	if assetKind == "token" {
		hash = transaction.TxHash
		index = transaction.LogIndex.String()
	}
	direction := "receive"
	if strings.EqualFold(transaction.From, address) {
		direction = "send"
	}
	timestamp, err := time.Parse(time.RFC3339Nano, transaction.Timestamp)
	if err != nil {
		timestamp = time.Unix(0, 0).UTC()
	}
	fee := big.NewInt(0)
	gasUsed, gasUsedOK := decimalBigInt(transaction.GasUsed)
	gasPrice, gasPriceOK := decimalBigInt(transaction.GasPrice)
	if gasUsedOK && gasPriceOK {
		fee.Mul(gasUsed, gasPrice)
	}
	status := "confirmed"
	if assetKind == "native" && !transaction.Status {
		status = "failed"
	}
	return EVMTransactionHistoryItem{
		ID:   hash + ":" + assetKind + ":" + direction + ":" + index,
		Hash: hash, ChainID: chain.ID, Type: direction, Direction: direction,
		Timestamp: timestamp.UTC().Format(time.RFC3339), From: transaction.From, To: transaction.To,
		TokenSymbol: symbol, Amount: formatExactUnits(value, decimals, decimals),
		Fee: formatExactUnits(fee, 18, 8), FeeSymbol: chain.NativeSymbol, Status: status,
	}
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
