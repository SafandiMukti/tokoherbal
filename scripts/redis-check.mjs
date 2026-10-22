// Membuktikan invalidasi cache: data yang diubah langsung terlihat, tidak menunggu TTL 60 detik.
// Pemakaian: node scripts/redis-check.mjs EMAIL_ADMIN PASSWORD_ADMIN
// Lihat juga log HIT/MISS di: docker compose logs -f product-service

const AUTH = process.env.AUTH_URL ?? "http://localhost:8081";
const SHOP = process.env.SHOP_URL ?? "http://localhost:8082";
const [email, password] = process.argv.slice(2);
if (!email || !password) { console.error("Pemakaian: node scripts/redis-check.mjs EMAIL_ADMIN PASSWORD_ADMIN"); process.exit(2); }

async function call(path, { base = SHOP, method = "GET", token, body } = {}) {
  const t0 = performance.now();
  const res = await fetch(base + path, {
    method,
    headers: { ...(body ? { "Content-Type": "application/json" } : {}), ...(token ? { Authorization: `Bearer ${token}` } : {}) },
    body: body ? JSON.stringify(body) : undefined,
  });
  const text = await res.text();
  let json = null; try { json = text ? JSON.parse(text) : null; } catch {}
  return { status: res.status, json, ms: Math.round((performance.now() - t0) * 10) / 10 };
}

const login = await call("/login", { base: AUTH, method: "POST", body: { email, password } });
if (login.status !== 200) throw new Error(`login gagal: ${login.status}`);
const token = login.json.access_token;

const tag = `CacheUji${Date.now()}`;
const created = await call("/products", { method: "POST", token, body: { name: tag, category: "uji", price: 1000, stock: 5 } });
if (created.status !== 201) throw new Error(`buat produk gagal: ${created.status} ${JSON.stringify(created.json)}`);
const id = created.json.id;
const q = `/products?q=${tag}`;

const a = await call(q);   // MISS: diisi dari database, lalu disimpan ke cache
const b = await call(q);   // HIT: dari Redis
console.log(`GET 1 (MISS): ${a.ms} ms, nama = ${a.json.items[0]?.name}`);
console.log(`GET 2 (HIT) : ${b.ms} ms, nama = ${b.json.items[0]?.name}`);

const upd = await call(`/products/${id}`, { method: "PUT", token, body: { name: `${tag}Ubah`, category: "uji", price: 1000, stock: 5 } });
if (upd.status !== 200) throw new Error(`ubah produk gagal: ${upd.status}`);
const c = await call(q);   // jika cache TIDAK dihapus, nama lama masih tampil sampai TTL habis
console.log(`GET 3 (setelah PUT): nama = ${c.json.items[0]?.name}`);

await call(`/products/${id}`, { method: "DELETE", token });
const ok = a.json.items[0]?.name === tag && b.json.items[0]?.name === tag && c.json.items[0]?.name === `${tag}Ubah`;
console.log(ok ? "LULUS: data baru langsung terlihat setelah diubah (cache dihapus saat data berubah)."
               : "GAGAL: data lama masih tampil, kemungkinan cache tidak diinvalidasi.");
process.exit(ok ? 0 : 1);
