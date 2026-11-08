import { useEffect, useId, useRef, useState } from "react";
import { prefersReducedMotion } from "../../utils/reducedMotion";

const clamp = (value, min, max) => Math.min(Math.max(value, min), max);

export default function TuxArtwork({
  className = "",
  activePart = null,
  interactive = false,
  blink = true,
  title = "Classic Tux illustration",
}) {
  const rawId = useId().replace(/:/g, "");
  const svgRef = useRef(null);
  const leftEyeTrackRef = useRef(null);
  const rightEyeTrackRef = useRef(null);
  const [blinking, setBlinking] = useState(false);

  useEffect(() => {
    if (!blink || prefersReducedMotion()) return undefined;

    let blinkTimer;
    let resetTimer;

    const schedule = () => {
      blinkTimer = window.setTimeout(
        () => {
          setBlinking(true);
          resetTimer = window.setTimeout(() => {
            setBlinking(false);
            schedule();
          }, 150);
        },
        2800 + Math.random() * 2800,
      );
    };

    schedule();

    return () => {
      window.clearTimeout(blinkTimer);
      window.clearTimeout(resetTimer);
    };
  }, [blink]);

  useEffect(() => {
    if (
      !interactive ||
      prefersReducedMotion() ||
      !window.matchMedia("(pointer: fine)").matches
    ) {
      return undefined;
    }

    const svg = svgRef.current;
    if (!svg) return undefined;

    const host = svg.closest("[data-tux-tracking]") ?? svg;
    let frame = 0;

    const moveEyes = (x, y) => {
      const transform = `translate(${x.toFixed(2)}px, ${y.toFixed(2)}px)`;
      if (leftEyeTrackRef.current) {
        leftEyeTrackRef.current.style.transform = transform;
      }
      if (rightEyeTrackRef.current) {
        rightEyeTrackRef.current.style.transform = transform;
      }
    };

    const onMove = (event) => {
      const rect = host.getBoundingClientRect();
      if (!rect.width || !rect.height) return;

      const nx = clamp(((event.clientX - rect.left) / rect.width) * 2 - 1, -1, 1);
      const ny = clamp(((event.clientY - rect.top) / rect.height) * 2 - 1, -1, 1);

      cancelAnimationFrame(frame);
      frame = requestAnimationFrame(() => moveEyes(nx * 2.6, ny * 2));
    };

    const onLeave = () => moveEyes(0, 0);

    host.addEventListener("pointermove", onMove, { passive: true });
    host.addEventListener("pointerleave", onLeave);

    return () => {
      cancelAnimationFrame(frame);
      host.removeEventListener("pointermove", onMove);
      host.removeEventListener("pointerleave", onLeave);
    };
  }, [interactive]);

  return (
    <svg
      ref={svgRef}
      viewBox="0 0 420 480"
      className={`tux-artwork ${blinking ? "is-blinking" : ""} ${className}`}
      role={title ? "img" : undefined}
      aria-label={title || undefined}
      aria-hidden={title ? undefined : "true"}
      xmlns="http://www.w3.org/2000/svg"
    >
      <defs>
        <radialGradient id={`body-${rawId}`} cx="42%" cy="28%" r="78%">
          <stop offset="0%" stopColor="#363636" />
          <stop offset="46%" stopColor="#1d1d1d" />
          <stop offset="100%" stopColor="#090909" />
        </radialGradient>
        <radialGradient id={`head-${rawId}`} cx="38%" cy="22%" r="82%">
          <stop offset="0%" stopColor="#3a3a3a" />
          <stop offset="52%" stopColor="#1b1b1b" />
          <stop offset="100%" stopColor="#070707" />
        </radialGradient>
        <linearGradient id={`face-${rawId}`} x1="0" y1="0" x2="0.25" y2="1">
          <stop offset="0%" stopColor="#ffffff" />
          <stop offset="72%" stopColor="#f7f7f4" />
          <stop offset="100%" stopColor="#e9e9e4" />
        </linearGradient>
        <linearGradient id={`belly-${rawId}`} x1="0" y1="0" x2="0.2" y2="1">
          <stop offset="0%" stopColor="#ffffff" />
          <stop offset="76%" stopColor="#f5f5f1" />
          <stop offset="100%" stopColor="#e8e8e2" />
        </linearGradient>
        <linearGradient id={`beak-${rawId}`} x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stopColor="#ffc94a" />
          <stop offset="55%" stopColor="#f5ae25" />
          <stop offset="100%" stopColor="#dc9010" />
        </linearGradient>
        <linearGradient id={`feet-${rawId}`} x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stopColor="#ffc247" />
          <stop offset="100%" stopColor="#dd9211" />
        </linearGradient>
        <filter id={`shadow-${rawId}`} x="-40%" y="-40%" width="180%" height="180%">
          <feGaussianBlur stdDeviation="10" />
        </filter>
        <filter id={`glow-${rawId}`} x="-80%" y="-80%" width="260%" height="260%">
          <feGaussianBlur stdDeviation="22" />
        </filter>
      </defs>

      <ellipse
        cx="210"
        cy="455"
        rx="128"
        ry="16"
        fill="#1c1c1c"
        opacity="0.16"
        filter={`url(#shadow-${rawId})`}
      />

      <g
        className="tux-part"
        data-part="feet"
        data-active={activePart === "feet" ? "true" : "false"}
      >
        <ellipse
          className="tux-part-glow"
          cx="210"
          cy="446"
          rx="112"
          ry="22"
          fill="#C5FF41"
          filter={`url(#glow-${rawId})`}
        />
        <g>
          <path
            d="M158 429c-27 0-48 8-48 19 0 11 30 17 60 13 19-3 28-12 24-21-3-7-17-11-36-11Z"
            fill={`url(#feet-${rawId})`}
          />
          <path
            d="M135 438c-6 0-12 1-17 3"
            fill="none"
            stroke="#a96a06"
            strokeOpacity="0.34"
            strokeWidth="2.4"
            strokeLinecap="round"
          />
          <path
            d="M153 435c-5 0-10 1-14 2"
            fill="none"
            stroke="#a96a06"
            strokeOpacity="0.34"
            strokeWidth="2.4"
            strokeLinecap="round"
          />
        </g>
        <g transform="translate(420 0) scale(-1 1)">
          <path
            d="M158 429c-27 0-48 8-48 19 0 11 30 17 60 13 19-3 28-12 24-21-3-7-17-11-36-11Z"
            fill={`url(#feet-${rawId})`}
          />
          <path
            d="M135 438c-6 0-12 1-17 3"
            fill="none"
            stroke="#a96a06"
            strokeOpacity="0.34"
            strokeWidth="2.4"
            strokeLinecap="round"
          />
          <path
            d="M153 435c-5 0-10 1-14 2"
            fill="none"
            stroke="#a96a06"
            strokeOpacity="0.34"
            strokeWidth="2.4"
            strokeLinecap="round"
          />
        </g>
      </g>

      <g
        className="tux-part"
        data-part="belly"
        data-active={activePart === "belly" ? "true" : "false"}
      >
        <ellipse
          className="tux-part-glow"
          cx="210"
          cy="310"
          rx="100"
          ry="120"
          fill="#C5FF41"
          filter={`url(#glow-${rawId})`}
        />
        <path
          d="M108 244c-30 28-34 88-12 130 12-18 16-56 30-92Z"
          fill="#0b0b0b"
        />
        <path
          d="M312 244c30 28 34 88 12 130-12-18-16-56-30-92Z"
          fill="#0b0b0b"
        />
        <path
          d="M210 118c-76 0-120 80-120 182 0 92 52 134 120 134s120-42 120-134c0-102-44-182-120-182Z"
          fill={`url(#body-${rawId})`}
        />
        <path
          d="M210 206c-40 0-61 54-61 112 0 56 25 84 61 84s61-28 61-84c0-58-21-112-61-112Z"
          fill={`url(#belly-${rawId})`}
        />
        <path
          d="M177 232c-13 18-19 45-20 72"
          fill="none"
          stroke="#ffffff"
          strokeOpacity="0.5"
          strokeWidth="3"
          strokeLinecap="round"
        />
      </g>

      <g
        className="tux-part"
        data-part="head"
        data-active={activePart === "head" ? "true" : "false"}
      >
        <ellipse
          className="tux-part-glow"
          cx="210"
          cy="160"
          rx="116"
          ry="104"
          fill="#C5FF41"
          filter={`url(#glow-${rawId})`}
        />
        <circle cx="210" cy="158" r="88" fill={`url(#head-${rawId})`} />
        <ellipse
          cx="178"
          cy="93"
          rx="28"
          ry="12"
          fill="#ffffff"
          opacity="0.13"
          transform="rotate(-22 178 93)"
        />
        <path
          d="M210 108c-32-24-72-18-84 16-12 36 20 70 84 100 64-30 96-64 84-100-12-34-52-40-84-16Z"
          fill={`url(#face-${rawId})`}
        />

        <g className="tux-eye-wrap" ref={leftEyeTrackRef}>
          <g className="tux-eye">
            <ellipse cx="184" cy="148" rx="12" ry="16" fill="#0c0c0c" />
            <circle cx="180.5" cy="142.5" r="3.6" fill="#ffffff" opacity="0.92" />
          </g>
        </g>
        <g className="tux-eye-wrap" ref={rightEyeTrackRef}>
          <g className="tux-eye">
            <ellipse cx="236" cy="148" rx="12" ry="16" fill="#0c0c0c" />
            <circle cx="232.5" cy="142.5" r="3.6" fill="#ffffff" opacity="0.92" />
          </g>
        </g>

        <path
          d="M210 151c-9 0-17 3-26 8l26 27 26-27c-9-5-17-8-26-8Z"
          fill={`url(#beak-${rawId})`}
        />
        <path
          d="M194 161c5-2 10-3 16-3s11 1 16 3"
          fill="none"
          stroke="#a56608"
          strokeOpacity="0.34"
          strokeWidth="2.4"
          strokeLinecap="round"
        />
        <path
          d="M190 167c6 3 13 4 20 4"
          fill="none"
          stroke="#ffffff"
          strokeOpacity="0.34"
          strokeWidth="3"
          strokeLinecap="round"
        />
      </g>
    </svg>
  );
}
