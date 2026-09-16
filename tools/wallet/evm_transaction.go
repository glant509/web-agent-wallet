package wallet

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"regexp"
	"strings"
)

var evmAddressPattern = regexp.MustCompile(`^0x[0-9a-fA-F]{40}$`)

var expectedEVMChainIDs = map[string]int64{
	"ethereum":  1,
	"base":      8453,
	"arbitrum":  42161,
	"optimism":  10,
	"bsc":       56,
	"polygon":   137,
	"avalanche": 43114,
}

type EVMTransactionRequest struct {
	ChainID      string `json:"chain_id"`
	From         string `json:"from"`
	To           string `json:"to"`
	TokenAddress string `json:"token_address,omitempty"`
	Amount       string `json:"amount,omitempty"`
	SendMax      bool   `json:"send_max,omitempty"`
}

type EVMTransactionPlan struct {
	ChainID              string `json:"chain_id"`
	NetworkChainID       string `json:"network_chain_id"`
	Nonce                string `json:"nonce"`
	TransactionType      int    `json:"transaction_type"`
	To                   string `json:"to"`
	Value                string `json:"value"`
	Data                 string `json:"data"`
	GasLimit             string `json:"gas_limit"`
	GasPrice             string `json:"gas_price,omitempty"`
	MaxFeePerGas         string `json:"max_fee_per_gas,omitempty"`
	MaxPriorityFeePerGas string `json:"max_priority_fee_per_gas,omitempty"`
	EstimatedFeeRaw      string `json:"estimated_fee_raw"`
	EstimatedFee         string `json:"estimated_fee"`
	NativeSymbol         string `json:"native_symbol"`
	TokenSymbol          string `json:"token_symbol"`
	TokenDecimals        int    `json:"token_decimals"`
	AmountRaw            string `json:"amount_raw"`
	Amount               string `json:"amount"`
	IsNative             bool   `json:"is_native"`
}

