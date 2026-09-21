import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import path from "node:path";
import test from "node:test";

const root = process.cwd();

test("platform runtime resolves packaged and browser environments", async () => {
  const source = await readFile(path.join(root, "internal/httpapi/ui/platform_runtime.js"), "utf8");
  assert.match(source, /chrome-extension/);
  assert.match(source, /10\.0\.2\.2:8081/);
  assert.match(source, /global\.location\.protocol === "file:"/);
  assert.match(source, /SecureWalletVault/);
  assert.match(source, /chrome\.storage\.local/);
});

test("native biometric adapters are wired for iOS and Android", async () => {
  const ios = await readFile(path.join(root, "clients/mobile/ios/App/App/AppDelegate.swift"), "utf8");
  const iosScene = await readFile(path.join(root, "clients/mobile/ios/App/App/SceneDelegate.swift"), "utf8");
  const android = await readFile(path.join(root, "clients/mobile/android/app/src/main/java/com/agentwallet/app/WalletBiometricsPlugin.java"), "utf8");
  assert.match(ios, /LocalAuthentication/);
  assert.match(ios, /biometryCurrentSet/);
  assert.match(android, /BiometricPrompt/);
  assert.match(android, /AUTH_BIOMETRIC_STRONG/);
  assert.match(android, /setInvalidatedByBiometricEnrollment\(true\)/);
  assert.match(iosScene, /AgentWalletBridgeViewController/);
  assert.match(iosScene, /sceneWillResignActive/);
  assert.match(iosScene, /privacyCover/);
});

test("extension is Manifest V3 without DApp injection", async () => {
  const manifest = JSON.parse(await readFile(path.join(root, "clients/extension/static/manifest.json"), "utf8"));
  assert.equal(manifest.manifest_version, 3);
  assert.equal(manifest.background.service_worker, "service-worker.js");
  assert.equal(manifest.content_scripts, undefined);
  assert.ok(!manifest.permissions.includes("scripting"));
});

test("packaged clients load wallet assets from their own bundle", async () => {
  for (const target of ["web", "mobile", "extension"]) {
    const bootstrap = await readFile(path.join(root, "dist", target, "bootstrap.js"), "utf8");
    assert.match(bootstrap, /const walletScriptBase = "\.\/";/);
    const html = await readFile(path.join(root, "dist", target, "index.html"), "utf8");
    const shared = await readFile(path.join(root, "dist", target, "shared.js"), "utf8");
    assert.match(html, /<script src="\.\/shared\.js\?v=20260921-history-bnb-avax-v1"><\/script>/);
    assert.doesNotMatch(html, /<script>/);
    assert.match(shared, /AgentWalletCore/);
    assert.match(shared, /AgentWalletAPI/);
    assert.match(shared, /AgentWalletUI/);
    assert.match(shared, /AgentWalletBiometrics/);
  }
});
