import { useId, useState } from "react";
import { useAuth } from "../../auth/context";
import { Alert, AppButton, PasswordField } from "./Ui";

function scorePassword(value) {
  let score = 0;
  if (value.length >= 8) score += 1;
  if (value.length >= 12) score += 1;
  if (/[A-Z]/.test(value) && /[a-z]/.test(value)) score += 1;
  if (/\d/.test(value)) score += 1;
  if (/[^A-Za-z0-9]/.test(value)) score += 1;
  return Math.min(score, 4);
}

const STRENGTH_LABELS = ["Too short", "Weak", "Fair", "Strong", "Strong"];

export default function PasswordForm({
  onSuccess,
  submitLabel = "Update password",
  showCurrent = true,
  className = "",
}) {
  const auth = useAuth();
  const prefix = useId();
  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmation, setConfirmation] = useState("");
  const [errors, setErrors] = useState({});
  const [formError, setFormError] = useState(null);
  const [success, setSuccess] = useState(false);
  const [submitting, setSubmitting] = useState(false);

  const strength = scorePassword(newPassword);

  const handleSubmit = async (event) => {
    event.preventDefault();
    if (submitting) return;

    const nextErrors = {};
    if (showCurrent && !currentPassword) nextErrors.currentPassword = "Enter your current password.";
    if (!newPassword) nextErrors.newPassword = "Enter a new password.";
    else if (newPassword.length < 8) nextErrors.newPassword = "Use at least 8 characters.";
    if (!confirmation) nextErrors.confirmation = "Confirm your new password.";
    else if (newPassword !== confirmation) nextErrors.confirmation = "Passwords do not match.";
    if (showCurrent && newPassword && newPassword === currentPassword) {
      nextErrors.newPassword = "Choose a password different from your current one.";
    }

    setErrors(nextErrors);
    setFormError(null);
    setSuccess(false);
    if (Object.keys(nextErrors).length) return;

    setSubmitting(true);
    try {
      await auth.changePassword({
        currentPassword,
        newPassword,
        passwordConfirmation: confirmation,
      });
      setCurrentPassword("");
      setNewPassword("");
      setConfirmation("");
      setSuccess(true);
      if (onSuccess) onSuccess();
    } catch (error) {
      setFormError(error);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <form className={`lab-password-form ${className}`} onSubmit={handleSubmit} noValidate>
      {formError && (
        <Alert tone="danger" title="Password could not be changed">
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

      {success && (
        <Alert tone="success" title="Password updated">
          Your new password is active. Other signed-in sessions have been ended by the server.
        </Alert>
      )}

      {showCurrent && (
        <PasswordField
          id={`${prefix}-current`}
          label="Current password"
          value={currentPassword}
          onChange={(event) => setCurrentPassword(event.target.value)}
          autoComplete="current-password"
          error={errors.currentPassword}
          required
          disabled={submitting}
        />
      )}

      <PasswordField
        id={`${prefix}-new`}
        label="New password"
        value={newPassword}
        onChange={(event) => setNewPassword(event.target.value)}
        autoComplete="new-password"
        hint="Minimum 8 characters. A longer, unique password is safer."
        error={errors.newPassword}
        required
        disabled={submitting}
      />

      {newPassword && (
        <div className="lab-strength" aria-live="polite">
          <div className="lab-strength__bars" aria-hidden="true">
            {[0, 1, 2, 3].map((index) => (
              <span key={index} className={index < strength ? "is-filled" : ""} />
            ))}
          </div>
          <span>{STRENGTH_LABELS[strength]}</span>
        </div>
      )}

      <PasswordField
        id={`${prefix}-confirm`}
        label="Confirm new password"
        value={confirmation}
        onChange={(event) => setConfirmation(event.target.value)}
        autoComplete="new-password"
        error={errors.confirmation}
        required
        disabled={submitting}
      />

      <AppButton type="submit" variant="primary" loading={submitting} disabled={submitting}>
        {submitLabel}
      </AppButton>
    </form>
  );
}
