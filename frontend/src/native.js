import { Capacitor } from "@capacitor/core";

export async function initNativeShell() {
  if (!Capacitor.isNativePlatform()) return;

  try {
    const [{ App: CapacitorApp }, { SplashScreen }, { StatusBar, Style }] =
      await Promise.all([
        import("@capacitor/app"),
        import("@capacitor/splash-screen"),
        import("@capacitor/status-bar"),
      ]);

    StatusBar.setStyle({ style: Style.Light }).catch(() => {});

    CapacitorApp.addListener("backButton", ({ canGoBack }) => {
      if (canGoBack) {
        window.history.back();
      } else {
        CapacitorApp.exitApp();
      }
    });

    window.setTimeout(() => {
      SplashScreen.hide().catch(() => {});
    }, 250);
  } catch (error) {
    console.warn("Edwins Lab native shell initialization failed:", error);
  }
}
