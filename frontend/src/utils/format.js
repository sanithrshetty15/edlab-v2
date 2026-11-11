function toDate(value) {
  if (!value) return null;
  if (value instanceof Date) return Number.isNaN(value.getTime()) ? null : value;
  if (typeof value !== "string") return null;

  const dateOnly = /^\d{4}-\d{2}-\d{2}$/.test(value);
  const date = new Date(dateOnly ? `${value}T00:00:00` : value);
  return Number.isNaN(date.getTime()) ? null : date;
}

export function formatDate(value, options = {}) {
  const date = toDate(value);
  if (!date) return "—";
  return new Intl.DateTimeFormat("en-IN", {
    day: "numeric",
    month: "short",
    year: "numeric",
    ...options,
  }).format(date);
}

export function formatDateTime(value) {
  const date = toDate(value);
  if (!date) return "—";
  return new Intl.DateTimeFormat("en-IN", {
    day: "numeric",
    month: "short",
    year: "numeric",
    hour: "numeric",
    minute: "2-digit",
  }).format(date);
}

export function formatTime(value) {
  const date = toDate(value);
  if (!date) return "—";
  return new Intl.DateTimeFormat("en-IN", {
    hour: "numeric",
    minute: "2-digit",
  }).format(date);
}

export function formatClockTime(value) {
  if (!value || typeof value !== "string") return "—";
  const match = value.match(/^(\d{1,2}):(\d{2})/);
  if (!match) return formatTime(value);
  const date = new Date();
  date.setHours(Number(match[1]), Number(match[2]), 0, 0);
  return formatTime(date);
}

export function formatMonthDay(value) {
  const date = toDate(value);
  if (!date) return { month: "—", day: "—", weekday: "—" };
  return {
    month: new Intl.DateTimeFormat("en-IN", { month: "short" }).format(date),
    day: new Intl.DateTimeFormat("en-IN", { day: "2-digit" }).format(date),
    weekday: new Intl.DateTimeFormat("en-IN", { weekday: "short" }).format(date),
  };
}

export function formatRelative(value) {
  const date = toDate(value);
  if (!date) return "—";
  const seconds = Math.round((date.getTime() - Date.now()) / 1000);
  const formatter = new Intl.RelativeTimeFormat("en", { numeric: "auto" });
  const ranges = [
    ["year", 31536000],
    ["month", 2592000],
    ["week", 604800],
    ["day", 86400],
    ["hour", 3600],
    ["minute", 60],
  ];

  for (const [unit, amount] of ranges) {
    if (Math.abs(seconds) >= amount) {
      return formatter.format(Math.round(seconds / amount), unit);
    }
  }
  return formatter.format(seconds, "second");
}

export function formatNumber(value) {
  const number = Number(value);
  if (!Number.isFinite(number)) return "—";
  return new Intl.NumberFormat("en-IN").format(number);
}

export function toIsoDate(date) {
  const value = date instanceof Date ? date : toDate(date);
  if (!value) return "";
  const year = value.getFullYear();
  const month = String(value.getMonth() + 1).padStart(2, "0");
  const day = String(value.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}

export function daysAgoIso(days) {
  const date = new Date();
  date.setDate(date.getDate() - days);
  return toIsoDate(date);
}

export function formatDateRange(from, to) {
  if (!from && !to) return "—";
  return `${formatDate(from)} – ${formatDate(to)}`;
}

export function initials(name) {
  if (!name || typeof name !== "string") return "EL";
  const parts = name.trim().split(/\s+/).filter(Boolean);
  if (!parts.length) return "EL";
  return parts
    .slice(0, 2)
    .map((part) => part.charAt(0).toUpperCase())
    .join("");
}

export function titleCase(value, fallback = "—") {
  if (!value) return fallback;
  return String(value)
    .replace(/[_-]+/g, " ")
    .replace(/\b\w/g, (character) => character.toUpperCase());
}

export function excerpt(value, limit = 160) {
  if (!value) return "";
  const text = String(value).replace(/\s+/g, " ").trim();
  if (text.length <= limit) return text;
  return `${text.slice(0, limit - 1)}…`;
}

export function getGreeting() {
  const hour = new Date().getHours();
  if (hour < 12) return "Good morning";
  if (hour < 17) return "Good afternoon";
  return "Good evening";
}

export function pluralize(count, singular, plural = `${singular}s`) {
  return Number(count) === 1 ? singular : plural;
}

export function safeHttpUrl(value) {
  if (!value || typeof value !== "string") return null;
  try {
    const url = new URL(value, window.location.origin);
    if (url.protocol !== "http:" && url.protocol !== "https:") return null;
    return url.toString();
  } catch {
    return null;
  }
}

export function sortByDateDesc(items, getValue = (item) => item.created_at) {
  return [...items].sort((a, b) => {
    const first = toDate(getValue(a))?.getTime() ?? 0;
    const second = toDate(getValue(b))?.getTime() ?? 0;
    return second - first;
  });
}

export function toDateTimeLocalValue(value) {
  const date = toDate(value);
  if (!date) return "";
  const offset = date.getTimezoneOffset() * 60000;
  return new Date(date.getTime() - offset).toISOString().slice(0, 16);
}

export function isSameDay(first, second) {
  const a = toDate(first);
  const b = toDate(second);
  if (!a || !b) return false;
  return a.toDateString() === b.toDateString();
}
