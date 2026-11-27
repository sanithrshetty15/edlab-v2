import { useCallback, useEffect, useState } from "react";
import { attendanceTerminalsApi } from "../../api/services";
import {
  Alert,
  Badge,
  Spinner,
  EmptyState,
} from "./Ui";
import { ShieldIcon } from "./AppIcons";
import { ThreeDotLoader } from "./ThreeDotLoader";
import { formatRelative } from "../../utils/format";

const TERMINAL_ONLINE_TONE = "success";
const TERMINAL_OFFLINE_TONE = "danger";

function ReaderStatusPanel({ pollingInterval = 5000 }) {
  const [terminals, setTerminals] = useState([]);
  const [terminalError, setTerminalError] = useState(null);
  const [isLoading, setIsLoading] = useState(true);

  const poll = useCallback(async () => {
    setIsLoading(true);
    setTerminalError(null);
    try {
      const result = await attendanceTerminalsApi.list({ limit: 1 });
      setTerminals(result.terminals || []);
    } catch (error) {
      setTerminalError(error);
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    poll();
    const timer = window.setInterval(poll, pollingInterval);
    return () => window.clearInterval(timer);
  }, [poll, pollingInterval]);

  if (terminals.length === 0 && !terminalError && isLoading) {
    return <Spinner rows={1} compact />;
  }

  if (terminalError) {
    return (
      <Alert tone="danger" title="Terminal status unavailable">
        {terminalError.message || "Could not load terminal status"}
      </Alert>
    );
  }

  if (terminals.length === 0) {
    return (
      <EmptyState
        icon={ShieldIcon}
        title="No attendance terminals"
        description="Provision an attendance device with device_type attendance_terminal."
      />
    );
  }

  const terminal = terminals[0];
  const reader = terminal.reader;
  const status = terminal.terminal_status;
  const online = Boolean(terminal.online);
  const readerConnected = reader?.connected;
  const readerReadiness = reader?.readiness;
  const lastSeen = reader?.last_seen_at;
  const slotsUsed = terminal.slots_used ?? 0;
  const slotCapacity = terminal.slot_capacity ?? 1000;
  const slotsAvailable = Math.max(0, slotCapacity - slotsUsed);

  const isReady = readerReadiness === "ready";
  const isOnline = online && readerConnected && isReady;

  const getReaderStatusTone = () => {
    if (!online) return TERMINAL_OFFLINE_TONE;
    if (!readerConnected) return TERMINAL_OFFLINE_TONE;
    if (isReady) return TERMINAL_ONLINE_TONE;
    return "warning";
  };

  const getReaderStatusLabel = () => {
    if (!online) return "Offline";
    if (!readerConnected) return "Reader disconnected";
    if (isReady) return "Reader ready";
    if (readerReadiness === "enrolling") return "Enrolling";
    if (readerReadiness === "not_ready") return "Not ready";
    return readerReadiness || "Unknown";
  };

  const readerTone = getReaderStatusTone();
  const readerLabel = getReaderStatusLabel();

  const readinessClass = isReady ? "lab-reader-status--ready" : "";
  const labelClass = isReady ? "lab-reader-status--label-ready" : "";

  const readyDotStyle = {
    animation: isReady ? "pulse 2s ease infinite" : "none",
  };

  const statusMessage = isOnline
    ? <p className="lab-reader-status__ready-message">Reader is ready, you can register a fingerprint</p>
    : <p className="lab-reader-status__waiting-message">Waiting for the reader</p>;

  return (
    <div className="lab-reader-status-panel">
      <div className="lab-reader-status-panel__head">
        <div>
          <strong>{terminal.name || terminal.device_uid || "Attendance terminal"}</strong>
          <Badge tone={online ? TERMINAL_ONLINE_TONE : TERMINAL_OFFLINE_TONE} dot>
            {online ? "Online" : "Offline"}
          </Badge>
        </div>
        <div>
          <Badge tone={readerTone} className={readinessClass} dot>
            {readerLabel}
          </Badge>
        </div>
      </div>

      <div className="lab-reader-status-panel__body">
        {lastSeen && (
          <div className="lab-reader-status-panel__last-seen">
            <span>Last seen</span>
            <strong>{formatRelative(lastSeen)}</strong>
          </div>
        )}

        <div className="lab-reader-status-panel__slots">
          <span>Slots</span>
          <strong>{slotsAvailable} free of {slotCapacity}</strong>
        </div>
      </div>

      {isOnline && statusMessage}

      {!isOnline && !isLoading && (
        <ThreeDotLoader
          size="sm"
          tone="dark"
          label="Loading"
          className="lab-reader-status__loader"
        />
      )}
    </div>
  );
}

export default function useReaderStatus(pollingInterval = 5000) {
  const [terminals, setTerminals] = useState([]);
  const [terminalError, setTerminalError] = useState(null);
  const [isLoading, setIsLoading] = useState(true);

  const poll = useCallback(async () => {
    try {
      const result = await attendanceTerminalsApi.list({ limit: 1 });
      setTerminals(result.terminals || []);
      setTerminalError(null);
    } catch (error) {
      setTerminalError(error);
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    poll();
    const timer = window.setInterval(poll, pollingInterval);
    return () => window.clearInterval(timer);
  }, [poll, pollingInterval]);

  return { terminals, terminalError, isLoading };
}

export { ReaderStatusPanel };
