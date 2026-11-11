import { useMemo, useState } from "react";
import { membersManagementApi, tasksApi } from "../../../api/services";
import { useApiResource } from "../../../hooks/useApiResource";
import { ConfirmDialog, Modal } from "../../../components/app/Modal";
import {
  Alert,
  AppButton,
  Badge,
  Checkbox,
  EmptyState,
  ErrorState,
  Field,
  LoadingState,
  PageHeader,
  Pagination,
  Panel,
  Select,
  Tabs,
  TextArea,
  TextInput,
} from "../../../components/app/Ui";
import {
  CalendarIcon,
  CheckSquareIcon,
  EditIcon,
  ExternalIcon,
  PlusIcon,
  SearchIcon,
  TrashIcon,
  UsersIcon,
} from "../../../components/app/AppIcons";
import {
  formatDate,
  formatDateTime,
  formatRelative,
  safeHttpUrl,
  titleCase,
} from "../../../utils/format";

const TASK_TYPES = ["assignment", "project", "report", "activity", "other"];
const YEAR_OPTIONS = [1, 2, 3, 4, 5, 6, 7, 8];

const VIEW_TABS = [
  { value: "all", label: "All tasks" },
  { value: "upcoming", label: "Upcoming" },
  { value: "overdue", label: "Overdue" },
];

function deadlineTone(task) {
  if (!task.deadline) return "neutral";
  const deadline = new Date(`${task.deadline}T23:59:59`);
  const today = new Date();
  if (deadline < today) return "danger";
  const days = Math.ceil((deadline - today) / 86400000);
  if (days <= 3) return "warning";
  return "success";
}

function deadlineLabel(task) {
  if (!task.deadline) return "No deadline";
  const deadline = new Date(`${task.deadline}T23:59:59`);
  if (deadline < new Date()) return `Overdue · ${formatDate(task.deadline)}`;
  return `Due ${formatRelative(task.deadline)}`;
}

