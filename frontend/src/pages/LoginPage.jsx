import { useEffect, useRef, useState } from "react";
import wordmarkLight from "../assets/images/edwins-lab-wordmark-light.png";
import { getApiBaseUrl, isApiConfigured } from "../api/client";
import { useAuth } from "../auth/context";
import { AppLink } from "../components/app/Link";
import {
  ArrowLeftIcon,
  ArrowRightIcon,
  GoogleIcon,
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
  ThreeDotLoader,
} from "../components/app/Ui";
import { navigate } from "../utils/router";

const RESEND_COOLDOWN_SECONDS = 60;

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

function validateAdmin({ email }) {
  const errors = {};
  if (!email.trim()) errors.email = "Enter your admin email.";
  else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.trim())) {
    errors.email = "Enter a valid email address.";
  }
  return errors;
}

function formatCountdown(seconds) {
  const minutes = Math.floor(seconds / 60);
  const remainder = seconds % 60;
  return `${minutes}:${String(remainder).padStart(2, "0")}`;
}

function readGoogleCallbackParams() {
  const params = new URLSearchParams(window.location.search);
  const code = params.get("google_code");
  const error = params.get("google_error");
  if (!code && !error) return null;
  return { code: code || "", error: error || "" };
}

function googleLoginErrorMessage(code) {
  switch (code) {
    case "access_denied":
      return "Google sign-in was cancelled. Try again or use an email verification code.";
    case "not_authorized":
      return "This Google account does not have administrator access. Use an authorized administrator account.";
    case "invalid_state":
      return "Google sign-in expired before it completed. Please try again.";
    case "invalid_token":
      return "Google could not verify that account. Please try again.";
    case "not_configured":
      return "Google sign-in is not configured yet. Use an email verification code.";
    default:
      return "Google sign-in could not be completed. Please try again.";
  }
}

