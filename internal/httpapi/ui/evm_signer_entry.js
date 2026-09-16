import { HDNodeWallet } from "ethers";

function walletForAccount(seed, accountIndex = 0) {
  if (!(seed instanceof Uint8Array)) {
    throw new Error("Wallet seed is unavailable");
  }
  if (!Number.isSafeInteger(accountIndex) || accountIndex < 0 || accountIndex > 2147483647) {
    throw new Error("Invalid BIP44 account index");
  }
  return HDNodeWallet.fromSeed(seed).derivePath(`m/44'/60'/${accountIndex}'/0/0`);
}

function privateKeyPath(chainId, accountIndex) {
  const evmChains = ["ethereum", "base", "arbitrum", "optimism", "bnb", "polygon", "avalanche"];
  if (evmChains.includes(chainId)) {
    return `m/44'/60'/${accountIndex}'/0/0`;
  }
  if (chainId === "bitcoin") {
    return `m/44'/0'/${accountIndex}'/0/0`;
  }
  if (chainId === "solana") {
    return `m/44'/501'/${accountIndex}'/0'`;
  }
  if (chainId === "sui") {
    return `m/44'/784'/${accountIndex}'/0'/0'`;
  }
  throw new Error(`Unsupported chain: ${chainId}`);
}

function bytesToHex(bytes) {
  return Array.from(bytes, (value) => value.toString(16).padStart(2, "0")).join("");
}

async function hmacSHA512(key, data) {
  const cryptoKey = await crypto.subtle.importKey(
    "raw",
    key,
    { name: "HMAC", hash: "SHA-512" },
    false,
    ["sign"]
  );
  return new Uint8Array(await crypto.subtle.sign("HMAC", cryptoKey, data));
}

async function deriveEd25519PrivateKey(seed, path) {
  let digest = await hmacSHA512(new TextEncoder().encode("ed25519 seed"), seed);
  let key = digest.slice(0, 32);
  let chainCode = digest.slice(32);
  const components = path.split("/").slice(1);
  for (const component of components) {
    if (!/^\d+'$/.test(component)) {
      throw new Error("Ed25519 derivation requires a hardened path");
    }
    const index = Number(component.slice(0, -1)) + 0x80000000;
    const data = new Uint8Array(37);
    data[0] = 0;
    data.set(key, 1);
    new DataView(data.buffer).setUint32(33, index, false);
    digest = await hmacSHA512(chainCode, data);
    key.fill(0);
    chainCode.fill(0);
    key = digest.slice(0, 32);
    chainCode = digest.slice(32);
  }
  digest.fill(0);
  chainCode.fill(0);
  return key;
}

async function derivePrivateKey(seed, chainId, accountIndex = 0) {
  if (!(seed instanceof Uint8Array)) {
    throw new Error("Wallet seed is unavailable");
  }
  if (!Number.isSafeInteger(accountIndex) || accountIndex < 0 || accountIndex > 2147483647) {
    throw new Error("Invalid BIP44 account index");
  }
  const path = privateKeyPath(chainId, accountIndex);
  if (chainId === "solana" || chainId === "sui") {
    const privateKey = await deriveEd25519PrivateKey(seed, path);
    const encoded = "0x" + bytesToHex(privateKey);
    privateKey.fill(0);
    return { privateKey: encoded, path, format: "Ed25519 seed · hex" };
  }
  const wallet = HDNodeWallet.fromSeed(seed).derivePath(path);
  return {
    privateKey: wallet.privateKey,
    path,
    format: chainId === "bitcoin" ? "secp256k1 private key · hex" : "EVM private key · hex"
  };
}

async function signTransaction(seed, accountIndex, transaction) {
  const wallet = walletForAccount(seed, accountIndex);
  return wallet.signTransaction(transaction);
}

window.WalletEVMSigner = {
  getAddress(seed, accountIndex = 0) {
    return walletForAccount(seed, accountIndex).address;
  },
  derivePrivateKey,
  signTransaction
};
