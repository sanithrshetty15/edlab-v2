import { useAuth } from "../../auth/context";
import MemberDashboard from "./MemberDashboard";
import AdminDashboard from "./AdminDashboard";

export default function DashboardPage() {
  const auth = useAuth();
  return auth.isAdmin ? <AdminDashboard /> : <MemberDashboard />;
}
