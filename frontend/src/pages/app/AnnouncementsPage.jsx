import { useMemo, useState } from "react";
import { useAuth } from "../../auth/context";
import { announcementsApi } from "../../api/services";
import { useApiResource } from "../../hooks/useApiResource";
import { ConfirmDialog, Modal } from "../../components/app/Modal";
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
  Tabs,
  TextArea,
  TextInput,
} from "../../components/app/Ui";
import {
  EditIcon,
  MegaphoneIcon,
  PinIcon,
  PlusIcon,
  SearchIcon,
  TrashIcon,
} from "../../components/app/AppIcons";
import {
  formatDateTime,
  formatRelative,
  toDateTimeLocalValue,
} from "../../utils/format";

const TABS = [
  { value: "all", label: "All" },
  { value: "published", label: "Published" },
  { value: "drafts", label: "Drafts & scheduled" },
  { value: "pinned", label: "Pinned" },
];

function AnnouncementForm({ announcement, onClose, onSaved }) {
  const [title, setTitle] = useState(announcement?.title || "");
  const [content, setContent] = useState(announcement?.content || "");
  const [important, setImportant] = useState(Boolean(announcement?.is_pinned));
  const [scheduledAt, setScheduledAt] = useState(
    announcement?.scheduled_at ? toDateTimeLocalValue(announcement.scheduled_at) : "",
  );
  const [errors, setErrors] = useState({});
  const [formError, setFormError] = useState(null);
  const [saving, setSaving] = useState(false);
  const editing = Boolean(announcement?.id);

  const handleSubmit = async (event) => {
    event.preventDefault();
    if (saving) return;

    const nextErrors = {};
    if (!title.trim()) nextErrors.title = "Title is required.";
    if (!content.trim()) nextErrors.content = "Content is required.";
    if (scheduledAt && Number.isNaN(new Date(scheduledAt).getTime())) {
      nextErrors.scheduledAt = "Choose a valid date and time.";
    }
    setErrors(nextErrors);
    setFormError(null);
    if (Object.keys(nextErrors).length) return;

    setSaving(true);
    try {
      let result;
      if (editing) {
        result = await announcementsApi.update(announcement.id, {
          title: title.trim(),
          content: content.trim(),
          is_important: important,
        });
      } else {
        result = await announcementsApi.create({
          title: title.trim(),
          content: content.trim(),
          is_important: important,
          scheduled_at: scheduledAt ? new Date(scheduledAt).toISOString() : undefined,
        });
      }
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
      title={editing ? "Edit announcement" : "New announcement"}
      description={
        editing
          ? "Update the published text or pin state. Scheduling is set when an announcement is created."
          : "Publish immediately or schedule a future publication."
      }
      size="lg"
      footer={
        <>
          <AppButton variant="ghost" onClick={onClose} disabled={saving}>
            Cancel
          </AppButton>
          <AppButton form="announcement-form" type="submit" variant="primary" loading={saving}>
            {editing ? "Save changes" : scheduledAt ? "Schedule announcement" : "Publish announcement"}
          </AppButton>
        </>
      }
    >
      <form id="announcement-form" className="lab-form-stack" onSubmit={handleSubmit} noValidate>
        {formError && (
          <Alert tone="danger" title="Announcement could not be saved">
            {formError.message}
          </Alert>
        )}
        <Field label="Title" htmlFor="announcement-title" error={errors.title} required>
          <TextInput
            id="announcement-title"
            value={title}
            maxLength={160}
            onChange={(event) => setTitle(event.target.value)}
            placeholder="What does the lab need to know?"
            disabled={saving}
          />
        </Field>
        <Field
          label="Content"
          htmlFor="announcement-content"
          error={errors.content}
          hint="Plain text. Line breaks are preserved."
          required
        >
          <TextArea
            id="announcement-content"
            rows={9}
            value={content}
            onChange={(event) => setContent(event.target.value)}
            placeholder="Write the full announcement."
            disabled={saving}
          />
        </Field>
        <Checkbox
          label="Pin this announcement to the top"
          id="announcement-important"
          checked={important}
          onChange={(event) => setImportant(event.target.checked)}
          disabled={saving}
        />
        {!editing && (
          <Field
            label="Schedule for later"
            htmlFor="announcement-schedule"
            hint="Leave blank to publish immediately."
            error={errors.scheduledAt}
          >
            <TextInput
              id="announcement-schedule"
              type="datetime-local"
              value={scheduledAt}
              onChange={(event) => setScheduledAt(event.target.value)}
              disabled={saving}
            />
          </Field>
        )}
      </form>
    </Modal>
  );
}

function AnnouncementDetail({ announcement, loading, error, onClose }) {
  return (
    <Modal
      open
      onClose={onClose}
      title={announcement?.title || "Announcement"}
      size="lg"
      footer={<AppButton variant="outline" onClick={onClose}>Close</AppButton>}
    >
      {loading ? (
        <LoadingState rows={5} />
      ) : error ? (
        <ErrorState error={error} title="Announcement could not be loaded" />
      ) : (
        <article className="lab-announcement-detail">
          <div className="lab-announcement-detail__meta">
            {announcement?.is_pinned && <Badge tone="accent" dot>Pinned</Badge>}
            <span>
              {announcement?.published_at
                ? `Published ${formatRelative(announcement.published_at)}`
                : announcement?.scheduled_at
                  ? `Scheduled ${formatDateTime(announcement.scheduled_at)}`
                  : "Draft"}
            </span>
            {announcement?.created_by?.name && <span>By {announcement.created_by.name}</span>}
          </div>
          <div className="lab-announcement-detail__content">{announcement?.content}</div>
        </article>
      )}
    </Modal>
  );
}

export default function AnnouncementsPage() {
  const auth = useAuth();
  const [tab, setTab] = useState("all");
  const [search, setSearch] = useState("");
  const [page, setPage] = useState(1);
  const [formOpen, setFormOpen] = useState(false);
  const [editing, setEditing] = useState(null);
  const [detail, setDetail] = useState(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detailError, setDetailError] = useState(null);
  const [deleteTarget, setDeleteTarget] = useState(null);
  const [actionBusy, setActionBusy] = useState(null);
  const [notice, setNotice] = useState(null);
  const [actionError, setActionError] = useState(null);

  const query = useMemo(() => {
    const params = { page, per_page: 12 };
    if (!auth.isAdmin) params.published = "true";
    if (auth.isAdmin && tab === "published") params.published = "true";
    if (auth.isAdmin && tab === "pinned") params.important = "true";
    return params;
  }, [auth.isAdmin, page, tab]);

  const resource = useApiResource(
    () => announcementsApi.list(query),
    `announcements-${JSON.stringify(query)}`,
  );

  const announcements = resource.data?.announcements ?? [];
  const filtered = announcements.filter((announcement) => {
    if (search && !announcement.title.toLowerCase().includes(search.toLowerCase())) return false;
    if (auth.isAdmin && tab === "drafts" && announcement.is_published) return false;
    return true;
  });

  const openDetail = async (announcement) => {
    setDetail(announcement);
    setDetailLoading(true);
    setDetailError(null);
    try {
      const result = await announcementsApi.get(announcement.id);
      setDetail(result.announcement);
    } catch (error) {
      setDetailError(error);
    } finally {
      setDetailLoading(false);
    }
  };

  const openEdit = async (announcement) => {
    setActionBusy(`${announcement.id}:edit`);
    setActionError(null);
    try {
      const result = await announcementsApi.get(announcement.id);
      setEditing(result.announcement);
      setFormOpen(true);
    } catch (error) {
      setActionError(error);
    } finally {
      setActionBusy(null);
    }
  };

  const runAction = async (id, action, successMessage) => {
    setActionBusy(`${id}:${action}`);
    setActionError(null);
    setNotice(null);
    try {
      if (action === "publish") await announcementsApi.publish(id);
      else if (action === "unpublish") await announcementsApi.unpublish(id);
      else if (action === "pin") await announcementsApi.togglePin(id);
      else if (action === "delete") await announcementsApi.remove(id);
      setNotice(successMessage);
      setDeleteTarget(null);
      resource.reload();
    } catch (error) {
      setActionError(error);
    } finally {
      setActionBusy(null);
    }
  };

  return (
    <>
      <PageHeader
        eyebrow={auth.isAdmin ? "Admin console" : "Member workspace"}
        title="Announcements"
        description={
          auth.isAdmin
            ? "Publish, schedule and maintain notices for the lab."
            : "Official notices, updates and reminders from the Edwins Lab team."
        }
        actions={
          auth.isAdmin ? (
            <AppButton
              variant="primary"
              onClick={() => {
                setEditing(null);
                setFormOpen(true);
              }}
            >
              <PlusIcon className="h-4 w-4" />
              New announcement
            </AppButton>
          ) : null
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

      <Panel padded={false} className="lab-list-panel">
        <div className="lab-toolbar">
          {auth.isAdmin ? (
            <Tabs
              tabs={TABS}
              value={tab}
              onChange={(value) => {
                setTab(value);
                setPage(1);
              }}
              label="Announcement filters"
            />
          ) : (
            <p className="lab-toolbar__label">Published announcements</p>
          )}
          <div className="lab-search">
            <SearchIcon className="h-4 w-4" />
            <input
              type="search"
              value={search}
              onChange={(event) => {
                setSearch(event.target.value);
                setPage(1);
              }}
              placeholder="Search titles"
              aria-label="Search announcements"
            />
          </div>
        </div>

        <div className="lab-list-panel__body">
          {resource.loading ? (
            <LoadingState rows={6} />
          ) : resource.error ? (
            <ErrorState error={resource.error} onRetry={resource.reload} title="Announcements are unavailable" />
          ) : filtered.length === 0 ? (
            <EmptyState
              icon={MegaphoneIcon}
              title={search ? "No matching announcements" : "No announcements yet"}
              description={
                search
                  ? "Try a different title search."
                  : auth.isAdmin
                    ? "Publish the first notice for the lab."
                    : "New notices from the lab team will appear here."
              }
              action={auth.isAdmin && !search ? (
                <AppButton variant="primary" onClick={() => setFormOpen(true)}>
                  <PlusIcon className="h-4 w-4" />
                  New announcement
                </AppButton>
              ) : null}
            />
          ) : (
            <div className="lab-announcement-list">
              {filtered.map((announcement) => (
                <article key={announcement.id} className="lab-announcement-card">
                  <div className="lab-announcement-card__top">
                    <div className="lab-announcement-card__badges">
                      {announcement.is_pinned && <Badge tone="accent"><PinIcon className="h-3 w-3" /> Pinned</Badge>}
                      <Badge tone={announcement.is_published ? "success" : "warning"} dot>
                        {announcement.is_published ? "Published" : "Scheduled"}
                      </Badge>
                    </div>
                    <time>
                      {announcement.published_at
                        ? formatRelative(announcement.published_at)
                        : announcement.scheduled_at
                          ? `Goes live ${formatDateTime(announcement.scheduled_at)}`
                          : "Not scheduled"}
                    </time>
                  </div>

                  <button type="button" className="lab-announcement-card__title" onClick={() => openDetail(announcement)}>
                    {announcement.title}
                  </button>
                  <p className="lab-announcement-card__meta">
                    {announcement.created_by?.name ? `By ${announcement.created_by.name} · ` : ""}
                    {announcement.is_published
                      ? `Published ${formatDateTime(announcement.published_at)}`
                      : `Created ${formatDateTime(announcement.created_at)}`}
                  </p>

                  {auth.isAdmin && (
                    <div className="lab-announcement-card__actions">
                      <AppButton variant="ghost" size="sm" onClick={() => openEdit(announcement)} loading={actionBusy === `${announcement.id}:edit`}>
                        <EditIcon className="h-3.5 w-3.5" />
                        Edit
                      </AppButton>
                      {announcement.is_published ? (
                        <AppButton
                          variant="ghost"
                          size="sm"
                          loading={actionBusy === `${announcement.id}:unpublish`}
                          onClick={() => runAction(announcement.id, "unpublish", "Announcement moved back to draft.")}
                        >
                          Unpublish
                        </AppButton>
                      ) : (
                        <AppButton
                          variant="ghost"
                          size="sm"
                          loading={actionBusy === `${announcement.id}:publish`}
                          onClick={() => runAction(announcement.id, "publish", "Announcement published.")}
                        >
                          Publish now
                        </AppButton>
                      )}
                      <AppButton
                        variant="ghost"
                        size="sm"
                        loading={actionBusy === `${announcement.id}:pin`}
                        onClick={() => runAction(announcement.id, "pin", announcement.is_pinned ? "Announcement unpinned." : "Announcement pinned.")}
                      >
                        <PinIcon className="h-3.5 w-3.5" />
                        {announcement.is_pinned ? "Unpin" : "Pin"}
                      </AppButton>
                      <AppButton
                        variant="ghost"
                        size="sm"
                        className="lab-danger-text"
                        onClick={() => setDeleteTarget(announcement)}
                      >
                        <TrashIcon className="h-3.5 w-3.5" />
                        Delete
                      </AppButton>
                    </div>
                  )}
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
        <AnnouncementForm
          key={editing?.id || "new"}
          announcement={editing}
          onClose={() => {
            setFormOpen(false);
            setEditing(null);
          }}
          onSaved={(result) => {
            setNotice(result.message || "Announcement saved.");
            resource.reload();
          }}
        />
      )}

      {detail && (
        <AnnouncementDetail
          announcement={detail}
          loading={detailLoading}
          error={detailError}
          onClose={() => setDetail(null)}
        />
      )}

      <ConfirmDialog
        open={Boolean(deleteTarget)}
        onClose={() => setDeleteTarget(null)}
        onConfirm={() => runAction(deleteTarget.id, "delete", "Announcement deleted.")}
        title="Delete this announcement?"
        description={`“${deleteTarget?.title || ""}” will be removed from the public API. This can only be undone directly in the database.`}
        confirmLabel="Delete announcement"
        destructive
        loading={actionBusy === `${deleteTarget?.id}:delete`}
      />
    </>
  );
}
