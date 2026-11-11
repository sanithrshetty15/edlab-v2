import { useEffect, useRef, useState } from "react";
import * as THREE from "three";

/**
 * Three.js particle morph.
 *
 * ~3200 lime particles travel between three sampled targets:
 *   1. a globe shell
 *   2. a Tux penguin silhouette
 *   3. the "EDWINS LAB" wordmark
 *
 * Targets 2 and 3 are rasterised into an offscreen canvas, the opaque pixels
 * are sampled and normalised into world units, so the shapes are real point
 * clouds rather than images.
 */

const SHAPE_KEYS = ["globe", "tux", "text"];
const SHAPE_LABELS = {
  globe: "Globe",
  tux: "Tux",
  text: "Edwins Lab",
};

const HOLD_SECONDS = 2.2;
const MORPH_SECONDS = 1.8;
const STEP_SECONDS = HOLD_SECONDS + MORPH_SECONDS;
const CYCLE_SECONDS = STEP_SECONDS * SHAPE_KEYS.length;

// Fixed world height for the orthographic camera; width follows the aspect.
const DESIGN_HEIGHT = 9;

const LIME = "#c5ff41";
const LIME_HIGHLIGHT = "#eaffb0";

/* ------------------------------------------------------------------ */
/* Shape rasterisation                                                 */
/* ------------------------------------------------------------------ */

function makeCanvas(width, height) {
  const canvas = document.createElement("canvas");
  canvas.width = width;
  canvas.height = height;
  return canvas;
}

/** Draws a front-facing Tux silhouette with cut-out belly and face. */
function drawTux(ctx, width, height) {
  const scale = Math.min(width, height) / 100;

  ctx.save();
  ctx.translate(width / 2, height / 2);
  ctx.scale(scale, scale);
  ctx.translate(-50, -50);
  ctx.fillStyle = "#000";

  // Feet
  ctx.beginPath();
  ctx.ellipse(32, 92, 13, 6, 0, 0, Math.PI * 2);
  ctx.ellipse(68, 92, 13, 6, 0, 0, Math.PI * 2);
  ctx.fill();

  // Flippers
  ctx.beginPath();
  ctx.ellipse(22, 58, 8, 21, -0.2, 0, Math.PI * 2);
  ctx.fill();
  ctx.beginPath();
  ctx.ellipse(78, 58, 8, 21, 0.2, 0, Math.PI * 2);
  ctx.fill();

  // Body and head
  ctx.beginPath();
  ctx.ellipse(50, 60, 29, 34, 0, 0, Math.PI * 2);
  ctx.fill();
  ctx.beginPath();
  ctx.ellipse(50, 26, 16, 17, 0, 0, Math.PI * 2);
  ctx.fill();

  // Cut out the belly and face so the silhouette reads as Tux
  ctx.globalCompositeOperation = "destination-out";
  ctx.beginPath();
  ctx.ellipse(50, 64, 20, 25, 0, 0, Math.PI * 2);
  ctx.fill();
  ctx.beginPath();
  ctx.ellipse(50, 27, 12, 12, 0, 0, Math.PI * 2);
  ctx.fill();
  ctx.globalCompositeOperation = "source-over";

  // Eyes
  ctx.beginPath();
  ctx.arc(44, 24, 2.6, 0, Math.PI * 2);
  ctx.arc(56, 24, 2.6, 0, Math.PI * 2);
  ctx.fill();

  // Beak
  ctx.beginPath();
  ctx.moveTo(43, 32);
  ctx.lineTo(57, 32);
  ctx.lineTo(50, 42);
  ctx.closePath();
  ctx.fill();

  ctx.restore();
}

