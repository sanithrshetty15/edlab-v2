import { navigate } from "../../utils/router";

export function AppLink({ to, children, onClick, className = "", ...props }) {
  const isExternal = /^https?:\/\//i.test(to || "");

  const handleClick = (event) => {
    if (onClick) onClick(event);
    if (event.defaultPrevented) return;
    if (isExternal || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) return;
    if (event.button !== 0) return;
    event.preventDefault();
    navigate(to);
  };

  return (
    <a
      href={to}
      className={className}
      onClick={handleClick}
      {...(isExternal ? { target: "_blank", rel: "noreferrer noopener" } : {})}
      {...props}
    >
      {children}
    </a>
  );
}
