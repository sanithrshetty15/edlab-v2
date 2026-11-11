import { useMemo, useState } from "react";
import { attendanceApi } from "../../api/services";
import { useApiResource } from "../../hooks/useApiResource";
import {
  AppButton,
  Badge,
  EmptyState,
  ErrorState,
  LoadingState,
  PageHeader,
  Pagination,
  Panel,
  StatCard,
} from "../../components/app/Ui";
import {
  CheckCircleIcon,
  ClockIcon,
  RefreshIcon,
  ShieldIcon,
  UsersIcon,
} from "../../components/app/AppIcons";
import {
  daysAgoIso,
  formatClockTime,
  formatDate,
  formatNumber,
  toIsoDate,
  titleCase,
} from "../../utils/format";

function minutesBetween(start, end) {
  const from = new Date(start).getTime();
  const to = end ? new Date(end).getTime() : Date.now();
  if (!Number.isFinite(from) || !Number.isFinite(to) || to < from) return 0;
  return Math.round((to - from) / 60000);
}

function formatDuration(minutes) {
  if (!Number.isFinite(minutes) || minutes <= 0) return "—";
  const hours = Math.floor(minutes / 60);
  const rest = minutes % 60;
  if (!hours) return `${rest}m`;
  return `${hours}h ${String(rest).padStart(2, "0")}m`;
}

function SessionItem({ record }) {
  const minutes = minutesBetween(record.entry_time, record.exit_time);
  return (
    <li className="lab-timeline__item">
      <span className={`lab-timeline__dot ${record.exit_time ? "is-done" : "is-live"}`} />
      <div className="lab-timeline__content">
        <div className="lab-timeline__row">
          <span>{formatDate(record.entry_date)}</span>
          {!record.exit_time ? (
            <Badge tone="accent">In lab now</Badge>
          ) : (
            <Badge tone="neutral">{formatDuration(minutes)}</Badge>
          )}
        </div>
        <p>
          {formatClockTime(record.entry_time)} in
          {record.exit_time
            ? ` · ${formatClockTime(record.exit_time)} out`
            : " · session open"}
        </p>
        <p className="lab-muted">
          {titleCase(record.source || "manual")}
          {record.device_reference ? ` · ${record.device_reference}` : ""}
        </p>
      </div>
    </li>
  );
}

