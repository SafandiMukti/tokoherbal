// Menguji perilaku saat Redis mati.
//   1) docker compose stop redis
//   2) node scripts/redis-down-check.mjs            (aplikasi harus tetap cepat dan 200)
//   3) docker compose start redis
//   4) node scripts/redis-down-check.mjs --recover  (menunggu Redis terdeteksi hidup lagi)

const SHOP = process.env.SHOP_URL ?? "http://localhost:8082";
const recover = process.argv.includes("--recover");

async function timed(path) {
  const t0 = performance.now();
  try {
    const res = await fetch(SHOP + path, { signal: AbortSignal.timeout(40000) });
    const text = await res.text();
    return { status: res.status, ms: Math.round(performance.now() - t0), text };
  } catch (e) {
    return { status: 0, ms: Math.round(performance.now() - t0), text: String(e.cause ?? e.message) };
  }
}

if (recover) {
  const start = Date.now();
  for (;;) {
    const h = await timed("/health");
    let redis = "?";
    try { redis = JSON.parse(h.text).redis; } catch {}
    const detik = Math.round((Date.now() - start) / 1000);
    console.log(`+${detik}s  /health: HTTP ${h.status}, redis=${redis}`);
    if (redis === "ok") { console.log("LULUS: Redis terdeteksi hidup lagi tanpa restart product-service."); process.exit(0); }
    if (Date.now() - start > 40000) { console.log("GAGAL: Redis belum terdeteksi setelah 40 detik. Pastikan container redis sudah berjalan."); process.exit(1); }
    await new Promise((r) => setTimeout(r, 2000));
  }
}

const results = [];
for (let i = 1; i <= 5; i++) {
  const r = await timed("/products");
  results.push(r);
  console.log(`GET /products #${i}: HTTP ${r.status} dalam ${r.ms} ms`);
}
const h = await timed("/health");
console.log(`GET /health     : HTTP ${h.status} dalam ${h.ms} ms  ${h.text.trim()}`);

const allOk = results.every((r) => r.status === 200);
const firstOk = results[0].ms < 3000;
const restFast = results.slice(1).every((r) => r.ms < 500);
const healthOk = h.status === 200 && h.ms < 1000;
console.log(`- semua 200            : ${allOk ? "ya" : "TIDAK"}`);
console.log(`- request pertama <3 s : ${firstOk ? "ya" : "TIDAK"} (sebelum perbaikan: ~33 detik)`);
console.log(`- request berikut <0,5s: ${restFast ? "ya" : "TIDAK"}`);
console.log(`- /health 200 dan cepat: ${healthOk ? "ya" : "TIDAK"}`);
const pass = allOk && firstOk && restFast && healthOk;
console.log(pass ? "LULUS: Redis mati tidak lagi membuat aplikasi lambat atau gagal." : "GAGAL: lihat poin yang TIDAK di atas.");
process.exit(pass ? 0 : 1);
