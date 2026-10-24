import Link from "next/link";
import { changeOrderStatusAction } from "@/app/actions/admin";
import { ActionForm } from "@/components/ActionForm";
import { guardAdmin } from "@/lib/admin";
import { authedGet } from "@/lib/authed";
import { formatDate, formatRupiah, statusLabel } from "@/lib/format";
import type { Order } from "@/lib/types";
import { redirect } from "next/navigation";

const LIMIT = 20;
const STATUSES = ["pending", "paid", "shipped", "completed", "cancelled"];

// status berikutnya yang boleh dipilih admin dari status sekarang (sama dengan aturan di product-service)
const NEXT: Record<string, { to: string; label: string; danger?: boolean }[]> = {
  pending: [
    { to: "paid", label: "Tandai dibayar" },
    { to: "cancelled", label: "Batalkan", danger: true },
  ],
  paid: [
    { to: "shipped", label: "Tandai dikirim" },
    { to: "cancelled", label: "Batalkan", danger: true },
  ],
  shipped: [{ to: "completed", label: "Tandai selesai" }],
};

type SP = Promise<Record<string, string | string[] | undefined>>;
const first = (v: string | string[] | undefined) => (Array.isArray(v) ? v[0] : v) ?? "";

export default async function AdminOrders({ searchParams }: { searchParams: SP }) {
  const blocked = await guardAdmin("/admin/pesanan");
  if (blocked) return blocked;

  const sp = await searchParams;
  const status = STATUSES.includes(first(sp.status)) ? first(sp.status) : "";
  const page = Math.max(1, parseInt(first(sp.page), 10) || 1);

  const qs = new URLSearchParams({ page: String(page), limit: String(LIMIT) });
  if (status) qs.set("status", status);
  const res = await authedGet<Order[]>(`/admin/orders?${qs}`);
  if (!res.ok && res.status === 401) redirect("/login?next=/admin/pesanan");

  const href = (p: number) => {
    const q = new URLSearchParams();
    if (status) q.set("status", status);
    if (p > 1) q.set("page", String(p));
    const s = q.toString();
    return s ? `/admin/pesanan?${s}` : "/admin/pesanan";
  };

  return (
    <>
      <div className="row-between">
        <h1>Semua pesanan</h1>
        <Link href="/admin/produk">← Kelola produk</Link>
      </div>
      <nav className="chips" aria-label="Filter status">
        <Link href="/admin/pesanan" className={status === "" ? "active" : ""}>
          Semua
        </Link>
        {STATUSES.map((s) => (
          <Link key={s} href={`/admin/pesanan?status=${s}`} className={status === s ? "active" : ""}>
            {statusLabel(s)}
          </Link>
        ))}
      </nav>

      {!res.ok ? (
        <p className="notice" role="alert">
          {res.status === 403 ? "Akun Anda tidak punya akses admin." : res.message}
        </p>
      ) : res.data.length === 0 ? (
        <p className="muted">Tidak ada pesanan.</p>
      ) : (
        <>
          <div className="table-wrap">
        <table className="table">
            <thead>
              <tr>
                <th>#</th>
                <th>Pengguna</th>
                <th>Tanggal</th>
                <th className="num">Total</th>
                <th>Status</th>
                <th>Tindakan</th>
              </tr>
            </thead>
            <tbody>
              {res.data.map((o) => (
                <tr key={o.id}>
                  <td>{o.id}</td>
                  <td>{o.user_id}</td>
                  <td>{formatDate(o.created_at)}</td>
                  <td className="num">{formatRupiah(o.total)}</td>
                  <td>
                    <span className={`status ${o.status}`}>{statusLabel(o.status)}</span>
                  </td>
                  <td>
                    <div className="row-gap">
                      {(NEXT[o.status] ?? []).map((n) => (
                        <ActionForm key={n.to} action={changeOrderStatusAction} className="inline">
                          <input type="hidden" name="order_id" value={o.id} />
                          <input type="hidden" name="status" value={n.to} />
                          <button type="submit" className={n.danger ? "danger small" : "secondary small"}>
                            {n.label}
                          </button>
                        </ActionForm>
                      ))}
                      {!NEXT[o.status] && <span className="muted">—</span>}
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
          <nav className="pagination" aria-label="Halaman">
            {page > 1 ? <Link href={href(page - 1)}>← Sebelumnya</Link> : <span />}
            <span>Halaman {page}</span>
            {res.data.length === LIMIT ? <Link href={href(page + 1)}>Berikutnya →</Link> : <span />}
          </nav>
        </>
      )}
    </>
  );
}
