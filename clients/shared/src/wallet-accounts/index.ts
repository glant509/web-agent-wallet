export interface WalletAccount {
  index: number;
  name: string;
}

export interface BIP44Network {
  id: string;
  label: string;
  coinType: number;
  path: string;
  curve: "secp256k1" | "ed25519";
}

export interface BIP44Keyring {
  standard: "BIP44";
  account: number;
  selectedAccount: number;
  index: number;
  accounts: WalletAccount[];
  networks: BIP44Network[];
  [key: string]: unknown;
}

export function createNetworks(accountIndex: number): BIP44Network[] {
  const account = Number.isSafeInteger(accountIndex) && accountIndex >= 0 ? accountIndex : 0;
  const evmPath = `m/44'/60'/${account}'/0/0`;
  return [
    { id: "bitcoin", label: "Bitcoin", coinType: 0, path: `m/44'/0'/${account}'/0/0`, curve: "secp256k1" },
    { id: "ethereum", label: "Ethereum", coinType: 60, path: evmPath, curve: "secp256k1" },
    { id: "base", label: "Base", coinType: 60, path: evmPath, curve: "secp256k1" },
    { id: "arbitrum", label: "Arbitrum", coinType: 60, path: evmPath, curve: "secp256k1" },
    { id: "optimism", label: "Optimism", coinType: 60, path: evmPath, curve: "secp256k1" },
    { id: "bnb", label: "BNB Chain", coinType: 60, path: evmPath, curve: "secp256k1" },
    { id: "polygon", label: "Polygon", coinType: 60, path: evmPath, curve: "secp256k1" },
    { id: "avalanche", label: "Avalanche", coinType: 60, path: evmPath, curve: "secp256k1" },
    { id: "solana", label: "Solana", coinType: 501, path: `m/44'/501'/${account}'/0'`, curve: "ed25519" },
    { id: "sui", label: "Sui", coinType: 784, path: `m/44'/784'/${account}'/0'/0'`, curve: "ed25519" }
  ];
}

export function createDefaultKeyring(): BIP44Keyring {
  return {
    standard: "BIP44",
    account: 0,
    selectedAccount: 0,
    index: 0,
    accounts: [{ index: 0, name: "账户 0" }],
    networks: createNetworks(0)
  };
}

export function normalizeKeyring(input: unknown): BIP44Keyring {
  const candidate = input && typeof input === "object" ? input as Record<string, unknown> : {};
  const source = candidate.standard === "BIP44" ? candidate : {};
  const sourceAccounts = Array.isArray(source.accounts) ? source.accounts : [];
  const accounts: WalletAccount[] = [];
  const seen = new Set<number>();
  sourceAccounts.forEach((entry) => {
    const account = entry && typeof entry === "object" ? entry as Record<string, unknown> : {};
    const index = Number(account.index);
    if (!Number.isSafeInteger(index) || index < 0 || index > 99 || seen.has(index)) return;
    seen.add(index);
    const rawName = typeof account.name === "string" ? account.name.trim() : "";
    accounts.push({ index, name: (rawName || `账户 ${index}`).slice(0, 24) });
  });
  if (accounts.length === 0) {
    const legacyAccount = Number(source.account);
    const index = Number.isSafeInteger(legacyAccount) && legacyAccount >= 0 && legacyAccount <= 99 ? legacyAccount : 0;
    accounts.push({ index, name: `账户 ${index}` });
  }
  accounts.sort((left, right) => left.index - right.index);
  const requested = Number(source.selectedAccount ?? source.account);
  const selectedAccount = accounts.some((account) => account.index === requested) ? requested : accounts[0].index;
  return {
    ...source,
    standard: "BIP44",
    account: selectedAccount,
    selectedAccount,
    index: 0,
    accounts,
    networks: createNetworks(selectedAccount)
  } as BIP44Keyring;
}

export function accountPath(chainId: string, accountIndex: number): string {
  return createNetworks(accountIndex).find((network) => network.id === chainId)?.path || `BIP44 account ${accountIndex}`;
}
