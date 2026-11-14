import { CheckIcon } from "../../../components/app/AppIcons";

const REGISTRATION_STEPS = [
  { id: "details", label: "Basic details" },
  { id: "fingerprint", label: "Fingerprint enrollment" },
  { id: "complete", label: "Registration complete" },
];

export default function MemberRegistrationProgress({ current = 1 }) {
  return (
    <ol className="lab-registration-progress" aria-label="Member registration progress">
      {REGISTRATION_STEPS.map((step, index) => {
        const number = index + 1;
        const state = number < current ? "done" : number === current ? "current" : "upcoming";
        return (
          <li key={step.id} className={`lab-registration-progress__step is-${state}`}>
            <span className="lab-registration-progress__index" aria-hidden="true">
              {state === "done" ? <CheckIcon className="h-3.5 w-3.5" /> : number}
            </span>
            <span>{step.label}</span>
          </li>
        );
      })}
    </ol>
  );
}
