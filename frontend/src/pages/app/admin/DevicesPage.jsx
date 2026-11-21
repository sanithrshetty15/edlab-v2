import { useCallback, useEffect, useState } from "react";
import { devicesApi } from "../../../api/services";
import {
  Alert,
  AppButton,
  Badge,
  EmptyState,
  ErrorState,
  LoadingState,
  PageHeader,
  Panel,
} from "../../../components/app/Ui";
import { RefreshIcon, ShieldIcon } from "../../../components/app/AppIcons";
import { formatDateTime, formatRelative } from "../../../utils/format";

// The backend records a heartbeat on every device request and the firmware
// beats every 30s, so a 4s poll is fresh without being wasteful.
const POLL_INTERVAL_MS = 4000;

const READINESS_LABELS = {
  not_ready: { label: "Not ready", tone: "danger" },
  initializing: { label: "Initializing", tone: "info" },
  ready: { label: "Ready", tone: "success" },
  waiting_for_finger: { label: "Waiting for finger", tone: "accent" },
  finger_detected: { label: "Finger detected", tone: "accent" },
  capture_failed: { label: "Capture failed", tone: "warning" },
  enrolling: { label: "Enrolling", tone: "accent" },
};

const ENROLLMENT_LABELS = {
  awaiting_sensor: "Waiting for the ESP32 to claim the session",
  sensor_initializing: "ESP32 acknowledged the command and is initializing the reader",
  waiting_first_scan: "Place the finger for the first scan",
  first_scan_captured: "First scan captured",
  waiting_for_removal: "Remove the finger",
  waiting_second_scan: "Place the same finger again",
  second_scan_captured: "Second scan captured",
  verifying: "Matching samples and storing the template",
  finalized: "Registration successful",
  failed: "Registration failed",
  cancelled: "Enrollment cancelled",
  expired: "Enrollment expired",
};

function readinessFor(reader, enrollment) {
  if (!reader.available) {
    return { label: "Disconnected", tone: "danger" };
  }
  if (enrollment?.is_active && enrollment.reader_key === reader.key) {
    // Never report the reader as ready merely because it was ready before the
    // enrollment started; the enrollment session owns it now.
    return { label: "Enrolling", tone: "accent" };
  }
  return READINESS_LABELS[reader.readiness] || READINESS_LABELS.not_ready;
}

function DeviceCard({ device, stale }) {
    const [rotating, setRotating] = useState(false);
  const [newApiKey, setNewApiKey] = useState("");
  const [rotateError, setRotateError] = useState("");

  const handleRotateKey = async () => {
    const confirmed = window.confirm(
      "Rotate this device API key? The old key will stop working immediately.",
    );
    if (!confirmed) return;

    setRotating(true);
    setRotateError("");
    setNewApiKey("");

    try {
      const result = await devicesApi.rotateKey(device.id);
      setNewApiKey(result.api_key);
    } catch (err) {
      setRotateError(err?.message || "Failed to rotate device API key.");
    } finally {
      setRotating(false);
    }
  };
  const readers = device.readers || [];
  const enrollment = device.enrollment || null;
  const online = Boolean(device.online);
  const lastReport = readers
    .map((reader) => reader.last_seen_at)
    .filter(Boolean)
    .sort()
    .at(-1);

  return (
    <Panel
      title={device.name || device.device_uid}
      subtitle={`${device.device_uid} · ${device.device_type}`}
      actions={
        <Badge tone={online ? "success" : "danger"} dot>
          {stale
            ? `Last known · ${online ? "online" : "offline"}`
            : online
              ? "ESP32 online"
              : "ESP32 offline"}
        </Badge>
      }
    >
      <div className="lab-device-stack">
        {!online && (
          <Alert tone="danger" title="No heartbeat received">
            This controller is offline. Last contact{" "}
            {device.last_seen_at ? formatRelative(device.last_seen_at) : "never"}. Reader
            values below are the last report and must not be treated as current.
          </Alert>
        )}

        <div className="lab-device-meta">
          <div>
            <span className="lab-device-meta__label">Last heartbeat</span>
            <strong>{device.last_seen_at ? formatRelative(device.last_seen_at) : "Never"}</strong>
          </div>
          <div>
            <span className="lab-device-meta__label">Last reader report</span>
            <strong>{lastReport ? formatRelative(lastReport) : "Never"}</strong>
          </div>
          <div>
            <span className="lab-device-meta__label">Reported at</span>
            <strong>{lastReport ? formatDateTime(lastReport) : "—"}</strong>
          </div>
        </div>

        {readers.length === 0 ? (
          <p className="lab-muted lab-small">
            This controller has not reported a fingerprint reader yet.
          </p>
        ) : (
          <ul className="lab-device-readers">
            {readers.map((reader) => {
              const status = readinessFor(reader, enrollment);
              return (
                <li key={reader.key} className="lab-device-reader">
                  <span className="lab-device-reader__icon" aria-hidden="true">
                    <ShieldIcon className="h-4 w-4" />
                  </span>
                  <div className="lab-device-reader__body">
                    <strong>{reader.label || reader.key}</strong>
                    <span className="lab-device-reader__meta">
                      UART {reader.uart} · {reader.key}
                    </span>
                  </div>
                  <div className="lab-device-reader__status">
                    <Badge
                      tone={reader.available ? "success" : "danger"}
                      dot
                    >
                      {reader.available ? "Connected" : "Disconnected"}
                    </Badge>
                    <Badge tone={status.tone}>{status.label}</Badge>
                  </div>
                </li>
              );
            })}
          </ul>
        )}

        {enrollment?.is_active ? (
          <div className="lab-enrollment-status">
            <div className="lab-enrollment-status__icon">
              <ShieldIcon className="h-5 w-5" />
            </div>
            <div>
              <div className="lab-enrollment-status__title">
                <h3>Enrollment in progress</h3>
                <Badge tone="warning">{String(enrollment.status).replaceAll("_", " ")}</Badge>
              </div>
              <p>
                {ENROLLMENT_LABELS[enrollment.status] || "Working on the enrollment session"} ·{" "}
                {enrollment.member_name || "Member"}
                {enrollment.fingerprint_slot ? ` · slot ${enrollment.fingerprint_slot}` : ""}
              </p>
              <p>Updated {formatRelative(enrollment.updated_at)}.</p>
            </div>
          </div>
        ) : null}
      </div>
              <div className="lab-device-actions" style={{ marginTop: "1rem" }}>
          <AppButton
            variant="outline"
            size="sm"
            onClick={handleRotateKey}
            loading={rotating}
            disabled={rotating}
          >
            Rotate API Key
          </AppButton>

          {rotateError && <Alert tone="danger">{rotateError}</Alert>}

          {newApiKey && (
            <Alert tone="warning" title="New API key — copy and store it now">
              <div style={{ wordBreak: "break-all", marginBottom: "0.75rem" }}>
                {newApiKey}
              </div>
              <AppButton
                variant="outline"
                size="sm"
                onClick={() => navigator.clipboard.writeText(newApiKey)}
              >
                Copy API Key
              </AppButton>
            </Alert>
          )}
        </div>
    </Panel>
  );
}

