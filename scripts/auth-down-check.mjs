// Cek ketahanan saat auth-service MATI.
// Pakai:  node scripts/auth-down-check.mjs EMAIL_ADMIN PASSWORD   (auth-service hidup, ambil token)
// Lalu:   docker compose stop auth-service
//         node scripts/auth-down-check.mjs --token TOKEN_YANG_TADI
// Harapan: GET /products tetap 200; endpoint login -> 503 + Retry-After;
//          request pertama < ~4 dtk, berikutnya cepat (breaker); /health tetap cepat.
const BASE = process.env.PRODUCT_URL ?? "http://localhost:8082";
const AUTH = process.env.AUTH_URL ?? "http://localhost:8081";
const a = process.argv.slice(2);

async function timed(label, path, headers = {}) {
  const t = Date.now();
  let status = "ERR", retry = "-";
  try {
    const r = await fetch(BASE + path, { headers, signal: AbortSignal.timeout(10000) });
    status = r.status; retry = r.headers.get("retry-after") ?? "-";
  } catch (e) { status = "ERR " + e.name; }
  const ms = Date.now() - t;
  console.log(`${label.padEnd(26)} status=${status} retry-after=${retry} ${ms} ms`);
  return { status, ms };
}

if (a[0] !== "--token") {
  const [email, password] = a;
  if (!email || !password) { console.log("Pakai: node scripts/auth-down-check.mjs EMAIL PASSWORD"); process.exit(1); }
  const r = await fetch(AUTH + "/login", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ email, password }) });
  const j = await r.json();
  if (!j.access_token) { console.log("Login gagal:", JSON.stringify(j)); process.exit(1); }
  console.log("Token (berlaku 15 menit). Sekarang jalankan:\n  docker compose stop auth-service\n  node scripts/auth-down-check.mjs --token " + j.access_token);
  process.exit(0);
}

const h = { Authorization: "Bearer " + a[1] };
let ok = true;
const pub = await timed("GET /products (publik)", "/products");
const health = await timed("GET /health", "/health");
const r1 = await timed("GET /cart #1 (login)", "/cart", h);
const r2 = await timed("GET /cart #2 (login)", "/cart", h);
const r3 = await timed("GET /cart #3 (login)", "/cart", h);
const r4 = await timed("GET /cart #4 (breaker)", "/cart", h);
if (pub.status !== 200) { ok = false; console.log("GAGAL: halaman publik harus tetap 200"); }
if (health.status !== 200 || health.ms > 1500) { ok = false; console.log("GAGAL: /health harus 200 dan cepat"); }
for (const r of [r1, r2, r3, r4]) if (r.status !== 503) { ok = false; console.log("GAGAL: endpoint login harus 503"); break; }
if (r4.ms > 500) { ok = false; console.log("GAGAL: setelah breaker terbuka harus langsung (<500 ms)"); }
console.log(ok ? "\nLULUS" : "\nADA YANG GAGAL");
process.exit(ok ? 0 : 1);
