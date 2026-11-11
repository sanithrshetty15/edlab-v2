import { Capacitor, registerPlugin } from "@capacitor/core";

const FileHandler = registerPlugin("FileHandler");
const DOWNLOADABLE = /\.(apk|pdf|zip|rar|7z|csv|xlsx?|docx?|pptx?|txt|png|jpe?g|gif|webp)$/i;

export function filenameFromUrl(url, fallback = "edwinslab-download") {
  try {
    const name = decodeURIComponent(new URL(url).pathname.split("/").filter(Boolean).pop() || "");
    return name || fallback;
  } catch {
    return fallback;
  }
}

export function isDownloadableUrl(url) {
  try {
    return DOWNLOADABLE.test(new URL(url).pathname);
  } catch {
    return false;
  }
}

export async function openNativeUrl(url) {
  if (Capacitor.isNativePlatform()) {
    await FileHandler.openUrl({ url });
    return;
  }
  window.open(url, "_blank", "noopener,noreferrer");
}

export async function downloadNativeFile(url, filename) {
  if (Capacitor.isNativePlatform()) {
    return FileHandler.download({
      url,
      filename: filename || filenameFromUrl(url),
    });
  }

  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = filename || filenameFromUrl(url);
  anchor.rel = "noreferrer noopener";
  document.body.appendChild(anchor);
  anchor.click();
  anchor.remove();
  return null;
}
