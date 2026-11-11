import { useEffect, useState } from "react";
import wordmarkLight from "../assets/images/edwins-lab-wordmark-light.png";
import { authApi } from "../api/services";
import { AppLink } from "../components/app/Link";
import { ArrowLeftIcon, ArrowRightIcon, LockIcon, MailIcon } from "../components/app/AppIcons";
import { Alert, AppButton, Field, PasswordField, TextInput } from "../components/app/Ui";
import { navigate } from "../utils/router";

export default function ForgotPasswordPage() {
  const [step, setStep] = useState("request");
  const [loginId, setLoginId] = useState("");
  const [otp, setOtp] = useState("");
  const [password, setPassword] = useState("");
  const [passwordConfirmation, setPasswordConfirmation] = useState("");
  const [errors, setErrors] = useState({});
  const [formError, setFormError] = useState(null);
  const [submitting, setSubmitting] = useState(false);
  const [sent, setSent] = useState(false);

  useEffect(() => {
    document.title = "Reset password — Edwins Lab";
    document.body.classList.add("login-mode");
    return () => document.body.classList.remove("login-mode");
  }, []);

  const requestCode = async (event) => {
    event.preventDefault();
    if (submitting) return;

    if (!loginId.trim()) {
      setErrors({ loginId: "Enter your lab login ID." });
      return;
    }

    setSubmitting(true);
    setFormError(null);
    try {
      await authApi.requestPasswordReset({ loginId: loginId.trim() });
      setSent(true);
      setStep("reset");
    } catch (error) {
      setFormError(error);
    } finally {
      setSubmitting(false);
    }
  };

  const resetPassword = async (event) => {
    event.preventDefault();
    if (submitting) return;

    const nextErrors = {};
    if (!otp.trim()) nextErrors.otp = "Enter the verification code from your email.";
    if (password.length < 8) nextErrors.password = "Use at least 8 characters.";
    if (passwordConfirmation !== password) {
      nextErrors.passwordConfirmation = "Passwords do not match.";
    }
    setErrors(nextErrors);
    setFormError(null);
    if (Object.keys(nextErrors).length) return;

    setSubmitting(true);
    try {
      await authApi.completePasswordReset({
        loginId: loginId.trim(),
        otpCode: otp.trim(),
        newPassword: password,
        passwordConfirmation,
      });
      navigate("/login", { replace: true });
    } catch (error) {
      setFormError(error);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="login-page forgot-page">
      <div className="login-page__grid" aria-hidden="true" />

      <section className="login-brand" aria-label="Edwins Lab">
        <div className="login-brand__top">
          <AppLink to="/" className="login-brand__wordmark" aria-label="Edwins Lab home">
            <img src={wordmarkLight} alt="Edwins Lab" />
          </AppLink>
        </div>
        <div className="login-brand__copy">
          <p className="login-brand__eyebrow">Account recovery</p>
          <h1>
            Get back
            <br />
            into the lab.
          </h1>
          <p className="login-brand__lead">
            We will email a one-time verification code to the address on your member
            record. Codes expire after a short window.
          </p>
        </div>
        <div className="login-brand__footer">
          <span>Edwins Lab</span>
          <span>Secure account recovery</span>
        </div>
      </section>

      <section className="login-panel" aria-label="Reset password">
        <div className="login-panel__inner">
          <AppLink to="/login" className="login-back">
            <ArrowLeftIcon className="h-4 w-4" />
            Back to sign in
          </AppLink>

          <header className="login-head">
            <p className="login-head__eyebrow">{step === "request" ? "Step 1 of 2" : "Step 2 of 2"}</p>
            <h2>{step === "request" ? "Reset your password" : "Enter your code"}</h2>
            <p>
              {step === "request"
                ? "Enter the login ID your lab account uses."
                : `If an account exists for ${loginId}, a code is on its way by email.`}
            </p>
          </header>

          {sent && step === "reset" && (
            <Alert tone="info" title="Check your email">
              Enter the verification code and choose a new password. If the email does
              not arrive, contact the lab team.
            </Alert>
          )}

          {formError && (
            <Alert tone="danger" title="Password reset failed">
              {formError.message}
            </Alert>
          )}

          {step === "request" ? (
            <form className="login-form" onSubmit={requestCode} noValidate>
              <Field label="Lab login ID" htmlFor="reset-login" error={errors.loginId} required>
                <div className="lab-input-wrap">
                  <MailIcon className="lab-input-icon" />
                  <TextInput
                    id="reset-login"
                    className={`lab-input--with-icon ${errors.loginId ? "lab-input--invalid" : ""}`}
                    value={loginId}
                    onChange={(event) => {
                      setLoginId(event.target.value);
                      setErrors({});
                    }}
                    placeholder="your.name"
                    autoComplete="username"
                    disabled={submitting}
                  />
                </div>
              </Field>
              <AppButton type="submit" size="lg" className="login-submit" loading={submitting}>
                Send verification code
                {!submitting && <ArrowRightIcon className="h-4 w-4" />}
              </AppButton>
            </form>
          ) : (
            <form className="login-form" onSubmit={resetPassword} noValidate>
              <Field
                label="Verification code"
                htmlFor="reset-otp"
                error={errors.otp}
                hint="The six-digit code from your email."
                required
              >
                <TextInput
                  id="reset-otp"
                  className={errors.otp ? "lab-input--invalid" : ""}
                  value={otp}
                  onChange={(event) => {
                    setOtp(event.target.value);
                    setErrors((current) => ({ ...current, otp: undefined }));
                  }}
                  inputMode="numeric"
                  autoComplete="one-time-code"
                  placeholder="000000"
                  disabled={submitting}
                />
              </Field>
              <PasswordField
                id="reset-password"
                label="New password"
                value={password}
                onChange={(event) => setPassword(event.target.value)}
                autoComplete="new-password"
                placeholder="At least 8 characters"
                error={errors.password}
                required
                disabled={submitting}
              />
              <PasswordField
                id="reset-password-confirmation"
                label="Confirm new password"
                value={passwordConfirmation}
                onChange={(event) => setPasswordConfirmation(event.target.value)}
                autoComplete="new-password"
                placeholder="Repeat your new password"
                error={errors.passwordConfirmation}
                required
                disabled={submitting}
              />
              <div className="login-form__meta">
                <span>
                  <LockIcon className="h-3.5 w-3.5" />
                  Sessions are signed out after reset
                </span>
              </div>
              <AppButton type="submit" size="lg" className="login-submit" loading={submitting}>
                Reset password
                {!submitting && <ArrowRightIcon className="h-4 w-4" />}
              </AppButton>
              <AppButton
                variant="ghost"
                onClick={() => {
                  setStep("request");
                  setFormError(null);
                  setOtp("");
                }}
                disabled={submitting}
              >
                Use a different login ID
              </AppButton>
            </form>
          )}
        </div>
      </section>
    </div>
  );
}
