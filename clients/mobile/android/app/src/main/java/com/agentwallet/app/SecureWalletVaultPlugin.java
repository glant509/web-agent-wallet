package com.agentwallet.app;

import android.content.Context;
import android.content.SharedPreferences;
import android.security.keystore.KeyGenParameterSpec;
import android.security.keystore.KeyProperties;
import android.util.Base64;

import com.getcapacitor.JSObject;
import com.getcapacitor.Plugin;
import com.getcapacitor.PluginCall;
import com.getcapacitor.PluginMethod;
import com.getcapacitor.annotation.CapacitorPlugin;

import java.nio.charset.StandardCharsets;
import java.security.KeyStore;

import javax.crypto.Cipher;
import javax.crypto.KeyGenerator;
import javax.crypto.SecretKey;
import javax.crypto.spec.GCMParameterSpec;

@CapacitorPlugin(name = "SecureWalletVault")
public class SecureWalletVaultPlugin extends Plugin {
    private static final String KEY_ALIAS = "agent_wallet_vault_key";
    private static final String PREFERENCES = "agent_wallet_secure_vault";
    private static final String CIPHERTEXT = "ciphertext";
    private static final String IV = "iv";

    @PluginMethod
    public void load(PluginCall call) {
        try {
            SharedPreferences preferences = preferences();
            String ciphertextValue = preferences.getString(CIPHERTEXT, null);
            String ivValue = preferences.getString(IV, null);
            JSObject result = new JSObject();
            if (ciphertextValue == null || ivValue == null) {
                result.put("value", null);
                call.resolve(result);
                return;
            }
            Cipher cipher = Cipher.getInstance("AES/GCM/NoPadding");
            cipher.init(
                Cipher.DECRYPT_MODE,
                getOrCreateKey(),
                new GCMParameterSpec(128, Base64.decode(ivValue, Base64.NO_WRAP))
            );
            byte[] plaintext = cipher.doFinal(Base64.decode(ciphertextValue, Base64.NO_WRAP));
            result.put("value", new String(plaintext, StandardCharsets.UTF_8));
            java.util.Arrays.fill(plaintext, (byte) 0);
            call.resolve(result);
        } catch (Exception error) {
            call.reject("Unable to load the encrypted wallet vault", error);
        }
    }

    @PluginMethod
    public void save(PluginCall call) {
        String value = call.getString("value");
        if (value == null) {
            call.reject("Vault value is required");
            return;
        }
        byte[] plaintext = value.getBytes(StandardCharsets.UTF_8);
        try {
            Cipher cipher = Cipher.getInstance("AES/GCM/NoPadding");
            cipher.init(Cipher.ENCRYPT_MODE, getOrCreateKey());
            byte[] ciphertext = cipher.doFinal(plaintext);
            boolean stored = preferences().edit()
                .putString(CIPHERTEXT, Base64.encodeToString(ciphertext, Base64.NO_WRAP))
                .putString(IV, Base64.encodeToString(cipher.getIV(), Base64.NO_WRAP))
                .commit();
            if (!stored) {
                throw new IllegalStateException("Encrypted wallet vault was not persisted");
            }
            java.util.Arrays.fill(ciphertext, (byte) 0);
            call.resolve();
        } catch (Exception error) {
            call.reject("Unable to save the encrypted wallet vault", error);
        } finally {
            java.util.Arrays.fill(plaintext, (byte) 0);
        }
    }

    @PluginMethod
    public void remove(PluginCall call) {
        try {
            if (!preferences().edit().clear().commit()) {
                throw new IllegalStateException("Encrypted wallet vault was not removed");
            }
            KeyStore keyStore = KeyStore.getInstance("AndroidKeyStore");
            keyStore.load(null);
            if (keyStore.containsAlias(KEY_ALIAS)) {
                keyStore.deleteEntry(KEY_ALIAS);
            }
            call.resolve();
        } catch (Exception error) {
            call.reject("Unable to remove the encrypted wallet vault", error);
        }
    }

    private SharedPreferences preferences() {
        return getContext().getSharedPreferences(PREFERENCES, Context.MODE_PRIVATE);
    }

    private SecretKey getOrCreateKey() throws Exception {
        KeyStore keyStore = KeyStore.getInstance("AndroidKeyStore");
        keyStore.load(null);
        if (keyStore.containsAlias(KEY_ALIAS)) {
            return (SecretKey) keyStore.getKey(KEY_ALIAS, null);
        }
        KeyGenerator generator = KeyGenerator.getInstance(KeyProperties.KEY_ALGORITHM_AES, "AndroidKeyStore");
        generator.init(new KeyGenParameterSpec.Builder(
            KEY_ALIAS,
            KeyProperties.PURPOSE_ENCRYPT | KeyProperties.PURPOSE_DECRYPT
        )
            .setBlockModes(KeyProperties.BLOCK_MODE_GCM)
            .setEncryptionPaddings(KeyProperties.ENCRYPTION_PADDING_NONE)
            .setKeySize(256)
            .setRandomizedEncryptionRequired(true)
            .build());
        return generator.generateKey();
    }
}
