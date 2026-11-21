import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { attendanceTerminalsApi, membersManagementApi } from "../../../api/services";
import { useApiResource } from "../../../hooks/useApiResource";
import { ConfirmDialog, Modal } from "../../../components/app/Modal";
import {
  Alert,
  AppButton,
  Badge,
  EmptyState,
  ErrorState,
  Field,
  LoadingState,
  PageHeader,
  Panel,
  Select,
  StatCard,
  TextInput,
} from "../../../components/app/Ui";
import {
  AlertIcon,
  CheckCircleIcon,
  ClockIcon,
  PlusIcon,
  RefreshIcon,
  ShieldIcon,
  TrashIcon,
} from "../../../components/app/AppIcons";
import { formatDateTime, formatRelative, formatTime, titleCase } from "../../../utils/format";

const FEED_POLL_MS = 3000;
const TERMINAL_POLL_MS = 5000;

const OUTCOME_META = {
  entry: { label: "Entry", tone: "success", icon: CheckCircleIcon, iconTone: "accent" },
  exit: { label: "Exit", tone: "info", icon: ClockIcon, iconTone: "info" },
  duplicate: { label: "Duplicate", tone: "warning", icon: ClockIcon, iconTone: "warning" },
  no_match: { label: "Not recognized", tone: "danger", icon: AlertIcon, iconTone: "warning" },
  unknown_slot: { label: "Unmapped slot", tone: "danger", icon: AlertIcon, iconTone: "warning" },
  member_inactive: { label: "Member inactive", tone: "danger", icon: AlertIcon, iconTone: "warning" },
  user_disabled: { label: "Account disabled", tone: "danger", icon: AlertIcon, iconTone: "warning" },
};

const OUTCOME_FALLBACK = {
  no_match: "Fingerprint not recognized",
  unknown_slot: "Fingerprint not enrolled on this terminal",
  member_inactive: "Inactive member",
  user_disabled: "Disabled member account",
};

const READINESS_LABELS = {
  not_ready: "Not ready",
  initializing: "Initializing",
  ready: "Ready",
  waiting_for_finger: "Waiting for finger",
  finger_detected: "Finger detected",
  capture_failed: "Capture failed",
  enrolling: "Enrolling",
};

const SESSION_LABELS = {
  pending: "Waiting for the terminal to collect this command",
  in_progress: "Terminal is working on this command",
  waiting_for_finger: "Place the member's finger on the terminal",
  finger_detected: "Finger detected",
  captured: "Fingerprint captured",
  completed: "Completed",
  failed: "Failed",
  cancelled: "Cancelled",
  expired: "Expired",
};

function sessionTone(status) {
  switch (status) {
    case "completed":
      return "success";
    case "failed":
      return "danger";
    case "cancelled":
    case "expired":
      return "neutral";
    default:
      return "warning";
  }
}

function LiveEvent({ event }) {
  const meta = OUTCOME_META[event.outcome] || {
    label: titleCase(event.outcome || "event"),
    tone: "neutral",
    icon: ClockIcon,
    iconTone: "neutral",
  };
  const Icon = meta.icon;
  const member = event.member;
  const secondary = member
    ? [member.login_id, member.usn].filter(Boolean).join(" · ")
    : event.member_identifier || "No member matched";
  const terminalName = event.terminal?.name || event.terminal?.device_uid || "Attendance terminal";

  return (
    <li className="lab-feed__item">
      <span className={`lab-activity__icon lab-activity__icon--${meta.iconTone}`} aria-hidden="true">
        <Icon className="h-4 w-4" />
      </span>
      <div className="lab-feed__copy">
        <div className="lab-live-event__title">
          <strong>{member?.name || OUTCOME_FALLBACK[event.outcome] || "Attendance event"}</strong>
          <Badge tone={meta.tone}>{meta.label}</Badge>
          {event.synced_late && <Badge tone="neutral">Synced late</Badge>}
        </div>
        <p>{secondary}</p>
        <p>
          {terminalName}
          {member?.academic_year ? ` · Year ${member.academic_year}` : ""}
          {event.fingerprint_slot ? ` · slot ${event.fingerprint_slot}` : ""}
        </p>
      </div>
      <div className="lab-live-event__time">
        <time dateTime={event.occurred_at}>{formatTime(event.occurred_at)}</time>
        <span>{formatRelative(event.occurred_at)}</span>
      </div>
    </li>
  );
}

