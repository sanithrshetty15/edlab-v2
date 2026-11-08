function Icon({ children, className = "h-5 w-5", ...props }) {
  return (
    <svg
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.8"
      strokeLinecap="round"
      strokeLinejoin="round"
      className={className}
      aria-hidden="true"
      focusable="false"
      {...props}
    >
      {children}
    </svg>
  );
}

export function ArrowUpRight(props) {
  return (
    <Icon {...props}>
      <path d="M7 17 17 7" />
      <path d="M8 7h9v9" />
    </Icon>
  );
}

export function ArrowLeft(props) {
  return (
    <Icon {...props}>
      <path d="M20 12H4" />
      <path d="m10 6-6 6 6 6" />
    </Icon>
  );
}

export function ArrowRight(props) {
  return (
    <Icon {...props}>
      <path d="M4 12h16" />
      <path d="m14 6 6 6-6 6" />
    </Icon>
  );
}

export function ArrowUp(props) {
  return (
    <Icon {...props}>
      <path d="M12 20V4" />
      <path d="m6 10 6-6 6 6" />
    </Icon>
  );
}

export function ArrowDown(props) {
  return (
    <Icon {...props}>
      <path d="M12 4v16" />
      <path d="m6 14 6 6 6-6" />
    </Icon>
  );
}

export function Play(props) {
  return (
    <Icon {...props}>
      <path d="M8 5.5v13l11-6.5-11-6.5Z" fill="currentColor" stroke="none" />
    </Icon>
  );
}

export function Menu(props) {
  return (
    <Icon {...props}>
      <path d="M4 8h16" />
      <path d="M4 16h11" />
    </Icon>
  );
}

export function Close(props) {
  return (
    <Icon {...props}>
      <path d="m6 6 12 12" />
      <path d="M18 6 6 18" />
    </Icon>
  );
}

export function Download(props) {
  return (
    <Icon {...props}>
      <path d="M12 3v12" />
      <path d="m7 10 5 5 5-5" />
      <path d="M5 21h14" />
    </Icon>
  );
}

export function Android(props) {
  return (
    <Icon {...props}>
      <path d="M7 9h10v9a1.5 1.5 0 0 1-1.5 1.5h-7A1.5 1.5 0 0 1 7 18V9Z" />
      <path d="M8.5 9 6.8 5.8" />
      <path d="m15.5 9 1.7-3.2" />
      <path d="M7 9a5 5 0 0 1 10 0" />
      <path d="M10 13h.01" />
      <path d="M14 13h.01" />
      <path d="M4.5 12v5" />
      <path d="M19.5 12v5" />
    </Icon>
  );
}

export function Globe(props) {
  return (
    <Icon {...props}>
      <circle cx="12" cy="12" r="9" />
      <path d="M3 12h18" />
      <path d="M12 3c2.4 2.4 3.6 5.4 3.6 9s-1.2 6.6-3.6 9c-2.4-2.4-3.6-5.4-3.6-9S9.6 5.4 12 3Z" />
    </Icon>
  );
}

export function Mail(props) {
  return (
    <Icon {...props}>
      <rect x="3" y="5" width="18" height="14" rx="2" />
      <path d="m3.5 7 8.5 6 8.5-6" />
    </Icon>
  );
}

export function Terminal(props) {
  return (
    <Icon {...props}>
      <rect x="3" y="4" width="18" height="16" rx="2" />
      <path d="m7 9 3 3-3 3" />
      <path d="M13 15h4" />
    </Icon>
  );
}

export function Cpu(props) {
  return (
    <Icon {...props}>
      <rect x="8" y="8" width="8" height="8" rx="1" />
      <path d="M12 3v3" />
      <path d="M12 18v3" />
      <path d="M3 12h3" />
      <path d="M18 12h3" />
      <path d="M6 6h.01" />
      <path d="M18 6h.01" />
      <path d="M6 18h.01" />
      <path d="M18 18h.01" />
      <rect x="5" y="5" width="14" height="14" rx="2" />
    </Icon>
  );
}

export function Users(props) {
  return (
    <Icon {...props}>
      <path d="M16 20v-1.5a4 4 0 0 0-4-4H7a4 4 0 0 0-4 4V20" />
      <circle cx="9.5" cy="7" r="3.5" />
      <path d="M17 4.5a3.5 3.5 0 0 1 0 6.7" />
      <path d="M21 20v-1.5a4 4 0 0 0-3-3.8" />
    </Icon>
  );
}

export function Sparkle(props) {
  return (
    <Icon {...props}>
      <path d="M12 3v4" />
      <path d="M12 17v4" />
      <path d="M3 12h4" />
      <path d="M17 12h4" />
      <path d="m5.6 5.6 2.8 2.8" />
      <path d="m15.6 15.6 2.8 2.8" />
      <path d="m18.4 5.6-2.8 2.8" />
      <path d="m8.4 15.6-2.8 2.8" />
    </Icon>
  );
}

export function CursorMove(props) {
  return (
    <Icon {...props}>
      <path d="M5 4l6 16 2.4-6.6L20 11 5 4Z" />
      <path d="M14 14l4 4" />
    </Icon>
  );
}

export function ChevronDown(props) {
  return (
    <Icon {...props}>
      <path d="m6 9 6 6 6-6" />
    </Icon>
  );
}

export function Check(props) {
  return (
    <Icon {...props}>
      <path d="m5 12 4.5 4.5L19 7" />
    </Icon>
  );
}
