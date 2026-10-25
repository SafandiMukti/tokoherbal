// Panggilan ke product-service dari SERVER Next.js (komponen server).
// Server di dalam Docker memakai nama service (http://product-service:8080),
// sedangkan browser memakai localhost. Karena itu ada dua alamat berbeda.

export type Product = {
  id: number;
  name: string;
  description: string;
  category: string;
  price: number;
  stock: number;
  image_url: string;
  created_at: string;
};

export type ProductList = {
  items: Product[];
  total: number;
  page: number;
  limit: number;
};

export type ListQuery = {
  q?: string;
  category?: string;
  sort?: string;
  page?: number;
};

const SERVER_BASE = process.env.PRODUCT_API_URL ?? "http://localhost:8082";
// Alamat gambar yang dibuka BROWSER. Sengaja bukan NEXT_PUBLIC_*: variabel itu ditanam saat build,
// sedangkan ini dibaca saat berjalan sehingga satu image Docker bisa dipakai di lingkungan mana pun.
const PUBLIC_BASE = process.env.PUBLIC_PRODUCT_URL ?? "http://localhost:8082";

// Panggilan server-ke-server dikecualikan dari rate limit per-IP di product-service
// (dari sudut pandangnya semua pengunjung web terlihat sebagai satu IP: container web ini).
export function internalHeaders(): Record<string, string> {
  const t = process.env.INTERNAL_TOKEN;
  return t ? { "X-Internal-Token": t } : {};
}

export class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
  ) {
    super(message);
  }
}

async function get<T>(path: string): Promise<T> {
  let res: Response;
  try {
    // no-store: cache sudah ditangani Redis di product-service, jangan dobel di sini
    res = await fetch(SERVER_BASE + path, {
      headers: internalHeaders(),
      cache: "no-store",
      signal: AbortSignal.timeout(5000),
    });
  } catch {
    throw new ApiError(503, "Layanan produk tidak dapat dihubungi");
  }
  if (!res.ok) {
    let msg = `Permintaan gagal (${res.status})`;
    try {
      const body = (await res.json()) as { error?: string };
      if (body.error) msg = body.error;
    } catch {
      /* body bukan JSON, pakai pesan default */
    }
    throw new ApiError(res.status, msg);
  }
  return (await res.json()) as T;
}

export function listProducts(q: ListQuery): Promise<ProductList> {
  const p = new URLSearchParams();
  if (q.q) p.set("q", q.q);
  if (q.category) p.set("category", q.category);
  if (q.sort) p.set("sort", q.sort);
  if (q.page && q.page > 1) p.set("page", String(q.page));
  const qs = p.toString();
  return get<ProductList>("/products" + (qs ? "?" + qs : ""));
}

export function getProduct(id: number): Promise<Product> {
  return get<Product>(`/products/${id}`);
}

export function listCategories(): Promise<string[]> {
  return get<string[]>("/categories");
}

// image_url dari API berupa path ("/uploads/abc.jpg"); browser butuh alamat lengkap
export function imageSrc(imageUrl: string): string {
  return imageUrl ? PUBLIC_BASE + imageUrl : "";
}
