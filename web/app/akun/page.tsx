import type { Metadata } from "next";
import Link from "next/link";
import { redirect } from "next/navigation";
import { ServiceDown } from "@/components/ServiceDown";
import { getSession, hasPendingRefresh } from "@/lib/authed";

export const metadata: Metadata = { title: "Akun saya" };

export default async function AccountPage() {
  const session = await getSession();
  if (!session) {
    // masih punya sesi tetapi belum bisa diperbarui -> auth-service bermasalah, bukan "belum login"
    if (await hasPendingRefresh()) return <ServiceDown title="Akun saya" />;
    redirect("/login?next=/akun");
  }
  const isAdmin = session.role === "admin";

  return (
    <div className="container page">
      <h1>Akun saya</h1>
      <div className="panel">
        <p>
          Nomor pengguna: <strong>{session.userId}</strong>. Peran: <span className="role-tag">{isAdmin ? "Admin" : "Pelanggan"}</span>
        </p>
        {isAdmin ? (
          <>
            <p className="muted">Akun admin dipakai untuk mengelola toko, bukan untuk berbelanja.</p>
            <div className="row-gap">
              <Link href="/admin/produk" className="btn">
                Kelola produk
              </Link>
              <Link href="/admin/pesanan" className="btn secondary">
                Pesanan masuk
              </Link>
            </div>
          </>
        ) : (
          <div className="row-gap">
            <Link href="/keranjang" className="btn">
              Keranjang
            </Link>
            <Link href="/pesanan" className="btn secondary">
              Pesanan saya
            </Link>
          </div>
        )}
      </div>
    </div>
  );
}
