import { useState } from "react";
import AppUpdateDialog from "./AppUpdateDialog";
import { useAppUpdate } from "../../hooks/useAppUpdate";

/**
 * Checks for a newer published release once the intro has finished.
 * Renders nothing when the manifest is unreachable or the app is current.
 */
export default function AppUpdateGate({ enabled = true }) {
  const { status, manifest, mandatory } = useAppUpdate(enabled);
  const [dismissed, setDismissed] = useState(false);

  if (!enabled || dismissed || status !== "available" || !manifest) return null;

  return (
    <AppUpdateDialog
      manifest={manifest}
      mandatory={mandatory}
      onDismiss={() => setDismissed(true)}
    />
  );
}
