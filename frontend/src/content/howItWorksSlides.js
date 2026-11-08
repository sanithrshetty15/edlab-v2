/*
 * Photographs for the automatic slideshow in "How the lab works".
 *
 * Replace or add entries here; the component derives everything from this
 * array. `fit` is optional and defaults to "cover". `surface` sets the layer
 * colour behind the photograph (used by the light wordmark frame).
 */

export const SLIDE_HOLD_MS = 3500;
export const SLIDE_TRANSITION_S = 1.15;

export const howItWorksSlides = [
  {
    src: "/images/lab/pic1.jpg",
    alt: "The Edwin's Lab wordmark beside a portrait, set on a light background.",
    objectPosition: "50% 50%",
    fit: "contain",
    surface: "light",
  },
  {
    src: "/images/lab/pic2.jpg",
    alt: "Students gathered around a laptop in the lab, pointing at the screen and comparing notes.",
    objectPosition: "50% 45%",
  },
  {
    src: "/images/lab/pic3.jpg",
    alt: "A student presenting at a projector screen while classmates listen from their desks.",
    objectPosition: "32% 50%",
  },
  {
    src: "/images/lab/pic4.jpg",
    alt: "A group of Edwins Lab students standing together for a portrait.",
    objectPosition: "50% 38%",
  },
  {
    src: "/images/lab/pic5.jpg",
    alt: "Students working together on laptops in front of a projector displaying the words Hammer Mindset.",
    objectPosition: "50% 50%",
  },
];
