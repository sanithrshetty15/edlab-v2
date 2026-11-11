import { useMemo, useState } from "react";
import { borrowingApi, returnsApi } from "../../../api/services";
import { useApiResource } from "../../../hooks/useApiResource";
import { Modal } from "../../../components/app/Modal";
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
  StatusBadge,
  Tabs,
  TextArea,
} from "../../../components/app/Ui";
import {
  BoxIcon,
  CheckIcon,
  ClockIcon,
  RefreshIcon,
  XCircleIcon,
} from "../../../components/app/AppIcons";
import { formatDate, formatDateTime, formatRelative, titleCase } from "../../../utils/format";

const BORROW_STATUS_OPTIONS = [
  { value: "", label: "All requests" },
  { value: "pending", label: "Pending" },
  { value: "approved", label: "Approved" },
  { value: "borrowed", label: "Borrowed" },
  { value: "returned", label: "Returned" },
  { value: "rejected", label: "Rejected" },
  { value: "overdue", label: "Overdue" },
];

const RETURN_STATUS_OPTIONS = [
  { value: "", label: "All returns" },
  { value: "pending", label: "Pending" },
  { value: "confirmed", label: "Confirmed" },
  { value: "rejected", label: "Rejected" },
];

function ActionDialog({ open, title, description, label, required, submitLabel, tone = "primary", onClose, onSubmit }) {
  const [value, setValue] = useState("");
  const [error, setError] = useState(null);
  const [saving, setSaving] = useState(false);

  const handleSubmit = async (event) => {
    event.preventDefault();
    if (required && !value.trim()) {
      setError(new Error(`${label} is required.`));
      return;
    }
    setSaving(true);
    setError(null);
    try {
      await onSubmit(value.trim());
      onClose();
    } catch (requestError) {
      setError(requestError);
    } finally {
      setSaving(false);
    }
  };

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={title}
      description={description}
      footer={
        <>
          <AppButton variant="ghost" onClick={onClose} disabled={saving}>Cancel</AppButton>
          <AppButton
            form="action-dialog-form"
            type="submit"
            variant={tone}
            loading={saving}
          >
            {submitLabel}
          </AppButton>
        </>
      }
    >
      <form id="action-dialog-form" className="lab-form-stack" onSubmit={handleSubmit}>
        {error && <Alert tone="danger">{error.message}</Alert>}
        <Field label={label} htmlFor="action-dialog-note" required={required}>
          <TextArea
            id="action-dialog-note"
            rows={4}
            value={value}
            onChange={(event) => setValue(event.target.value)}
            disabled={saving}
          />
        </Field>
      </form>
    </Modal>
  );
}

