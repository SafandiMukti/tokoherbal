import Link from "next/link";
import { redirect } from "next/navigation";
import { checkoutAction, removeCartItemAction, setCartQtyAction } from "@/app/actions/shop";
import { ActionForm } from "@/components/ActionForm";
import { authedGet, getSession, hasPendingRefresh } from "@/lib/authed";
import { imageSrc } from "@/lib/api";
import { formatRupiah } from "@/lib/format";
import type { Cart } from "@/lib/types";

export default async function CartPage() {
  const session = await getSession();
  if (!session) {
    if (await hasPendingRefresh()) return <Unavailable />;
    redirect("/login?next=/keranjang");
  }

  const cart = await authedGet<Cart>("/cart");
  if (!cart.ok) {
    if (cart.status === 401) redirect("/login?next=/keranjang");
    return <Unavailable message={cart.message} />;
  }
  const { items, total } = cart.data;
  const problem = items.find((i) => i.qty > i.stock);

  return (
    <>
      <h1>Keranjang</h1>
      {items.length === 0 ? (
        <p className="muted">
          Keranjang masih kosong. <Link href="/">Lihat produk</Link>
        </p>
      ) : (
        <>
          <ul className="cart-list">
            {items.map((i) => (
              <li key={i.product_id} className="cart-row">
                {i.image_url ? (
                  // eslint-disable-next-line @next/next/no-img-element
                  <img className="thumb" src={imageSrc(i.image_url)} alt="" />
                ) : (
                  <div className="thumb placeholder" aria-hidden="true">🌿</div>
                )}
                <div className="cart-info">
                  <Link href={`/products/${i.product_id}`}>{i.name}</Link>
                  <div className="muted">{formatRupiah(i.price)} · stok {i.stock}</div>
                  {i.qty > i.stock && (
                    <div className="warn">Jumlah melebihi stok. Kurangi sebelum memesan.</div>
                  )}
                </div>
                <ActionForm action={setCartQtyAction} className="inline">
                  <input type="hidden" name="product_id" value={i.product_id} />
                  <input
                    type="number"
                    name="qty"
                    defaultValue={i.qty}
                    min={1}
                    max={99}
                    aria-label={`Jumlah ${i.name}`}
                    required
                  />
                  <button type="submit" className="secondary">Ubah</button>
                </ActionForm>
                <strong className="subtotal">{formatRupiah(i.subtotal)}</strong>
                <ActionForm action={removeCartItemAction} className="inline">
                  <input type="hidden" name="product_id" value={i.product_id} />
                  <button type="submit" className="link-button">Hapus</button>
                </ActionForm>
              </li>
            ))}
          </ul>
          <p className="cart-total">
            Total: <strong>{formatRupiah(total)}</strong>
          </p>
          <ActionForm action={checkoutAction}>
            <button type="submit" disabled={Boolean(problem)}>
              Pesan sekarang
            </button>
          </ActionForm>
          <p className="muted">Pembayaran disimulasikan: admin yang menandai pesanan sebagai dibayar.</p>
        </>
      )}
    </>
  );
}

function Unavailable({ message }: { message?: string }) {
  return (
    <>
      <h1>Keranjang</h1>
      <p className="notice" role="alert">
        {message ?? "Layanan login sedang tidak tersedia. Sesi Anda aman, coba muat ulang beberapa saat lagi."}
      </p>
    </>
  );
}