/** Draws the EDWINS LAB wordmark. */
function drawWordmark(ctx, width, height) {
  const text = "EDWINS LAB";
  const family = '"Orbitron", "Poppins", system-ui, sans-serif';

  ctx.textAlign = "center";
  ctx.textBaseline = "middle";
  ctx.fillStyle = "#000";

  let size = height * 0.86;
  ctx.font = `900 ${size}px ${family}`;

  const maxWidth = width * 0.94;
  const measured = ctx.measureText(text).width;
  if (measured > maxWidth) {
    size *= maxWidth / measured;
    ctx.font = `900 ${size}px ${family}`;
  }

  ctx.fillText(text, width / 2, height / 2);
}

/**
 * Samples the opaque pixels of a drawing and maps them into world units,
 * preserving the drawing's aspect ratio.
 */
function sampleShape(draw, options) {
  const {
    count,
    canvasWidth,
    canvasHeight,
    maxWorldWidth,
    maxWorldHeight,
    step = 2,
  } = options;

  const canvas = makeCanvas(canvasWidth, canvasHeight);
  const ctx = canvas.getContext("2d", { willReadFrequently: true });
  if (!ctx) return new Float32Array(count * 3);

  draw(ctx, canvasWidth, canvasHeight);

  const { data } = ctx.getImageData(0, 0, canvasWidth, canvasHeight);
  const points = [];

  for (let y = 0; y < canvasHeight; y += step) {
    for (let x = 0; x < canvasWidth; x += step) {
      if (data[(y * canvasWidth + x) * 4 + 3] > 128) {
        points.push([x, y]);
      }
    }
  }

  const positions = new Float32Array(count * 3);
  if (points.length === 0) return positions;

  // Shuffle so duplicates are spread across the shape.
  for (let i = points.length - 1; i > 0; i -= 1) {
    const j = Math.floor(Math.random() * (i + 1));
    [points[i], points[j]] = [points[j], points[i]];
  }

  let minX = Infinity;
  let minY = Infinity;
  let maxX = -Infinity;
  let maxY = -Infinity;
  for (const [x, y] of points) {
    if (x < minX) minX = x;
    if (x > maxX) maxX = x;
    if (y < minY) minY = y;
    if (y > maxY) maxY = y;
  }

  const boundsWidth = Math.max(maxX - minX, 1);
  const boundsHeight = Math.max(maxY - minY, 1);
  const scale = Math.min(
    maxWorldWidth / boundsWidth,
    maxWorldHeight / boundsHeight,
  );
  const centerX = (minX + maxX) / 2;
  const centerY = (minY + maxY) / 2;

  for (let i = 0; i < count; i += 1) {
    const [x, y] = points[i % points.length];
    positions[i * 3] = (x - centerX) * scale + (Math.random() - 0.5) * 1.4 * scale;
    positions[i * 3 + 1] =
      -(y - centerY) * scale + (Math.random() - 0.5) * 1.4 * scale;
    positions[i * 3 + 2] = (Math.random() - 0.5) * 0.6;
  }

  return positions;
}

/** Builds a wireframe globe of latitude rings and meridian arcs. */
function buildGlobe(count, radius) {
  const points = [];
  const latitudeRings = 9;
  const meridianArcs = 14;
  const ringSteps = 132;
  const arcSteps = 78;
  const depth = 0.62;

  for (let ringIndex = 1; ringIndex < latitudeRings; ringIndex += 1) {
    const phi = (ringIndex / latitudeRings) * Math.PI;
    const y = Math.cos(phi) * radius;
    const ring = Math.sin(phi) * radius;

    for (let step = 0; step < ringSteps; step += 1) {
      const theta = (step / ringSteps) * Math.PI * 2;
      points.push([
        Math.cos(theta) * ring,
        y,
        Math.sin(theta) * ring * depth,
      ]);
    }
  }

  for (let arcIndex = 0; arcIndex < meridianArcs; arcIndex += 1) {
    const theta = (arcIndex / meridianArcs) * Math.PI;

    for (let step = 0; step <= arcSteps; step += 1) {
      const phi = (step / arcSteps) * Math.PI;
      points.push([
        Math.cos(theta) * Math.sin(phi) * radius,
        Math.cos(phi) * radius,
        Math.sin(theta) * Math.sin(phi) * radius * depth,
      ]);
    }
  }

  const positions = new Float32Array(count * 3);

  for (let i = 0; i < count; i += 1) {
    const point = points[Math.floor(Math.random() * points.length)];
    positions[i * 3] = point[0] + (Math.random() - 0.5) * 0.05;
    positions[i * 3 + 1] = point[1] + (Math.random() - 0.5) * 0.05;
    positions[i * 3 + 2] = point[2];
  }

  return positions;
}

