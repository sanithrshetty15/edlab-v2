import { useMemo, useState } from "react";
import { useAuth } from "../../auth/context";
import { eventsApi } from "../../api/services";
import { useApiResource } from "../../hooks/useApiResource";
import { ConfirmDialog, Modal } from "../../components/app/Modal";
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
  Tabs,
  TextArea,
  TextInput,
} from "../../components/app/Ui";
import {
  CalendarIcon,
  EditIcon,
  ExternalIcon,
  PlusIcon,
  SearchIcon,
  TrashIcon,
} from "../../components/app/AppIcons";
import {
  formatClockTime,
  formatDate,
  formatDateTime,
  formatMonthDay,
  formatTime,
  safeHttpUrl,
  titleCase,
} from "../../utils/format";

const EVENT_TYPES = ["workshop", "seminar", "talk", "hackathon", "training", "other"];

const VIEW_TABS = [
  { value: "upcoming", label: "Upcoming" },
  { value: "past", label: "Past" },
  { value: "all", label: "All events" },
];

function EventForm({ event, onClose, onSaved }) {
  const [title, setTitle] = useState(event?.title || "");
  const [description, setDescription] = useState(event?.description || "");
  const [eventType, setEventType] = useState(event?.event_type || "workshop");
  const [date, setDate] = useState(event?.date || "");
  const [startTime, setStartTime] = useState(event?.start_time?.slice(0, 5) || "10:00");
  const [endTime, setEndTime] = useState(event?.end_time?.slice(0, 5) || "17:00");
  const [venue, setVenue] = useState(event?.venue || "");
  const [externalLinks, setExternalLinks] = useState((event?.external_links || []).join("\n"));
  const [errors, setErrors] = useState({});
  const [formError, setFormError] = useState(null);
  const [saving, setSaving] = useState(false);
  const editing = Boolean(event?.id);

  const handleSubmit = async (submitEvent) => {
    submitEvent.preventDefault();
    if (saving) return;

    const nextErrors = {};
    if (!title.trim()) nextErrors.title = "Title is required.";
    if (!date) nextErrors.date = "Choose an event date.";
    if (!startTime) nextErrors.startTime = "Start time is required.";
    if (!endTime) nextErrors.endTime = "End time is required.";
    if (date && startTime && endTime) {
      const start = new Date(`${date}T${startTime}:00`);
      const end = new Date(`${date}T${endTime}:00`);
      if (end <= start) nextErrors.endTime = "End time must be after the start time.";
    }

    const links = externalLinks
      .split(/[\n,]+/)
      .map((link) => link.trim())
      .filter(Boolean);
    const invalidLink = links.find((link) => !safeHttpUrl(link));
    if (invalidLink) nextErrors.externalLinks = `“${invalidLink}” is not a valid http or https URL.`;

    setErrors(nextErrors);
    setFormError(null);
    if (Object.keys(nextErrors).length) return;

    setSaving(true);
    try {
      const payload = {
        title: title.trim(),
        description: description.trim(),
        event_type: eventType,
        date,
        start_time: startTime,
        end_time: endTime,
        venue: venue.trim(),
        external_links: links,
      };
      const result = editing
        ? await eventsApi.update(event.id, payload)
        : await eventsApi.create(payload);
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
      title={editing ? "Edit event" : "New event"}
      description="Events are visible to signed-in members on the lab calendar."
      size="lg"
      footer={
        <>
          <AppButton variant="ghost" onClick={onClose} disabled={saving}>
            Cancel
          </AppButton>
          <AppButton form="event-form" type="submit" variant="primary" loading={saving}>
            {editing ? "Save changes" : "Create event"}
          </AppButton>
        </>
      }
    >
      <form id="event-form" className="lab-form-stack" onSubmit={handleSubmit} noValidate>
        {formError && (
          <Alert tone="danger" title="Event could not be saved">
            {formError.message}
          </Alert>
        )}

        <Field label="Title" htmlFor="event-title" error={errors.title} required>
          <TextInput
            id="event-title"
            value={title}
            onChange={(inputEvent) => setTitle(inputEvent.target.value)}
            placeholder="Event name"
            disabled={saving}
          />
        </Field>

        <Field label="Event type" htmlFor="event-type" required>
          <Select
            id="event-type"
            value={eventType}
            onChange={(inputEvent) => setEventType(inputEvent.target.value)}
            disabled={saving}
          >
            {EVENT_TYPES.map((type) => (
              <option key={type} value={type}>{titleCase(type)}</option>
            ))}
          </Select>
        </Field>

        <div className="lab-form-grid lab-form-grid--3">
          <Field label="Date" htmlFor="event-date" error={errors.date} required>
            <TextInput
              id="event-date"
              type="date"
              value={date}
              onChange={(inputEvent) => setDate(inputEvent.target.value)}
              disabled={saving}
            />
          </Field>
          <Field label="Start time" htmlFor="event-start" error={errors.startTime} required>
            <TextInput
              id="event-start"
              type="time"
              value={startTime}
              onChange={(inputEvent) => setStartTime(inputEvent.target.value)}
              disabled={saving}
            />
          </Field>
          <Field label="End time" htmlFor="event-end" error={errors.endTime} required>
            <TextInput
              id="event-end"
              type="time"
              value={endTime}
              onChange={(inputEvent) => setEndTime(inputEvent.target.value)}
              disabled={saving}
            />
          </Field>
        </div>

        <Field label="Venue" htmlFor="event-venue" hint="Leave blank if the venue is still to be announced.">
          <TextInput
            id="event-venue"
            value={venue}
            onChange={(inputEvent) => setVenue(inputEvent.target.value)}
            placeholder="Lab, seminar hall or online"
            disabled={saving}
          />
        </Field>

        <Field label="Description" htmlFor="event-description">
          <TextArea
            id="event-description"
            rows={5}
            value={description}
            onChange={(inputEvent) => setDescription(inputEvent.target.value)}
            placeholder="What will happen at this event?"
            disabled={saving}
          />
        </Field>

        <Field
          label="External links"
          htmlFor="event-links"
          hint="One URL per line, for example a registration or resource link."
          error={errors.externalLinks}
        >
          <TextArea
            id="event-links"
            rows={3}
            value={externalLinks}
            onChange={(inputEvent) => setExternalLinks(inputEvent.target.value)}
            placeholder="https://example.com/register"
            disabled={saving}
          />
        </Field>
      </form>
    </Modal>
  );
}

function EventDetail({ event, loading, error, onClose }) {
  return (
    <Modal
      open
      onClose={onClose}
      title={event?.title || "Event"}
      size="lg"
      footer={<AppButton variant="outline" onClick={onClose}>Close</AppButton>}
    >
      {loading ? (
        <LoadingState rows={5} />
      ) : error ? (
        <ErrorState error={error} title="Event could not be loaded" />
      ) : (
        <article className="lab-event-detail">
          <div className="lab-event-detail__meta">
            <Badge tone="info">{titleCase(event?.event_type)}</Badge>
            <span>{formatDate(event?.date)}</span>
            <span>{formatTime(event?.start_time)} – {formatTime(event?.end_time)}</span>
            <span>{event?.venue || "Venue to be announced"}</span>
          </div>
          <div className="lab-event-detail__description">
            {event?.description || "No description has been added for this event."}
          </div>
          {event?.external_links?.length > 0 && (
            <div className="lab-event-detail__links">
              <h3>Links</h3>
              {event.external_links.map((link) => {
                const url = safeHttpUrl(link);
                if (!url) return null;
                return (
                  <a key={link} href={url} target="_blank" rel="noreferrer noopener">
                    {link}
                    <ExternalIcon className="h-3.5 w-3.5" />
                  </a>
                );
              })}
            </div>
          )}
          <p className="lab-muted lab-small">
            Created {event?.created_at ? formatDateTime(event.created_at) : "—"}
            {event?.created_by?.name ? ` by ${event.created_by.name}` : ""}
          </p>
        </article>
      )}
    </Modal>
  );
}

export default function EventsPage() {
  const auth = useAuth();
  const [view, setView] = useState("upcoming");
  const [type, setType] = useState("");
  const [search, setSearch] = useState("");
  const [page, setPage] = useState(1);
  const [formOpen, setFormOpen] = useState(false);
  const [editing, setEditing] = useState(null);
  const [editLoading, setEditLoading] = useState(false);
  const [detail, setDetail] = useState(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detailError, setDetailError] = useState(null);
  const [deleteTarget, setDeleteTarget] = useState(null);
  const [deleting, setDeleting] = useState(false);
  const [notice, setNotice] = useState(null);
  const [actionError, setActionError] = useState(null);

  const query = useMemo(() => {
    const params = { page, per_page: 12 };
    if (view === "upcoming") params.upcoming = "true";
    if (view === "past") params.past = "true";
    if (type) params.event_type = type;
    return params;
  }, [page, type, view]);

  const resource = useApiResource(
    () => eventsApi.list(query),
    `events-${JSON.stringify(query)}`,
  );

  const events = (resource.data?.events ?? []).filter((event) =>
    search ? event.title.toLowerCase().includes(search.toLowerCase()) : true,
  );

  const openDetail = async (event) => {
    setDetail(event);
    setDetailLoading(true);
    setDetailError(null);
    try {
      const result = await eventsApi.get(event.id);
      setDetail(result.event);
    } catch (error) {
      setDetailError(error);
    } finally {
      setDetailLoading(false);
    }
  };

  const openEdit = async (event) => {
    setEditLoading(true);
    setActionError(null);
    try {
      const result = await eventsApi.get(event.id);
      setEditing(result.event);
      setFormOpen(true);
    } catch (error) {
      setActionError(error);
    } finally {
      setEditLoading(false);
    }
  };

  const handleDelete = async () => {
    if (!deleteTarget || deleting) return;
    setDeleting(true);
    setActionError(null);
    try {
      await eventsApi.remove(deleteTarget.id);
      setNotice("Event deleted.");
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
        eyebrow={auth.isAdmin ? "Admin console" : "Member workspace"}
        title="Events"
        description={
          auth.isAdmin
            ? "Schedule workshops, talks, hackathons and lab sessions."
            : "Workshops, talks, hackathons and sessions coming up at Edwins Lab."
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
              New event
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
        <div className="lab-toolbar lab-toolbar--wrap">
          <Tabs
            tabs={VIEW_TABS}
            value={view}
            onChange={(value) => {
              setView(value);
              setPage(1);
            }}
            label="Event period"
          />
          <div className="lab-toolbar__right">
            <Select
              value={type}
              onChange={(event) => {
                setType(event.target.value);
                setPage(1);
              }}
              aria-label="Filter by event type"
              className="lab-select--compact"
            >
              <option value="">All types</option>
              {EVENT_TYPES.map((eventType) => (
                <option key={eventType} value={eventType}>{titleCase(eventType)}</option>
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
                placeholder="Search events"
                aria-label="Search events"
              />
            </div>
          </div>
        </div>

        <div className="lab-list-panel__body">
          {resource.loading ? (
            <LoadingState rows={6} />
          ) : resource.error ? (
            <ErrorState error={resource.error} onRetry={resource.reload} title="Events are unavailable" />
          ) : events.length === 0 ? (
            <EmptyState
              icon={CalendarIcon}
              title={search ? "No matching events" : "No events in this view"}
              description={
                search
                  ? "Try a different event search."
                  : auth.isAdmin
                    ? "Create the first event for the lab calendar."
                    : "New events will appear here once the lab team schedules them."
              }
              action={auth.isAdmin && !search ? (
                <AppButton variant="primary" onClick={() => setFormOpen(true)}>
                  <PlusIcon className="h-4 w-4" />
                  New event
                </AppButton>
              ) : null}
            />
          ) : (
            <div className="lab-event-grid">
              {events.map((event) => {
                const parts = formatMonthDay(event.date);
                return (
                  <article key={event.id} className="lab-event-card">
                    <div className="lab-event-card__date">
                      <span>{parts.weekday}</span>
                      <strong>{parts.day}</strong>
                      <span>{parts.month}</span>
                    </div>
                    <div className="lab-event-card__body">
                      <div className="lab-event-card__top">
                        <Badge tone="info">{titleCase(event.event_type)}</Badge>
                        {event.date >= new Date().toISOString().slice(0, 10) && (
                          <Badge tone="accent" dot>Upcoming</Badge>
                        )}
                      </div>
                      <button type="button" className="lab-event-card__title" onClick={() => openDetail(event)}>
                        {event.title}
                      </button>
                      <p className="lab-event-card__meta">
                        <CalendarIcon className="h-3.5 w-3.5" />
                        {formatDate(event.date)}
                        <span>·</span>
                        {formatClockTime(event.start_time)} – {formatClockTime(event.end_time)}
                      </p>
                      <p className="lab-event-card__venue">{event.venue || "Venue to be announced"}</p>

                      {auth.isAdmin && (
                        <div className="lab-event-card__actions">
                          <AppButton
                            variant="ghost"
                            size="sm"
                            onClick={() => openEdit(event)}
                            loading={editLoading && editing?.id === event.id}
                          >
                            <EditIcon className="h-3.5 w-3.5" />
                            Edit
                          </AppButton>
                          <AppButton
                            variant="ghost"
                            size="sm"
                            className="lab-danger-text"
                            onClick={() => setDeleteTarget(event)}
                          >
                            <TrashIcon className="h-3.5 w-3.5" />
                            Delete
                          </AppButton>
                        </div>
                      )}
                    </div>
                  </article>
                );
              })}
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
        <EventForm
          key={editing?.id || "new"}
          event={editing}
          onClose={() => {
            setFormOpen(false);
            setEditing(null);
          }}
          onSaved={(result) => {
            setNotice(result.message || "Event saved.");
            resource.reload();
          }}
        />
      )}

      {detail && (
        <EventDetail
          event={detail}
          loading={detailLoading}
          error={detailError}
          onClose={() => setDetail(null)}
        />
      )}

      <ConfirmDialog
        open={Boolean(deleteTarget)}
        onClose={() => setDeleteTarget(null)}
        onConfirm={handleDelete}
        title="Delete this event?"
        description={`“${deleteTarget?.title || ""}” will be removed from the lab calendar.`}
        confirmLabel="Delete event"
        destructive
        loading={deleting}
      />
    </>
  );
}
