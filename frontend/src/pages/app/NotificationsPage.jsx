import { useMemo, useState } from "react";
import { useAuth } from "../../auth/context";
import { notificationsApi } from "../../api/services";
import { useApiResource } from "../../hooks/useApiResource";
import { navigate } from "../../utils/router";
import { AppLink } from "../../components/app/Link";
import {
  Alert,
  AppButton,
  Badge,
  EmptyState,
  ErrorState,
  LoadingState,
  PageHeader,
  Pagination,
  Panel,
  Tabs,
} from "../../components/app/Ui";
import {
  BellIcon,
  CalendarIcon,
  CheckCircleIcon,
  CheckSquareIcon,
  MegaphoneIcon,
  MessageIcon,
  ShieldIcon,
  TrashIcon,
} from "../../components/app/AppIcons";
import { formatRelative, titleCase } from "../../utils/format";

const VIEW_TABS = [
  { value: "all", label: "All" },
  { value: "unread", label: "Unread" },
  { value: "read", label: "Read" },
];

const TYPE_META = {
  message: { icon: MessageIcon, label: "Message" },
  message_reply: { icon: MessageIcon, label: "Reply" },
  message_forward: { icon: MessageIcon, label: "Forwarded" },
  announcement: { icon: MegaphoneIcon, label: "Announcement" },
  event: { icon: CalendarIcon, label: "Event" },
  task: { icon: CheckSquareIcon, label: "Task" },
  submission_verification: { icon: CheckSquareIcon, label: "Submission" },
  submission_verified: { icon: CheckCircleIcon, label: "Verified" },
  inventory_approval: { icon: ShieldIcon, label: "Borrowing" },
  inventory_rejected: { icon: ShieldIcon, label: "Borrowing" },
  inventory_approved: { icon: ShieldIcon, label: "Borrowing" },
  return_confirmed: { icon: ShieldIcon, label: "Return" },
  password_reset: { icon: ShieldIcon, label: "Security" },
  password_changed: { icon: ShieldIcon, label: "Security" },
  contact_message: { icon: MessageIcon, label: "Contact" },
};

function metaFor(type) {
  return TYPE_META[type] || { icon: BellIcon, label: titleCase(type || "notification") };
}

function linkFor(notification, isAdmin) {
  switch (notification.subject_type) {
    case "Announcement":
      return "/app/announcements";
    case "Event":
      return "/app/events";
    case "Task":
    case "TaskSubmissionStatus":
      return isAdmin ? "/app/tasks" : "/app/my-tasks";
    case "GroupMessage":
      return "/app/messages";
    case "BorrowRecord":
    case "ReturnRequest":
      return isAdmin ? "/app/borrowing" : null;
    default:
      return null;
  }
}

function notifyChanged() {
  if (typeof window !== "undefined") {
    window.dispatchEvent(new CustomEvent("lab:notifications-changed"));
  }
}

