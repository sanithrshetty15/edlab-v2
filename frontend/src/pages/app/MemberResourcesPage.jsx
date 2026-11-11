import { useEffect, useMemo, useState } from "react";
import { borrowingApi, inventoryApi, returnsApi } from "../../api/services";
import { useApiResource } from "../../hooks/useApiResource";
import { Modal } from "../../components/app/Modal";
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
  TextInput,
} from "../../components/app/Ui";
import {
  BoxIcon,
  RefreshIcon,
  SearchIcon,
  ShieldIcon,
} from "../../components/app/AppIcons";
import { formatDate, formatRelative, titleCase } from "../../utils/format";

const VIEW_TABS = [
  { value: "catalog", label: "Borrow equipment" },
  { value: "borrows", label: "My borrowing" },
  { value: "returns", label: "My returns" },
];

function RequestDialog({ item, onClose, onSaved }) {
  const [quantity, setQuantity] = useState("1");
  const [purpose, setPurpose] = useState("");
  const [error, setError] = useState(null);
  const [formError, setFormError] = useState(null);
  const [saving, setSaving] = useState(false);

  const handleSubmit = async (event) => {
    event.preventDefault();
    if (saving) return;

    const amount = Number(quantity);
    if (!Number.isInteger(amount) || amount < 1) {
      setError("Choose a quantity of at least one.");
      return;
    }
    if (amount > item.available_quantity) {
      setError(`Only ${item.available_quantity} available.`);
      return;
    }
    if (!purpose.trim()) {
      setError("Tell the lab what you need it for.");
      return;
    }

    setSaving(true);
    setError(null);
    setFormError(null);
    try {
      const result = await borrowingApi.create({
        item_id: item.id,
        quantity: amount,
        purpose: purpose.trim(),
      });
      onSaved(result);
      onClose();
    } catch (requestError) {
      setFormError(requestError);
    } finally {
      setSaving(false);
    }
  };

  return (
    <Modal
      open
      onClose={onClose}
      title={`Request ${item.name}`}
      description={`${item.available_quantity} of ${item.total_quantity} available${
        item.storage_location ? ` · stored at ${item.storage_location}` : ""
      }`}
      footer={
        <>
          <AppButton variant="ghost" onClick={onClose} disabled={saving}>
            Cancel
          </AppButton>
          <AppButton form="borrow-request-form" type="submit" variant="primary" loading={saving}>
            Submit request
          </AppButton>
        </>
      }
    >
      <form id="borrow-request-form" className="lab-form-stack" onSubmit={handleSubmit} noValidate>
        {formError && (
          <Alert tone="danger" title="Request could not be submitted">
            {formError.message}
          </Alert>
        )}
        <Field
          label="Quantity"
          htmlFor="borrow-quantity"
          error={error?.startsWith("Choose") || error?.startsWith("Only") ? error : undefined}
          required
        >
          <TextInput
            id="borrow-quantity"
            type="number"
            min="1"
            max={item.available_quantity}
            value={quantity}
            onChange={(event) => setQuantity(event.target.value)}
            disabled={saving}
          />
        </Field>
        <Field
          label="Purpose"
          htmlFor="borrow-purpose"
          error={error?.startsWith("Tell") ? error : undefined}
          hint="Explain how you will use the equipment so an admin can review it."
          required
        >
          <TextArea
            id="borrow-purpose"
            rows={4}
            value={purpose}
            onChange={(event) => setPurpose(event.target.value)}
            placeholder="Project, event or lab activity"
            disabled={saving}
          />
        </Field>
        <p className="lab-muted">
          Requests are reviewed by an admin. Approved items are due before the end of
          the current lab semester.
        </p>
      </form>
    </Modal>
  );
}

