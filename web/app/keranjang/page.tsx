import type { Metadata } from "next";
import Link from "next/link";
import { redirect } from "next/navigation";
import { checkoutAction, removeCartItemAction, setCartQtyAction } from "@/app/actions/shop";
import { ActionForm } from "@/components/ActionForm";
import { CartThumb } from "@/components/CartThumb";
import { QtyInput } from "@/components/QtyInput";
import { ServiceDown } from "@/components/ServiceDown";
import { authedGet, getSession, hasPendingRefresh } from "@/lib/authed";
import { formatRupiah } from "@/lib/format";
import type { Cart } from "@/lib/types";

export const metadata: Metadata = { title: "Keranjang" };

export default async function CartPage() {
  const session = await getSession();
  if (!session) {
    if (await hasPendingRefresh()) return <ServiceDown title="Keranjang" />;
    redirect("/login?next=/keranjang");
  }
  if (session.role === "admin") redirect("/admin/pesanan"); // admin tidak berbelanja

  const cart = await authedGet<Cart>("/cart");
  if (!cart.ok) {
    if (cart.status === 401) redirect("/login?next=/keranjang");
    return <ServiceDown title="Keranjang" message={cart.message} />;
  }
  const { items, total } = cart.data;
  const problem = items.find((i) => i.qty > i.stock);
  const count = items.reduce((n, i) => n + i.qty, 0);

  return (
    <div className="container page">
      <h1>Keranjang</h1>
      {items.length === 0 ? (
        <div className="empty">
          <h2>Keranjang masih kosong</h2>
          <p className="muted">Pilih produk yang kamu suka, lalu taruh di sini.</p>
          <Link href="/" className="btn">
            Lihat produk
          </Link>
        </div>
      ) : (
        <div className="cart-layout">
          <ul className="cart-list">
            {items.map((i) => (
              <li key={i.product_id} className="cart-row">
                <CartThumb imageUrl={i.image_url} name={i.name} />
                <div className="cart-info">
                  <Link href={`/products/${i.product_id}`}>{i.name}</Link>
                  <div className="muted">
                    {formatRupiah(i.price)} per buah, stok {i.stock}
                  </div>
                  {i.qty > i.stock && <div className="warn">Jumlah melebihi stok. Kurangi sebelum memesan.</div>}
                </div>
                <div className="cart-edit">
                  <ActionForm action={setCartQtyAction}>
                    <input type="hidden" name="product_id" value={i.product_id} />
                    <QtyInput key={i.qty} label={`Jumlah ${i.name}`} defaultValue={i.qty} max={99} />
                    <button type="submit" className="secondary small">
                      Ubah
                    </button>
                  </ActionForm>
                </div>
                <div className="cart-sub">
                  <strong className="subtotal">{formatRupiah(i.subtotal)}</strong>
                  <ActionForm action={removeCartItemAction}>
                    <input type="hidden" name="product_id" value={i.product_id} />
                    <button type="submit" className="link-button">
                      Hapus
                    </button>
                  </ActionForm>
                </div>
              </li>
            ))}
          </ul>
          <aside className="summary" aria-label="Ringkasan belanja">
            <h2>Ringkasan</h2>
            <p className="cart-total">
              Total ({count} barang): <strong>{formatRupiah(total)}</strong>
            </p>
            <ActionForm action={checkoutAction}>
              <button type="submit" disabled={Boolean(problem)}>
                Pesan sekarang
              </button>
            </ActionForm>
            <p className="small-print">Pembayaran disimulasikan: admin yang menandai pesanan sebagai dibayar.</p>
          </aside>
        </div>
      )}
    </div>
  );
}
