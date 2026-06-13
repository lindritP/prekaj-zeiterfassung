import { createContext, useCallback, useContext, useEffect, useState, type ReactNode } from "react";
import type { Arbeiter, TokenResponse } from "@/api/types";
import { api, refreshSession, setAccessToken, setOnAuthLost } from "@/api/client";

type AuthState = "loading" | "authenticated" | "anonymous";

interface AuthContextValue {
  state: AuthState;
  arbeiter: Arbeiter | null;
  login: (email: string, passwort: string) => Promise<void>;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<AuthState>("loading");
  const [arbeiter, setArbeiter] = useState<Arbeiter | null>(null);

  const applySession = useCallback((res: TokenResponse) => {
    setAccessToken(res.access_token);
    setArbeiter(res.arbeiter);
    setState("authenticated");
  }, []);

  const clearSession = useCallback(() => {
    setAccessToken(null);
    setArbeiter(null);
    setState("anonymous");
  }, []);

  // Client meldet "Session verloren" (Refresh fehlgeschlagen) -> ausloggen.
  useEffect(() => {
    setOnAuthLost(clearSession);
  }, [clearSession]);

  // Session beim Laden wiederherstellen (Refresh via httpOnly-Cookie). Single-flight,
  // daher StrictMode-sicher.
  useEffect(() => {
    let cancelled = false;
    refreshSession()
      .then((res) => {
        if (cancelled) return;
        if (res) applySession(res);
        else clearSession();
      })
      .catch(() => {
        if (!cancelled) clearSession();
      });
    return () => {
      cancelled = true;
    };
  }, [applySession, clearSession]);

  const login = useCallback(
    async (email: string, passwort: string) => {
      const res = await api.post<TokenResponse>("/auth/login", { email, passwort, client: "web" }, { auth: false });
      applySession(res);
    },
    [applySession],
  );

  const logout = useCallback(async () => {
    try {
      await api.post("/auth/logout", undefined, { auth: false });
    } catch {
      // Cookie wird serverseitig invalidiert; lokaler State wird ohnehin geleert.
    }
    clearSession();
  }, [clearSession]);

  return <AuthContext value={{ state, arbeiter, login, logout }}>{children}</AuthContext>;
}

// eslint-disable-next-line react-refresh/only-export-components
export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within AuthProvider");
  return ctx;
}
