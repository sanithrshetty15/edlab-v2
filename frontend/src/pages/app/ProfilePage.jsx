import { useMemo, useState } from "react";
import { useAuth } from "../../auth/context";
import { attendanceApi, membersApi } from "../../api/services";
import { useApiResource } from "../../hooks/useApiResource";
import { AppLink } from "../../components/app/Link";
import { Modal } from "../../components/app/Modal";
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
  Panel,
  StatCard,
  StatusBadge,
  TextArea,
  TextInput,
} from "../../components/app/Ui";
import {
  CalendarIcon,
  ClockIcon,
  EditIcon,
  ExternalIcon,
  LockIcon,
  MailIcon,
  ShieldIcon,
  UserIcon,
} from "../../components/app/AppIcons";
import {
  formatClockTime,
  formatDate,
  formatNumber,
  formatRelative,
  safeHttpUrl,
  titleCase,
} from "../../utils/format";

function SocialLink({ label, value }) {
  const url = safeHttpUrl(value);
  if (!url) return null;
  return (
    <a className="lab-social-link" href={url} target="_blank" rel="noreferrer noopener">
      <span>{label}</span>
      <ExternalIcon className="h-3.5 w-3.5" />
    </a>
  );
}

function EditProfileDialog({ open, onClose, member, onUpdated }) {
  const [githubUrl, setGithubUrl] = useState(member?.github_url || "");
  const [linkedinUrl, setLinkedinUrl] = useState(member?.linkedin_url || "");
  const [portfolioUrl, setPortfolioUrl] = useState(member?.portfolio_url || "");
  const [bio, setBio] = useState(member?.bio || "");
  const [interests, setInterests] = useState((member?.interests || []).join(", "));
  const [errors, setErrors] = useState({});
  const [formError, setFormError] = useState(null);
  const [saving, setSaving] = useState(false);

  const validateUrl = (value, label) => {
    if (!value.trim()) return null;
    const url = safeHttpUrl(value.trim());
    if (!url) return `${label} must be a valid http or https URL.`;
    return null;
  };

  const handleSubmit = async (event) => {
    event.preventDefault();
    if (saving) return;

    const nextErrors = {
      github: validateUrl(githubUrl, "GitHub"),
      linkedin: validateUrl(linkedinUrl, "LinkedIn"),
      portfolio: validateUrl(portfolioUrl, "Portfolio"),
    };
    Object.keys(nextErrors).forEach((key) => {
      if (!nextErrors[key]) delete nextErrors[key];
    });

    if (bio.length > 2000) nextErrors.bio = "Keep your bio under 2,000 characters.";
    if (interests.length > 400) nextErrors.interests = "Keep interests under 400 characters.";

    setErrors(nextErrors);
    setFormError(null);
    if (Object.keys(nextErrors).length) return;

    setSaving(true);
    try {
      const result = await membersApi.updateProfile({
        github_url: githubUrl.trim() || null,
        linkedin_url: linkedinUrl.trim() || null,
        portfolio_url: portfolioUrl.trim() || null,
        bio: bio.trim() || null,
        interests: interests
          .split(",")
          .map((item) => item.trim())
          .filter(Boolean),
      });
      onUpdated(result.member);
      onClose();
    } catch (error) {
      setFormError(error);
    } finally {
      setSaving(false);
    }
  };

  const socialFields = [
    { id: "profile-github", label: "GitHub URL", value: githubUrl, setter: setGithubUrl, error: errors.github, placeholder: "https://github.com/username" },
    { id: "profile-linkedin", label: "LinkedIn URL", value: linkedinUrl, setter: setLinkedinUrl, error: errors.linkedin, placeholder: "https://linkedin.com/in/username" },
    { id: "profile-portfolio", label: "Portfolio URL", value: portfolioUrl, setter: setPortfolioUrl, error: errors.portfolio, placeholder: "https://your-site.dev" },
  ];

  return (
    <Modal
      open={open}
      onClose={onClose}
      title="Edit profile"
      description="These details are shown to the lab team and on your member profile."
      size="lg"
      footer={
        <>
          <AppButton variant="ghost" onClick={onClose} disabled={saving}>
            Cancel
          </AppButton>
          <AppButton form="edit-profile-form" type="submit" variant="primary" loading={saving}>
            Save changes
          </AppButton>
        </>
      }
    >
      <form id="edit-profile-form" className="lab-form-stack" onSubmit={handleSubmit} noValidate>
        {formError && (
          <Alert tone="danger" title="Profile could not be updated">
            <p>{formError.message}</p>
            {formError.details?.length > 0 && (
              <ul className="lab-alert__list">
                {formError.details.map((detail) => (
                  <li key={detail}>{detail}</li>
                ))}
              </ul>
            )}
          </Alert>
        )}

        <div className="lab-form-grid">
          {socialFields.map((field) => (
            <Field key={field.id} label={field.label} htmlFor={field.id} error={field.error}>
              <TextInput
                id={field.id}
                type="url"
                value={field.value}
                placeholder={field.placeholder}
                onChange={(event) => field.setter(event.target.value)}
                disabled={saving}
              />
            </Field>
          ))}
        </div>

        <Field
          label="Bio"
          htmlFor="profile-bio"
          hint={`${bio.length}/2000 characters`}
          error={errors.bio}
        >
          <TextArea
            id="profile-bio"
            rows={5}
            value={bio}
            placeholder="A short introduction about your work and interests."
            onChange={(event) => setBio(event.target.value)}
            disabled={saving}
          />
        </Field>

        <Field
          label="Interests"
          htmlFor="profile-interests"
          hint="Separate interests with commas, for example: robotics, embedded systems, design."
          error={errors.interests}
        >
          <TextInput
            id="profile-interests"
            value={interests}
            onChange={(event) => setInterests(event.target.value)}
            disabled={saving}
          />
        </Field>
      </form>
    </Modal>
  );
}

