import Link from "next/link";
import { redirect } from "next/navigation";
import { getSession, hasPendingRefresh } from "@/lib/authed";

export default async function AccountPage() {
  const session = await getSession();
  if (!session) {
    // masih punya sesi tetapi belum bisa diperbarui -> auth-service sedang bermasalah, bukan "belum login"
    if (await hasPendingRefresh()) {
      return (
        <>
          <h1>Akun saya</h1>
          <p className="notice" role="alert">
            Layanan login sedang tidak tersedia. Sesi Anda aman, coba muat ulang beberapa saat lagi.
          </p>
        </>
      );
    }
    redirect("/login?next=/akun");
  }

  return (
    <>
      <h1>Akun saya</h1>
      <p>
        Nomor pengguna: <strong>{session.userId}</strong> · Peran: <span className="tag">{session.role}</span>
      </p>
      <ul>
        <li>
          <Link href="/keranjang">Keranjang</Link>
        </li>
        <li>
          <Link href="/pesanan">Pesanan saya</Link>
        </li>
      </ul>
    </>
  );
}
