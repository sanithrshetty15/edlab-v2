import { useEffect } from "react";
import wordmarkLight from "../../assets/images/edwins-lab-wordmark-light.png";
import { useAuth } from "../../auth/context";
import { AppLink } from "../../components/app/Link";
import PasswordForm from "../../components/app/PasswordForm";
import { AppButton, Panel } from "../../components/app/Ui";
import { ArrowLeftIcon, LockIcon, LogoutIcon } from "../../components/app/AppIcons";
import { navigate } from "../../utils/router";

export default function PasswordChangePage({ mandatory = false, onComplete }) {
  const auth = useAuth();

  useEffect(() => {
    document.title = mandatory ? "Set a new password — Edwins Lab" : "Change password — Edwins Lab";
    document.body.classList.add("password-mode");
    return () => document.body.classList.remove("password-mode");
  }, [mandatory]);

  const handleLogout = async () => {
    await auth.logout();
    navigate("/login", { replace: true });
  };

  return (
    <div className="password-page">
      <div className="password-page__grid" aria-hidden="true" />

      <header className="password-page__top">
        <AppLink to={mandatory ? "/app/change-password" : "/app/settings"} className="password-page__brand">
          <img src={wordmarkLight} alt="Edwins Lab" />
        </AppLink>
        <button type="button" className="password-page__signout" onClick={handleLogout}>
          <LogoutIcon className="h-4 w-4" />
          Sign out
        </button>
      </header>

      <main className="password-page__main">
        <div className="password-page__icon">
          <LockIcon className="h-5 w-5" />
        </div>
        <p className="lab-eyebrow">{mandatory ? "Action required" : "Account security"}</p>
        <h1 className="password-page__title">
          {mandatory ? "Set a new password before continuing." : "Update your password."}
        </h1>
        <p className="password-page__lead">
          {mandatory
            ? "Your account is using a temporary password. Choose a private password to unlock the member platform."
            : "Choose a new password for your Edwins Lab account."}
        </p>

        <Panel className="password-page__panel">
          <PasswordForm
            submitLabel={mandatory ? "Save password and continue" : "Update password"}
            onSuccess={() => {
              if (mandatory && onComplete) onComplete();
            }}
          />
        </Panel>

        <div className="password-page__footer">
          {!mandatory && (
            <AppButton variant="ghost" onClick={() => navigate("/app/settings")}>
              <ArrowLeftIcon className="h-4 w-4" />
              Back to settings
            </AppButton>
          )}
          <span>Signed in as {auth.user?.email}</span>
        </div>
      </main>
    </div>
  );
}