export default function MemberAttendancePage() {
  const [page, setPage] = useState(1);
  const query = useMemo(() => ({ page, per_page: 15 }), [page]);

  const resource = useApiResource(
    () =>
      Promise.all([
        attendanceApi.today(),
        attendanceApi.list(query),
        attendanceApi.dailySummary({ date_from: daysAgoIso(29), date_to: toIsoDate(new Date()) }),
        attendanceApi.summary({ date_from: daysAgoIso(29), date_to: toIsoDate(new Date()) }),
      ]).then(([today, history, daily, summary]) => ({ today, history, daily, summary })),
    `member-attendance-${page}`,
  );

  const today = resource.data?.today?.today ?? [];
  const todaySummary = resource.data?.today?.summary;
  const history = resource.data?.history?.attendance ?? [];
  const historyPagination = resource.data?.history?.pagination;
  const daily = resource.data?.daily?.daily_summary ?? [];
  const openSessions = resource.data?.summary?.summary?.total_login_events ?? 0;

  const totalMinutes = today.reduce(
    (sum, record) => sum + minutesBetween(record.entry_time, record.exit_time),
    0,
  );
  const activeNow = today.some((record) => !record.exit_time);
  const daysPresent = daily.length;

  return (
    <>
      <PageHeader
        eyebrow="Member portal"
        title="My attendance"
        description="Your lab check-ins and check-outs, recorded by the lab's attendance system."
        actions={
          <AppButton variant="outline" onClick={resource.reload} loading={resource.loading}>
            <RefreshIcon className="h-4 w-4" />
            Refresh
          </AppButton>
        }
      />

      {resource.error ? (
        <ErrorState error={resource.error} onRetry={resource.reload} title="Attendance is unavailable" />
      ) : (
        <>
          <div className="lab-stat-grid">
            <StatCard
              label="Sessions today"
              value={formatNumber(today.length)}
              hint={todaySummary?.unique_members ? "Across the whole lab" : "Your recorded entries"}
              icon={ClockIcon}
              tone="accent"
              loading={resource.loading}
            />
            <StatCard
              label="Time in lab today"
              value={formatDuration(totalMinutes)}
              hint={activeNow ? "A session is currently open" : "Completed sessions only"}
              icon={CheckCircleIcon}
              loading={resource.loading}
            />
            <StatCard
              label="Days present (30d)"
              value={formatNumber(daysPresent)}
              hint="Days with at least one check-in"
              icon={UsersIcon}
              loading={resource.loading}
            />
            <StatCard
              label="Open sessions (30d)"
              value={formatNumber(openSessions)}
              hint="Sessions missing a check-out"
              icon={ShieldIcon}
              tone={openSessions ? "warning" : "neutral"}
              loading={resource.loading}
            />
          </div>

          <div className="lab-dashboard-grid">
            <Panel title="Today" subtitle="Your live lab activity for today.">
              {resource.loading ? (
                <LoadingState rows={3} compact />
              ) : today.length === 0 ? (
                <EmptyState
                  icon={ClockIcon}
                  title="No check-in yet today"
                  description="Check in with the lab attendance device or ask an admin to record your entry."
                />
              ) : (
                <ul className="lab-timeline">
                  {today.map((record) => (
                    <SessionItem key={record.id} record={record} />
                  ))}
                </ul>
              )}
            </Panel>

            <Panel title="Last 30 days" subtitle="Daily session rhythm.">
              {resource.loading ? (
                <LoadingState rows={4} compact />
              ) : daily.length === 0 ? (
                <EmptyState
                  icon={ClockIcon}
                  title="No recent activity"
                  description="Your last 30 days will appear here after your first check-in."
                />
              ) : (
                <ul className="lab-activity-list">
                  {daily.slice(0, 10).map((day) => (
                    <li key={day.date}>
                      <div>
                        <strong>{formatDate(day.date)}</strong>
                        <span>
                          {day.logins} in · {day.logouts} out
                        </span>
                      </div>
                      <div className="lab-activity-list__bar" aria-hidden="true">
                        <span style={{ width: `${Math.min(100, Number(day.total_events) * 18)}%` }} />
                      </div>
                      <Badge tone="info">{day.total_events} events</Badge>
                    </li>
                  ))}
                </ul>
              )}
            </Panel>
          </div>

          <Panel
            title="Attendance history"
            subtitle="Newest lab sessions first."
            padded={false}
          >
            {resource.loading ? (
              <div className="lab-panel__body">
                <LoadingState rows={5} compact />
              </div>
            ) : history.length === 0 ? (
              <div className="lab-panel__body">
                <EmptyState
                  icon={ClockIcon}
                  title="No attendance recorded"
                  description="Once the lab records your first check-in, your session history appears here."
                />
              </div>
            ) : (
              <div className="lab-table-wrap">
                <table className="lab-table">
                  <thead>
                    <tr>
                      <th>Date</th>
                      <th>Check-in</th>
                      <th>Check-out</th>
                      <th>Duration</th>
                      <th>Source</th>
                    </tr>
                  </thead>
                  <tbody>
                    {history.map((record) => (
                      <tr key={record.id}>
                        <td>{formatDate(record.entry_date)}</td>
                        <td>{formatClockTime(record.entry_time)}</td>
                        <td>
                          {record.exit_time
                            ? formatClockTime(record.exit_time)
                            : <Badge tone="accent">Open</Badge>}
                        </td>
                        <td>{formatDuration(minutesBetween(record.entry_time, record.exit_time))}</td>
                        <td>{titleCase(record.source || "manual")}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
            {historyPagination && (
              <Pagination
                page={historyPagination.page}
                totalPages={historyPagination.total_pages}
                totalCount={historyPagination.total_count}
                onPageChange={setPage}
                loading={resource.loading}
              />
            )}
          </Panel>
        </>
      )}
    </>
  );
}
