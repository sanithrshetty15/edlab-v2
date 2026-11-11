import { useEffect, useRef, useState } from "react";
import wordmarkLight from "../assets/images/edwins-lab-wordmark-light.png";
import { useAuth } from "../auth/context";
import { AppLink } from "../components/app/Link";
import {
  ArrowRightIcon,
  BookIcon,
  ChartIcon,
  BoxIcon,
  CalendarIcon,
  CheckSquareIcon,
  ChevronDownIcon,
  CloseIcon,
  DashboardIcon,
  ExternalIcon,
  LogoutIcon,
  MegaphoneIcon,
  MenuIcon,
  MessageIcon,
  SettingsIcon,
  UserIcon,
  UsersIcon,
  ClockIcon,
  BellIcon,
  AlertIcon,
} from "../components/app/AppIcons";
import { Avatar, Badge } from "../components/app/Ui";
import { notificationsApi } from "../api/services";
import { useApiResource } from "../hooks/useApiResource";
import { useNetworkStatus } from "../hooks/useNetworkStatus";
import { navigate, usePathname } from "../utils/router";

const MEMBER_GROUPS = [
  {
    label: "Workspace",
    items: [
      { to: "/app/dashboard", label: "Dashboard", icon: DashboardIcon },
      { to: "/app/announcements", label: "Announcements", icon: MegaphoneIcon },
      { to: "/app/events", label: "Events", icon: CalendarIcon },
      { to: "/app/my-attendance", label: "My attendance", icon: ClockIcon },
      { to: "/app/my-tasks", label: "My tasks", icon: CheckSquareIcon },
      { to: "/app/resources", label: "Lab resources", icon: BoxIcon },
      { to: "/app/messages", label: "Messages", icon: MessageIcon },
      { to: "/app/notifications", label: "Notifications", icon: BellIcon },
    ],
  },
  {
    label: "Account",
    items: [
      { to: "/app/profile", label: "Profile", icon: UserIcon },
      { to: "/app/settings", label: "Settings", icon: SettingsIcon },
    ],
  },
];

const ADMIN_GROUPS = [
  {
    label: "Overview",
    items: [
      { to: "/app/dashboard", label: "Dashboard", icon: DashboardIcon },
      { to: "/app/members", label: "Members", icon: UsersIcon },
      { to: "/app/attendance", label: "Attendance", icon: ClockIcon },
    ],
  },
  {
    label: "Programme",
    items: [
      { to: "/app/announcements", label: "Announcements", icon: MegaphoneIcon },
      { to: "/app/events", label: "Events", icon: CalendarIcon },
      { to: "/app/tasks", label: "Tasks", icon: CheckSquareIcon },
      { to: "/app/messages", label: "Messages", icon: MessageIcon },
    ],
  },
  {
    label: "Operations",
    items: [
      { to: "/app/inventory", label: "Inventory", icon: BoxIcon },
      { to: "/app/borrowing", label: "Borrowing", icon: BookIcon },
      { to: "/app/analytics", label: "Analytics", icon: ChartIcon },
    ],
  },
  {
    label: "Account",
    items: [
      { to: "/app/notifications", label: "Notifications", icon: BellIcon },
      { to: "/app/settings", label: "Settings", icon: SettingsIcon },
    ],
  },
];

const PAGE_TITLES = {
  "/app/dashboard": "Dashboard",
  "/app/profile": "Profile",
  "/app/announcements": "Announcements",
  "/app/events": "Events",
  "/app/messages": "Messages",
  "/app/my-attendance": "My attendance",
  "/app/my-tasks": "My tasks",
  "/app/resources": "Lab resources",
  "/app/notifications": "Notifications",
  "/app/settings": "Settings",
  "/app/members": "Members",
  "/app/attendance": "Attendance",
  "/app/tasks": "Tasks",
  "/app/inventory": "Inventory",
  "/app/borrowing": "Borrowing",
  "/app/analytics": "Analytics",
};

function isActive(pathname, to) {
  if (to === "/app/dashboard") return pathname === to;
  return pathname === to || pathname.startsWith(`${to}/`);
}

function SidebarLink({ item, pathname, onNavigate }) {
  const Icon = item.icon;
  const active = isActive(pathname, item.to);
  return (
    <AppLink
      to={item.to}
      className={`lab-nav-link ${active ? "is-active" : ""}`}
      aria-current={active ? "page" : undefined}
      onClick={onNavigate}
    >
      <Icon className="lab-nav-link__icon" />
      <span>{item.label}</span>
      {active && <span className="lab-nav-link__rail" aria-hidden="true" />}
    </AppLink>
  );
}