func PrepareEVMTransaction(ctx context.Context, request EVMTransactionRequest) (EVMTransactionPlan, error) {
	chain, err := resolveEVMChain(request.ChainID)
	if err != nil {
		return EVMTransactionPlan{}, err
	}
	from := normalizeEVMAddress(request.From)
	to := normalizeEVMAddress(request.To)
	if !evmAddressPattern.MatchString(from) {
		return EVMTransactionPlan{}, fmt.Errorf("invalid sender address")
	}
	if !evmAddressPattern.MatchString(to) {
		return EVMTransactionPlan{}, fmt.Errorf("invalid recipient address")
	}
	rpcURL, err := resolveChainRPCURL(chain.ID)
	if err != nil {
		return EVMTransactionPlan{}, err
	}

	networkChainID, err := evmRPCString(ctx, rpcURL, "eth_chainId", []any{})
	if err != nil {
		return EVMTransactionPlan{}, fmt.Errorf("load chain id: %w", err)
	}
	parsedChainID, err := hexToBigInt(networkChainID)
	if err != nil {
		return EVMTransactionPlan{}, fmt.Errorf("decode chain id: %w", err)
	}
	if expected, ok := expectedEVMChainIDs[chain.ID]; !ok || parsedChainID.Cmp(big.NewInt(expected)) != 0 {
		return EVMTransactionPlan{}, fmt.Errorf("configured rpc chain id does not match %s", chain.ID)
	}
	nonce, err := evmRPCString(ctx, rpcURL, "eth_getTransactionCount", []any{from, "pending"})
	if err != nil {
		return EVMTransactionPlan{}, fmt.Errorf("load account nonce: %w", err)
	}
	nativeBalanceRaw, err := evmRPCString(ctx, rpcURL, "eth_getBalance", []any{from, "pending"})
	if err != nil {
		return EVMTransactionPlan{}, fmt.Errorf("load native balance: %w", err)
	}
	nativeBalance, err := hexToBigInt(nativeBalanceRaw)
	if err != nil {
		return EVMTransactionPlan{}, err
	}

	isNative := strings.TrimSpace(request.TokenAddress) == ""
	tokenSymbol := chain.NativeSymbol
	tokenDecimals := 18
	txTo := to
	txValue := big.NewInt(0)
	txData := "0x"
	var amount *big.Int

	if isNative {
		if request.SendMax {
			amount = big.NewInt(0)
		} else {
			amount, err = parseDecimalUnits(request.Amount, tokenDecimals)
			if err != nil {
				return EVMTransactionPlan{}, err
			}
			txValue.Set(amount)
		}
	} else {
		if request.SendMax {
			return EVMTransactionPlan{}, fmt.Errorf("send_max is only supported for native assets")
		}
		token, ok := trackedEVMToken(chain, request.TokenAddress)
		if !ok {
			return EVMTransactionPlan{}, fmt.Errorf("token is not supported on %s", chain.ID)
		}
		tokenSymbol = token.Symbol
		tokenDecimals = token.Decimals
		amount, err = parseDecimalUnits(request.Amount, tokenDecimals)
		if err != nil {
			return EVMTransactionPlan{}, err
		}
		txTo = token.Address
		txData = "0xa9059cbb" + leftPadHex(strings.TrimPrefix(strings.ToLower(to), "0x"), 64) + leftPadHex(amount.Text(16), 64)
		tokenBalanceRaw, balanceErr := fetchERC20Balance(ctx, rpcURL, token.Address, from)
		if balanceErr != nil {
			return EVMTransactionPlan{}, fmt.Errorf("load %s balance: %w", token.Symbol, balanceErr)
		}
		tokenBalance, parseErr := hexToBigInt(tokenBalanceRaw)
		if parseErr != nil {
			return EVMTransactionPlan{}, parseErr
		}
		if tokenBalance.Cmp(amount) < 0 {
			return EVMTransactionPlan{}, fmt.Errorf("insufficient %s balance", token.Symbol)
		}
	}

	call := map[string]string{
		"from":  from,
		"to":    txTo,
		"value": hexQuantity(txValue),
		"data":  txData,
	}
	estimatedGasRaw, err := evmRPCString(ctx, rpcURL, "eth_estimateGas", []any{call})
	if err != nil {
		return EVMTransactionPlan{}, fmt.Errorf("estimate gas: %w", err)
	}
	estimatedGas, err := hexToBigInt(estimatedGasRaw)
	if err != nil {
		return EVMTransactionPlan{}, err
	}
	gasLimit := new(big.Int).Add(new(big.Int).Mul(estimatedGas, big.NewInt(115)), big.NewInt(99))
	gasLimit.Div(gasLimit, big.NewInt(100))

	gasPriceRaw, err := evmRPCString(ctx, rpcURL, "eth_gasPrice", []any{})
	if err != nil {
		return EVMTransactionPlan{}, fmt.Errorf("load gas price: %w", err)
	}
	gasPrice, err := hexToBigInt(gasPriceRaw)
	if err != nil {
		return EVMTransactionPlan{}, err
	}
	transactionType := 0
	maxFeePerGas := new(big.Int).Set(gasPrice)
	maxPriorityFeePerGas := big.NewInt(0)
	baseFee := latestBaseFee(ctx, rpcURL)
	if baseFee.Sign() > 0 {
		transactionType = 2
		priorityRaw, priorityErr := evmRPCString(ctx, rpcURL, "eth_maxPriorityFeePerGas", []any{})
		if priorityErr == nil {
			if parsed, parseErr := hexToBigInt(priorityRaw); parseErr == nil {
				maxPriorityFeePerGas = parsed
			}
		}
		if maxPriorityFeePerGas.Sign() <= 0 {
			maxPriorityFeePerGas = new(big.Int).Sub(gasPrice, baseFee)
			if maxPriorityFeePerGas.Sign() <= 0 {
				maxPriorityFeePerGas = big.NewInt(1_000_000_000)
			}
		}
		maxFeePerGas = new(big.Int).Add(new(big.Int).Mul(baseFee, big.NewInt(2)), maxPriorityFeePerGas)
	}
	estimatedFee := new(big.Int).Mul(gasLimit, maxFeePerGas)

	if isNative && request.SendMax {
		amount = new(big.Int).Sub(nativeBalance, estimatedFee)
		if amount.Sign() <= 0 {
			return EVMTransactionPlan{}, fmt.Errorf("native balance is not enough to pay the network fee")
		}
		txValue.Set(amount)
	} else if isNative {
		required := new(big.Int).Add(amount, estimatedFee)
		if nativeBalance.Cmp(required) < 0 {
			return EVMTransactionPlan{}, fmt.Errorf("native balance is not enough for amount plus network fee")
		}
	} else if nativeBalance.Cmp(estimatedFee) < 0 {
		return EVMTransactionPlan{}, fmt.Errorf("%s balance is not enough to pay the network fee", chain.NativeSymbol)
	}

	plan := EVMTransactionPlan{
		ChainID:         chain.ID,
		NetworkChainID:  networkChainID,
		Nonce:           nonce,
		TransactionType: transactionType,
		To:              txTo,
		Value:           hexQuantity(txValue),
		Data:            txData,
		GasLimit:        hexQuantity(gasLimit),
		EstimatedFeeRaw: estimatedFee.String(),
		EstimatedFee:    formatExactUnits(estimatedFee, 18, 8),
		NativeSymbol:    chain.NativeSymbol,
		TokenSymbol:     tokenSymbol,
		TokenDecimals:   tokenDecimals,
		AmountRaw:       amount.String(),
		Amount:          formatExactUnits(amount, tokenDecimals, tokenDecimals),
		IsNative:        isNative,
	}
	if transactionType == 2 {
		plan.MaxFeePerGas = hexQuantity(maxFeePerGas)
		plan.MaxPriorityFeePerGas = hexQuantity(maxPriorityFeePerGas)
	} else {
		plan.GasPrice = hexQuantity(gasPrice)
	}
	return plan, nil
}

