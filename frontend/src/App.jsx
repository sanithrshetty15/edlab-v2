import { useEffect } from "react";
import "./App.css";
import Intro from "./components/intro/Intro";
import Navbar from "./components/layout/Navbar";
import ExplorePage from "./pages/ExplorePage";
import { Hero, About, Linux, Testimonials, Download, Footer } from "./sections";
import { refreshScrollTriggers } from "./utils/animations";
import { usePathname } from "./utils/router";

const HOME_TITLE = "Edwins Lab — College Open Source Technology Lab";
const EXPLORE_TITLE = "Explore the Lab — Edwins Lab";

export default function App() {
  const pathname = usePathname();
  const isExplore = pathname === "/explore";

  useEffect(() => refreshScrollTriggers(), [pathname]);

  useEffect(() => {
    document.title = isExplore ? EXPLORE_TITLE : HOME_TITLE;
  }, [isExplore]);

  if (isExplore) {
    return <ExplorePage />;
  }

  return (
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
