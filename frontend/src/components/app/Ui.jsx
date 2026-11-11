import { useId, useState } from "react";
import { AlertIcon, CheckIcon, EyeIcon, EyeOffIcon, InfoIcon, WarningIcon } from "./AppIcons";

export function Spinner({ className = "h-4 w-4" }) {
  return <span className={`lab-spinner ${className}`} aria-hidden="true" />;
}

export function AppButton({
  children,
  variant = "primary",
  size = "md",
  loading = false,
  className = "",
  type = "button",
  disabled,
  ...props
}) {
  return (
    <button
      type={type}
      className={`lab-button lab-button--${variant} lab-button--${size} ${className}`}
      disabled={disabled || loading}
      {...props}
    >
      {loading && <Spinner className={size === "sm" ? "h-3.5 w-3.5" : "h-4 w-4"} />}
      {children}
    </button>
  );
}

export function IconButton({ label, children, className = "", ...props }) {
  return (
    <button
      type="button"
      className={`lab-icon-button ${className}`}
      aria-label={label}
      title={label}
      {...props}
    >
      {children}
    </button>
  );
}

export function PageHeader({ eyebrow, title, description, actions, children }) {
  return (
    <header className="lab-page-header">
      <div>
        {eyebrow && <p className="lab-eyebrow">{eyebrow}</p>}
        <h1 className="lab-page-title">{title}</h1>
        {description && <p className="lab-page-description">{description}</p>}
        {children}
      </div>
      {actions && <div className="lab-page-actions">{actions}</div>}
    </header>
  );
}

export function Panel({
  title,
  subtitle,
  actions,
  children,
  className = "",
  padded = true,
  as: Component = "section",
}) {
  return (
    <Component className={`lab-panel ${className}`}>
      {(title || subtitle || actions) && (
        <div className="lab-panel__head">
          <div>
            {title && <h2 className="lab-panel__title">{title}</h2>}
            {subtitle && <p className="lab-panel__subtitle">{subtitle}</p>}
          </div>
          {actions && <div className="lab-panel__actions">{actions}</div>}
        </div>
      )}
      <div className={padded ? "lab-panel__body" : undefined}>{children}</div>
    </Component>
  );
}

export function StatCard({ label, value, hint, icon: IconComponent, tone = "neutral", loading }) {
  return (
    <div className={`lab-stat lab-stat--${tone}`}>
      <div className="lab-stat__top">
        <span className="lab-stat__label">{label}</span>
        {IconComponent && (
          <span className="lab-stat__icon">
            <IconComponent className="h-4 w-4" />
          </span>
        )}
      </div>
      {loading ? (
        <span className="lab-skeleton lab-skeleton--value" />
      ) : (
        <span className="lab-stat__value">{value ?? "—"}</span>
      )}
      {hint && <span className="lab-stat__hint">{hint}</span>}
    </div>
  );
}

const BADGE_TONES = {
  neutral: "lab-badge--neutral",
  accent: "lab-badge--accent",
  success: "lab-badge--success",
  warning: "lab-badge--warning",
  danger: "lab-badge--danger",
  info: "lab-badge--info",
};

export function Badge({ children, tone = "neutral", dot = false, className = "" }) {
  return (
    <span className={`lab-badge ${BADGE_TONES[tone] || BADGE_TONES.neutral} ${className}`}>
      {dot && <span className="lab-badge__dot" aria-hidden="true" />}
      {children}
    </span>
  );
}

function statusTone(status) {
  switch (status) {
    case "active":
    case "approved":
    case "verified":
    case "returned":
    case "published":
    case "completed":
      return "success";
    case "pending":
    case "scheduled":
    case "submitted":
    case "in_progress":
      return "warning";
    case "inactive":
    case "deactivated":
    case "rejected":
    case "overdue":
    case "cancelled":
    case "deleted":
      return "danger";
    default:
      return "neutral";
  }
}

export function StatusBadge({ status, children }) {
  return (
    <Badge tone={statusTone(status)} dot>
      {children || status || "—"}
    </Badge>
  );
}

export function Avatar({ name, size = "md", className = "" }) {
  const parts = String(name || "Edwins Lab")
    .trim()
    .split(/\s+/)
    .filter(Boolean);
  const letters = (parts.length ? parts.slice(0, 2).map((part) => part[0]) : ["E", "L"])
    .join("")
    .toUpperCase();

  return (
    <span className={`lab-avatar lab-avatar--${size} ${className}`} aria-hidden="true">
      {letters}
    </span>
  );
}

