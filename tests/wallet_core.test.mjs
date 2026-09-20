import assert from "node:assert/strict";
import test from "node:test";

globalThis.window = globalThis;
await import("../dist/web/shared.js");

test("wallet core creates and unlocks an AES-GCM vault", async () => {
  const core = globalThis.AgentWalletCore;
  const password = "TestPass1!";
  const mnemonic = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about";
  const keyring = { standard: "BIP44", selectedAccount: 0 };
  const created = await core.createVault(password, mnemonic, keyring, 10_000);

  assert.equal(created.vault.cipher, "AES-256-GCM");
  assert.equal(created.vault.kdf, "PBKDF2-SHA-256");
  assert.equal(created.vault.kdfParams.iterations, 10_000);
  assert.notEqual(created.vault.ciphertext, mnemonic);

  const unlocked = await core.unlockVault(created.vault, password);
  assert.equal(unlocked.payload.mnemonic, mnemonic);
  assert.deepEqual(unlocked.payload.keyring, keyring);
  await assert.rejects(() => core.unlockVault(created.vault, "WrongPass1!"));
});

test("shared validation and UI formatting remain deterministic", () => {
  assert.equal(globalThis.AgentWalletCore.isValidPassword("Valid123!"), true);
  assert.equal(globalThis.AgentWalletCore.isValidPassword("short"), false);
  assert.equal(
    globalThis.AgentWalletUI.maskAddress("0x1234567890abcdef1234567890abcdef1234abcd"),
    "0x1234***abcd"
  );
});

test("BIP44 account domain normalizes legacy and duplicate accounts", () => {
  const accounts = globalThis.AgentWalletAccounts;
  const normalized = accounts.normalizeKeyring({
    standard: "BIP44",
    account: 2,
    accounts: [
      { index: 2, name: " Trading " },
      { index: 2, name: "Duplicate" },
      { index: 0, name: "" },
      { index: 101, name: "Invalid" }
    ]
  });
  assert.deepEqual(normalized.accounts, [
    { index: 0, name: "账户 0" },
    { index: 2, name: "Trading" }
  ]);
  assert.equal(normalized.selectedAccount, 2);
  assert.equal(accounts.accountPath("ethereum", 2), "m/44'/60'/2'/0/0");
  assert.equal(accounts.accountPath("solana", 2), "m/44'/501'/2'/0'");
});

test("history and transaction domains normalize, merge, filter and build signing input", () => {
  const history = globalThis.AgentWalletHistory;
  const remote = history.normalizeItem({
    hash: "0xabc",
    chain_id: "ethereum",
    direction: "receive",
    type: "receive",
    token_symbol: "ETH",
    amount: "1",
    timestamp: "2026-09-19T10:00:00Z"
  });
  const duplicate = { ...remote, id: "local-copy" };
  const older = { ...remote, hash: "0xdef", amount: "2", timestamp: "2026-09-18T10:00:00Z" };
  assert.equal(history.merge([remote], [duplicate, older]).length, 2);
  assert.equal(history.filter([older, remote], { chainId: "ethereum", type: "receive" })[0].hash, "0xabc");
  assert.equal(history.typeLabel("send"), "发送");

  const transactions = globalThis.AgentWalletTransactions;
  assert.equal(transactions.isValidRecipient("ethereum", "0x" + "12".repeat(20)), true);
  assert.equal(transactions.isValidAmount("0.0001"), true);
  assert.equal(transactions.isValidAmount("0"), false);
  assert.deepEqual(transactions.forSigning({
    transaction_type: 2,
    network_chain_id: "1",
    nonce: "3",
    to: "0x" + "12".repeat(20),
    value: "1",
    data: "0x",
    gas_limit: "21000",
    max_fee_per_gas: "10",
    max_priority_fee_per_gas: "2"
  }), {
    type: 2,
    chainId: "1",
    nonce: "3",
    to: "0x" + "12".repeat(20),
    value: "1",
    data: "0x",
    gasLimit: "21000",
    maxFeePerGas: "10",
    maxPriorityFeePerGas: "2"
  });
});

test("market domain parses candle observations and calculates change", () => {
  const market = globalThis.AgentWalletMarket;
  const chart = market.parseObservation([
    "BTC chart:",
    "- vs_currency: usd",
    "- interval: 15m",
    "Candles:",
    "- 2026-09-19T10:00:00Z open=100 high=112 low=98 close=110",
    "- 2026-09-19T10:15:00Z open=110 high=120 low=108 close=115"
  ].join("\n"));
  assert.equal(chart.title, "BTC chart");
  assert.equal(chart.candles.length, 2);
  assert.deepEqual(market.calculateChange(chart.candles), {
    className: "positive",
    label: "+15.00%",
    percent: 15
  });
  assert.deepEqual(market.stats(chart.candles), [
    { label: "H", value: "120.0000" },
    { label: "L", value: "98.0000" },
    { label: "O", value: "100.0000" },
    { label: "C", value: "115.0000" }
  ]);
});
