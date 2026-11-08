import { useLayoutEffect, useRef } from "react";
import { Container, Button } from "../common";
import { ArrowDown } from "../common/Icons";
import { labGallery } from "../../content/exploreGallery";
import { exploreHero } from "../../content/exploreContent";
import { gsap } from "../../utils/animations";
import { prefersReducedMotion } from "../../utils/reducedMotion";
import LabImage from "./LabImage";

export default function ExploreHero() {
  const sectionRef = useRef(null);

  useLayoutEffect(() => {
    const section = sectionRef.current;
    if (!section || prefersReducedMotion()) return undefined;

    const ctx = gsap.context(() => {
      const intro = gsap.timeline({
        defaults: { ease: "power3.out" },
        delay: 0.12,
      });

      intro
        .fromTo(
          "[data-hero-topline]",
          { autoAlpha: 0, y: 14 },
          { autoAlpha: 1, y: 0, duration: 0.7 },
        )
        .fromTo(
          "[data-hero-line]",
          { yPercent: 115 },
          { yPercent: 0, duration: 1.05, stagger: 0.1 },
          0.12,
        )
        .fromTo(
          "[data-hero-intro]",
          { autoAlpha: 0, y: 20 },
          { autoAlpha: 1, y: 0, duration: 0.8 },
          0.52,
        )
        .fromTo(
          "[data-hero-support]",
          { autoAlpha: 0, y: 16 },
          { autoAlpha: 1, y: 0, duration: 0.7 },
          0.68,
        )
        .fromTo(
          "[data-hero-actions] > *",
          { autoAlpha: 0, y: 14 },
          { autoAlpha: 1, y: 0, duration: 0.6, stagger: 0.08 },
          0.82,
        )
        .fromTo(
          "[data-hero-chip]",
          { autoAlpha: 0, x: -10 },
          { autoAlpha: 1, x: 0, duration: 0.5, stagger: 0.05 },
          0.98,
        )
        .fromTo(
          "[data-hero-frame]",
          {
            clipPath: "inset(100% 0% 0% 0% round 28px)",
            autoAlpha: 0.5,
          },
          {
            clipPath: "inset(0% 0% 0% 0% round 28px)",
            autoAlpha: 1,
            duration: 1.2,
            stagger: 0.14,
          },
          0.38,
        )
        .fromTo(
          "[data-hero-note]",
          { autoAlpha: 0, y: 18 },
          { autoAlpha: 1, y: 0, duration: 0.7 },
          1.05,
        )
        .fromTo(
          "[data-hero-foot]",
          { autoAlpha: 0 },
          { autoAlpha: 1, duration: 0.8 },
          1.1,
        );

      const parallax = gsap.timeline({
        defaults: { ease: "none" },
        scrollTrigger: {
          trigger: section,
          start: "top top",
          end: "bottom top",
          scrub: 0.8,
        },
      });

      parallax
        .to("[data-hero-copy]", { yPercent: -5, autoAlpha: 0.55 }, 0)
        .to("[data-hero-visual]", { yPercent: -7 }, 0)
        .to("[data-hero-frame--secondary]", { yPercent: 8 }, 0)
        .to("[data-hero-glow]", { yPercent: 14, scale: 1.06 }, 0);
    }, section);

    return () => ctx.revert();
  }, []);

  const photographicFrames = labGallery.filter((item) => item.fit !== "contain");
  const mainImage = photographicFrames[0] ?? labGallery[0];
  const secondaryImage = photographicFrames[1] ?? mainImage;

  return (
    <section
      ref={sectionRef}
      className="explore-hero"
      aria-labelledby="explore-title"
    >
      <div className="explore-hero__bg" aria-hidden="true">
        <div className="explore-hero__grid" />
        <div className="explore-hero__glow" data-hero-glow />
        <div className="explore-hero__glow explore-hero__glow--lime" />
        <div className="explore-hero__orbit" />
      </div>

      <Container className="explore-hero__inner">
        <p className="explore-hero__topline" data-hero-topline>
          {exploreHero.eyebrow}
        </p>

        <div className="explore-hero__layout">
          <div className="explore-hero__copy" data-hero-copy>
            <h1 id="explore-title" className="explore-hero__title">
              {exploreHero.titleLines.map((line) => (
                <span key={line} className="explore-hero__line">
                  <span className="explore-hero__line-inner" data-hero-line>
                    {line}
                  </span>
                </span>
              ))}
            </h1>

            <p className="explore-hero__intro" data-hero-intro>
              {exploreHero.intro}
            </p>

            <p className="explore-hero__support" data-hero-support>
              <span aria-hidden="true" />
              {exploreHero.support}
            </p>

            <div className="explore-hero__actions" data-hero-actions>
              <Button href="#lab" variant="primary" size="lg">
                Enter the lab
                <ArrowDown className="h-4 w-4" />
              </Button>
              <a href="#gallery" className="explore-hero__text-link">
                Or start with the gallery
              </a>
            </div>

            <ul className="explore-hero__chips">
              {exploreHero.chips.map((chip) => (
                <li key={chip} className="explore-hero__chip" data-hero-chip>
                  <span aria-hidden="true" />
                  {chip}
                </li>
              ))}
            </ul>
          </div>

          <div className="explore-hero__visual" data-hero-visual>
            <figure
              className="explore-hero__frame explore-hero__frame--main"
              data-hero-frame
            >
              <LabImage
                src={mainImage?.src}
                alt={mainImage?.alt ?? ""}
                index={0}
                slotLabel={mainImage?.filename}
                priority
                fit={mainImage?.fit}
                position={mainImage?.position}
                surface={mainImage?.surface}
              />
            </figure>

            <figure
              className="explore-hero__frame explore-hero__frame--secondary"
              data-hero-frame
              data-hero-frame--secondary
            >
              <LabImage
                src={secondaryImage?.src}
                alt={secondaryImage?.alt ?? ""}
                index={1}
                slotLabel={secondaryImage?.filename}
                fit={secondaryImage?.fit}
                position={secondaryImage?.position}
                surface={secondaryImage?.surface}
              />
            </figure>

            <div className="explore-hero__note" data-hero-note>
              <span className="explore-hero__note-mark" aria-hidden="true">
                /
              </span>
              <p>{exploreHero.visualCaption}</p>
            </div>

            <span className="explore-hero__vertical" aria-hidden="true">
              Edwin&apos;s Lab / Open source
            </span>
          </div>
        </div>

        <div className="explore-hero__footer" data-hero-foot>
          <a href="#lab" className="explore-scroll-cue">
            <span className="explore-scroll-cue__line" aria-hidden="true" />
            <span>{exploreHero.scrollLabel}</span>
            <ArrowDown className="h-3.5 w-3.5" />
          </a>
          <p>{exploreHero.meta}</p>
        </div>
      </Container>
    </section>
  );
}
