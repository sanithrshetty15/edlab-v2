import { useLayoutEffect, useRef } from "react";
import { gsap } from "../../utils/animations";
import { prefersReducedMotion } from "../../utils/reducedMotion";
import LabImage from "./LabImage";

export default function RevealFrame({
  item,
  index = 0,
  className = "",
  ratio = "4 / 5",
  priority = false,
  overlay = null,
}) {
  const frameRef = useRef(null);

  useLayoutEffect(() => {
    const root = frameRef.current;
    if (!root || prefersReducedMotion()) return undefined;

    const frame = root.querySelector("[data-reveal-media]");
    const image = root.querySelector(".lab-image__img");

    if (!frame) return undefined;

    const ctx = gsap.context(() => {
      gsap.fromTo(
        frame,
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
            trigger: root,
            start: "top 86%",
            once: true,
          },
        },
      );

      if (image) {
        gsap.fromTo(
          image,
          { scale: 1.14, yPercent: -3.5 },
          {
            scale: 1.04,
            yPercent: 3.5,
            ease: "none",
            scrollTrigger: {
              trigger: root,
              start: "top bottom",
              end: "bottom top",
              scrub: 1,
            },
          },
        );
      }
    }, root);

    return () => ctx.revert();
  }, []);

  return (
    <figure
      ref={frameRef}
      className={`reveal-frame ${className}`}
      style={{ aspectRatio: ratio }}
    >
      <div className="reveal-frame__media" data-reveal-media>
        <LabImage
          src={item?.src}
          alt={item?.alt ?? ""}
          index={index}
          slotLabel={item?.filename}
          priority={priority}
          fit={item?.fit}
          position={item?.position}
          surface={item?.surface}
        />
        {overlay}
      </div>
    </figure>
  );
}
