import { useState } from "react";
import { membersManagementApi } from "../../../api/services";
import { useApiResource } from "../../../hooks/useApiResource";
import MemberEnrollmentDialog from "./MemberEnrollmentDialog";
import MemberRegistrationProgress from "./MemberRegistrationProgress";
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
  Select,
  SegmentedControl,
  StatusBadge,
  TextInput,
} from "../../../components/app/Ui";
import {
  EditIcon,
  ArrowUpIcon,
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

const IDENTIFIER_TYPE_OPTIONS = [
  { value: "usn", label: "USN" },
  { value: "roll_number", label: "Roll Number" },
];

const REGISTRATION_ACADEMIC_YEARS = [1, 2, 3, 4];

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
    if (!usn.trim() && !rollNumber.trim()) {
      nextErrors.usn = "Enter a USN or a Roll Number.";
    }
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
          <Field
            label="USN"
            htmlFor="member-usn"
            error={errors.usn}
            hint="Provide either a USN or a Roll Number."
          >
            <TextInput
              id="member-usn"
              value={usn}
              onChange={(event) => setUsn(event.target.value)}
              disabled={saving}
            />
          </Field>
          <Field
            label="Roll number"
            htmlFor="member-roll"
            error={errors.rollNumber}
          >
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
    academic_year: "1",
    identifier_type: "",
    identifier: "",
    branch: "",
    section: "",
    email: "",
    phone: "",
  });
  const [errors, setErrors] = useState({});
  const [formError, setFormError] = useState(null);
  const [saving, setSaving] = useState(false);

  const identifierLabel =
    values.identifier_type === "usn"
      ? "USN"
      : values.identifier_type === "roll_number"
        ? "Roll Number"
        : "Identifier";
  const loginIdSource =
    values.identifier_type === "usn"
      ? "USN"
      : values.identifier_type === "roll_number"
        ? "Roll Number"
        : "selected identifier";

  const update = (key) => (event) => {
    setValues((current) => ({ ...current, [key]: event.target.value }));
  };

  const updateIdentifierType = (identifierType) => {
    setValues((current) => {
      if (current.identifier_type === identifierType) return current;
      return { ...current, identifier_type: identifierType, identifier: "" };
    });
    setErrors((current) => ({
      ...current,
      identifierType: undefined,
      identifier: undefined,
    }));
  };

  const handleSubmit = async (event) => {
    event.preventDefault();
    if (saving) return;

    const identifier = values.identifier.trim();
    const nextErrors = {};
    if (!values.name.trim()) nextErrors.name = "Name is required.";
    if (!values.identifier_type) {
      nextErrors.identifierType = "Select USN or Roll Number.";
    }
    if (!identifier) {
      nextErrors.identifier = `${identifierLabel} is required.`;
    } else if (values.identifier_type === "usn" && !/^[A-Z0-9]{6,15}$/i.test(identifier)) {
      nextErrors.identifier = "USN must be 6-15 letters and digits.";
    } else if (values.identifier_type === "roll_number" && !/^[\w-]+$/.test(identifier)) {
      nextErrors.identifier = "Roll Number contains invalid characters.";
    }
    if (!values.branch.trim()) nextErrors.branch = "Branch is required.";
    if (!values.section.trim()) nextErrors.section = "Section is required.";
    if (values.email.trim() && !/^[^\s@]+@gmail\.com$/i.test(values.email.trim())) {
      nextErrors.email = "Enter a valid Gmail address ending in @gmail.com.";
    }
    if (values.phone.trim() && !/^\+?[1-9]\d{7,14}$/.test(values.phone.trim())) {
      nextErrors.phone = "Use an international format such as +919876543210.";
    }

    setErrors(nextErrors);
    setFormError(null);
    if (Object.keys(nextErrors).length) return;

    const isUSN = values.identifier_type === "usn";
    setSaving(true);
    try {
      const result = await membersManagementApi.create({
        name: values.name.trim(),
        academic_year: Number(values.academic_year),
        identifier_type: values.identifier_type,
        usn: isUSN ? identifier.toUpperCase() : "",
        roll_number: isUSN ? "" : identifier,
        branch: values.branch.trim(),
        section: values.section.trim(),
        email: values.email.trim(),
        phone: values.phone.trim(),
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
      title="Add member · Member information"
      description="Start with the details needed to create a member record."
      size="lg"
      footer={
        <>
          <AppButton variant="ghost" onClick={onClose} disabled={saving}>
            Cancel
          </AppButton>
          <AppButton form="create-member-form" type="submit" variant="primary" loading={saving}>
            Continue to fingerprint
          </AppButton>
        </>
      }
    >
      <form id="create-member-form" className="lab-form-stack" onSubmit={handleSubmit} noValidate>
        <MemberRegistrationProgress current={1} />

        {formError && (
          <Alert tone="danger" title="Member could not be created">
            {formError.message}
          </Alert>
        )}

        <Alert tone="info" title="What happens next">
          A login ID is created from the {loginIdSource}. The account stays inactive until
          fingerprint enrollment is complete, and the member can set a password using Forgot
          password after activation.
        </Alert>

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
          <Field label="Academic year" htmlFor="new-member-year" required>
            <Select
              id="new-member-year"
              value={values.academic_year}
              onChange={update("academic_year")}
              disabled={saving}
            >
              {REGISTRATION_ACADEMIC_YEARS.map((year) => (
                <option key={year} value={year}>Year {year}</option>
              ))}
            </Select>
          </Field>
          <Field
            label="Identifier type"
            error={errors.identifierType}
            hint="Choose whether this member signs in with a USN or a Roll Number."
            required
            className="lab-span-2"
          >
            <SegmentedControl
              options={IDENTIFIER_TYPE_OPTIONS}
              value={values.identifier_type}
              onChange={updateIdentifierType}
              label="Member identifier type"
              disabled={saving}
            />
          </Field>
          <Field
            label={identifierLabel}
            htmlFor="new-member-identifier"
            error={errors.identifier}
            hint={
              values.identifier_type === "usn"
                ? "Enter the member's university seat number."
                : values.identifier_type === "roll_number"
                  ? "Enter the member's college roll number."
                  : "Select an identifier type above to enter the value."
            }
            required
          >
            <TextInput
              id="new-member-identifier"
              value={values.identifier}
              onChange={update("identifier")}
              placeholder={
                values.identifier_type === "usn"
                  ? "4AL25CS000"
                  : values.identifier_type === "roll_number"
                    ? "Roll number"
                    : "Select an identifier type first"
              }
              autoCapitalize={values.identifier_type === "usn" ? "characters" : "none"}
              spellCheck="false"
              disabled={saving || !values.identifier_type}
            />
          </Field>
          <Field label="Branch" htmlFor="new-member-branch" error={errors.branch} required>
            <TextInput
              id="new-member-branch"
              value={values.branch}
              onChange={update("branch")}
              placeholder="CSE"
              disabled={saving}
            />
          </Field>
          <Field label="Section" htmlFor="new-member-section" error={errors.section} required>
            <TextInput
              id="new-member-section"
              value={values.section}
              onChange={update("section")}
              placeholder="A"
              disabled={saving}
            />
          </Field>
          <Field
            label="Gmail address"
            htmlFor="new-member-email"
            error={errors.email}
            hint="Optional. Confirmation and account recovery emails are sent here."
          >
            <TextInput
              id="new-member-email"
              type="email"
              value={values.email}
              onChange={update("email")}
              autoComplete="email"
              placeholder="member@gmail.com"
              disabled={saving}
            />
          </Field>
          <Field
            label="Phone"
            htmlFor="new-member-phone"
            error={errors.phone}
            hint="Optional. Use an international format."
          >
            <TextInput
              id="new-member-phone"
              type="tel"
              value={values.phone}
              onChange={update("phone")}
              autoComplete="tel"
              placeholder="+919876543210"
              disabled={saving}
            />
          </Field>
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
  const [selectedIds, setSelectedIds] = useState(new Set());
  const [detail, setDetail] = useState(null);
  const [detailEnrollment, setDetailEnrollment] = useState(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detailError, setDetailError] = useState(null);
  const [editing, setEditing] = useState(null);
  const [creating, setCreating] = useState(false);
  const [enrolling, setEnrolling] = useState(null);
  const [confirmAction, setConfirmAction] = useState(null);
  const [actionBusy, setActionBusy] = useState(null);
  const [otpAction, setOtpAction] = useState(null);
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

  const handlePromoteSelected = async () => {
    if (selectedIds.size === 0) return;
    setActionBusy("promote_selected");
    setActionError(null);
    try {
      const memberIds = Array.from(selectedIds);
      for (const id of memberIds) {
        await membersManagementApi.updateStatus(id, "active");
      }
      setNotice(`Promoted ${memberIds.length} member(s) to active.`);
      resource.reload();
    } catch (error) {
      setActionError(error);
    } finally {
      setActionBusy(null);
      setSelectedIds(new Set());
    }
  };

  const handlePromoteAllActive = async () => {
    setActionBusy("promote_all_active");
    setActionError(null);
    try {
      const result = await membersManagementApi.listMembers({
        status: "inactive",
        page: 1,
        per_page: 1000,
      });
      const inactiveMembers = result.members ?? [];
      const memberIds = inactiveMembers.map((m) => m.id);
      if (memberIds.length === 0) {
        setNotice("No inactive members to promote.");
        return;
      }
      for (const id of memberIds) {
        await membersManagementApi.updateStatus(id, "active");
      }
      setNotice(`Promoted ${memberIds.length} member(s) to active.`);
      resource.reload();
    } catch (error) {
      setActionError(error);
    } finally {
      setActionBusy(null);
    }
  };

  const handleOtpAction = async (type, memberId) => {
    setOtpAction({ type, memberId });
  };

  const handleOtpConfirm = async (otpCode) => {
    if (!otpAction) return;
    setActionBusy(otpAction.type);
    setActionError(null);
    try {
      if (otpAction.type === "delete") {
        await membersManagementApi.deactivate(otpAction.member.id);
        setNotice("Member account deactivated successfully.");
        resource.reload();
      } else if (otpAction.type === "promote") {
        await membersManagementApi.updateStatus(otpAction.member.id, "active");
        setNotice("Member promoted to active successfully.");
        resource.reload();
      }
      setOtpAction(null);
    } catch (error) {
      setActionError(error);
    } finally {
      setOtpAction(null);
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
            <AppButton variant="primary" onClick={handlePromoteSelected} disabled={selectedIds.size === 0 || actionBusy}>
              <ArrowUpIcon className="h-3.5 w-3.5 me-2" />
              Promote selected
            </AppButton>
            <AppButton variant="primary" onClick={handlePromoteAllActive} disabled={actionBusy}>
              <ArrowUpIcon className="h-3.5 w-3.5 me-2" />
              Promote all active
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
                    <th>
                      <label className="lab-table-selector">
                        <input
                          type="checkbox"
                          checked={selectedIds.size === members.length && members.length > 0}
                          onChange={() => {
                            if (selectedIds.size === members.length && members.length > 0) {
                              setSelectedIds(new Set());
                            } else {
                              setSelectedIds(new Set(members.map((m) => m.id)));
                            }
                          }}
                        />
                        <span className="lab-checkbox-icon" aria-hidden="true" />
                      </label>
                    </th>
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
                          <label className="lab-table-selector">
                            <input
                              type="checkbox"
                              checked={selectedIds.has(member.id)}
                              onChange={() => {
                                if (selectedIds.has(member.id)) {
                                  setSelectedIds(new Set([...selectedIds].filter((id) => id !== member.id)));
                                } else {
                                  setSelectedIds(new Set([...selectedIds, member.id]));
                                }
                              }}
                            />
                            <span className="lab-checkbox-icon" aria-hidden="true" />
                          </label>
                          <Avatar name={member.name} size="sm" />
                          <div>
                            <strong>{member.name}</strong>
                            <span>{member.login_id || member.email}</span>
                          </div>
                        </div>
                      </td>
                      <td>
                        <div className="lab-member-academic">
                          <span>
                            {member.usn ||
                              (member.roll_number ? `Roll ${member.roll_number}` : "No identifier")}
                          </span>
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
