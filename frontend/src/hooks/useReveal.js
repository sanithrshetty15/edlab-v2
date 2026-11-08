import { useEffect, useRef } from "react";
import { gsap } from "../utils/animations";
import { prefersReducedMotion } from "../utils/reducedMotion";

export function useReveal({
  selector = "[data-reveal]",
  y = 36,
  duration = 0.9,
  stagger = 0.08,
  start = "top 82%",
  once = true,
} = {}) {
  const ref = useRef(null);

  useEffect(() => {
    const scope = ref.current;
    if (!scope || prefersReducedMotion()) return undefined;

    const targets = scope.querySelectorAll(selector);
    if (!targets.length) return undefined;

    const ctx = gsap.context(() => {
      gsap.from(targets, {
        autoAlpha: 0,
        y,
        duration,
        stagger,
        ease: "power3.out",
        clearProps: "transform,opacity,visibility",
        scrollTrigger: {
          trigger: scope,
          start,
          once,
        },
      });
    }, scope);

    return () => ctx.revert();
  }, [duration, once, selector, stagger, start, y]);

  return ref;
}
