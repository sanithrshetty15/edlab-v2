import { useState } from "react";
import { membersManagementApi } from "../../../api/services";
import { useApiResource } from "../../../hooks/useApiResource";
import MemberEnrollmentDialog from "./MemberEnrollmentDialog";
import { ConfirmDialog, Modal } from "../../../components/app/Modal";
import {
  Alert,
  AppButton,
  Avatar,
  Badge,
  DefinitionList,
  EmptyState,
  ErrorState,
  Field,
  LoadingState,
  PageHeader,
  Pagination,
  Panel,
  PasswordField,
  Select,
  StatusBadge,
  TextInput,
} from "../../../components/app/Ui";
import {
  EditIcon,
  ExternalIcon,
  KeyIcon,
  MailIcon,
  PlusIcon,
  SearchIcon,
  ShieldIcon,
  UserIcon,
  UsersIcon,
  XCircleIcon,
} from "../../../components/app/AppIcons";
import {
  formatDate,
  formatRelative,
  safeHttpUrl,
  titleCase,
} from "../../../utils/format";

const STATUS_OPTIONS = [
  { value: "", label: "All statuses" },
  { value: "active", label: "Active" },
  { value: "pending_enrollment", label: "Awaiting fingerprint" },
  { value: "inactive", label: "Inactive" },
  { value: "deactivated", label: "Deactivated" },
];

