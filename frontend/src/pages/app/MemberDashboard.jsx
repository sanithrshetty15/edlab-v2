import { useAuth } from "../../auth/context";
import { attendanceApi, announcementsApi, eventsApi, membersApi, messagesApi } from "../../api/services";
import { useApiResource } from "../../hooks/useApiResource";
import { AppLink } from "../../components/app/Link";
import {
  ArrowRightIcon,
  CalendarIcon,
  CheckCircleIcon,
  ClockIcon,
  InfoIcon,
  MegaphoneIcon,
  MessageIcon,
  PinIcon,
  ShieldIcon,
  UserIcon,
  WarningIcon,
} from "../../components/app/AppIcons";
import {
  Avatar,
  Badge,
  EmptyState,
  ErrorState,
  LoadingState,
  PageHeader,
  Panel,
  StatCard,
  StatusBadge,
} from "../../components/app/Ui";
import {
  daysAgoIso,
  excerpt,
  formatClockTime,
  formatDate,
  formatMonthDay,
  formatNumber,
  formatRelative,
  formatTime,
  getGreeting,
  toIsoDate,
  titleCase,
} from "../../utils/format";

function AttendancePreview({ resource }) {
  if (resource.loading) return <LoadingState rows={3} compact />;
  if (resource.error) {
    return <ErrorState error={resource.error} onRetry={resource.reload} compact />;
  }

  const records = resource.data?.attendance ?? [];
  if (!records.length) {
    return (
      <EmptyState
        icon={ClockIcon}
        title="No check-ins yet"
        description="Your lab attendance will appear here after your first biometric or manual check-in."
      />
    );
  }

  return (
    <ul className="lab-timeline">
      {records.map((record) => (
        <li key={record.id} className="lab-timeline__item">
          <span className={`lab-timeline__dot ${record.exit_time ? "is-done" : "is-live"}`} />
          <div className="lab-timeline__content">
            <div className="lab-timeline__row">
              <span>{formatDate(record.entry_date)}</span>
              {!record.exit_time && <Badge tone="accent">In lab</Badge>}
            </div>
            <p>
              {formatClockTime(record.entry_time?.slice(11, 16) || record.entry_time)} in
              {record.exit_time
                ? ` · ${formatClockTime(record.exit_time?.slice(11, 16) || record.exit_time)} out`
                : " · session open"}
            </p>
          </div>
        </li>
      ))}
    </ul>
  );
}

function AnnouncementPreview({ resource }) {
  if (resource.loading) return <LoadingState rows={3} compact />;
  if (resource.error) {
    return <ErrorState error={resource.error} onRetry={resource.reload} compact />;
  }

  const announcements = resource.data?.announcements ?? [];
  if (!announcements.length) {
    return (
      <EmptyState
        icon={MegaphoneIcon}
        title="No announcements yet"
        description="Published lab announcements will appear here."
      />
    );
  }

  return (
    <ul className="lab-feed">
      {announcements.map((announcement) => (
        <li key={announcement.id} className="lab-feed__item">
          <span className="lab-feed__icon">
            {announcement.is_pinned ? <PinIcon className="h-4 w-4" /> : <MegaphoneIcon className="h-4 w-4" />}
          </span>
          <div className="lab-feed__copy">
            <AppLink to="/app/announcements" className="lab-feed__title">
              {announcement.title}
            </AppLink>
            <p>
              {announcement.published_at ? formatRelative(announcement.published_at) : "Scheduled"}
              {announcement.created_by?.name ? ` · ${announcement.created_by.name}` : ""}
            </p>
          </div>
          {announcement.is_pinned && <Badge tone="accent">Pinned</Badge>}
        </li>
      ))}
    </ul>
  );
}

function EventPreview({ resource }) {
  if (resource.loading) return <LoadingState rows={3} compact />;
  if (resource.error) {
    return <ErrorState error={resource.error} onRetry={resource.reload} compact />;
  }

  const events = resource.data?.events ?? [];
  if (!events.length) {
    return (
      <EmptyState
        icon={CalendarIcon}
        title="No upcoming events"
        description="Workshops, talks and hackathons will be listed here when scheduled."
      />
    );
  }

  return (
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
              <p>
                {formatTime(event.start_time)} · {event.venue || "Venue to be announced"}
              </p>
            </div>
            <Badge tone="info">{titleCase(event.event_type)}</Badge>
          </li>
        );
      })}
    </ul>
  );
}

