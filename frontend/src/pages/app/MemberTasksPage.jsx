import { useMemo, useState } from "react";
import { tasksApi } from "../../api/services";
import { useApiResource } from "../../hooks/useApiResource";
import { Modal } from "../../components/app/Modal";
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
  Select,
  StatCard,
  StatusBadge,
  Tabs,
} from "../../components/app/Ui";
import {
  CalendarIcon,
  CheckCircleIcon,
  CheckSquareIcon,
  ClockIcon,
  ExternalIcon,
  RefreshIcon,
  SendIcon,
  WarningIcon,
} from "../../components/app/AppIcons";
import {
  formatDate,
  formatDateTime,
  formatRelative,
  safeHttpUrl,
  titleCase,
} from "../../utils/format";

const VIEW_TABS = [
  { value: "all", label: "All" },
  { value: "open", label: "Open" },
  { value: "submitted", label: "Submitted" },
  { value: "verified", label: "Verified" },
  { value: "overdue", label: "Overdue" },
];

const TASK_TYPES = ["assignment", "project", "report", "activity", "other"];

function isOverdue(task) {
  if (!task.deadline) return false;
  const deadline = new Date(`${task.deadline}T23:59:59`);
  return deadline.getTime() < Date.now();
}

function statusTone(task) {
  if (task.is_verified) return "success";
  if (task.is_submitted) return "warning";
  if (isOverdue(task)) return "danger";
  return "neutral";
}

function statusLabel(task) {
  if (task.is_verified) return "Verified";
  if (task.is_submitted) return "Submitted";
  if (task.submission_status === "rejected") return "Changes requested";
  if (isOverdue(task)) return "Overdue";
  return "Not submitted";
}

function TaskDetail({ task, loading, error, submitting, onSubmit, onClose }) {
  const canSubmit =
    task &&
    !task.is_verified &&
    task.submission_status !== "submitted" &&
    task.submission_status !== "verified";

  return (
    <Modal
      open
      onClose={onClose}
      title={task?.title || "Task details"}
      description={task?.task_type ? titleCase(task.task_type) : undefined}
      size="lg"
      footer={
        <>
          <AppButton variant="outline" onClick={onClose} disabled={submitting}>
            Close
          </AppButton>
          {canSubmit && (
            <AppButton variant="primary" onClick={onSubmit} loading={submitting}>
              <SendIcon className="h-4 w-4" />
              Mark as submitted
            </AppButton>
          )}
        </>
      }
    >
      {loading ? (
        <LoadingState rows={6} />
      ) : error ? (
        <ErrorState error={error} title="Task could not be loaded" />
      ) : task ? (
        <article className="lab-task-detail">
          <div className="lab-task-detail__badges">
            <Badge tone="info">{titleCase(task.task_type)}</Badge>
            <StatusBadge status={task.submission_status || (isOverdue(task) ? "overdue" : "pending")}>
              {statusLabel(task)}
            </StatusBadge>
            {task.deadline && (
              <Badge tone={isOverdue(task) ? "danger" : "neutral"} dot>
                Deadline {formatDate(task.deadline)}
              </Badge>
            )}
          </div>

          <div className="lab-task-detail__grid">
            <div>
              <span>Published</span>
              <strong>{task.published_at ? formatDate(task.published_at) : "—"}</strong>
            </div>
            <div>
              <span>Deadline</span>
              <strong>{formatDate(task.deadline)}</strong>
            </div>
            <div>
              <span>Set by</span>
              <strong>{task.created_by?.name || "Edwins Lab"}</strong>
            </div>
            <div>
              <span>Submission</span>
              <strong>{statusLabel(task)}</strong>
            </div>
          </div>

          {task.description && (
            <section>
              <h3>Description</h3>
              <p>{task.description}</p>
            </section>
          )}
          {task.instructions && (
            <section>
              <h3>Instructions</h3>
              <p>{task.instructions}</p>
            </section>
          )}
          {task.required_attachments && (
            <section>
              <h3>Required attachments</h3>
              <p>{task.required_attachments}</p>
            </section>
          )}
          {task.submission?.submitted_at && (
            <section>
              <h3>Your submission</h3>
              <p>
                Submitted {formatDateTime(task.submission.submitted_at)}
                {task.submission.verified_at
                  ? ` · reviewed ${formatRelative(task.submission.verified_at)}`
                  : ""}
              </p>
            </section>
          )}
          {safeHttpUrl(task.external_link) && (
            <a
              className="lab-social-link"
              href={safeHttpUrl(task.external_link)}
              target="_blank"
              rel="noreferrer noopener"
            >
              Open external resource
              <ExternalIcon className="h-3.5 w-3.5" />
            </a>
          )}
        </article>
      ) : null}
    </Modal>
  );
}

