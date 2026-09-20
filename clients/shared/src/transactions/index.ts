export interface EVMTransactionPlan {
  transaction_type: number;
  network_chain_id: string | number;
  nonce: string | number;
  to: string;
  value: string;
  data: string;
  gas_limit: string;
  gas_price?: string;
  max_fee_per_gas?: string;
  max_priority_fee_per_gas?: string;
  [key: string]: unknown;
}

export function isValidRecipient(chainId: string, address: unknown): boolean {
  const value = String(address || "").trim();
  if (!value) return false;
  if (["ethereum", "base", "arbitrum", "optimism", "bnb", "polygon", "avalanche"].includes(chainId)) {
    return /^0x[0-9a-fA-F]{40}$/.test(value);
  }
  if (chainId === "bitcoin") return /^(bc1[ac-hj-np-z02-9]{11,71}|[13][a-km-zA-HJ-NP-Z1-9]{25,34})$/.test(value);
  if (chainId === "solana") return /^[1-9A-HJ-NP-Za-km-z]{32,44}$/.test(value);
  if (chainId === "sui") return /^0x[0-9a-fA-F]{64}$/.test(value);
  return value.length >= 20;
}

export function isValidAmount(amount: unknown): boolean {
  const value = String(amount || "").trim();
  return /^\d+(\.\d+)?$/.test(value) && Number(value) > 0;
}

export function forSigning(plan: EVMTransactionPlan): Record<string, unknown> {
  const transaction: Record<string, unknown> = {
    type: plan.transaction_type,
    chainId: plan.network_chain_id,
    nonce: plan.nonce,
    to: plan.to,
    value: plan.value,
    data: plan.data,
    gasLimit: plan.gas_limit
  };
  if (plan.transaction_type === 2) {
    transaction.maxFeePerGas = plan.max_fee_per_gas;
    transaction.maxPriorityFeePerGas = plan.max_priority_fee_per_gas;
  } else {
    transaction.gasPrice = plan.gas_price;
  }
  return transaction;
}
