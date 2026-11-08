import { useLayoutEffect, useRef } from "react";
import { Container } from "../common";
import { ArrowUpRight } from "../common/Icons";
import wordmarkLight from "../../assets/images/edwins-lab-wordmark-light.png";
import { exploreFooter } from "../../content/exploreContent";
import { site } from "../../content/labContent";
import { useReducedMotion } from "../../hooks/useMediaQuery";
import { gsap } from "../../utils/animations";
import { navigate } from "../../utils/router";
import InteractiveContourField from "../shared/InteractiveContourField";

export default function ExploreFooter() {
  const footerRef = useRef(null);
  const reducedMotion = useReducedMotion();
  const year = new Date().getFullYear();

  useLayoutEffect(() => {
    const footer = footerRef.current;
    if (!footer || reducedMotion) return undefined;

    const ctx = gsap.context(() => {
      gsap.set("[data-interactive-contours]", { autoAlpha: 0 });
      gsap.set("[data-footer-brand]", { autoAlpha: 0, scale: 0.96 });
      gsap.set("[data-footer-line]", {
        scaleX: 0,
        transformOrigin: "left center",
      });
      gsap.set("[data-footer-wordmark]", { yPercent: 105 });
      gsap.set("[data-footer-contact]", { autoAlpha: 0, y: 24 });
      gsap.set("[data-footer-reveal]", { autoAlpha: 0, y: 24 });
      gsap.set("[data-footer-legal]", { autoAlpha: 0, y: 12 });

      const timeline = gsap.timeline({
        scrollTrigger: {
          trigger: footer,
          start: "top 80%",
          once: true,
        },
      });

      timeline
        .to(
          "[data-interactive-contours]",
          { autoAlpha: 1, duration: 1.2, ease: "power2.out" },
          0,
        )
        .to(
          "[data-footer-brand]",
          { autoAlpha: 1, scale: 1, duration: 0.8, ease: "power3.out" },
          0.05,
        )
        .to(
          "[data-footer-line]",
          { scaleX: 1, duration: 0.8, ease: "power3.out" },
          0.1,
        )
        .to(
          "[data-footer-wordmark]",
          { yPercent: 0, duration: 1.1, ease: "power4.out" },
          0.35,
        )
        .to(
          "[data-footer-contact]",
          {
            autoAlpha: 1,
            y: 0,
            duration: 0.7,
            stagger: 0.08,
            ease: "power3.out",
          },
          0.75,
        )
        .to(
          "[data-footer-reveal]",
          {
            autoAlpha: 1,
            y: 0,
            duration: 0.6,
            stagger: 0.08,
            ease: "power3.out",
          },
          1.15,
        )
        .to(
          "[data-footer-legal]",
          { autoAlpha: 1, y: 0, duration: 0.5, ease: "power2.out" },
          1.4,
        );
    }, footer);

    return () => ctx.revert();
  }, [reducedMotion]);

  const handleHome = (event) => {
    if (
      event.metaKey ||
      event.ctrlKey ||
      event.shiftKey ||
      event.altKey
    ) {
      return;
    }

    event.preventDefault();
    navigate("/");
  };

  return (
    <footer
      ref={footerRef}
      id="contact"
      className="explore-footer"
      aria-labelledby="explore-footer-title"
    >
      <InteractiveContourField />
      <div className="explore-footer__glow" aria-hidden="true" />

      <Container className="explore-footer__inner">
        <div className="explore-footer__top">
          <a
            href="/"
            onClick={handleHome}
            className="explore-footer__brand"
            data-footer-brand
            aria-label="Edwins Lab, back to home"
          >
            <img
              src={wordmarkLight}
              alt="Edwins Lab"
              width="554"
              height="252"
              loading="lazy"
              className="h-9 w-auto sm:h-11"
            />
          </a>

          <nav
            aria-label="Explore footer"
            className="explore-footer__nav"
            data-footer-contact
          >
            {exploreFooter.nav.map((item) => (
              <a key={item.label} href={item.href}>
                {item.label}
              </a>
            ))}
            <a href={`mailto:${exploreFooter.email}`}>Contact</a>
          </nav>
        </div>

        <div className="explore-footer__line" data-footer-line aria-hidden="true" />

        <div className="explore-footer__statement" data-footer-contact>
          <span className="explore-eyebrow">{exploreFooter.eyebrow}</span>
          <p>{exploreFooter.statement}</p>
        </div>

        <div className="explore-footer__wordmark-mask">
          <h2
            id="explore-footer-title"
            className="explore-footer__wordmark"
            data-footer-wordmark
          >
            EDWIN&apos;S LAB<span aria-hidden="true">.</span>
          </h2>
        </div>

        <div className="explore-footer__bottom">
          <div data-footer-contact>
            <p className="explore-footer__closing">
              {exploreFooter.closing}
            </p>
            <a
              href={`mailto:${exploreFooter.email}`}
              className="explore-footer__email"
            >
              {exploreFooter.email}
              <ArrowUpRight className="h-4 w-4" />
            </a>
          </div>

          <div className="explore-footer__back" data-footer-reveal>
            <span>{exploreFooter.copyright}</span>
            <a href="/" onClick={handleHome}>
              {exploreFooter.backToHome}
              <ArrowUpRight className="h-3.5 w-3.5" />
            </a>
            <a href="#explore-top">Back to top</a>
          </div>
        </div>

        <div className="explore-footer__legal" data-footer-legal>
          <p>
            © {year} {site.name}
          </p>
          <p>{site.tagline}</p>
          <p>Built in the open.</p>
        </div>
      </Container>
    </footer>
  );
}