export default function MemberTasksPage() {
  const [view, setView] = useState("all");
  const [type, setType] = useState("");
  const [page, setPage] = useState(1);
  const [detail, setDetail] = useState(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detailError, setDetailError] = useState(null);
  const [submitting, setSubmitting] = useState(false);
  const [notice, setNotice] = useState(null);
  const [actionError, setActionError] = useState(null);

  const query = useMemo(
    () => ({
      page,
      per_page: 20,
      task_type: type || undefined,
    }),
    [page, type],
  );

  const resource = useApiResource(
    () => tasksApi.list(query),
    `member-tasks-${JSON.stringify(query)}`,
  );

  const tasks = useMemo(() => resource.data?.tasks ?? [], [resource.data]);
  const pagination = resource.data?.pagination;

  const filtered = useMemo(() => {
    return tasks.filter((task) => {
      if (view === "open") return !task.is_submitted && !task.is_verified && !isOverdue(task);
      if (view === "submitted") return task.is_submitted && !task.is_verified;
      if (view === "verified") return task.is_verified;
      if (view === "overdue") return isOverdue(task) && !task.is_verified;
      return true;
    });
  }, [tasks, view]);

  const openTask = async (task) => {
    setDetail(task);
    setDetailLoading(true);
    setDetailError(null);
    try {
      const result = await tasksApi.get(task.id);
      setDetail(result.task);
    } catch (error) {
      setDetailError(error);
    } finally {
      setDetailLoading(false);
    }
  };

  const submitTask = async () => {
    if (!detail || submitting) return;
    setSubmitting(true);
    setActionError(null);
    try {
      const result = await tasksApi.submit(detail.id);
      setNotice(result.message || "Task marked as submitted.");
      setDetail((current) =>
        current
          ? {
              ...current,
              is_submitted: true,
              submission_status: result.submission?.status || "submitted",
              submission: { ...(current.submission || {}), ...(result.submission || {}) },
            }
          : current,
      );
      resource.reload();
    } catch (error) {
      setActionError(error);
    } finally {
      setSubmitting(false);
    }
  };

  const openCount = tasks.filter(
    (task) => !task.is_submitted && !task.is_verified && !isOverdue(task),
  ).length;
  const submittedCount = tasks.filter((task) => task.is_submitted && !task.is_verified).length;
  const verifiedCount = tasks.filter((task) => task.is_verified).length;
  const overdueCount = tasks.filter((task) => isOverdue(task) && !task.is_verified).length;

  return (
    <>
      <PageHeader
        eyebrow="Member portal"
        title="My tasks"
        description="Assignments, projects and activities set for your academic year."
        actions={
          <AppButton variant="outline" onClick={resource.reload} loading={resource.loading}>
            <RefreshIcon className="h-4 w-4" />
            Refresh
          </AppButton>
        }
      />

      {notice && (
        <Alert tone="success" title="Done" onDismiss={() => setNotice(null)}>
          {notice}
        </Alert>
      )}
      {actionError && (
        <Alert tone="danger" title="Task action failed" onDismiss={() => setActionError(null)}>
          {actionError.message}
        </Alert>
      )}

      <div className="lab-stat-grid">
        <StatCard label="Open" value={openCount} hint="Waiting for your submission" icon={ClockIcon} tone="accent" loading={resource.loading} />
        <StatCard label="Submitted" value={submittedCount} hint="Awaiting review" icon={SendIcon} loading={resource.loading} />
        <StatCard label="Verified" value={verifiedCount} hint="Reviewed by an admin" icon={CheckCircleIcon} loading={resource.loading} />
        <StatCard label="Overdue" value={overdueCount} hint="Past the deadline" icon={WarningIcon} tone={overdueCount ? "warning" : "neutral"} loading={resource.loading} />
      </div>

      <Panel padded={false}>
        <div className="lab-toolbar lab-toolbar--wrap">
          <Tabs
            tabs={VIEW_TABS}
            value={view}
            onChange={setView}
            label="Task view"
          />
          <div className="lab-toolbar__right">
            <Select
              value={type}
              onChange={(event) => {
                setType(event.target.value);
                setPage(1);
              }}
              aria-label="Filter by task type"
              className="lab-select--compact"
            >
              <option value="">All types</option>
              {TASK_TYPES.map((taskType) => (
                <option key={taskType} value={taskType}>{titleCase(taskType)}</option>
              ))}
            </Select>
          </div>
        </div>

        <div className="lab-list-panel__body">
          {resource.loading ? (
            <LoadingState rows={6} />
          ) : resource.error ? (
            <ErrorState error={resource.error} onRetry={resource.reload} title="Tasks are unavailable" />
          ) : filtered.length === 0 ? (
            <EmptyState
              icon={CheckSquareIcon}
              title={view === "all" ? "No tasks yet" : "Nothing in this view"}
              description={
                view === "all"
                  ? "When an admin publishes a task for your academic year it will appear here."
                  : "Try another filter or check back after your admins publish more work."
              }
            />
          ) : (
            <div className="lab-task-list">
              {filtered.map((task) => (
                <article key={task.id} className="lab-task-card">
                  <div className="lab-task-card__main">
                    <div className="lab-task-card__top">
                      <Badge tone="info">{titleCase(task.task_type)}</Badge>
                      <Badge tone={statusTone(task)} dot>{statusLabel(task)}</Badge>
                      <Badge tone={isOverdue(task) ? "danger" : "neutral"}>
                        <CalendarIcon className="h-3 w-3" />
                        {formatDate(task.deadline)}
                      </Badge>
                    </div>
                    <button
                      type="button"
                      className="lab-task-card__title"
                      onClick={() => openTask(task)}
                    >
                      {task.title}
                    </button>
                    <div className="lab-task-card__meta">
                      <span>
                        <ClockIcon className="h-3.5 w-3.5" />
                        {task.published_at ? `Published ${formatRelative(task.published_at)}` : "Published"}
                      </span>
                      <span>
                        <CalendarIcon className="h-3.5 w-3.5" />
                        Deadline {formatDate(task.deadline)}
                      </span>
                    </div>
                    {task.required_attachments && (
                      <p className="lab-task-card__note">Required: {task.required_attachments}</p>
                    )}
                  </div>
                  <div className="lab-task-card__actions">
                    <AppButton variant="outline" size="sm" onClick={() => openTask(task)}>
                      Open
                    </AppButton>
                  </div>
                </article>
              ))}
            </div>
          )}
        </div>

        {pagination && (
          <Pagination
            page={pagination.page}
            totalPages={pagination.total_pages}
            totalCount={pagination.total_count}
            onPageChange={setPage}
            loading={resource.loading}
          />
        )}
      </Panel>

      {detail && (
        <TaskDetail
          task={detail}
          loading={detailLoading}
          error={detailError}
          submitting={submitting}
          onSubmit={submitTask}
          onClose={() => {
            if (submitting) return;
            setDetail(null);
            setDetailError(null);
          }}
        />
      )}
    </>
  );
}
