import { useLayoutEffect, useRef, useState } from "react";
import { Container } from "../common";
import { ArrowRight } from "../common/Icons";
import { labGallery } from "../../content/exploreGallery";
import { exploreGallery as copy } from "../../content/exploreContent";
import { gsap } from "../../utils/animations";
import { useMediaQuery, useReducedMotion } from "../../hooks/useMediaQuery";
import { useReveal } from "../../hooks/useReveal";
import LabImage from "./LabImage";

const label = (index) => String(index + 1).padStart(2, "0");

/* Landscape cards: the ring radius is derived from the rendered card width. */
const CARD_ASPECT = 16 / 9;
const CARD_GAP = 1.12;

function GalleryHeading() {
  return (
    <div className="explore-gallery__header" data-gallery-head>
      <div>
        <p className="explore-eyebrow" data-reveal>
          {copy.eyebrow}
        </p>
        <h2 id="gallery-title" className="explore-display mt-6" data-reveal>
          {copy.title}
        </h2>
      </div>
      <p className="explore-gallery__note" data-reveal>
        {copy.note}
      </p>
    </div>
  );
}

function RingCard({ item, index, decorative = false, priority = false }) {
  return (
    <figure
      className="explore-ring-card"
      data-ring-card
      aria-hidden={decorative ? "true" : undefined}
    >
      <LabImage
        src={item.src}
        alt={item.alt}
        index={index}
        slotLabel={item.filename}
        priority={priority}
        fit="cover"
        position={item.position}
        surface={item.surface}
      />
      <figcaption className="explore-ring-card__label">
        <span>{label(index)}</span>
        <span>{item.title}</span>
      </figcaption>
    </figure>
  );
}