function MessagePreview({ resource }) {
  if (resource.loading) return <LoadingState rows={3} compact />;
  if (resource.error) {
    return <ErrorState error={resource.error} onRetry={resource.reload} compact />;
  }

  const messages = resource.data?.messages ?? [];
  if (!messages.length) {
    return (
      <EmptyState
        icon={MessageIcon}
        title="No messages yet"
        description="The lab conversation will appear here once someone posts."
      />
    );
  }

  return (
    <ul className="lab-feed">
      {[...messages].reverse().map((message) => (
        <li key={message.id} className="lab-feed__item">
          <span className="lab-feed__icon">
            <MessageIcon className="h-4 w-4" />
          </span>
          <div className="lab-feed__copy">
            <AppLink to="/app/messages" className="lab-feed__title">
              {message.user?.name || "Administrator"}
            </AppLink>
            <p>{excerpt(message.content, 110)}</p>
          </div>
          <span className="lab-feed__time">{formatRelative(message.created_at)}</span>
        </li>
      ))}
    </ul>
  );
}

export default function MemberDashboard() {
  const auth = useAuth();
  const profileResource = useApiResource(() => membersApi.profile(), "member-dashboard-profile");
  const todayResource = useApiResource(() => attendanceApi.today(), "member-dashboard-today");
  const recentAttendance = useApiResource(
    () => attendanceApi.list({ per_page: 6 }),
    "member-dashboard-attendance",
  );
  const dailyResource = useApiResource(
    () => attendanceApi.dailySummary({ date_from: daysAgoIso(29), date_to: toIsoDate(new Date()) }),
    "member-dashboard-daily",
  );
  const announcementsResource = useApiResource(
    () => announcementsApi.list({ published: "true", per_page: 5 }),
    "member-dashboard-announcements",
  );
  const eventsResource = useApiResource(
    () => eventsApi.list({ upcoming: "true", per_page: 5 }),
    "member-dashboard-events",
  );
  const messagesResource = useApiResource(
    () => messagesApi.list({ per_page: 5 }),
    "member-dashboard-messages",
  );

  const member = profileResource.data?.member || auth.user?.member || {};
  const displayName = member.name || auth.user?.name || auth.user?.email || "Member";
  const firstName = displayName.split(" ")[0];
  const todaySummary = todayResource.data?.summary;
  const visitsLast30 = (dailyResource.data?.daily_summary ?? []).reduce(
    (total, day) => total + Number(day.total_events || 0),
    0,
  );
  const currentStatus = todaySummary?.total_present ?? 0;

  return (
    <>
      <PageHeader
        eyebrow="Member workspace"
        title={`${getGreeting()}, ${firstName}.`}
        description="Your lab activity, notices and conversations in one place."
        actions={
          <AppLink to="/app/profile" className="lab-button lab-button--outline lab-button--md">
            <UserIcon className="h-4 w-4" />
            View profile
          </AppLink>
        }
      />

      <div className="lab-dashboard-grid">
        <Panel className="lab-profile-summary lab-span-2" padded={false}>
          <div className="lab-profile-summary__banner" aria-hidden="true" />
          <div className="lab-profile-summary__body">
            <Avatar name={displayName} size="xl" />
            <div className="lab-profile-summary__identity">
              <div className="lab-profile-summary__name-row">
                <h2>{displayName}</h2>
                <StatusBadge status={member.status} />
              </div>
              <p>{auth.user?.email || member.email}</p>
              <div className="lab-profile-summary__chips">
                {member.usn && <Badge tone="neutral">USN {member.usn}</Badge>}
                {member.roll_number && <Badge tone="neutral">Roll {member.roll_number}</Badge>}
                {member.branch && <Badge tone="info">{member.branch}</Badge>}
                {member.section && <Badge tone="neutral">Section {member.section}</Badge>}
              </div>
            </div>
          </div>
          <div className="lab-profile-summary__details">
            <div>
              <span>Academic year</span>
              <strong>{member.academic_year ? `Year ${member.academic_year}` : "—"}</strong>
            </div>
            <div>
              <span>Login ID</span>
              <strong>{member.login_id || "—"}</strong>
            </div>
            <div>
              <span>Member since</span>
              <strong>{member.created_at ? formatDate(member.created_at) : "—"}</strong>
            </div>
          </div>
          {profileResource.error && (
            <div className="lab-profile-summary__warning">
              <WarningIcon className="h-4 w-4" />
              Full member profile is unavailable right now.
            </div>
          )}
        </Panel>

        <Panel title="Account status" subtitle="Your current access and account health." className="lab-account-panel">
          <ul className="lab-status-list">
            <li>
              <span className={`lab-status-dot ${auth.isAuthenticated ? "is-live" : "is-muted"}`} />
              <div>
                <strong>Session</strong>
                <span>Signed in on this browser tab</span>
              </div>
              <Badge tone="success">Active</Badge>
            </li>
            <li>
              <span className={`lab-status-dot ${member.status === "active" ? "is-live" : "is-muted"}`} />
              <div>
                <strong>Member status</strong>
                <span>{member.status ? "Official membership record" : "Profile service unavailable"}</span>
              </div>
              <StatusBadge status={member.status}>{member.status ? titleCase(member.status) : "Unknown"}</StatusBadge>
            </li>
            <li>
              <span className="lab-status-dot is-live" />
              <div>
                <strong>Password</strong>
                <span>You can change it from account settings</span>
              </div>
              <Badge tone="success">Current</Badge>
            </li>
          </ul>
          <div className="lab-inline-link-row">
            <AppLink to="/app/settings" className="lab-text-link">
              Open account settings
              <ArrowRightIcon className="h-3.5 w-3.5" />
            </AppLink>
          </div>
        </Panel>

        <div className="lab-stat-grid lab-stat-grid--3 lab-span-3">
          <StatCard
            label="Currently in the lab"
            value={formatNumber(currentStatus)}
            hint={todaySummary?.date ? `As of ${formatDate(todaySummary.date)}` : "Today"}
            icon={CheckCircleIcon}
            tone={currentStatus > 0 ? "accent" : "neutral"}
            loading={todayResource.loading}
          />
          <StatCard
            label="Visits today"
            value={formatNumber(todayResource.data?.today?.length ?? todaySummary?.total_visited ?? 0)}
            hint="Check-ins recorded today"
            icon={ClockIcon}
            loading={todayResource.loading}
          />
          <StatCard
            label="Visits in 30 days"
            value={formatNumber(visitsLast30)}
            hint="All recorded lab sessions"
            icon={CalendarIcon}
            loading={dailyResource.loading}
          />
        </div>

        <Panel
          title="Recent attendance"
          subtitle="Your latest lab check-ins and check-outs."
          className="lab-span-2"
          actions={
            <AppLink to="/app/profile" className="lab-text-link">
              View profile
              <ArrowRightIcon className="h-3.5 w-3.5" />
            </AppLink>
          }
        >
          <AttendancePreview resource={recentAttendance} />
        </Panel>

        <Panel
          title="Latest announcements"
          subtitle="Published notices from the lab team."
          actions={
            <AppLink to="/app/announcements" className="lab-text-link">
              All announcements
              <ArrowRightIcon className="h-3.5 w-3.5" />
            </AppLink>
          }
        >
          <AnnouncementPreview resource={announcementsResource} />
        </Panel>

        <Panel
          title="Upcoming events"
          subtitle="What is next on the lab calendar."
          actions={
            <AppLink to="/app/events" className="lab-text-link">
              All events
              <ArrowRightIcon className="h-3.5 w-3.5" />
            </AppLink>
          }
        >
          <EventPreview resource={eventsResource} />
        </Panel>

        <Panel
          title="Recent messages"
          subtitle="The latest notes from the lab channel."
          className="lab-span-2"
          actions={
            <AppLink to="/app/messages" className="lab-text-link">
              Open messages
              <ArrowRightIcon className="h-3.5 w-3.5" />
            </AppLink>
          }
        >
          <MessagePreview resource={messagesResource} />
        </Panel>

        <Panel title="Quick navigation" subtitle="Jump back into the work." className="lab-quick-panel lab-span-3">
          <div className="lab-quick-grid">
            <AppLink to="/app/announcements" className="lab-quick-link">
              <span><MegaphoneIcon className="h-4 w-4" /></span>
              <div>
                <strong>Announcements</strong>
                <p>Read the latest lab notices</p>
              </div>
              <ArrowRightIcon className="h-4 w-4" />
            </AppLink>
            <AppLink to="/app/events" className="lab-quick-link">
              <span><CalendarIcon className="h-4 w-4" /></span>
              <div>
                <strong>Events</strong>
                <p>See workshops and sessions</p>
              </div>
              <ArrowRightIcon className="h-4 w-4" />
            </AppLink>
            <AppLink to="/app/messages" className="lab-quick-link">
              <span><MessageIcon className="h-4 w-4" /></span>
              <div>
                <strong>Messages</strong>
                <p>Join the lab conversation</p>
              </div>
              <ArrowRightIcon className="h-4 w-4" />
            </AppLink>
            <AppLink to="/app/settings" className="lab-quick-link">
              <span><ShieldIcon className="h-4 w-4" /></span>
              <div>
                <strong>Account</strong>
                <p>Password and session controls</p>
              </div>
              <ArrowRightIcon className="h-4 w-4" />
            </AppLink>
          </div>
        </Panel>

        <div className="lab-info-strip lab-span-3">
          <InfoIcon className="h-4 w-4" />
          <p>
            Attendance is recorded through the lab biometric device or by an administrator.
            Contact the lab team if a session is missing.
          </p>
        </div>
      </div>
    </>
  );
}
