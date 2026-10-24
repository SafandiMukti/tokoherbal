// Form GET biasa: tanpa JavaScript di browser, URL-nya bisa dibagikan dan di-bookmark.
export function Filters({
  q,
  category,
  sort,
  categories,
}: {
  q: string;
  category: string;
  sort: string;
  categories: string[];
}) {
  return (
    <form className="filters" action="/" method="get">
      <input type="search" name="q" defaultValue={q} placeholder="Cari produk…" aria-label="Cari produk" />
      <select name="category" defaultValue={category} aria-label="Kategori">
        <option value="">Semua kategori</option>
        {categories.map((c) => (
          <option key={c} value={c}>
            {c}
          </option>
        ))}
      </select>
      <select name="sort" defaultValue={sort} aria-label="Urutkan">
        <option value="newest">Terbaru</option>
        <option value="price_asc">Harga termurah</option>
        <option value="price_desc">Harga termahal</option>
        <option value="name">Nama A–Z</option>
      </select>
      <button type="submit">Terapkan</button>
    </form>
  );
}
