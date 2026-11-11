import { analyticsApi, announcementsApi, borrowingApi, eventsApi, tasksApi } from "../../api/services";
import { useApiResource } from "../../hooks/useApiResource";
import { AppLink } from "../../components/app/Link";
import {
  AlertIcon,
  ArrowRightIcon,
  BoxIcon,
  CalendarIcon,
  CheckSquareIcon,
  ClockIcon,
  MegaphoneIcon,
  MessageIcon,
  ChartIcon,
  UsersIcon,
} from "../../components/app/AppIcons";
import {
  Badge,
  EmptyState,
  ErrorState,
  LoadingState,
  PageHeader,
  Panel,
  StatCard,
} from "../../components/app/Ui";
import {
  daysAgoIso,
  formatDate,
  formatMonthDay,
  formatNumber,
  formatRelative,
  formatTime,
  titleCase,
  toIsoDate,
} from "../../utils/format";

function MiniBars({ data, valueKey, labelKey, emptyMessage }) {
  if (!data?.length) return <p className="lab-muted lab-small">{emptyMessage}</p>;
  const max = Math.max(...data.map((item) => Number(item[valueKey]) || 0), 1);

  return (
    <div className="lab-bars">
      {data.map((item) => {
        const value = Number(item[valueKey]) || 0;
        return (
          <div key={String(item[labelKey])} className="lab-bars__row">
            <span className="lab-bars__label">{titleCase(item[labelKey])}</span>
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

function ActivityList({ activity }) {
  if (!activity.length) {
    return (
      <EmptyState
        icon={ClockIcon}
        title="No activity recorded"
        description="Recent publishing, events and requests will be listed here."
      />
    );
  }

  return (
    <ul className="lab-activity">
      {activity.map((item) => (
        <li key={item.id} className="lab-activity__item">
          <span className={`lab-activity__icon lab-activity__icon--${item.tone}`}>
            {item.icon}
          </span>
          <div>
            <AppLink to={item.to} className="lab-activity__title">
              {item.title}
            </AppLink>
            <p>
              {item.type}
              {item.actor ? ` · ${item.actor}` : ""}
            </p>
          </div>
          <time>{formatRelative(item.at)}</time>
        </li>
      ))}
    </ul>
  );
}

export default function AdminDashboard() {
  const dashboardResource = useApiResource(() => analyticsApi.dashboard(), "admin-dashboard");
  const attendanceResource = useApiResource(
    () => analyticsApi.attendance({ date_from: daysAgoIso(29), date_to: toIsoDate(new Date()) }),
    "admin-dashboard-attendance",
  );
  const announcementsResource = useApiResource(
    () => announcementsApi.list({ per_page: 5 }),
    "admin-dashboard-announcements",
  );
  const eventsResource = useApiResource(
    () => eventsApi.list({ upcoming: "true", per_page: 5 }),
    "admin-dashboard-events",
  );
  const tasksResource = useApiResource(
    () => tasksApi.list({ per_page: 5 }),
    "admin-dashboard-tasks",
  );
  const borrowingResource = useApiResource(
    () => borrowingApi.adminList({ pending: "true", per_page: 5 }),
    "admin-dashboard-borrowing",
  );

  const summary = dashboardResource.data?.dashboard?.summary ?? {};
  const trends = dashboardResource.data?.dashboard?.trends ?? {};
  const attendance = attendanceResource.data?.attendance;
  const announcements = announcementsResource.data?.announcements ?? [];
  const events = eventsResource.data?.events ?? [];
  const tasks = tasksResource.data?.tasks ?? [];
  const borrowRequests = borrowingResource.data?.requests ?? [];

  const activity = [
    ...announcements.map((item) => ({
      id: `announcement-${item.id}`,
      type: "Announcement",
      title: item.title,
      actor: item.created_by?.name,
      at: item.published_at || item.created_at,
      to: "/app/announcements",
      tone: "accent",
      icon: <MegaphoneIcon className="h-4 w-4" />,
    })),
    ...events.map((item) => ({
      id: `event-${item.id}`,
      type: "Event",
      title: item.title,
      actor: item.created_by?.name,
      at: item.created_at,
      to: "/app/events",
      tone: "info",
      icon: <CalendarIcon className="h-4 w-4" />,
    })),
    ...tasks.map((item) => ({
      id: `task-${item.id}`,
      type: "Task",
      title: item.title,
      actor: item.created_by?.name,
      at: item.published_at || item.created_at,
      to: "/app/tasks",
      tone: "warning",
      icon: <CheckSquareIcon className="h-4 w-4" />,
    })),
    ...borrowRequests.map((item) => ({
      id: `borrow-${item.id}`,
      type: "Borrow request",
      title: `${item.member?.name || "Member"} requested ${item.item?.name || "an item"}`,
      actor: null,
      at: item.borrowed_at,
      to: "/app/borrowing",
      tone: "neutral",
      icon: <BoxIcon className="h-4 w-4" />,
    })),
  ]
    .filter((item) => item.at)
    .sort((a, b) => new Date(b.at) - new Date(a.at))
    .slice(0, 7);

  const attendanceDays = [...(attendance?.daily_summary ?? [])].reverse().slice(-10).map((day) => ({
    date: day.date,
    members: day.unique_members,
  }));

  return (
    <>
      <PageHeader
        eyebrow="Admin console"
        title="Lab overview"
        description={`Operational picture for ${formatDate(new Date())}. All figures come from the live API.`}
        actions={
          <>
            <AppLink to="/app/members" className="lab-button lab-button--outline lab-button--md">
              <UsersIcon className="h-4 w-4" />
              Manage members
            </AppLink>
            <AppLink to="/app/announcements" className="lab-button lab-button--primary lab-button--md">
              <MegaphoneIcon className="h-4 w-4" />
              New announcement
            </AppLink>
          </>
        }
      />

      {dashboardResource.error && (
        <ErrorState
          error={dashboardResource.error}
          title="Dashboard summary is unavailable"
          onRetry={dashboardResource.reload}
        />
      )}

      <div className="lab-stat-grid lab-stat-grid--4">
        <StatCard
          label="Active members"
          value={formatNumber(summary.total_members)}
          hint={`${formatNumber(summary.active_today)} joined today`}
          icon={UsersIcon}
          tone="accent"
          loading={dashboardResource.loading}
        />
        <StatCard
          label="Present now"
          value={formatNumber(summary.present_now)}
          hint="Open attendance sessions"
          icon={ClockIcon}
          loading={dashboardResource.loading}
        />
        <StatCard
          label="Pending tasks"
          value={formatNumber(summary.pending_tasks)}
          hint="Submissions awaiting verification"
          icon={CheckSquareIcon}
          tone={Number(summary.pending_tasks) > 0 ? "warning" : "neutral"}
          loading={dashboardResource.loading}
        />
        <StatCard
          label="Borrow requests"
          value={formatNumber(summary.pending_borrow_requests)}
          hint="Awaiting approval"
          icon={BoxIcon}
          tone={Number(summary.pending_borrow_requests) > 0 ? "warning" : "neutral"}
          loading={dashboardResource.loading}
        />
      </div>

      <div className="lab-stat-grid lab-stat-grid--4">
        <StatCard
          label="Return requests"
          value={formatNumber(summary.pending_return_requests)}
          hint="Awaiting confirmation"
          icon={ArrowRightIcon}
          loading={dashboardResource.loading}
        />
        <StatCard
          label="Inventory items"
          value={formatNumber(summary.total_items)}
          hint={`${formatNumber(summary.available_items)} with stock available`}
          icon={BoxIcon}
          loading={dashboardResource.loading}
        />
        <StatCard
          label="New contact messages"
          value={formatNumber(summary.new_contact_messages)}
          hint="Public enquiries to review"
          icon={MessageIcon}
          tone="neutral"
          loading={dashboardResource.loading}
        />
        <StatCard
          label="Unread notifications"
          value={formatNumber(summary.unread_notifications)}
          hint="Across all active accounts"
          icon={AlertIcon}
          tone={Number(summary.unread_notifications) > 0 ? "warning" : "neutral"}
          loading={dashboardResource.loading}
        />
      </div>

      <div className="lab-dashboard-grid lab-dashboard-grid--admin">
        <Panel
          title="Attendance overview"
          subtitle="Last 30 days of lab activity."
          actions={
            <AppLink to="/app/attendance" className="lab-text-link">
              Attendance
              <ArrowRightIcon className="h-3.5 w-3.5" />
            </AppLink>
          }
          className="lab-span-2"
        >
          {attendanceResource.loading ? (
            <LoadingState rows={4} compact />
          ) : attendanceResource.error ? (
            <ErrorState error={attendanceResource.error} onRetry={attendanceResource.reload} compact />
          ) : (
            <div className="lab-attendance-overview">
              <div className="lab-attendance-overview__stats">
                <div>
                  <span>Open sessions</span>
                  <strong>{formatNumber(attendance?.summary?.total_login_events)}</strong>
                </div>
                <div>
                  <span>Check-outs</span>
                  <strong>{formatNumber(attendance?.summary?.total_logout_events)}</strong>
                </div>
                <div>
                  <span>Unique visitors</span>
                  <strong>{formatNumber(attendance?.summary?.unique_members_present)}</strong>
                </div>
                <div>
                  <span>Present now</span>
                  <strong>{formatNumber(attendance?.current_occupancy?.present_now)}</strong>
                </div>
              </div>

              <div className="lab-mini-chart" aria-label="Unique members per day, last ten recorded days">
                {attendanceDays.length ? (
                  attendanceDays.map((day) => {
                    const max = Math.max(...attendanceDays.map((entry) => Number(entry.members) || 1), 1);
                    const height = Math.max(((Number(day.members) || 0) / max) * 100, 4);
                    const parts = formatMonthDay(day.date);
                    return (
                      <div key={day.date} className="lab-mini-chart__column">
                        <span className="lab-mini-chart__value">{day.members ?? 0}</span>
                        <span className="lab-mini-chart__bar" style={{ height: `${height}%` }} />
                        <span className="lab-mini-chart__label">{parts.day} {parts.month}</span>
                      </div>
                    );
                  })
                ) : (
                  <p className="lab-muted lab-small">No attendance recorded in this period.</p>
                )}
              </div>

              <div className="lab-attendance-overview__branches">
                <h3>By branch</h3>
                <MiniBars
                  data={attendance?.by_branch}
                  valueKey="unique_members"
                  labelKey="branch"
                  emptyMessage="No branch activity in this period."
                />
              </div>
            </div>
          )}
        </Panel>

        <Panel
          title="Pending work"
          subtitle="Requests that need an administrator."
          actions={
            <AppLink to="/app/borrowing" className="lab-text-link">
              Borrowing
              <ArrowRightIcon className="h-3.5 w-3.5" />
            </AppLink>
          }
        >
          <ul className="lab-pending-list">
            <li>
              <span className="lab-pending-list__icon"><CheckSquareIcon className="h-4 w-4" /></span>
              <div>
                <strong>Task submissions</strong>
                <p>Waiting for verification</p>
              </div>
              <Badge tone={Number(summary.pending_tasks) > 0 ? "warning" : "success"}>
                {formatNumber(summary.pending_tasks)}
              </Badge>
            </li>
            <li>
              <span className="lab-pending-list__icon"><BoxIcon className="h-4 w-4" /></span>
              <div>
                <strong>Borrow requests</strong>
                <p>Waiting for approval</p>
              </div>
              <Badge tone={Number(summary.pending_borrow_requests) > 0 ? "warning" : "success"}>
                {formatNumber(summary.pending_borrow_requests)}
              </Badge>
            </li>
            <li>
              <span className="lab-pending-list__icon"><ArrowRightIcon className="h-4 w-4" /></span>
              <div>
                <strong>Return requests</strong>
                <p>Waiting for confirmation</p>
              </div>
              <Badge tone={Number(summary.pending_return_requests) > 0 ? "warning" : "success"}>
                {formatNumber(summary.pending_return_requests)}
              </Badge>
            </li>
            <li>
              <span className="lab-pending-list__icon"><MessageIcon className="h-4 w-4" /></span>
              <div>
                <strong>Contact messages</strong>
                <p>New public enquiries</p>
              </div>
              <Badge tone={Number(summary.new_contact_messages) > 0 ? "warning" : "success"}>
                {formatNumber(summary.new_contact_messages)}
              </Badge>
            </li>
          </ul>
        </Panel>

        <Panel
          title="Recent announcements"
          subtitle="Newest notices from the lab team."
          actions={
            <AppLink to="/app/announcements" className="lab-text-link">
              Manage
              <ArrowRightIcon className="h-3.5 w-3.5" />
            </AppLink>
          }
        >
          {announcementsResource.loading ? (
            <LoadingState rows={4} compact />
          ) : announcementsResource.error ? (
            <ErrorState error={announcementsResource.error} onRetry={announcementsResource.reload} compact />
          ) : announcements.length ? (
            <ul className="lab-feed">
              {announcements.map((announcement) => (
                <li key={announcement.id} className="lab-feed__item">
                  <span className="lab-feed__icon"><MegaphoneIcon className="h-4 w-4" /></span>
                  <div className="lab-feed__copy">
                    <AppLink to="/app/announcements" className="lab-feed__title">
                      {announcement.title}
                    </AppLink>
                    <p>
                      {announcement.is_published ? "Published" : "Draft"}
                      {announcement.published_at ? ` · ${formatRelative(announcement.published_at)}` : ""}
                    </p>
                  </div>
                  {announcement.is_pinned && <Badge tone="accent">Pinned</Badge>}
                </li>
              ))}
            </ul>
          ) : (
            <EmptyState
              icon={MegaphoneIcon}
              title="No announcements"
              description="Create the first announcement from the announcements workspace."
            />
          )}
        </Panel>

        <Panel
          title="Upcoming events"
          subtitle="Scheduled workshops and sessions."
          actions={
            <AppLink to="/app/events" className="lab-text-link">
              Manage
              <ArrowRightIcon className="h-3.5 w-3.5" />
            </AppLink>
          }
        >
          {eventsResource.loading ? (
            <LoadingState rows={4} compact />
          ) : eventsResource.error ? (
            <ErrorState error={eventsResource.error} onRetry={eventsResource.reload} compact />
          ) : events.length ? (
            <ul className="lab-event-mini-list">
              {events.map((event) => {
                const parts = formatMonthDay(event.date);
                return (
                  <li key={event.id} className="lab-event-mini">
                    <div className="lab-event-mini__date">
                      <span>{parts.month}</span>
                      <strong>{parts.day}</strong>
                    </div>
                    <div className="lab-event-mini__copy">
                      <AppLink to="/app/events" className="lab-feed__title">
                        {event.title}
                      </AppLink>
                      <p>{formatTime(event.start_time)} · {event.venue || "Venue TBA"}</p>
                    </div>
                    <Badge tone="info">{titleCase(event.event_type)}</Badge>
                  </li>
                );
              })}
            </ul>
          ) : (
            <EmptyState
              icon={CalendarIcon}
              title="No upcoming events"
              description="Schedule the next lab session from the events workspace."
            />
          )}
        </Panel>

        <Panel
          title="Recent activity"
          subtitle="Latest records from across the platform."
          className="lab-span-2"
        >
          <ActivityList activity={activity} />
        </Panel>

        <Panel title="Quick actions" subtitle="Common admin workflows." className="lab-span-2">
          <div className="lab-quick-grid lab-quick-grid--3">
            <AppLink to="/app/members" className="lab-quick-link">
              <span><UsersIcon className="h-4 w-4" /></span>
              <div><strong>Members</strong><p>Search, inspect and update accounts</p></div>
              <ArrowRightIcon className="h-4 w-4" />
            </AppLink>
            <AppLink to="/app/analytics" className="lab-quick-link">
              <span><ChartIcon className="h-4 w-4" /></span>
              <div><strong>Analytics</strong><p>Review members, attendance and operations</p></div>
              <ArrowRightIcon className="h-4 w-4" />
            </AppLink>
            <AppLink to="/app/inventory" className="lab-quick-link">
              <span><BoxIcon className="h-4 w-4" /></span>
              <div><strong>Inventory</strong><p>Manage stock and available equipment</p></div>
              <ArrowRightIcon className="h-4 w-4" />
            </AppLink>
          </div>
        </Panel>
      </div>

      <div className="lab-trend-strip">
        <div>
          <p>Members by academic year</p>
          <MiniBars
            data={Object.entries(trends.members_by_academic_year || {}).map(([year, count]) => ({
              year,
              count,
            }))}
            valueKey="count"
            labelKey="year"
            emptyMessage="No member data."
          />
        </div>
        <div>
          <p>Inventory by category</p>
          <MiniBars
            data={Object.entries(trends.inventory_by_category || {}).map(([category, count]) => ({
              category,
              count,
            }))}
            valueKey="count"
            labelKey="category"
            emptyMessage="No inventory data."
          />
        </div>
      </div>
    </>
  );
}
