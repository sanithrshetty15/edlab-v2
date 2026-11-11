import { useEffect, useState } from "react";
import { Network } from "@capacitor/network";

function browserStatus() {
  if (typeof navigator === "undefined") return { connected: true, connectionType: "unknown" };
  return {
    connected: navigator.onLine !== false,
    connectionType: navigator.connection?.type || "unknown",
  };
}

export function useNetworkStatus() {
  const [status, setStatus] = useState(browserStatus);

  useEffect(() => {
    let active = true;
    let listenerHandle = null;

    Network.getStatus()
      .then((next) => {
        if (active) setStatus(next);
      })
      .catch(() => {
        if (active) setStatus(browserStatus());
      });

    Network.addListener("networkStatusChange", (next) => {
      if (active) setStatus(next);
    })
      .then((handle) => {
        if (active) listenerHandle = handle;
        else handle?.remove();
      })
      .catch(() => {});

    const handleOnline = () => setStatus((current) => ({ ...current, connected: true }));
    const handleOffline = () => setStatus((current) => ({ ...current, connected: false }));
    window.addEventListener("online", handleOnline);
    window.addEventListener("offline", handleOffline);

    return () => {
      active = false;
      listenerHandle?.remove();
      window.removeEventListener("online", handleOnline);
      window.removeEventListener("offline", handleOffline);
    };
  }, []);

  return status;
}
