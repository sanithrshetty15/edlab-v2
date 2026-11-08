export function Container({ className = "", children, ...props }) {
  return (
    <div
      className={`mx-auto w-full max-w-[1280px] px-5 sm:px-7 lg:px-10 ${className}`}
      {...props}
    >
      {children}
    </div>
  );
}

export function SectionHeading({
  eyebrow,
  title,
  description,
  id,
  dark = false,
  align = "left",
  className = "",
}) {
  const alignment =
    align === "center" ? "items-center text-center" : "items-start text-left";

  return (
    <div className={`flex flex-col ${alignment} ${className}`}>
      {eyebrow && (
        <p
          className={`mb-5 inline-flex items-center gap-3 font-orbitron text-[0.6875rem] font-semibold uppercase tracking-[0.2em] ${
            dark ? "text-maze-lime" : "text-edwins-dark/65"
          }`}
        >
          <span
            className="inline-block h-1.5 w-1.5 rounded-full bg-maze-lime"
            aria-hidden="true"
          />
          {eyebrow}
        </p>
      )}
      {title && (
        <h2
          id={id}
          className={`max-w-[19ch] text-balance font-orbitron text-[clamp(2rem,4vw,3.75rem)] font-semibold leading-[1.06] tracking-[-0.035em] ${
            dark ? "text-white" : "text-edwins-dark"
          }`}
        >
          {title}
        </h2>
      )}
      {description && (
        <p
          className={`mt-6 max-w-[46ch] text-pretty text-[0.9375rem] leading-7 sm:text-base ${
            dark ? "text-white/62" : "text-edwins-dark/62"
          }`}
        >
          {description}
        </p>
      )}
    </div>
  );
}
