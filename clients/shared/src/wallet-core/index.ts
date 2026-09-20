export interface WalletVault {
  id: string;
  version: number;
  cipher: "AES-256-GCM";
  kdf: "PBKDF2-SHA-256";
  kdfParams: { iterations: number; hash: "SHA-256" };
  salt: string;
  iv: string;
  ciphertext: string;
  createdAt: string;
}

export interface WalletPayload<Keyring = unknown> {
  mnemonic: string;
  keyring: Keyring;
  createdAt: string;
}

export function isValidPassword(password: string): boolean {
  return /^[A-Za-z0-9!@#$%^&*.]{8,16}$/.test(password);
}

export function randomBytes(length: number): Uint8Array<ArrayBuffer> {
  return crypto.getRandomValues(new Uint8Array(length));
}

export function randomHex(byteLength: number): string {
  return Array.from(randomBytes(byteLength), (value) => value.toString(16).padStart(2, "0")).join("");
}

export function bytesToBase64(bytes: Uint8Array): string {
  let binary = "";
  bytes.forEach((value) => { binary += String.fromCharCode(value); });
  return btoa(binary);
}

export function base64ToBytes(value: string): Uint8Array<ArrayBuffer> {
  const binary = atob(value);
  const bytes = new Uint8Array(binary.length);
  for (let index = 0; index < binary.length; index += 1) {
    bytes[index] = binary.charCodeAt(index);
  }
  return bytes;
}

export async function deriveVaultKey(password: string, salt: Uint8Array<ArrayBuffer>, iterations: number): Promise<CryptoKey> {
  const passwordBytes = new TextEncoder().encode(password);
  try {
    const material = await crypto.subtle.importKey("raw", passwordBytes, "PBKDF2", false, ["deriveKey"]);
    return await crypto.subtle.deriveKey(
      { name: "PBKDF2", hash: "SHA-256", salt, iterations },
      material,
      { name: "AES-GCM", length: 256 },
      false,
      ["encrypt", "decrypt"]
    );
  } finally {
    passwordBytes.fill(0);
  }
}

export async function deriveBIP39Seed(mnemonic: string, passphrase = ""): Promise<Uint8Array<ArrayBuffer>> {
  const mnemonicBytes = new TextEncoder().encode(mnemonic.normalize("NFKD"));
  const salt = new TextEncoder().encode(("mnemonic" + passphrase).normalize("NFKD"));
  try {
    const material = await crypto.subtle.importKey("raw", mnemonicBytes, "PBKDF2", false, ["deriveBits"]);
    const bits = await crypto.subtle.deriveBits(
      { name: "PBKDF2", hash: "SHA-512", salt, iterations: 2048 },
      material,
      512
    );
    return new Uint8Array(bits);
  } finally {
    mnemonicBytes.fill(0);
    salt.fill(0);
  }
}

export async function createVault<Keyring>(
  password: string,
  mnemonic: string,
  keyring: Keyring,
  iterations = 600_000
): Promise<{ vault: WalletVault; masterKey: CryptoKey }> {
  const salt = randomBytes(16);
  const iv = randomBytes(12);
  const masterKey = await deriveVaultKey(password, salt, iterations);
  const plaintext = new TextEncoder().encode(JSON.stringify({ mnemonic, keyring, createdAt: new Date().toISOString() }));
  try {
    const encrypted = await crypto.subtle.encrypt({ name: "AES-GCM", iv }, masterKey, plaintext);
    const createdAt = new Date().toISOString();
    return {
      masterKey,
      vault: {
        id: "primary",
        version: 1,
        cipher: "AES-256-GCM",
        kdf: "PBKDF2-SHA-256",
        kdfParams: { iterations, hash: "SHA-256" },
        salt: bytesToBase64(salt),
        iv: bytesToBase64(iv),
        ciphertext: bytesToBase64(new Uint8Array(encrypted)),
        createdAt
      }
    };
  } finally {
    plaintext.fill(0);
    salt.fill(0);
    iv.fill(0);
  }
}

export async function unlockVault<Keyring>(
  vault: WalletVault,
  password: string
): Promise<{ masterKey: CryptoKey; payload: WalletPayload<Keyring> }> {
  const salt = base64ToBytes(vault.salt);
  const iv = base64ToBytes(vault.iv);
  const ciphertext = base64ToBytes(vault.ciphertext);
  try {
    const masterKey = await deriveVaultKey(password, salt, vault.kdfParams.iterations);
    const decrypted = await crypto.subtle.decrypt({ name: "AES-GCM", iv }, masterKey, ciphertext);
    const plaintext = new Uint8Array(decrypted);
    try {
      const payload = JSON.parse(new TextDecoder().decode(plaintext)) as WalletPayload<Keyring>;
      if (!payload || typeof payload.mnemonic !== "string") {
        throw new Error("Invalid wallet vault payload");
      }
      return { masterKey, payload };
    } finally {
      plaintext.fill(0);
    }
  } finally {
    salt.fill(0);
    iv.fill(0);
    ciphertext.fill(0);
  }
}
