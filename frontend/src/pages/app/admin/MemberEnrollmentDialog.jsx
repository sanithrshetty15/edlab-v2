import { useCallback, useEffect, useRef, useState } from "react";
import { membersManagementApi } from "../../../api/services";
import { ConfirmDialog, Modal } from "../../../components/app/Modal";
import {
  Alert,
  AppButton,
  Badge,
  DefinitionList,
  Spinner,
} from "../../../components/app/Ui";
import {
  CheckIcon,
  ClockIcon,
  RefreshIcon,
  ShieldIcon,
  XCircleIcon,
} from "../../../components/app/AppIcons";
import { formatDate } from "../../../utils/format";

const ENROLLMENT_STEPS = [
  {
    status: "awaiting_sensor",
    title: "Enrollment reader ready",
    description: "The configured R307S reader will claim this session shortly.",
  },
  {
    status: "waiting_first_scan",
    title: "First fingerprint scan",
    description: "Place the member's finger on the enrollment sensor.",
  },
  {
    status: "first_scan_captured",
    title: "First scan captured",
    description: "Keep the finger still while the sensor processes the image.",
  },
  {
    status: "waiting_for_removal",
    title: "Remove finger",
    description: "Lift the finger so the sensor is ready for the second scan.",
  },
  {
    status: "waiting_second_scan",
    title: "Second fingerprint scan",
    description: "Place the same finger on the sensor again.",
  },
  {
    status: "second_scan_captured",
    title: "Second scan captured",
    description: "The reader is checking that both scans match.",
  },
  {
    status: "verifying",
    title: "Verifying and saving",
    description: "The reader is saving the template and confirming the slot.",
  },
  {
    status: "finalized",
    title: "Enrollment complete",
    description: "The member account is now active.",
  },
];

const ACTIVE_STATUSES = new Set([
  "awaiting_sensor",
  "waiting_first_scan",
  "first_scan_captured",
  "waiting_for_removal",
  "waiting_second_scan",
  "second_scan_captured",
  "verifying",
]);

const TERMINAL_STATUSES = new Set(["finalized", "failed", "cancelled", "expired"]);

function statusIndex(status) {
  return ENROLLMENT_STEPS.findIndex((step) => step.status === status);
}

function EnrollmentSteps({ status }) {
  const current = statusIndex(status);
  return (
    <ol className="lab-enrollment-steps" aria-label="Fingerprint enrollment steps">
      {ENROLLMENT_STEPS.filter((step) => step.status !== "finalized").map((step, index) => {
        const state =
          current === -1
            ? "upcoming"
            : index < current
              ? "done"
              : index === current
                ? "current"
                : "upcoming";
        return (
          <li key={step.status} className={`lab-enrollment-step lab-enrollment-step--${state}`}>
            <span className="lab-enrollment-step__index" aria-hidden="true">
              {state === "done" ? <CheckIcon className="h-3.5 w-3.5" /> : index + 1}
            </span>
            <div>
              <strong>{step.title}</strong>
              <p>{step.description}</p>
            </div>
          </li>
        );
      })}
    </ol>
  );
}

function statusCopy(session) {
  if (!session) {
    return {
      title: "Ready to enroll",
      description: "Start the enrollment session when the member is ready at the sensor.",
    };
  }
  if (session.status === "pending_details") {
    return {
      title: "Ready to enroll",
      description: "Start the enrollment session when the member is ready at the sensor.",
    };
  }
  if (session.status === "finalized") {
    return {
      title: "Member activated",
      description: "Both scans matched and the fingerprint slot was saved successfully.",
    };
  }
  if (session.status === "failed") {
    return {
      title: "Enrollment failed",
      description:
        session.error_message ||
        "The reader could not complete enrollment. Check the sensor and try again.",
    };
  }
  if (session.status === "expired") {
    return {
      title: "Enrollment timed out",
      description:
        "The sensor did not finish within the enrollment window. The member is still pending and can be retried.",
    };
  }
  if (session.status === "cancelled") {
    return {
      title: "Enrollment cancelled",
      description: "This enrollment attempt was cancelled. No member was activated.",
    };
  }
  const step = ENROLLMENT_STEPS.find((item) => item.status === session.status);
  return {
    title: step?.title || "Enrollment in progress",
    description: step?.description || "Keep the member at the enrollment sensor.",
  };
}

