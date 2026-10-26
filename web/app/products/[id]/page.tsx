import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";
import { AddToCart, type Viewer } from "@/components/AddToCart";
import { Icon } from "@/components/Icon";
import { ProductImage } from "@/components/ProductCard";
import { ApiError, getProduct, type Product } from "@/lib/api";
import { getSession } from "@/lib/authed";
import { labelFor } from "@/lib/categories";
import { formatRupiah } from "@/lib/format";

export async function generateMetadata({ params }: { params: Promise<{ id: string }> }): Promise<Metadata> {
  const n = Number((await params).id);
  if (!Number.isInteger(n) || n <= 0) return {};
  try {
    return { title: (await getProduct(n)).name };
  } catch {
    return {};
  }
}

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
      <div className="container page">
        <p className="notice" role="alert">
          {e instanceof ApiError ? e.message : "Terjadi kesalahan"}. Muat ulang halaman ini beberapa saat lagi.
        </p>
      </div>
    );
  }

  const session = await getSession();
  const viewer: Viewer = !session ? "guest" : session.role === "admin" ? "admin" : "customer";
  const low = product.stock > 0 && product.stock <= 5;

  return (
    <div className="container page">
      <Link href="/" className="crumb">
        <Icon name="chevron-left" size={18} />
        Semua produk
      </Link>
      <article className="detail">
        <div className="detail-art">
          <ProductImage product={product} />
        </div>
        <div>
          <Link href={`/?category=${encodeURIComponent(product.category)}`} className="card-cat">
            {labelFor(product.category)}
          </Link>
          <h1>{product.name}</h1>
          <p className="price big">{formatRupiah(product.price)}</p>
          <p className="detail-desc">{product.description}</p>
          <p className="stock-line">
            <span className={`dot${product.stock === 0 ? " out" : low ? " low" : ""}`} aria-hidden="true" />
            {product.stock === 0 ? "Stok habis" : low ? `Stok tinggal ${product.stock}` : `Stok tersedia: ${product.stock}`}
          </p>
          <AddToCart productId={product.id} stock={product.stock} viewer={viewer} />
        </div>
      </article>
    </div>
  );
}
