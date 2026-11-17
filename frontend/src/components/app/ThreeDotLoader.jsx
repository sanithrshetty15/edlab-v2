/**
 * ThreeDotLoader – a reusable three-dot bouncing animation.
 *
 * Visual: three white dots that bounce sequentially with smooth easing.
 * Uses pure CSS keyframes — no animation library needed.
 *
 * Props:
 *   size   – "sm" | "md" | "lg"  (default "md")
 *   className – extra classes to apply to the wrapper
 *   label  – accessible label for screen readers (default "Loading")
 */
export function ThreeDotLoader({ size = "md", tone = "white", className = "", label = "Loading" }) {
  return (
    <span
      className={`lab-dot-loader lab-dot-loader--${size} ${tone === "dark" ? "lab-dot-loader--dark" : ""} ${className}`}
      role="status"
      aria-label={label}
    >
      <span className="lab-dot-loader__dot" aria-hidden="true" />
      <span className="lab-dot-loader__dot" aria-hidden="true" />
      <span className="lab-dot-loader__dot" aria-hidden="true" />
    </span>
  );
}

export default ThreeDotLoader;
