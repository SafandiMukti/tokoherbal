import Link from "next/link";
import { logoutAction } from "@/app/actions/auth";
import { getSession } from "@/lib/authed";

export async function Header() {
  const session = await getSession();
  return (
    <header className="site-header">
      <Link href="/" className="brand">
        🌿 TokoHerbal
      </Link>
      <nav className="header-nav">
        {session ? (
          <>
            {session.role === "admin" && <Link href="/admin/produk">Admin</Link>}
            <Link href="/keranjang">Keranjang</Link>
            <Link href="/pesanan">Pesanan</Link>
            <Link href="/akun">Akun</Link>
            <form action={logoutAction}>
              <button type="submit" className="link-button">
                Keluar
              </button>
            </form>
          </>
        ) : (
          <>
            <Link href="/login">Masuk</Link>
            <Link href="/register">Daftar</Link>
          </>
        )}
      </nav>
    </header>
  );
}
