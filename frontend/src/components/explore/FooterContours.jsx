import { useLayoutEffect, useRef } from "react";
import { useReducedMotion } from "../../hooks/useMediaQuery";
import { gsap } from "../../utils/animations";

const VIEW_WIDTH = 1440;
const VIEW_HEIGHT = 900;
const LINE_COUNT = 62;
const LINE_SPACING = 34;
const BASE_START = -100;
const ANGLE = (36 * Math.PI) / 180;
const TAN = Math.tan(ANGLE);
const AMPLITUDE = 24;
const WAVELENGTH = 1050;
const PERIOD = 14;
const PHASE_STEP = 0.35;
const X_START = -270;
const X_END = 1710;
const X_STEP = 90;
const TAU = Math.PI * 2;

function pathData(index, phase) {
  const baseY = BASE_START + index * LINE_SPACING;
  const linePhase = phase + index * PHASE_STEP;
  const bow = 14 * Math.sin(phase * 0.5 + 0.4);
  let d = "";

  for (let x = X_START; x <= X_END; x += X_STEP) {
    const wave = AMPLITUDE * Math.sin((x / WAVELENGTH) * TAU + linePhase);
    const arc = bow * Math.sin(Math.min(1, Math.max(0, (x + 200) / VIEW_WIDTH)) * Math.PI);
    const y = baseY - x * TAN + wave + arc;
    d += `${d ? "L" : "M"}${x.toFixed(1)} ${y.toFixed(1)}`;
  }

  return d;
}

const STATIC_PATHS = Array.from({ length: LINE_COUNT }, (_, index) =>
  pathData(index, 0),
);

export default function FooterContours() {
  const layerRef = useRef(null);
  const pathsRef = useRef([]);
  const reducedMotion = useReducedMotion();

  useLayoutEffect(() => {
    if (reducedMotion) return undefined;

    const layer = layerRef.current;
    if (!layer) return undefined;

    const paths = pathsRef.current;
    let visible = false;
    let clock = 0;
    let lastTick = null;
    let lastFrame = -1;

    const render = (time) => {
      if (lastTick === null) lastTick = time;
      const delta = Math.min(0.12, Math.max(0, time - lastTick));
      lastTick = time;

      if (!visible || document.hidden) return;

      clock += delta;
      const frame = Math.floor(time * 30);
      if (frame === lastFrame) return;
      lastFrame = frame;

      const phase = (clock / PERIOD) * TAU;
      for (let index = 0; index < paths.length; index += 1) {
        if (paths[index]) {
          paths[index].setAttribute("d", pathData(index, phase));
        }
      }
    };

    const observer = new IntersectionObserver((entries) => {
      visible = entries.some((entry) => entry.isIntersecting);
      if (!visible) lastTick = null;
    });
    observer.observe(layer);

    const onVisibilityChange = () => {
      lastTick = null;
    };

    gsap.ticker.add(render);
    document.addEventListener("visibilitychange", onVisibilityChange);

    return () => {
      gsap.ticker.remove(render);
      observer.disconnect();
      document.removeEventListener("visibilitychange", onVisibilityChange);
    };
  }, [reducedMotion]);

  return (
    <div
      ref={layerRef}
      className="explore-footer__contours"
      data-footer-contours
      aria-hidden="true"
    >
      <svg
        viewBox={`0 0 ${VIEW_WIDTH} ${VIEW_HEIGHT}`}
        preserveAspectRatio="xMidYMid slice"
        focusable="false"
      >
        <g className="explore-footer__contour-lines">
          {STATIC_PATHS.map((d, index) => (
            <path
              key={index}
              ref={(element) => {
                pathsRef.current[index] = element;
              }}
              d={d}
              vectorEffect="non-scaling-stroke"
            />
          ))}
        </g>
      </svg>
    </div>
  );
}
