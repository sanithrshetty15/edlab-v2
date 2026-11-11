import { useState } from "react";
import { useAuth } from "../../auth/context";
import { AppLink } from "../../components/app/Link";
import PasswordForm from "../../components/app/PasswordForm";
import {
  Alert,
  AppButton,
  Badge,
  DefinitionList,
  PageHeader,
  Panel,
} from "../../components/app/Ui";
import {
  ArrowRightIcon,
  InfoIcon,
  LockIcon,
  LogoutIcon,
  MailIcon,
  ShieldIcon,
  UserIcon,
} from "../../components/app/AppIcons";
import { navigate } from "../../utils/router";
import { titleCase } from "../../utils/format";

export default function SettingsPage() {
  const auth = useAuth();
  const [notice, setNotice] = useState(null);
  const [signingOut, setSigningOut] = useState(false);
  const displayName =
    auth.user?.admin?.name ||
    auth.user?.member?.name ||
    auth.user?.name ||
    auth.user?.email ||
    "—";

  const handleLogout = async () => {
    setSigningOut(true);
    await auth.logout();
    navigate("/login", { replace: true });
  };

  return (
    <>
      <PageHeader
        eyebrow="Account"
        title="Settings"
        description="Manage your password, review account details and control this session."
      />

      {notice && (
        <Alert tone="success" title="Saved" onDismiss={() => setNotice(null)}>
          {notice}
        </Alert>
      )}

      {auth.mustChangePassword && (
        <Alert tone="warning" title="Password change required">
          Your account is still using a temporary password. Update it below to keep full access.
        </Alert>
      )}

      <div className="lab-settings-grid">
        <Panel
          title="Account"
          subtitle="Your signed-in identity and access level."
          actions={
            <Badge tone={auth.isAdmin ? "accent" : "info"}>
              {auth.isAdmin ? "Administrator" : "Member"}
            </Badge>
          }
        >
          <DefinitionList
            items={[
              { label: "Name", value: displayName },
              { label: "Email", value: auth.user?.email || "—" },
              { label: "Role", value: titleCase(auth.user?.role) },
              { label: "Account ID", value: auth.user?.id || "—" },
            ]}
          />
          <div className="lab-inline-link-row">
            <AppLink to="/app/profile" className="lab-text-link">
              <UserIcon className="h-3.5 w-3.5" />
              View profile
              <ArrowRightIcon className="h-3.5 w-3.5" />
            </AppLink>
          </div>
        </Panel>

        <Panel title="Security" subtitle="Choose a strong password unique to Edwins Lab.">
          <div className="lab-settings-security">
            <div className="lab-settings-security__status">
              <span className="lab-settings-security__icon">
                <LockIcon className="h-4 w-4" />
              </span>
              <div>
                <strong>Password</strong>
                <p>
                  {auth.mustChangePassword
                    ? "Temporary password in use — change it now."
                    : "Your password is up to date."}
                </p>
              </div>
              <Badge tone={auth.mustChangePassword ? "warning" : "success"}>
                {auth.mustChangePassword ? "Action needed" : "Current"}
              </Badge>
            </div>
            <PasswordForm onSuccess={() => setNotice("Your password was updated.")} />
          </div>
        </Panel>

        <Panel title="Session" subtitle="Control the current browser session.">
          <ul className="lab-session-list">
            <li>
              <ShieldIcon className="h-4 w-4" />
              <div>
                <strong>Storage</strong>
                <p>Session token stored in this tab only</p>
              </div>
              <Badge tone="success">Session</Badge>
            </li>
            <li>
              <UserIcon className="h-4 w-4" />
              <div>
                <strong>Signed in as</strong>
                <p>{auth.user?.email || "—"}</p>
              </div>
            </li>
            <li>
              <MailIcon className="h-4 w-4" />
              <div>
                <strong>Session end</strong>
                <p>Closing the tab ends local access. The server token expires automatically.</p>
              </div>
            </li>
          </ul>
          <AppButton variant="outline" onClick={handleLogout} loading={signingOut}>
            <LogoutIcon className="h-4 w-4" />
            Sign out of this session
          </AppButton>
        </Panel>

        <Panel title="Lab account policies" subtitle="Managed by the Edwins Lab administrator team.">
          <ul className="lab-guideline-list">
            <li><InfoIcon className="h-4 w-4" /> Profile fields are reviewed by lab administrators.</li>
            <li><ShieldIcon className="h-4 w-4" /> Account status changes are applied by the lab team.</li>
            <li><LockIcon className="h-4 w-4" /> Passwords are never displayed or logged by this interface.</li>
          </ul>
          <p className="lab-muted lab-small">
            For account corrections, contact an administrator from the lab team.
          </p>
        </Panel>
      </div>
    </>
  );
}
