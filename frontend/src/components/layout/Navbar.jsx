import { useCallback, useEffect, useRef, useState } from "react";
import wordmark from "../../assets/images/edwins-lab-wordmark.png";
import { navigation, site } from "../../content/labContent";
import { Button } from "../common";
import { Download, Menu } from "../common/Icons";
import MenuOverlay from "./MenuOverlay";

export default function Navbar() {
  const [open, setOpen] = useState(false);
  const [scrolled, setScrolled] = useState(false);
  const [activeSection, setActiveSection] = useState("");
  const menuButtonRef = useRef(null);

  const closeMenu = useCallback(() => setOpen(false), []);

  useEffect(() => {
    const onScroll = () => {
      setScrolled(window.scrollY > 12);
      if (window.scrollY < 180) setActiveSection("");
    };

    onScroll();
    window.addEventListener("scroll", onScroll, { passive: true });
    return () => window.removeEventListener("scroll", onScroll);
  }, []);

  useEffect(() => {
    const sections = navigation
      .map((item) => document.getElementById(item.href.slice(1)))
      .filter(Boolean);

    if (!sections.length) return undefined;

    const observer = new IntersectionObserver(
      (entries) => {
        const visible = entries
          .filter((entry) => entry.isIntersecting)
          .sort((a, b) => b.intersectionRatio - a.intersectionRatio);

        if (visible[0]) setActiveSection(visible[0].target.id);
      },
      { rootMargin: "-38% 0px -52% 0px", threshold: [0, 0.2, 0.5] },
    );

    sections.forEach((section) => observer.observe(section));

    return () => observer.disconnect();
  }, []);

  return (
    <>
      <header className="fixed inset-x-0 top-0 z-50 px-3 pt-3 sm:px-5 sm:pt-4">
        <nav
          aria-label="Primary"
          className={`nav-pill relative mx-auto flex max-w-[1240px] items-center justify-between gap-2 rounded-full border px-3 py-2.5 pl-5 transition-[background-color,border-color,box-shadow] duration-500 sm:px-3 sm:pl-5 ${
            scrolled
              ? "nav-pill--scrolled border-edwins-dark/10 bg-white/92 shadow-[0_12px_40px_rgba(28,28,28,0.09)] backdrop-blur-xl"
              : "border-transparent bg-white/80 shadow-[0_8px_30px_rgba(28,28,28,0.05)] backdrop-blur-md"
          }`}
        >
          <a
            href="#top"
            className="flex shrink-0 items-center"
            aria-label="Edwins Lab, back to top"
          >
            <img
              src={wordmark}
              alt="Edwins Lab"
              className="h-9 w-auto transition-opacity duration-300 hover:opacity-70 sm:h-10 lg:h-11"
            />
          </a>

          <div className="hidden lg:absolute lg:left-1/2 lg:top-1/2 lg:flex lg:-translate-x-1/2 lg:-translate-y-1/2 lg:items-center lg:gap-0.5">
            {navigation.map((item) => {
              const isActive = activeSection === item.href.slice(1);
              return (
                <a
                  key={item.label}
                  href={item.href}
                  aria-current={isActive ? "location" : undefined}
                  className={`nav-link relative rounded-full px-3 py-2 text-sm font-medium transition-colors duration-300 ${
                    isActive
                      ? "text-edwins-dark"
                      : "text-edwins-dark/65 hover:text-edwins-dark"
                  }`}
                >
                  {item.label}
                  <span
                    className={`absolute bottom-0.5 left-1/2 h-0.5 w-0.5 -translate-x-1/2 rounded-full bg-maze-lime transition-opacity duration-300 ${
                      isActive ? "opacity-100" : "opacity-0"
                    }`}
                    aria-hidden="true"
                  />
                </a>
              );
            })}
          </div>

          <div className="flex items-center gap-1.5 sm:gap-2">
            <Button
              href="#download"
              variant="primary"
              size="sm"
              className="nav-download whitespace-nowrap"
            >
              <Download className="h-4 w-4" />
              <span className="hidden min-[375px]:inline">Download app</span>
              <span className="min-[375px]:hidden">App</span>
            </Button>
            <button
              ref={menuButtonRef}
              type="button"
              onClick={() => setOpen(true)}
              className="flex h-10 w-10 items-center justify-center rounded-full border border-edwins-dark/15 bg-edwins-dark text-maze-lime transition-[background-color,color,transform] duration-300 hover:bg-maze-lime hover:text-edwins-dark active:scale-95"
              aria-label="Open menu"
              aria-expanded={open}
              aria-controls="site-menu"
            >
              <Menu className="h-[1.125rem] w-[1.125rem]" />
            </button>
          </div>
        </nav>
      </header>

      <MenuOverlay
        open={open}
        onClose={closeMenu}
        links={navigation}
        email={site.email}
        menuButtonRef={menuButtonRef}
      />
    </>
  );
}
