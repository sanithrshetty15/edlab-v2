import { useEffect, useState } from "react";
import wordmarkLight from "../assets/images/edwins-lab-wordmark-light.png";
import { isApiConfigured } from "../api/client";
import { useAuth } from "../auth/context";
import { AppLink } from "../components/app/Link";
import {
  ArrowLeftIcon,
  ArrowRightIcon,
  LockIcon,
  MailIcon,
  ShieldIcon,
  UserIcon,
} from "../components/app/AppIcons";
import {
  Alert,
  AppButton,
  Field,
  PasswordField,
  SegmentedControl,
  Tabs,
  TextInput,
} from "../components/app/Ui";
import { navigate } from "../utils/router";

const MODES = [
  { value: "member", label: "Member login" },
  { value: "admin", label: "Admin login" },
];

const LOGIN_TYPES = [
  { value: "usn", label: "USN" },
  { value: "roll_number", label: "Roll number" },
];

function validateMember({ loginType, loginId, password }) {
  const errors = {};
  if (!LOGIN_TYPES.some((option) => option.value === loginType)) {
    errors.loginType = "Choose USN or roll number.";
  }
  if (!loginId.trim()) errors.loginId = "Enter your USN or roll number.";
  if (!password) errors.password = "Enter your password.";
  return errors;
}

function validateAdmin({ email, password }) {
  const errors = {};
  if (!email.trim()) errors.email = "Enter your admin email.";
  else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.trim())) {
    errors.email = "Enter a valid email address.";
  }
  if (!password) errors.password = "Enter your password.";
  return errors;
}

