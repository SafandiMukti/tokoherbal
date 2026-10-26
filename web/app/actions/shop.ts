"use server";

import { revalidatePath } from "next/cache";
import { redirect } from "next/navigation";
import { authedRequest, getSession, type AuthedResult } from "@/lib/authed";
import type { Cart, Order, ShopState } from "@/lib/types";

const MAX_QTY = 99; // sama dengan batas di product-service

function loginRedirect(next: string): never {
  redirect(`/login?next=${encodeURIComponent(next)}`);
}

// 401 berarti sesi habis: arahkan ke login. Galat lain dikembalikan sebagai pesan.
function fail<T>(r: AuthedResult<T>, next: string): ShopState | null {
  if (r.ok) return null;
  if (r.status === 401) loginRedirect(next);
  return { error: r.message };
}

const ADMIN_MSG = "Akun admin tidak dapat berbelanja. Gunakan akun pelanggan.";

// Admin mengelola toko, bukan berbelanja. API juga menolak (403); ini agar pesannya jelas
// dan tidak ada permintaan yang sia-sia.
async function adminBlocked(): Promise<boolean> {
  return (await getSession())?.role === "admin";
}

function intField(form: FormData, name: string): number {
  const n = Number(form.get(name));
  return Number.isInteger(n) ? n : NaN;
}

// "Tambah ke keranjang" menambah ke jumlah yang sudah ada. API hanya punya PUT (jumlah mutlak),
// jadi baca keranjang dulu lalu tulis jumlah barunya.
export async function addToCartAction(_prev: ShopState, form: FormData): Promise<ShopState> {
  if (await adminBlocked()) return { error: ADMIN_MSG };
  const productId = intField(form, "product_id");
  const qty = intField(form, "qty");
  if (!(productId > 0) || !(qty >= 1)) return { error: "Jumlah tidak valid." };

  const cart = await authedRequest<Cart>("GET", "/cart");
  const e1 = fail(cart, `/products/${productId}`);
  if (e1) return e1;

  const existing = cart.ok ? (cart.data.items.find((i) => i.product_id === productId)?.qty ?? 0) : 0;
  const next = Math.min(existing + qty, MAX_QTY);
  const put = await authedRequest<Cart>("PUT", `/cart/items/${productId}`, { qty: next });
  const e2 = fail(put, `/products/${productId}`);
  if (e2) return e2;

  revalidatePath("/keranjang");
  return { ok: `Ditambahkan. Jumlah di keranjang: ${next}.` };
}

export async function setCartQtyAction(_prev: ShopState, form: FormData): Promise<ShopState> {
  if (await adminBlocked()) return { error: ADMIN_MSG };
  const productId = intField(form, "product_id");
  const qty = intField(form, "qty");
  if (!(productId > 0) || !(qty >= 1 && qty <= MAX_QTY)) return { error: `Jumlah harus 1 sampai ${MAX_QTY}.` };

  const r = await authedRequest<Cart>("PUT", `/cart/items/${productId}`, { qty });
  const e = fail(r, "/keranjang");
  if (e) return e;
  revalidatePath("/keranjang");
  return {};
}

export async function removeCartItemAction(_prev: ShopState, form: FormData): Promise<ShopState> {
  if (await adminBlocked()) return { error: ADMIN_MSG };
  const productId = intField(form, "product_id");
  if (!(productId > 0)) return { error: "Produk tidak valid." };
  const r = await authedRequest<Cart>("DELETE", `/cart/items/${productId}`);
  const e = fail(r, "/keranjang");
  if (e) return e;
  revalidatePath("/keranjang");
  return {};
}

export async function checkoutAction(): Promise<ShopState> {
  if (await adminBlocked()) return { error: ADMIN_MSG };
  // body kosong = pesan seluruh isi keranjang (keranjang dikosongkan oleh server)
  const r = await authedRequest<Order>("POST", "/orders", {});
  const e = fail(r, "/keranjang");
  if (e) return e;
  revalidatePath("/keranjang");
  redirect(`/pesanan/${(r as { data: Order }).data.id}?baru=1`);
}

export async function cancelOrderAction(_prev: ShopState, form: FormData): Promise<ShopState> {
  if (await adminBlocked()) return { error: ADMIN_MSG };
  const id = intField(form, "order_id");
  if (!(id > 0)) return { error: "Pesanan tidak valid." };
  const r = await authedRequest<Order>("PATCH", `/orders/${id}/status`, { status: "cancelled" });
  const e = fail(r, `/pesanan/${id}`);
  if (e) return e;
  revalidatePath(`/pesanan/${id}`);
  revalidatePath("/pesanan");
  // tanpa pesan sukses: halaman dirender ulang dan form ini hilang; statusnya sendiri yang menjadi umpan balik
  return {};
}
