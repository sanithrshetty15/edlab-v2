import { useSyncExternalStore } from "react";

const listeners = new Set();
let popstateBound = false;

function normalize(path) {
  const trimmed = path.replace(/\/+$/, "");
  return trimmed || "/";
}

function emit() {
  listeners.forEach((listener) => listener());
}

function getSnapshot() {
  if (typeof window === "undefined") return "/";
  return normalize(window.location.pathname);
}

function subscribe(listener) {
  if (!popstateBound && typeof window !== "undefined") {
    popstateBound = true;
    window.addEventListener("popstate", () => {
      window.scrollTo({ top: 0, left: 0, behavior: "auto" });
      emit();
    });
  }

  listeners.add(listener);
  return () => listeners.delete(listener);
}

export function usePathname() {
  return useSyncExternalStore(subscribe, getSnapshot, () => "/");
}

export function navigate(to) {
  if (typeof window === "undefined") return;

  const url = new URL(to, window.location.origin);
  const target = normalize(url.pathname);

  if (target === getSnapshot()) return;

  window.history.pushState({}, "", url);
  window.scrollTo({ top: 0, left: 0, behavior: "auto" });
  emit();
}
