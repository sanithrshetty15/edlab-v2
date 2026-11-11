import { useMemo, useState } from "react";
import { analyticsApi } from "../../../api/services";
import { useApiResource } from "../../../hooks/useApiResource";
import {
  Badge,
  EmptyState,
  ErrorState,
  LoadingState,
  PageHeader,
  Panel,
  Select,
  StatCard,
} from "../../../components/app/Ui";
import {
  AlertIcon,
  BoxIcon,
  CalendarIcon,
  CheckSquareIcon,
  ChartIcon,
  MegaphoneIcon,
  UsersIcon,
} from "../../../components/app/AppIcons";
import {
  daysAgoIso,
  formatDate,
  formatMonthDay,
  formatNumber,
  formatRelative,
  titleCase,
  toIsoDate,
} from "../../../utils/format";

function BarList({ data, valueKey, labelKey, emptyMessage, formatLabel = titleCase }) {
  if (!data?.length) return <p className="lab-muted lab-small">{emptyMessage}</p>;
  const max = Math.max(...data.map((item) => Number(item[valueKey]) || 0), 1);

  return (
    <div className="lab-bars">
      {data.map((item) => {
        const value = Number(item[valueKey]) || 0;
        return (
          <div key={String(item[labelKey] ?? "none")} className="lab-bars__row">
            <span className="lab-bars__label">{item[labelKey] ? formatLabel(item[labelKey]) : "Unassigned"}</span>
            <div className="lab-bars__track">
              <span style={{ width: `${Math.max((value / max) * 100, value ? 4 : 0)}%` }} />
            </div>
            <strong>{formatNumber(value)}</strong>
          </div>
        );
      })}
    </div>
  );
}

function DailyChart({ data }) {
  if (!data?.length) {
    return (
      <EmptyState
        icon={ChartIcon}
        title="No attendance history"
        description="Daily attendance will appear here when records exist in the selected period."
      />
    );
  }

  const days = [...data].reverse();
  const max = Math.max(...days.map((day) => Number(day.unique_members) || 1), 1);

  return (
    <div className="lab-mini-chart lab-mini-chart--wide" aria-label="Unique members per day">
      {days.map((day) => {
        const height = Math.max(((Number(day.unique_members) || 0) / max) * 100, 3);
        const parts = formatMonthDay(day.date);
        return (
          <div key={day.date} className="lab-mini-chart__column">
            <span className="lab-mini-chart__value">{day.unique_members ?? 0}</span>
            <span className="lab-mini-chart__bar" style={{ height: `${height}%` }} />
            <span className="lab-mini-chart__label">{parts.day} {parts.month}</span>
          </div>
        );
      })}
    </div>
  );
}