function buildTargets(count, aspect) {
  const worldWidth = DESIGN_HEIGHT * aspect;

  return {
    globe: buildGlobe(count, Math.min(worldWidth, DESIGN_HEIGHT) * 0.4),
    tux: sampleShape(drawTux, {
      count,
      canvasWidth: 440,
      canvasHeight: 440,
      maxWorldWidth: worldWidth * 0.58,
      maxWorldHeight: DESIGN_HEIGHT * 0.86,
      step: 2,
    }),
    text: sampleShape(drawWordmark, {
      count,
      canvasWidth: 760,
      canvasHeight: 220,
      maxWorldWidth: worldWidth * 0.82,
      maxWorldHeight: DESIGN_HEIGHT * 0.62,
      step: 2,
    }),
  };
}

/* ------------------------------------------------------------------ */
/* Sprite                                                              */
/* ------------------------------------------------------------------ */

function createSprite() {
  const size = 64;
  const canvas = makeCanvas(size, size);
  const ctx = canvas.getContext("2d");
  if (!ctx) return null;

  const gradient = ctx.createRadialGradient(
    size / 2,
    size / 2,
    0,
    size / 2,
    size / 2,
    size / 2,
  );
  gradient.addColorStop(0, "rgba(255,255,255,1)");
  gradient.addColorStop(0.35, "rgba(255,255,255,0.82)");
  gradient.addColorStop(1, "rgba(255,255,255,0)");

  ctx.fillStyle = gradient;
  ctx.fillRect(0, 0, size, size);

  const texture = new THREE.CanvasTexture(canvas);
  texture.colorSpace = THREE.SRGBColorSpace;
  return texture;
}

/* ------------------------------------------------------------------ */
/* Component                                                           */
/* ------------------------------------------------------------------ */

