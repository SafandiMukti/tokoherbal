export type CartItem = {
  product_id: number;
  name: string;
  price: number;
  qty: number;
  stock: number;
  image_url: string;
  subtotal: number;
};

export type Cart = { items: CartItem[]; total: number };

export type OrderItem = { product_id: number; name: string; qty: number; price: number };

export type Order = {
  id: number;
  user_id: number;
  total: number;
  status: string;
  created_at: string;
  items: OrderItem[];
};

// Hasil aksi form: pesan galat atau pesan sukses singkat.
export type ShopState = { error?: string; ok?: string };
