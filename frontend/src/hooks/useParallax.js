import { useEffect, useRef } from "react";
import { gsap } from "../utils/animations";
import { prefersReducedMotion } from "../utils/reducedMotion";

export function useParallax({
  distance = 40,
  start = "top bottom",
  end = "bottom top",
  triggerRef,
} = {}) {
  const ref = useRef(null);
  const trigger = triggerRef ?? ref;

  useEffect(() => {
    const target = ref.current;
    const triggerElement = trigger.current;
    if (!target || !triggerElement || prefersReducedMotion()) return undefined;

    const ctx = gsap.context(() => {
      gsap.fromTo(
        target,
        { y: distance },
        {
          y: -distance,
          ease: "none",
          scrollTrigger: {
            trigger: triggerElement,
            start,
            end,
            scrub: 1,
          },
        },
      );
    }, target);

    return () => ctx.revert();
  }, [distance, end, start, trigger]);

  return ref;
}
