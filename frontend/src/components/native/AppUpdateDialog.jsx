import { useMemo, useState } from "react";
import { Modal } from "../app/Modal";
import { AppButton } from "../app/Ui";
import { downloadReleaseApk, installReleaseApk } from "../../services/appUpdater";

const ERROR_MESSAGES = {
  checksum_mismatch:
    "The downloaded file did not match its SHA-256 checksum and was discarded. Please download the update again.",
  corrupted_apk: "The downloaded APK is corrupted. Please download the update again.",
  wrong_package: "The published update does not belong to Edwins Lab.",
  manifest_unavailable: "The update manifest could not be reached. Please try again later.",
  invalid_manifest: "The published release information is invalid, so the update was stopped.",
  offline: "You are offline. Connect to the internet and try again.",
  download_failed: "The update could not be downloaded. Check your connection and try again.",
  install_permission_required:
    "Android needs permission to install Edwins Lab updates. Allow “Install unknown apps” for Edwins Lab in the settings screen that opened, then try again.",
  installer_unavailable: "No Android package installer is available on this device.",
  install_failed: "The Android installer could not be opened. Please try again.",
};

function formatSize(bytes) {
  if (!Number.isFinite(bytes) || bytes <= 0) return null;
  if (bytes < 1024 * 1024) return `${Math.round(bytes / 1024)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

function releaseMessage(error) {
  if (!error) return "The update could not be completed. Please try again.";
  return ERROR_MESSAGES[error.code] || error.message || "The update could not be completed.";
}

export default function AppUpdateDialog({ manifest, mandatory = false, onDismiss }) {
  const [phase, setPhase] = useState("idle");
  const [progress, setProgress] = useState(0);
  const [error, setError] = useState(null);
  const [filePath, setFilePath] = useState(null);

  const sizeLabel = useMemo(() => formatSize(manifest?.sizeBytes), [manifest]);

  const busy = phase === "downloading" || phase === "installing";
  const dismissible = !mandatory && !busy;

  const install = async (path) => {
    setPhase("installing");
    setError(null);
    try {
      await installReleaseApk(path);
      setPhase("install_prompt");
    } catch (installError) {
      setError(installError);
      setPhase(
        installError?.code === "install_permission_required" ? "permission" : "error",
      );
    }
  };

  const startUpdate = async () => {
    if (!manifest) return;

    if (filePath) {
      await install(filePath);
      return;
    }

    setPhase("downloading");
    setProgress(0);
    setError(null);

    try {
      const result = await downloadReleaseApk({
        url: manifest.apkUrl,
        sha256: manifest.sha256,
        fileName: manifest.fileName,
        onProgress: setProgress,
      });
      setProgress(100);
      setFilePath(result.filePath);
      await install(result.filePath);
    } catch (downloadError) {
      setError(downloadError);
      setPhase("error");
    }
  };

  const footer = (
    <>
      {!mandatory && !busy && phase !== "install_prompt" && (
        <AppButton variant="ghost" onClick={onDismiss}>
          Later
        </AppButton>
      )}
      {phase === "install_prompt" ? (
        <AppButton variant="primary" onClick={onDismiss}>
          Done
        </AppButton>
      ) : (
        <AppButton
          variant="primary"
          onClick={startUpdate}
          loading={busy}
          disabled={busy}
        >
          {phase === "downloading"
            ? `Downloading… ${progress}%`
            : phase === "installing"
              ? "Opening installer…"
              : phase === "permission"
                ? "Try installation again"
                : phase === "error"
                  ? "Try again"
                  : "Update Now"}
        </AppButton>
      )}
    </>
  );

  return (
    <Modal
      open
      onClose={onDismiss}
      dismissible={dismissible}
      title={`Edwins Lab ${manifest.versionName} is available`}
      description={
        mandatory
          ? "This update is required to keep using the app."
          : "Update to get the latest fixes and improvements."
      }
      size="sm"
      footer={footer}
      closeLabel="Close update dialog"
    >
      <div className="app-update">
        <dl className="app-update__meta">
          <div>
            <dt>New version</dt>
            <dd>
              {manifest.versionName} (build {manifest.versionCode})
            </dd>
          </div>
          {sizeLabel && (
            <div>
              <dt>Download size</dt>
              <dd>{sizeLabel}</dd>
            </div>
          )}
        </dl>

        {mandatory && <p className="app-update__mandatory">Required update</p>}

        {manifest.releaseNotes?.length > 0 && (
          <div className="app-update__notes">
            <p className="app-update__notes-title">What&apos;s new</p>
            <ul>
              {manifest.releaseNotes.map((note) => (
                <li key={note}>{note}</li>
              ))}
            </ul>
          </div>
        )}

        {phase === "downloading" && (
          <div className="app-update__progress" aria-live="polite">
            <div
              className="app-update__progress-track"
              role="progressbar"
              aria-valuemin={0}
              aria-valuemax={100}
              aria-valuenow={progress}
              aria-label="Update download progress"
            >
              <span
                className="app-update__progress-bar"
                style={{ width: `${progress}%` }}
              />
            </div>
            <p className="app-update__progress-label">
              Downloading update… {progress}%
            </p>
            <p className="app-update__hint">
              The file is checked against its SHA-256 checksum before Android installs it.
            </p>
          </div>
        )}

        {phase === "installing" && (
          <p className="app-update__hint" aria-live="polite">
            Opening Android&apos;s package installer…
          </p>
        )}

        {phase === "permission" && (
          <p className="app-update__error" role="alert">
            {releaseMessage(error)}
          </p>
        )}

        {phase === "error" && (
          <p className="app-update__error" role="alert">
            {releaseMessage(error)}
          </p>
        )}

        {phase === "install_prompt" && (
          <p className="app-update__hint" aria-live="polite">
            Android is asking you to confirm the installation. Approve it in the system
            dialog; Edwins Lab only opens the installer and never installs silently.
          </p>
        )}
      </div>
    </Modal>
  );
}