function BorrowRequestsPanel() {
  const [status, setStatus] = useState("");
  const [page, setPage] = useState(1);
  const [dialog, setDialog] = useState(null);
  const [notice, setNotice] = useState(null);
  const [actionError, setActionError] = useState(null);

  const query = useMemo(
    () => ({
      page,
      per_page: 20,
      status: status && status !== "overdue" ? status : undefined,
      overdue: status === "overdue" ? "true" : undefined,
    }),
    [page, status],
  );

  const resource = useApiResource(
    () => borrowingApi.adminList(query),
    `borrow-requests-${JSON.stringify(query)}`,
  );

  const requests = resource.data?.requests ?? [];
  const pagination = resource.data?.pagination;

  const runAction = async (request, action, reason) => {
    setActionError(null);
    try {
      if (action === "approve") {
        const result = await borrowingApi.approve(request.id, {
          approval_reason: reason || undefined,
        });
        setNotice(result.message || "Borrow request approved.");
      } else {
        const result = await borrowingApi.reject(request.id, { reason });
        setNotice(result.message || "Borrow request rejected.");
      }
      resource.reload();
    } catch (error) {
      setActionError(error);
      throw error;
    }
  };

  return (
    <>
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

      <div className="lab-toolbar lab-toolbar--wrap">
        <div className="lab-summary-pills">
          <Badge tone="warning">{pagination?.pending_count ?? 0} pending</Badge>
          <Badge tone="danger">{pagination?.overdue_count ?? 0} overdue</Badge>
        </div>
        <div className="lab-toolbar__right">
          <Select
            value={status}
            onChange={(event) => {
              setStatus(event.target.value);
              setPage(1);
            }}
            aria-label="Filter borrow requests"
            className="lab-select--compact"
          >
            {BORROW_STATUS_OPTIONS.map((option) => (
              <option key={option.value} value={option.value}>{option.label}</option>
            ))}
          </Select>
          <AppButton variant="outline" size="sm" onClick={() => resource.reload()} loading={resource.loading}>
            <RefreshIcon className="h-3.5 w-3.5" />
            Refresh
          </AppButton>
        </div>
      </div>

      <div className="lab-list-panel__body">
        {resource.loading ? (
          <LoadingState rows={7} />
        ) : resource.error ? (
          <ErrorState error={resource.error} onRetry={resource.reload} title="Borrow requests are unavailable" />
        ) : requests.length === 0 ? (
          <EmptyState
            icon={BoxIcon}
            title="No borrow requests found"
            description="Requests submitted by members will appear here."
          />
        ) : (
          <div className="lab-table-wrap">
            <table className="lab-table">
              <thead>
                <tr>
                  <th>Member</th>
                  <th>Item</th>
                  <th>Qty</th>
                  <th>Requested</th>
                  <th>Due</th>
                  <th>Status</th>
                  <th aria-label="Actions" />
                </tr>
              </thead>
              <tbody>
                {requests.map((request) => (
                  <tr key={request.id}>
                    <td>
                      <div className="lab-member-academic">
                        <strong>{request.member?.name || "—"}</strong>
                        <small>{request.member?.login_id || request.member?.email || "—"}</small>
                      </div>
                    </td>
                    <td>
                      <div className="lab-member-academic">
                        <strong>{request.item?.name || "—"}</strong>
                        <small>{request.item?.category || "—"}</small>
                      </div>
                    </td>
                    <td>{request.quantity}</td>
                    <td>{formatDate(request.borrowed_at)}</td>
                    <td>
                      <div className="lab-due-cell">
                        <span>{formatDate(request.due_date)}</span>
                        {request.days_overdue > 0 && (
                          <Badge tone="danger">{request.days_overdue}d overdue</Badge>
                        )}
                      </div>
                    </td>
                    <td><StatusBadge status={request.status}>{titleCase(request.status)}</StatusBadge></td>
                    <td>
                      {request.status === "pending" ? (
                        <div className="lab-row-actions">
                          <AppButton
                            variant="ghost"
                            size="sm"
                            onClick={() => setDialog({ request, action: "approve" })}
                          >
                            <CheckIcon className="h-3.5 w-3.5" />
                            Approve
                          </AppButton>
                          <AppButton
                            variant="ghost"
                            size="sm"
                            className="lab-danger-text"
                            onClick={() => setDialog({ request, action: "reject" })}
                          >
                            <XCircleIcon className="h-3.5 w-3.5" />
                            Reject
                          </AppButton>
                        </div>
                      ) : (
                        <span className="lab-muted lab-small">
                          {request.approved_by ? `By ${request.approved_by}` : "—"}
                        </span>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {pagination && (
        <Pagination
          page={page}
          totalPages={Math.max(Math.ceil((pagination.total_count || 0) / 20), 1)}
          totalCount={pagination.total_count}
          onPageChange={setPage}
          loading={resource.loading}
        />
      )}

      {dialog && (
        <ActionDialog
          key={`${dialog.request.id}-${dialog.action}`}
          open
          title={dialog.action === "approve" ? "Approve borrow request" : "Reject borrow request"}
          description={`${dialog.request.member?.name || "Member"} · ${dialog.request.item?.name || "Item"} × ${dialog.request.quantity}`}
          label={dialog.action === "approve" ? "Approval note (optional)" : "Rejection reason"}
          required={dialog.action === "reject"}
          submitLabel={dialog.action === "approve" ? "Approve request" : "Reject request"}
          tone={dialog.action === "reject" ? "danger" : "primary"}
          onClose={() => setDialog(null)}
          onSubmit={(reason) => runAction(dialog.request, dialog.action, reason)}
        />
      )}
    </>
  );
}

function ReturnRequestsPanel() {
  const [status, setStatus] = useState("");
  const [page, setPage] = useState(1);
  const [dialog, setDialog] = useState(null);
  const [notice, setNotice] = useState(null);
  const [actionError, setActionError] = useState(null);

  const query = useMemo(
    () => ({ page, per_page: 20, status: status || undefined }),
    [page, status],
  );

  const resource = useApiResource(
    () => returnsApi.adminList(query),
    `return-requests-${JSON.stringify(query)}`,
  );

  const requests = resource.data?.requests ?? [];
  const pagination = resource.data?.pagination;

  const runAction = async (request, action, value) => {
    setActionError(null);
    try {
      if (action === "confirm") {
        const result = await returnsApi.confirm(request.id, {
          condition_note: value || undefined,
        });
        setNotice(result.message || "Return confirmed.");
      } else {
        const result = await returnsApi.reject(request.id, { reason: value });
        setNotice(result.message || "Return request rejected.");
      }
      resource.reload();
    } catch (error) {
      setActionError(error);
      throw error;
    }
  };

  return (
    <>
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

      <div className="lab-toolbar lab-toolbar--wrap">
        <div className="lab-summary-pills">
          <Badge tone="warning">{pagination?.pending_count ?? 0} pending</Badge>
        </div>
        <div className="lab-toolbar__right">
          <Select
            value={status}
            onChange={(event) => {
              setStatus(event.target.value);
              setPage(1);
            }}
            aria-label="Filter return requests"
            className="lab-select--compact"
          >
            {RETURN_STATUS_OPTIONS.map((option) => (
              <option key={option.value} value={option.value}>{option.label}</option>
            ))}
          </Select>
          <AppButton variant="outline" size="sm" onClick={() => resource.reload()} loading={resource.loading}>
            <RefreshIcon className="h-3.5 w-3.5" />
            Refresh
          </AppButton>
        </div>
      </div>

      <div className="lab-list-panel__body">
        {resource.loading ? (
          <LoadingState rows={7} />
        ) : resource.error ? (
          <ErrorState error={resource.error} onRetry={resource.reload} title="Return requests are unavailable" />
        ) : requests.length === 0 ? (
          <EmptyState
            icon={ClockIcon}
            title="No return requests found"
            description="Return requests submitted by members will appear here."
          />
        ) : (
          <div className="lab-table-wrap">
            <table className="lab-table">
              <thead>
                <tr>
                  <th>Member</th>
                  <th>Item</th>
                  <th>Borrowed</th>
                  <th>Requested</th>
                  <th>Note</th>
                  <th>Status</th>
                  <th aria-label="Actions" />
                </tr>
              </thead>
              <tbody>
                {requests.map((request) => (
                  <tr key={request.id}>
                    <td>
                      <div className="lab-member-academic">
                        <strong>{request.member?.name || "—"}</strong>
                        <small>{request.member?.login_id || request.member?.email || "—"}</small>
                      </div>
                    </td>
                    <td>
                      <div className="lab-member-academic">
                        <strong>{request.item?.name || "—"}</strong>
                        <small>{request.borrow_record?.quantity || 0} unit(s)</small>
                      </div>
                    </td>
                    <td>{formatDate(request.borrow_record?.borrowed_at)}</td>
                    <td title={formatDateTime(request.created_at)}>{formatRelative(request.created_at)}</td>
                    <td>{request.condition_note || "—"}</td>
                    <td><StatusBadge status={request.status}>{titleCase(request.status)}</StatusBadge></td>
                    <td>
                      {request.status === "pending" ? (
                        <div className="lab-row-actions">
                          <AppButton
                            variant="ghost"
                            size="sm"
                            onClick={() => setDialog({ request, action: "confirm" })}
                          >
                            <CheckIcon className="h-3.5 w-3.5" />
                            Confirm
                          </AppButton>
                          <AppButton
                            variant="ghost"
                            size="sm"
                            className="lab-danger-text"
                            onClick={() => setDialog({ request, action: "reject" })}
                          >
                            <XCircleIcon className="h-3.5 w-3.5" />
                            Reject
                          </AppButton>
                        </div>
                      ) : (
                        <span className="lab-muted lab-small">
                          {request.confirmed_by ? `By ${request.confirmed_by}` : "—"}
                        </span>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {pagination && (
        <Pagination
          page={page}
          totalPages={Math.max(Math.ceil((pagination.total_count || 0) / 20), 1)}
          totalCount={pagination.total_count}
          onPageChange={setPage}
          loading={resource.loading}
        />
      )}

      {dialog && (
        <ActionDialog
          key={`${dialog.request.id}-${dialog.action}`}
          open
          title={dialog.action === "confirm" ? "Confirm return" : "Reject return request"}
          description={`${dialog.request.member?.name || "Member"} · ${dialog.request.item?.name || "Item"}`}
          label={dialog.action === "confirm" ? "Condition note (optional)" : "Rejection reason"}
          required={dialog.action === "reject"}
          submitLabel={dialog.action === "confirm" ? "Confirm return" : "Reject request"}
          tone={dialog.action === "reject" ? "danger" : "primary"}
          onClose={() => setDialog(null)}
          onSubmit={(value) => runAction(dialog.request, dialog.action, value)}
        />
      )}
    </>
  );
}

export default function BorrowingPage() {
  const [tab, setTab] = useState("borrowing");

  return (
    <>
      <PageHeader
        eyebrow="Admin console"
        title="Borrowing & returns"
        description="Approve equipment requests and confirm returned stock."
      />

      <Panel padded={false}>
        <div className="lab-toolbar lab-toolbar--wrap">
          <Tabs
            tabs={[
              { value: "borrowing", label: "Borrow requests" },
              { value: "returns", label: "Returns" },
            ]}
            value={tab}
            onChange={setTab}
            label="Borrowing and returns"
          />
        </div>
        {tab === "borrowing" ? <BorrowRequestsPanel /> : <ReturnRequestsPanel />}
      </Panel>
    </>
  );
}
