// Konstanta dan helper sesi. Token disimpan di cookie httpOnly: JavaScript di
// browser TIDAK bisa membacanya, jadi pencurian lewat XSS tidak mungkin.

export const ACCESS_COOKIE = "th_access";
export const REFRESH_COOKIE = "th_refresh";
export const REFRESH_MAX_AGE = 7 * 24 * 60 * 60; // sama dengan umur refresh token di auth-service

export type TokenResponse = {
  access_token: string;
  refresh_token: string;
  token_type: string;
  expires_in: number; // detik
};

export type Session = { userId: number; role: string; exp: number };

// Opsi cookie. COOKIE_SECURE=true wajib saat memakai HTTPS (produksi sungguhan);
// di http://localhost cookie "secure" akan ditolak browser, jadi default-nya mati.
export function cookieOptions(maxAge: number) {
  return {
    httpOnly: true,
    sameSite: "lax" as const,
    secure: process.env.COOKIE_SECURE === "true",
    path: "/",
    maxAge,
  };
}

// Membaca isi token TANPA memverifikasi tanda tangan. Itu aman di sini karena hasilnya
// hanya dipakai untuk tampilan dan jadwal refresh; keputusan akses tetap diambil
// auth-service lewat /verify setiap kali token dipakai.
export function decodeToken(token: string | undefined): Session | null {
  if (!token) return null;
  const parts = token.split(".");
  if (parts.length !== 3) return null;
  try {
    const p = JSON.parse(Buffer.from(parts[1], "base64url").toString("utf8")) as {
      sub?: string;
      role?: string;
      exp?: number;
    };
    const userId = Number(p.sub);
    if (!Number.isInteger(userId) || userId <= 0 || !p.role || !p.exp) return null;
    return { userId, role: p.role, exp: p.exp };
  } catch {
    return null;
  }
}

// next hanya boleh berupa path di situs ini ("/akun"), bukan "//evil.com" atau "https://...".
export function safeNext(next: string | null | undefined): string {
  if (!next || !next.startsWith("/") || next.startsWith("//") || next.includes("\\")) return "/";
  return next;
}
