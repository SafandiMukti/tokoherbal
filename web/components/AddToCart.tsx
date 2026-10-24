import Link from "next/link";
import { addToCartAction } from "@/app/actions/shop";
import { ActionForm } from "./ActionForm";

export function AddToCart({
  productId,
  stock,
  loggedIn,
}: {
  productId: number;
  stock: number;
  loggedIn: boolean;
}) {
  if (stock <= 0) return null;
  if (!loggedIn) {
    return (
      <p>
        <Link href={`/login?next=/products/${productId}`}>Masuk</Link> untuk membeli.
      </p>
    );
  }
  return (
    <ActionForm action={addToCartAction} className="add-to-cart">
      <input type="hidden" name="product_id" value={productId} />
      <label>
        Jumlah
        <input type="number" name="qty" defaultValue={1} min={1} max={Math.min(stock, 99)} required />
      </label>
      <button type="submit">Tambah ke keranjang</button>
    </ActionForm>
  );
}
