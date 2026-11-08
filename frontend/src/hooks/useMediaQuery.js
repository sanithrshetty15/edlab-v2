import { useCallback, useSyncExternalStore } from "react";

export function useMediaQuery(query) {
  const subscribe = useCallback(
    (onChange) => {
      if (typeof window === "undefined") return () => {};

      const media = window.matchMedia(query);
      media.addEventListener("change", onChange);
      return () => media.removeEventListener("change", onChange);
    },
    [query],
  );

  const getSnapshot = useCallback(() => {
    if (typeof window === "undefined") return false;
    return window.matchMedia(query).matches;
  }, [query]);

  return useSyncExternalStore(subscribe, getSnapshot, () => false);
}

export function useReducedMotion() {
  return useMediaQuery("(prefers-reduced-motion: reduce)");
}
