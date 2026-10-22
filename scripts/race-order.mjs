// Uji "dua pembeli rebutan stok terakhir" lewat API sungguhan.
// Pemakaian (dari folder root proyek, setelah docker compose up):
//   node scripts/race-order.mjs EMAIL_ADMIN PASSWORD_ADMIN
// Hasil yang benar: tepat satu pesanan 201 dan satu 409, stok akhir 0.
// Catatan: produk uji tetap ada di database (sudah punya riwayat pesanan, tidak bisa dihapus).

const AUTH = process.env.AUTH_URL ?? "http://localhost:8081";
const SHOP = process.env.SHOP_URL ?? "http://localhost:8082";
const [adminEmail, adminPassword] = process.argv.slice(2);
if (!adminEmail || !adminPassword) {
  console.error("Pemakaian: node scripts/race-order.mjs EMAIL_ADMIN PASSWORD_ADMIN");
  process.exit(2);
}

async function call(base, path, { method = "GET", token, body } = {}) {
  const res = await fetch(base + path, {
    method,
    headers: {
      ...(body ? { "Content-Type": "application/json" } : {}),
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    body: body ? JSON.stringify(body) : undefined,
  });
  const text = await res.text();
  let json = null;
  try { json = text ? JSON.parse(text) : null; } catch { /* bukan JSON */ }
  return { status: res.status, json };
}

async function login(email, password) {
  const r = await call(AUTH, "/login", { method: "POST", body: { email, password } });
  if (r.status !== 200) throw new Error(`login ${email} gagal: ${r.status} ${JSON.stringify(r.json)}`);
  return r.json.access_token;
}

async function newUser(tag) {
  const email = `race-${Date.now()}-${tag}@example.com`;
  const reg = await call(AUTH, "/register", { method: "POST", body: { email, password: "rahasia123" } });
  if (reg.status !== 201) throw new Error(`register gagal: ${reg.status} ${JSON.stringify(reg.json)}`);
  return login(email, "rahasia123");
}

const admin = await login(adminEmail, adminPassword);
const created = await call(SHOP, "/products", {
  method: "POST",
  token: admin,
  body: { name: `Uji Rebutan ${Date.now()}`, description: "stok 1 untuk uji konkurensi", category: "uji", price: 1000, stock: 1 },
});
if (created.status !== 201) throw new Error(`buat produk gagal: ${created.status} ${JSON.stringify(created.json)}`);
const productId = created.json.id;
console.log(`Produk uji id=${productId}, stok awal=1`);

const [t1, t2] = await Promise.all([newUser("a"), newUser("b")]);

// kedua permintaan dikirim SERENTAK
const order = (token) => call(SHOP, "/orders", { method: "POST", token, body: { items: [{ product_id: productId, qty: 1 }] } });
const results = await Promise.all([order(t1), order(t2)]);
results.forEach((r, i) => console.log(`Pembeli ${i + 1}: HTTP ${r.status}`, r.json?.error ?? `pesanan #${r.json?.id}`));

const final = await call(SHOP, `/products/${productId}`);
const codes = results.map((r) => r.status).sort();
const ok = codes[0] === 201 && codes[1] === 409 && final.json.stock === 0;
console.log(`Stok akhir: ${final.json.stock}`);
console.log(ok ? "LULUS: satu pembeli berhasil, satu ditolak, stok tidak minus." : "GAGAL: hasil tidak sesuai harapan.");
process.exit(ok ? 0 : 1);
