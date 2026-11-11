import { useEffect, useState } from "react";
import { applyServiceWorkerUpdate } from "../../pwa/registerServiceWorker";

export default function PwaUpdateBanner() {
  const [visible, setVisible] = useState(false);
  const [dismissed, setDismissed] = useState(false);

  useEffect(() => {
    const onUpdate = () => {
      setVisible(true);
      setDismissed(false);
    };

    window.addEventListener("edwinslab:update-available", onUpdate);
    return () => window.removeEventListener("edwinslab:update-available", onUpdate);
  }, []);

  if (!visible || dismissed) return null;

  return (
    <div className="pwa-update" role="status" aria-live="polite">
      <div className="pwa-update__inner">
        <p className="pwa-update__copy">
          <span className="pwa-update__dot" aria-hidden="true" />
          A new version of Edwins Lab is ready.
        </p>
        <div className="pwa-update__actions">
          <button
            type="button"
            className="pwa-update__reload"
            onClick={() => applyServiceWorkerUpdate()}
          >
            Reload
          </button>
          <button
            type="button"
            className="pwa-update__dismiss"
            onClick={() => setDismissed(true)}
          >
            Later
          </button>
        </div>
      </div>
    </div>
  );
}
