import { useMemo, useState } from "react";
import { inventoryApi } from "../../../api/services";
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
  StatCard,
  TextArea,
  TextInput,
} from "../../../components/app/Ui";
import {
  BoxIcon,
  EditIcon,
  PlusIcon,
  SearchIcon,
  TrashIcon,
  UsersIcon,
} from "../../../components/app/AppIcons";
import { formatDate, formatNumber, titleCase } from "../../../utils/format";

function InventoryForm({ item, onClose, onSaved }) {
  const [name, setName] = useState(item?.name || "");
  const [category, setCategory] = useState(item?.category || "");
  const [description, setDescription] = useState(item?.description || "");
  const [totalQuantity, setTotalQuantity] = useState(item?.total_quantity ?? 1);
  const [storageLocation, setStorageLocation] = useState(item?.storage_location || "");
  const [condition, setCondition] = useState(item?.condition || "");
  const [errors, setErrors] = useState({});
  const [formError, setFormError] = useState(null);
  const [saving, setSaving] = useState(false);
  const editing = Boolean(item?.id);

  const handleSubmit = async (event) => {
    event.preventDefault();
    if (saving) return;

    const nextErrors = {};
    if (!name.trim()) nextErrors.name = "Item name is required.";
    if (!category.trim()) nextErrors.category = "Category is required.";
    if (totalQuantity === "" || Number(totalQuantity) < 0 || Number.isNaN(Number(totalQuantity))) {
      nextErrors.totalQuantity = "Enter a non-negative quantity.";
    }
    setErrors(nextErrors);
    setFormError(null);
    if (Object.keys(nextErrors).length) return;

    setSaving(true);
    try {
      const payload = {
        name: name.trim(),
        category: category.trim(),
        description: description.trim(),
        total_quantity: Number(totalQuantity),
        storage_location: storageLocation.trim(),
        condition: condition.trim(),
      };
      const result = editing
        ? await inventoryApi.update(item.id, payload)
        : await inventoryApi.create(payload);
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
      title={editing ? "Edit inventory item" : "New inventory item"}
      description="Changes here affect live stock and borrowing availability."
      size="lg"
      footer={
        <>
          <AppButton variant="ghost" onClick={onClose} disabled={saving}>
            Cancel
          </AppButton>
          <AppButton form="inventory-form" type="submit" variant="primary" loading={saving}>
            {editing ? "Save item" : "Create item"}
          </AppButton>
        </>
      }
    >
      <form id="inventory-form" className="lab-form-stack" onSubmit={handleSubmit} noValidate>
        {formError && (
          <Alert tone="danger" title="Item could not be saved">
            {formError.message}
          </Alert>
        )}
        {editing && (
          <Alert tone="warning" title="Changing total quantity">
            Available quantity is recalculated against existing loans when the total changes.
          </Alert>
        )}

        <div className="lab-form-grid">
          <Field label="Item name" htmlFor="inventory-name" error={errors.name} required>
            <TextInput
              id="inventory-name"
              value={name}
              onChange={(event) => setName(event.target.value)}
              disabled={saving}
            />
          </Field>
          <Field label="Category" htmlFor="inventory-category" error={errors.category} required>
            <TextInput
              id="inventory-category"
              value={category}
              onChange={(event) => setCategory(event.target.value)}
              placeholder="e.g. Development boards"
              disabled={saving}
            />
          </Field>
        </div>

        <Field label="Description" htmlFor="inventory-description">
          <TextArea
            id="inventory-description"
            rows={4}
            value={description}
            onChange={(event) => setDescription(event.target.value)}
            disabled={saving}
          />
        </Field>

        <div className="lab-form-grid lab-form-grid--3">
          <Field label="Total quantity" htmlFor="inventory-quantity" error={errors.totalQuantity} required>
            <TextInput
              id="inventory-quantity"
              type="number"
              min="0"
              step="1"
              value={totalQuantity}
              onChange={(event) => setTotalQuantity(event.target.value)}
              disabled={saving}
            />
          </Field>
          <Field label="Storage location" htmlFor="inventory-location">
            <TextInput
              id="inventory-location"
              value={storageLocation}
              onChange={(event) => setStorageLocation(event.target.value)}
              placeholder="e.g. Cabinet B, shelf 2"
              disabled={saving}
            />
          </Field>
          <Field label="Condition" htmlFor="inventory-condition">
            <TextInput
              id="inventory-condition"
              value={condition}
              onChange={(event) => setCondition(event.target.value)}
              placeholder="e.g. good"
              disabled={saving}
            />
          </Field>
        </div>
      </form>
    </Modal>
  );
}

function InventoryDetail({ item, loading, error, onClose, onEdit }) {
  return (
    <Modal
      open
      onClose={onClose}
      title={item?.name || "Inventory item"}
      size="lg"
      footer={
        <>
          <AppButton variant="ghost" onClick={onClose}>Close</AppButton>
          <AppButton variant="primary" onClick={onEdit} disabled={!item}>
            <EditIcon className="h-4 w-4" />
            Edit item
          </AppButton>
        </>
      }
    >
      {loading ? (
        <LoadingState rows={6} />
      ) : error ? (
        <ErrorState error={error} title="Inventory item could not be loaded" />
      ) : item ? (
        <div className="lab-inventory-detail">
          <div className="lab-inventory-detail__badges">
            <Badge tone="info">{item.category}</Badge>
            <Badge tone={item.available_quantity > 0 ? "success" : "danger"} dot>
              {item.available_quantity > 0 ? "Available" : "Out of stock"}
            </Badge>
            {item.condition && <Badge tone="neutral">{titleCase(item.condition)}</Badge>}
          </div>

          <div className="lab-inventory-detail__stats">
            <div>
              <span>Total quantity</span>
              <strong>{formatNumber(item.total_quantity)}</strong>
            </div>
            <div>
              <span>Available</span>
              <strong>{formatNumber(item.available_quantity)}</strong>
            </div>
            <div>
              <span>On loan</span>
              <strong>{formatNumber(Number(item.total_quantity) - Number(item.available_quantity))}</strong>
            </div>
          </div>

          <section>
            <h3>Description</h3>
            <p>{item.description || "No description provided."}</p>
          </section>

          <div className="lab-inventory-detail__meta">
            <div>
              <span>Storage location</span>
              <strong>{item.storage_location || "—"}</strong>
            </div>
            <div>
              <span>Added</span>
              <strong>{formatDate(item.created_at)}</strong>
            </div>
          </div>

          <section>
            <h3>Recent borrowing history</h3>
            {item.borrow_history?.length ? (
              <div className="lab-table-wrap">
                <table className="lab-table lab-table--compact">
                  <thead>
                    <tr>
                      <th>Member</th>
                      <th>Borrowed</th>
                      <th>Due</th>
                      <th>Status</th>
                    </tr>
                  </thead>
                  <tbody>
                    {item.borrow_history.map((record) => (
                      <tr key={record.id}>
                        <td>{record.member?.name || "—"}</td>
                        <td>{formatDate(record.borrowed_at)}</td>
                        <td>{formatDate(record.due_date)}</td>
                        <td><Badge tone="neutral">{titleCase(record.status)}</Badge></td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            ) : (
              <p className="lab-muted lab-small">No borrowing history recorded.</p>
            )}
          </section>
        </div>
      ) : null}
    </Modal>
  );
}

export default function InventoryPage() {
  const [searchInput, setSearchInput] = useState("");
  const [filters, setFilters] = useState({ page: 1, search: "", category: "", includeOutOfStock: false });
  const [selectedItem, setSelectedItem] = useState(null);
  const [loadingItem, setLoadingItem] = useState(false);
  const [itemError, setItemError] = useState(null);
  const [formItem, setFormItem] = useState(null);
  const [formOpen, setFormOpen] = useState(false);
  const [editLoading, setEditLoading] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState(null);
  const [deleting, setDeleting] = useState(false);
  const [notice, setNotice] = useState(null);
  const [actionError, setActionError] = useState(null);

  const query = useMemo(
    () => ({
      page: filters.page,
      per_page: 25,
      search: filters.search || undefined,
      category: filters.category || undefined,
      in_stock: filters.includeOutOfStock ? "false" : undefined,
    }),
    [filters],
  );

  const resource = useApiResource(
    () => inventoryApi.list(query),
    `inventory-${JSON.stringify(query)}`,
  );
  const statsResource = useApiResource(() => inventoryApi.stats(), "inventory-stats");

  const items = resource.data?.items ?? [];
  const stats = statsResource.data?.stats;

  const openDetail = async (item) => {
    setSelectedItem({ ...item, borrow_history: [] });
    setLoadingItem(true);
    setItemError(null);
    try {
      const result = await inventoryApi.get(item.id);
      setSelectedItem({ ...result.item, borrow_history: result.borrow_history || [] });
    } catch (error) {
      setItemError(error);
    } finally {
      setLoadingItem(false);
    }
  };

  const openEdit = async (item) => {
    setEditLoading(true);
    setActionError(null);
    try {
      const result = await inventoryApi.get(item.id);
      setFormItem(result.item);
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
      await inventoryApi.remove(deleteTarget.id);
      setNotice("Inventory item removed from active stock.");
      setDeleteTarget(null);
      resource.reload();
      statsResource.reload();
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
        title="Inventory"
        description="Manage lab equipment, stock levels and storage details."
        actions={
          <AppButton
            variant="primary"
            onClick={() => {
              setFormItem(null);
              setFormOpen(true);
            }}
          >
            <PlusIcon className="h-4 w-4" />
            New item
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

      <div className="lab-stat-grid lab-stat-grid--4">
        <StatCard
          label="Inventory items"
          value={formatNumber(stats?.total_items)}
          hint="Active item records"
          icon={BoxIcon}
          loading={statsResource.loading}
        />
        <StatCard
          label="Total stock"
          value={formatNumber(stats?.total_stock)}
          hint="All quantities"
          icon={BoxIcon}
          loading={statsResource.loading}
        />
        <StatCard
          label="Available stock"
          value={formatNumber(stats?.available_stock)}
          hint="Ready to borrow"
          icon={BoxIcon}
          tone="accent"
          loading={statsResource.loading}
        />
        <StatCard
          label="On loan"
          value={formatNumber(stats?.borrowed_stock)}
          hint="Currently borrowed"
          icon={UsersIcon}
          loading={statsResource.loading}
        />
      </div>

      <Panel padded={false}>
        <form
          className="lab-toolbar lab-toolbar--wrap"
          onSubmit={(event) => {
            event.preventDefault();
            setFilters((current) => ({ ...current, search: searchInput.trim(), page: 1 }));
          }}
        >
          <div className="lab-search">
            <SearchIcon className="h-4 w-4" />
            <input
              type="search"
              value={searchInput}
              onChange={(event) => setSearchInput(event.target.value)}
              placeholder="Search inventory"
              aria-label="Search inventory"
            />
          </div>
          <div className="lab-toolbar__right">
            <Select
              value={filters.category}
              onChange={(event) =>
                setFilters((current) => ({ ...current, category: event.target.value, page: 1 }))
              }
              aria-label="Filter by category"
              className="lab-select--compact"
            >
              <option value="">All categories</option>
              {(stats?.categories ?? []).map((category) => (
                <option key={category.category} value={category.category}>{category.category}</option>
              ))}
            </Select>
            <Checkbox
              label="Include out of stock"
              checked={filters.includeOutOfStock}
              onChange={(event) =>
                setFilters((current) => ({
                  ...current,
                  includeOutOfStock: event.target.checked,
                  page: 1,
                }))
              }
            />
            <AppButton type="submit" variant="outline" size="sm">Apply</AppButton>
          </div>
        </form>

        <div className="lab-list-panel__body">
          {resource.loading ? (
            <LoadingState rows={7} />
          ) : resource.error ? (
            <ErrorState error={resource.error} onRetry={resource.reload} title="Inventory is unavailable" />
          ) : items.length === 0 ? (
            <EmptyState
              icon={BoxIcon}
              title="No inventory items found"
              description="Adjust the filters or add an item to the inventory."
              action={
                <AppButton variant="primary" onClick={() => setFormOpen(true)}>
                  <PlusIcon className="h-4 w-4" />
                  New item
                </AppButton>
              }
            />
          ) : (
            <div className="lab-table-wrap">
              <table className="lab-table">
                <thead>
                  <tr>
                    <th>Item</th>
                    <th>Category</th>
                    <th>Stock</th>
                    <th>Location</th>
                    <th>Condition</th>
                    <th aria-label="Actions" />
                  </tr>
                </thead>
                <tbody>
                  {items.map((item) => (
                    <tr key={item.id}>
                      <td>
                        <div className="lab-member-academic">
                          <strong>{item.name}</strong>
                          <small>Added {formatDate(item.created_at)}</small>
                        </div>
                      </td>
                      <td>{item.category}</td>
                      <td>
                        <div className="lab-stock-cell">
                          <strong>{formatNumber(item.available_quantity)}</strong>
                          <span>of {formatNumber(item.total_quantity)} available</span>
                        </div>
                      </td>
                      <td>{item.storage_location || "—"}</td>
                      <td>{item.condition ? <Badge tone="neutral">{titleCase(item.condition)}</Badge> : "—"}</td>
                      <td>
                        <div className="lab-row-actions">
                          <AppButton variant="ghost" size="sm" onClick={() => openDetail(item)}>
                            View
                          </AppButton>
                          <AppButton
                            variant="ghost"
                            size="sm"
                            onClick={() => openEdit(item)}
                            loading={editLoading && formItem?.id === item.id}
                          >
                            <EditIcon className="h-3.5 w-3.5" />
                          </AppButton>
                          <AppButton
                            variant="ghost"
                            size="sm"
                            className="lab-danger-text"
                            onClick={() => setDeleteTarget(item)}
                          >
                            <TrashIcon className="h-3.5 w-3.5" />
                          </AppButton>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>

        {resource.data?.pagination && (
          <Pagination
            page={resource.data.pagination.page}
            totalPages={resource.data.pagination.total_pages}
            totalCount={resource.data.pagination.total_count}
            onPageChange={(page) => setFilters((current) => ({ ...current, page }))}
            loading={resource.loading}
          />
        )}
      </Panel>

      {selectedItem && (
        <InventoryDetail
          item={selectedItem}
          loading={loadingItem}
          error={itemError}
          onClose={() => {
            setSelectedItem(null);
            setItemError(null);
          }}
          onEdit={() => {
            setSelectedItem(null);
            openEdit(selectedItem);
          }}
        />
      )}

      {formOpen && (
        <InventoryForm
          key={formItem?.id || "new"}
          item={formItem}
          onClose={() => {
            setFormOpen(false);
            setFormItem(null);
          }}
          onSaved={(result) => {
            setNotice(result.message || "Inventory item saved.");
            resource.reload();
            statsResource.reload();
          }}
        />
      )}

      <ConfirmDialog
        open={Boolean(deleteTarget)}
        onClose={() => setDeleteTarget(null)}
        onConfirm={handleDelete}
        title="Remove this inventory item?"
        description={`“${deleteTarget?.name || ""}” will be removed from active inventory. Existing borrowing records are kept.`}
        confirmLabel="Remove item"
        destructive
        loading={deleting}
      />
    </>
  );
}