function TaskForm({ task, onClose, onSaved }) {
  const [title, setTitle] = useState(task?.title || "");
  const [description, setDescription] = useState(task?.description || "");
  const [taskType, setTaskType] = useState(task?.task_type || "assignment");
  const [deadline, setDeadline] = useState(task?.deadline || "");
  const [instructions, setInstructions] = useState(task?.instructions || "");
  const [externalLink, setExternalLink] = useState(task?.external_link || "");
  const [requiredAttachments, setRequiredAttachments] = useState(task?.required_attachments || "");
  const [targetYears, setTargetYears] = useState(task?.target_academic_years || []);
  const [errors, setErrors] = useState({});
  const [formError, setFormError] = useState(null);
  const [saving, setSaving] = useState(false);
  const editing = Boolean(task?.id);

  const toggleYear = (year) => {
    setTargetYears((current) =>
      current.includes(year) ? current.filter((item) => item !== year) : [...current, year].sort(),
    );
  };

  const handleSubmit = async (event) => {
    event.preventDefault();
    if (saving) return;

    const nextErrors = {};
    if (!title.trim()) nextErrors.title = "Title is required.";
    if (!deadline) nextErrors.deadline = "Choose a deadline.";
    if (!targetYears.length) nextErrors.targetYears = "Select at least one academic year.";
    if (externalLink.trim() && !safeHttpUrl(externalLink.trim())) {
      nextErrors.externalLink = "Use a valid http or https URL.";
    }
    setErrors(nextErrors);
    setFormError(null);
    if (Object.keys(nextErrors).length) return;

    setSaving(true);
    try {
      const payload = {
        title: title.trim(),
        description: description.trim(),
        task_type: taskType,
        deadline,
        instructions: instructions.trim(),
        external_link: externalLink.trim(),
        required_attachments: requiredAttachments.trim(),
        target_academic_years: targetYears,
      };
      const result = editing
        ? await tasksApi.update(task.id, payload)
        : await tasksApi.create(payload);
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
      title={editing ? "Edit task" : "New task"}
      description="Tasks are published to members in the selected academic years."
      size="lg"
      footer={
        <>
          <AppButton variant="ghost" onClick={onClose} disabled={saving}>
            Cancel
          </AppButton>
          <AppButton form="task-form" type="submit" variant="primary" loading={saving}>
            {editing ? "Save changes" : "Create task"}
          </AppButton>
        </>
      }
    >
      <form id="task-form" className="lab-form-stack" onSubmit={handleSubmit} noValidate>
        {formError && (
          <Alert tone="danger" title="Task could not be saved">
            {formError.message}
          </Alert>
        )}

        <Field label="Title" htmlFor="task-title" error={errors.title} required>
          <TextInput
            id="task-title"
            value={title}
            onChange={(event) => setTitle(event.target.value)}
            disabled={saving}
          />
        </Field>

        <div className="lab-form-grid">
          <Field label="Task type" htmlFor="task-type" required>
            <Select
              id="task-type"
              value={taskType}
              onChange={(event) => setTaskType(event.target.value)}
              disabled={saving}
            >
              {TASK_TYPES.map((type) => (
                <option key={type} value={type}>{titleCase(type)}</option>
              ))}
            </Select>
          </Field>
          <Field label="Deadline" htmlFor="task-deadline" error={errors.deadline} required>
            <TextInput
              id="task-deadline"
              type="date"
              value={deadline}
              onChange={(event) => setDeadline(event.target.value)}
              disabled={saving}
            />
          </Field>
        </div>

        <Field label="Description" htmlFor="task-description">
          <TextArea
            id="task-description"
            rows={4}
            value={description}
            onChange={(event) => setDescription(event.target.value)}
            disabled={saving}
          />
        </Field>

        <Field label="Instructions" htmlFor="task-instructions" hint="Shown to members as the submission brief.">
          <TextArea
            id="task-instructions"
            rows={4}
            value={instructions}
            onChange={(event) => setInstructions(event.target.value)}
            disabled={saving}
          />
        </Field>

        <div className="lab-form-grid">
          <Field label="External link" htmlFor="task-link" error={errors.externalLink}>
            <TextInput
              id="task-link"
              type="url"
              value={externalLink}
              onChange={(event) => setExternalLink(event.target.value)}
              placeholder="https://…"
              disabled={saving}
            />
          </Field>
          <Field label="Required attachments" htmlFor="task-attachments" hint="Describe any required files.">
            <TextInput
              id="task-attachments"
              value={requiredAttachments}
              onChange={(event) => setRequiredAttachments(event.target.value)}
              placeholder="e.g. Report PDF"
              disabled={saving}
            />
          </Field>
        </div>

        <Field label="Target academic years" error={errors.targetYears} required>
          <div className="lab-checkbox-grid">
            {YEAR_OPTIONS.map((year) => (
              <Checkbox
                key={year}
                label={`Year ${year}`}
                checked={targetYears.includes(year)}
                onChange={() => toggleYear(year)}
                disabled={saving}
              />
            ))}
          </div>
        </Field>
      </form>
    </Modal>
  );
}

function TaskDetail({ task, loading, error, onClose, onVerify }) {
  return (
    <Modal
      open
      onClose={onClose}
      title={task?.title || "Task"}
      size="lg"
      footer={
        <>
          <AppButton variant="ghost" onClick={onClose}>Close</AppButton>
          <AppButton variant="primary" onClick={onVerify} disabled={!task}>
            <CheckSquareIcon className="h-4 w-4" />
            Review submission
          </AppButton>
        </>
      }
    >
      {loading ? (
        <LoadingState rows={6} />
      ) : error ? (
        <ErrorState error={error} title="Task could not be loaded" />
      ) : task ? (
        <div className="lab-task-detail">
          <div className="lab-task-detail__badges">
            <Badge tone="info">{titleCase(task.task_type)}</Badge>
            <Badge tone={deadlineTone(task)} dot>{deadlineLabel(task)}</Badge>
          </div>
          <div className="lab-task-detail__grid">
            <div>
              <span>Published</span>
              <strong>{task.published_at ? formatDateTime(task.published_at) : "—"}</strong>
            </div>
            <div>
              <span>Deadline</span>
              <strong>{formatDate(task.deadline)}</strong>
            </div>
            <div>
              <span>Target years</span>
              <strong>{(task.target_academic_years || []).map((year) => `Year ${year}`).join(", ") || "—"}</strong>
            </div>
            <div>
              <span>Created by</span>
              <strong>{task.created_by?.name || "—"}</strong>
            </div>
          </div>
          <section>
            <h3>Description</h3>
            <p>{task.description || "No description provided."}</p>
          </section>
          <section>
            <h3>Instructions</h3>
            <p>{task.instructions || "No additional instructions."}</p>
          </section>
          {task.required_attachments && (
            <section>
              <h3>Required attachments</h3>
              <p>{task.required_attachments}</p>
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
        </div>
      ) : null}
    </Modal>
  );
}

function VerifySubmissionDialog({ task, members, onClose, onVerified }) {
  const [memberId, setMemberId] = useState("");
  const [action, setAction] = useState("verify");
  const [formError, setFormError] = useState(null);
  const [saving, setSaving] = useState(false);

  const handleSubmit = async (event) => {
    event.preventDefault();
    if (!memberId) {
      setFormError(new Error("Choose a member."));
      return;
    }
    setSaving(true);
    setFormError(null);
    try {
      const result = await tasksApi.verify(task.id, { member_id: memberId, action });
      onVerified(result);
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
      title="Review task submission"
      description={`Task: ${task.title}`}
      footer={
        <>
          <AppButton variant="ghost" onClick={onClose} disabled={saving}>
            Cancel
          </AppButton>
          <AppButton
            form="verify-submission-form"
            type="submit"
            variant={action === "reject" ? "danger" : "primary"}
            loading={saving}
          >
            {action === "reject" ? "Reject submission" : "Verify submission"}
          </AppButton>
        </>
      }
    >
      <form id="verify-submission-form" className="lab-form-stack" onSubmit={handleSubmit}>
        {formError && (
          <Alert tone="danger" title="Submission could not be updated">
            {formError.message}
          </Alert>
        )}
        <Alert tone="info" title="Submission lookup">
          The API verifies by task and member. Choose the member whose submission you are actioning.
        </Alert>
        <Field label="Member" htmlFor="verify-member" required>
          <Select
            id="verify-member"
            value={memberId}
            onChange={(event) => setMemberId(event.target.value)}
            disabled={saving}
          >
            <option value="">Select a member</option>
            {members.map((member) => (
              <option key={member.id} value={member.id}>
                {member.name} · {member.login_id || member.usn}
              </option>
            ))}
          </Select>
        </Field>
        <Field label="Action" htmlFor="verify-action" required>
          <Select
            id="verify-action"
            value={action}
            onChange={(event) => setAction(event.target.value)}
            disabled={saving}
          >
            <option value="verify">Verify submission</option>
            <option value="reject">Reject submission</option>
          </Select>
        </Field>
      </form>
    </Modal>
  );
}

export default function TasksPage() {
  const [view, setView] = useState("all");
  const [type, setType] = useState("");
  const [search, setSearch] = useState("");
  const [page, setPage] = useState(1);
  const [formOpen, setFormOpen] = useState(false);
  const [editing, setEditing] = useState(null);
  const [editLoadingId, setEditLoadingId] = useState(null);
  const [detail, setDetail] = useState(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detailError, setDetailError] = useState(null);
  const [verifyTask, setVerifyTask] = useState(null);
  const [deleteTarget, setDeleteTarget] = useState(null);
  const [deleting, setDeleting] = useState(false);
  const [notice, setNotice] = useState(null);
  const [actionError, setActionError] = useState(null);

  const query = useMemo(() => {
    const params = { page, per_page: 12 };
    if (view === "upcoming") params.upcoming = "true";
    if (view === "overdue") params.overdue = "true";
    if (type) params.task_type = type;
    return params;
  }, [page, type, view]);

  const resource = useApiResource(
    () => tasksApi.list(query),
    `tasks-${JSON.stringify(query)}`,
  );
  const membersResource = useApiResource(
    () => membersManagementApi.list({ per_page: 100 }),
    "task-verification-members",
  );

  const tasks = (resource.data?.tasks ?? []).filter((task) =>
    search ? task.title.toLowerCase().includes(search.toLowerCase()) : true,
  );
  const members = membersResource.data?.members ?? [];

  const openDetail = async (task) => {
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

  const openEdit = async (task) => {
    setEditLoadingId(task.id);
    setActionError(null);
    try {
      const result = await tasksApi.get(task.id);
      setEditing(result.task);
      setFormOpen(true);
    } catch (error) {
      setActionError(error);
    } finally {
      setEditLoadingId(null);
    }
  };

  const handleDelete = async () => {
    if (!deleteTarget || deleting) return;
    setDeleting(true);
    setActionError(null);
    try {
      await tasksApi.remove(deleteTarget.id);
      setNotice("Task deleted.");
      setDeleteTarget(null);
      resource.reload();
    } catch (error) {
      setActionError(error);
    } finally {
      setDeleting(false);
    }
  };

  return (
    <>
      <PageHeader
        eyebrow="Admin console"
        title="Tasks"
        description="Publish assignments, projects and activities, then verify member submissions."
        actions={
          <AppButton
            variant="primary"
            onClick={() => {
              setEditing(null);
              setFormOpen(true);
            }}
          >
            <PlusIcon className="h-4 w-4" />
            New task
          </AppButton>
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

      <Panel padded={false}>
        <div className="lab-toolbar lab-toolbar--wrap">
          <Tabs
            tabs={VIEW_TABS}
            value={view}
            onChange={(value) => {
              setView(value);
              setPage(1);
            }}
            label="Task filters"
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
            <div className="lab-search">
              <SearchIcon className="h-4 w-4" />
              <input
                type="search"
                value={search}
                onChange={(event) => {
                  setSearch(event.target.value);
                  setPage(1);
                }}
                placeholder="Search tasks"
                aria-label="Search tasks"
              />
            </div>
          </div>
        </div>

        <div className="lab-list-panel__body">
          {resource.loading ? (
            <LoadingState rows={6} />
          ) : resource.error ? (
            <ErrorState error={resource.error} onRetry={resource.reload} title="Tasks are unavailable" />
          ) : tasks.length === 0 ? (
            <EmptyState
              icon={CheckSquareIcon}
              title={search ? "No matching tasks" : "No tasks in this view"}
              description={
                search
                  ? "Try a different task search."
                  : "Create the first task for the selected academic years."
              }
              action={
                !search ? (
                  <AppButton variant="primary" onClick={() => setFormOpen(true)}>
                    <PlusIcon className="h-4 w-4" />
                    New task
                  </AppButton>
                ) : null
              }
            />
          ) : (
            <div className="lab-task-list">
              {tasks.map((task) => (
                <article key={task.id} className="lab-task-card">
                  <div className="lab-task-card__main">
                    <div className="lab-task-card__top">
                      <Badge tone="info">{titleCase(task.task_type)}</Badge>
                      <Badge tone={deadlineTone(task)} dot>{deadlineLabel(task)}</Badge>
                    </div>
                    <button type="button" className="lab-task-card__title" onClick={() => openDetail(task)}>
                      {task.title}
                    </button>
                    <div className="lab-task-card__meta">
                      <span><CalendarIcon className="h-3.5 w-3.5" /> Deadline {formatDate(task.deadline)}</span>
                      <span><UsersIcon className="h-3.5 w-3.5" />
                        {(task.target_academic_years || []).map((year) => `Year ${year}`).join(", ") || "No target years"}
                      </span>
                    </div>
                    {task.required_attachments && (
                      <p className="lab-task-card__note">Required: {task.required_attachments}</p>
                    )}
                  </div>
                  <div className="lab-task-card__actions">
                    <AppButton variant="ghost" size="sm" onClick={() => setVerifyTask(task)}>
                      <CheckSquareIcon className="h-3.5 w-3.5" />
                      Review
                    </AppButton>
                    <AppButton
                      variant="ghost"
                      size="sm"
                      onClick={() => openEdit(task)}
                      loading={editLoadingId === task.id}
                    >
                      <EditIcon className="h-3.5 w-3.5" />
                      Edit
                    </AppButton>
                    <AppButton
                      variant="ghost"
                      size="sm"
                      className="lab-danger-text"
                      onClick={() => setDeleteTarget(task)}
                    >
                      <TrashIcon className="h-3.5 w-3.5" />
                      Delete
                    </AppButton>
                  </div>
                </article>
              ))}
            </div>
          )}
        </div>

        {resource.data?.pagination && (
          <Pagination
            page={resource.data.pagination.page}
            totalPages={resource.data.pagination.total_pages}
            totalCount={resource.data.pagination.total_count}
            onPageChange={setPage}
            loading={resource.loading}
          />
        )}
      </Panel>

      {formOpen && (
        <TaskForm
          key={editing?.id || "new"}
          task={editing}
          onClose={() => {
            setFormOpen(false);
            setEditing(null);
          }}
          onSaved={(result) => {
            setNotice(result.message || "Task saved.");
            resource.reload();
          }}
        />
      )}

      {detail && (
        <TaskDetail
          task={detail}
          loading={detailLoading}
          error={detailError}
          onClose={() => setDetail(null)}
          onVerify={() => {
            setVerifyTask(detail);
            setDetail(null);
          }}
        />
      )}

      {verifyTask && (
        <VerifySubmissionDialog
          task={verifyTask}
          members={members}
          onClose={() => setVerifyTask(null)}
          onVerified={(result) => {
            setNotice(result.message || "Submission updated.");
            resource.reload();
          }}
        />
      )}

      <ConfirmDialog
        open={Boolean(deleteTarget)}
        onClose={() => setDeleteTarget(null)}
        onConfirm={handleDelete}
        title="Delete this task?"
        description={`“${deleteTarget?.title || ""}” will be removed from the task list.`}
        confirmLabel="Delete task"
        destructive
        loading={deleting}
      />
    </>
  );
}
