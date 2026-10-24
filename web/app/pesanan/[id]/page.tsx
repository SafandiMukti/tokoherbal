import Link from "next/link";
import { notFound, redirect } from "next/navigation";
import { cancelOrderAction } from "@/app/actions/shop";
import { ActionForm } from "@/components/ActionForm";
import { authedGet, getSession, hasPendingRefresh } from "@/lib/authed";
import { formatDate, formatRupiah, statusLabel } from "@/lib/format";
import type { Order } from "@/lib/types";

export default async function OrderPage({
  params,
  searchParams,
}: {
  params: Promise<{ id: string }>;
  searchParams: Promise<{ baru?: string }>;
}) {
  const { id } = await params;
  const { baru } = await searchParams;
  const n = Number(id);
  if (!Number.isInteger(n) || n <= 0) notFound();

  const here = `/pesanan/${n}`;
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

  const res = await authedGet<Order>(`/orders/${n}`);
  if (!res.ok) {
    if (res.status === 401) redirect(`/login?next=${encodeURIComponent(here)}`);
    if (res.status === 404) notFound(); // juga untuk pesanan milik orang lain: tidak bocor
    return (
      <p className="notice" role="alert">
        {res.message}
      </p>
    );
  }
  const o = res.data;
  const cancellable = o.status === "pending" || o.status === "paid";

  return (
    <>
      <p>
        <Link href="/pesanan">← Semua pesanan</Link>
      </p>
      {baru && (
        <p className="notice ok" role="status">
          Pesanan berhasil dibuat. Terima kasih!
        </p>
      )}
      <h1>Pesanan #{o.id}</h1>
      <p>
        <span className={`status ${o.status}`}>{statusLabel(o.status)}</span>{" "}
        <span className="muted">{formatDate(o.created_at)}</span>
      </p>
      {o.status === "cancelled" && (
        <p className="notice" role="status">
          Pesanan ini dibatalkan dan stok produk sudah dikembalikan.
        </p>
      )}
      <ul className="order-items">
        {o.items.map((i) => (
          <li key={i.product_id}>
            <span>
              {i.name} × {i.qty}
            </span>
            <span>{formatRupiah(i.price * i.qty)}</span>
          </li>
        ))}
      </ul>
      <p className="cart-total">
        Total: <strong>{formatRupiah(o.total)}</strong>
      </p>
      {cancellable && (
        <ActionForm action={cancelOrderAction}>
          <input type="hidden" name="order_id" value={o.id} />
          <button type="submit" className="danger">
            Batalkan pesanan
          </button>
        </ActionForm>
      )}
    </>
  );
}
