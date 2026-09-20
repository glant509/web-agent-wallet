package com.agentwallet.app;

import android.os.Build;
import android.security.keystore.KeyGenParameterSpec;
import android.security.keystore.KeyProperties;
import android.util.Base64;

import androidx.annotation.NonNull;
import androidx.biometric.BiometricManager;
import androidx.biometric.BiometricPrompt;
import androidx.core.content.ContextCompat;
import androidx.fragment.app.FragmentActivity;

import com.getcapacitor.JSObject;
import com.getcapacitor.Plugin;
import com.getcapacitor.PluginCall;
import com.getcapacitor.PluginMethod;
import com.getcapacitor.annotation.CapacitorPlugin;

import java.nio.charset.StandardCharsets;
import java.security.KeyStore;
import java.util.concurrent.Executor;

import javax.crypto.Cipher;
import javax.crypto.KeyGenerator;
import javax.crypto.SecretKey;
import javax.crypto.spec.GCMParameterSpec;

@CapacitorPlugin(name = "WalletBiometrics")
public class WalletBiometricsPlugin extends Plugin {
    private static final String KEY_ALIAS = "agent_wallet_biometric_unlock_key";
    private static final String PREFERENCES = "agent_wallet_biometric_unlock";
    private static final String CIPHERTEXT = "credential_ciphertext";
    private static final String IV = "credential_iv";
    private static final int AUTHENTICATORS = BiometricManager.Authenticators.BIOMETRIC_STRONG;

    @PluginMethod
    public void status(PluginCall call) {
        int result = BiometricManager.from(getContext()).canAuthenticate(AUTHENTICATORS);
        JSObject response = new JSObject();
        response.put("available", result == BiometricManager.BIOMETRIC_SUCCESS);
        response.put("enrolled", result != BiometricManager.BIOMETRIC_ERROR_NONE_ENROLLED);
        response.put("enabled", preferences().contains(CIPHERTEXT) && preferences().contains(IV));
        response.put("biometryType", "fingerprint");
        response.put("label", "指纹/生物识别");
        response.put("statusCode", result);
        call.resolve(response);
    }

    @PluginMethod
    public void saveCredential(PluginCall call) {
        String credential = call.getString("credential");
        if (credential == null || credential.isEmpty()) {
            call.reject("Credential is required");
            return;
        }
        try {
            Cipher cipher = createEncryptCipher();
            authenticate(call, cipher, call.getString("reason", "启用钱包快捷解锁"), result -> {
                byte[] plaintext = credential.getBytes(StandardCharsets.UTF_8);
                try {
                    byte[] ciphertext = result.getCryptoObject().getCipher().doFinal(plaintext);
                    boolean stored = preferences().edit()
                        .putString(CIPHERTEXT, Base64.encodeToString(ciphertext, Base64.NO_WRAP))
                        .putString(IV, Base64.encodeToString(result.getCryptoObject().getCipher().getIV(), Base64.NO_WRAP))
                        .commit();
                    java.util.Arrays.fill(ciphertext, (byte) 0);
                    if (!stored) throw new IllegalStateException("Credential was not persisted");
                    call.resolve();
                } catch (Exception error) {
                    call.reject("Unable to save biometric credential", error);
                } finally {
                    java.util.Arrays.fill(plaintext, (byte) 0);
                }
            });
        } catch (Exception error) {
            call.reject("Unable to initialize biometric credential", error);
        }
    }

    @PluginMethod
    public void authenticate(PluginCall call) {
        String ciphertext = preferences().getString(CIPHERTEXT, null);
        String iv = preferences().getString(IV, null);
        if (ciphertext == null || iv == null) {
            call.reject("Biometric unlock is not enabled");
            return;
        }
        try {
            Cipher cipher = createDecryptCipher(Base64.decode(iv, Base64.NO_WRAP));
            authenticate(call, cipher, call.getString("reason", "验证身份以解锁钱包"), result -> {
                byte[] plaintext = null;
                try {
                    plaintext = result.getCryptoObject().getCipher().doFinal(Base64.decode(ciphertext, Base64.NO_WRAP));
                    JSObject response = new JSObject();
                    response.put("credential", new String(plaintext, StandardCharsets.UTF_8));
                    call.resolve(response);
                } catch (Exception error) {
                    removeStoredCredential();
                    call.reject("Unable to decrypt biometric credential", error);
                } finally {
                    if (plaintext != null) java.util.Arrays.fill(plaintext, (byte) 0);
                }
            });
        } catch (Exception error) {
            removeStoredCredential();
            call.reject("Unable to initialize biometric authentication", error);
        }
    }

