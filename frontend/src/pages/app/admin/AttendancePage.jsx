import { useMemo, useState } from "react";
import { attendanceApi, membersManagementApi } from "../../../api/services";
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
  Pagination,
  Panel,
  Select,
  StatCard,
  TextInput,
} from "../../../components/app/Ui";
import {
  CheckCircleIcon,
  ClockIcon,
  PlusIcon,
  RefreshIcon,
  UsersIcon,
} from "../../../components/app/AppIcons";
import {
  daysAgoIso,
  formatClockTime,
  formatDate,
  formatNumber,
  toIsoDate,
  titleCase,
} from "../../../utils/format";

const SOURCE_OPTIONS = ["manual", "biometric", "card", "other"];

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

export default function AttendancePage() {
  const [filters, setFilters] = useState({
    page: 1,
    date: "",
    memberId: "",
    branch: "",
    year: "",
    source: "",
  });
  const [formOpen, setFormOpen] = useState(false);
  const [logoutTarget, setLogoutTarget] = useState(null);
  const [logoutBusy, setLogoutBusy] = useState(false);
  const [notice, setNotice] = useState(null);
  const [actionError, setActionError] = useState(null);

  const query = useMemo(
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

  const resource = useApiResource(
    () => attendanceApi.list(query),
    `attendance-${JSON.stringify(query)}`,
  );
  const analyticsResource = useApiResource(
    () => attendanceApi.analytics({ date_from: daysAgoIso(29), date_to: toIsoDate(new Date()) }),
    "attendance-analytics",
  );
  const membersResource = useApiResource(
    () => membersManagementApi.list({ per_page: 100, status: "active" }),
    "attendance-member-options",
  );

  const records = resource.data?.attendance ?? [];
  const analytics = analyticsResource.data?.analytics;
  const members = membersResource.data?.members ?? [];

  const handleLogout = async () => {
    if (!logoutTarget || logoutBusy) return;
    setLogoutBusy(true);
    setActionError(null);
    try {
      await attendanceApi.recordLogout(logoutTarget.id);
      setNotice("Logout recorded.");
      setLogoutTarget(null);
      resource.reload();
      analyticsResource.reload();
    } catch (error) {
      setActionError(error);
    } finally {
      setLogoutBusy(false);
    }
  };

  return (
    <>
      <PageHeader
        eyebrow="Admin console"
        title="Attendance"
        description="Review lab sessions, record manual entries and close open check-ins."
        actions={
          <>
            <AppButton variant="outline" onClick={() => resource.reload()} loading={resource.loading}>
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
                {[1, 2, 3, 4, 5, 6].map((year) => (
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
              onClick={() => setFilters({ page: 1, date: "", memberId: "", branch: "", year: "", source: "" })}
            >
              Clear filters
            </AppButton>
          </div>
        </div>

        <div className="lab-list-panel__body">
          {resource.loading ? (
            <LoadingState rows={8} />
          ) : resource.error ? (
            <ErrorState error={resource.error} onRetry={resource.reload} title="Attendance records are unavailable" />
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
                    <tr key={record.id}>
                      <td>
                        <div className="lab-member-academic">
                          <strong>{record.member?.name || "Unknown member"}</strong>
                          <small>{record.member?.login_id || record.member?.usn || "—"}</small>
                        </div>
                      </td>
                      <td>{formatDate(record.entry_date)}</td>
                      <td>{formatClockTime(record.entry_time?.slice(11, 16) || record.entry_time)}</td>
                      <td>
                        {record.exit_time
                          ? formatClockTime(record.exit_time?.slice(11, 16) || record.exit_time)
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
          )}
        </div>

        {resource.data?.pagination && (
          <Pagination
            page={resource.data.pagination.page}
            totalPages={resource.data.pagination.total_pages}
            totalCount={resource.data.pagination.total_count}
            onPageChange={(page) => setFilters((current) => ({ ...current, page }))}
            loading={resource.loading}
          />
        )}
      </Panel>

      {!analyticsResource.loading && analytics?.by_branch?.length > 0 && (
        <Panel title="Attendance by branch" subtitle="Last 30 days of recorded events.">
          <div className="lab-bars">
            {analytics.by_branch.map((branch) => {
              const max = Math.max(...analytics.by_branch.map((entry) => Number(entry.events) || 0), 1);
              return (
                <div key={branch.branch || "unassigned"} className="lab-bars__row">
                  <span className="lab-bars__label">{branch.branch || "Unassigned"}</span>
                  <div className="lab-bars__track">
                    <span style={{ width: `${Math.max(((Number(branch.events) || 0) / max) * 100, 4)}%` }} />
                  </div>
                  <strong>{formatNumber(branch.events)}</strong>
                </div>
              );
            })}
          </div>
        </Panel>
      )}

      {formOpen && (
        <ManualAttendanceDialog
          members={members}
          onClose={() => setFormOpen(false)}
          onSaved={(result) => {
            setNotice(result.message || "Attendance recorded.");
            resource.reload();
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
    </>
  );
}
