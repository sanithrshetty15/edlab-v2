import { useEffect, useMemo, useState } from "react";
import { useAuth } from "../../auth/context";
import { messagesApi } from "../../api/services";
import { useApiResource } from "../../hooks/useApiResource";
import { ConfirmDialog, Modal } from "../../components/app/Modal";
import {
  Alert,
  AppButton,
  Avatar,
  Badge,
  EmptyState,
  ErrorState,
  Field,
  LoadingState,
  PageHeader,
  Pagination,
  Panel,
  TextArea,
} from "../../components/app/Ui";
import {
  InfoIcon,
  MessageIcon,
  PinIcon,
  RefreshIcon,
  ReplyIcon,
  SearchIcon,
  SendIcon,
  TrashIcon,
  EditIcon,
} from "../../components/app/AppIcons";
import { formatDateTime, formatRelative } from "../../utils/format";

function MessageCard({ message, currentUserId, isAdmin, onReply, onEdit, onDelete, onPin, actionBusy, allMessages }) {
  const own = message.user?.id === currentUserId || message.user_id === currentUserId;
  const parent = message.reply_to_id
    ? allMessages.find((item) => item.id === message.reply_to_id)
    : null;

  return (
    <article className={`lab-message ${message.is_pinned ? "is-pinned" : ""}`}>
      <Avatar name={message.user?.name || "Administrator"} size="sm" />
      <div className="lab-message__body">
        <div className="lab-message__head">
          <strong>{own ? "You" : message.user?.name || "Administrator"}</strong>
          <time title={formatDateTime(message.created_at)}>{formatRelative(message.created_at)}</time>
          {message.is_edited && <span className="lab-message__edited">edited</span>}
          {message.is_pinned && <Badge tone="accent"><PinIcon className="h-3 w-3" /> Pinned</Badge>}
        </div>

        {parent && (
          <div className="lab-message__reply-context">
            <ReplyIcon className="h-3.5 w-3.5" />
            <span>{parent.content}</span>
          </div>
        )}

        <p className="lab-message__content">{message.content}</p>

        <div className="lab-message__actions">
          <AppButton variant="ghost" size="sm" onClick={() => onReply(message)}>
            <ReplyIcon className="h-3.5 w-3.5" />
            Reply
          </AppButton>
          {own && (
            <AppButton variant="ghost" size="sm" onClick={() => onEdit(message)}>
              <EditIcon className="h-3.5 w-3.5" />
              Edit
            </AppButton>
          )}
          {isAdmin && (
            <AppButton
              variant="ghost"
              size="sm"
              loading={actionBusy === `${message.id}:pin`}
              onClick={() => onPin(message)}
            >
              <PinIcon className="h-3.5 w-3.5" />
              {message.is_pinned ? "Unpin" : "Pin"}
            </AppButton>
          )}
          {(own || isAdmin) && (
            <AppButton variant="ghost" size="sm" className="lab-danger-text" onClick={() => onDelete(message)}>
              <TrashIcon className="h-3.5 w-3.5" />
              Delete
            </AppButton>
          )}
        </div>
      </div>
    </article>
  );
}