    @PluginMethod
    public void removeCredential(PluginCall call) {
        try {
            removeStoredCredential();
            call.resolve();
        } catch (Exception error) {
            call.reject("Unable to remove biometric credential", error);
        }
    }

    private interface AuthenticationSuccess {
        void accept(BiometricPrompt.AuthenticationResult result);
    }

    private void authenticate(PluginCall call, Cipher cipher, String reason, AuthenticationSuccess success) {
        FragmentActivity activity = (FragmentActivity) getActivity();
        Executor executor = ContextCompat.getMainExecutor(getContext());
        BiometricPrompt prompt = new BiometricPrompt(activity, executor, new BiometricPrompt.AuthenticationCallback() {
            @Override
            public void onAuthenticationError(int errorCode, @NonNull CharSequence errString) {
                call.reject("Biometric authentication failed: " + errString, String.valueOf(errorCode));
            }

            @Override
            public void onAuthenticationSucceeded(@NonNull BiometricPrompt.AuthenticationResult result) {
                success.accept(result);
            }

            @Override
            public void onAuthenticationFailed() {
                notifyListeners("biometricAttemptFailed", new JSObject());
            }
        });
        BiometricPrompt.PromptInfo info = new BiometricPrompt.PromptInfo.Builder()
            .setTitle("Agent Wallet")
            .setSubtitle(reason)
            .setAllowedAuthenticators(AUTHENTICATORS)
            .setNegativeButtonText("取消")
            .build();
        getActivity().runOnUiThread(() -> prompt.authenticate(info, new BiometricPrompt.CryptoObject(cipher)));
    }

    private Cipher createEncryptCipher() throws Exception {
        Cipher cipher = Cipher.getInstance("AES/GCM/NoPadding");
        cipher.init(Cipher.ENCRYPT_MODE, getOrCreateKey());
        return cipher;
    }

    private Cipher createDecryptCipher(byte[] iv) throws Exception {
        Cipher cipher = Cipher.getInstance("AES/GCM/NoPadding");
        cipher.init(Cipher.DECRYPT_MODE, getOrCreateKey(), new GCMParameterSpec(128, iv));
        return cipher;
    }

    @SuppressWarnings("deprecation")
    private SecretKey getOrCreateKey() throws Exception {
        KeyStore keyStore = KeyStore.getInstance("AndroidKeyStore");
        keyStore.load(null);
        if (keyStore.containsAlias(KEY_ALIAS)) {
            return (SecretKey) keyStore.getKey(KEY_ALIAS, null);
        }
        KeyGenParameterSpec.Builder builder = new KeyGenParameterSpec.Builder(
            KEY_ALIAS,
            KeyProperties.PURPOSE_ENCRYPT | KeyProperties.PURPOSE_DECRYPT
        )
            .setBlockModes(KeyProperties.BLOCK_MODE_GCM)
            .setEncryptionPaddings(KeyProperties.ENCRYPTION_PADDING_NONE)
            .setKeySize(256)
            .setUserAuthenticationRequired(true)
            .setInvalidatedByBiometricEnrollment(true);
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.R) {
            builder.setUserAuthenticationParameters(0, KeyProperties.AUTH_BIOMETRIC_STRONG);
        } else {
            builder.setUserAuthenticationValidityDurationSeconds(-1);
        }
        KeyGenerator generator = KeyGenerator.getInstance(KeyProperties.KEY_ALGORITHM_AES, "AndroidKeyStore");
        generator.init(builder.build());
        return generator.generateKey();
    }

    private void removeStoredCredential() {
        preferences().edit().clear().commit();
        try {
            KeyStore keyStore = KeyStore.getInstance("AndroidKeyStore");
            keyStore.load(null);
            if (keyStore.containsAlias(KEY_ALIAS)) keyStore.deleteEntry(KEY_ALIAS);
        } catch (Exception ignored) {
            // Preferences are already cleared; stale key material is unusable without ciphertext.
        }
    }

    private android.content.SharedPreferences preferences() {
        return getContext().getSharedPreferences(PREFERENCES, android.content.Context.MODE_PRIVATE);
    }
}
