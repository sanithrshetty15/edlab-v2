import { useEffect, useLayoutEffect, useRef, useState } from "react";
import { exploreLab } from "../../content/exploreContent";
import {
  SLIDE_HOLD_MS,
  SLIDE_TRANSITION_S,
  howItWorksSlides,
} from "../../content/howItWorksSlides";
import { useReducedMotion } from "../../hooks/useMediaQuery";
import { gsap } from "../../utils/animations";

const label = (index) => String(index + 1).padStart(2, "0");

const surfaceColor = (slide) =>
  slide?.surface === "light" ? "#f2f0e9" : "#0c0e10";

function applySlide(layer, slide) {
  if (!layer || !slide) return;
  const image = layer.querySelector("img");
  if (image) {
    image.src = slide.src;
    image.style.objectFit = slide.fit || "cover";
    image.style.objectPosition = slide.objectPosition || "center";
  }
  layer.style.backgroundColor = surfaceColor(slide);
}

export default function HowItWorksSlideshow() {
  const frameRef = useRef(null);
  const layerARef = useRef(null);
  const layerBRef = useRef(null);
  const [activeIndex, setActiveIndex] = useState(0);
  const reducedMotion = useReducedMotion();
  const firstSlide = howItWorksSlides[0];

  /* Entrance reveal, matching the old frame's clip-in. */
  useLayoutEffect(() => {
    const frame = frameRef.current;
    if (!frame || reducedMotion) return undefined;

    const media = frame.querySelector("[data-slideshow-media]");
    if (!media) return undefined;

    const ctx = gsap.context(() => {
      gsap.fromTo(
        media,
        {
          clipPath: "inset(16% 10% 16% 10% round 26px)",
          autoAlpha: 0.35,
        },
        {
          clipPath: "inset(0% 0% 0% 0% round 26px)",
          autoAlpha: 1,
          duration: 1.15,
          ease: "power3.out",
          scrollTrigger: {
            trigger: frame,
            start: "top 86%",
            once: true,
          },
        },
      );
    }, frame);

    return () => ctx.revert();
  }, [reducedMotion]);

  /* Slideshow engine: preload, crossfade, pause/resume, full cleanup. */
  useEffect(() => {
    if (reducedMotion || howItWorksSlides.length < 2) return undefined;

    const frame = frameRef.current;
    const layerA = layerARef.current;
    const layerB = layerBRef.current;
    if (!frame || !layerA || !layerB) return undefined;

    const layers = [layerA, layerB];
    const count = howItWorksSlides.length;
    let front = 0;
    let index = 0;
    let timer = 0;
    let timeline = null;
    let paused = false;
    let visible = true;
    let disposed = false;
    let preloadToken = 0;

    const preload = (slide) =>
      new Promise((resolve) => {
        const image = new Image();
        const done = () => resolve(true);
        image.onload = () => {
          if (typeof image.decode === "function") {
            image.decode().then(done, done);
          } else {
            done();
          }
        };
        image.onerror = () => resolve(false);
        image.src = slide.src;
      });

    const schedule = () => {
      window.clearTimeout(timer);
      timer = 0;
      if (paused || disposed) return;
      timer = window.setTimeout(() => {
        timer = 0;
        advance((index + 1) % count);
      }, SLIDE_HOLD_MS);
    };

    const startTransition = (next) => {
      if (disposed || timeline) return;

      const slide = howItWorksSlides[next];
      const incoming = layers[1 - front];
      const outgoing = layers[front];
      const incomingImage = incoming.querySelector("img");
      const outgoingImage = outgoing.querySelector("img");

      applySlide(incoming, slide);
      if (incomingImage) incomingImage.alt = slide.alt;
      if (outgoingImage) outgoingImage.alt = "";

      gsap.set(incoming, {
        visibility: "visible",
        opacity: 0.4,
        zIndex: 2,
        clipPath: "inset(0% 100% 0% 0%)",
        scale: 1.05,
      });
      gsap.set(outgoing, { zIndex: 1 });

      timeline = gsap.timeline({
        onComplete: () => {
          gsap.set(outgoing, { autoAlpha: 0 });
          gsap.set(incoming, {
            clipPath: "inset(0% 0% 0% 0%)",
            opacity: 1,
            scale: 1,
          });
          front = 1 - front;
          index = next;
          timeline = null;
          setActiveIndex(next);
          schedule();
        },
      });

      timeline
        .to(
          incoming,
          {
            clipPath: "inset(0% 0% 0% 0%)",
            opacity: 1,
            scale: 1,
            duration: SLIDE_TRANSITION_S,
            ease: "power3.inOut",
          },
          0,
        )
        .to(
          outgoing,
          {
            opacity: 0.55,
            scale: 1.03,
            duration: SLIDE_TRANSITION_S,
            ease: "power3.inOut",
          },
          0,
        );
    };

    const advance = (target, attempts = 0) => {
      if (disposed || paused || timeline) return;
      if (attempts >= count) {
        schedule();
        return;
      }

      const token = ++preloadToken;
      preload(howItWorksSlides[target]).then((loaded) => {
        if (disposed || token !== preloadToken) return;
        if (!loaded) {
          advance((target + 1) % count, attempts + 1);
          return;
        }
        if (paused) {
          schedule();
          return;
        }
        startTransition(target);
      });
    };

    const updatePaused = () => {
      const next = document.hidden || !visible;
      if (next === paused) return;
      paused = next;
      if (paused) {
        window.clearTimeout(timer);
        timer = 0;
      } else {
        schedule();
      }
    };

    applySlide(layerA, howItWorksSlides[0]);
    const firstImage = layerA.querySelector("img");
    if (firstImage) firstImage.alt = howItWorksSlides[0].alt;

    const observer = new IntersectionObserver(
      (entries) => {
        visible = entries.some((entry) => entry.isIntersecting);
        updatePaused();
      },
      { threshold: 0.15 },
    );
    observer.observe(frame);

    const onVisibilityChange = () => updatePaused();
    document.addEventListener("visibilitychange", onVisibilityChange);

    schedule();

    return () => {
      disposed = true;
      preloadToken += 1;
      window.clearTimeout(timer);
      if (timeline) {
        timeline.kill();
        timeline = null;
      }
      observer.disconnect();
      document.removeEventListener("visibilitychange", onVisibilityChange);
      gsap.killTweensOf(layers);
    };
  }, [reducedMotion]);

  const mediaStyle = { aspectRatio: "4 / 5" };

  return (
    <figure
      ref={frameRef}
      className="reveal-frame explore-story__media explore-slideshow"
      style={mediaStyle}
    >
      <div
        className="reveal-frame__media explore-slideshow__media"
        data-slideshow-media
      >
        {reducedMotion ? (
          <img
            className="explore-slideshow__img"
            src={firstSlide?.src}
            alt={firstSlide?.alt ?? ""}
            style={{
              objectFit: firstSlide?.fit || "cover",
              objectPosition: firstSlide?.objectPosition || "center",
            }}
            draggable="false"
          />
        ) : (
          <div className="explore-slideshow__stack">
            <div
              ref={layerARef}
              className="explore-slideshow__layer"
              style={{ backgroundColor: surfaceColor(firstSlide) }}
            >
              <img
                className="explore-slideshow__img"
                src={firstSlide?.src}
                alt={firstSlide?.alt ?? ""}
                style={{
                  objectFit: firstSlide?.fit || "cover",
                  objectPosition: firstSlide?.objectPosition || "center",
                }}
                draggable="false"
                decoding="async"
              />
            </div>
            <div
              ref={layerBRef}
              className="explore-slideshow__layer"
              aria-hidden="true"
              style={{ visibility: "hidden" }}
            >
              <img
                className="explore-slideshow__img"
                alt=""
                draggable="false"
                decoding="async"
              />
            </div>
          </div>
        )}

        <figcaption className="explore-media-caption">
          <span>Lab frame / {label(activeIndex)}</span>
          <span>{exploreLab.imageCaption}</span>
        </figcaption>
      </div>
    </figure>
  );
}
