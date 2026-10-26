import type { Metadata } from "next";
import Link from "next/link";
import { redirect } from "next/navigation";
import { ServiceDown } from "@/components/ServiceDown";
import { authedGet, getSession, hasPendingRefresh } from "@/lib/authed";
import { formatDate, formatRupiah, statusLabel } from "@/lib/format";
import type { Order } from "@/lib/types";

export const metadata: Metadata = { title: "Pesanan saya" };

export default async function OrdersPage() {
  const session = await getSession();
  if (!session) {
    if (await hasPendingRefresh()) return <ServiceDown title="Pesanan saya" />;
    redirect("/login?next=/pesanan");
  }
  if (session.role === "admin") redirect("/admin/pesanan");

  const orders = await authedGet<Order[]>("/orders");
  if (!orders.ok && orders.status === 401) redirect("/login?next=/pesanan");

  return (
    <div className="container page">
      <h1>Pesanan saya</h1>
      {!orders.ok ? (
        <p className="notice" role="alert">
          {orders.message}
        </p>
      ) : orders.data.length === 0 ? (
        <div className="empty">
          <h2>Belum ada pesanan</h2>
          <p className="muted">Pesanan yang kamu buat akan muncul di sini.</p>
          <Link href="/" className="btn">
            Mulai belanja
          </Link>
        </div>
      ) : (
        <ul className="order-list">
          {orders.data.map((o) => (
            <li key={o.id}>
              <Link href={`/pesanan/${o.id}`}>Pesanan #{o.id}</Link>
              <span className={`status ${o.status}`}>{statusLabel(o.status)}</span>
              <span className="order-meta">{formatDate(o.created_at)}</span>
              <strong className="order-total">{formatRupiah(o.total)}</strong>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