export default function ParticleMorph({ className = "" }) {
  const mountRef = useRef(null);
  const [shape, setShape] = useState("globe");
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    const mount = mountRef.current;
    if (!mount) return undefined;

    let disposed = false;
    let frameId = 0;
    let renderer = null;
    let geometry = null;
    let material = null;
    let sprite = null;
    let scene = null;
    let camera = null;
    let clock = null;
    let targets = null;
    let resizeObserver = null;
    let visibilityObserver = null;
    let visible = true;
    let reportedShape = "globe";
    let reportedPhase = "hold";

    const prefersReducedMotion = window.matchMedia(
      "(prefers-reduced-motion: reduce)",
    ).matches;

    const cleanup = () => {
      if (frameId) cancelAnimationFrame(frameId);
      if (resizeObserver) resizeObserver.disconnect();
      if (visibilityObserver) visibilityObserver.disconnect();
      if (geometry) geometry.dispose();
      if (material) material.dispose();
      if (sprite) sprite.dispose();
      if (renderer) {
        renderer.dispose();
        renderer.forceContextLoss?.();
        if (renderer.domElement.parentNode === mount) {
          mount.removeChild(renderer.domElement);
        }
      }
    };

    const boot = async () => {
      // Wait for Orbitron so the wordmark is sampled from the brand face.
      try {
        if (document.fonts?.ready) {
          await Promise.race([
            document.fonts.ready,
            new Promise((resolve) => setTimeout(resolve, 1500)),
          ]);
        }
      } catch {
        // Font loading unavailable: fall back to the next font in the stack.
      }
      if (disposed) return;

      const width = Math.max(mount.clientWidth, 1);
      const height = Math.max(mount.clientHeight, 1);
      const aspect = width / height;
      const worldWidth = DESIGN_HEIGHT * aspect;

      try {
        renderer = new THREE.WebGLRenderer({
          antialias: true,
          alpha: true,
          powerPreference: "high-performance",
        });
      } catch {
        setFailed(true);
        return;
      }

      renderer.setPixelRatio(Math.min(window.devicePixelRatio || 1, 2));
      renderer.setClearColor(0x000000, 0);
      renderer.outputColorSpace = THREE.SRGBColorSpace;
      renderer.domElement.style.display = "block";
      renderer.domElement.style.width = "100%";
      renderer.domElement.style.height = "100%";
      renderer.domElement.setAttribute("aria-hidden", "true");
      renderer.setSize(width, height, false);
      mount.appendChild(renderer.domElement);

      scene = new THREE.Scene();
      camera = new THREE.OrthographicCamera(
        -worldWidth / 2,
        worldWidth / 2,
        DESIGN_HEIGHT / 2,
        -DESIGN_HEIGHT / 2,
        0.1,
        100,
      );
      camera.position.z = 10;

      const compact = width < 640;
      const count = compact ? 1600 : 3200;
      targets = buildTargets(count, aspect);

      const positions = new Float32Array(targets.globe);
      const colors = new Float32Array(count * 3);
      const delays = new Float32Array(count);
      const seeds = new Float32Array(count);
      const driftX = new Float32Array(count);
      const driftY = new Float32Array(count);

      const lime = new THREE.Color(LIME);
      const highlight = new THREE.Color(LIME_HIGHLIGHT);

      for (let i = 0; i < count; i += 1) {
        const shade = 0.55 + Math.random() * 0.45;
        const tint = Math.random() > 0.82;
        const color = tint ? highlight : lime;
        colors[i * 3] = color.r * shade;
        colors[i * 3 + 1] = color.g * shade;
        colors[i * 3 + 2] = color.b * shade;

        delays[i] = Math.random();
        seeds[i] = Math.random() * Math.PI * 2;
        driftX[i] = Math.random() - 0.5;
        driftY[i] = Math.random() - 0.5;
      }

      geometry = new THREE.BufferGeometry();
      geometry.setAttribute("position", new THREE.BufferAttribute(positions, 3));
      geometry.setAttribute("color", new THREE.BufferAttribute(colors, 3));

      sprite = createSprite();
      material = new THREE.PointsMaterial({
        size: (compact ? 1.7 : 2.4) * Math.min(window.devicePixelRatio || 1, 2),
        map: sprite,
        transparent: true,
        opacity: 0.95,
        depthWrite: false,
        blending: THREE.AdditiveBlending,
        vertexColors: true,
        sizeAttenuation: false,
      });

      const points = new THREE.Points(geometry, material);
      scene.add(points);

      clock = new THREE.Clock();

      const applyLayout = () => {
        const nextWidth = Math.max(mount.clientWidth, 1);
        const nextHeight = Math.max(mount.clientHeight, 1);
        const nextAspect = nextWidth / nextHeight;
        const nextWorldWidth = DESIGN_HEIGHT * nextAspect;

        camera.left = -nextWorldWidth / 2;
        camera.right = nextWorldWidth / 2;
        camera.top = DESIGN_HEIGHT / 2;
        camera.bottom = -DESIGN_HEIGHT / 2;
        camera.updateProjectionMatrix();

        renderer.setSize(nextWidth, nextHeight, false);
        targets = buildTargets(count, nextAspect);
      };

      const setReportedShape = (key) => {
        if (reportedShape === key) return;
        reportedShape = key;
        mount.dataset.shape = key;
        setShape(key);
      };

      mount.dataset.shape = "globe";
      mount.dataset.phase = "hold";
      mount.dataset.ready = "true";

      const renderFrame = () => {
        const elapsed = clock.getElapsedTime();
        const positionsArray = geometry.attributes.position.array;

        if (!visible) {
          frameId = requestAnimationFrame(renderFrame);
          return;
        }

        if (prefersReducedMotion) {
          positionsArray.set(targets.globe);
          geometry.attributes.position.needsUpdate = true;
          renderer.render(scene, camera);
          return;
        }

        const cycleTime = elapsed % CYCLE_SECONDS;
        const stepIndex = Math.floor(cycleTime / STEP_SECONDS);
        const localTime = cycleTime - stepIndex * STEP_SECONDS;
        const morphing = localTime > HOLD_SECONDS;
        const progress = morphing
          ? (localTime - HOLD_SECONDS) / MORPH_SECONDS
          : 0;

        const fromKey = SHAPE_KEYS[stepIndex];
        const toKey = SHAPE_KEYS[(stepIndex + 1) % SHAPE_KEYS.length];
        const from = targets[fromKey];
        const to = targets[morphing ? toKey : fromKey];

        for (let i = 0; i < count; i += 1) {
          const i3 = i * 3;
          const delay = delays[i] * 0.45;
          const spread = 1 - delay;
          const p = progress <= 0
            ? 0
            : Math.min(1, Math.max(0, (progress - delay) / spread));
          const eased = p * p * (3 - 2 * p);

          const wobble = Math.sin(elapsed * 1.15 + seeds[i]) * 0.03;
          const wobbleY = Math.cos(elapsed * 0.95 + seeds[i]) * 0.03;

          let x = from[i3] + (to[i3] - from[i3]) * eased;
          let y = from[i3 + 1] + (to[i3 + 1] - from[i3 + 1]) * eased;
          let z = from[i3 + 2] + (to[i3 + 2] - from[i3 + 2]) * eased;

          if (eased > 0 && eased < 1) {
            const arc = Math.sin(eased * Math.PI) * 1.1;
            x += driftX[i] * arc;
            y += driftY[i] * arc;
          }

          positionsArray[i3] = x + wobble;
          positionsArray[i3 + 1] = y + wobbleY;
          positionsArray[i3 + 2] = z;
        }

        geometry.attributes.position.needsUpdate = true;
        renderer.render(scene, camera);

        setReportedShape(progress > 0.5 ? toKey : fromKey);

        const phase = progress > 0 ? "morph" : "hold";
        if (reportedPhase !== phase) {
          reportedPhase = phase;
          mount.dataset.phase = phase;
        }

        frameId = requestAnimationFrame(renderFrame);
      };

      renderFrame();

      if (!prefersReducedMotion) {
        resizeObserver = new ResizeObserver(applyLayout);
        resizeObserver.observe(mount);

        visibilityObserver = new IntersectionObserver(
          ([entry]) => {
            visible = entry.isIntersecting;
          },
          { threshold: 0 },
        );
        visibilityObserver.observe(mount);
      }
    };

    boot();

    return () => {
      disposed = true;
      cleanup();
    };
  }, []);

  return (
    <div className={`particle-stage ${className}`} data-failed={failed}>
      <div className="particle-stage__hud" aria-hidden="true">
        <span className="particle-stage__eyebrow">Particle study</span>
        <span className="particle-stage__shape">{SHAPE_LABELS[shape]}</span>
      </div>

      <div
        ref={mountRef}
        className="particle-stage__canvas"
        role="img"
        aria-label="Lime-green particles that morph between a globe, a Tux penguin silhouette and the Edwins Lab wordmark"
      />

      {failed && (
        <p className="particle-stage__fallback">
          Edwins Lab — particle study
        </p>
      )}

      <div className="particle-stage__legend" aria-hidden="true">
        <span>Globe</span>
        <span>Tux</span>
        <span>Edwins Lab</span>
      </div>
    </div>
  );
}
