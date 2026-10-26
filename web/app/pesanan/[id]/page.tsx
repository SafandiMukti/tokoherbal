import Link from "next/link";
import { notFound, redirect } from "next/navigation";
import { cancelOrderAction } from "@/app/actions/shop";
import { ActionForm } from "@/components/ActionForm";
import { Icon } from "@/components/Icon";
import { ServiceDown } from "@/components/ServiceDown";
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
    if (await hasPendingRefresh()) return <ServiceDown title="Pesanan" />;
    redirect(`/login?next=${encodeURIComponent(here)}`);
  }
  if (session.role === "admin") redirect("/admin/pesanan");

  const res = await authedGet<Order>(`/orders/${n}`);
  if (!res.ok) {
    if (res.status === 401) redirect(`/login?next=${encodeURIComponent(here)}`);
    if (res.status === 404) notFound(); // juga untuk pesanan milik orang lain: tidak bocor
    return <ServiceDown title="Pesanan" message={res.message} />;
  }
  const o = res.data;
  const cancellable = o.status === "pending" || o.status === "paid";

  const flow = ["pending", "paid", "shipped", "completed"];
  const flowLabel = ["Dipesan", "Dibayar", "Dikirim", "Selesai"];
  const at = flow.indexOf(o.status);

  return (
    <div className="container page">
      <Link href="/pesanan" className="crumb">
        <Icon name="chevron-left" size={18} />
        Semua pesanan
      </Link>
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
      {o.status === "cancelled" ? (
        <p className="notice" role="status">
          Pesanan ini dibatalkan dan stok produk sudah dikembalikan.
        </p>
      ) : (
        <ol className="steps" aria-label="Tahap pesanan">
          {flow.map((st, idx) => (
            <li key={st} className={idx < at ? "done" : idx === at ? "current" : ""} aria-current={idx === at ? "step" : undefined}>
              {flowLabel[idx]}
            </li>
          ))}
        </ol>
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
      <p className="order-total-line cart-total">
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
    </div>
  );
}