function TerminalCard({ terminal, onEnroll, onMappings }) {
  const status = terminal.terminal_status;
  const reader = terminal.reader;
  const online = Boolean(terminal.online);
  const session = terminal.active_session;
  const cleanup = terminal.cleanup_session;

  return (
    <article className="lab-terminal-card">
      <div className="lab-terminal-card__head">
        <div>
          <strong>{terminal.name || terminal.device_uid}</strong>
          <span>{terminal.device_uid}</span>
        </div>
        <Badge tone={online ? "success" : "danger"} dot>
          {online ? "Online" : "Offline"}
        </Badge>
      </div>

      <div className="lab-device-meta">
        <div>
          <span className="lab-device-meta__label">Last seen</span>
          <strong>{terminal.last_seen_at ? formatRelative(terminal.last_seen_at) : "Never"}</strong>
        </div>
        <div>
          <span className="lab-device-meta__label">Mappings</span>
          <strong>{terminal.mapping_count ?? 0} fingerprints</strong>
        </div>
        <div>
          <span className="lab-device-meta__label">Pending queue</span>
          <strong>{status?.queue_depth ?? 0} events</strong>
        </div>
      </div>

      <div className="lab-terminal-card__status">
        <Badge tone={reader?.available ? "success" : reader?.connected ? "warning" : "danger"} dot>
          {reader?.connected ? "Reader connected" : "Reader disconnected"}
        </Badge>
        {reader?.readiness && <Badge tone="neutral">{READINESS_LABELS[reader.readiness] || reader.readiness}</Badge>}
        {status?.firmware_version && <Badge tone="neutral">FW {status.firmware_version}</Badge>}
        <Badge tone={online ? "neutral" : "warning"}>
          {terminal.terminal_status?.stale ? "Status stale" : `Updated ${formatRelative(status?.reported_at || status?.updated_at)}`}
        </Badge>
      </div>

      {session?.is_active && (
        <Alert tone="info" title={`${titleCase(session.operation)} in progress`}>
          {session.member_name || "Member"}
          {session.fingerprint_slot ? ` · slot ${session.fingerprint_slot}` : ""} ·{" "}
          {SESSION_LABELS[session.status] || session.status}
        </Alert>
      )}
      {cleanup && !session?.is_active && (
        <Alert tone="warning" title="Cleanup pending">
          The terminal must delete an unfinished template before another enrollment can start.
        </Alert>
      )}
      {!reader?.connected && (
        <p className="lab-muted lab-small">
          This terminal has not reported a connected fingerprint reader. Enrollment waits until it does.
        </p>
      )}

      <div className="lab-terminal-card__actions">
        <AppButton variant="primary" size="sm" onClick={() => onEnroll(terminal)}>
          <PlusIcon className="h-4 w-4" />
          Enroll fingerprint
        </AppButton>
        <AppButton variant="outline" size="sm" onClick={() => onMappings(terminal)}>
          <ShieldIcon className="h-4 w-4" />
          Mappings
        </AppButton>
      </div>
    </article>
  );
}

