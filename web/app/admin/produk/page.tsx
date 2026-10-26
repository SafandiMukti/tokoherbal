import Link from "next/link";
import { ApiError, listProducts } from "@/lib/api";
import { guardAdmin } from "@/lib/admin";
import { formatRupiah } from "@/lib/format";
import { ProductImage } from "@/components/ProductCard";
import { labelFor } from "@/lib/categories";
import { Pagination } from "@/components/Pagination";

type SP = Promise<Record<string, string | string[] | undefined>>;
const first = (v: string | string[] | undefined) => (Array.isArray(v) ? v[0] : v) ?? "";

export default async function AdminProducts({ searchParams }: { searchParams: SP }) {
  const blocked = await guardAdmin("/admin/produk");
  if (blocked) return blocked;

  const sp = await searchParams;
  const page = Math.max(1, parseInt(first(sp.page), 10) || 1);

  let list;
  try {
    list = await listProducts({ page, sort: "newest" });
  } catch (e) {
    return (
      <p className="notice" role="alert">
        {e instanceof ApiError ? e.message : "Terjadi kesalahan"}
      </p>
    );
  }

  return (
    <>
      <div className="row-between">
        <h1>Kelola produk</h1>
        <div className="row-gap">
          <Link href="/admin/produk/baru" className="btn">
            + Tambah produk
          </Link>
        </div>
      </div>
      {first(sp.dihapus) && (
        <p className="notice ok" role="status">
          Produk dihapus.
        </p>
      )}
      <div className="table-wrap">
        <table className="table">
        <thead>
          <tr>
            <th>Produk</th>
            <th>Kategori</th>
            <th className="num">Harga</th>
            <th className="num">Stok</th>
            <th />
          </tr>
        </thead>
        <tbody>
          {list.items.map((p) => (
            <tr key={p.id}>
              <td>
                <div className="prod-cell">
                  <div className="mini">
                    <ProductImage product={p} />
                  </div>
                  <span>{p.name}</span>
                </div>
              </td>
              <td>{labelFor(p.category)}</td>
              <td className="num">{formatRupiah(p.price)}</td>
              <td className="num">{p.stock}</td>
              <td>
                <Link href={`/admin/produk/${p.id}`}>Ubah</Link>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
        </div>
      <Pagination page={list.page} limit={list.limit} total={list.total} params={{}} path="/admin/produk" />
    </>
  );
}
