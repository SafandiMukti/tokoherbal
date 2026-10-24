import Link from "next/link";
import { notFound } from "next/navigation";
import { deleteProductAction, updateProductAction, uploadImageAction } from "@/app/actions/admin";
import { ActionForm } from "@/components/ActionForm";
import { ProductImage } from "@/components/ProductCard";
import { ProductFields } from "@/components/ProductForm";
import { ApiError, getProduct, type Product } from "@/lib/api";
import { guardAdmin } from "@/lib/admin";

export default async function EditProduct({
  params,
  searchParams,
}: {
  params: Promise<{ id: string }>;
  searchParams: Promise<{ baru?: string }>;
}) {
  const { id } = await params;
  const { baru } = await searchParams;
  const n = Number(id);
  if (!Number.isInteger(n) || n <= 0) notFound();

  const blocked = await guardAdmin(`/admin/produk/${n}`);
  if (blocked) return blocked;

  let product: Product;
  try {
    product = await getProduct(n);
  } catch (e) {
    if (e instanceof ApiError && e.status === 404) notFound();
    return (
      <p className="notice" role="alert">
        {e instanceof ApiError ? e.message : "Terjadi kesalahan"}
      </p>
    );
  }

  return (
    <section className="narrow-wide">
      <p>
        <Link href="/admin/produk">← Kelola produk</Link>
      </p>
      <h1>Ubah produk #{product.id}</h1>
      {baru && (
        <p className="notice ok" role="status">
          Produk dibuat. Anda bisa mengunggah gambarnya di bawah.
        </p>
      )}

      <ActionForm action={updateProductAction}>
        <ProductFields product={product} />
        <button type="submit">Simpan perubahan</button>
      </ActionForm>

      <h2>Gambar</h2>
      <div className="image-admin">
        <ProductImage product={product} />
        <ActionForm action={uploadImageAction}>
          <input type="hidden" name="id" value={product.id} />
          <input type="file" name="image" accept="image/jpeg,image/png,image/webp" required />
          <button type="submit">Unggah</button>
          <p className="muted">JPG, PNG, atau WebP. Maksimal 2 MB.</p>
        </ActionForm>
      </div>

      <h2>Hapus</h2>
      <ActionForm action={deleteProductAction}>
        <input type="hidden" name="id" value={product.id} />
        <button type="submit" className="danger">
          Hapus produk
        </button>
        <p className="muted">Produk yang pernah dipesan tidak bisa dihapus; ubah stoknya menjadi 0.</p>
      </ActionForm>
    </section>
  );
}