function AttendanceHistory({ resource }) {
  if (resource.loading) return <LoadingState rows={4} compact />;
  if (resource.error) {
    return <ErrorState error={resource.error} onRetry={resource.reload} compact />;
  }

  const records = resource.data?.attendance ?? [];
  if (!records.length) {
    return (
      <EmptyState
        icon={ClockIcon}
        title="No attendance records"
        description="Check-ins recorded by the lab device or an administrator will appear here."
      />
    );
  }

  return (
    <div className="lab-table-wrap">
      <table className="lab-table">
        <thead>
          <tr>
            <th>Date</th>
            <th>Entry</th>
            <th>Exit</th>
            <th>Duration</th>
            <th>Source</th>
          </tr>
        </thead>
        <tbody>
          {records.map((record) => {
            const entry = record.entry_time?.slice(11, 16) || record.entry_time;
            const exit = record.exit_time?.slice(11, 16) || record.exit_time;
            let duration = "—";
            if (record.entry_time && record.exit_time) {
              const minutes = Math.max(
                Math.round(
                  (new Date(record.exit_time).getTime() - new Date(record.entry_time).getTime()) / 60000,
                ),
                0,
              );
              duration = `${Math.floor(minutes / 60)}h ${minutes % 60}m`;
            }
            return (
              <tr key={record.id}>
                <td>{formatDate(record.entry_date)}</td>
                <td>{formatClockTime(entry)}</td>
                <td>{exit ? formatClockTime(exit) : <Badge tone="accent">In lab</Badge>}</td>
                <td>{duration}</td>
                <td>{titleCase(record.source)}</td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}

export default function ProfilePage() {
  const auth = useAuth();
  const [editing, setEditing] = useState(false);
  const [notice, setNotice] = useState(null);

  const profileResource = useApiResource(
    () => membersApi.profile(),
    "member-profile-page",
    { enabled: !auth.isAdmin },
  );
  const attendanceResource = useApiResource(
    () => attendanceApi.list({ per_page: 12 }),
    "member-profile-attendance",
    { enabled: !auth.isAdmin },
  );
  const todayResource = useApiResource(
    () => attendanceApi.today(),
    "member-profile-today",
    { enabled: !auth.isAdmin },
  );

  const member = profileResource.data?.member || auth.user?.member || {};
  const displayName = member.name || auth.user?.name || auth.user?.email || "Member";
  const interests = useMemo(
    () => (Array.isArray(member.interests) ? member.interests.filter(Boolean) : []),
    [member.interests],
  );

  const handleUpdated = (updatedMember) => {
    profileResource.setData((current) => ({
      ...(current || {}),
      success: true,
      member: { ...(current?.member || {}), ...updatedMember },
    }));
    setNotice("Your profile changes were saved.");
  };

  if (auth.isAdmin) {
    return (
      <>
        <PageHeader
          eyebrow="Account"
          title="Administrator profile"
          description="Your administrator identity and access level."
        />
        <div className="lab-profile-admin-grid">
          <Panel className="lab-admin-identity" padded={false}>
            <div className="lab-profile-summary__banner" aria-hidden="true" />
            <div className="lab-profile-summary__body">
              <Avatar name={auth.user?.name || auth.user?.email || "Admin"} size="xl" />
              <div className="lab-profile-summary__identity">
                <div className="lab-profile-summary__name-row">
                  <h2>{auth.user?.name || "Administrator"}</h2>
                  <Badge tone="accent">{auth.user?.admin?.is_primary ? "Primary admin" : "Admin"}</Badge>
                </div>
                <p>{auth.user?.email}</p>
              </div>
            </div>
          </Panel>
          <Panel title="Account details" subtitle="Managed by the Edwins Lab administrator team.">
            <DefinitionList
              items={[
                { label: "Email", value: auth.user?.email },
                { label: "Role", value: titleCase(auth.user?.role) },
                { label: "Account ID", value: auth.user?.id },
                { label: "Session", value: "Active" },
              ]}
            />
            <div className="lab-inline-link-row">
              <AppLink to="/app/settings" className="lab-text-link">
                <LockIcon className="h-3.5 w-3.5" />
                Password and session settings
              </AppLink>
            </div>
          </Panel>
          <Panel title="Administrator access" subtitle="Capabilities available to this account.">
            <ul className="lab-capability-list">
              <li><ShieldIcon className="h-4 w-4" /> Member records and account status</li>
              <li><ClockIcon className="h-4 w-4" /> Attendance records and manual entries</li>
              <li><CalendarIcon className="h-4 w-4" /> Announcements, events and tasks</li>
              <li><LockIcon className="h-4 w-4" /> Inventory, borrowing and returns</li>
            </ul>
          </Panel>
        </div>
      </>
    );
  }

  return (
    <>
      <PageHeader
        eyebrow="Member profile"
        title="Your profile"
        description="Keep your lab profile and public details up to date."
        actions={
          <AppButton variant="primary" onClick={() => setEditing(true)} disabled={profileResource.loading}>
            <EditIcon className="h-4 w-4" />
            Edit profile
          </AppButton>
        }
      />

      {notice && (
        <Alert tone="success" title="Saved" onDismiss={() => setNotice(null)}>
          {notice}
        </Alert>
      )}

      {profileResource.error && (
        <ErrorState
          error={profileResource.error}
          title="Member profile is unavailable"
          onRetry={profileResource.reload}
        />
      )}

      <div className="lab-profile-grid">
        <Panel className="lab-profile-panel" padded={false}>
          <div className="lab-profile-summary__banner" aria-hidden="true" />
          <div className="lab-profile-summary__body">
            <Avatar name={displayName} size="xl" />
            <div className="lab-profile-summary__identity">
              <div className="lab-profile-summary__name-row">
                <h2>{displayName}</h2>
                <StatusBadge status={member.status} />
              </div>
              <p>{auth.user?.email || member.email}</p>
              <div className="lab-profile-summary__chips">
                {member.usn && <Badge tone="neutral">USN {member.usn}</Badge>}
                {member.roll_number && <Badge tone="neutral">Roll {member.roll_number}</Badge>}
                {member.branch && <Badge tone="info">{member.branch}</Badge>}
              </div>
            </div>
          </div>
          <div className="lab-profile-summary__details">
            <div>
              <span>Academic year</span>
              <strong>{member.academic_year ? `Year ${member.academic_year}` : "—"}</strong>
            </div>
            <div>
              <span>Section</span>
              <strong>{member.section || "—"}</strong>
            </div>
            <div>
              <span>Member since</span>
              <strong>{member.created_at ? formatDate(member.created_at) : "—"}</strong>
            </div>
          </div>
        </Panel>

        <Panel title="Contact" subtitle="Official contact details on record.">
          <ul className="lab-contact-list">
            <li>
              <MailIcon className="h-4 w-4" />
              <div>
                <span>Email</span>
                <strong>{member.email || auth.user?.email || "—"}</strong>
              </div>
            </li>
            <li>
              <UserIcon className="h-4 w-4" />
              <div>
                <span>Login ID</span>
                <strong>{member.login_id || "—"}</strong>
              </div>
            </li>
            <li>
              <ShieldIcon className="h-4 w-4" />
              <div>
                <span>Phone</span>
                <strong>
                  {member.phone || (member.phone_verified ? "On file" : "Not provided")}
                </strong>
              </div>
              {member.phone_verified && <Badge tone="success">Verified</Badge>}
            </li>
          </ul>
          <div className="lab-inline-link-row">
            <AppLink to="/app/settings" className="lab-text-link">
              <LockIcon className="h-3.5 w-3.5" />
              Password and session settings
            </AppLink>
          </div>
        </Panel>

        <Panel
          title="About"
          subtitle="Your bio and areas of interest."
          className="lab-span-2"
          actions={
            <AppButton variant="ghost" size="sm" onClick={() => setEditing(true)}>
              <EditIcon className="h-3.5 w-3.5" />
              Edit
            </AppButton>
          }
        >
          {profileResource.loading ? (
            <LoadingState rows={3} compact />
          ) : (
            <>
              <p className="lab-profile-bio">
                {member.bio || "No bio added yet. Use Edit profile to introduce yourself."}
              </p>
              {interests.length > 0 && (
                <div className="lab-interest-chips">
                  {interests.map((interest) => (
                    <Badge key={interest} tone="neutral">{interest}</Badge>
                  ))}
                </div>
              )}
              <div className="lab-social-links">
                <SocialLink label="GitHub" value={member.github_url} />
                <SocialLink label="LinkedIn" value={member.linkedin_url} />
                <SocialLink label="Portfolio" value={member.portfolio_url} />
              </div>
              {!member.github_url && !member.linkedin_url && !member.portfolio_url && (
                <p className="lab-muted lab-small">No public links added.</p>
              )}
            </>
          )}
        </Panel>

        <div className="lab-stat-grid lab-stat-grid--3 lab-span-2">
          <StatCard
            label="Currently in the lab"
            value={formatNumber(todayResource.data?.summary?.total_present)}
            hint="Your open session"
            icon={ClockIcon}
            loading={todayResource.loading}
          />
          <StatCard
            label="Check-ins today"
            value={formatNumber(todayResource.data?.today?.length ?? 0)}
            hint="Entries recorded today"
            icon={CalendarIcon}
            loading={todayResource.loading}
          />
          <StatCard
            label="Profile status"
            value={member.status ? titleCase(member.status) : "—"}
            hint={member.created_at ? `Member since ${formatDate(member.created_at)}` : "Active member"}
            icon={UserIcon}
            tone="accent"
            loading={profileResource.loading}
          />
        </div>

        <Panel
          title="Attendance history"
          subtitle="Your most recent lab sessions."
          className="lab-span-2"
          actions={
            <span className="lab-muted lab-small">
              {attendanceResource.data?.total_count
                ? `${attendanceResource.data.total_count} records`
                : ""}
            </span>
          }
        >
          <AttendanceHistory resource={attendanceResource} />
          {attendanceResource.data?.attendance?.length > 0 && (
            <p className="lab-muted lab-small lab-table-note">
              Last updated {formatRelative(attendanceResource.data.attendance[0]?.created_at)}
            </p>
          )}
        </Panel>
      </div>

      <EditProfileDialog
        key={`${member.id || "member"}-${editing}`}
        open={editing}
        onClose={() => setEditing(false)}
        member={member}
        onUpdated={handleUpdated}
      />
    </>
  );
}
