import { useCallback, useEffect, useId, useRef } from "react";
import { createPortal } from "react-dom";
import { CloseIcon } from "./AppIcons";
import { AppButton } from "./Ui";

const FOCUSABLE =
  'a[href], button:not([disabled]), textarea:not([disabled]), input:not([disabled]), select:not([disabled]), [tabindex]:not([tabindex="-1"])';

export function Modal({
  open,
  onClose,
  title,
  description,
  children,
  footer,
  size = "md",
  className = "",
  closeLabel = "Close dialog",
  dismissible = true,
}) {
  const dialogRef = useRef(null);
  const previousFocusRef = useRef(null);
  const titleId = useId();
  const descriptionId = useId();

  const handleKeyDown = useCallback(
    (event) => {
      if (event.key === "Escape") {
        event.stopPropagation();
        if (dismissible) onClose();
        return;
      }

      if (event.key !== "Tab" || !dialogRef.current) return;
      const focusable = Array.from(dialogRef.current.querySelectorAll(FOCUSABLE)).filter(
        (element) => element.offsetParent !== null || element === document.activeElement,
      );
      if (!focusable.length) {
        event.preventDefault();
        return;
      }

      const first = focusable[0];
      const last = focusable[focusable.length - 1];
      if (event.shiftKey && document.activeElement === first) {
        event.preventDefault();
        last.focus();
      } else if (!event.shiftKey && document.activeElement === last) {
        event.preventDefault();
        first.focus();
      }
    },
    [onClose, dismissible],
  );

  useEffect(() => {
    if (!open) return undefined;

    previousFocusRef.current = document.activeElement;
    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";

    const focusTimer = window.setTimeout(() => {
      const dialog = dialogRef.current;
      if (!dialog) return;
      const target = dialog.querySelector(FOCUSABLE) || dialog;
      target.focus();
    }, 30);

    window.addEventListener("keydown", handleKeyDown);
    return () => {
      window.clearTimeout(focusTimer);
      window.removeEventListener("keydown", handleKeyDown);
      document.body.style.overflow = previousOverflow;
      const previous = previousFocusRef.current;
      if (previous && typeof previous.focus === "function") previous.focus();
    };
  }, [open, handleKeyDown]);

  if (!open) return null;

  return createPortal(
    <div className="lab-modal-layer">
      <button
        type="button"
        className="lab-modal-backdrop"
        aria-label={dismissible ? "Close dialog" : "Dialog backdrop"}
        onClick={dismissible ? onClose : undefined}
        tabIndex={-1}
      />
      <div
        ref={dialogRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        aria-describedby={description ? descriptionId : undefined}
        tabIndex={-1}
        className={`lab-modal lab-modal--${size} ${className}`}
      >
        <div className="lab-modal__head">
          <div>
            <h2 id={titleId} className="lab-modal__title">
              {title}
            </h2>
            {description && (
              <p id={descriptionId} className="lab-modal__description">
                {description}
              </p>
            )}
          </div>
          {dismissible && (
            <button
              type="button"
              className="lab-icon-button lab-icon-button--ghost"
              onClick={onClose}
              aria-label={closeLabel}
            >
              <CloseIcon className="h-4 w-4" />
            </button>
          )}
        </div>

        <div className="lab-modal__body">{children}</div>
        {footer && <div className="lab-modal__footer">{footer}</div>}
      </div>
    </div>,
    document.body,
  );
}

export function ConfirmDialog({
  open,
  onClose,
  onConfirm,
  title = "Are you sure?",
  description,
  confirmLabel = "Confirm",
  cancelLabel = "Cancel",
  loading = false,
  destructive = false,
}) {
  const confirmLockedRef = useRef(false);

  useEffect(() => {
    if (!open || !loading) confirmLockedRef.current = false;
  }, [open, loading]);

  const handleConfirm = () => {
    if (loading || confirmLockedRef.current) return;
    confirmLockedRef.current = true;
    onConfirm?.();
  };

  const handleClose = () => {
    if (loading) return;
    onClose?.();
  };

  return (
    <Modal
      open={open}
      onClose={handleClose}
      dismissible={!loading}
      title={title}
      description={description}
      size="sm"
      footer={
        <>
          <AppButton variant="ghost" onClick={handleClose} disabled={loading}>
            {cancelLabel}
          </AppButton>
          <AppButton
            variant={destructive ? "danger" : "primary"}
            onClick={handleConfirm}
            loading={loading}
          >
            {confirmLabel}
          </AppButton>
        </>
      }
    >
      <p className="lab-muted">This action is applied to the live Edwins Lab records.</p>
    </Modal>
  );
}