export default function MemberEnrollmentDialog({
  member,
  initialEnrollment,
  onClose,
  onFinalized,
  onCancelled,
}) {
  const [session, setSession] = useState(initialEnrollment || null);
  const [busy, setBusy] = useState(null);
  const [error, setError] = useState(null);
  const [confirmCancel, setConfirmCancel] = useState(false);
  const [success, setSuccess] = useState(initialEnrollment?.status === "finalized");
  const finalizedRef = useRef(success);

  const isActive = Boolean(
    session?.is_active || (session && ACTIVE_STATUSES.has(session.status)),
  );
  const isTerminal = Boolean(session && TERMINAL_STATUSES.has(session.status));
  const canClose = !isActive || success;
  const copy = statusCopy(session);

  const applyResult = useCallback(
    (result) => {
      if (result?.enrollment) {
        setSession(result.enrollment);
      }
      if (result?.enrollment?.status === "finalized") {
        setSuccess(true);
        if (!finalizedRef.current) {
          finalizedRef.current = true;
          onFinalized?.(result);
        }
      }
    },
    [onFinalized],
  );

  useEffect(() => {
    if (!isActive || success) return undefined;
    let stopped = false;
    let timer = null;

    const poll = async () => {
      try {
        const result = await membersManagementApi.getFingerprintEnrollment(member.id);
        if (stopped) return;
        applyResult(result);
        setError(null);
        if (result?.enrollment?.status === "finalized") {
          window.clearInterval(timer);
        }
      } catch (pollError) {
        if (!stopped) setError(pollError);
      }
    };

    poll();
    timer = window.setInterval(poll, 2000);
    return () => {
      stopped = true;
      window.clearInterval(timer);
    };
  }, [applyResult, isActive, member?.id, success]);

  const handleStart = async () => {
    if (busy) return;
    setBusy("start");
    setError(null);
    try {
      const result = await membersManagementApi.startFingerprintEnrollment(member.id);
      applyResult(result);
    } catch (startError) {
      setError(startError);
    } finally {
      setBusy(null);
    }
  };

  const handleRefresh = async () => {
    if (busy) return;
    setBusy("refresh");
    setError(null);
    try {
      const result = await membersManagementApi.getFingerprintEnrollment(member.id);
      applyResult(result);
    } catch (refreshError) {
      setError(refreshError);
    } finally {
      setBusy(null);
    }
  };

  const handleCancel = async () => {
    if (busy) return;
    setBusy("cancel");
    setError(null);
    try {
      const result = await membersManagementApi.cancelFingerprintEnrollment(member.id);
      setConfirmCancel(false);
      onCancelled?.(result);
    } catch (cancelError) {
      setError(cancelError);
    } finally {
      setBusy(null);
    }
  };

  const handleClose = () => {
    if (!canClose) return;
    onClose?.();
  };

  const footer =
    success || session?.status === "cancelled" ? (
      <AppButton variant="primary" onClick={handleClose}>
        {success ? "Done" : "Close"}
      </AppButton>
    ) : (
      <>
        <AppButton
          variant="ghost"
          onClick={() => setConfirmCancel(true)}
          disabled={Boolean(busy)}
        >
          Cancel registration
        </AppButton>
        {isActive ? (
          <AppButton variant="outline" onClick={handleRefresh} loading={busy === "refresh"}>
            <RefreshIcon className="h-4 w-4" />
            Refresh status
          </AppButton>
        ) : (
          <AppButton variant="primary" onClick={handleStart} loading={busy === "start"}>
            {session && ["failed", "expired"].includes(session.status)
              ? "Retry enrollment"
              : "Start enrollment"}
          </AppButton>
        )}
      </>
    );

  return (
    <>
      <Modal
        open
        onClose={handleClose}
        dismissible={canClose}
        title={success ? "Member activated" : "Fingerprint enrollment"}
        description={`${member?.name || "Member"} · ${member?.login_id || member?.email || ""}`}
        size="lg"
        footer={footer}
      >
        <div className="lab-form-stack">
          {error && (
            <Alert tone="danger" title="Enrollment status could not be updated">
              {error.message}
            </Alert>
          )}

          {success ? (
            <div className="lab-enrollment-success" role="status">
              <span className="lab-enrollment-success__icon">
                <CheckIcon className="h-6 w-6" />
              </span>
              <div>
                <h3>Fingerprint enrollment complete</h3>
                <p>
                  {member?.name || "The member"} has been activated. The login and logout
                  readers can now recognize this member.
                </p>
              </div>
            </div>
          ) : (
            <>
              <div className="lab-enrollment-status">
                <div className="lab-enrollment-status__icon">
                  {isActive ? (
                    <Spinner className="h-5 w-5" />
                  ) : session?.status === "failed" || session?.status === "expired" ? (
                    <XCircleIcon className="h-5 w-5" />
                  ) : (
                    <ShieldIcon className="h-5 w-5" />
                  )}
                </div>
                <div>
                  <div className="lab-enrollment-status__title">
                    <h3>{copy.title}</h3>
                    {session?.status && (
                      <Badge tone={isActive ? "warning" : isTerminal ? "danger" : "neutral"}>
                        {session.status.replaceAll("_", " ")}
                      </Badge>
                    )}
                  </div>
                  <p>{copy.description}</p>
                </div>
              </div>

              {session?.status === "failed" && (
                <Alert tone="danger" title="What happened?">
                  {session.error_message ||
                    "The reader reported an enrollment failure. Retry after checking the sensor."}
                  {session.cleanup_required && (
                    <span className="lab-enrollment-cleanup-note">
                      The sensor slot will be cleaned up automatically before the next attempt.
                    </span>
                  )}
                </Alert>
              )}

              {session?.status === "expired" && (
                <Alert tone="warning" title="Enrollment window elapsed">
                  The member account remains pending. Start a new attempt when the member is
                  available.
                </Alert>
              )}

              <EnrollmentSteps status={session?.status} />

              <DefinitionList
                items={[
                  {
                    label: "Enrollment reader",
                    value: session?.device?.name || "Any active enrollment reader",
                  },
                  {
                    label: "Attempts",
                    value: session?.attempts ?? 0,
                  },
                  {
                    label: "Session expires",
                    value: session?.expires_at ? formatDate(session.expires_at) : "When started",
                  },
                  {
                    label: "Fingerprint slot",
                    value: session?.fingerprint_slot
                      ? `Slot ${session.fingerprint_slot}`
                      : "Assigned by the sensor",
                  },
                ]}
              />

              <div className="lab-enrollment-note">
                <ClockIcon className="h-4 w-4" />
                <p>
                  The member is not activated until both scans match and the backend finalizes
                  the fingerprint slot. Raw fingerprint images and templates never leave the
                  sensor.
                </p>
              </div>
            </>
          )}
        </div>
      </Modal>

      <ConfirmDialog
        open={confirmCancel}
        onClose={() => setConfirmCancel(false)}
        onConfirm={handleCancel}
        title="Cancel this registration?"
        description={`${member?.name || "This member"} will not be activated. The pending account will be deleted and any sensor slot used during this attempt will be cleaned up.`}
        confirmLabel="Cancel registration"
        destructive
        loading={busy === "cancel"}
      />
    </>
  );
}
