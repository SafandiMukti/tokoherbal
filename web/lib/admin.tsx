import type { ReactNode } from "react";
import { redirect } from "next/navigation";
import { getSession, hasPendingRefresh } from "./authed";

// Penjaga halaman admin: mengembalikan null bila boleh lanjut, atau tampilan pengganti.
// Ini hanya kenyamanan pengguna; yang benar-benar menolak non-admin adalah API (403).
export async function guardAdmin(here: string): Promise<ReactNode | null> {
  const session = await getSession();
  if (!session) {
    if (await hasPendingRefresh()) {
      return (
        <p className="notice" role="alert">
          Layanan login sedang tidak tersedia. Sesi Anda aman, coba muat ulang beberapa saat lagi.
        </p>
      );
    }
    redirect(`/login?next=${encodeURIComponent(here)}`);
  }
  if (session.role !== "admin") {
    return (
      <>
        <h1>Khusus admin</h1>
        <p className="notice" role="alert">
          Akun Anda tidak punya akses ke halaman ini.
        </p>
      </>
    );
  }
  return null;
}