function EnrollmentDialog({ terminal, members, onClose, onChanged }) {
  const [memberId, setMemberId] = useState("");
  const [slot, setSlot] = useState("");
  const [session, setSession] = useState(null);
  const [error, setError] = useState(null);
  const [notice, setNotice] = useState(null);
  const [starting, setStarting] = useState(false);
  const [cancelling, setCancelling] = useState(false);

  const active = Boolean(session?.is_active);

  useEffect(() => {
    if (!session?.session_uid || !session.is_active) return undefined;
    let stopped = false;
    let timer = null;

    const poll = async () => {
      try {
        const result = await attendanceTerminalsApi.getEnrollment(terminal.id, session.session_uid);
        if (stopped) return;
        setSession(result.enrollment);
        setError(null);
        if (result.enrollment?.status === "completed") {
          setNotice("Fingerprint enrolled. The device-specific mapping is saved.");
          onChanged?.();
        } else if (result.enrollment?.is_active) {
          timer = window.setTimeout(poll, 2000);
        }
      } catch (pollError) {
        if (stopped) return;
        setError(pollError);
        timer = window.setTimeout(poll, 4000);
      }
    };

    timer = window.setTimeout(poll, 1200);
    return () => {
      stopped = true;
      window.clearTimeout(timer);
    };
  }, [session?.session_uid, session?.is_active, terminal.id, onChanged]);

  const start = async (event) => {
    event.preventDefault();
    if (!memberId) {
      setError(new Error("Choose the member who will use this fingerprint."));
      return;
    }
    setStarting(true);
    setError(null);
    setNotice(null);
    try {
      const payload = { member_id: Number(memberId) };
      if (slot.trim()) payload.fingerprint_slot = Number(slot);
      const result = await attendanceTerminalsApi.startEnrollment(terminal.id, payload);
      setSession(result.enrollment);
      onChanged?.();
    } catch (startError) {
      setError(startError);
    } finally {
      setStarting(false);
    }
  };

  const cancel = async () => {
    if (!session?.session_uid) return;
    setCancelling(true);
    setError(null);
    try {
      const result = await attendanceTerminalsApi.cancelEnrollment(terminal.id, session.session_uid);
      setSession(result.enrollment);
      setNotice("Enrollment cancelled.");
      onChanged?.();
    } catch (cancelError) {
      setError(cancelError);
    } finally {
      setCancelling(false);
    }
  };

  const selectedMember = members.find((member) => String(member.id) === String(memberId));

  return (
    <Modal
      open
      onClose={onClose}
      title={`Enroll fingerprint · ${terminal.name || terminal.device_uid}`}
      description="The fingerprint template stays inside this terminal's reader. Edwins Lab stores only the device-specific slot mapping."
      footer={
        <>
          {session && !active && (
            <AppButton
              variant="outline"
              onClick={() => {
                setSession(null);
                setNotice(null);
                setError(null);
              }}
            >
              Enroll another
            </AppButton>
          )}
          {active ? (
            <AppButton variant="ghost" onClick={cancel} loading={cancelling} disabled={starting}>
              Cancel enrollment
            </AppButton>
          ) : (
            <AppButton variant={session ? "primary" : "ghost"} onClick={onClose}>
              Close
            </AppButton>
          )}
          {!session && (
            <AppButton form="attendance-enroll-form" type="submit" variant="primary" loading={starting}>
              Start enrollment
            </AppButton>
          )}
        </>
      }
    >
      <div className="lab-form-stack">
        {error && (
          <Alert tone="danger" title="Enrollment problem">
            {error.message}
          </Alert>
        )}
        {notice && (
          <Alert tone={session?.status === "failed" ? "danger" : "success"} title="Enrollment status">
            {notice}
          </Alert>
        )}

        {session ? (
          <div className="lab-enrollment-status">
            <div className="lab-enrollment-status__icon">
              <ShieldIcon className="h-5 w-5" />
            </div>
            <div>
              <div className="lab-enrollment-status__title">
                <h3>{titleCase(session.operation)} fingerprint</h3>
                <Badge tone={sessionTone(session.status)} dot>
                  {titleCase(session.status)}
                </Badge>
              </div>
              <p>{SESSION_LABELS[session.status] || session.status}</p>
              <p>
                {session.member_name || selectedMember?.name || "Member"}
                {session.fingerprint_slot ? ` · slot ${session.fingerprint_slot}` : ""}
                {session.claimed_at ? ` · claimed ${formatRelative(session.claimed_at)}` : ""}
              </p>
              {session.error_message && <p>{session.error_message}</p>}
            </div>
          </div>
        ) : (
          <form id="attendance-enroll-form" className="lab-form-stack" onSubmit={start} noValidate>
            <Field label="Member" htmlFor="attendance-enroll-member" required hint="Only active members can be enrolled.">
              <Select
                id="attendance-enroll-member"
                value={memberId}
                onChange={(event) => setMemberId(event.target.value)}
                disabled={starting}
              >
                <option value="">Select an active member</option>
                {members.map((member) => (
                  <option key={member.id} value={member.id}>
                    {member.name} · {member.login_id}
                  </option>
                ))}
              </Select>
            </Field>
            <Field
              label="Fingerprint slot"
              htmlFor="attendance-enroll-slot"
              hint="Optional. Leave blank to allocate the next free slot on this terminal (1–127)."
            >
              <TextInput
                id="attendance-enroll-slot"
                type="number"
                min="1"
                max="127"
                value={slot}
                onChange={(event) => setSlot(event.target.value)}
                disabled={starting}
              />
            </Field>
            <p className="lab-muted lab-small">
              The terminal takes over from here. The member scans twice (or as its reader requires) and the slot is
              confirmed automatically.
            </p>
          </form>
        )}
      </div>
    </Modal>
  );
}

