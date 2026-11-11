import { lazy, Suspense, useEffect } from "react";
import { Capacitor } from "@capacitor/core";
import "./App.css";
import Intro from "./components/intro/Intro";
import Navbar from "./components/layout/Navbar";
import PwaUpdateBanner from "./components/pwa/PwaUpdateBanner";
import ExplorePage from "./pages/ExplorePage";
import { Hero, About, Linux, Testimonials, Download, Footer } from "./sections";
import { refreshScrollTriggers } from "./utils/animations";
import { navigate, usePathname } from "./utils/router";

const AppPortal = lazy(() => import("./app/AppPortal"));

const HOME_TITLE = "Edwins Lab — College Open Source Technology Lab";
const EXPLORE_TITLE = "Explore the Lab — Edwins Lab";

function PortalLoading() {
  return (
    <div className="flex min-h-screen items-center justify-center bg-edwins-dark px-6 text-white">
      <p className="font-orbitron text-[0.6875rem] font-semibold uppercase tracking-[0.2em] text-white/55">
        Opening Edwins Lab portal…
      </p>
    </div>
  );
}

export default function App() {
  const pathname = usePathname();
  const isNativeApp = Capacitor.isNativePlatform();
  const isExplore = pathname === "/explore";
  const isPortalPath =
    pathname === "/login" ||
    pathname === "/forgot-password" ||
    pathname === "/app" ||
    pathname.startsWith("/app/");
  const isRemovedPublicRoute = pathname === "/register";

  useEffect(() => refreshScrollTriggers(), [pathname]);

  useEffect(() => {
    if (isNativeApp) {
      if (!isPortalPath) navigate("/login", { replace: true });
      return;
    }

    if (isPortalPath || isRemovedPublicRoute) {
      navigate("/", { replace: true });
    }
  }, [isNativeApp, isPortalPath, isRemovedPublicRoute]);

  useEffect(() => {
    if (isNativeApp) return;

    if (isExplore) {
      document.title = EXPLORE_TITLE;
    } else if (!isPortalPath) {
      document.title = HOME_TITLE;
    }
  }, [isExplore, isNativeApp, isPortalPath]);

  let content;

  if (isNativeApp) {
    content = (
      <Suspense fallback={<PortalLoading />}>
        <AppPortal />
      </Suspense>
    );
  } else if (isExplore) {
    content = <ExplorePage />;
  } else {
    content = (
      <Intro>
        <div id="top" className="app bg-edwins-neutral">
          <a href="#main" className="skip-link">
            Skip to content
          </a>
          <Navbar />
          <main id="main" tabIndex={-1}>
            <Hero />
            <About />
            <Linux />
            <Testimonials />
            <Download />
          </main>
          <Footer />
        </div>
      </Intro>
    );
  }

  return (
    <>
      <PwaUpdateBanner />
      {content}
    </>
  );
}
