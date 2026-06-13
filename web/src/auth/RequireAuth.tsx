import { Navigate, Outlet, useLocation } from "react-router-dom";
import { useAuth } from "@/auth/AuthContext";

/** Schützt verschachtelte Routen: lädt -> Spinner, anonym -> /login, sonst Outlet. */
export function RequireAuth() {
  const { state } = useAuth();
  const location = useLocation();

  if (state === "loading") {
    return (
      <div className="flex min-h-screen items-center justify-center text-muted-foreground">Lädt…</div>
    );
  }
  if (state === "anonymous") {
    return <Navigate to="/login" state={{ from: location }} replace />;
  }
  return <Outlet />;
}
