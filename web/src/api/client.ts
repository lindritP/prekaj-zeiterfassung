import type { TokenResponse } from "@/api/types";

const BASE = import.meta.env.VITE_API_BASE_URL ?? "/api/v1";

/** Strukturierter API-Fehler aus dem Backend-Format { error: { code, message } }. */
export class ApiError extends Error {
  constructor(
    public status: number,
    public code: string,
    message: string,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

// ── Auth-Brücke: von der AuthProvider gesetzt. Token nur im Speicher (kein localStorage). ──
let accessToken: string | null = null;
let onAuthLost: () => void = () => {};

export function setAccessToken(token: string | null): void {
  accessToken = token;
}
export function setOnAuthLost(fn: () => void): void {
  onAuthLost = fn;
}

// ── Single-flight Refresh: parallele 401 (und der StrictMode-Doppeleffekt) lösen nur EINEN
//    /auth/refresh-Request aus. Rotierende Refresh-Tokens dürfen nicht doppelt eingelöst werden. ──
let refreshPromise: Promise<TokenResponse | null> | null = null;

export function refreshSession(): Promise<TokenResponse | null> {
  refreshPromise ??= (async () => {
    const res = await fetch(`${BASE}/auth/refresh`, { method: "POST" });
    if (!res.ok) return null;
    const data = (await res.json()) as TokenResponse;
    accessToken = data.access_token;
    return data;
  })().finally(() => {
    refreshPromise = null;
  });
  return refreshPromise;
}

async function rawFetch(path: string, init: RequestInit, withAuth: boolean): Promise<Response> {
  const headers = new Headers(init.headers);
  if (withAuth && accessToken) headers.set("Authorization", `Bearer ${accessToken}`);
  return fetch(`${BASE}${path}`, { ...init, headers });
}

async function toError(res: Response): Promise<ApiError> {
  let code = "error";
  let message = res.statusText || "Fehler";
  try {
    const body = (await res.json()) as Partial<{ error: { code: string; message: string } }>;
    if (body.error) {
      code = body.error.code ?? code;
      message = body.error.message ?? message;
    }
  } catch {
    // kein JSON-Body
  }
  return new ApiError(res.status, code, message);
}

interface FetchOpts {
  auth?: boolean; // default true
}

/** Zentraler Fetch-Wrapper: Bearer-Token + 401→Refresh→Retry (einmal). */
export async function apiFetch<T>(path: string, init: RequestInit = {}, opts: FetchOpts = {}): Promise<T> {
  const auth = opts.auth ?? true;
  let res = await rawFetch(path, init, auth);

  if (res.status === 401 && auth) {
    const session = await refreshSession();
    if (session) {
      res = await rawFetch(path, init, true);
    } else {
      onAuthLost();
      throw await toError(res);
    }
  }

  if (!res.ok) throw await toError(res);
  if (res.status === 204) return undefined as T;
  const ct = res.headers.get("content-type") ?? "";
  return (ct.includes("application/json") ? await res.json() : await res.text()) as T;
}

const jsonInit = (method: string, body?: unknown): RequestInit => ({
  method,
  headers: body === undefined ? undefined : { "Content-Type": "application/json" },
  body: body === undefined ? undefined : JSON.stringify(body),
});

/** Typisierte Helfer. */
export const api = {
  get: <T>(path: string, opts?: FetchOpts) => apiFetch<T>(path, { method: "GET" }, opts),
  post: <T>(path: string, body?: unknown, opts?: FetchOpts) => apiFetch<T>(path, jsonInit("POST", body), opts),
  patch: <T>(path: string, body?: unknown, opts?: FetchOpts) => apiFetch<T>(path, jsonInit("PATCH", body), opts),
  del: <T>(path: string, opts?: FetchOpts) => apiFetch<T>(path, { method: "DELETE" }, opts),
};