export default function NotificationsPage() {
  const auth = useAuth();
  const [view, setView] = useState("all");
  const [page, setPage] = useState(1);
  const [notice, setNotice] = useState(null);
  const [actionError, setActionError] = useState(null);
  const [busyId, setBusyId] = useState(null);
  const [markingAll, setMarkingAll] = useState(false);

  const query = useMemo(
    () => ({
      page,
      per_page: 20,
      read: view === "all" ? undefined : view === "read" ? "true" : "false",
    }),
    [page, view],
  );

  const resource = useApiResource(
    () => notificationsApi.list(query),
    `notifications-${JSON.stringify(query)}`,
  );

  const notifications = resource.data?.notifications ?? [];
  const pagination = resource.data?.pagination;
  const unreadCount = pagination?.unread_count ?? 0;

  const updateLocal = (updater) => {
    resource.setData((current) => {
      if (!current) return current;
      return {
        ...current,
        notifications: updater(current.notifications || []),
        pagination: current.pagination
          ? { ...current.pagination, unread_count: Math.max(0, unreadCount - 1) }
          : current.pagination,
      };
    });
  };

  const openNotification = async (notification) => {
    const link = linkFor(notification, auth.isAdmin);
    if (!notification.is_read) {
      setBusyId(notification.id);
      try {
        await notificationsApi.markRead([notification.id]);
        updateLocal((items) =>
          items.map((item) =>
            item.id === notification.id
              ? { ...item, is_read: true, read_at: new Date().toISOString() }
              : item,
          ),
        );
        notifyChanged();
      } catch (error) {
        setActionError(error);
      } finally {
        setBusyId(null);
      }
    }
    if (link) navigate(link);
  };

  const markAllRead = async () => {
    if (markingAll) return;
    setMarkingAll(true);
    setActionError(null);
    try {
      await notificationsApi.markAllRead();
      setNotice("All notifications marked as read.");
      resource.reload();
      notifyChanged();
    } catch (error) {
      setActionError(error);
    } finally {
      setMarkingAll(false);
    }
  };

  const removeNotification = async (notification) => {
    if (busyId) return;
    setBusyId(notification.id);
    setActionError(null);
    try {
      await notificationsApi.remove(notification.id);
      resource.reload();
      notifyChanged();
    } catch (error) {
      setActionError(error);
    } finally {
      setBusyId(null);
    }
  };

  return (
    <>
      <PageHeader
        eyebrow="Notifications"
        title="Notification centre"
        description="Messages, announcements, task reviews and borrowing updates for your account."
        actions={
          <AppButton
            variant="outline"
            onClick={markAllRead}
            loading={markingAll}
            disabled={unreadCount === 0}
          >
            <CheckCircleIcon className="h-4 w-4" />
            Mark all read
          </AppButton>
        }
      />

      {notice && (
        <Alert tone="success" title="Done" onDismiss={() => setNotice(null)}>
          {notice}
        </Alert>
      )}
      {actionError && (
        <Alert tone="danger" title="Notification action failed" onDismiss={() => setActionError(null)}>
          {actionError.message}
        </Alert>
      )}

      <Panel padded={false}>
        <div className="lab-toolbar lab-toolbar--wrap">
          <Tabs
            tabs={VIEW_TABS.map((tab) =>
              tab.value === "unread" && unreadCount
                ? { ...tab, count: unreadCount }
                : tab,
            )}
            value={view}
            onChange={(value) => {
              setView(value);
              setPage(1);
            }}
            label="Notification filters"
          />
          <div className="lab-toolbar__right">
            <Badge tone={unreadCount ? "accent" : "neutral"} dot={unreadCount > 0}>
              {unreadCount} unread
            </Badge>
          </div>
        </div>

        <div className="lab-list-panel__body">
          {resource.loading ? (
            <LoadingState rows={6} />
          ) : resource.error ? (
            <ErrorState
              error={resource.error}
              onRetry={resource.reload}
              title="Notifications are unavailable"
            />
          ) : notifications.length === 0 ? (
            <EmptyState
              icon={BellIcon}
              title={view === "unread" ? "You're all caught up" : "No notifications yet"}
              description={
                view === "unread"
                  ? "New lab activity will appear here when it needs your attention."
                  : "Announcements, task updates and borrowing decisions will show up here."
              }
            />
          ) : (
            <ul className="lab-notification-list">
              {notifications.map((notification) => {
                const meta = metaFor(notification.notification_type);
                const Icon = meta.icon;
                const link = linkFor(notification, auth.isAdmin);
                return (
                  <li
                    key={notification.id}
                    className={`lab-notification ${notification.is_read ? "" : "is-unread"}`}
                  >
                    <button
                      type="button"
                      className="lab-notification__main"
                      onClick={() => openNotification(notification)}
                      disabled={busyId === notification.id}
                    >
                      <span className="lab-notification__icon">
                        <Icon className="h-4 w-4" />
                      </span>
                      <span className="lab-notification__copy">
                        <span className="lab-notification__top">
                          <strong>{notification.title}</strong>
                          <time>{formatRelative(notification.created_at)}</time>
                        </span>
                        {notification.body && (
                          <span className="lab-notification__body">{notification.body}</span>
                        )}
                        <span className="lab-notification__meta">
                          <Badge tone={notification.is_read ? "neutral" : "accent"}>
                            {meta.label}
                          </Badge>
                          {!notification.is_read && <span className="lab-unread-dot">Unread</span>}
                          {link && <span>Open</span>}
                        </span>
                      </span>
                    </button>
                    <div className="lab-notification__actions">
                      <button
                        type="button"
                        className="lab-icon-button lab-icon-button--ghost"
                        aria-label="Delete notification"
                        title="Delete notification"
                        disabled={busyId === notification.id}
                        onClick={() => removeNotification(notification)}
                      >
                        <TrashIcon className="h-4 w-4" />
                      </button>
                    </div>
                  </li>
                );
              })}
            </ul>
          )}
        </div>

        {pagination && pagination.total_pages > 1 && (
          <Pagination
            page={pagination.page}
            totalPages={pagination.total_pages}
            totalCount={pagination.total_count}
            onPageChange={setPage}
            loading={resource.loading}
          />
        )}
      </Panel>

      <p className="lab-notifications-footnote">
        Looking for the lab conversation?{" "}
        <AppLink to="/app/messages">Open messages</AppLink>.
      </p>
    </>
  );
}
