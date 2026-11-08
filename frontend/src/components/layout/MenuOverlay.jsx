import { useEffect, useRef } from "react";
import { createPortal } from "react-dom";
import { gsap } from "../../utils/animations";
import { prefersReducedMotion } from "../../utils/reducedMotion";
import wordmarkLight from "../../assets/images/edwins-lab-wordmark-light.png";
import { Button } from "../common";
import { ArrowUpRight, Close, Mail } from "../common/Icons";

export default function MenuOverlay({
  open,
  onClose,
  links,
  email,
  menuButtonRef,
}) {
  const overlayRef = useRef(null);
  const closeRef = useRef(null);
  const hasOpenedRef = useRef(false);

  useEffect(() => {
    const overlay = overlayRef.current;
    if (!overlay) return undefined;

    if (!open) {
      if (hasOpenedRef.current) {
        if (prefersReducedMotion()) {
          gsap.set(overlay, { autoAlpha: 0 });
          overlay.style.visibility = "hidden";
          overlay.style.pointerEvents = "none";
          return undefined;
        }

        const closeTimeline = gsap.to(overlay, {
          autoAlpha: 0,
          duration: 0.25,
          ease: "power2.in",
          onComplete: () => {
            overlay.style.visibility = "hidden";
            overlay.style.pointerEvents = "none";
          },
        });
        return () => closeTimeline.kill();
      }

      overlay.style.visibility = "hidden";
      overlay.style.pointerEvents = "none";
      return undefined;
    }

    hasOpenedRef.current = true;
    overlay.style.visibility = "visible";
    overlay.style.pointerEvents = "auto";

    const previousOverflow = document.body.style.overflow;
    const menuButton = menuButtonRef?.current;
    const root = document.getElementById("root");
    document.body.style.overflow = "hidden";
    root?.setAttribute("inert", "");

    const focusTimer = window.setTimeout(
      () => closeRef.current?.focus(),
      80,
    );

    const linksEl = overlay.querySelectorAll("[data-menu-link]");
    const asideEl = overlay.querySelectorAll("[data-menu-aside] > *");

    let timeline;

    if (prefersReducedMotion()) {
      gsap.set(overlay, { autoAlpha: 1 });
      gsap.set([...linksEl, ...asideEl], { autoAlpha: 1, y: 0 });
    } else {
      timeline = gsap.timeline({
        defaults: { ease: "power3.out" },
      });

      timeline
        .fromTo(
          overlay,
          { opacity: 0 },
          { opacity: 1, duration: 0.35, ease: "power2.out" },
        )
        .fromTo(
          linksEl,
          { autoAlpha: 0, y: 34 },
          { autoAlpha: 1, y: 0, duration: 0.6, stagger: 0.06 },
          "-=0.12",
        )
        .fromTo(
          asideEl,
          { autoAlpha: 0, y: 22 },
          { autoAlpha: 1, y: 0, duration: 0.5, stagger: 0.05 },
          "-=0.42",
        );
    }

    const focusableSelector =
      'a[href], button:not([disabled]), [tabindex]:not([tabindex="-1"])';

    const onKeyDown = (event) => {
      if (event.key === "Escape") {
        event.preventDefault();
        onClose();
        return;
      }

      if (event.key !== "Tab") return;

      const focusable = Array.from(
        overlay.querySelectorAll(focusableSelector),
      ).filter((element) => element.offsetParent !== null);
      if (!focusable.length) return;

      const first = focusable[0];
      const last = focusable[focusable.length - 1];

      if (event.shiftKey && document.activeElement === first) {
        event.preventDefault();
        last.focus();
      } else if (!event.shiftKey && document.activeElement === last) {
        event.preventDefault();
        first.focus();
      }
    };

    document.addEventListener("keydown", onKeyDown);

    return () => {
      window.clearTimeout(focusTimer);
      timeline?.kill();
      document.body.style.overflow = previousOverflow;
      document.removeEventListener("keydown", onKeyDown);
      root?.removeAttribute("inert");
      menuButton?.focus();
    };
  }, [menuButtonRef, onClose, open]);

  if (typeof document === "undefined") return null;

  return createPortal(
    <div
      ref={overlayRef}
      id="site-menu"
      className="menu-overlay invisible fixed inset-0 z-[90] overflow-y-auto bg-edwins-dark text-white opacity-0"
      role="dialog"
      aria-modal="true"
      aria-label="Site menu"
      aria-hidden={!open}
      inert={!open}
    >
      <div className="menu-overlay__grid" aria-hidden="true" />
      <div className="relative z-10 flex min-h-full flex-col">
        <div className="flex items-center justify-between px-5 pb-6 pt-5 sm:px-8 lg:px-12 lg:pt-7">
          <a
            href="#top"
            onClick={onClose}
            className="inline-flex items-center gap-2"
            aria-label="Edwins Lab home"
          >
            <img
              src={wordmarkLight}
              alt="Edwins Lab"
              className="h-6 w-auto sm:h-7"
            />
          </a>
          <button
            ref={closeRef}
            type="button"
            onClick={onClose}
            className="flex h-11 w-11 items-center justify-center rounded-full border border-white/15 text-white transition-colors duration-300 hover:border-maze-lime hover:bg-maze-lime hover:text-edwins-dark"
            aria-label="Close menu"
          >
            <Close className="h-5 w-5" />
          </button>
        </div>

        <div className="flex flex-1 items-center px-5 py-8 sm:px-8 lg:px-12">
          <div className="grid w-full gap-12 lg:grid-cols-[1.25fr_0.75fr] lg:items-end">
            <nav aria-label="Menu" className="flex flex-col">
              <p
                className="mb-6 font-orbitron text-[0.625rem] font-semibold uppercase tracking-[0.24em] text-white/55"
                data-menu-aside
              >
                Navigate
              </p>
              {links.map((link, index) => (
                <a
                  key={link.label}
                  href={link.href}
                  onClick={onClose}
                  data-menu-link
                  className="menu-link group flex items-baseline gap-4 border-b border-white/10 py-3.5 sm:py-4"
                >
                  <span className="w-7 font-poppins text-[0.6875rem] font-medium tabular-nums text-white/45 transition-colors group-hover:text-maze-lime">
                    {String(index + 1).padStart(2, "0")}
                  </span>
                  <span className="font-orbitron text-[clamp(1.75rem,5vw,3.75rem)] font-medium leading-none tracking-[-0.03em] text-white transition-colors duration-300 group-hover:text-maze-lime">
                    {link.label}
                  </span>
                  <ArrowUpRight className="ml-auto h-5 w-5 self-center text-white/25 transition-all duration-300 group-hover:-translate-y-1 group-hover:translate-x-1 group-hover:text-maze-lime" />
                </a>
              ))}
            </nav>

            <div className="flex flex-col gap-8 lg:pb-4" data-menu-aside>
              <div>
                <p className="font-orbitron text-[0.625rem] font-semibold uppercase tracking-[0.24em] text-white/55">
                  The lab
                </p>
                <p className="mt-4 max-w-[28ch] text-sm leading-7 text-white/60">
                  A welcoming point for people who learn by exploring Linux,
                  sharing what they discover and building in the open.
                </p>
              </div>

              <a
                href={`mailto:${email}`}
                className="group inline-flex items-center gap-3 text-sm font-medium text-white transition-colors hover:text-maze-lime"
              >
                <span className="flex h-10 w-10 items-center justify-center rounded-full border border-white/15 transition-colors group-hover:border-maze-lime">
                  <Mail className="h-4 w-4" />
                </span>
                {email}
              </a>

              <Button
                href="#download"
                variant="primary"
                size="md"
                onClick={onClose}
                className="self-start"
              >
                Download app
                <ArrowUpRight className="h-4 w-4" />
              </Button>
            </div>
          </div>
        </div>

        <div className="flex flex-wrap items-center justify-between gap-3 border-t border-white/10 px-5 py-5 font-poppins text-[0.6875rem] uppercase tracking-[0.16em] text-white/45 sm:px-8 lg:px-12">
          <span>Edwins Lab</span>
          <span>Learn. Build. Innovate.</span>
        </div>
      </div>
    </div>,
    document.body,
  );
}
