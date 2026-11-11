import { useCallback, useEffect, useState } from "react";

function readDisplayMode() {
  if (typeof window === "undefined") return false;
  if (window.navigator.standalone) return true;
  if (typeof window.matchMedia !== "function") return false;

  return ["(display-mode: standalone)", "(display-mode: fullscreen)", "(display-mode: minimal-ui)"].some(
    (query) => window.matchMedia(query).matches,
  );
}

function readPlatform() {
  if (typeof navigator === "undefined") {
    return { isIos: false, isAndroid: false, isSafari: false, browser: "unknown" };
  }

  const ua = navigator.userAgent || "";
  const isIpad =
    /iPad/.test(ua) ||
    (navigator.platform === "MacIntel" && navigator.maxTouchPoints > 1);
  const isIos = /iPhone|iPod/.test(ua) || isIpad;
  const isAndroid = /Android/i.test(ua);
  const isSafari = /Safari/i.test(ua) && !/Chrome|Chromium|CriOS|Edg|OPR|SamsungBrowser/i.test(ua);
  const isEdge = /Edg(A|iOS)?\//i.test(ua);
  const isChrome = /Chrome|Chromium|CriOS/i.test(ua) && !isEdge;
  const isSamsung = /SamsungBrowser/i.test(ua);

  let browser = "other";
  if (isSafari) browser = "safari";
  else if (isEdge) browser = "edge";
  else if (isSamsung) browser = "samsung";
  else if (isChrome) browser = "chrome";

  return { isIos, isAndroid, isSafari, browser };
}

export function usePwaInstall() {
  const platform = readPlatform();
  const [deferredPrompt, setDeferredPrompt] = useState(null);
  const [isInstalled, setIsInstalled] = useState(readDisplayMode);
  const [status, setStatus] = useState(() => {
    if (readDisplayMode()) return "installed";
    return platform.isIos ? "ios" : "unavailable";
  });

  useEffect(() => {
    const onBeforeInstallPrompt = (event) => {
      event.preventDefault();
      setDeferredPrompt(event);
      setStatus((current) =>
        current === "installed" || current === "prompt-open" || current === "accepted"
          ? current
          : "available",
      );
    };

    const onAppInstalled = () => {
      setDeferredPrompt(null);
      setIsInstalled(true);
      setStatus("installed");
    };

    window.addEventListener("beforeinstallprompt", onBeforeInstallPrompt);
    window.addEventListener("appinstalled", onAppInstalled);

    return () => {
      window.removeEventListener("beforeinstallprompt", onBeforeInstallPrompt);
      window.removeEventListener("appinstalled", onAppInstalled);
    };
  }, []);

  useEffect(() => {
    if (typeof window.matchMedia !== "function") return undefined;

    const queries = [
      "(display-mode: standalone)",
      "(display-mode: fullscreen)",
      "(display-mode: minimal-ui)",
    ].map((query) => window.matchMedia(query));

    const onChange = () => {
      if (queries.some((media) => media.matches)) {
        setIsInstalled(true);
        setStatus("installed");
      }
    };

    queries.forEach((media) => media.addEventListener("change", onChange));
    return () => queries.forEach((media) => media.removeEventListener("change", onChange));
  }, []);

  const promptInstall = useCallback(async () => {
    if (!deferredPrompt) return "unavailable";

    setStatus("prompt-open");

    try {
      deferredPrompt.prompt();
      const { outcome } = await deferredPrompt.userChoice;
      setDeferredPrompt(null);

      if (outcome === "accepted") {
        setStatus("accepted");
        return "accepted";
      }

      setStatus("dismissed");
      return "dismissed";
    } catch {
      setDeferredPrompt(null);
      setStatus("unavailable");
      return "error";
    }
  }, [deferredPrompt]);

  const canInstall = Boolean(deferredPrompt) && !isInstalled;

  return {
    ...platform,
    canInstall,
    isInstalled,
    status: isInstalled ? "installed" : canInstall && status === "unavailable" ? "available" : status,
    promptInstall,
  };
}
