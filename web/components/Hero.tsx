import Link from "next/link";
import { DEFAULT_SHELF, labelFor } from "@/lib/categories";
import { Icon } from "./Icon";
import { Vessel } from "./Vessel";

// Hero: judul, pencarian, dan rak botol. Tiap botol adalah satu kategori; mengkliknya
// memfilter katalog, mengkliknya lagi melepas filter.
export function Hero({
  q,
  category,
  categories,
  compact,
}: {
  q: string;
  category: string;
  categories: string[];
  compact: boolean;
}) {
  const shelf = (categories.length ? categories : DEFAULT_SHELF).slice(0, 6);
  return (
    <section className={`hero on-dark${compact ? " compact" : ""}`} aria-label="Cari produk">
      <div className="container hero-inner">
        <h1>Ramuan tradisional, tinggal pilih.</h1>
        <p className="hero-sub">Jamu, madu, minyak, dan perawatan herbal. Cari, taruh di keranjang, lalu pesan.</p>
        <form className="search" action="/" method="get" role="search">
          {category && <input type="hidden" name="category" value={category} />}
          <input type="search" name="q" defaultValue={q} placeholder="Cari jamu, madu, teh…" aria-label="Cari produk" />
          <button type="submit">
            <Icon name="search" size={18} />
            Cari
          </button>
        </form>
        <div className="shelf-wrap">
          <nav className="shelf" aria-label="Kategori">
            {shelf.map((c) => {
              const active = c === category;
              return (
                <Link key={c} href={active ? "/" : `/?category=${encodeURIComponent(c)}`} className="shelf-item" aria-current={active ? "true" : undefined}>
                  <Vessel category={c} className="vessel" />
                  <span className="shelf-label">{labelFor(c)}</span>
                </Link>
              );
            })}
          </nav>
        </div>
      </div>
    </section>
  );
}
