import { ApiError, listCategories, listProducts } from "@/lib/api";
import { Filters } from "@/components/Filters";
import { Pagination } from "@/components/Pagination";
import { ProductCard } from "@/components/ProductCard";

type SearchParams = Promise<Record<string, string | string[] | undefined>>;

const first = (v: string | string[] | undefined) => (Array.isArray(v) ? v[0] : v) ?? "";

export default async function Home({ searchParams }: { searchParams: SearchParams }) {
  const sp = await searchParams;
  const q = first(sp.q).trim();
  const category = first(sp.category);
  const sort = first(sp.sort) || "newest";
  const page = Math.max(1, parseInt(first(sp.page), 10) || 1);

  try {
    const [list, categories] = await Promise.all([
      listProducts({ q, category, sort, page }),
      listCategories().catch(() => [] as string[]), // filter kategori opsional
    ]);

    const params: Record<string, string> = {};
    if (q) params.q = q;
    if (category) params.category = category;
    if (sort !== "newest") params.sort = sort;

    return (
      <>
        <h1>Produk</h1>
        <Filters q={q} category={category} sort={sort} categories={categories} />
        {list.items.length === 0 ? (
          <p className="muted">Tidak ada produk yang cocok.</p>
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
    return (
      <>
        <h1>Produk</h1>
        <p className="notice" role="alert">
          {msg}. Coba muat ulang beberapa saat lagi.
        </p>
      </>
    );
  }
}