func BroadcastEVMTransaction(ctx context.Context, chainID, rawTransaction string) (string, error) {
	chain, err := resolveEVMChain(chainID)
	if err != nil {
		return "", err
	}
	raw := strings.TrimSpace(rawTransaction)
	if !strings.HasPrefix(raw, "0x") || len(raw) < 4 || len(raw) > 512*1024 {
		return "", fmt.Errorf("invalid signed transaction")
	}
	rpcURL, err := resolveChainRPCURL(chain.ID)
	if err != nil {
		return "", err
	}
	hash, err := evmRPCString(ctx, rpcURL, "eth_sendRawTransaction", []any{raw})
	if err != nil {
		return "", fmt.Errorf("broadcast transaction: %w", err)
	}
	if !regexp.MustCompile(`^0x[0-9a-fA-F]{64}$`).MatchString(hash) {
		return "", fmt.Errorf("rpc returned an invalid transaction hash")
	}
	return hash, nil
}

func trackedEVMToken(chain evmChain, address string) (trackedToken, bool) {
	normalized := strings.ToLower(normalizeEVMAddress(address))
	for _, token := range chain.Tokens {
		if strings.ToLower(token.Address) == normalized {
			return token, true
		}
	}
	return trackedToken{}, false
}

func parseDecimalUnits(value string, decimals int) (*big.Int, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || strings.HasPrefix(trimmed, "-") || strings.HasPrefix(trimmed, "+") {
		return nil, fmt.Errorf("amount must be greater than zero")
	}
	parts := strings.Split(trimmed, ".")
	if len(parts) > 2 || parts[0] == "" {
		return nil, fmt.Errorf("invalid amount")
	}
	if decimals < 0 || decimals > 255 {
		return nil, fmt.Errorf("invalid token decimals")
	}
	whole := parts[0]
	fraction := ""
	if len(parts) == 2 {
		fraction = parts[1]
	}
	if !regexp.MustCompile(`^[0-9]+$`).MatchString(whole) || (fraction != "" && !regexp.MustCompile(`^[0-9]+$`).MatchString(fraction)) {
		return nil, fmt.Errorf("invalid amount")
	}
	if len(fraction) > decimals {
		return nil, fmt.Errorf("amount has more than %d decimal places", decimals)
	}
	normalized := strings.TrimLeft(whole+fraction+strings.Repeat("0", decimals-len(fraction)), "0")
	if normalized == "" {
		return nil, fmt.Errorf("amount must be greater than zero")
	}
	amount, ok := new(big.Int).SetString(normalized, 10)
	if !ok || amount.Sign() <= 0 {
		return nil, fmt.Errorf("invalid amount")
	}
	return amount, nil
}

func formatExactUnits(value *big.Int, decimals, maxFraction int) string {
	if value == nil {
		return "0"
	}
	digits := value.String()
	if decimals == 0 {
		return digits
	}
	if len(digits) <= decimals {
		digits = strings.Repeat("0", decimals-len(digits)+1) + digits
	}
	point := len(digits) - decimals
	fraction := strings.TrimRight(digits[point:], "0")
	if maxFraction >= 0 && len(fraction) > maxFraction {
		fraction = strings.TrimRight(fraction[:maxFraction], "0")
	}
	if fraction == "" {
		return digits[:point]
	}
	return digits[:point] + "." + fraction
}

func latestBaseFee(ctx context.Context, rpcURL string) *big.Int {
	result, err := evmRPCJSON(ctx, rpcURL, "eth_getBlockByNumber", []any{"latest", false})
	if err != nil {
		return big.NewInt(0)
	}
	var block struct {
		BaseFeePerGas string `json:"baseFeePerGas"`
	}
	if json.Unmarshal(result, &block) != nil {
		return big.NewInt(0)
	}
	baseFee, err := hexToBigInt(block.BaseFeePerGas)
	if err != nil {
		return big.NewInt(0)
	}
	return baseFee
}

func evmRPCString(ctx context.Context, rpcURL, method string, params []any) (string, error) {
	result, err := evmRPCJSON(ctx, rpcURL, method, params)
	if err != nil {
		return "", err
	}
	var value string
	if err := json.Unmarshal(result, &value); err != nil {
		return "", fmt.Errorf("decode %s result: %w", method, err)
	}
	return value, nil
}

func evmRPCJSON(ctx context.Context, rpcURL, method string, params []any) (json.RawMessage, error) {
	payload, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": method, "params": params})
	if err != nil {
		return nil, fmt.Errorf("marshal rpc payload: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, rpcURL, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("build rpc request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := walletHTTPClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("request rpc: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 8*1024))
		return nil, fmt.Errorf("rpc returned %s: %s", response.Status, strings.TrimSpace(string(body)))
	}
	var rpcResponse struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(response.Body).Decode(&rpcResponse); err != nil {
		return nil, fmt.Errorf("decode rpc response: %w", err)
	}
	if rpcResponse.Error != nil {
		return nil, fmt.Errorf("rpc error: %s", rpcResponse.Error.Message)
	}
	if len(rpcResponse.Result) == 0 || bytes.Equal(rpcResponse.Result, []byte("null")) {
		return nil, fmt.Errorf("rpc returned no result for %s", method)
	}
	return rpcResponse.Result, nil
}

func hexQuantity(value *big.Int) string {
	if value == nil || value.Sign() == 0 {
		return "0x0"
	}
	return "0x" + value.Text(16)
}
