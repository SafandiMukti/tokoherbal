import Link from "next/link";
import { imageSrc, type Product } from "@/lib/api";
import { formatRupiah } from "@/lib/format";

export function ProductImage({ product }: { product: Product }) {
  const src = imageSrc(product.image_url);
  if (!src) return <div className="img placeholder" aria-hidden="true">🌿</div>;
  // <img> biasa: gambar berasal dari service lain, tidak perlu optimasi next/image
  // eslint-disable-next-line @next/next/no-img-element
  return <img className="img" src={src} alt={product.name} loading="lazy" />;
}

export function ProductCard({ product }: { product: Product }) {
  return (
    <Link href={`/products/${product.id}`} className="card">
      <ProductImage product={product} />
      <div className="card-body">
        <span className="tag">{product.category}</span>
        <h3>{product.name}</h3>
        <p className="price">{formatRupiah(product.price)}</p>
        {product.stock === 0 && <p className="muted">Stok habis</p>}
      </div>
    </Link>
  );
}