export default function LoginPage({ sessionError, expired = false }) {
  const auth = useAuth();
  const [mode, setMode] = useState("member");
  const [loginType, setLoginType] = useState("usn");
  const [loginId, setLoginId] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [errors, setErrors] = useState({});
  const [formError, setFormError] = useState(null);
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    document.title = "Sign in — Edwins Lab";
    document.body.classList.add("login-mode");
    return () => document.body.classList.remove("login-mode");
  }, []);

  const switchMode = (nextMode) => {
    setMode(nextMode);
    setErrors({});
    setFormError(null);
    setPassword("");
  };

  const handleSubmit = async (event) => {
    event.preventDefault();
    if (submitting) return;

    const nextErrors = mode === "member"
      ? validateMember({ loginType, loginId, password })
      : validateAdmin({ email, password });

    setErrors(nextErrors);
    setFormError(null);
    if (Object.keys(nextErrors).length) return;

    setSubmitting(true);
    try {
      const result =
        mode === "member"
          ? await auth.loginMember({
              loginType,
              loginId: loginId.trim(),
              password,
            })
          : await auth.loginAdmin({ email: email.trim(), password });

      if (result.mustChangePassword) {
        navigate("/app/change-password", { replace: true });
      } else {
        navigate("/app/dashboard", { replace: true });
      }
    } catch (error) {
      setFormError(error);
      if (error?.status === 401) setPassword("");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="login-page">
      <div className="login-page__grid" aria-hidden="true" />

      <section className="login-brand" aria-label="Edwins Lab">
        <div className="login-brand__top">
          <AppLink to="/" className="login-brand__wordmark" aria-label="Edwins Lab home">
            <img src={wordmarkLight} alt="Edwins Lab" />
          </AppLink>
          <span className="login-brand__status">
            <span className="login-brand__status-dot" aria-hidden="true" />
            Systems online
          </span>
        </div>

        <div className="login-brand__copy">
          <p className="login-brand__eyebrow">Member &amp; admin portal</p>
          <h1>
            Build. Explore.
            <br />
            Contribute.
          </h1>
          <p className="login-brand__lead">
            One place for the lab: announcements, events, attendance, tasks, inventory
            and the people who make it work.
          </p>
        </div>

        <div className="login-brand__footer">
          <span>Edwins Lab</span>
          <span>College Open Source Technology Lab</span>
        </div>
      </section>

      <section className="login-panel" aria-label="Sign in">
        <div className="login-panel__inner">
          <AppLink to="/" className="login-back">
            <ArrowLeftIcon className="h-4 w-4" />
            Back to public website
          </AppLink>

          <header className="login-head">
            <p className="login-head__eyebrow">Welcome back</p>
            <h2>Sign in to your account</h2>
            <p>
              Use your lab credentials. On the Android app your session is stored in
              encrypted device storage.
            </p>
          </header>

          {!isApiConfigured() && (
            <Alert tone="warning" title="API connection is not configured">
              Set <code>VITE_API_BASE_URL</code> in the frontend environment to the deployed
              Rails API URL, then restart the dev server or rebuild.
            </Alert>
          )}

          {expired && (
            <Alert tone="warning" title="Session expired">
              Your session has ended. Sign in again to continue where you left off.
            </Alert>
          )}

          {sessionError && !expired && (
            <Alert tone="warning" title="Session could not be verified">
              {sessionError.message || "Check your connection and sign in again."}
            </Alert>
          )}

          {formError && (
            <Alert tone="danger" title="Sign-in failed">
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

          <Tabs tabs={MODES} value={mode} onChange={switchMode} label="Login type" className="login-tabs" />

          <form className="login-form" onSubmit={handleSubmit} noValidate>
            {mode === "member" ? (
              <>
                <div className="login-form__field">
                  <span className="lab-label" id="login-id-label">
                    Sign in with
                  </span>
                  <SegmentedControl
                    options={LOGIN_TYPES}
                    value={loginType}
                    onChange={(value) => {
                      setLoginType(value);
                      setErrors((current) => ({ ...current, loginType: undefined }));
                    }}
                    label="Member login identifier"
                    disabled={submitting}
                  />
                </div>

                <Field
                  label={loginType === "usn" ? "USN" : "Roll number"}
                  htmlFor="member-login-id"
                  error={errors.loginId}
                  required
                >
                  <div className="lab-input-wrap">
                    <UserIcon className="lab-input-icon" />
                    <TextInput
                      id="member-login-id"
                      className={`lab-input--with-icon ${errors.loginId ? "lab-input--invalid" : ""}`}
                      value={loginId}
                      onChange={(event) => setLoginId(event.target.value)}
                      placeholder={loginType === "usn" ? "4AL25CS000" : "Your roll number"}
                      autoComplete="username"
                      autoCapitalize="characters"
                      spellCheck="false"
                      aria-invalid={errors.loginId ? "true" : undefined}
                      disabled={submitting}
                    />
                  </div>
                </Field>
              </>
            ) : (
              <Field label="Admin email" htmlFor="admin-email" error={errors.email} required>
                <div className="lab-input-wrap">
                  <MailIcon className="lab-input-icon" />
                  <TextInput
                    id="admin-email"
                    type="email"
                    className={`lab-input--with-icon ${errors.email ? "lab-input--invalid" : ""}`}
                    value={email}
                    onChange={(event) => setEmail(event.target.value)}
                    placeholder="name@edwinslab.org"
                    autoComplete="username"
                    spellCheck="false"
                    aria-invalid={errors.email ? "true" : undefined}
                    disabled={submitting}
                  />
                </div>
              </Field>
            )}

            <PasswordField
              id="login-password"
              label="Password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              autoComplete="current-password"
              placeholder="Enter your password"
              error={errors.password}
              required
              disabled={submitting}
            />

            <div className="login-form__meta">
              <span>
                <LockIcon className="h-3.5 w-3.5" />
                Encrypted in transit
              </span>
              <span>
                <ShieldIcon className="h-3.5 w-3.5" />
                Server-side sessions
              </span>
            </div>

            <AppButton
              type="submit"
              size="lg"
              className="login-submit"
              loading={submitting}
              disabled={submitting}
            >
              {submitting ? "Signing in…" : "Sign in"}
              {!submitting && <ArrowRightIcon className="h-4 w-4" />}
            </AppButton>
          </form>

          <p className="login-help">
            Forgot your password? <AppLink to="/forgot-password">Reset it with email</AppLink>.
            <br />
            Member accounts are created by lab administrators.
          </p>

          <p className="login-legal">
            By signing in you agree to follow the Edwins Lab rules and responsible-use
            policies.
          </p>
        </div>
      </section>
    </div>
  );
}
