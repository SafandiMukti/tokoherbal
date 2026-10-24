"use server";

import { revalidatePath } from "next/cache";
import { redirect } from "next/navigation";
import { authedRequest, type AuthedResult } from "@/lib/authed";
import type { Order, ShopState } from "@/lib/types";
import type { Product } from "@/lib/api";

const MAX_IMAGE = 2 * 1024 * 1024; // sama dengan batas di product-service

function loginRedirect(next: string): never {
  redirect(`/login?next=${encodeURIComponent(next)}`);
}

function fail<T>(r: AuthedResult<T>, next: string): ShopState | null {
  if (r.ok) return null;
  if (r.status === 401) loginRedirect(next);
  if (r.status === 403) return { error: "Akun Anda tidak punya akses admin." };
  return { error: r.message };
}

function readProduct(form: FormData) {
  return {
    name: String(form.get("name") ?? ""),
    category: String(form.get("category") ?? ""),
    description: String(form.get("description") ?? ""),
    price: Number(form.get("price")),
    stock: Number(form.get("stock")),
  };
}

function checkProduct(p: ReturnType<typeof readProduct>): string | null {
  if (!p.name.trim()) return "Nama wajib diisi.";
  if (!p.category.trim()) return "Kategori wajib diisi.";
  if (!Number.isInteger(p.price) || p.price < 0) return "Harga harus bilangan bulat, 0 atau lebih.";
  if (!Number.isInteger(p.stock) || p.stock < 0) return "Stok harus bilangan bulat, 0 atau lebih.";
  return null;
}

export async function createProductAction(_prev: ShopState, form: FormData): Promise<ShopState> {
  const input = readProduct(form);
  const bad = checkProduct(input);
  if (bad) return { error: bad };

  const r = await authedRequest<Product>("POST", "/products", input);
  const e = fail(r, "/admin/produk/baru");
  if (e) return e;
  revalidatePath("/admin/produk");
  // lanjut ke halaman ubah agar gambar bisa langsung diunggah
  redirect(`/admin/produk/${(r as { data: Product }).data.id}?baru=1`);
}

export async function updateProductAction(_prev: ShopState, form: FormData): Promise<ShopState> {
  const id = Number(form.get("id"));
  if (!Number.isInteger(id) || id <= 0) return { error: "Produk tidak valid." };
  const input = readProduct(form);
  const bad = checkProduct(input);
  if (bad) return { error: bad };

  const r = await authedRequest<Product>("PUT", `/products/${id}`, input);
  const e = fail(r, `/admin/produk/${id}`);
  if (e) return e;
  revalidatePath("/admin/produk");
  revalidatePath(`/products/${id}`);
  return { ok: "Perubahan disimpan." };
}

export async function deleteProductAction(_prev: ShopState, form: FormData): Promise<ShopState> {
  const id = Number(form.get("id"));
  if (!Number.isInteger(id) || id <= 0) return { error: "Produk tidak valid." };
  const r = await authedRequest<void>("DELETE", `/products/${id}`);
  const e = fail(r, `/admin/produk/${id}`);
  if (e) return e; // contoh: 409 "produk sudah pernah dipesan..." tampil apa adanya
  revalidatePath("/admin/produk");
  redirect("/admin/produk?dihapus=1");
}

export async function uploadImageAction(_prev: ShopState, form: FormData): Promise<ShopState> {
  const id = Number(form.get("id"));
  const file = form.get("image");
  if (!Number.isInteger(id) || id <= 0) return { error: "Produk tidak valid." };
  if (!(file instanceof File) || file.size === 0) return { error: "Pilih file gambar terlebih dahulu." };
  if (file.size > MAX_IMAGE) return { error: "Ukuran gambar maksimal 2 MB." };

  // teruskan hanya field "image"; jenis file diperiksa product-service lewat isi file, bukan nama/ekstensi
  const out = new FormData();
  out.set("image", file, file.name);
  const r = await authedRequest<Product>("POST", `/products/${id}/image`, out);
  const e = fail(r, `/admin/produk/${id}`);
  if (e) return e;
  revalidatePath(`/admin/produk/${id}`);
  revalidatePath(`/products/${id}`);
  revalidatePath("/admin/produk");
  return { ok: "Gambar diunggah." };
}

export async function changeOrderStatusAction(_prev: ShopState, form: FormData): Promise<ShopState> {
  const id = Number(form.get("order_id"));
  const status = String(form.get("status") ?? "");
  if (!Number.isInteger(id) || id <= 0 || !status) return { error: "Data tidak valid." };
  const r = await authedRequest<Order>("PATCH", `/orders/${id}/status`, { status });
  const e = fail(r, "/admin/pesanan");
  if (e) return e;
  revalidatePath("/admin/pesanan");
  return {};
}
