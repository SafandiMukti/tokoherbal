import Link from "next/link";
import { notFound } from "next/navigation";
import { ApiError, getProduct, type Product } from "@/lib/api";
import { formatRupiah } from "@/lib/format";
import { AddToCart } from "@/components/AddToCart";
import { ProductImage } from "@/components/ProductCard";
import { getSession } from "@/lib/authed";

export default async function ProductPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const n = Number(id);
  if (!Number.isInteger(n) || n <= 0) notFound();

  let product: Product;
  try {
    product = await getProduct(n);
  } catch (e) {
    if (e instanceof ApiError && e.status === 404) notFound();
    return (
      <p className="notice" role="alert">
        {e instanceof ApiError ? e.message : "Terjadi kesalahan"}. Coba muat ulang beberapa saat lagi.
      </p>
    );
  }

  const loggedIn = (await getSession()) !== null;

  return (
    <>
      <p>
        <Link href="/">← Semua produk</Link>
      </p>
      <article className="detail">
        <ProductImage product={product} />
        <div>
          <span className="tag">{product.category}</span>
          <h1>{product.name}</h1>
          <p className="price big">{formatRupiah(product.price)}</p>
          <p>{product.description}</p>
          <p className="muted">{product.stock > 0 ? `Stok tersedia: ${product.stock}` : "Stok habis"}</p>
          <AddToCart productId={product.id} stock={product.stock} loggedIn={loggedIn} />
        </div>
      </article>
    </>
  );
}
