import { useEffect, useRef } from "react";
import * as THREE from "three";
import { GLTFLoader } from "three/examples/jsm/loaders/GLTFLoader.js";
import { RoomEnvironment } from "three/examples/jsm/environments/RoomEnvironment.js";
import { ScrollTrigger } from "../../utils/animations";
import { prefersReducedMotion } from "../../utils/reducedMotion";

/**
 * TUX — custom Blender model
 * --------------------------
 * Source: "Linux mascot Tux.blend".
 * Inspected structure (Blender 5.2.3):
 *   - one mesh object, "Tux-printable" (feet at the origin, Z-up, front facing -Y)
 *   - subdivision surface modifier, 3 used material slots (black / white / orange)
 *   - no armature, no shape keys, no UV-dependent look, no animation clips
 *   - native bounds: 0.137 x 0.102 x 0.188 (metres)
 *
 * Exported to public/models/tux.glb with modifiers applied, Y-up conversion and
 * the three Principled BSDF materials preserved. The GLB stays untouched at
 * runtime; all fitting happens on the loaded scene graph.
 *
 * Because the asset is a single rigid mesh, per-part head tracking, eye tracking
 * and blinking are NOT supported by the model and are intentionally not faked.
 * Only whole-model motion (float, cursor lean, scroll parallax) is applied.
 *
 * glTF is Y-up with the face toward +Z — the same side as the camera — so no
 * corrective rotation is required. The model is uniformly scaled to
 * TARGET_HEIGHT so the motion budget below keeps its meaning in model units.
 */

const MODEL_URL = `${import.meta.env.BASE_URL}models/tux.glb`;

const MODEL = {
  targetHeight: 1, // normalized working height, in world units
  padding: 1.2, // camera framing padding around the animated bounds
};

const ANIM = {
  floatAmp: 0.018,
  floatSpeed: 0.95,
  yawMax: (14 * Math.PI) / 180,
  pitchMax: (7 * Math.PI) / 180,
  cursorPosAmp: 0.02,
  scrollRange: 0.12,
  idleRot: 0.02,
  lerpPointer: 0.06,
};

const clamp = (value, min, max) => Math.min(Math.max(value, min), max);

function disposeObject(object) {
  object.traverse((child) => {
    if (child.geometry) child.geometry.dispose();
    if (!child.material) return;
    const materials = Array.isArray(child.material) ? child.material : [child.material];
    materials.forEach((material) => {
      // textures are never shared with the rest of the app, so each one is ours
      Object.values(material).forEach((value) => {
        if (value && value.isTexture) value.dispose();
      });
      material.dispose();
    });
  });
}

function fitCameraToObject(camera, box, size, canvasW, canvasH, anim, padding) {
  // Expand the measured bounds by the animation budget so the model stays in
  // frame at the extremes of float / cursor offset / scroll travel.
  const expandedH =
    size.y + anim.floatAmp * 2 + anim.scrollRange + anim.cursorPosAmp * 2;
  const expandedW = size.x + anim.cursorPosAmp * 2 + 0.04;
  const expandedD = size.z + 0.06;

  const vFov = (camera.fov * Math.PI) / 180;
  const aspect = canvasW / Math.max(canvasH, 1);
  const hFov = 2 * Math.atan(Math.tan(vFov / 2) * aspect);

  const distForH = expandedH / 2 / Math.tan(vFov / 2);
  const distForW = expandedW / 2 / Math.tan(hFov / 2);
  const dist = Math.max(distForH, distForW) * padding + expandedD / 2;

  const center = box.getCenter(new THREE.Vector3());
  // Aim slightly above the geometric center to leave the feet breathing room.
  const targetY = center.y + size.y * 0.03;
  camera.position.set(0, targetY + size.y * 0.04, dist);
  camera.near = Math.max(0.1, dist - 5);
  camera.far = dist + 10;
  camera.lookAt(0, targetY, 0);
  camera.updateProjectionMatrix();
}

