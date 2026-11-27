import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { attendanceApi, membersManagementApi, attendanceTerminalsApi } from "../../../api/services";
import { useApiResource } from "../../../hooks/useApiResource";
import { ConfirmDialog, Modal } from "../../../components/app/Modal";
import {
  Alert,
  AppButton,
  Avatar,
  Badge,
  EmptyState,
  ErrorState,
  Field,
  LoadingState,
  Panel,
  SegmentedControl,
  Select,
  StatusBadge,
  StatCard,
  Tabs,
} from "../../../components/app/Ui";
import {
  CheckCircleIcon,
  ClockIcon,
  PlusIcon,
  RefreshIcon,
  UsersIcon,
  AlertIcon,
} from "../../../components/app/AppIcons";
import {
  daysAgoIso,
  formatClockTime,
  formatDate,
  formatNumber,
  toIsoDate,
  titleCase,
  formatRelative,
} from "../../../utils/format";

const SOURCE_OPTIONS = ["manual", "biometric", "card", "other"];

function LiveEvent({ event }) {
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

function ManualAttendanceDialog({ members, onClose, onSaved }) {
  const [memberId, setMemberId] = useState("");
  const [date, setDate] = useState(toIsoDate(new Date()));
  const [time, setTime] = useState(
    `${String(new Date().getHours()).padStart(2, "0")}:${String(new Date().getMinutes()).padStart(2, "0")}`,
  );
  const [source, setSource] = useState("manual");
  const [deviceReference, setDeviceReference] = useState("");
  const [errors, setErrors] = useState({});
  const [formError, setFormError] = useState(null);
  const [saving, setSaving] = useState(false);

  const handleSubmit = async (event) => {
    event.preventDefault();
    if (saving) return;

    const nextErrors = {};
    if (!memberId) nextErrors.memberId = "Choose a member.";
    if (!date) nextErrors.date = "Choose an entry date.";
    if (!time) nextErrors.time = "Choose an entry time.";
    setErrors(nextErrors);
    setFormError(null);
    if (Object.keys(nextErrors).length) return;

    setSaving(true);
    try {
      const result = await attendanceApi.create({
        member_id: memberId,
        entry_date: date,
        entry_time: `${date}T${time}:00`,
        source,
        device_reference: deviceReference.trim() || undefined,
      });
      onSaved(result);
      onClose();
    } catch (error) {
      setFormError(error);
    } finally {
      setSaving(false);
    }
  };

  return (
    <Modal
      open
      onClose={onClose}
      title="Record manual attendance"
      description="Creates an official attendance entry for an active member."
      footer={
        <>
          <AppButton variant="ghost" onClick={onClose} disabled={saving}>
            Cancel
          </AppButton>
          <AppButton form="attendance-form" type="submit" variant="primary" loading={saving}>
            Record entry
          </AppButton>
        </>
      }
    >
      <form id="attendance-form" className="lab-form-stack" onSubmit={handleSubmit} noValidate>
        {formError && (
          <Alert tone="danger" title="Attendance could not be recorded">
            {formError.message}
          </Alert>
        )}
        <Field label="Member" htmlFor="attendance-member" error={errors.memberId} required>
          <Select
            id="attendance-member"
            value={memberId}
            onChange={(event) => setMemberId(event.target.value)}
            disabled={saving}
          >
            <option value="">Select an active member</option>
            {members.map((member) => (
              <option key={member.id} value={member.id}>
                {member.name} · {member.login_id}
              </option>
            ))}
          </Select>
        </Field>
        <div className="lab-form-grid">
          <Field label="Entry date" htmlFor="attendance-date" error={errors.date} required>
            <TextInput
              id="attendance-date"
              type="date"
              value={date}
              onChange={(event) => setDate(event.target.value)}
              disabled={saving}
            />
          </Field>
          <Field label="Entry time" htmlFor="attendance-time" error={errors.time} required>
            <TextInput
              id="attendance-time"
              type="time"
              value={time}
              onChange={(event) => setTime(event.target.value)}
              disabled={saving}
            />
          </Field>
        </div>
        <Field label="Source" htmlFor="attendance-source">
          <Select
            id="attendance-source"
            value={source}
            onChange={(event) => setSource(event.target.value)}
            disabled={saving}
          >
            {SOURCE_OPTIONS.map((option) => (
              <option key={option} value={option}>{titleCase(option)}</option>
            ))}
          </Select>
        </Field>
        <Field
          label="Device reference"
          htmlFor="attendance-device"
          hint="Optional identifier for a device or manual register."
        >
          <TextInput
            id="attendance-device"
            value={deviceReference}
            onChange={(event) => setDeviceReference(event.target.value)}
            disabled={saving}
          />
        </Field>
      </form>
    </Modal>
  );
}

function MemberDetailPanel({ member, onClose }) {
  return (
    <Modal
      open
      onClose={onClose}
      title={member?.name || "Member record"}
      description={member ? `${member.login_id || "No login ID"} · ${member.email || "No email"}` : undefined}
      size="lg"
      footer={
        <>
          <AppButton variant="ghost" onClick={onClose}>Close</AppButton>
        </>
      }
    >
      {member ? (
        <div className="lab-member-detail">
          <div className="lab-member-detail__identity">
            <Avatar name={member.name} size="lg" />
            <div>
              <p className="lab-muted lab-small">
                Member since {formatDate(member.created_at)}
              </p>
            </div>
          </div>

          <DefinitionList
            items={[
              { label: "Login ID", value: member.login_id },
              { label: "USN", value: member.usn },
              { label: "Academic year", value: member.academic_year ? `Year ${member.academic_year}` : "—" },
              { label: "Branch", value: member.branch },
              { label: "Section", value: member.section },
              { label: "Email", value: member.email },
              { label: "Phone", value: member.phone || "—" },
            ]}
          />

          <div className="lab-member-detail__section">
            <h3>Status</h3>
            <StatusBadge status={member.status}>
              {member.status === "pending_enrollment" ? "Awaiting fingerprint" : undefined}
            </StatusBadge>
            {member.must_change_password && <Badge tone="warning">Password reset required</Badge>}
          </div>

          <div className="lab-member-detail__section">
            <h3>Today's attendance</h3>
            <p>
              <strong>Entry:</strong> {"—"}
            </p>
            <p>
              <strong>Exit:</strong> {"—"}
            </p>
          </div>

          <div className="lab-member-detail__section">
            <h3>Completed hours</h3>
            <p>Today: {"—"} hours</p>
            <p>This week: {"—"} hours</p>
            <p>This month: {"—"} hours</p>
            <p>All time: {"—"} hours</p>
          </div>

          <div className="lab-member-detail__section">
            <h3>Recent sessions</h3>
            <p>Most recent: {"No sessions recorded"}</p>
          </div>
        </div>
      ) : null}
    </Modal>
  );
}

export default function AttendancePage() {
  const [filters, setFilters] = useState({
    page: 1,
    date: "",
    memberId: "",
    branch: "",
    year: "",
    source: "",
    outcomeFilter: "",
    terminalFilter: "",
    tab: "live",
  });
  const [formOpen, setFormOpen] = useState(false);
  const [logoutTarget, setLogoutTarget] = useState(null);
  const [logoutBusy, setLogoutBusy] = useState(false);
  const [notice, setNotice] = useState(null);
  const [actionError, setActionError] = useState(null);
  const [tab, setTab] = useState("live");

  const recordsQuery = useMemo(
    () => ({
      page: filters.page,
      per_page: 25,
      date: filters.date || undefined,
      member_id: filters.memberId || undefined,
      branch: filters.branch || undefined,
      academic_year: filters.year || undefined,
      source: filters.source || undefined,
    }),
    [filters],
  );

  const liveQuery = useMemo(
    () => ({
      limit: 50,
      after_id: filters.liveAfterId || undefined,
    }),
    [filters.liveAfterId],
  );

  const recordsResource = useApiResource(
    () => attendanceApi.list(recordsQuery),
    `attendance-records-${JSON.stringify(recordsQuery)}`,
  );
  const liveResource = useApiResource(
    () => attendanceTerminalsApi.live(liveQuery),
    `attendance-live-${JSON.stringify(liveQuery)}`,
  );
  const analyticsResource = useApiResource(
    () => attendanceApi.analytics({ date_from: daysAgoIso(29), date_to: toIsoDate(new Date()) }),
    "attendance-analytics",
  );
  const membersResource = useApiResource(
    () => membersManagementApi.list({ per_page: 100, status: "active" }),
    "attendance-member-options",
  );

  const records = recordsResource.data?.attendance ?? [];
  const liveEvents = liveResource.data?.events ?? [];
  const liveAfterId = liveResource.latest_id ?? 0;
  const analytics = analyticsResource.data?.analytics;
  const members = membersResource.data?.members ?? [];

  const [statusBusy, setStatusBusy] = useState(null);
  const [detailMember, setDetailMember] = useState(null);

  const handleTabChange = useCallback(
    (newTab) => {
      setTab(newTab);
      if (newTab === "records") {
        setFilters((current) => ({ ...current, page: 1, outcomeFilter: "", terminalFilter: "" }));
      }
    },
    [],
  );

  const handleLogout = async () => {
    if (!logoutTarget || logoutBusy) return;
    setLogoutBusy(true);
    setActionError(null);
    try {
      await attendanceApi.recordLogout(logoutTarget.id);
      setNotice("Logout recorded.");
      setLogoutTarget(null);
      recordsResource.reload();
      liveResource.reload();
      analyticsResource.reload();
    } catch (error) {
      setActionError(error);
    } finally {
      setLogoutBusy(false);
    }
  };

  useEffect(() => {
    if (tab !== "live") return undefined;
    let stopped = false;
    let timer = null;
    let controller = null;

    const poll = async () => {
      controller = new AbortController();
      try {
        const result = await attendanceTerminalsApi.live(liveQuery, { signal: controller.signal });
        if (stopped) return;
        const incoming = [...(result.events || [])].reverse();
        setLiveEventsLocal(incoming);
        if (result.latest_id) liveAfterIdRef.current = result.latest_id;
        setLiveError(null);
      } catch (pollError) {
        if (stopped || pollError?.code === "aborted") return;
        setLiveError(pollError);
      } finally {
        if (!stopped) {
          setLiveLoading(false);
          timer = window.setTimeout(poll, 3000);
        }
      }
    };

    poll();
    return () => {
      stopped = true;
      window.clearTimeout(timer);
      controller?.abort();
    };
  }, [tab, liveQuery]);

  const liveAfterIdRef = useRef(liveAfterId);

  const [liveEventsLocal, setLiveEventsLocal] = useState([]);
  const [liveError, setLiveError] = useState(null);
  const [liveLoading, setLiveLoading] = useState(true);

  const openMemberDetail = useCallback((member) => {
    setDetailMember(member);
  }, []);

  const closeMemberDetail = useCallback(() => {
    setDetailMember(null);
  }, []);

  return (
    <>
      <PageHeader
        eyebrow="Admin console"
        title="Attendance"
        description="Review lab sessions, record manual entries and close open check-ins."
        actions={
          <>
            <AppButton variant="outline" onClick={() => recordsResource.reload()} loading={recordsResource.loading}>
              <RefreshIcon className="h-4 w-4" />
              Refresh
            </AppButton>
            <AppButton variant="primary" onClick={() => setFormOpen(true)}>
              <PlusIcon className="h-4 w-4" />
              Record entry
            </AppButton>
          </>
        }
      />

      {notice && (
        <Alert tone="success" title="Done" onDismiss={() => setNotice(null)}>
          {notice}
        </Alert>
      )}
      {actionError && (
        <Alert tone="danger" title="Action failed" onDismiss={() => setActionError(null)}>
          {actionError.message}
        </Alert>
      )}

      <div className="lab-stat-grid lab-stat-grid--4">
        <StatCard
          label="Events in 30 days"
          value={formatNumber(analytics?.total_events)}
          hint="Entry and exit events"
          icon={ClockIcon}
          loading={analyticsResource.loading}
        />
        <StatCard
          label="Unique visitors"
          value={formatNumber(analytics?.unique_members)}
          hint="Members who visited the lab"
          icon={UsersIcon}
          tone="accent"
          loading={analyticsResource.loading}
        />
        <StatCard
          label="Open sessions"
          value={formatNumber(analytics?.login_events)}
          hint="Entries without an exit"
          icon={CheckCircleIcon}
          loading={analyticsResource.loading}
        />
        <StatCard
          label="Average per day"
          value={analytics?.average_daily_attendance ?? "—"}
          hint="Recorded events per day"
          icon={ClockIcon}
          loading={analyticsResource.loading}
        />
      </div>

      <Panel padded={false}>
        <div className="lab-toolbar lab-toolbar--wrap">
          <div className="lab-filter-row">
            <Field label="Date" htmlFor="attendance-filter-date" className="lab-field--inline">
              <TextInput
                id="attendance-filter-date"
                type="date"
                value={filters.date}
                onChange={(event) =>
                  setFilters((current) => ({ ...current, date: event.target.value, page: 1 }))
                }
              />
            </Field>
            <Field label="Member" htmlFor="attendance-filter-member" className="lab-field--inline">
              <Select
                id="attendance-filter-member"
                value={filters.memberId}
                onChange={(event) =>
                  setFilters((current) => ({ ...current, memberId: event.target.value, page: 1 }))
                }
                className="lab-select--compact"
              >
                <option value="">All members</option>
                {members.map((member) => (
                  <option key={member.id} value={member.id}>{member.name}</option>
                ))}
              </Select>
            </Field>
            <Field label="Year" htmlFor="attendance-filter-year" className="lab-field--inline">
              <Select
                id="attendance-filter-year"
                value={filters.year}
                onChange={(event) =>
                  setFilters((current) => ({ ...current, year: event.target.value, page: 1 }))
                }
                className="lab-select--compact"
              >
                <option value="">All years</option>
                {[1, 2, 3, 4].map((year) => (
                  <option key={year} value={year}>Year {year}</option>
                ))}
              </Select>
            </Field>
            <Field label="Source" htmlFor="attendance-filter-source" className="lab-field--inline">
              <Select
                id="attendance-filter-source"
                value={filters.source}
                onChange={(event) =>
                  setFilters((current) => ({ ...current, source: event.target.value, page: 1 }))
                }
                className="lab-select--compact"
              >
                <option value="">All sources</option>
                {SOURCE_OPTIONS.map((source) => (
                  <option key={source} value={source}>{titleCase(source)}</option>
                ))}
              </Select>
            </Field>
            <AppButton
              variant="ghost"
              size="sm"
              onClick={() =>
                setFilters({
                  page: 1,
                  date: "",
                  memberId: "",
                  branch: "",
                  year: "",
                  source: "",
                  outcomeFilter: "",
                  terminalFilter: "",
                  tab: "records",
                })
              }
            >
              Clear filters
            </AppButton>
          </div>
        </div>

        <div className="lab-toolbar lab-toolbar--wrap">
          <Tabs
            tabs={[
              { value: "live", label: "Live feed" },
              { value: "records", label: "Records" },
            ]}
            value={tab}
            onChange={handleTabChange}
            lab="tabs"
          />
        </div>

        <div className="lab-list-panel__body">
          {tab === "records" ? (
            <>{recordsResource.loading ? (
              <LoadingState rows={8} />
            ) : recordsResource.error ? (
              <ErrorState error={recordsResource.error} onRetry={recordsResource.reload} title="Attendance records are unavailable" />
            ) : records.length === 0 ? (
              <EmptyState
                icon={ClockIcon}
                title="No attendance records found"
                description="Adjust the filters or record a manual entry."
                action={
                  <AppButton variant="primary" onClick={() => setFormOpen(true)}>
                    <PlusIcon className="h-4 w-4" />
                    Record entry
                  </AppButton>
                }
              />
            ) : (
              <div className="lab-table-wrap">
                <table className="lab-table">
                  <thead>
                    <tr>
                      <th>Member</th>
                      <th>Date</th>
                      <th>Entry</th>
                      <th>Exit</th>
                      <th>Source</th>
                      <th>Recorded by</th>
                      <th aria-label="Actions" />
                    </tr>
                  </thead>
                  <tbody>
                    {records.map((record) => (
                      <tr
                        key={record.id}
                        onClick={() => openMemberDetail(record.member)}
                        style={{ cursor: "pointer" }}
                      >
                        <td>
                          <div className="lab-member-academic">
                            <strong>{record.member?.name || "Unknown member"}</strong>
                            <small>{record.member?.login_id || record.member?.usn || "—"}</small>
                          </div>
                        </td>
                        <td>{formatDate(record.entry_date)}</td>
                        <td>{formatClockTime(record.entry_time)}</td>
                        <td>
                          {record.exit_time
                            ? formatClockTime(record.exit_time)
                              : <Badge tone="accent" dot>Open</Badge>}
                        </td>
                        <td><Badge tone="neutral">{titleCase(record.source)}</Badge></td>
                        <td>{record.created_by || "Biometric sync"}</td>
                        <td>
                          {!record.exit_time && (
                            <AppButton variant="ghost" size="sm" onClick={() => setLogoutTarget(record)}>
                              Record logout
                            </AppButton>
                          )}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}</>
          ) : tab === "live" ? (
            <Panel title="Live feed" subtitle="Fingerprint scans appear here as the attendance terminal submits them.">
              {liveLoading ? (
                <LoadingState label="Waiting for attendance events" rows={5} />
              ) : liveError && !liveEvents.length ? (
                <ErrorState
                  error={liveError}
                  title="The live feed is unavailable"
                  onRetry={() => liveResource.reload()}
                  compact
                />
              ) : liveEvents.length ? (
                <ul className="lab-feed lab-live-feed">
                  {liveEvents.map((event) => (
                    <LiveEvent key={event.id} event={event} />
                  ))}
                </ul>
              ) : (
                <EmptyState
                  icon={ClockIcon}
                  title={liveEvents.length ? "No events match these filters" : "No attendance events yet"}
                  description={
                    liveEvents.length
                      ? "Clear the filters to see every scan received by the backend."
                      : "Scan a finger on an enrolled attendance terminal and the event will appear here within a few seconds."
                  }
                />
              )}
            </Panel>
          ) : null}

          {recordsResource.data?.pagination && (
            <Pagination
              page={recordsResource.data.pagination.page}
              totalPages={recordsResource.data.pagination.total_pages}
              totalCount={recordsResource.data.pagination.total_count}
              onPageChange={(page) => setFilters((current) => ({ ...current, page }))}
              loading={recordsResource.loading}
            />
          )}
        </div>
      </Panel>

      {formOpen && (
        <ManualAttendanceDialog
          members={members}
          onClose={() => setFormOpen(false)}
          onSaved={(result) => {
            setNotice(result.message || "Attendance recorded.");
            recordsResource.reload();
            liveResource.reload();
            analyticsResource.reload();
          }}
        />
      )}

      <ConfirmDialog
        open={Boolean(logoutTarget)}
        onClose={() => setLogoutTarget(null)}
        onConfirm={handleLogout}
        title="Record logout?"
        description={`Close the open session for ${logoutTarget?.member?.name || "this member"} at the current time.`}
        confirmLabel="Record logout"
        loading={logoutBusy}
      />

      {detailMember && <MemberDetailPanel member={detailMember} onClose={closeMemberDetail} />}
    </>
  );
}