function DesktopGallery() {
  const sectionRef = useRef(null);
  const sceneRef = useRef(null);
  const pivotRef = useRef(null);
  const ringRef = useRef(null);
  const progressRef = useRef(null);
  const [activeIndex, setActiveIndex] = useState(0);
  const count = labGallery.length;
  const looping = count > 2;
  const ringItems = looping
    ? [labGallery[count - 1], ...labGallery, labGallery[0]]
    : labGallery;
  const ringOffset = looping ? 1 : 0;

  useLayoutEffect(() => {
    const scene = sceneRef.current;
    const pivot = pivotRef.current;
    const ring = ringRef.current;
    if (!scene || !pivot || !ring || count < 1) return undefined;

    const layout = () => {
      const rect = scene.getBoundingClientRect();
      const stepDegrees = Math.min(30, 360 / count);
      const stepRadians = (stepDegrees * Math.PI) / 180;
      const heightBudget = Math.max(120, rect.height - 24);

      let cardWidth = Math.min(
        rect.width * 0.345,
        heightBudget * CARD_ASPECT,
        440,
      );
      cardWidth = Math.max(168, cardWidth);

      /*
       * Adjacent cards are stepRadians apart on the ring. For their centres to
       * be at least CARD_GAP * cardWidth apart in 3D, the chord of that angle
       * must satisfy 2 * r * sin(step / 2) >= cardWidth * CARD_GAP.
       */
      let radius =
        count > 1
          ? (cardWidth * CARD_GAP) / (2 * Math.sin(stepRadians / 2))
          : 0;
      const maxRadius = Math.max(rect.width * 0.86, 1);

      if (count > 1 && radius > maxRadius) {
        radius = maxRadius;
        cardWidth = (radius * 2 * Math.sin(stepRadians / 2)) / CARD_GAP;
      }

      const cardHeight = cardWidth / CARD_ASPECT;
      const chord = 2 * radius * Math.sin(stepRadians / 2);
      const perspective = Math.round(
        Math.min(2600, Math.max(1200, radius * 2.2)),
      );

      if (import.meta.env.DEV && count > 1 && chord + 0.5 < cardWidth * CARD_GAP) {
        console.warn(
          `[Gallery] Card centres are ${chord.toFixed(1)}px apart, expected at least ${(cardWidth * CARD_GAP).toFixed(1)}px.`,
        );
      }

      scene.style.perspective = `${perspective}px`;
      /* Centre the ring on the perspective plane so wide cards are not
         pushed toward the camera and clipped by the stage. */
      pivot.style.transform = `translateZ(${-radius.toFixed(2)}px)`;
      scene.dataset.cardWidth = cardWidth.toFixed(1);
      scene.dataset.cardHeight = cardHeight.toFixed(1);
      scene.dataset.ringRadius = radius.toFixed(1);
      scene.dataset.cardChord = chord.toFixed(1);
      ring.style.setProperty("--card-w", `${cardWidth.toFixed(2)}px`);
      ring.style.setProperty("--card-h", `${cardHeight.toFixed(2)}px`);

      ring.querySelectorAll("[data-ring-card]").forEach((card, index) => {
        const angle = (index - ringOffset) * stepDegrees;
        card.style.transform = `rotateY(${angle}deg) translateZ(${radius.toFixed(2)}px)`;
      });
    };

    let frame = 0;
    const scheduleLayout = () => {
      cancelAnimationFrame(frame);
      frame = requestAnimationFrame(layout);
    };

    layout();
    window.addEventListener("resize", scheduleLayout);
    const resizeObserver = new ResizeObserver(scheduleLayout);
    resizeObserver.observe(scene);

    return () => {
      cancelAnimationFrame(frame);
      window.removeEventListener("resize", scheduleLayout);
      resizeObserver.disconnect();
    };
  }, [count, ringOffset]);

  useLayoutEffect(() => {
    const section = sectionRef.current;
    const ring = ringRef.current;
    if (!section || !ring) return undefined;

    const stepDegrees = Math.min(30, 360 / count);
    const spin = -stepDegrees * (count - 1);

    const ctx = gsap.context(() => {
      gsap.from("[data-gallery-head] > *", {
        autoAlpha: 0,
        y: 22,
        duration: 0.8,
        stagger: 0.1,
        ease: "power3.out",
        scrollTrigger: {
          trigger: section,
          start: "top 82%",
          once: true,
        },
      });

      gsap.from("[data-ring-card]", {
        autoAlpha: 0,
        duration: 0.9,
        stagger: 0.05,
        ease: "power2.out",
        scrollTrigger: {
          trigger: section,
          start: "top 68%",
          once: true,
        },
      });

      gsap.fromTo(
        ring,
        { rotationY: 0 },
        {
          rotationY: spin,
          ease: "none",
          scrollTrigger: {
            trigger: section,
            start: "top top",
            end: () =>
              `+=${window.innerHeight * Math.max(2.4, count * 0.55)}`,
            pin: true,
            scrub: 0.65,
            anticipatePin: 1,
            invalidateOnRefresh: true,
            onUpdate: (self) => {
              const next = Math.round(self.progress * (count - 1));
              setActiveIndex((current) =>
                current === next ? current : next,
              );

              if (progressRef.current) {
                progressRef.current.style.transform = `scaleX(${self.progress})`;
              }
            },
          },
        },
      );
    }, section);

    return () => ctx.revert();
  }, [count]);

  const current = labGallery[activeIndex] ?? labGallery[0];

  return (
    <section
      ref={sectionRef}
      id="gallery"
      className="explore-gallery explore-gallery--pinned"
      aria-labelledby="gallery-title"
    >
      <div className="explore-gallery__ambient" aria-hidden="true" />

      <Container className="explore-gallery__body">
        <GalleryHeading />

        <div className="explore-gallery__stage">
          <div className="explore-gallery__stage-glow" aria-hidden="true" />
          <div className="explore-carousel" ref={sceneRef}>
            <div className="explore-carousel__pivot" ref={pivotRef}>
              <div className="explore-carousel__ring" ref={ringRef}>
                {ringItems.map((item, index) => {
                  const photoIndex = (index - ringOffset + count) % count;
                  return (
                    <RingCard
                      key={`${item.id}-${index}`}
                      item={item}
                      index={photoIndex}
                      decorative={
                        index < ringOffset || index >= ringOffset + count
                      }
                      priority={index === ringOffset}
                    />
                  );
                })}
              </div>
            </div>
          </div>
          <div className="explore-gallery__floor" aria-hidden="true" />
        </div>

        <div className="explore-gallery__caption-row">
          <div className="explore-gallery__caption">
            <div
              className="explore-gallery__caption-inner"
              key={current?.id}
            >
              <span className="explore-gallery__caption-index">
                Frame {label(activeIndex)} / {label(count - 1)}
              </span>
              <h3>{current?.title}</h3>
              <p>{current?.caption}</p>
            </div>
          </div>

          <div className="explore-gallery__aside">
            <span className="explore-gallery__hint">
              {copy.scrollHint}
              <ArrowRight className="h-3.5 w-3.5" />
            </span>
            <span className="explore-gallery__counter">
              <strong>{label(activeIndex)}</strong>
              <span aria-hidden="true">/</span>
              {label(count - 1)}
            </span>
          </div>
        </div>

        <div className="explore-gallery__progress" aria-hidden="true">
          <span ref={progressRef} />
        </div>
      </Container>
    </section>
  );
}

