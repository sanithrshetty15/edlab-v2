import { useLayoutEffect, useRef } from "react";
import { Container, Button } from "../components/common";
import { ArrowDown, ArrowRight, ArrowUpRight } from "../components/common/Icons";
import TuxStage from "../components/tux/TuxStage";
import { hero } from "../content/labContent";
import { gsap, SITE_REVEAL_EVENT } from "../utils/animations";
import { prefersReducedMotion } from "../utils/reducedMotion";
import { navigate } from "../utils/router";

export default function Hero() {
  const sectionRef = useRef(null);

  useLayoutEffect(() => {
    if (prefersReducedMotion()) {
      // For reduced motion users, show all content immediately
      const section = sectionRef.current;
      if (section) {
        const select = gsap.utils.selector(section);
        const targets = [
          select("[data-hero-eyebrow]"),
          select("[data-hero-title] > span"),
          select("[data-hero-body]"),
          select("[data-hero-actions] > *"),
          select("[data-hero-chip]"),
          select("[data-hero-foot]"),
          select("[data-hero-stage-inner]"),
        ];
        targets.forEach(
          (t) =>
            t &&
            gsap.set(t, {
              autoAlpha: 1,
              y: 0,
              clearProps: "opacity,visibility,transform",
            }),
        );
      }
      return undefined;
    }

    const section = sectionRef.current;
    if (!section) return undefined;

    const select = gsap.utils.selector(section);
    const eyebrow = select("[data-hero-eyebrow]");
    const titleLines = select("[data-hero-title] > span");
    const body = select("[data-hero-body]");
    const actions = select("[data-hero-actions] > *");
    const chips = select("[data-hero-chip]");
    const foot = select("[data-hero-foot]");
    const stageFrame = select("[data-hero-stage]");
    const stageInner = select("[data-hero-stage-inner]");
    // The copy wrapper is display:contents on small screens, so the parallax
    // moves its children instead of the wrapper.
    const copyParts = select("[data-hero-copy] > *");
    const grid = select(".hero__grid");
    const halo = select(".tux-stage__halo");

    // Set initial VISIBLE state: the entrance only animates position/scale, so
    // content stays readable even if the intro never fires.
    const hiddenTargets = [
      eyebrow,
      titleLines,
      body,
      actions,
      chips,
      foot,
      stageInner,
    ];
    gsap.set(hiddenTargets, { autoAlpha: 1 });

    gsap.set(eyebrow, { y: 12 });
    gsap.set(titleLines, { y: 26 });
    gsap.set(body, { y: 18 });
    gsap.set(actions, { y: 16 });
    gsap.set(chips, { y: 12 });
    gsap.set(foot, { y: 10 });
    gsap.set(stageInner, { y: 22, scale: 0.97 });

    let timeline;
    let scrollTimeline;
    let fallbackTimer;
    let started = false;

    const start = () => {
      if (started) return;
      started = true;

      window.clearTimeout(fallbackTimer);
      window.removeEventListener(SITE_REVEAL_EVENT, start);

      timeline = gsap.timeline({
        defaults: { ease: "power3.out" },
        onComplete: () => {
          // One ScrollTrigger drives the whole hero exit: copy drifts up and
          // fades back, the Tux frame rises while easing down in scale, and the
          // grid/glow recede. The 3D model keeps its own internal scroll drift.
          scrollTimeline = gsap.timeline({
            defaults: { ease: "none" },
            scrollTrigger: {
              trigger: section,
              start: "top top",
              end: "bottom top",
              scrub: 0.7,
            },
          });

          scrollTimeline
            .to(copyParts, { yPercent: -6, autoAlpha: 0.6 }, 0)
            .to(stageFrame, { yPercent: -7, scale: 0.94 }, 0)
            .to(grid, { autoAlpha: 0.45 }, 0)
            .to(halo, { scale: 1.06 }, 0);
        },
      });

      timeline
        .to(eyebrow, { autoAlpha: 1, y: 0, duration: 0.55 }, 0)
        .to(titleLines, { autoAlpha: 1, y: 0, duration: 0.85, stagger: 0.09 }, 0.08)
        .to(stageInner, { autoAlpha: 1, y: 0, scale: 1, duration: 1.1 }, 0.16)
        .to(body, { autoAlpha: 1, y: 0, duration: 0.7 }, 0.38)
        .to(actions, { autoAlpha: 1, y: 0, duration: 0.6, stagger: 0.07 }, 0.54)
        .to(chips, { autoAlpha: 1, y: 0, duration: 0.5, stagger: 0.05 }, 0.72)
        .to(foot, { autoAlpha: 1, y: 0, duration: 0.55 }, 0.84);
    };

    const introActive = Boolean(
      document.querySelector('#root > [role="presentation"]'),
    );

    if (introActive) {
      window.addEventListener(SITE_REVEAL_EVENT, start, { once: true });
      fallbackTimer = window.setTimeout(start, 4600);
    } else {
      start();
    }

    return () => {
      window.clearTimeout(fallbackTimer);
      window.removeEventListener(SITE_REVEAL_EVENT, start);
      timeline?.kill();
      scrollTimeline?.scrollTrigger?.kill();
      scrollTimeline?.kill();
      // Always ensure content is visible on unmount
      gsap.set(hiddenTargets, {
        clearProps: "opacity,visibility,transform",
        autoAlpha: 1,
      });
    };
  }, []);

  return (
    <section
      ref={sectionRef}
      className="hero relative overflow-hidden"
      aria-labelledby="hero-title"
    >
      <div className="hero__grid" aria-hidden="true" />

      <Container className="hero__inner relative z-10">
        <div className="hero__layout">
          <div className="hero__copy" data-hero-copy>
            <div className="hero__lead">
              <p className="hero__eyebrow" data-hero-eyebrow>
                <span className="hero__eyebrow-dot" aria-hidden="true" />
                <span className="orbitron-text font-orbitron text-maze-lime uppercase tracking-wider">
                  {hero.eyebrow}
                </span>
              </p>

              <h1 id="hero-title" className="hero__title" data-hero-title>
                {hero.titleLines.map((line) => (
                  <span key={line} className="hero__title-line orbitron-text">
                    {line}
                  </span>
                ))}
                <span className="hero__subtitle">
                  <span className="hero__subtitle-line" aria-hidden="true" />
                  {hero.subtitle}
                </span>
              </h1>
            </div>

            <div className="hero__rest">
              <p className="hero__body" data-hero-body>
                <span className="poppins-text">{hero.body}</span>
              </p>

              <div className="hero__actions" data-hero-actions>
                <Button
                  href={hero.primaryCta.href}
                  variant="primary"
                  size="lg"
                  className="hero__cta hero__cta--primary"
                  onClick={(event) => {
                    if (
                      event.metaKey ||
                      event.ctrlKey ||
                      event.shiftKey ||
                      event.altKey
                    ) {
                      return;
                    }
                    event.preventDefault();
                    navigate(hero.primaryCta.href);
                  }}
                >
                  <span className="hero__play" aria-hidden="true">
                    <ArrowRight className="hero__play-icon" />
                  </span>
                  {hero.primaryCta.label}
                </Button>
                <Button
                  href={hero.secondaryCta.href}
                  variant="outline"
                  size="lg"
                  className="hero__cta hero__cta--secondary"
                >
                  {hero.secondaryCta.label}
                  <ArrowUpRight className="hero__cta-arrow" />
                </Button>
              </div>

              <ul className="hero__chips">
                {hero.chips.map((chip) => (
                  <li key={chip} className="hero__chip" data-hero-chip>
                    <span className="hero__chip-dot" aria-hidden="true" />
                    {chip}
                  </li>
                ))}
              </ul>
            </div>
          </div>

          <div className="hero__stage" data-hero-stage>
            <div className="hero__stage-inner" data-hero-stage-inner>
              <TuxStage />
            </div>
          </div>
        </div>

        <div className="hero__footer" data-hero-foot>
          <a href="#about" className="hero__scroll group">
            <span className="hero__scroll-line" aria-hidden="true">
              <span />
            </span>
            <span>Scroll to explore</span>
            <ArrowDown className="hero__scroll-arrow" />
          </a>
          <p className="hero__meta">Edwins Lab / Open source practice</p>
        </div>
      </Container>
    </section>
  );
}
