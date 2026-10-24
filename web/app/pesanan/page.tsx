import Link from "next/link";
import { redirect } from "next/navigation";
import { authedGet, getSession, hasPendingRefresh } from "@/lib/authed";
import { formatDate, formatRupiah, statusLabel } from "@/lib/format";
import type { Order } from "@/lib/types";

export default async function OrdersPage() {
  const session = await getSession();
  if (!session) {
    if (await hasPendingRefresh()) {
      return (
        <>
          <h1>Pesanan saya</h1>
          <p className="notice" role="alert">
            Layanan login sedang tidak tersedia. Sesi Anda aman, coba muat ulang beberapa saat lagi.
          </p>
        </>
      );
    }
    redirect("/login?next=/pesanan");
  }

  const orders = await authedGet<Order[]>("/orders");
  if (!orders.ok && orders.status === 401) redirect("/login?next=/pesanan");

  return (
    <>
      <h1>Pesanan saya</h1>
      {!orders.ok ? (
        <p className="notice" role="alert">
          {orders.message}
        </p>
      ) : orders.data.length === 0 ? (
        <p className="muted">
          Belum ada pesanan. <Link href="/">Mulai belanja</Link>
        </p>
      ) : (
        <ul className="order-list">
          {orders.data.map((o) => (
            <li key={o.id}>
              <Link href={`/pesanan/${o.id}`}>Pesanan #{o.id}</Link>
              <span className={`status ${o.status}`}>{statusLabel(o.status)}</span>
              <span className="muted">{formatDate(o.created_at)}</span>
              <strong>{formatRupiah(o.total)}</strong>
            </li>
          ))}
        </ul>
      )}
    </>
  );
}
