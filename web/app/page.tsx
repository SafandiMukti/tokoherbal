import Link from "next/link";
import { ApiError, listCategories, listProducts } from "@/lib/api";
import { labelFor } from "@/lib/categories";
import { Hero } from "@/components/Hero";
import { Icon } from "@/components/Icon";
import { Pagination } from "@/components/Pagination";
import { ProductCard } from "@/components/ProductCard";

type SearchParams = Promise<Record<string, string | string[] | undefined>>;

const first = (v: string | string[] | undefined) => (Array.isArray(v) ? v[0] : v) ?? "";

const SORTS: [string, string][] = [
  ["newest", "Terbaru"],
  ["price_asc", "Harga termurah"],
  ["price_desc", "Harga termahal"],
  ["name", "Nama A sampai Z"],
];

export default async function Home({ searchParams }: { searchParams: SearchParams }) {
  const sp = await searchParams;
  const q = first(sp.q).trim();
  const category = first(sp.category);
  const sort = first(sp.sort) || "newest";
  const page = Math.max(1, parseInt(first(sp.page), 10) || 1);
  const filtered = Boolean(q || category || page > 1);

  const categories = await listCategories().catch(() => [] as string[]); // rak tetap tampil dengan bawaan

  let body: React.ReactNode;
  try {
    const list = await listProducts({ q, category, sort, page });

    const params: Record<string, string> = {};
    if (q) params.q = q;
    if (category) params.category = category;
    if (sort !== "newest") params.sort = sort;

    // tautan untuk melepas satu filter
    const without = (drop: "q" | "category") => {
      const p = new URLSearchParams();
      if (q && drop !== "q") p.set("q", q);
      if (category && drop !== "category") p.set("category", category);
      if (sort !== "newest") p.set("sort", sort);
      const s = p.toString();
      return s ? `/?${s}` : "/";
    };

    body = (
      <>
        <div className="toolbar">
          <h2>{category ? labelFor(category) : q ? "Hasil pencarian" : "Semua produk"}</h2>
          <form action="/" method="get">
            {q && <input type="hidden" name="q" value={q} />}
            {category && <input type="hidden" name="category" value={category} />}
            <label htmlFor="sort" className="muted">
              {list.total} produk, urutkan
            </label>
            <select id="sort" name="sort" defaultValue={sort}>
              {SORTS.map(([v, l]) => (
                <option key={v} value={v}>
                  {l}
                </option>
              ))}
            </select>
            <button type="submit" className="secondary small">
              Terapkan
            </button>
          </form>
        </div>

        {(q || category) && (
          <div className="active-filters">
            {category && (
              <Link href={without("category")} className="chip" aria-label={`Hapus filter kategori ${category}`}>
                Kategori: {labelFor(category)}
                <Icon name="close" size={14} />
              </Link>
            )}
            {q && (
              <Link href={without("q")} className="chip" aria-label={`Hapus pencarian ${q}`}>
                Kata kunci: {q}
                <Icon name="close" size={14} />
              </Link>
            )}
          </div>
        )}

        {list.items.length === 0 ? (
          <div className="empty">
            <h2>Tidak ada produk yang cocok</h2>
            <p className="muted">Coba kata kunci lain, atau lepas filter yang aktif.</p>
            <Link href="/" className="btn secondary">
              Lihat semua produk
            </Link>
          </div>
        ) : (
          <div className="grid">
            {list.items.map((p) => (
              <ProductCard key={p.id} product={p} />
            ))}
          </div>
        )}
        <Pagination page={list.page} limit={list.limit} total={list.total} params={params} />
      </>
    );
  } catch (e) {
    const msg = e instanceof ApiError ? e.message : "Terjadi kesalahan";
    body = (
      <p className="notice" role="alert">
        {msg}. Muat ulang halaman ini beberapa saat lagi.
      </p>
    );
  }

  return (
    <>
      <Hero q={q} category={category} categories={categories} compact={filtered} />
      <div className="container page">{body}</div>
    </>
  );
}
