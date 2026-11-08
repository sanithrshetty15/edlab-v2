import { Component, Suspense, lazy, useCallback, useState } from "react";
import TuxArtwork from "./TuxArtwork";

const Tux3D = lazy(() => import("./Tux3D"));

function supportsWebGL() {
  if (typeof window === "undefined") return false;

  try {
    const canvas = document.createElement("canvas");
    return Boolean(
      window.WebGLRenderingContext &&
        (canvas.getContext("webgl2") || canvas.getContext("webgl")),
    );
  } catch {
    return false;
  }
}

class TuxErrorBoundary extends Component {
  constructor(props) {
    super(props);
    this.state = { failed: false };
  }

  static getDerivedStateFromError() {
    return { failed: true };
  }

  componentDidCatch(error) {
    console.error("[TuxStage] 3D render failed", error);
  }

  render() {
    if (this.state.failed) return this.props.fallback;
    return this.props.children;
  }
}

export default function TuxStage({ className = "" }) {
  const [webglAvailable] = useState(supportsWebGL);
  const [renderFailed, setRenderFailed] = useState(false);
  const [modelReady, setModelReady] = useState(false);

  const handleReady = useCallback(() => setModelReady(true), []);
  const handleFailure = useCallback(() => {
    setModelReady(false);
    setRenderFailed(true);
  }, []);

  const show3D = webglAvailable && !renderFailed;
  // The illustrated Tux doubles as the loading state and the error fallback.
  const showArtwork = !show3D || !modelReady;

  return (
    <div className={`tux-stage ${className}`}>
      {/* Subtle lime halo and decorative rings - behind canvas (z-index 0) */}
      <div className="tux-stage__halo" aria-hidden="true" />
      <div
        className="tux-stage__ring tux-stage__ring--outer"
        aria-hidden="true"
      />
      <div
        className="tux-stage__ring tux-stage__ring--inner"
        aria-hidden="true"
      />

      <div className="tux-stage__canvas" data-tux-tracking>
        {show3D && (
          <Suspense fallback={null}>
            <TuxErrorBoundary fallback={null}>
              <Tux3D onReady={handleReady} onFailure={handleFailure} />
            </TuxErrorBoundary>
          </Suspense>
        )}

        {showArtwork && (
          <div
            className={`tux-stage__fallback${
              show3D ? " tux-stage__fallback--overlay" : ""
            }`}
          >
            <TuxArtwork
              title={
                show3D
                  ? "Classic Tux, loading the interactive 3D model"
                  : "Classic Tux illustration"
              }
              interactive
            />
          </div>
        )}
      </div>

      {!show3D && (
        <p className="sr-only">
          3D view is unavailable in this browser. Showing the illustrated Tux.
        </p>
      )}
    </div>
  );
}
