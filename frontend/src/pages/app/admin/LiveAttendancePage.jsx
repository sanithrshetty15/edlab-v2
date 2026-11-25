export default function LiveAttendancePage() {
  const { pathname } = window.location;
  if (pathname.includes("live-attendance") || pathname.includes("attendance-terminals")) {
    window.location.href = "/app/attendance";
  }
  return null;
}