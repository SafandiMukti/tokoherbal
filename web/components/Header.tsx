import Link from "next/link";
import { logoutAction } from "@/app/actions/auth";
import { getSession } from "@/lib/authed";
import { Brand } from "./Brand";
import { Icon } from "./Icon";

// Tiga keadaan: tamu, pelanggan, admin. Admin mengelola toko dan tidak berbelanja,
// jadi keranjang dan pesanan pribadi tidak ditampilkan untuknya.
export async function Header() {
  const session = await getSession();
  const isAdmin = session?.role === "admin";

  return (
    <header className="site-header">
      <div className="container header-inner">
        <Brand />
        <nav className="nav" aria-label="Menu utama">
          {!session && (
            <>
              <Link href="/login">Masuk</Link>
              <Link href="/register" className="btn small">
                Daftar
              </Link>
            </>
          )}
          {session && !isAdmin && (
            <>
              <Link href="/keranjang">
                <Icon name="cart" />
                <span className="nav-text">Keranjang</span>
              </Link>
              <Link href="/pesanan">
                <Icon name="box" />
                <span className="nav-text">Pesanan</span>
              </Link>
              <Link href="/akun">
                <Icon name="user" />
                <span className="nav-text">Akun</span>
              </Link>
            </>
          )}
          {isAdmin && (
            <>
              <span className="role-tag">Admin</span>
              <Link href="/admin/produk">Kelola produk</Link>
              <Link href="/admin/pesanan">Pesanan masuk</Link>
            </>
          )}
          {session && (
            <form action={logoutAction}>
              <button type="submit" className="link-button">
                Keluar
              </button>
            </form>
          )}
        </nav>
      </div>
    </header>
  );
}
