import { useEffect } from "react";
import "./app.css";
import { AuthProvider } from "../auth/AuthContext";
import { useAuth } from "../auth/context";
import AppLayout from "./AppLayout";
import { PageLoading, Panel, AppButton } from "../components/app/Ui";
import { AppLink } from "../components/app/Link";
import LoginPage from "../pages/LoginPage";
import ForgotPasswordPage from "../pages/ForgotPasswordPage";
import PasswordChangePage from "../pages/app/PasswordChangePage";
import DashboardPage from "../pages/app/DashboardPage";
import ProfilePage from "../pages/app/ProfilePage";
import AnnouncementsPage from "../pages/app/AnnouncementsPage";
import EventsPage from "../pages/app/EventsPage";
import MessagesPage from "../pages/app/MessagesPage";
import MemberAttendancePage from "../pages/app/MemberAttendancePage";
import MemberTasksPage from "../pages/app/MemberTasksPage";
import NotificationsPage from "../pages/app/NotificationsPage";
import MemberResourcesPage from "../pages/app/MemberResourcesPage";
import SettingsPage from "../pages/app/SettingsPage";
import MembersPage from "../pages/app/admin/MembersPage";
import AttendancePage from "../pages/app/admin/AttendancePage";
import LiveAttendancePage from "../pages/app/admin/LiveAttendancePage";
import TasksPage from "../pages/app/admin/TasksPage";
import InventoryPage from "../pages/app/admin/InventoryPage";
import BorrowingPage from "../pages/app/admin/BorrowingPage";
import AnalyticsPage from "../pages/app/admin/AnalyticsPage";
import DevicesPage from "../pages/app/admin/DevicesPage";
import NotFoundPage from "../pages/app/NotFoundPage";
import { navigate, usePathname } from "../utils/router";

const ADMIN_ROUTES = new Set([
  "/app/members",
  "/app/attendance",
  "/app/live-attendance",
  "/app/tasks",
  "/app/inventory",
  "/app/borrowing",
  "/app/analytics",
  "/app/devices",
]);

function AccessDenied() {
  return (
    <Panel>
      <div className="lab-denied">
        <p className="lab-eyebrow">Access restricted</p>
        <h2>This area is for administrators.</h2>
        <p className="lab-muted">
          Your member account does not have permission to open this workspace.
        </p>
        <AppButton variant="primary" onClick={() => navigate("/app/dashboard")}>
          Return to dashboard
        </AppButton>
      </div>
    </Panel>
  );
}

function routeContent(pathname, isAdmin) {
  switch (pathname) {
    case "/app/dashboard":
      return <DashboardPage />;
    case "/app/profile":
      return <ProfilePage />;
    case "/app/announcements":
      return <AnnouncementsPage />;
    case "/app/events":
      return <EventsPage />;
    case "/app/messages":
      return <MessagesPage />;
    case "/app/my-attendance":
      return <MemberAttendancePage />;
    case "/app/my-tasks":
      return <MemberTasksPage />;
    case "/app/resources":
      return <MemberResourcesPage />;
    case "/app/notifications":
      return <NotificationsPage />;
    case "/app/settings":
      return <SettingsPage />;
    case "/app/members":
      return isAdmin ? <MembersPage /> : <AccessDenied />;
    case "/app/attendance":
      return isAdmin ? <AttendancePage /> : <AccessDenied />;
    case "/app/live-attendance":
      return isAdmin ? <LiveAttendancePage /> : <AccessDenied />;
    case "/app/tasks":
      return isAdmin ? <TasksPage /> : <AccessDenied />;
    case "/app/inventory":
      return isAdmin ? <InventoryPage /> : <AccessDenied />;
    case "/app/borrowing":
      return isAdmin ? <BorrowingPage /> : <AccessDenied />;
    case "/app/analytics":
      return isAdmin ? <AnalyticsPage /> : <AccessDenied />;
    case "/app/devices":
      return isAdmin ? <DevicesPage /> : <AccessDenied />;
    default:
      return <NotFoundPage />;
  }
}

function PortalRoutes() {
  const auth = useAuth();
  const pathname = usePathname();
  const isLogin = pathname === "/login";
  const isForgotPassword = pathname === "/forgot-password";
  const isPasswordRoute = pathname === "/app/change-password";

  useEffect(() => {
    if (auth.status === "loading") return;

    if (!auth.isAuthenticated) {
      if (!isLogin && !isForgotPassword) navigate("/login", { replace: true });
      return;
    }

    if (isLogin || isForgotPassword) {
      navigate("/app/dashboard", { replace: true });
      return;
    }

    if (auth.mustChangePassword && !isPasswordRoute) {
      navigate("/app/change-password", { replace: true });
      return;
    }

    if (!auth.mustChangePassword && isPasswordRoute) {
      navigate("/app/dashboard", { replace: true });
      return;
    }

    if (pathname === "/app") {
      navigate("/app/dashboard", { replace: true });
    }
  }, [
    auth.status,
    auth.isAuthenticated,
    auth.mustChangePassword,
    isForgotPassword,
    isLogin,
    isPasswordRoute,
    pathname,
  ]);

  if (auth.status === "loading") {
    return <PageLoading label="Restoring your Edwins Lab session" />;
  }

  if (!auth.isAuthenticated) {
    if (isLogin) return <LoginPage sessionError={auth.sessionError} expired={auth.expired} />;
    if (isForgotPassword) return <ForgotPasswordPage />;
    return <PageLoading label="Taking you to sign in" />;
  }

  if (isLogin || isForgotPassword || pathname === "/app" || (!isPasswordRoute && auth.mustChangePassword)) {
    return <PageLoading label="Opening your workspace" />;
  }

  if (isPasswordRoute) {
    return (
      <PasswordChangePage
        mandatory={auth.mustChangePassword}
        onComplete={() => navigate("/app/dashboard", { replace: true })}
      />
    );
  }

  if (ADMIN_ROUTES.has(pathname) && !auth.isAdmin) {
    return (
      <AppLayout key={pathname}>
        <AccessDenied />
      </AppLayout>
    );
  }

  return (
    <AppLayout key={pathname}>
      {routeContent(pathname, auth.isAdmin)}
      <footer className="lab-footer">
        <span>Edwins Lab member platform</span>
        <AppLink to="/">Back to the public site</AppLink>
      </footer>
    </AppLayout>
  );
}

export default function AppPortal() {
  return (
    <AuthProvider>
      <PortalRoutes />
    </AuthProvider>
  );
}
