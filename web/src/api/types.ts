// DTO-Typen passend zu den Backend-Responses (/api/v1).
// In Part 2 werden weitere Typen (Zeitbuchung, Urlaub, Dokument, …) ergänzt.

export type Rolle = "arbeiter" | "admin";

export interface Arbeiter {
  id: string;
  name: string;
  email: string;
  rolle: Rolle;
}

export interface TokenResponse {
  access_token: string;
  refresh_token?: string; // nur Mobile
  arbeiter: Arbeiter;
}

export interface ApiErrorBody {
  error: { code: string; message: string };
}