function ReturnDialog({ borrow, onClose, onSaved }) {
  const [note, setNote] = useState("");
  const [formError, setFormError] = useState(null);
  const [saving, setSaving] = useState(false);

  const handleSubmit = async (event) => {
    event.preventDefault();
    if (saving) return;

    setSaving(true);
    setFormError(null);
    try {
      const result = await returnsApi.create({
        borrow_record_id: borrow.id,
        condition_note: note.trim() || undefined,
      });
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
      title={`Return ${borrow.item?.name || "equipment"}`}
      description={`Borrowed ${formatDate(borrow.borrowed_at)} · due ${formatDate(borrow.due_date)}`}
      footer={
        <>
          <AppButton variant="ghost" onClick={onClose} disabled={saving}>
            Cancel
          </AppButton>
          <AppButton form="return-request-form" type="submit" variant="primary" loading={saving}>
            Request return
          </AppButton>
        </>
      }
    >
      <form id="return-request-form" className="lab-form-stack" onSubmit={handleSubmit}>
        {formError && (
          <Alert tone="danger" title="Return could not be requested">
            {formError.message}
          </Alert>
        )}
        <Field
          label="Condition note"
          htmlFor="return-condition"
          hint="Optional. Mention any damage or missing parts."
        >
          <TextArea
            id="return-condition"
            rows={4}
            value={note}
            onChange={(event) => setNote(event.target.value)}
            disabled={saving}
          />
        </Field>
      </form>
    </Modal>
  );
}

export default function MemberResourcesPage() {
  const [view, setView] = useState("catalog");
  const [search, setSearch] = useState("");
  const [category, setCategory] = useState("");
  const [page, setPage] = useState(1);
  const [requestItem, setRequestItem] = useState(null);
  const [returnBorrow, setReturnBorrow] = useState(null);
  const [notice, setNotice] = useState(null);
  const [actionError, setActionError] = useState(null);

  const catalogQuery = useMemo(
    () => ({ page, per_page: 20, search: search.trim() || undefined }),
    [page, search],
  );

  const catalog = useApiResource(
    () => inventoryApi.list(catalogQuery),
    `member-resources-${JSON.stringify(catalogQuery)}`,
  );
  const borrows = useApiResource(
    () => borrowingApi.list({ per_page: 50 }),
    "member-borrows",
  );
  const returns = useApiResource(
    () => returnsApi.list({ per_page: 50 }),
    "member-returns",
  );

  const reloadBorrows = borrows.reload;
  const reloadReturns = returns.reload;

  useEffect(() => {
    if (view === "borrows") reloadBorrows();
    if (view === "returns") reloadReturns();
  }, [view, reloadBorrows, reloadReturns]);

  const items = catalog.data?.items ?? [];
  const pagination = catalog.data?.pagination;
  const borrowList = borrows.data?.requests ?? [];
  const returnList = returns.data?.requests ?? [];
  const categories = [...new Set(items.map((item) => item.category).filter(Boolean))].sort();
  const visibleItems = category
    ? items.filter((item) => item.category === category)
    : items;
  const activeBorrows = borrowList.filter((borrow) => borrow.status === "approved" || borrow.status === "borrowed");

  const reloadAll = () => {
    catalog.reload();
    borrows.reload();
    returns.reload();
  };

  return (
    <>
      <PageHeader
        eyebrow="Member portal"
        title="Lab resources"
        description="Request lab equipment, track due dates and submit returns."
        actions={
          <AppButton variant="outline" onClick={reloadAll} loading={catalog.loading || borrows.loading}>
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
        <Alert tone="danger" title="Resource action failed" onDismiss={() => setActionError(null)}>
          {actionError.message}
        </Alert>
      )}

      <Panel padded={false}>
        <div className="lab-toolbar lab-toolbar--wrap">
          <Tabs
            tabs={VIEW_TABS.map((tab) =>
              tab.value === "borrows" && activeBorrows.length
                ? { ...tab, count: activeBorrows.length }
                : tab,
            )}
            value={view}
            onChange={setView}
            label="Resource views"
          />
          {view === "catalog" && (
            <div className="lab-toolbar__right">
              <Select
                value={category}
                onChange={(event) => setCategory(event.target.value)}
                aria-label="Filter by category"
                className="lab-select--compact"
              >
                <option value="">All categories</option>
                {categories.map((value) => (
                  <option key={value} value={value}>{titleCase(value)}</option>
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
                  placeholder="Search equipment"
                  aria-label="Search equipment"
                />
              </div>
            </div>
          )}
        </div>

        <div className="lab-list-panel__body">
          {view === "catalog" && (
            catalog.loading ? (
              <LoadingState rows={6} />
            ) : catalog.error ? (
              <ErrorState error={catalog.error} onRetry={catalog.reload} title="Inventory is unavailable" />
            ) : visibleItems.length === 0 ? (
              <EmptyState
                icon={BoxIcon}
                title={search || category ? "No matching equipment" : "No equipment available"}
                description={
                  search || category
                    ? "Try another search or category."
                    : "Available lab equipment will appear here when stock is added."
                }
              />
            ) : (
              <div className="lab-resource-grid">
                {visibleItems.map((item) => (
                  <article key={item.id} className="lab-resource-card">
                    <div className="lab-resource-card__top">
                      <Badge tone="info">{titleCase(item.category || "equipment")}</Badge>
                      <Badge tone={item.available_quantity > 0 ? "success" : "neutral"} dot>
                        {item.available_quantity} available
                      </Badge>
                    </div>
                    <h3>{item.name}</h3>
                    <div className="lab-resource-card__meta">
                      <span>Total {item.total_quantity}</span>
                      {item.storage_location && <span>{item.storage_location}</span>}
                      {item.condition && <span>{titleCase(item.condition)}</span>}
                    </div>
                    <div className="lab-resource-card__actions">
                      <AppButton
                        variant="primary"
                        size="sm"
                        onClick={() => setRequestItem(item)}
                        disabled={item.available_quantity < 1}
                      >
                        Request
                      </AppButton>
                    </div>
                  </article>
                ))}
              </div>
            )
          )}

          {view === "borrows" && (
            borrows.loading ? (
              <LoadingState rows={5} />
            ) : borrows.error ? (
              <ErrorState error={borrows.error} onRetry={borrows.reload} title="Borrow history is unavailable" />
            ) : borrowList.length === 0 ? (
              <EmptyState
                icon={BoxIcon}
                title="No borrowing yet"
                description="Request equipment from the catalog and your borrow records will appear here."
                action={
                  <AppButton variant="outline" onClick={() => setView("catalog")}>
                    Browse equipment
                  </AppButton>
                }
              />
            ) : (
              <div className="lab-table-wrap">
                <table className="lab-table">
                  <thead>
                    <tr>
                      <th>Item</th>
                      <th>Qty</th>
                      <th>Borrowed</th>
                      <th>Due</th>
                      <th>Status</th>
                      <th aria-label="Actions" />
                    </tr>
                  </thead>
                  <tbody>
                    {borrowList.map((borrow) => {
                      const canReturn = borrow.status === "approved" || borrow.status === "borrowed";
                      return (
                        <tr key={borrow.id}>
                          <td>
                            <div className="lab-member-academic">
                              <strong>{borrow.item?.name || "—"}</strong>
                              <small>{titleCase(borrow.item?.category || "equipment")}</small>
                            </div>
                          </td>
                          <td>{borrow.quantity}</td>
                          <td>{formatDate(borrow.borrowed_at)}</td>
                          <td>{formatDate(borrow.due_date)}</td>
                          <td><StatusBadge status={borrow.status}>{titleCase(borrow.status)}</StatusBadge></td>
                          <td>
                            {canReturn && (
                              <AppButton variant="ghost" size="sm" onClick={() => setReturnBorrow(borrow)}>
                                <ShieldIcon className="h-3.5 w-3.5" />
                                Request return
                              </AppButton>
                            )}
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>
            )
          )}

          {view === "returns" && (
            returns.loading ? (
              <LoadingState rows={5} />
            ) : returns.error ? (
              <ErrorState error={returns.error} onRetry={returns.reload} title="Returns are unavailable" />
            ) : returnList.length === 0 ? (
              <EmptyState
                icon={ShieldIcon}
                title="No return requests"
                description="Submit a return from your borrowing history and its review status will appear here."
              />
            ) : (
              <div className="lab-table-wrap">
                <table className="lab-table">
                  <thead>
                    <tr>
                      <th>Item</th>
                      <th>Requested</th>
                      <th>Status</th>
                      <th>Condition note</th>
                    </tr>
                  </thead>
                  <tbody>
                    {returnList.map((request) => (
                      <tr key={request.id}>
                        <td>
                          <div className="lab-member-academic">
                            <strong>{request.item?.name || "—"}</strong>
                            <small>{titleCase(request.item?.category || "equipment")}</small>
                          </div>
                        </td>
                        <td>{formatRelative(request.created_at)}</td>
                        <td><StatusBadge status={request.status}>{titleCase(request.status)}</StatusBadge></td>
                        <td>{request.condition_note || "—"}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )
          )}
        </div>

        {view === "catalog" && pagination && (
          <Pagination
            page={pagination.page}
            totalPages={pagination.total_pages ?? Math.ceil(pagination.total_count / (pagination.per_page || 20))}
            totalCount={pagination.total_count}
            onPageChange={setPage}
            loading={catalog.loading}
          />
        )}
      </Panel>

      {requestItem && (
        <RequestDialog
          item={requestItem}
          onClose={() => {
            setRequestItem(null);
            setActionError(null);
          }}
          onSaved={(result) => {
            setNotice(result.message || "Borrow request submitted.");
            reloadAll();
          }}
        />
      )}

      {returnBorrow && (
        <ReturnDialog
          borrow={returnBorrow}
          onClose={() => setReturnBorrow(null)}
          onSaved={(result) => {
            setNotice(result.message || "Return requested.");
            reloadAll();
          }}
        />
      )}
    </>
  );
}