export default function Tux3D({ onReady, onFailure }) {
  const mountRef = useRef(null);

  useEffect(() => {
    const mount = mountRef.current;
    if (!mount) return undefined;

    const reducedMotion = prefersReducedMotion();
    let renderer;

    try {
      renderer = new THREE.WebGLRenderer({
        antialias: true,
        alpha: true,
        powerPreference: "high-performance",
      });
    } catch {
      onFailure?.();
      return undefined;
    }

    try {
      renderer.setPixelRatio(Math.min(window.devicePixelRatio || 1, 2));
      renderer.setClearColor(0x000000, 0);
      renderer.outputColorSpace = THREE.SRGBColorSpace;
      renderer.toneMapping = THREE.ACESFilmicToneMapping;
      renderer.toneMappingExposure = 1.0;
      renderer.shadowMap.enabled = true;
      renderer.shadowMap.type = THREE.PCFShadowMap;
      renderer.domElement.style.display = "block";
      renderer.domElement.style.width = "100%";
      renderer.domElement.style.height = "100%";
      renderer.domElement.setAttribute("aria-hidden", "true");
      mount.appendChild(renderer.domElement);
    } catch {
      onFailure?.();
      return undefined;
    }

    const scene = new THREE.Scene();
    // Modest FOV to avoid perspective distortion
    const camera = new THREE.PerspectiveCamera(32, 1, 0.1, 50);

    // Environment (subtle, for material response)
    const roomEnvironment = new RoomEnvironment();
    const pmremGenerator = new THREE.PMREMGenerator(renderer);
    const environmentMap = pmremGenerator.fromScene(roomEnvironment, 0.04).texture;
    roomEnvironment.dispose();
    pmremGenerator.dispose();
    scene.environment = environmentMap;
    scene.environmentIntensity = 0.32;

    // --- Lighting: soft, low-medium contrast so the black body keeps its form ---
    scene.add(new THREE.HemisphereLight(0xffffff, 0x6b6d66, 0.6));

    const keyLight = new THREE.DirectionalLight(0xfff6ea, 1.3);
    keyLight.position.set(2.5, 4.5, 3.5);
    keyLight.castShadow = true;
    keyLight.shadow.mapSize.set(1024, 1024);
    const shadowExtent = 1.2;
    keyLight.shadow.camera.left = -shadowExtent;
    keyLight.shadow.camera.right = shadowExtent;
    keyLight.shadow.camera.top = shadowExtent;
    keyLight.shadow.camera.bottom = -shadowExtent;
    keyLight.shadow.camera.near = 0.5;
    keyLight.shadow.camera.far = 12;
    keyLight.shadow.radius = 3;
    keyLight.shadow.bias = -0.0005;
    scene.add(keyLight);

    // Lime rim from behind to separate the dark body from the lime halo
    const rimLight = new THREE.DirectionalLight(0xc5ff41, 0.35);
    rimLight.position.set(-2, 2.5, -3);
    scene.add(rimLight);

    // Cool fill opposite the key
    const fillLight = new THREE.DirectionalLight(0xf0f4ff, 0.4);
    fillLight.position.set(-3, 2, 2.5);
    scene.add(fillLight);

    // --- Contact shadow (grounded, sits just under the feet) ---
    const ground = new THREE.Mesh(
      new THREE.CircleGeometry(0.55, 48),
      new THREE.ShadowMaterial({ color: 0x1c1c1c, opacity: 0.16 }),
    );
    ground.rotation.x = -Math.PI / 2;
    ground.position.y = 0.001;
    ground.receiveShadow = true;
    scene.add(ground);

    const pivot = new THREE.Group();
    pivot.name = "tuxPivot";
    scene.add(pivot);

    // --- Interaction state ---
    const pointer = { x: 0, y: 0 };
    const target = { x: 0, y: 0 };
    const scrollState = { progress: 0 };

    const onPointerMove = (event) => {
      if (event.pointerType === "touch") return;
      const rect = mount.getBoundingClientRect();
      if (!rect.width || !rect.height) return;
      target.x = clamp(((event.clientX - rect.left) / rect.width) * 2 - 1, -1, 1);
      target.y = clamp(((event.clientY - rect.top) / rect.height) * 2 - 1, -1, 1);
    };

    const onPointerLeave = () => {
      target.x = 0;
      target.y = 0;
    };

    let scrollTrigger;
    if (!reducedMotion) {
      window.addEventListener("pointermove", onPointerMove, { passive: true });
      document.documentElement.addEventListener("pointerleave", onPointerLeave);

      const section = mount.closest("section") ?? mount;
      scrollTrigger = ScrollTrigger.create({
        trigger: section,
        start: "top top",
        end: "bottom top",
        onUpdate: (self) => {
          scrollState.progress = self.progress;
        },
      });
    }

    // --- Load the authored model ---
    let disposed = false;
    let modelBounds = null;
    let frameId = 0;
    let visible = true;

    const fit = () => {
      if (!modelBounds) return;
      const width = mount.clientWidth;
      const height = mount.clientHeight;
      if (!width || !height) return;
      camera.aspect = width / height;
      fitCameraToObject(
        camera,
        modelBounds.box,
        modelBounds.size,
        width,
        height,
        ANIM,
        MODEL.padding,
      );
      renderer.setSize(width, height, false);
      if (reducedMotion) renderer.render(scene, camera);
    };

    const resizeObserver = new ResizeObserver(() => fit());
    resizeObserver.observe(mount);

    const timer = new THREE.Timer();
    const baseRootY = 0; // feet normalized to y = 0 when the model is placed

    const animate = (timestamp) => {
      frameId = requestAnimationFrame(animate);
      if (!visible || document.hidden) return;

      timer.update(timestamp);
      const time = timer.getElapsed();

      pointer.x += (target.x - pointer.x) * ANIM.lerpPointer;
      pointer.y += (target.y - pointer.y) * ANIM.lerpPointer;

      const float = Math.sin(time * ANIM.floatSpeed) * ANIM.floatAmp;
      const scrollOffset = -scrollState.progress * ANIM.scrollRange;

      // Whole-model motion only: the asset has no parts or rig to drive.
      pivot.position.x = clamp(pointer.x * ANIM.cursorPosAmp, -ANIM.cursorPosAmp, ANIM.cursorPosAmp);
      pivot.position.y = baseRootY + float + scrollOffset;
      pivot.rotation.y = clamp(
        pointer.x * ANIM.yawMax + Math.sin(time * 0.32) * ANIM.idleRot,
        -ANIM.yawMax,
        ANIM.yawMax,
      );
      pivot.rotation.x = clamp(-pointer.y * ANIM.pitchMax * 0.5, -ANIM.pitchMax, ANIM.pitchMax);
      pivot.rotation.z = clamp(pointer.x * 0.015, -0.025, 0.025);

      renderer.render(scene, camera);
    };

    const io = new IntersectionObserver(
      (entries) => {
        visible = entries[0]?.isIntersecting ?? false;
      },
      { rootMargin: "120px" },
    );
    io.observe(mount);

    const loader = new GLTFLoader();
    loader.load(
      MODEL_URL,
      (gltf) => {
        if (disposed) {
          // The effect was torn down while the request was in flight.
          disposeObject(gltf.scene);
          return;
        }

        const model = gltf.scene;
        model.name = "tuxModel";
        model.traverse((child) => {
          if (!child.isMesh) return;
          child.castShadow = true;
          child.receiveShadow = true;
          // Double-sided material + tight camera framing: never cull the body.
          child.frustumCulled = false;
        });

        // Normalize: uniform scale to TARGET_HEIGHT, then center on X/Z and
        // rest the lowest point (the feet) on y = 0.
        const rawBox = new THREE.Box3().setFromObject(model);
        const rawSize = rawBox.getSize(new THREE.Vector3());
        const rawCenter = rawBox.getCenter(new THREE.Vector3());
        const scale = MODEL.targetHeight / Math.max(rawSize.y, 0.0001);

        model.scale.setScalar(scale);
        model.position.set(
          -rawCenter.x * scale,
          -rawBox.min.y * scale,
          -rawCenter.z * scale,
        );
        pivot.add(model);

        const box = new THREE.Box3().setFromObject(pivot);
        modelBounds = { box, size: box.getSize(new THREE.Vector3()) };

        fit();
        onReady?.();

        if (!reducedMotion) animate();
      },
      undefined,
      (error) => {
        if (disposed) return;
        console.error(`[Tux3D] Failed to load ${MODEL_URL}`, error);
        // Let the stage swap in the illustrated fallback.
        onFailure?.();
      },
    );

    const onVisibility = () => {
      if (!document.hidden && !reducedMotion) timer.getDelta?.();
    };
    document.addEventListener("visibilitychange", onVisibility);

    const onContextLost = (event) => {
      event.preventDefault();
      onFailure?.();
    };
    renderer.domElement.addEventListener("webglcontextlost", onContextLost);

    return () => {
      disposed = true;
      cancelAnimationFrame(frameId);
      document.removeEventListener("visibilitychange", onVisibility);
      io.disconnect();
      resizeObserver.disconnect();
      window.removeEventListener("pointermove", onPointerMove);
      document.documentElement.removeEventListener("pointerleave", onPointerLeave);
      scrollTrigger?.kill();
      renderer.domElement.removeEventListener("webglcontextlost", onContextLost);

      // Dispose the loaded model first (geometry, materials, textures).
      pivot.traverse((object) => {
        if (object.isMesh) disposeObject(object);
      });
      scene.remove(pivot);
      ground.geometry.dispose();
      ground.material.dispose();

      scene.environment = null;
      environmentMap.dispose();
      renderer.dispose();
      renderer.forceContextLoss?.();
      if (renderer.domElement.parentNode === mount) {
        mount.removeChild(renderer.domElement);
      }
    };
  }, [onFailure, onReady]);

  return <div ref={mountRef} className="h-full w-full" />;
}
