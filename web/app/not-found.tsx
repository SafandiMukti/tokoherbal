import Link from "next/link";
import { Vessel } from "@/components/Vessel";

export default function NotFound() {
  return (
    <div className="container page">
      <div className="empty">
        <div className="empty-art">
          <Vessel category="jamu" />
        </div>
        <h1>Halaman tidak ditemukan</h1>
        <p className="muted">Alamatnya mungkin salah, atau produknya sudah dihapus.</p>
        <Link href="/" className="btn">
          Kembali ke daftar produk
        </Link>
      </div>
    </div>
  );
}