function EditMemberDialog({ member, onClose, onSaved }) {
  const [name, setName] = useState(member.name || "");
  const [usn, setUsn] = useState(member.usn || "");
  const [rollNumber, setRollNumber] = useState(member.roll_number || "");
  const [academicYear, setAcademicYear] = useState(member.academic_year || "");
  const [branch, setBranch] = useState(member.branch || "");
  const [section, setSection] = useState(member.section || "");
  const [phone, setPhone] = useState(member.phone || "");
  const [errors, setErrors] = useState({});
  const [formError, setFormError] = useState(null);
  const [saving, setSaving] = useState(false);

  const handleSubmit = async (event) => {
    event.preventDefault();
    if (saving) return;

    const nextErrors = {};
    if (!name.trim()) nextErrors.name = "Name is required.";
    if (!usn.trim()) nextErrors.usn = "USN is required.";
    if (phone && !/^\+?[1-9]\d{7,14}$/.test(phone.trim())) {
      nextErrors.phone = "Use an international format such as +919876543210.";
    }
    if (academicYear && (Number(academicYear) < 1 || Number(academicYear) > 8)) {
      nextErrors.academicYear = "Academic year must be between 1 and 8.";
    }
    setErrors(nextErrors);
    setFormError(null);
    if (Object.keys(nextErrors).length) return;

    const payload = {
      name: name.trim(),
      usn: usn.trim(),
      roll_number: rollNumber.trim(),
      academic_year: academicYear,
      branch: branch.trim(),
      section: section.trim(),
      phone: phone.trim(),
    };

    const unchanged = Object.entries(payload).every(([key, value]) => {
      const current = member[key];
      if (key === "academic_year") return String(current || "") === String(value || "");
      return String(current || "") === String(value || "");
    });
    if (unchanged) {
      setFormError(new Error("No fields have changed."));
      return;
    }

    setSaving(true);
    try {
      const result = await membersManagementApi.updateFields(member.id, payload);
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
      title="Edit official member fields"
      description="These fields are part of the official member record."
      size="lg"
      footer={
        <>
          <AppButton variant="ghost" onClick={onClose} disabled={saving}>
            Cancel
          </AppButton>
          <AppButton form="edit-member-form" type="submit" variant="primary" loading={saving}>
            Save fields
          </AppButton>
        </>
      }
    >
      <form id="edit-member-form" className="lab-form-stack" onSubmit={handleSubmit} noValidate>
        {formError && (
          <Alert tone="danger" title="Member could not be updated">
            {formError.message}
          </Alert>
        )}

        <div className="lab-form-grid">
          <Field label="Full name" htmlFor="member-name" error={errors.name} required>
            <TextInput
              id="member-name"
              value={name}
              onChange={(event) => setName(event.target.value)}
              disabled={saving}
            />
          </Field>
          <Field label="USN" htmlFor="member-usn" error={errors.usn} required>
            <TextInput
              id="member-usn"
              value={usn}
              onChange={(event) => setUsn(event.target.value)}
              disabled={saving}
            />
          </Field>
          <Field label="Roll number" htmlFor="member-roll">
            <TextInput
              id="member-roll"
              value={rollNumber}
              onChange={(event) => setRollNumber(event.target.value)}
              disabled={saving}
            />
          </Field>
          <Field label="Academic year" htmlFor="member-year" error={errors.academicYear}>
            <Select
              id="member-year"
              value={academicYear}
              onChange={(event) => setAcademicYear(event.target.value)}
              disabled={saving}
            >
              <option value="">Not set</option>
              {[1, 2, 3, 4, 5, 6, 7, 8].map((year) => (
                <option key={year} value={year}>Year {year}</option>
              ))}
            </Select>
          </Field>
          <Field label="Branch" htmlFor="member-branch">
            <TextInput
              id="member-branch"
              value={branch}
              onChange={(event) => setBranch(event.target.value)}
              disabled={saving}
            />
          </Field>
          <Field label="Section" htmlFor="member-section">
            <TextInput
              id="member-section"
              value={section}
              onChange={(event) => setSection(event.target.value)}
              disabled={saving}
            />
          </Field>
        </div>

        <Field
          label="Phone"
          htmlFor="member-phone"
          hint="Changing the phone updates the linked user record as well."
          error={errors.phone}
        >
          <TextInput
            id="member-phone"
            type="tel"
            value={phone}
            onChange={(event) => setPhone(event.target.value)}
            placeholder="+919876543210"
            disabled={saving}
          />
        </Field>
      </form>
    </Modal>
  );
}

function CreateMemberDialog({ onClose, onCreated }) {
  const [values, setValues] = useState({
    name: "",
    login_id: "",
    email: "",
    phone: "",
    usn: "",
    roll_number: "",
    academic_year: "1",
    branch: "",
    section: "",
    password: "",
    password_confirmation: "",
  });
  const [errors, setErrors] = useState({});
  const [formError, setFormError] = useState(null);
  const [saving, setSaving] = useState(false);

  const update = (key) => (event) => {
    setValues((current) => ({ ...current, [key]: event.target.value }));
  };

  const handleSubmit = async (event) => {
    event.preventDefault();
    if (saving) return;

    const nextErrors = {};
    if (!values.name.trim()) nextErrors.name = "Name is required.";
    if (!values.login_id.trim()) nextErrors.login_id = "Login ID is required.";
    else if (!/^[\w.-]+$/.test(values.login_id.trim())) {
      nextErrors.login_id = "Use letters, numbers, dots, or hyphens only.";
    }
    if (!values.email.trim()) nextErrors.email = "Email is required.";
    else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(values.email.trim())) {
      nextErrors.email = "Enter a valid email address.";
    }
    if (!values.phone.trim()) nextErrors.phone = "Phone is required.";
    else if (!/^\+?[1-9]\d{7,14}$/.test(values.phone.trim())) {
      nextErrors.phone = "Use an international format such as +919876543210.";
    }
    if (!values.usn.trim()) nextErrors.usn = "USN is required.";
    else if (!/^[A-Z0-9]{6,15}$/i.test(values.usn.trim())) {
      nextErrors.usn = "USN must be 6-15 letters and digits.";
    }
    if (!values.branch.trim()) nextErrors.branch = "Branch is required.";
    if (!values.password) nextErrors.password = "Temporary password is required.";
    else if (values.password.length < 8) {
      nextErrors.password = "Use at least 8 characters.";
    }
    if (values.password !== values.password_confirmation) {
      nextErrors.password_confirmation = "Passwords do not match.";
    }

    setErrors(nextErrors);
    setFormError(null);
    if (Object.keys(nextErrors).length) return;

    setSaving(true);
    try {
      const result = await membersManagementApi.create({
        name: values.name.trim(),
        login_id: values.login_id.trim(),
        email: values.email.trim(),
        phone: values.phone.trim(),
        usn: values.usn.trim().toUpperCase(),
        roll_number: values.roll_number.trim(),
        academic_year: values.academic_year,
        branch: values.branch.trim(),
        section: values.section.trim(),
        password: values.password,
        password_confirmation: values.password_confirmation,
      });
      onCreated(result);
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
      title="Add member · Step 1 of 2"
      description="Create the official member record and sign-in credentials. The account stays inactive until fingerprint enrollment is complete."
      size="lg"
      footer={
        <>
          <AppButton variant="ghost" onClick={onClose} disabled={saving}>
            Cancel
          </AppButton>
          <AppButton form="create-member-form" type="submit" variant="primary" loading={saving}>
            Save details & continue
          </AppButton>
        </>
      }
    >
      <form id="create-member-form" className="lab-form-stack" onSubmit={handleSubmit} noValidate>
        {formError && (
          <Alert tone="danger" title="Member could not be created">
            {formError.message}
          </Alert>
        )}

        <div className="lab-form-grid">
          <Field label="Full name" htmlFor="new-member-name" error={errors.name} required>
            <TextInput
              id="new-member-name"
              value={values.name}
              onChange={update("name")}
              autoComplete="name"
              disabled={saving}
            />
          </Field>
          <Field label="Login ID" htmlFor="new-member-login" error={errors.login_id} required>
            <TextInput
              id="new-member-login"
              value={values.login_id}
              onChange={update("login_id")}
              autoComplete="off"
              spellCheck="false"
              disabled={saving}
            />
          </Field>
          <Field label="Email" htmlFor="new-member-email" error={errors.email} required>
            <TextInput
              id="new-member-email"
              type="email"
              value={values.email}
              onChange={update("email")}
              autoComplete="email"
              disabled={saving}
            />
          </Field>
          <Field label="Phone" htmlFor="new-member-phone" error={errors.phone} required>
            <TextInput
              id="new-member-phone"
              type="tel"
              value={values.phone}
              onChange={update("phone")}
              placeholder="+919876543210"
              autoComplete="tel"
              disabled={saving}
            />
          </Field>
          <Field label="USN" htmlFor="new-member-usn" error={errors.usn} required>
            <TextInput
              id="new-member-usn"
              value={values.usn}
              onChange={update("usn")}
              placeholder="4AL25CS000"
              autoCapitalize="characters"
              spellCheck="false"
              disabled={saving}
            />
          </Field>
          <Field label="Roll number" htmlFor="new-member-roll">
            <TextInput
              id="new-member-roll"
              value={values.roll_number}
              onChange={update("roll_number")}
              disabled={saving}
            />
          </Field>
          <Field label="Academic year" htmlFor="new-member-year" required>
            <Select
              id="new-member-year"
              value={values.academic_year}
              onChange={update("academic_year")}
              disabled={saving}
            >
              {[1, 2, 3, 4, 5, 6, 7, 8].map((year) => (
                <option key={year} value={year}>Year {year}</option>
              ))}
            </Select>
          </Field>
          <Field label="Branch" htmlFor="new-member-branch" error={errors.branch} required>
            <TextInput
              id="new-member-branch"
              value={values.branch}
              onChange={update("branch")}
              disabled={saving}
            />
          </Field>
          <Field label="Section" htmlFor="new-member-section">
            <TextInput
              id="new-member-section"
              value={values.section}
              onChange={update("section")}
              disabled={saving}
            />
          </Field>
          <PasswordField
            id="new-member-password"
            label="Temporary password"
            value={values.password}
            onChange={update("password")}
            autoComplete="new-password"
            error={errors.password}
            required
            disabled={saving}
          />
          <PasswordField
            id="new-member-password-confirmation"
            label="Confirm password"
            value={values.password_confirmation}
            onChange={update("password_confirmation")}
            autoComplete="new-password"
            error={errors.password_confirmation}
            required
            disabled={saving}
          />
        </div>
      </form>
    </Modal>
  );
}

function MemberDetail({
  member,
  enrollment,
  loading,
  error,
  onClose,
  onEdit,
  onEnroll,
  onForceReset,
  onDeactivate,
  onStatusChange,
  statusBusy,
}) {
  return (
    <Modal
      open
      onClose={onClose}
      title={member?.name || "Member record"}
      description={member ? `${member.login_id || "No login ID"} · ${member.email || "No email"}` : undefined}
      size="lg"
      footer={
        <>
          <AppButton variant="ghost" onClick={onClose}>Close</AppButton>
          <AppButton variant="outline" onClick={onForceReset} disabled={!member}>
            <KeyIcon className="h-4 w-4" />
            Force password reset
          </AppButton>
          <AppButton variant="outline" onClick={onEdit} disabled={!member}>
            <EditIcon className="h-4 w-4" />
            Edit fields
          </AppButton>
          {member?.status === "pending_enrollment" && (
            <AppButton variant="primary" onClick={onEnroll}>
              <ShieldIcon className="h-4 w-4" />
              Fingerprint enrollment
            </AppButton>
          )}
        </>
      }
    >
      {loading ? (
        <LoadingState rows={8} />
      ) : error ? (
        <ErrorState error={error} title="Member could not be loaded" />
      ) : member ? (
        <div className="lab-member-detail">
          <div className="lab-member-detail__identity">
            <Avatar name={member.name} size="lg" />
            <div>
              <div className="lab-member-detail__badges">
                <StatusBadge status={member.status}>
                  {member.status === "pending_enrollment" ? "Awaiting fingerprint" : undefined}
                </StatusBadge>
                {member.status === "pending_enrollment" && (
                  <Badge tone="warning">
                    {enrollment?.status
                      ? `Enrollment: ${titleCase(enrollment.status)}`
                      : "Enrollment not started"}
                  </Badge>
                )}
                {member.must_change_password && <Badge tone="warning">Password reset required</Badge>}
                {member.phone_verified && <Badge tone="success">Phone verified</Badge>}
              </div>
              <p className="lab-muted lab-small">
                Member since {formatDate(member.created_at)}
              </p>
            </div>
          </div>

          <div className="lab-member-detail__section">
            <h3>Status</h3>
            {member.status === "pending_enrollment" ? (
              <Alert tone="warning" title="Fingerprint enrollment required">
                This account cannot be activated until the member completes two matching
                fingerprint scans on the enrollment reader.
              </Alert>
            ) : (
              <div className="lab-member-detail__status-actions">
                {["active", "inactive", "deactivated"].map((status) => (
                  <AppButton
                    key={status}
                    variant={member.status === status ? "primary" : "outline"}
                    size="sm"
                    disabled={member.status === status || Boolean(statusBusy)}
                    loading={statusBusy === status}
                    onClick={() => onStatusChange(status)}
                  >
                    {titleCase(status)}
                  </AppButton>
                ))}
              </div>
            )}
          </div>

          <DefinitionList
            items={[
              { label: "Login ID", value: member.login_id },
              { label: "USN", value: member.usn },
              { label: "Roll number", value: member.roll_number },
              { label: "Academic year", value: member.academic_year ? `Year ${member.academic_year}` : "—" },
              { label: "Branch", value: member.branch },
              { label: "Section", value: member.section },
              { label: "Email", value: member.email },
              { label: "Phone", value: member.phone || "—" },
            ]}
          />

          <div className="lab-member-detail__section">
            <h3>Account</h3>
            <ul className="lab-contact-list">
              <li>
                <ShieldIcon className="h-4 w-4" />
                <div>
                  <span>User account</span>
                  <strong>
                    {member.user?.active ? "Active" : "Inactive"} · {titleCase(member.user?.role)}
                  </strong>
                </div>
              </li>
              <li>
                <MailIcon className="h-4 w-4" />
                <div>
                  <span>Account email</span>
                  <strong>{member.user?.email || member.email || "—"}</strong>
                </div>
              </li>
              <li>
                <UserIcon className="h-4 w-4" />
                <div>
                  <span>Last updated</span>
                  <strong>{member.updated_at ? formatRelative(member.updated_at) : "—"}</strong>
                </div>
              </li>
            </ul>
          </div>

          <div className="lab-member-detail__section">
            <h3>Profile</h3>
            <p className="lab-profile-bio">
              {member.bio || "No bio provided."}
            </p>
            {Array.isArray(member.interests) && member.interests.length > 0 && (
              <div className="lab-interest-chips">
                {member.interests.map((interest) => (
                  <Badge key={interest} tone="neutral">{interest}</Badge>
                ))}
              </div>
            )}
            <div className="lab-social-links">
              {[
                ["GitHub", member.github_url],
                ["LinkedIn", member.linkedin_url],
                ["Portfolio", member.portfolio_url],
              ].map(([label, value]) => {
                const url = safeHttpUrl(value);
                if (!url) return null;
                return (
                  <a key={label} href={url} target="_blank" rel="noreferrer noopener" className="lab-social-link">
                    {label}
                    <ExternalIcon className="h-3.5 w-3.5" />
                  </a>
                );
              })}
            </div>
          </div>

          {member.status !== "pending_enrollment" && (
            <div className="lab-member-detail__danger">
              <div>
                <h3>Deactivate member</h3>
                <p>Revokes access and marks the member record as deactivated.</p>
              </div>
              <AppButton variant="danger" onClick={onDeactivate}>
                <XCircleIcon className="h-4 w-4" />
                Deactivate
              </AppButton>
            </div>
          )}
        </div>
      ) : null}
    </Modal>
  );
}

export default function MembersPage() {
  const [searchInput, setSearchInput] = useState("");
  const [filters, setFilters] = useState({ search: "", status: "", year: "", page: 1 });
  const [selectedId, setSelectedId] = useState(null);
  const [detail, setDetail] = useState(null);
  const [detailEnrollment, setDetailEnrollment] = useState(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detailError, setDetailError] = useState(null);
  const [editing, setEditing] = useState(null);
  const [creating, setCreating] = useState(false);
  const [enrolling, setEnrolling] = useState(null);
  const [confirmAction, setConfirmAction] = useState(null);
  const [actionBusy, setActionBusy] = useState(null);
  const [notice, setNotice] = useState(null);
  const [actionError, setActionError] = useState(null);

  const query = {
    page: filters.page,
    per_page: 20,
    search: filters.search || undefined,
    status: filters.status || undefined,
    academic_year: filters.year || undefined,
  };

  const resource = useApiResource(
    () => membersManagementApi.list(query),
    `members-${JSON.stringify(query)}`,
  );

  const members = resource.data?.members ?? [];

  const loadMember = async (id) => {
    setSelectedId(id);
    setDetail(null);
    setDetailEnrollment(null);
    setDetailLoading(true);
    setDetailError(null);
    try {
      const result = await membersManagementApi.get(id);
      setDetail(result.member);
      setDetailEnrollment(result.enrollment || null);
    } catch (error) {
      setDetailError(error);
    } finally {
      setDetailLoading(false);
    }
  };

  const refreshMember = async () => {
    if (!selectedId) return;
    const result = await membersManagementApi.get(selectedId);
    setDetail(result.member);
    setDetailEnrollment(result.enrollment || null);
  };

  const handleStatusChange = async (status) => {
    if (!detail || actionBusy) return;
    setActionBusy(status);
    setActionError(null);
    try {
      const result = await membersManagementApi.updateStatus(detail.id, status);
      setDetail((current) => ({ ...current, ...result.member }));
      setNotice(`Member status changed to ${status}.`);
      resource.reload();
    } catch (error) {
      setActionError(error);
    } finally {
      setActionBusy(null);
    }
  };

  const handleConfirmAction = async () => {
    if (!confirmAction) return;
    setActionBusy(confirmAction.type);
    setActionError(null);
    try {
      if (confirmAction.type === "reset") {
        const result = await membersManagementApi.forcePasswordReset(confirmAction.member.id);
        setDetail((current) => current ? { ...current, must_change_password: true } : current);
        setNotice(result.message || "Password reset forced.");
      } else if (confirmAction.type === "deactivate") {
        await membersManagementApi.deactivate(confirmAction.member.id);
        setNotice("Member account deactivated.");
        setDetail(null);
        setSelectedId(null);
        resource.reload();
      }
      setConfirmAction(null);
    } catch (error) {
      setActionError(error);
    } finally {
      setActionBusy(null);
    }
  };

  return (
    <>
      <PageHeader
        eyebrow="Admin console"
        title="Members"
        description="Search, review and manage official member records."
        actions={
          <AppButton variant="primary" onClick={() => setCreating(true)}>
            <PlusIcon className="h-4 w-4" />
            Add member
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
              placeholder="Search by member name"
              aria-label="Search members by name"
            />
          </div>
          <div className="lab-toolbar__right">
            <Select
              value={filters.status}
              onChange={(event) =>
                setFilters((current) => ({ ...current, status: event.target.value, page: 1 }))
              }
              aria-label="Filter by status"
              className="lab-select--compact"
            >
              {STATUS_OPTIONS.map((option) => (
                <option key={option.value} value={option.value}>{option.label}</option>
              ))}
            </Select>
            <Select
              value={filters.year}
              onChange={(event) =>
                setFilters((current) => ({ ...current, year: event.target.value, page: 1 }))
              }
              aria-label="Filter by academic year"
              className="lab-select--compact"
            >
              <option value="">All years</option>
              {[1, 2, 3, 4, 5, 6, 7, 8].map((year) => (
                <option key={year} value={year}>Year {year}</option>
              ))}
            </Select>
            <AppButton type="submit" variant="outline" size="sm">
              Apply
            </AppButton>
          </div>
        </form>

        <div className="lab-list-panel__body">
          {resource.loading ? (
            <LoadingState rows={8} />
          ) : resource.error ? (
            <ErrorState error={resource.error} onRetry={resource.reload} title="Member records are unavailable" />
          ) : members.length === 0 ? (
            <EmptyState
              icon={UsersIcon}
              title="No members found"
              description="Adjust the search or filters to find a member record."
            />
          ) : (
            <div className="lab-table-wrap">
              <table className="lab-table lab-table--members">
                <thead>
                  <tr>
                    <th>Member</th>
                    <th>Academic</th>
                    <th>Status</th>
                    <th>Account</th>
                    <th>Joined</th>
                    <th aria-label="Actions" />
                  </tr>
                </thead>
                <tbody>
                  {members.map((member) => (
                    <tr key={member.id}>
                      <td>
                        <div className="lab-member-cell">
                          <Avatar name={member.name} size="sm" />
                          <div>
                            <strong>{member.name}</strong>
                            <span>{member.login_id || member.email}</span>
                          </div>
                        </div>
                      </td>
                      <td>
                        <div className="lab-member-academic">
                          <span>{member.usn || "No USN"}</span>
                          <small>
                            {member.academic_year ? `Year ${member.academic_year}` : "Year —"}
                            {member.branch ? ` · ${member.branch}` : ""}
                          </small>
                        </div>
                      </td>
                      <td>
                        <StatusBadge status={member.status}>
                          {member.status === "pending_enrollment"
                            ? "Awaiting fingerprint"
                            : undefined}
                        </StatusBadge>
                      </td>
                      <td>
                        <div className="lab-member-account">
                          {member.status === "pending_enrollment" ? (
                            <Badge tone="warning">
                              {member.enrollment_status
                                ? titleCase(member.enrollment_status.replaceAll("_", " "))
                                : "Enrollment not started"}
                            </Badge>
                          ) : (
                            <Badge tone={member.must_change_password ? "warning" : "success"}>
                              {member.must_change_password ? "Reset pending" : "Password set"}
                            </Badge>
                          )}
                          {member.phone_verified && <Badge tone="info">Phone verified</Badge>}
                        </div>
                      </td>
                      <td>{formatDate(member.created_at)}</td>
                      <td>
                        <AppButton variant="ghost" size="sm" onClick={() => loadMember(member.id)}>
                          View
                        </AppButton>
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

      {(selectedId || detailLoading) && (
        <MemberDetail
          member={detail}
          enrollment={detailEnrollment}
          loading={detailLoading}
          error={detailError}
          statusBusy={actionBusy}
          onClose={() => {
            setSelectedId(null);
            setDetail(null);
            setDetailEnrollment(null);
            setDetailError(null);
          }}
          onEdit={() => setEditing(detail)}
          onEnroll={() => setEnrolling({ member: detail, enrollment: detailEnrollment })}
          onForceReset={() => setConfirmAction({ type: "reset", member: detail })}
          onDeactivate={() => setConfirmAction({ type: "deactivate", member: detail })}
          onStatusChange={handleStatusChange}
        />
      )}

      {creating && (
        <CreateMemberDialog
          onClose={() => setCreating(false)}
          onCreated={(result) => {
            setCreating(false);
            setNotice(
              result.message ||
                "Member details saved. Continue with fingerprint enrollment.",
            );
            setFilters((current) => ({ ...current, page: 1 }));
            resource.reload();
            setEnrolling({
              member: result.member,
              enrollment: result.enrollment || null,
            });
          }}
        />
      )}

      {enrolling && (
        <MemberEnrollmentDialog
          member={enrolling.member}
          initialEnrollment={enrolling.enrollment}
          onClose={() => setEnrolling(null)}
          onFinalized={(result) => {
            setNotice(result.message || "Fingerprint enrollment complete. Member activated.");
            resource.reload();
            if (selectedId && result.member?.id === selectedId) {
              setDetail(result.member);
              setDetailEnrollment(result.enrollment || null);
            }
          }}
          onCancelled={(result) => {
            setNotice(result.message || "Registration cancelled.");
            setEnrolling(null);
            setSelectedId(null);
            setDetail(null);
            setDetailEnrollment(null);
            resource.reload();
          }}
        />
      )}

      {editing && (
        <EditMemberDialog
          key={editing.id}
          member={editing}
          onClose={() => setEditing(null)}
          onSaved={async (result) => {
            setNotice(result.message || "Member fields updated.");
            resource.reload();
            try {
              await refreshMember();
            } catch (error) {
              setActionError(error);
            }
          }}
        />
      )}

      <ConfirmDialog
        open={Boolean(confirmAction)}
        onClose={() => setConfirmAction(null)}
        onConfirm={handleConfirmAction}
        title={
          confirmAction?.type === "reset"
            ? "Force a password reset?"
            : "Deactivate this member?"
        }
        description={
          confirmAction?.type === "reset"
            ? `${confirmAction?.member?.name || "This member"} will be required to set a new password on the next sign-in.`
            : `${confirmAction?.member?.name || "This member"} will lose access and the record will be marked deactivated.`
        }
        confirmLabel={confirmAction?.type === "reset" ? "Force reset" : "Deactivate member"}
        destructive={confirmAction?.type === "deactivate"}
        loading={Boolean(actionBusy)}
      />
    </>
  );
}
