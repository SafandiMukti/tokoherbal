import { Icon } from "@/components/Icon";
import Link from "next/link";
import { createProductAction } from "@/app/actions/admin";
import { ActionForm } from "@/components/ActionForm";
import { ProductFields } from "@/components/ProductForm";
import { guardAdmin } from "@/lib/admin";

export default async function NewProduct() {
  const blocked = await guardAdmin("/admin/produk/baru");
  if (blocked) return blocked;
  return (
    <section className="narrow-wide">
      <Link href="/admin/produk" className="crumb">
        <Icon name="chevron-left" size={18} />
        Kelola produk
      </Link>
      <h1>Tambah produk</h1>
      <ActionForm action={createProductAction}>
        <ProductFields />
        <button type="submit">Simpan</button>
      </ActionForm>
      <p className="muted">Gambar bisa diunggah setelah produk disimpan.</p>
    </section>
  );
}
