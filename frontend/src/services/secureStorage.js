import { Capacitor, registerPlugin } from "@capacitor/core";

const SecureStorage = registerPlugin("SecureStorage");

export function isNativeSecureStorageAvailable() {
  return Capacitor.isNativePlatform();
}

export const secureStorage = {
  async get() {
    if (!isNativeSecureStorageAvailable()) return null;
    try {
      const result = await SecureStorage.get();
      return typeof result?.value === "string" ? result.value : null;
    } catch {
      return null;
    }
  },
  async set(value) {
    if (!isNativeSecureStorageAvailable()) return;
    await SecureStorage.set({ value });
  },
  async remove() {
    if (!isNativeSecureStorageAvailable()) return;
    try {
      await SecureStorage.remove();
    } catch {
      // The in-memory session is already cleared.
    }
  },
};
