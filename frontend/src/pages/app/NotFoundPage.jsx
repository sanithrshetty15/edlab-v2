import { AppLink } from "../../components/app/Link";
import { Panel } from "../../components/app/Ui";
import { ArrowLeftIcon, DashboardIcon } from "../../components/app/AppIcons";

export default function NotFoundPage() {
  return (
    <Panel>
      <div className="lab-not-found">
        <p className="lab-eyebrow">404</p>
        <h1>This page does not exist.</h1>
        <p className="lab-muted">
          The page may have moved, or the address may be incorrect.
        </p>
        <div className="lab-not-found__actions">
          <AppLink to="/app/dashboard" className="lab-button lab-button--primary lab-button--md">
            <DashboardIcon className="h-4 w-4" />
            Go to dashboard
          </AppLink>
          <AppLink to="/" className="lab-button lab-button--outline lab-button--md">
            <ArrowLeftIcon className="h-4 w-4" />
            Public site
          </AppLink>
        </div>
      </div>
    </Panel>
  );
}
