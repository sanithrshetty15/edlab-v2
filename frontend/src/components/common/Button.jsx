const base =
  "group inline-flex shrink-0 items-center justify-center gap-2.5 rounded-full font-poppins font-semibold leading-none transition-[background-color,color,border-color,transform,box-shadow] duration-300 ease-out active:translate-y-px disabled:pointer-events-none disabled:opacity-50";

const variants = {
  primary:
    "bg-maze-lime text-edwins-dark shadow-[0_10px_30px_rgba(197,255,65,0.24)] hover:bg-[#b7f22f] hover:shadow-[0_14px_36px_rgba(197,255,65,0.38)]",
  dark:
    "bg-edwins-dark text-white hover:bg-black hover:shadow-[0_14px_34px_rgba(28,28,28,0.18)]",
  outline:
    "border border-edwins-dark/20 bg-transparent text-edwins-dark hover:border-edwins-dark hover:bg-edwins-dark hover:text-white",
  light:
    "border border-edwins-dark/10 bg-white text-edwins-dark hover:border-edwins-dark/20 hover:bg-edwins-neutral",
  ghost:
    "bg-transparent text-edwins-dark hover:bg-edwins-dark/[0.06]",
};

const sizes = {
  sm: "px-4 py-2.5 text-[0.8125rem]",
  md: "px-5 py-3.5 text-sm",
  lg: "px-6 py-4 text-[0.9375rem]",
};

export function Button({
  href,
  as,
  variant = "primary",
  size = "md",
  className = "",
  children,
  type = "button",
  ...props
}) {
  const Component = href ? "a" : as || "button";
  const componentProps =
    Component === "button" ? { type, ...props } : { href, ...props };

  return (
    <Component
      className={`${base} ${variants[variant]} ${sizes[size]} ${className}`}
      {...componentProps}
    >
      {children}
    </Component>
  );
}
