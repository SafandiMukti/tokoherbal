// Panggilan dari SERVER Next.js ke auth-service. Browser tidak pernah memanggilnya langsung.
import type { TokenResponse } from "./session";

const AUTH_BASE = process.env.AUTH_API_URL ?? "http://localhost:8081";

export type AuthResult<T> =
  | { ok: true; data: T }
  | { ok: false; status: number; message: string }; // status 0 = auth-service tidak terjangkau

async function call<T>(path: string, body: unknown): Promise<AuthResult<T>> {
  let res: Response;
  try {
    res = await fetch(AUTH_BASE + path, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
      cache: "no-store",
      signal: AbortSignal.timeout(5000),
    });
  } catch {
    return { ok: false, status: 0, message: "Layanan login sedang tidak tersedia, coba lagi sebentar." };
  }
  if (res.status === 204) return { ok: true, data: undefined as T };
  let json: unknown = null;
  try {
    json = await res.json();
  } catch {
    /* bukan JSON */
  }
  if (!res.ok) {
    const msg = (json as { error?: string } | null)?.error ?? `Permintaan gagal (${res.status})`;
    return { ok: false, status: res.status, message: msg };
  }
  return { ok: true, data: json as T };
}

export const login = (email: string, password: string) => call<TokenResponse>("/login", { email, password });
export const register = (email: string, password: string) =>
  call<{ id: number; email: string; role: string }>("/register", { email, password });
export const refresh = (refreshToken: string) => call<TokenResponse>("/refresh", { refresh_token: refreshToken });
export const logout = (refreshToken: string) => call<void>("/logout", { refresh_token: refreshToken });