function MobileGallery() {
  const revealRef = useReveal({ start: "top 86%", y: 26 });

  return (
    <section
      ref={revealRef}
      id="gallery"
      className="explore-gallery explore-gallery--flow"
      aria-labelledby="gallery-title"
    >
      <div className="explore-gallery__ambient" aria-hidden="true" />
      <Container>
        <GalleryHeading />
        <p className="explore-gallery__mobile-hint" data-reveal>
          <span aria-hidden="true">/</span>
          {copy.mobileHint}
        </p>
      </Container>

      <div
        className="explore-mobile-track"
        role="list"
        aria-label="Edwins Lab gallery"
      >
        {labGallery.map((item, index) => (
          <article
            key={item.id}
            role="listitem"
            className="explore-mobile-card"
            data-reveal
          >
            <div className="explore-mobile-card__media">
              <LabImage
                src={item.src}
                alt={item.alt}
                index={index}
                slotLabel={item.filename}
                priority={index === 0}
                fit="cover"
                position={item.position}
                surface={item.surface}
              />
              <span className="explore-mobile-card__number">
                {label(index)}
              </span>
            </div>
            <div className="explore-mobile-card__copy">
              <h3>{item.title}</h3>
              <p>{item.caption}</p>
            </div>
          </article>
        ))}
      </div>

      <Container>
        <p className="explore-gallery__image-note">{copy.imageNote}</p>
      </Container>
    </section>
  );
}

function StaticGallery() {
  return (
    <section
      id="gallery"
      className="explore-gallery explore-gallery--flow"
      aria-labelledby="gallery-title"
    >
      <div className="explore-gallery__ambient" aria-hidden="true" />
      <Container>
        <GalleryHeading />
        <div className="explore-static-grid">
          {labGallery.map((item, index) => (
            <figure key={item.id} className="explore-static-card">
              <div className="explore-static-card__media">
                <LabImage
                  src={item.src}
                  alt={item.alt}
                  index={index}
                  slotLabel={item.filename}
                  priority={index < 2}
                  fit="cover"
                  position={item.position}
                  surface={item.surface}
                />
                <span className="explore-mobile-card__number">
                  {label(index)}
                </span>
              </div>
              <figcaption>
                <h3>{item.title}</h3>
                <p>{item.caption}</p>
              </figcaption>
            </figure>
          ))}
        </div>
        <p className="explore-gallery__image-note">{copy.imageNote}</p>
      </Container>
    </section>
  );
}

export default function GallerySection() {
  const isDesktop = useMediaQuery("(min-width: 768px)");
  const reducedMotion = useReducedMotion();

  if (labGallery.length < 2 || reducedMotion) {
    return <StaticGallery />;
  }

  if (!isDesktop) {
    return <MobileGallery />;
  }

  return <DesktopGallery />;
}
