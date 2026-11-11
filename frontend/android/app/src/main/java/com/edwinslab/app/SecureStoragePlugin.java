package com.edwinslab.app;

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

@CapacitorPlugin(name = "SecureStorage")
public class SecureStoragePlugin extends Plugin {
    private static final String KEY_ALIAS = "edwinslab_session_key";
    private static final String PREFS_NAME = "edwinslab_secure_storage";
    private static final String TOKEN_KEY = "session_token";
    private static final String CIPHER_TRANSFORMATION = "AES/GCM/NoPadding";
    private static final int GCM_TAG_LENGTH_BITS = 128;

    private SharedPreferences preferences() {
        return getContext().getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE);
    }

    private SecretKey getOrCreateKey() throws Exception {
        KeyStore keyStore = KeyStore.getInstance("AndroidKeyStore");
        keyStore.load(null);

        if (keyStore.containsAlias(KEY_ALIAS)) {
            return (SecretKey) keyStore.getKey(KEY_ALIAS, null);
        }

        KeyGenerator generator = KeyGenerator.getInstance(
            KeyProperties.KEY_ALGORITHM_AES,
            "AndroidKeyStore"
        );
        generator.init(
            new KeyGenParameterSpec.Builder(
                KEY_ALIAS,
                KeyProperties.PURPOSE_ENCRYPT | KeyProperties.PURPOSE_DECRYPT
            )
                .setBlockModes(KeyProperties.BLOCK_MODE_GCM)
                .setEncryptionPaddings(KeyProperties.ENCRYPTION_PADDING_NONE)
                .setRandomizedEncryptionRequired(true)
                .build()
        );
        return generator.generateKey();
    }

    @PluginMethod
    public void get(PluginCall call) {
        String stored = preferences().getString(TOKEN_KEY, null);
        if (stored == null || stored.isEmpty()) {
            call.resolve();
            return;
        }

        try {
            String[] parts = stored.split(":", 2);
            if (parts.length != 2) throw new IllegalStateException("Malformed value");

            byte[] iv = Base64.decode(parts[0], Base64.NO_WRAP);
            byte[] payload = Base64.decode(parts[1], Base64.NO_WRAP);

            Cipher cipher = Cipher.getInstance(CIPHER_TRANSFORMATION);
            cipher.init(
                Cipher.DECRYPT_MODE,
                getOrCreateKey(),
                new GCMParameterSpec(GCM_TAG_LENGTH_BITS, iv)
            );
            String value = new String(cipher.doFinal(payload), StandardCharsets.UTF_8);

            JSObject result = new JSObject();
            result.put("value", value);
            call.resolve(result);
        } catch (Exception error) {
            preferences().edit().remove(TOKEN_KEY).apply();
            call.resolve();
        }
    }

    @PluginMethod
    public void set(PluginCall call) {
        String value = call.getString("value");
        if (value == null) {
            call.reject("A value is required");
            return;
        }

        try {
            Cipher cipher = Cipher.getInstance(CIPHER_TRANSFORMATION);
            cipher.init(Cipher.ENCRYPT_MODE, getOrCreateKey());

            byte[] payload = cipher.doFinal(value.getBytes(StandardCharsets.UTF_8));
            String stored = Base64.encodeToString(cipher.getIV(), Base64.NO_WRAP)
                + ":"
                + Base64.encodeToString(payload, Base64.NO_WRAP);

            boolean saved = preferences().edit().putString(TOKEN_KEY, stored).commit();
            if (!saved) {
                call.reject("Secure storage could not be written");
                return;
            }
            call.resolve();
        } catch (Exception error) {
            call.reject("Secure storage is unavailable", error);
        }
    }

    @PluginMethod
    public void remove(PluginCall call) {
        preferences().edit().remove(TOKEN_KEY).apply();
        call.resolve();
    }
}
