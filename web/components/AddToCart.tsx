import Link from "next/link";
import { addToCartAction } from "@/app/actions/shop";
import { ActionForm } from "./ActionForm";
import { QtyInput } from "./QtyInput";

export type Viewer = "guest" | "customer" | "admin";

export function AddToCart({ productId, stock, viewer }: { productId: number; stock: number; viewer: Viewer }) {
  // Admin mengelola toko dan tidak berbelanja (API juga menolaknya dengan 403).
  if (viewer === "admin") {
    return (
      <div className="admin-note">
        <p>Akun admin tidak dipakai untuk berbelanja. Produk ini dikelola dari panel admin.</p>
        <Link href={`/admin/produk/${productId}`} className="btn secondary">
          Ubah produk
        </Link>
      </div>
    );
  }
  if (stock <= 0) return null;
  if (viewer === "guest") {
    return (
      <p>
        <Link href={`/login?next=/products/${productId}`} className="btn">
          Masuk untuk membeli
        </Link>
      </p>
    );
  }
  return (
    <ActionForm action={addToCartAction} className="add-to-cart">
      <input type="hidden" name="product_id" value={productId} />
      <div className="field">
        <label>Jumlah</label>
        <QtyInput label="Jumlah" max={Math.min(stock, 99)} />
      </div>
      <button type="submit">Tambah ke keranjang</button>
    </ActionForm>
  );
}
