import type { Product } from "@/lib/api";

// Isian produk, dipakai oleh form tambah dan form ubah.
export function ProductFields({ product }: { product?: Product }) {
  return (
    <div className="form-grid">
      {product && <input type="hidden" name="id" value={product.id} />}
      <label>
        Nama
        <input name="name" defaultValue={product?.name} required maxLength={120} />
      </label>
      <label>
        Kategori
        <input name="category" defaultValue={product?.category} required maxLength={50} placeholder="jamu, madu, rempah…" />
      </label>
      <label>
        Harga (Rp)
        <input type="number" name="price" defaultValue={product?.price ?? 0} min={0} step={1} required />
      </label>
      <label>
        Stok
        <input type="number" name="stock" defaultValue={product?.stock ?? 0} min={0} step={1} required />
      </label>
      <label className="wide">
        Deskripsi
        <textarea name="description" defaultValue={product?.description} rows={4} maxLength={2000} />
      </label>
    </div>
  );
}
