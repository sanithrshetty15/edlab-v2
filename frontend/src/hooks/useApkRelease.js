import { useEffect, useState } from "react";

const METADATA_URL = "/downloads/edwinslab-apk.json";

function formatBytes(bytes) {
  if (!Number.isFinite(bytes) || bytes <= 0) return null;
  if (bytes < 1024 * 1024) return `${Math.round(bytes / 1024)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

export function useApkRelease() {
  const [state, setState] = useState({
    status: "loading",
    release: null,
    error: null,
  });

  useEffect(() => {
    let active = true;
    const controller = new AbortController();

    async function load() {
      try {
        const response = await fetch(METADATA_URL, {
          cache: "no-store",
          signal: controller.signal,
        });

        if (!response.ok) throw new Error("APK metadata was not found.");

        const data = await response.json();
        if (!data || typeof data.file !== "string" || !data.file.endsWith(".apk")) {
          throw new Error("APK metadata is invalid.");
        }
        if (!data.file.startsWith("/downloads/")) {
          throw new Error("APK path is not allowed.");
        }

        const head = await fetch(data.file, {
          method: "HEAD",
          cache: "no-store",
          signal: controller.signal,
        });

        if (!head.ok) throw new Error("APK file is not available.");

        const headerSize = Number(head.headers.get("content-length"));
        const sizeBytes = Number.isFinite(headerSize) && headerSize > 0 ? headerSize : data.sizeBytes;

        if (!active) return;

        setState({
          status: "available",
          release: {
            ...data,
            sizeBytes,
            sizeLabel: formatBytes(sizeBytes),
          },
          error: null,
        });
      } catch (error) {
        if (!active || controller.signal.aborted) return;
        setState({
          status: "unavailable",
          release: null,
          error,
        });
      }
    }

    load();

    return () => {
      active = false;
      controller.abort();
    };
  }, []);

  return state;
}