export default function LoginPage({ sessionError, expired = false }) {
  const auth = useAuth();
  const { completeGoogleLogin } = auth;
  const [googleCallback] = useState(readGoogleCallbackParams);
  const [mode, setMode] = useState(() => (googleCallback ? "admin" : "member"));
  const [loginType, setLoginType] = useState("usn");
  const [loginId, setLoginId] = useState("");
  const [email, setEmail] = useState("");
  const [otp, setOtp] = useState("");
  const [password, setPassword] = useState("");
  const [errors, setErrors] = useState({});
  const [formError, setFormError] = useState(() =>
    googleCallback?.error ? new Error(googleLoginErrorMessage(googleCallback.error)) : null,
  );
  const [statusMessage, setStatusMessage] = useState(null);
  const [googleCompleting, setGoogleCompleting] = useState(() =>
    Boolean(googleCallback?.code && !googleCallback?.error),
  );
  const [submitting, setSubmitting] = useState(() =>
    Boolean(googleCallback?.code && !googleCallback?.error),
  );
  const [adminLoginInitiated, setAdminLoginInitiated] = useState(false);
  const [emailStatus, setEmailStatus] = useState("queued");
  const [resendSeconds, setResendSeconds] = useState(0);
  const googleCallbackHandled = useRef(false);

  useEffect(() => {
    if (!googleCallback || googleCallbackHandled.current) return;
    googleCallbackHandled.current = true;

    const cleanUrl = new URL(window.location.href);
    cleanUrl.searchParams.delete("google_code");
    cleanUrl.searchParams.delete("google_error");
    window.history.replaceState({}, "", `${cleanUrl.pathname}${cleanUrl.search}${cleanUrl.hash}`);

    if (googleCallback.error) return;

    completeGoogleLogin(googleCallback.code)
      .then((result) => {
        if (!result?.user) {
          throw new Error("The server did not complete the admin session.");
        }
        if (result.mustChangePassword) {
          navigate("/app/change-password", { replace: true });
        } else {
          navigate("/app/dashboard", { replace: true });
        }
      })
      .catch((error) => setFormError(error))
      .finally(() => {
        setGoogleCompleting(false);
        setSubmitting(false);
      });
  }, [completeGoogleLogin, googleCallback]);

  useEffect(() => {
    document.title = "Sign in — Edwins Lab";
    document.body.classList.add("login-mode");
    return () => document.body.classList.remove("login-mode");
  }, []);

  useEffect(() => {
    if (resendSeconds <= 0) return undefined;
    const timer = window.setTimeout(
      () => setResendSeconds((current) => Math.max(current - 1, 0)),
      1000,
    );
    return () => window.clearTimeout(timer);
  }, [resendSeconds]);

  const switchMode = (nextMode) => {
    if (nextMode === mode || submitting) return;
    setMode(nextMode);
    setErrors({});
    setFormError(null);
    setStatusMessage(null);
    setPassword("");
    setOtp("");
    setAdminLoginInitiated(false);
    setResendSeconds(0);
  };

  const beginAdminOtp = (result, nextEmail) => {
    setAdminLoginInitiated(true);
    setEmail(nextEmail);
    setOtp("");
    setErrors({});
    setFormError(null);
    setStatusMessage(null);
    setEmailStatus(result?.emailStatus || "queued");
    setResendSeconds(RESEND_COOLDOWN_SECONDS);
  };

  const resetAdminOtp = () => {
    setAdminLoginInitiated(false);
    setOtp("");
    setErrors({});
    setFormError(null);
    setStatusMessage(null);
    setResendSeconds(0);
  };

  const handleSubmit = async (event) => {
    event.preventDefault();
    if (submitting) return;

    const nextErrors =
      mode === "member"
        ? validateMember({ loginType, loginId, password })
        : validateAdmin({ email });

    setErrors(nextErrors);
    setFormError(null);
    setStatusMessage(null);
    if (Object.keys(nextErrors).length) return;

    setSubmitting(true);
    try {
      if (mode === "member") {
        const result = await auth.loginMember({
          loginType,
          loginId: loginId.trim(),
          password,
        });
        if (result.mustChangePassword) {
          navigate("/app/change-password", { replace: true });
        } else {
          navigate("/app/dashboard", { replace: true });
        }
        return;
      }

      const nextEmail = email.trim();
      const result = await auth.loginAdmin({ email: nextEmail });
      beginAdminOtp(result, nextEmail);
    } catch (error) {
      setFormError(error);
      if (error?.status === 401) setPassword("");
    } finally {
      setSubmitting(false);
    }
  };

  const resendOtp = async () => {
    if (submitting || resendSeconds > 0) return;
    setErrors({});
    setFormError(null);
    setStatusMessage(null);
    setSubmitting(true);
    try {
      const result = await auth.loginAdmin({ email: email.trim() });
      setEmailStatus(result?.emailStatus || "queued");
      setResendSeconds(RESEND_COOLDOWN_SECONDS);
      setStatusMessage("A new verification code was queued for delivery.");
    } catch (error) {
      setFormError(error);
    } finally {
      setSubmitting(false);
    }
  };

  const handleOtpSubmit = async (event) => {
    event.preventDefault();
    if (submitting) return;

    const code = otp.trim();
    if (!/^\d{4}$/.test(code)) {
      setErrors({ otp: "Enter the 4-digit code from your email." });
      return;
    }

    setErrors({});
    setFormError(null);
    setStatusMessage(null);
    setSubmitting(true);
    try {
      const result = await auth.verifyAndCompleteAdminLogin(code);
      if (!result?.user) {
        throw new Error("The server did not complete the admin session.");
      }
      if (result.mustChangePassword) {
        navigate("/app/change-password", { replace: true });
      } else {
        navigate("/app/dashboard", { replace: true });
      }
    } catch (error) {
      setFormError(error);
      if (error?.status === 401) setOtp("");
    } finally {
      setSubmitting(false);
    }
  };

  const emailDeliveryCopy =
    emailStatus === "sent"
      ? `We sent a 4-digit verification code to ${email}.`
      : `We queued a 4-digit verification code for ${email}. It should arrive shortly.`;

  return (
    <div className="login-page">
      <div className="login-page__grid" aria-hidden="true" />

      <section className="login-brand" aria-label="Edwins Lab">
        <div className="login-brand__top">
          <AppLink to="/" className="login-brand__wordmark" aria-label="Edwins Lab home">
            <img src={wordmarkLight} alt="Edwins Lab" />
          </AppLink>
          <span className="login-brand__status">
            <span className="login-brand__status-dot" aria-hidden="true" /> Systems online
          </span>
        </div>

        <div className="login-brand__copy">
          <p className="login-brand__eyebrow">Member &amp; admin portal</p>
          <h1>
            Build. Explore.
            <br /> Contribute.
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
            <ArrowLeftIcon className="h-4 w-4" /> Back to public website
          </AppLink>

          {adminLoginInitiated ? (
            <>
              <header className="login-head">
                <p className="login-head__eyebrow">Admin verification</p>
                <h2>Enter your verification code</h2>
                <p>
                  {emailDeliveryCopy} The code expires five minutes after it is
                  created and can be used once.
                </p>
              </header>

              {statusMessage && (
                <Alert tone="success" title="Code requested">
                  {statusMessage}
                </Alert>
              )}

              {formError && (
                <Alert tone="danger" title="Verification failed">
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

              <form className="login-form" onSubmit={handleOtpSubmit} noValidate>
                <Field
                  label="4-digit verification code"
                  htmlFor="admin-otp"
                  error={errors.otp}
                  hint="Check the inbox for the administrator email address above."
                  required
                >
                  <div className="lab-input-wrap">
                    <LockIcon className="lab-input-icon" />
                    <TextInput
                      id="admin-otp"
                      type="text"
                      className={`lab-input--with-icon login-otp-input ${
                        errors.otp ? "lab-input--invalid" : ""
                      }`}
                      value={otp}
                      onChange={(event) => {
                        setOtp(event.target.value.replace(/\D/g, "").slice(0, 4));
                        setErrors((current) => ({ ...current, otp: undefined }));
                      }}
                      placeholder="0000"
                      maxLength={4}
                      inputMode="numeric"
                      pattern="[0-9]{4}"
                      autoComplete="one-time-code"
                      autoFocus
                      spellCheck="false"
                      aria-invalid={errors.otp ? "true" : undefined}
                      disabled={submitting}
                    />
                  </div>
                </Field>

                <div className="login-form__meta">
                  <span>
                    <LockIcon className="h-3.5 w-3.5" /> Encrypted in transit
                  </span>
                  <span>
                    <ShieldIcon className="h-3.5 w-3.5" /> Server-side sessions
                  </span>
                </div>

                <AppButton
                  type="submit"
                  size="lg"
                  className="login-submit"
                  loading={submitting}
                  disabled={submitting}
                >
                  Verify and sign in
                  <ArrowRightIcon className="h-4 w-4" />
                </AppButton>

                <div className="login-otp-actions">
                  <p className="login-help">
                    Did not receive the code?{" "}
                    {resendSeconds > 0 ? (
                      <span className="login-resend-countdown" aria-live="polite">
                        Resend available in {formatCountdown(resendSeconds)}
                      </span>
                    ) : (
                      <button
                        type="button"
                        className="login-resend-button"
                        onClick={resendOtp}
                        disabled={submitting}
                      >
                        Resend code
                      </button>
                    )}
                  </p>
                  <button
                    type="button"
                    className="login-resend-button"
                    onClick={resetAdminOtp}
                    disabled={submitting}
                  >
                    Use a different email address
                  </button>
                </div>
              </form>
            </>
          ) : (
            <>
              <header className="login-head">
                <p className="login-head__eyebrow">Welcome back</p>
                <h2>Sign in to your account</h2>
                <p>
                  Members sign in with their lab login ID and password. Administrators
                  can use a one-time verification code by email or an authorized Google
                  account.
                </p>
              </header>

              {!isApiConfigured() && (
                <Alert tone="warning" title="API connection is not configured">
                  Set <code>VITE_API_BASE_URL</code> in the frontend environment to the
                  deployed Edwins Lab API URL, then restart or rebuild.
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

              {googleCompleting && (
                <Alert tone="info" title="Completing Google sign-in">
                  <div className="login-completing-state">
                    <ThreeDotLoader size="sm" />
                    <span>Verifying your Google account and creating your admin session…</span>
                  </div>
                </Alert>
              )}

              <Tabs
                tabs={MODES}
                value={mode}
                onChange={switchMode}
                label="Login type"
                className="login-tabs"
              />

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
                          className={`lab-input--with-icon ${
                            errors.loginId ? "lab-input--invalid" : ""
                          }`}
                          value={loginId}
                          onChange={(event) => setLoginId(event.target.value)}
                          placeholder={
                            loginType === "usn" ? "4AL25CS000" : "Your roll number"
                          }
                          autoComplete="username"
                          autoCapitalize="characters"
                          spellCheck="false"
                          aria-invalid={errors.loginId ? "true" : undefined}
                          disabled={submitting}
                        />
                      </div>
                    </Field>

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
                        <LockIcon className="h-3.5 w-3.5" /> Encrypted in transit
                      </span>
                      <span>
                        <ShieldIcon className="h-3.5 w-3.5" /> Server-side sessions
                      </span>
                    </div>

                    <AppButton
                      type="submit"
                      size="lg"
                      className="login-submit"
                      loading={submitting}
                      disabled={submitting}
                    >
                      Sign in
                      <ArrowRightIcon className="h-4 w-4" />
                    </AppButton>
                  </>
                ) : (
                  <>
                    <Field
                      label="Admin email"
                      htmlFor="admin-email"
                      error={errors.email}
                      hint="We will email a 4-digit verification code to this address."
                      required
                    >
                      <div className="lab-input-wrap">
                        <MailIcon className="lab-input-icon" />
                        <TextInput
                          id="admin-email"
                          type="email"
                          className={`lab-input--with-icon ${
                            errors.email ? "lab-input--invalid" : ""
                          }`}
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

                    <div className="login-form__meta">
                      <span>
                        <LockIcon className="h-3.5 w-3.5" /> Encrypted in transit
                      </span>
                      <span>
                        <ShieldIcon className="h-3.5 w-3.5" /> Server-side sessions
                      </span>
                    </div>

                    <AppButton
                      type="submit"
                      size="lg"
                      className="login-submit"
                      loading={submitting}
                      disabled={submitting}
                    >
                      Send verification code
                      <ArrowRightIcon className="h-4 w-4" />
                    </AppButton>

                    {isApiConfigured() && (
                      <div className="login-alternate-auth">
                        <div className="login-provider-divider" role="separator" aria-label="Alternative sign-in options">
                          <span>OR</span>
                        </div>
                        <a
                          className={`lab-button lab-button--outline lab-button--lg login-google-button ${
                            submitting ? "login-google-button--disabled" : ""
                          }`}
                          href={`${getApiBaseUrl()}/api/v1/oauth/google/start`}
                          aria-disabled={submitting || undefined}
                          tabIndex={submitting ? -1 : undefined}
                          onClick={(event) => {
                            if (submitting) event.preventDefault();
                          }}
                        >
                          <GoogleIcon className="h-4 w-4" />
                          Continue with Google
                        </a>
                        <p className="login-google-note">
                          Only existing Edwins Lab administrators can use Google sign-in.
                        </p>
                      </div>
                    )}
                  </>
                )}

                <p className="login-help">
                  Forgot your password?{" "}
                  <AppLink to="/forgot-password">Reset it with email</AppLink>.
                  <br />
                  Member accounts are created by lab administrators.
                </p>
              </form>
            </>
          )}

          <p className="login-legal">
            By signing in you agree to follow the Edwins Lab rules and responsible-use
            policies.
          </p>
        </div>
      </section>
    </div>
  );
}