export default function AppLayout({ children }) {
  const auth = useAuth();
  const pathname = usePathname();
  const [mobileOpen, setMobileOpen] = useState(false);
  const [userMenuOpen, setUserMenuOpen] = useState(false);
  const [loggingOut, setLoggingOut] = useState(false);
  const userMenuRef = useRef(null);
  const menuButtonRef = useRef(null);

  const groups = auth.isAdmin ? ADMIN_GROUPS : MEMBER_GROUPS;
  const displayName =
    auth.user?.admin?.name || auth.user?.member?.name || auth.user?.name || auth.user?.email || "Edwins Lab";
  const roleLabel = auth.isAdmin
    ? auth.user?.admin?.is_primary
      ? "Primary admin"
      : "Administrator"
    : "Member";
  const pageTitle = PAGE_TITLES[pathname] || "Edwins Lab";
  const network = useNetworkStatus();
  const unreadResource = useApiResource(
    () => notificationsApi.unreadCount(),
    "layout-unread-count",
  );
  const unreadCount = unreadResource.data?.unread_count ?? 0;
  const reloadUnread = unreadResource.reload;

  useEffect(() => {
    const refresh = () => reloadUnread();
    const interval = window.setInterval(refresh, 60000);
    window.addEventListener("lab:notifications-changed", refresh);
    return () => {
      window.clearInterval(interval);
      window.removeEventListener("lab:notifications-changed", refresh);
    };
  }, [reloadUnread]);

  useEffect(() => {
    document.body.classList.add("app-mode");
    return () => document.body.classList.remove("app-mode");
  }, []);

  useEffect(() => {
    if (!mobileOpen) return undefined;
    const onKeyDown = (event) => {
      if (event.key === "Escape") setMobileOpen(false);
    };
    window.addEventListener("keydown", onKeyDown);
    const previous = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    return () => {
      window.removeEventListener("keydown", onKeyDown);
      document.body.style.overflow = previous;
    };
  }, [mobileOpen]);

  useEffect(() => {
    if (!userMenuOpen) return undefined;
    const onPointerDown = (event) => {
      if (!userMenuRef.current?.contains(event.target)) setUserMenuOpen(false);
    };
    const onKeyDown = (event) => {
      if (event.key === "Escape") setUserMenuOpen(false);
    };
    document.addEventListener("pointerdown", onPointerDown);
    document.addEventListener("keydown", onKeyDown);
    return () => {
      document.removeEventListener("pointerdown", onPointerDown);
      document.removeEventListener("keydown", onKeyDown);
    };
  }, [userMenuOpen]);

  useEffect(() => {
    const base = "Edwins Lab Portal";
    document.title = `${pageTitle} — ${base}`;
  }, [pageTitle]);

  const handleLogout = async () => {
    setLoggingOut(true);
    await auth.logout();
    setLoggingOut(false);
    navigate("/login", { replace: true });
  };

  return (
    <div className="lab-app">
      <a href="#app-main" className="lab-skip-link">
        Skip to content
      </a>

      <div
        className={`lab-sidebar-backdrop ${mobileOpen ? "is-visible" : ""}`}
        aria-hidden={!mobileOpen}
        onClick={() => setMobileOpen(false)}
      />

      <aside
        id="app-sidebar"
        className={`lab-sidebar ${mobileOpen ? "is-open" : ""}`}
        aria-label="Application navigation"
      >
        <div className="lab-sidebar__brand">
          <AppLink to="/app/dashboard" className="lab-sidebar__wordmark" aria-label="Edwins Lab dashboard">
            <img src={wordmarkLight} alt="Edwins Lab" />
          </AppLink>
          <span className="lab-sidebar__kicker">{auth.isAdmin ? "Admin console" : "Member portal"}</span>
          <button
            type="button"
            className="lab-sidebar__close"
            onClick={() => setMobileOpen(false)}
            aria-label="Close navigation"
          >
            <CloseIcon className="h-5 w-5" />
          </button>
        </div>

        <nav className="lab-sidebar__nav">
          {groups.map((group) => (
            <div key={group.label} className="lab-nav-group">
              <p className="lab-nav-group__label">{group.label}</p>
              {group.items.map((item) => (
                <SidebarLink
                  key={item.to}
                  item={item}
                  pathname={pathname}
                  onNavigate={() => setMobileOpen(false)}
                />
              ))}
            </div>
          ))}
        </nav>

        <div className="lab-sidebar__footer">
          <div className="lab-sidebar__session">
            <Avatar name={displayName} size="sm" />
            <div>
              <p className="lab-sidebar__user">{displayName}</p>
              <p className="lab-sidebar__role">{roleLabel}</p>
            </div>
          </div>
          <AppLink to="/" className="lab-sidebar__site-link">
            <ExternalIcon className="h-3.5 w-3.5" />
            View public site
          </AppLink>
        </div>
      </aside>

      <div className="lab-shell">
        <header className="lab-topbar">
          <div className="lab-topbar__start">
            <button
              ref={menuButtonRef}
              type="button"
              className="lab-icon-button lab-topbar__menu"
              onClick={() => setMobileOpen(true)}
              aria-label="Open navigation"
              aria-expanded={mobileOpen}
              aria-controls="app-sidebar"
            >
              <MenuIcon className="h-5 w-5" />
            </button>
            <div className="lab-topbar__crumb">
              <span>{auth.isAdmin ? "Admin" : "Member"}</span>
              <ArrowRightIcon className="h-3 w-3" />
              <strong>{pageTitle}</strong>
            </div>
          </div>

          <div className="lab-topbar__end">
            <Badge tone={auth.isAdmin ? "accent" : "info"} dot>
              {roleLabel}
            </Badge>

            <AppLink
              to="/app/notifications"
              className="lab-topbar__bell"
              aria-label={
                unreadCount
                  ? `Notifications, ${unreadCount} unread`
                  : "Notifications"
              }
              title="Notifications"
            >
              <BellIcon className="h-4 w-4" />
              {unreadCount > 0 && (
                <span className="lab-topbar__bell-count">
                  {unreadCount > 9 ? "9+" : unreadCount}
                </span>
              )}
            </AppLink>

            <div className="lab-user-menu" ref={userMenuRef}>
              <button
                type="button"
                className="lab-user-trigger"
                onClick={() => setUserMenuOpen((open) => !open)}
                aria-haspopup="menu"
                aria-expanded={userMenuOpen}
              >
                <Avatar name={displayName} size="sm" />
                <span className="lab-user-trigger__copy">
                  <span>{displayName}</span>
                  <small>{auth.user?.email}</small>
                </span>
                <ChevronDownIcon className={`h-3.5 w-3.5 ${userMenuOpen ? "is-rotated" : ""}`} />
              </button>

              {userMenuOpen && (
                <div className="lab-user-menu__panel" role="menu">
                  <div className="lab-user-menu__head">
                    <Avatar name={displayName} size="md" />
                    <div>
                      <p>{displayName}</p>
                      <span>{auth.user?.email}</span>
                    </div>
                  </div>
                  <AppLink to="/app/profile" className="lab-user-menu__item" role="menuitem">
                    <UserIcon className="h-4 w-4" />
                    Profile
                  </AppLink>
                  <AppLink to="/app/notifications" className="lab-user-menu__item" role="menuitem">
                    <BellIcon className="h-4 w-4" />
                    Notifications
                    {unreadCount > 0 && <span className="lab-user-menu__count">{unreadCount}</span>}
                  </AppLink>
                  <AppLink to="/app/settings" className="lab-user-menu__item" role="menuitem">
                    <SettingsIcon className="h-4 w-4" />
                    Account settings
                  </AppLink>
                  <AppLink to="/" className="lab-user-menu__item" role="menuitem">
                    <ExternalIcon className="h-4 w-4" />
                    Public website
                  </AppLink>
                  <div className="lab-user-menu__divider" />
                  <button
                    type="button"
                    className="lab-user-menu__item lab-user-menu__item--danger"
                    onClick={handleLogout}
                    role="menuitem"
                    disabled={loggingOut}
                  >
                    <LogoutIcon className="h-4 w-4" />
                    {loggingOut ? "Signing out…" : "Sign out"}
                  </button>
                </div>
              )}
            </div>
          </div>
        </header>

        {!network.connected && (
          <div className="lab-offline-banner" role="status">
            <AlertIcon className="h-4 w-4" />
            <div>
              <strong>You are offline</strong>
              <span>Saved screens stay available. New data will load when the connection returns.</span>
            </div>
          </div>
        )}

        <main id="app-main" className="lab-main" tabIndex={-1}>
          {children}
        </main>
      </div>
    </div>
  );
}
