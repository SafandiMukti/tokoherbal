// proxy.ts (di Next.js 15 namanya middleware.ts) berjalan SEBELUM setiap halaman dirender.
// Tugasnya: memperbarui access token yang habis/hampir habis memakai refresh token,
// sehingga pengguna tidak tiba-tiba "keluar" setiap 15 menit.
import { NextResponse, type NextRequest } from "next/server";
import { refresh } from "@/lib/auth-api";
import {
  ACCESS_COOKIE,
  REFRESH_COOKIE,
  REFRESH_MAX_AGE,
  cookieOptions,
  decodeToken,
  type TokenResponse,
} from "@/lib/session";

type Outcome =
  | { kind: "ok"; tokens: TokenResponse }
  | { kind: "invalid" } // refresh token ditolak: sesi benar-benar berakhir
  | { kind: "unavailable" }; // auth-service mati: JANGAN mengeluarkan pengguna

// Refresh token berotasi dan auth-service mendeteksi pemakaian ulang (dianggap pencurian,
// semua sesi dicabut). Browser sering mengirim beberapa request sekaligus (halaman + prefetch),
// jadi tanpa penjagaan, request kedua memakai refresh token lama dan pengguna terlempar.
// Solusi: satu refresh per token (single-flight), hasilnya disimpan sebentar untuk
// request yang tiba terlambat.
const GRACE_MS = 15_000;
const g = globalThis as unknown as { __refreshFlights?: Map<string, Promise<Outcome>> };
const flights = (g.__refreshFlights ??= new Map());

function refreshOnce(refreshToken: string): Promise<Outcome> {
  const existing = flights.get(refreshToken);
  if (existing) return existing;

  const p = refresh(refreshToken).then<Outcome>((r) => {
    if (r.ok) return { kind: "ok", tokens: r.data };
    return r.status === 401 || r.status === 400 ? { kind: "invalid" } : { kind: "unavailable" };
  });
  flights.set(refreshToken, p);
  // hasil sukses/invalid disimpan sebentar; hasil "unavailable" dibuang agar bisa dicoba lagi
  void p.then((o) => setTimeout(() => flights.delete(refreshToken), o.kind === "unavailable" ? 0 : GRACE_MS));
  return p;
}

const REFRESH_BEFORE_SECONDS = 20;

export async function proxy(request: NextRequest) {
  const access = request.cookies.get(ACCESS_COOKIE)?.value;
  const refreshToken = request.cookies.get(REFRESH_COOKIE)?.value;
  if (!refreshToken) return NextResponse.next();

  const session = decodeToken(access);
  const now = Math.floor(Date.now() / 1000);
  if (session && session.exp - now > REFRESH_BEFORE_SECONDS) return NextResponse.next();

  const out = await refreshOnce(refreshToken);

  if (out.kind === "ok") {
    // perbarui cookie untuk request ini (agar halaman yang sedang dirender memakai token baru)
    request.cookies.set(ACCESS_COOKIE, out.tokens.access_token);
    request.cookies.set(REFRESH_COOKIE, out.tokens.refresh_token);
    const res = NextResponse.next({ request });
    // dan untuk browser (request berikutnya)
    res.cookies.set(ACCESS_COOKIE, out.tokens.access_token, cookieOptions(out.tokens.expires_in));
    res.cookies.set(REFRESH_COOKIE, out.tokens.refresh_token, cookieOptions(REFRESH_MAX_AGE));
    return res;
  }

  if (out.kind === "invalid") {
    request.cookies.delete(ACCESS_COOKIE);
    request.cookies.delete(REFRESH_COOKIE);
    const res = NextResponse.next({ request });
    res.cookies.delete(ACCESS_COOKIE);
    res.cookies.delete(REFRESH_COOKIE);
    return res;
  }

  // auth-service sedang mati: biarkan apa adanya. Halaman publik tetap jalan,
  // halaman yang butuh login akan menampilkan pesan "layanan tidak tersedia".
  return NextResponse.next();
}

export const config = {
  // lewati berkas statis, gambar, dan healthcheck
  matcher: ["/((?!_next/static|_next/image|favicon.ico|api/health).*)"],
};
