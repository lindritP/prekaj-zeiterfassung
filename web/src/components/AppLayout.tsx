import { Outlet } from "react-router-dom";
import { LogOut } from "lucide-react";
import { useAuth } from "@/auth/AuthContext";
import { Button } from "@/components/ui/button";

/** Eingeloggte Shell: Topbar (App-Name + Benutzer + Logout) + Inhalt. */
export function AppLayout() {
  const { arbeiter, logout } = useAuth();
  return (
    <div className="min-h-screen bg-background">
      <header className="border-b">
        <div className="mx-auto flex h-14 max-w-6xl items-center justify-between px-4">
          <span className="font-semibold">Prekaj-Zeiterfassung · Admin</span>
          <div className="flex items-center gap-3 text-sm">
            <span className="text-muted-foreground">{arbeiter?.name}</span>
            <Button variant="outline" size="sm" onClick={() => void logout()}>
              <LogOut className="mr-1 size-4" /> Abmelden
            </Button>
          </div>
        </div>
      </header>
      <main className="mx-auto max-w-6xl p-4">
        <Outlet />
      </main>
    </div>
  );
}