export default function DevicesPage() {
  const [data, setData] = useState(null);
  const [error, setError] = useState(null);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [visible, setVisible] = useState(
    () => typeof document === "undefined" || !document.hidden,
  );
  const [nonce, setNonce] = useState(0);

  useEffect(() => {
    if (typeof document === "undefined") return undefined;
    const handleVisibility = () => setVisible(!document.hidden);
    document.addEventListener("visibilitychange", handleVisibility);
    return () => document.removeEventListener("visibilitychange", handleVisibility);
  }, []);

  useEffect(() => {
    if (!visible) return undefined;

    let stopped = false;
    let timer = null;
    let controller = null;

    const poll = async () => {
      controller = new AbortController();
      setRefreshing(true);
      try {
        const result = await devicesApi.list({ signal: controller.signal });
        if (stopped) return;
        setData(result);
        setError(null);
      } catch (pollError) {
        if (stopped || pollError?.code === "aborted") return;
        setError(pollError);
      } finally {
        if (!stopped) {
          setLoading(false);
          setRefreshing(false);
          timer = window.setTimeout(poll, POLL_INTERVAL_MS);
        }
      }
    };

    poll();
    return () => {
      stopped = true;
      window.clearTimeout(timer);
      controller?.abort();
    };
  }, [visible, nonce]);

  const refresh = useCallback(() => setNonce((current) => current + 1), []);
  const devices = data?.devices || [];
  const threshold = data?.online_threshold_seconds;

  return (
    <>
      <PageHeader
        eyebrow="Access control"
        title="Devices"
        description={
          threshold
            ? `Live controller availability, refreshed every ${POLL_INTERVAL_MS / 1000}s. A device is offline after ${threshold}s without a heartbeat.`
            : `Live controller availability, refreshed every ${POLL_INTERVAL_MS / 1000}s.`
        }
        actions={
          <AppButton variant="outline" onClick={refresh} loading={refreshing && !loading}>
            <RefreshIcon className="h-4 w-4" />
            Refresh now
          </AppButton>
        }
      />

      {error && data ? (
        <Alert tone="warning" title="Live updates paused">
          {error.message} The values below are the last known snapshot from{" "}
          {formatRelative(data.server_time)} and may be out of date.
        </Alert>
      ) : null}

      {loading && !data ? (
        <LoadingState label="Loading device availability" rows={3} />
      ) : error && !data ? (
        <ErrorState
          error={error}
          title="Device availability is unavailable"
          onRetry={refresh}
        />
      ) : devices.length === 0 ? (
        <EmptyState
          icon={ShieldIcon}
          title="No devices registered"
          description="Provision an ESP32 controller to see its online state, reader readiness and enrollment progress here."
        />
      ) : (
        <div className="lab-device-grid">
          {devices.map((device) => (
            <DeviceCard key={device.id} device={device} stale={Boolean(error)} />
          ))}
        </div>
      )}
    </>
  );
}
