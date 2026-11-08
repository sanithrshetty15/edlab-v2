import { gsap } from "gsap";
import { ScrollTrigger } from "gsap/ScrollTrigger";

gsap.registerPlugin(ScrollTrigger);

export { gsap, ScrollTrigger };

export const SITE_REVEAL_EVENT = "edwins:site-reveal";

export const EASES = {
  out: "power3.out",
  inOut: "power2.inOut",
  expo: "expo.out",
};

export function refreshScrollTriggers() {
  if (typeof window === "undefined") return;

  const refresh = () => ScrollTrigger.refresh();

  if (document.fonts?.ready) {
    document.fonts.ready.then(refresh);
  }

  window.addEventListener("load", refresh, { once: true });

  return () => window.removeEventListener("load", refresh);
}
