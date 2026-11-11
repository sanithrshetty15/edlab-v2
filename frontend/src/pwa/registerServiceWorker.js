const UPDATE_EVENT = "edwinslab:update-available";
const CONTROLLER_EVENT = "edwinslab:controller-changed";

let registration = null;
let updateRequested = false;

function announceUpdate(reg) {
  if (typeof window === "undefined") return;
  window.dispatchEvent(new CustomEvent(UPDATE_EVENT, { detail: reg }));
}

function watchRegistration(reg) {
  if (reg.waiting && navigator.serviceWorker.controller) {
    announceUpdate(reg);
    return;
  }

  reg.addEventListener("updatefound", () => {
    const installing = reg.installing;
    if (!installing) return;

    installing.addEventListener("statechange", () => {
      if (installing.state === "installed" && navigator.serviceWorker.controller) {
        announceUpdate(reg);
      }
    });
  });
}

export function initServiceWorker() {
  if (typeof window === "undefined") return;
  if (!("serviceWorker" in navigator)) return;
  if (!import.meta.env.PROD) return;
  if (window.location.protocol !== "https:" && window.location.hostname !== "localhost") {
    return;
  }

  window.addEventListener("load", async () => {
    try {
      registration = await navigator.serviceWorker.register("/sw.js", { scope: "/" });
      watchRegistration(registration);

      navigator.serviceWorker.addEventListener("controllerchange", () => {
        if (updateRequested) {
          updateRequested = false;
          window.location.reload();
          return;
        }
        window.dispatchEvent(new CustomEvent(CONTROLLER_EVENT));
      });

      document.addEventListener("visibilitychange", () => {
        if (document.visibilityState === "visible") {
          registration?.update().catch(() => {});
        }
      });
    } catch (error) {
      console.warn("Edwins Lab service worker registration failed:", error);
    }
  });
}

export async function applyServiceWorkerUpdate() {
  updateRequested = true;
  const reg = registration || (await navigator.serviceWorker?.getRegistration?.());
  const waiting = reg?.waiting;

  if (waiting) {
    waiting.postMessage({ type: "SKIP_WAITING" });
    return;
  }

  updateRequested = false;
  window.location.reload();
}

export { UPDATE_EVENT, CONTROLLER_EVENT };
