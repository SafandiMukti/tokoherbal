// Panggilan ke product-service yang membawa access token pengguna (dari cookie).
import { cookies } from "next/headers";
import { internalHeaders } from "./api";
import { ACCESS_COOKIE, REFRESH_COOKIE, decodeToken, type Session } from "./session";

const SERVER_BASE = process.env.PRODUCT_API_URL ?? "http://localhost:8082";

// Sesi saat ini, atau null bila belum login / token sudah habis.
export async function getSession(): Promise<Session | null> {
  const store = await cookies();
  const s = decodeToken(store.get(ACCESS_COOKIE)?.value);
  if (!s || s.exp <= Math.floor(Date.now() / 1000)) return null;
  return s;
}

// Ada refresh token tetapi tidak ada sesi aktif: proxy gagal memperbaruinya karena
// auth-service tidak terjangkau (kalau token ditolak, proxy sudah menghapus cookie-nya).
export async function hasPendingRefresh(): Promise<boolean> {
  const store = await cookies();
  return store.has(REFRESH_COOKIE);
}

export type AuthedResult<T> =
  | { ok: true; data: T }
  | { ok: false; status: number; message: string }; // 401 = perlu login, 503 = layanan mati

export async function authedRequest<T>(method: string, path: string, body?: unknown): Promise<AuthedResult<T>> {
  const store = await cookies();
  const token = store.get(ACCESS_COOKIE)?.value;
  if (!token) return { ok: false, status: 401, message: "perlu login" };
  const isForm = body instanceof FormData;
  let res: Response;
  try {
    res = await fetch(SERVER_BASE + path, {
      method,
      headers: {
        Authorization: `Bearer ${token}`,
        ...internalHeaders(),
        // untuk FormData, Content-Type (beserta boundary) diisi otomatis oleh fetch
        ...(body !== undefined && !isForm ? { "Content-Type": "application/json" } : {}),
      },
      body: body === undefined ? undefined : isForm ? (body as FormData) : JSON.stringify(body),
      cache: "no-store",
      signal: AbortSignal.timeout(isForm ? 20000 : 8000),
    });
  } catch {
    return { ok: false, status: 503, message: "Layanan tidak dapat dihubungi" };
  }
  if (!res.ok) {
    let message = `Permintaan gagal (${res.status})`;
    try {
      const b = (await res.json()) as { error?: string };
      if (b.error) message = b.error;
    } catch {
      /* bukan JSON */
    }
    if (res.status === 503) message = "Layanan sedang tidak tersedia, coba lagi sebentar.";
    return { ok: false, status: res.status, message };
  }
  if (res.status === 204) return { ok: true, data: undefined as T };
  return { ok: true, data: (await res.json()) as T };
}

export const authedGet = <T>(path: string) => authedRequest<T>("GET", path);