export default function AnalyticsPage() {
  const [days, setDays] = useState("30");

  const attendanceQuery = useMemo(
    () => ({
      date_from: daysAgoIso(Number(days) - 1),
      date_to: toIsoDate(new Date()),
    }),
    [days],
  );

  const members = useApiResource(() => analyticsApi.members(), "analytics-members");
  const attendance = useApiResource(
    () => analyticsApi.attendance(attendanceQuery),
    `analytics-attendance-${days}`,
  );
  const inventory = useApiResource(() => analyticsApi.inventory(), "analytics-inventory");
  const tasks = useApiResource(() => analyticsApi.tasks(), "analytics-tasks");
  const announcements = useApiResource(() => analyticsApi.announcements(), "analytics-announcements");
  const contact = useApiResource(() => analyticsApi.contact(), "analytics-contact");

  const memberStats = members.data?.members;
  const attendanceStats = attendance.data?.attendance;
  const inventoryStats = inventory.data?.inventory;
  const taskStats = tasks.data?.tasks;
  const announcementStats = announcements.data?.announcements;
  const contactStats = contact.data?.contact;

  return (
    <>
      <PageHeader
        eyebrow="Admin console"
        title="Analytics"
        description="Live operational reporting from members, attendance, inventory, tasks and communications."
        actions={
          <Select
            value={days}
            onChange={(event) => setDays(event.target.value)}
            aria-label="Attendance range"
            className="lab-select--compact"
          >
            <option value="7">Last 7 days</option>
            <option value="30">Last 30 days</option>
            <option value="90">Last 90 days</option>
          </Select>
        }
      />

      <div className="lab-stat-grid lab-stat-grid--4">
        <StatCard
          label="Active members"
          value={formatNumber(memberStats?.active ?? memberStats?.total)}
          hint={`${formatNumber(memberStats?.total)} total active records`}
          icon={UsersIcon}
          tone="accent"
          loading={members.loading}
        />
        <StatCard
          label="Present now"
          value={formatNumber(attendanceStats?.current_occupancy?.present_now)}
          hint="Open attendance sessions"
          icon={CalendarIcon}
          loading={attendance.loading}
        />
        <StatCard
          label="Available stock"
          value={formatNumber(inventoryStats?.summary?.available_stock)}
          hint={`${formatNumber(inventoryStats?.summary?.total_items)} item records`}
          icon={BoxIcon}
          loading={inventory.loading}
        />
        <StatCard
          label="Pending submissions"
          value={formatNumber(taskStats?.summary?.pending_submissions)}
          hint={`${formatNumber(taskStats?.summary?.overdue_tasks)} overdue tasks`}
          icon={CheckSquareIcon}
          tone={Number(taskStats?.summary?.pending_submissions) > 0 ? "warning" : "neutral"}
          loading={tasks.loading}
        />
      </div>

      <div className="lab-analytics-grid">
        <Panel
          title="Members"
          subtitle="Composition of the active member base."
          actions={<Badge tone="accent">{formatNumber(memberStats?.total)} active</Badge>}
        >
          {members.loading ? (
            <LoadingState rows={6} />
          ) : members.error ? (
            <ErrorState error={members.error} onRetry={members.reload} compact />
          ) : (
            <div className="lab-analytics-stack">
              <div>
                <h3 className="lab-analytics-subtitle">By academic year</h3>
                <BarList
                  data={memberStats?.by_academic_year}
                  valueKey="count"
                  labelKey="academic_year"
                  formatLabel={(value) => `Year ${value}`}
                  emptyMessage="No member year data."
                />
              </div>
              <div>
                <h3 className="lab-analytics-subtitle">By branch</h3>
                <BarList
                  data={memberStats?.by_branch}
                  valueKey="count"
                  labelKey="branch"
                  emptyMessage="No branch data."
                />
              </div>
            </div>
          )}
        </Panel>

        <Panel
          title="Attendance"
          subtitle={`Summary for the last ${days} days.`}
          actions={<Badge tone="info">{formatNumber(attendanceStats?.summary?.unique_members_present)} visitors</Badge>}
          className="lab-span-2"
        >
          {attendance.loading ? (
            <LoadingState rows={6} />
          ) : attendance.error ? (
            <ErrorState error={attendance.error} onRetry={attendance.reload} compact />
          ) : (
            <div className="lab-analytics-stack">
              <div className="lab-attendance-overview__stats lab-attendance-overview__stats--compact">
                <div>
                  <span>Open sessions</span>
                  <strong>{formatNumber(attendanceStats?.summary?.total_login_events)}</strong>
                </div>
                <div>
                  <span>Check-outs</span>
                  <strong>{formatNumber(attendanceStats?.summary?.total_logout_events)}</strong>
                </div>
                <div>
                  <span>Unique visitors</span>
                  <strong>{formatNumber(attendanceStats?.summary?.unique_members_present)}</strong>
                </div>
                <div>
                  <span>Present now</span>
                  <strong>{formatNumber(attendanceStats?.current_occupancy?.present_now)}</strong>
                </div>
              </div>
              <DailyChart data={attendanceStats?.daily_summary} />
            </div>
          )}
        </Panel>

        <Panel title="Inventory" subtitle="Stock health and low-stock items.">
          {inventory.loading ? (
            <LoadingState rows={5} />
          ) : inventory.error ? (
            <ErrorState error={inventory.error} onRetry={inventory.reload} compact />
          ) : (
            <div className="lab-analytics-stack">
              <div className="lab-inline-stats">
                <div><span>Total stock</span><strong>{formatNumber(inventoryStats?.summary?.total_stock)}</strong></div>
                <div><span>On loan</span><strong>{formatNumber(inventoryStats?.summary?.borrowed_stock)}</strong></div>
                <div><span>Overdue loans</span><strong>{formatNumber(inventoryStats?.overdue_borrows_count)}</strong></div>
              </div>
              <BarList
                data={inventoryStats?.by_category}
                valueKey="total_quantity"
                labelKey="category"
                emptyMessage="No inventory categories."
              />
              {inventoryStats?.low_stock_items?.length > 0 && (
                <div>
                  <h3 className="lab-analytics-subtitle">Low stock</h3>
                  <ul className="lab-low-stock-list">
                    {inventoryStats.low_stock_items.map((item) => (
                      <li key={item.id}>
                        <span>{item.name}</span>
                        <Badge tone={item.available_quantity === 0 ? "danger" : "warning"}>
                          {item.available_quantity}/{item.total_quantity}
                        </Badge>
                      </li>
                    ))}
                  </ul>
                </div>
              )}
            </div>
          )}
        </Panel>

        <Panel title="Tasks" subtitle="Task delivery and submission status.">
          {tasks.loading ? (
            <LoadingState rows={5} />
          ) : tasks.error ? (
            <ErrorState error={tasks.error} onRetry={tasks.reload} compact />
          ) : (
            <div className="lab-analytics-stack">
              <div className="lab-inline-stats">
                <div><span>Published</span><strong>{formatNumber(taskStats?.summary?.published_tasks)}</strong></div>
                <div><span>Verified</span><strong>{formatNumber(taskStats?.summary?.verified_submissions)}</strong></div>
                <div><span>Rejected</span><strong>{formatNumber(taskStats?.summary?.rejected_submissions)}</strong></div>
              </div>
              <BarList
                data={taskStats?.by_type}
                valueKey="count"
                labelKey="task_type"
                emptyMessage="No task data."
              />
            </div>
          )}
        </Panel>

        <Panel title="Announcements" subtitle="Publication and read-through.">
          {announcements.loading ? (
            <LoadingState rows={5} />
          ) : announcements.error ? (
            <ErrorState error={announcements.error} onRetry={announcements.reload} compact />
          ) : (
            <div className="lab-analytics-stack">
              <div className="lab-inline-stats">
                <div><span>Published</span><strong>{formatNumber(announcementStats?.summary?.published_announcements)}</strong></div>
                <div><span>Pinned</span><strong>{formatNumber(announcementStats?.summary?.pinned_announcements)}</strong></div>
                <div><span>Total reads</span><strong>{formatNumber(announcementStats?.summary?.total_reads)}</strong></div>
              </div>
              {announcementStats?.recent?.length ? (
                <ul className="lab-read-list">
                  {announcementStats.recent.slice(0, 5).map((announcement) => (
                    <li key={announcement.id}>
                      <div className="lab-read-list__row">
                        <span>{announcement.title}</span>
                        <strong>{Math.round(Number(announcement.read_percentage) || 0)}%</strong>
                      </div>
                      <div className="lab-read-list__track">
                        <span style={{ width: `${Math.min(Number(announcement.read_percentage) || 0, 100)}%` }} />
                      </div>
                      <p>{formatRelative(announcement.published_at)}</p>
                    </li>
                  ))}
                </ul>
              ) : (
                <p className="lab-muted lab-small">No published announcement analytics.</p>
              )}
            </div>
          )}
        </Panel>

        <Panel title="Member registrations" subtitle="Newest active member records.">
          {members.loading ? (
            <LoadingState rows={5} />
          ) : members.error ? (
            <ErrorState error={members.error} onRetry={members.reload} compact />
          ) : memberStats?.recent_registrations?.length ? (
            <ul className="lab-activity">
              {memberStats.recent_registrations.slice(0, 6).map((member) => (
                <li key={member.id} className="lab-activity__item">
                  <span className="lab-activity__icon lab-activity__icon--accent">
                    <UsersIcon className="h-4 w-4" />
                  </span>
                  <div>
                    <strong className="lab-activity__title">{member.name}</strong>
                    <p>{member.login_id} · {member.branch || "No branch"}</p>
                  </div>
                  <time>{formatRelative(member.created_at)}</time>
                </li>
              ))}
            </ul>
          ) : (
            <EmptyState
              icon={UsersIcon}
              title="No registrations"
              description="New member registrations will appear here."
            />
          )}
        </Panel>

        <Panel title="Contact enquiries" subtitle="Public messages received by the lab.">
          {contact.loading ? (
            <LoadingState rows={5} />
          ) : contact.error ? (
            <ErrorState error={contact.error} onRetry={contact.reload} compact />
          ) : (
            <div className="lab-analytics-stack">
              <div className="lab-inline-stats">
                <div><span>New</span><strong>{formatNumber(contactStats?.summary?.new_messages)}</strong></div>
                <div><span>In progress</span><strong>{formatNumber(contactStats?.summary?.in_progress)}</strong></div>
                <div><span>Resolved</span><strong>{formatNumber(contactStats?.summary?.resolved_messages)}</strong></div>
              </div>
              <BarList
                data={contactStats?.by_category}
                valueKey="count"
                labelKey="category"
                emptyMessage="No contact enquiries."
              />
            </div>
          )}
        </Panel>

        <Panel title="Announcement engagement" subtitle="Most recent publication performance.">
          {announcements.loading ? (
            <LoadingState rows={5} />
          ) : announcements.error ? (
            <ErrorState error={announcements.error} onRetry={announcements.reload} compact />
          ) : announcementStats?.recent?.length ? (
            <div className="lab-table-wrap">
              <table className="lab-table lab-table--compact">
                <thead>
                  <tr>
                    <th>Announcement</th>
                    <th>Published</th>
                    <th>Reads</th>
                    <th>Read rate</th>
                  </tr>
                </thead>
                <tbody>
                  {announcementStats.recent.slice(0, 6).map((announcement) => (
                    <tr key={announcement.id}>
                      <td>{announcement.title}</td>
                      <td>{formatDate(announcement.published_at)}</td>
                      <td>{formatNumber(announcement.read_count)}</td>
                      <td>{Math.round(Number(announcement.read_percentage) || 0)}%</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <EmptyState
              icon={MegaphoneIcon}
              title="No announcement data"
              description="Publish an announcement to start collecting read analytics."
            />
          )}
        </Panel>

        <Panel title="Data notes" subtitle="How these figures are produced." className="lab-span-2">
          <ul className="lab-guideline-list">
            <li><AlertIcon className="h-4 w-4" /> Figures are computed directly from live API records, not cached samples.</li>
            <li><CalendarIcon className="h-4 w-4" /> Attendance ranges use calendar dates; open sessions are entries without an exit time.</li>
            <li><CheckSquareIcon className="h-4 w-4" /> Task submission totals depend on records created when tasks are published.</li>
          </ul>
        </Panel>
      </div>
    </>
  );
}