function MappingsDialog({ terminal, onClose, onChanged }) {
  const [mappings, setMappings] = useState(null);
  const [error, setError] = useState(null);
  const [loading, setLoading] = useState(true);
  const [removeTarget, setRemoveTarget] = useState(null);
  const [removing, setRemoving] = useState(false);
  const [queued, setQueued] = useState(null);

  const load = useCallback(async () => {
    try {
      const result = await attendanceTerminalsApi.mappings(terminal.id);
      setMappings(result.mappings || []);
      setError(null);
    } catch (loadError) {
      setError(loadError);
    } finally {
      setLoading(false);
    }
  }, [terminal.id]);

  useEffect(() => {
    const timer = window.setTimeout(load, 0);
    return () => window.clearTimeout(timer);
  }, [load]);

  const confirmRemove = async () => {
    if (!removeTarget) return;
    setRemoving(true);
    setError(null);
    try {
      const result = await attendanceTerminalsApi.startEnrollment(terminal.id, {
        member_id: removeTarget.member_id,
        operation: "delete",
      });
      setQueued(result.enrollment);
      setRemoveTarget(null);
      onChanged?.();
      await load();
    } catch (removeError) {
      setError(removeError);
    } finally {
      setRemoving(false);
    }
  };

  return (
    <>
      <Modal
        open
        onClose={onClose}
        title={`Mappings · ${terminal.name || terminal.device_uid}`}
        description="Slot numbers are specific to this terminal. The same slot on another reader can belong to a different member."
        size="lg"
        footer={
          <AppButton variant="ghost" onClick={onClose}>
            Close
          </AppButton>
        }
      >
        {error && (
          <Alert tone="danger" title="Mappings unavailable">
            {error.message}
          </Alert>
        )}
        {queued && (
          <Alert tone="warning" title="Removal queued">
            {queued.member_name || "The member"} is queued for removal. The mapping disappears after the terminal
            deletes the fingerprint template and confirms the command.
          </Alert>
        )}

        {loading ? (
          <LoadingState label="Loading mappings" rows={3} compact />
        ) : mappings?.length ? (
          <div className="lab-table-wrap">
            <table className="lab-table lab-table--compact">
              <thead>
                <tr>
                  <th>Member</th>
                  <th>Slot</th>
                  <th>Reader</th>
                  <th>Enrolled</th>
                  <th>Status</th>
                  <th aria-label="Actions" />
                </tr>
              </thead>
              <tbody>
                {mappings.map((mapping) => (
                  <tr key={mapping.id}>
                    <td>
                      <div className="lab-member-academic">
                        <strong>{mapping.member?.name || `Member ${mapping.member_id}`}</strong>
                        <small>{mapping.member?.login_id || mapping.member?.usn || "—"}</small>
                      </div>
                    </td>
                    <td>{mapping.fingerprint_slot ?? "—"}</td>
                    <td>{mapping.reader_key || "legacy"}</td>
                    <td>{formatDateTime(mapping.created_at)}</td>
                    <td>
                      <Badge tone={mapping.status === "enrolled" ? "success" : "warning"} dot>
                        {titleCase(mapping.status)}
                      </Badge>
                    </td>
                    <td>
                      <AppButton variant="ghost" size="sm" onClick={() => setRemoveTarget(mapping)}>
                        <TrashIcon className="h-4 w-4" />
                        Remove
                      </AppButton>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          <EmptyState
            icon={ShieldIcon}
            title="No fingerprints mapped"
            description="Enroll a fingerprint to create the first device-specific slot mapping."
          />
        )}
      </Modal>

      <ConfirmDialog
        open={Boolean(removeTarget)}
        onClose={() => setRemoveTarget(null)}
        onConfirm={confirmRemove}
        title="Remove this fingerprint mapping?"
        description={`Queue the terminal to delete the template for ${
          removeTarget?.member?.name || "this member"
        } (slot ${removeTarget?.fingerprint_slot ?? "—"}).`}
        confirmLabel="Queue removal"
        loading={removing}
        destructive
      />
    </>
  );
}

export default function LiveAttendancePage() {
  const [terminalsData, setTerminalsData] = useState(null);
  const [terminalsError, setTerminalsError] = useState(null);
  const [terminalsLoading, setTerminalsLoading] = useState(true);
  const [feed, setFeed] = useState([]);
  const [feedError, setFeedError] = useState(null);
  const [feedLoading, setFeedLoading] = useState(true);
  const [outcomeFilter, setOutcomeFilter] = useState("");
  const [terminalFilter, setTerminalFilter] = useState("");
  const [enrollTerminal, setEnrollTerminal] = useState(null);
  const [mappingsTerminal, setMappingsTerminal] = useState(null);
  const [terminalNonce, setTerminalNonce] = useState(0);
  const [feedNonce, setFeedNonce] = useState(0);
  const [visible, setVisible] = useState(
    () => typeof document === "undefined" || !document.hidden,
  );
  const latestRef = useRef(0);

  const membersResource = useApiResource(
    () => membersManagementApi.list({ per_page: 100, status: "active" }),
    "live-attendance-members",
  );
  const members = membersResource.data?.members ?? [];

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
      try {
        const params = latestRef.current
          ? { after_id: latestRef.current, limit: 50 }
          : { limit: 50 };
        const result = await attendanceTerminalsApi.live(params, { signal: controller.signal });
        if (stopped) return;
        const incoming = [...(result.events || [])].reverse();
        if (!latestRef.current) {
          setFeed(incoming);
        } else if (incoming.length) {
          setFeed((current) => [...incoming, ...current].slice(0, 120));
        }
        if (result.latest_id) latestRef.current = result.latest_id;
        setFeedError(null);
      } catch (pollError) {
        if (stopped || pollError?.code === "aborted") return;
        setFeedError(pollError);
      } finally {
        if (!stopped) {
          setFeedLoading(false);
          timer = window.setTimeout(poll, FEED_POLL_MS);
        }
      }
    };

    poll();
    return () => {
      stopped = true;
      window.clearTimeout(timer);
      controller?.abort();
    };
  }, [visible, feedNonce]);

  useEffect(() => {
    if (!visible) return undefined;
    let stopped = false;
    let timer = null;
    let controller = null;

    const poll = async () => {
      controller = new AbortController();
      try {
        const result = await attendanceTerminalsApi.list({ signal: controller.signal });
        if (stopped) return;
        setTerminalsData(result);
        setTerminalsError(null);
      } catch (pollError) {
        if (stopped || pollError?.code === "aborted") return;
        setTerminalsError(pollError);
      } finally {
        if (!stopped) {
          setTerminalsLoading(false);
          timer = window.setTimeout(poll, TERMINAL_POLL_MS);
        }
      }
    };

    poll();
    return () => {
      stopped = true;
      window.clearTimeout(timer);
      controller?.abort();
    };
  }, [visible, terminalNonce]);

  const refresh = useCallback(() => {
    latestRef.current = 0;
    setFeed([]);
    setFeedLoading(true);
    setFeedNonce((current) => current + 1);
    setTerminalNonce((current) => current + 1);
  }, []);

  const reloadTerminals = useCallback(() => {
    setTerminalNonce((current) => current + 1);
  }, []);

  const terminals = terminalsData?.terminals ?? [];
  const onlineCount = terminals.filter((terminal) => terminal.online).length;
  const mappingCount = terminals.reduce((total, terminal) => total + (terminal.mapping_count || 0), 0);

  const filteredEvents = useMemo(
    () =>
      feed.filter((event) => {
        if (outcomeFilter && event.outcome !== outcomeFilter) return false;
        if (terminalFilter && String(event.terminal?.id) !== String(terminalFilter)) return false;
        return true;
      }),
    [feed, outcomeFilter, terminalFilter],
  );

  const feedPaused = Boolean(feedError);

  return (
    <>
      <PageHeader
        eyebrow="Attendance terminals"
        title="Live attendance"
        description="Fingerprint scans appear here as the attendance terminal submits them. The server decides entry or exit using the same rules as the existing attendance system."
        actions={
          <>
            <Badge tone={feedPaused ? "warning" : "success"} dot>
              {feedPaused ? "Live paused" : "Live"}
            </Badge>
            <AppButton variant="outline" onClick={refresh} loading={feedLoading && !feed.length}>
              <RefreshIcon className="h-4 w-4" />
              Refresh
            </AppButton>
          </>
        }
      />

      {feedError && feed.length > 0 && (
        <Alert tone="warning" title="Live updates paused">
          {feedError.message} Showing the last events received. The feed retries automatically.
        </Alert>
      )}
      {terminalsError && (
        <Alert tone="warning" title="Terminal status paused">
          {terminalsError.message} Values below are the last known snapshot.
        </Alert>
      )}

      <div className="lab-stat-grid lab-stat-grid--3">
        <StatCard
          label="Terminals online"
          value={`${onlineCount}/${terminals.length}`}
          hint="Attendance readers reporting a heartbeat"
          icon={ShieldIcon}
          tone={terminals.length && onlineCount === terminals.length ? "accent" : "neutral"}
          loading={terminalsLoading && !terminalsData}
        />
        <StatCard
          label="Enrolled fingerprints"
          value={mappingCount}
          hint="Device-specific slot mappings"
          icon={CheckCircleIcon}
          loading={terminalsLoading && !terminalsData}
        />
        <StatCard
          label="Events loaded"
          value={feed.length}
          hint="Newest 120 attendance events"
          icon={ClockIcon}
          loading={feedLoading && !feed.length}
        />
      </div>

      <div className="lab-dashboard-grid lab-dashboard-grid--admin">
        <Panel
          title="Live feed"
          subtitle={`Polling every ${FEED_POLL_MS / 1000}s. Duplicate scans and rejected fingerprints are shown, not hidden.`}
          className="lab-span-2"
          actions={
            <div className="lab-live-toolbar">
              <Select
                aria-label="Filter by outcome"
                value={outcomeFilter}
                onChange={(event) => setOutcomeFilter(event.target.value)}
                className="lab-select--compact"
              >
                <option value="">All outcomes</option>
                {Object.entries(OUTCOME_META).map(([value, meta]) => (
                  <option key={value} value={value}>{meta.label}</option>
                ))}
              </Select>
              <Select
                aria-label="Filter by terminal"
                value={terminalFilter}
                onChange={(event) => setTerminalFilter(event.target.value)}
                className="lab-select--compact"
              >
                <option value="">All terminals</option>
                {terminals.map((terminal) => (
                  <option key={terminal.id} value={terminal.id}>
                    {terminal.name || terminal.device_uid}
                  </option>
                ))}
              </Select>
            </div>
          }
        >
          {feedLoading && !feed.length ? (
            <LoadingState label="Waiting for attendance events" rows={5} />
          ) : feedError && !feed.length ? (
            <ErrorState
              error={feedError}
              title="The live feed is unavailable"
              onRetry={refresh}
              compact
            />
          ) : filteredEvents.length ? (
            <ul className="lab-feed lab-live-feed">
              {filteredEvents.map((event) => (
                <LiveEvent key={event.id} event={event} />
              ))}
            </ul>
          ) : (
            <EmptyState
              icon={ClockIcon}
              title={feed.length ? "No events match these filters" : "No attendance events yet"}
              description={
                feed.length
                  ? "Clear the filters to see every scan received by the backend."
                  : "Scan a finger on an enrolled attendance terminal and the event will appear here within a few seconds."
              }
            />
          )}
        </Panel>

        <Panel title="Terminals" subtitle="Reader health and enrollment state.">
          {terminalsLoading && !terminalsData ? (
            <LoadingState label="Loading attendance terminals" rows={3} compact />
          ) : terminalsError && !terminalsData ? (
            <ErrorState error={terminalsError} title="Terminals unavailable" onRetry={refresh} compact />
          ) : terminals.length ? (
            <div className="lab-device-stack">
              {terminals.map((terminal) => (
                <TerminalCard
                  key={terminal.id}
                  terminal={terminal}
                  onEnroll={setEnrollTerminal}
                  onMappings={setMappingsTerminal}
                />
              ))}
            </div>
          ) : (
            <EmptyState
              icon={ShieldIcon}
              title="No attendance terminals"
              description="Provision an attendance device with device_type attendance_terminal, then point its firmware at this API."
            />
          )}
        </Panel>
      </div>

      {enrollTerminal && (
        <EnrollmentDialog
          terminal={enrollTerminal}
          members={members}
          onClose={() => setEnrollTerminal(null)}
          onChanged={reloadTerminals}
        />
      )}
      {mappingsTerminal && (
        <MappingsDialog
          terminal={mappingsTerminal}
          onClose={() => setMappingsTerminal(null)}
          onChanged={reloadTerminals}
        />
      )}
    </>
  );
}
