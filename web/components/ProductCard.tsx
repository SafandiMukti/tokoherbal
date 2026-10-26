import Link from "next/link";
import { imageSrc, type Product } from "@/lib/api";
import { labelFor } from "@/lib/categories";
import { formatRupiah } from "@/lib/format";
import { ProductArt } from "./ProductArt";

// Foto yang diunggah admin, atau ilustrasi sesuai kategori bila belum ada foto.
export function ProductImage({ product }: { product: Pick<Product, "image_url" | "category" | "name"> }) {
  const src = imageSrc(product.image_url);
  if (!src) return <ProductArt category={product.category} name={product.name} />;
  // <img> biasa: gambar berasal dari service lain, tidak perlu optimasi next/image
  // eslint-disable-next-line @next/next/no-img-element
  return <img className="photo" src={src} alt={product.name} loading="lazy" />;
}

export function ProductCard({ product }: { product: Product }) {
  const out = product.stock === 0;
  return (
    <Link href={`/products/${product.id}`} className={`card${out ? " is-out" : ""}`}>
      <ProductImage product={product} />
      <div className="card-body">
        <div className="card-cat">{labelFor(product.category)}</div>
        <h3 className="card-name">{product.name}</h3>
        <div className="card-foot">
          <p className="price">{formatRupiah(product.price)}</p>
          {out ? (
            <span className="stock-flag out">Habis</span>
          ) : product.stock <= 5 ? (
            <span className="stock-flag">Tinggal {product.stock}</span>
          ) : null}
        </div>
      </div>
    </Link>
  );
}