export function EmptyState({ icon: IconComponent, title, description, action, className = "" }) {
  return (
    <div className={`lab-empty ${className}`}>
      {IconComponent && (
        <span className="lab-empty__icon">
          <IconComponent className="h-5 w-5" />
        </span>
      )}
      <h3 className="lab-empty__title">{title}</h3>
      {description && <p className="lab-empty__description">{description}</p>}
      {action && <div className="lab-empty__action">{action}</div>}
    </div>
  );
}

export function LoadingState({ label = "Loading", rows = 3, compact = false }) {
  return (
    <div className={`lab-loading ${compact ? "lab-loading--compact" : ""}`} role="status">
      <span className="lab-sr-only">{label}</span>
      {Array.from({ length: rows }).map((_, index) => (
        <span
          key={index}
          className="lab-skeleton lab-skeleton--row"
          style={{ width: `${92 - index * 11}%` }}
        />
      ))}
    </div>
  );
}

function errorMessage(error) {
  if (!error) return "Something went wrong.";
  if (typeof error === "string") return error;
  return error.message || "Something went wrong.";
}

export function ErrorState({ error, title = "Couldn’t load this section", onRetry, compact = false }) {
  return (
    <div className={`lab-error ${compact ? "lab-error--compact" : ""}`} role="alert">
      <span className="lab-error__icon">
        <AlertIcon className="h-4 w-4" />
      </span>
      <div>
        <p className="lab-error__title">{title}</p>
        <p className="lab-error__message">{errorMessage(error)}</p>
        {error?.status && <p className="lab-error__meta">Server response: HTTP {error.status}</p>}
      </div>
      {onRetry && (
        <AppButton variant="outline" size="sm" onClick={onRetry} className="lab-error__retry">
          Retry
        </AppButton>
      )}
    </div>
  );
}

const ALERT_ICONS = {
  info: InfoIcon,
  success: CheckIcon,
  warning: WarningIcon,
  danger: AlertIcon,
};

export function Alert({ tone = "info", title, children, action, onDismiss, className = "" }) {
  const IconComponent = ALERT_ICONS[tone] || InfoIcon;
  return (
    <div className={`lab-alert lab-alert--${tone} ${className}`} role={tone === "danger" ? "alert" : "status"}>
      <span className="lab-alert__icon">
        <IconComponent className="h-4 w-4" />
      </span>
      <div className="lab-alert__content">
        {title && <p className="lab-alert__title">{title}</p>}
        <div className="lab-alert__body">{children}</div>
      </div>
      {action}
      {onDismiss && (
        <button type="button" className="lab-alert__dismiss" onClick={onDismiss} aria-label="Dismiss">
          ×
        </button>
      )}
    </div>
  );
}

export function Field({ label, htmlFor, hint, error, required, children, className = "" }) {
  return (
    <div className={`lab-field ${className}`}>
      {label && (
        <label className="lab-label" htmlFor={htmlFor}>
          {label}
          {required && <span className="lab-required">*</span>}
        </label>
      )}
      {children}
      {error && <p className="lab-field__error">{error}</p>}
      {!error && hint && <p className="lab-field__hint">{hint}</p>}
    </div>
  );
}

export function TextInput({ className = "", ...props }) {
  return <input className={`lab-input ${className}`} {...props} />;
}

export function TextArea({ className = "", rows = 5, ...props }) {
  return <textarea className={`lab-input lab-textarea ${className}`} rows={rows} {...props} />;
}

export function Select({ className = "", children, ...props }) {
  return (
    <div className="lab-select-wrap">
      <select className={`lab-input lab-select ${className}`} {...props}>
        {children}
      </select>
    </div>
  );
}

export function PasswordField({
  label = "Password",
  id,
  value,
  onChange,
  autoComplete = "current-password",
  placeholder,
  error,
  hint,
  required = false,
  disabled = false,
}) {
  const generatedId = useId();
  const inputId = id || generatedId;
  const [visible, setVisible] = useState(false);

  return (
    <Field label={label} htmlFor={inputId} error={error} hint={hint} required={required}>
      <div className="lab-password-wrap">
        <input
          id={inputId}
          className={`lab-input lab-input--password ${error ? "lab-input--invalid" : ""}`}
          type={visible ? "text" : "password"}
          value={value}
          onChange={onChange}
          autoComplete={autoComplete}
          placeholder={placeholder}
          required={required}
          disabled={disabled}
          aria-invalid={error ? "true" : undefined}
        />
        <button
          type="button"
          className="lab-password-toggle"
          onClick={() => setVisible((current) => !current)}
          aria-label={visible ? "Hide password" : "Show password"}
          aria-pressed={visible}
          tabIndex={0}
        >
          {visible ? <EyeOffIcon className="h-4 w-4" /> : <EyeIcon className="h-4 w-4" />}
        </button>
      </div>
    </Field>
  );
}

