import { useEffect, useState } from "react";
import {
  fetchReleaseManifest,
  getInstalledAppInfo,
  isNativeUpdateSupported,
  resolveUpdate,
} from "../services/appUpdater";

const EMPTY = {
  status: "idle",
  manifest: null,
  mandatory: false,
  error: null,
};

/**
 * Checks the published release manifest against the installed Android build.
 * Runs only inside the native shell and never throws into the UI.
 */
export function useAppUpdate(enabled = true) {
  const [state, setState] = useState(() =>
    enabled && isNativeUpdateSupported() ? { ...EMPTY, status: "checking" } : EMPTY,
  );

  useEffect(() => {
    if (!enabled || !isNativeUpdateSupported()) return undefined;

    let active = true;
    const controller = new AbortController();

    (async () => {
      try {
        const [manifest, installed] = await Promise.all([
          fetchReleaseManifest({ signal: controller.signal }),
          getInstalledAppInfo(),
        ]);

        const decision = resolveUpdate(manifest, installed);
        if (!active) return;

        setState({
          status: decision.available ? "available" : "current",
          manifest: decision.available ? manifest : null,
          mandatory: decision.mandatory,
          error: null,
        });
      } catch (error) {
        if (!active) return;
        setState({
          status: error?.code === "offline" ? "offline" : "error",
          manifest: null,
          mandatory: false,
          error,
        });
      }
    })();

    return () => {
      active = false;
      controller.abort();
    };
  }, [enabled]);

  return state;
}
