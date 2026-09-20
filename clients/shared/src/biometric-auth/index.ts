type BiometryType = "face" | "fingerprint" | "iris" | "none";

export interface BiometricStatus {
  available: boolean;
  enrolled: boolean;
  enabled: boolean;
  biometryType: BiometryType;
  label: string;
}

interface NativeBiometricPlugin {
  status(): Promise<Partial<BiometricStatus>>;
  saveCredential(options: { credential: string; reason: string }): Promise<void>;
  authenticate(options: { reason: string }): Promise<{ credential?: string }>;
  removeCredential(): Promise<void>;
}

function plugin(): NativeBiometricPlugin | null {
  const capacitor = (window as Window & { Capacitor?: { Plugins?: Record<string, unknown> } }).Capacitor;
  return (capacitor?.Plugins?.WalletBiometrics as NativeBiometricPlugin | undefined) || null;
}

function labelFor(type: BiometryType): string {
  if (type === "face") return "Face ID";
  if (type === "fingerprint") return "指纹/Touch ID";
  if (type === "iris") return "虹膜识别";
  return "生物识别";
}

export async function status(): Promise<BiometricStatus> {
  const native = plugin();
  if (!native) return { available: false, enrolled: false, enabled: false, biometryType: "none", label: "生物识别" };
  const result = await native.status();
  const biometryType = result.biometryType || "none";
  return {
    available: Boolean(result.available),
    enrolled: Boolean(result.enrolled),
    enabled: Boolean(result.enabled),
    biometryType,
    label: result.label || labelFor(biometryType)
  };
}

export async function enable(credential: string, reason = "启用钱包快捷解锁"): Promise<void> {
  if (!credential) throw new Error("Credential is required");
  const native = plugin();
  if (!native) throw new Error("Biometric authentication is unavailable");
  await native.saveCredential({ credential, reason });
}

export async function authenticate(reason = "验证身份以解锁钱包"): Promise<string> {
  const native = plugin();
  if (!native) throw new Error("Biometric authentication is unavailable");
  const result = await native.authenticate({ reason });
  if (!result.credential) throw new Error("Biometric credential is unavailable");
  return result.credential;
}

export async function disable(): Promise<void> {
  const native = plugin();
  if (native) await native.removeCredential();
}
