import { useState } from "react";

export default function LabImage({
  src,
  alt,
  index = 0,
  slotLabel,
  priority = false,
  className = "",
  imgClassName = "",
  fit = "cover",
  position = "center",
  surface = "dark",
}) {
  const [status, setStatus] = useState(src ? "loading" : "error");
  const [currentSrc, setCurrentSrc] = useState(src);

  if (src !== currentSrc) {
    setCurrentSrc(src);
    setStatus(src ? "loading" : "error");
  }

  const number = String(index + 1).padStart(2, "0");

  return (
    <div
      className={`lab-image ${surface === "light" ? "lab-image--light" : ""} ${className}`}
    >
      {src && status !== "error" && (
        <img
          src={src}
          alt={alt}
          loading={priority ? "eager" : "lazy"}
          decoding="async"
          draggable="false"
          className={`lab-image__img ${status === "loaded" ? "is-loaded" : ""} ${imgClassName}`}
          style={{ objectFit: fit, objectPosition: position }}
          onLoad={() => setStatus("loaded")}
          onError={() => setStatus("error")}
        />
      )}

      {status === "loading" && (
        <span className="lab-image__skeleton" aria-hidden="true" />
      )}

      {status === "error" && (
        <div
          className="lab-image__placeholder"
          role="img"
          aria-label={`${alt} Photograph not added yet.`}
        >
          <span className="lab-image__index" aria-hidden="true">
            {number}
          </span>
          <span className="lab-image__slot">
            Photograph slot {number}
          </span>
          <span className="lab-image__file">
            {slotLabel || src || "Add a photograph"}
          </span>
          <span className="lab-image__prompt">
            Add the real lab photograph to public/images/lab
          </span>
        </div>
      )}
    </div>
  );
}