export default function MessagesPage() {
  const auth = useAuth();
  const [search, setSearch] = useState("");
  const [debouncedSearch, setDebouncedSearch] = useState("");
  const [page, setPage] = useState(1);
  const [content, setContent] = useState("");
  const [replyTo, setReplyTo] = useState(null);
  const [editing, setEditing] = useState(null);
  const [editContent, setEditContent] = useState("");
  const [deleteTarget, setDeleteTarget] = useState(null);
  const [sending, setSending] = useState(false);
  const [savingEdit, setSavingEdit] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [actionBusy, setActionBusy] = useState(null);
  const [formError, setFormError] = useState(null);
  const [notice, setNotice] = useState(null);
  const [actionError, setActionError] = useState(null);

  useEffect(() => {
    const timer = window.setTimeout(() => {
      setDebouncedSearch(search.trim());
      setPage(1);
    }, 350);
    return () => window.clearTimeout(timer);
  }, [search]);

  const isSearching = debouncedSearch.length > 0;
  const resourceKey = isSearching ? `messages-search-${debouncedSearch}` : `messages-page-${page}`;
  const resource = useApiResource(
    () =>
      isSearching
        ? messagesApi.search(debouncedSearch)
        : messagesApi.list({ page, per_page: 25 }),
    resourceKey,
  );

  const pinnedResource = useApiResource(
    () => messagesApi.pinned(),
    "messages-pinned",
    { enabled: auth.isAdmin },
  );

  const messages = useMemo(
    () => (isSearching ? resource.data?.results ?? [] : resource.data?.messages ?? []),
    [isSearching, resource.data],
  );
  const pinnedMessages = pinnedResource.data?.messages ?? [];

  const submitMessage = async (event) => {
    event.preventDefault();
    const trimmed = content.trim();
    if (!trimmed || sending) return;
    if (trimmed.length > 2000) {
      setFormError(new Error("Messages are limited to 2,000 characters."));
      return;
    }

    setSending(true);
    setFormError(null);
    setNotice(null);
    try {
      if (replyTo) {
        await messagesApi.reply(replyTo.id, { content: trimmed });
      } else {
        await messagesApi.create({ content: trimmed });
      }
      setContent("");
      setReplyTo(null);
      setNotice(replyTo ? "Reply posted." : "Message posted.");
      if (!isSearching) resource.reload();
      else setSearch("");
    } catch (error) {
      setFormError(error);
    } finally {
      setSending(false);
    }
  };

  const submitEdit = async (event) => {
    event.preventDefault();
    if (!editing || savingEdit) return;
    const trimmed = editContent.trim();
    if (!trimmed) return;

    setSavingEdit(true);
    setActionError(null);
    try {
      await messagesApi.edit(editing.id, { content: trimmed });
      setNotice("Message updated.");
      setEditing(null);
      resource.reload();
    } catch (error) {
      setActionError(error);
    } finally {
      setSavingEdit(false);
    }
  };

  const handleDelete = async () => {
    if (!deleteTarget || deleting) return;
    setDeleting(true);
    setActionError(null);
    try {
      await messagesApi.remove(deleteTarget.id);
      setNotice("Message deleted.");
      setDeleteTarget(null);
      resource.reload();
      if (auth.isAdmin) pinnedResource.reload();
    } catch (error) {
      setActionError(error);
    } finally {
      setDeleting(false);
    }
  };

  const handlePin = async (message) => {
    setActionBusy(`${message.id}:pin`);
    setActionError(null);
    try {
      if (message.is_pinned) await messagesApi.unpin(message.id);
      else await messagesApi.pin(message.id);
      setNotice(message.is_pinned ? "Message unpinned." : "Message pinned.");
      resource.reload();
      pinnedResource.reload();
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
        title="Messages"
        description={
          auth.isAdmin
            ? "The lab group conversation, with moderator controls."
            : "The shared Edwins Lab conversation."
        }
        actions={
          <AppButton variant="outline" onClick={() => resource.reload()} loading={resource.loading}>
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
      {(actionError || formError) && (
        <Alert tone="danger" title="Message action failed" onDismiss={() => {
          setActionError(null);
          setFormError(null);
        }}>
          {(actionError || formError).message}
        </Alert>
      )}

      <div className="lab-messages-layout">
        <Panel padded={false} className="lab-chat-panel">
          <div className="lab-chat-toolbar">
            <div className="lab-search">
              <SearchIcon className="h-4 w-4" />
              <input
                type="search"
                value={search}
                onChange={(event) => setSearch(event.target.value)}
                placeholder="Search the conversation"
                aria-label="Search messages"
              />
            </div>
            {isSearching && (
              <Badge tone="info">Search results</Badge>
            )}
          </div>

          <div className="lab-chat-feed">
            {resource.loading ? (
              <LoadingState rows={7} />
            ) : resource.error ? (
              <ErrorState
                error={resource.error}
                title="Messages are unavailable"
                onRetry={resource.reload}
              />
            ) : messages.length === 0 ? (
              <EmptyState
                icon={MessageIcon}
                title={isSearching ? "No matching messages" : "No messages yet"}
                description={
                  isSearching
                    ? "Try a different phrase."
                    : "Start the conversation by posting the first message."
                }
              />
            ) : (
              messages.map((message) => (
                <MessageCard
                  key={message.id}
                  message={message}
                  allMessages={messages}
                  currentUserId={auth.user?.id}
                  isAdmin={auth.isAdmin}
                  actionBusy={actionBusy}
                  onReply={(target) => {
                    setReplyTo(target);
                    document.getElementById("message-composer")?.focus();
                  }}
                  onEdit={(target) => {
                    setEditing(target);
                    setEditContent(target.content);
                  }}
                  onDelete={setDeleteTarget}
                  onPin={handlePin}
                />
              ))
            )}
          </div>

          {!isSearching && resource.data?.pagination && (
            <Pagination
              page={resource.data.pagination.page}
              totalPages={resource.data.pagination.total_pages}
              totalCount={resource.data.pagination.total_count}
              onPageChange={setPage}
              loading={resource.loading}
            />
          )}

          <form className="lab-composer" onSubmit={submitMessage}>
            {replyTo && (
              <div className="lab-composer__reply">
                <ReplyIcon className="h-3.5 w-3.5" />
                <span>Replying to {replyTo.user?.name || "Administrator"}: {replyTo.content}</span>
                <button type="button" onClick={() => setReplyTo(null)} aria-label="Cancel reply">
                  ×
                </button>
              </div>
            )}
            <label className="lab-sr-only" htmlFor="message-composer">
              Write a message
            </label>
            <TextArea
              id="message-composer"
              rows={3}
              value={content}
              maxLength={2000}
              onChange={(event) => setContent(event.target.value)}
              placeholder="Write to the lab…"
              disabled={sending}
            />
            <div className="lab-composer__footer">
              <span className="lab-muted lab-small">{content.length}/2000</span>
              <AppButton type="submit" variant="primary" loading={sending} disabled={!content.trim()}>
                <SendIcon className="h-4 w-4" />
                {replyTo ? "Post reply" : "Send message"}
              </AppButton>
            </div>
          </form>
        </Panel>

        <div className="lab-messages-aside">
          {auth.isAdmin && (
            <Panel title="Pinned messages" subtitle="Moderator shortcuts.">
              {pinnedResource.loading ? (
                <LoadingState rows={3} compact />
              ) : pinnedResource.error ? (
                <ErrorState error={pinnedResource.error} onRetry={pinnedResource.reload} compact />
              ) : pinnedMessages.length ? (
                <ul className="lab-pinned-messages">
                  {pinnedMessages.map((message) => (
                    <li key={message.id}>
                      <PinIcon className="h-3.5 w-3.5" />
                      <div>
                        <p>{message.content}</p>
                        <span>{message.user?.name || "Administrator"} · {formatRelative(message.created_at)}</span>
                      </div>
                    </li>
                  ))}
                </ul>
              ) : (
                <p className="lab-muted lab-small">No pinned messages.</p>
              )}
            </Panel>
          )}

          <Panel title="Conversation guide" subtitle="Keep the channel useful for everyone.">
            <ul className="lab-guideline-list">
              <li><InfoIcon className="h-4 w-4" /> Stay on topics relevant to the lab.</li>
              <li><MessageIcon className="h-4 w-4" /> Be clear and respectful in replies.</li>
              <li><PinIcon className="h-4 w-4" /> Administrators may pin important messages.</li>
            </ul>
            <p className="lab-muted lab-small">
              Messages are visible to signed-in lab members and administrators.
            </p>
          </Panel>
        </div>
      </div>

      <Modal
        open={Boolean(editing)}
        onClose={() => setEditing(null)}
        title="Edit message"
        description="Your edit is visible to everyone in the conversation."
        footer={
          <>
            <AppButton variant="ghost" onClick={() => setEditing(null)} disabled={savingEdit}>
              Cancel
            </AppButton>
            <AppButton form="edit-message-form" type="submit" variant="primary" loading={savingEdit}>
              Save changes
            </AppButton>
          </>
        }
      >
        <form id="edit-message-form" onSubmit={submitEdit}>
          <Field label="Message" htmlFor="edit-message-content">
            <TextArea
              id="edit-message-content"
              rows={5}
              value={editContent}
              maxLength={2000}
              onChange={(event) => setEditContent(event.target.value)}
            />
          </Field>
        </form>
      </Modal>

      <ConfirmDialog
        open={Boolean(deleteTarget)}
        onClose={() => setDeleteTarget(null)}
        onConfirm={handleDelete}
        title="Delete this message?"
        description="The message will be removed from the conversation."
        confirmLabel="Delete message"
        destructive
        loading={deleting}
      />
    </>
  );
}
