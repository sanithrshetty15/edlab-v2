import { Capacitor, registerPlugin } from "@capacitor/core";

const Updater = registerPlugin("AppUpdater");

export const EXPECTED_PACKAGE_ID = "com.edwinslab.app";

const DEFAULT_MANIFEST_URL = "https://edwinslab.vercel.app/downloads/edwinslab-apk.json";
const ENV = import.meta.env || {};
export const APK_MANIFEST_URL = (ENV.VITE_APK_MANIFEST_URL || DEFAULT_MANIFEST_URL).trim();

const MANIFEST_TIMEOUT_MS = 12000;

export class UpdateError extends Error {
  constructor(message, { code = "update_error", cause } = {}) {
    super(message);
    this.name = "UpdateError";
    this.code = code;
    if (cause) this.cause = cause;
  }
}

export function isNativeUpdateSupported() {
  return Capacitor.isNativePlatform();
}

function isFinitePositive(value) {
  return Number.isFinite(value) && value > 0;
}

function parseReleaseNotes(value) {
  if (Array.isArray(value)) {
    return value.map((item) => String(item).trim()).filter(Boolean).slice(0, 12);
  }
  if (typeof value === "string") {
    return value
      .split(/\r?\n/)
      .map((line) => line.replace(/^[-*•]\s*/, "").trim())
      .filter(Boolean)
      .slice(0, 12);
  }
  return [];
}

function resolveHttpsUrl(value, baseUrl) {
  if (typeof value !== "string" || !value.trim()) return null;

  try {
    const url = new URL(value.trim(), baseUrl);
    return url.protocol === "https:" ? url.toString() : null;
  } catch {
    return null;
  }
}

/**
 * Validates and normalizes the remote release manifest.
 * Throws UpdateError when the manifest cannot be trusted or used.
 */
export function normalizeManifest(data, { manifestUrl = APK_MANIFEST_URL } = {}) {
  if (!data || typeof data !== "object") {
    throw new UpdateError("The release manifest is invalid.", { code: "invalid_manifest" });
  }

  const versionCode = Number(data.versionCode);
  if (!isFinitePositive(versionCode) || !Number.isInteger(versionCode)) {
    throw new UpdateError("The release manifest has no valid version code.", {
      code: "invalid_manifest",
    });
  }

  const versionName = typeof data.versionName === "string" ? data.versionName.trim() : "";
  if (!versionName) {
    throw new UpdateError("The release manifest has no version name.", {
      code: "invalid_manifest",
    });
  }

  const apkUrl =
    resolveHttpsUrl(data.apkUrl, manifestUrl) || resolveHttpsUrl(data.file, manifestUrl);
  if (!apkUrl) {
    throw new UpdateError("The release manifest has no secure APK URL.", {
      code: "invalid_manifest",
    });
  }

  const sha256 = typeof data.sha256 === "string" ? data.sha256.replace(/[^a-fA-F0-9]/g, "").toLowerCase() : "";
  if (sha256.length !== 64) {
    throw new UpdateError("The release manifest has no valid SHA-256 checksum.", {
      code: "invalid_manifest",
    });
  }

  if (data.packageId && data.packageId !== EXPECTED_PACKAGE_ID) {
    throw new UpdateError("The release manifest belongs to another application.", {
      code: "wrong_package",
    });
  }

  const minimumSupportedVersion = Number(data.minimumSupportedVersion);
  const sizeBytes = Number(data.sizeBytes);

  return {
    versionName,
    versionCode,
    apkUrl,
    sha256,
    fileName: apkUrl.split("/").pop() || `EdwinsLab-${versionName}.apk`,
    releaseNotes: parseReleaseNotes(data.releaseNotes),
    mandatoryUpdate: data.mandatoryUpdate === true || data.mandatoryUpdate === "true",
    minimumSupportedVersion: isFinitePositive(minimumSupportedVersion)
      ? Math.floor(minimumSupportedVersion)
      : 0,
    sizeBytes: isFinitePositive(sizeBytes) ? sizeBytes : null,
    packageId: data.packageId || EXPECTED_PACKAGE_ID,
    releasedAt: typeof data.releasedAt === "string" ? data.releasedAt : null,
  };
}

/**
 * Compares the installed build with the published release.
 * Updates are only offered when the remote versionCode is newer.
 */
export function resolveUpdate(manifest, installed) {
  const installedCode = Number(installed?.versionCode) || 0;
  const available = Boolean(manifest) && manifest.versionCode > installedCode;
  const mandatory =
    available &&
    (manifest.mandatoryUpdate || installedCode < (manifest.minimumSupportedVersion || 0));

  return {
    available,
    mandatory,
    installedVersionCode: installedCode,
    installedVersionName: installed?.versionName || null,
  };
}

export async function fetchReleaseManifest({ signal, timeout = MANIFEST_TIMEOUT_MS } = {}) {
  if (typeof navigator !== "undefined" && navigator.onLine === false) {
    throw new UpdateError("You are offline.", { code: "offline" });
  }

  const controller = new AbortController();

  if (signal) {
    if (signal.aborted) controller.abort();
    else signal.addEventListener("abort", () => controller.abort(), { once: true });
  }

  const timer = window.setTimeout(() => controller.abort(), timeout);

  let response;
  try {
    response = await fetch(APK_MANIFEST_URL, {
      cache: "no-store",
      signal: controller.signal,
      headers: { Accept: "application/json" },
    });
  } catch (error) {
    const offline = typeof navigator !== "undefined" && navigator.onLine === false;
    throw new UpdateError(
      offline ? "You are offline." : "The update server could not be reached.",
      { code: offline ? "offline" : "manifest_unavailable", cause: error },
    );
  } finally {
    window.clearTimeout(timer);
  }

  if (!response.ok) {
    throw new UpdateError("The release manifest is not available.", {
      code: "manifest_unavailable",
    });
  }

  let data;
  try {
    data = await response.json();
  } catch (error) {
    throw new UpdateError("The release manifest could not be read.", {
      code: "invalid_manifest",
      cause: error,
    });
  }

  return normalizeManifest(data);
}

export async function getInstalledAppInfo() {
  if (!Capacitor.isNativePlatform()) return null;

  try {
    return await Updater.getAppInfo();
  } catch (error) {
    throw new UpdateError("The installed app version could not be read.", {
      code: "app_info_unavailable",
      cause: error,
    });
  }
}

export async function downloadReleaseApk({ url, sha256, fileName, onProgress }) {
  let handle = null;

  if (typeof onProgress === "function") {
    handle = await Updater.addListener("downloadProgress", (event) => {
      if (!event || event.phase !== "downloading") return;
      const progress = Math.max(0, Math.min(100, Number(event.progress) || 0));
      onProgress(progress);
    }).catch(() => null);
  }

  try {
    return await Updater.downloadApk({ url, sha256, fileName });
  } catch (error) {
    throw new UpdateError(
      error?.message || "The update could not be downloaded.",
      { code: error?.code || "download_failed", cause: error },
    );
  } finally {
    handle?.remove?.();
  }
}

export async function installReleaseApk(filePath) {
  try {
    return await Updater.installApk({ filePath });
  } catch (error) {
    throw new UpdateError(
      error?.message || "The Android installer could not be opened.",
      { code: error?.code || "install_failed", cause: error },
    );
  }
}