export function Checkbox({ label, id, checked, onChange, disabled = false }) {
  const generatedId = useId();
  const inputId = id || generatedId;
  return (
    <label className="lab-checkbox" htmlFor={inputId}>
      <input
        id={inputId}
        type="checkbox"
        checked={checked}
        onChange={onChange}
        disabled={disabled}
      />
      <span className="lab-checkbox__box" aria-hidden="true" />
      <span>{label}</span>
    </label>
  );
}

export function SegmentedControl({ options, value, onChange, label, disabled = false }) {
  return (
    <div className="lab-segmented" role="group" aria-label={label}>
      {options.map((option) => (
        <button
          key={option.value}
          type="button"
          className={`lab-segmented__item ${value === option.value ? "is-active" : ""}`}
          onClick={() => onChange(option.value)}
          aria-pressed={value === option.value}
          disabled={disabled}
        >
          {option.label}
        </button>
      ))}
    </div>
  );
}

export function Tabs({ tabs, value, onChange, label = "Views", className = "" }) {
  const handleKeyDown = (event, index) => {
    if (event.key !== "ArrowRight" && event.key !== "ArrowLeft") return;
    event.preventDefault();
    const direction = event.key === "ArrowRight" ? 1 : -1;
    const next = (index + direction + tabs.length) % tabs.length;
    onChange(tabs[next].value);
    const target = event.currentTarget.parentElement?.querySelectorAll("[role='tab']")[next];
    target?.focus();
  };

  return (
    <div className={`lab-tabs ${className}`} role="tablist" aria-label={label}>
      {tabs.map((tab, index) => (
        <button
          key={tab.value}
          type="button"
          role="tab"
          aria-selected={value === tab.value}
          className={`lab-tabs__tab ${value === tab.value ? "is-active" : ""}`}
          onClick={() => onChange(tab.value)}
          onKeyDown={(event) => handleKeyDown(event, index)}
          tabIndex={value === tab.value ? 0 : -1}
        >
          {tab.label}
          {tab.count !== undefined && <span className="lab-tabs__count">{tab.count}</span>}
        </button>
      ))}
    </div>
  );
}

export function Pagination({ page, totalPages, totalCount, onPageChange, loading = false }) {
  const safeTotal = Math.max(Number(totalPages) || 1, 1);
  if (safeTotal <= 1) {
    return totalCount !== undefined ? (
      <p className="lab-pagination__summary">{totalCount} records</p>
    ) : null;
  }

  return (
    <div className="lab-pagination">
      <p className="lab-pagination__summary">
        Page {page} of {safeTotal}
        {totalCount !== undefined ? ` · ${totalCount} records` : ""}
      </p>
      <div className="lab-pagination__controls">
        <AppButton
          variant="outline"
          size="sm"
          onClick={() => onPageChange(Math.max(page - 1, 1))}
          disabled={page <= 1 || loading}
        >
          Previous
        </AppButton>
        <AppButton
          variant="outline"
          size="sm"
          onClick={() => onPageChange(Math.min(page + 1, safeTotal))}
          disabled={page >= safeTotal || loading}
        >
          Next
        </AppButton>
      </div>
    </div>
  );
}

export function DefinitionList({ items, columns = 2 }) {
  return (
    <dl className={`lab-definition-list lab-definition-list--${columns}`}>
      {items.map((item) => (
        <div key={item.label} className="lab-definition-list__item">
          <dt>{item.label}</dt>
          <dd>{item.value ?? "—"}</dd>
        </div>
      ))}
    </dl>
  );
}

export function PageLoading({ label = "Loading Edwins Lab" }) {
  return (
    <div className="lab-boot" role="status">
      <div className="lab-boot__mark">EL</div>
      <p>{label}</p>
      <Spinner className="h-5 w-5" />
    </div>
  );
}
