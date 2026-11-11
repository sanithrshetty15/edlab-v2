import { Capacitor } from "@capacitor/core";
import { navigate } from "./utils/router";
import {
  downloadNativeFile,
  filenameFromUrl,
  isDownloadableUrl,
  openNativeUrl,
} from "./services/nativeFiles";

function installLinkHandler() {
  document.addEventListener("click", (event) => {
    const anchor = event.target?.closest?.("a[href]");
    if (!anchor) return;

    const raw = anchor.getAttribute("href");
    if (!raw || raw.startsWith("#") || raw.startsWith("mailto:") || raw.startsWith("tel:")) {
      return;
    }

    let url;
    try {
      url = new URL(raw, window.location.href);
    } catch {
      return;
    }

    if (url.protocol !== "http:" && url.protocol !== "https:") return;

    const external = url.origin !== window.location.origin;
    const opensNewTab = anchor.target === "_blank";
    const downloads = anchor.hasAttribute("download") || isDownloadableUrl(url.href);

    if (!downloads && !external && !opensNewTab) return;

    event.preventDefault();
    if (downloads) {
      downloadNativeFile(url.href, filenameFromUrl(url.href)).catch(() => {
        openNativeUrl(url.href).catch(() => {});
      });
      return;
    }

    openNativeUrl(url.href).catch(() => {});
  });
}

export async function initNativeShell() {
  if (!Capacitor.isNativePlatform()) return;

  installLinkHandler();

  try {
    const [{ App: CapacitorApp }, { SplashScreen }, { StatusBar, Style }] =
      await Promise.all([
        import("@capacitor/app"),
        import("@capacitor/splash-screen"),
        import("@capacitor/status-bar"),
      ]);

    StatusBar.setStyle({ style: Style.Light }).catch(() => {});

    CapacitorApp.addListener("backButton", ({ canGoBack }) => {
      const pathname = window.location.pathname.replace(/\/+$/, "") || "/";
      if (canGoBack) {
        window.history.back();
        return;
      }
      if (pathname.startsWith("/app") && pathname !== "/app/dashboard") {
        navigate("/app/dashboard", { replace: true });
        return;
      }
      CapacitorApp.exitApp();
    });

    window.setTimeout(() => {
      SplashScreen.hide().catch(() => {});
    }, 250);
  } catch (error) {
    console.warn("Edwins Lab native shell initialization failed:", error);
  }
}
