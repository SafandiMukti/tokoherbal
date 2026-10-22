// Mengirim banyak permintaan berurutan lalu melaporkan kapan mulai dijawab 429.
// Pemakaian: node scripts/rate-limit-test.mjs [BATAS=60] [JUMLAH=BATAS+10]
// PERHATIAN: setelah skrip ini, IP Anda terblokir sampai 1 menit. Tunggu sebelum menguji manual lagi.

const SHOP = process.env.SHOP_URL ?? "http://localhost:8082";
const limit = Number(process.argv[2] ?? 60);
const total = Number(process.argv[3] ?? limit + 10);

let firstBlocked = null; let ok = 0; let blocked = 0; let retryAfter = null;
for (let i = 1; i <= total; i++) {
  const res = await fetch(`${SHOP}/products`);
  await res.arrayBuffer();
  if (res.status === 429) { blocked++; firstBlocked ??= i; retryAfter ??= res.headers.get("Retry-After"); }
  else if (res.status === 200) ok++;
  else { console.error(`Status tak terduga di permintaan ${i}: ${res.status}`); process.exit(1); }
}
console.log(`Dikirim ${total} permintaan: ${ok} berhasil (200), ${blocked} ditolak (429).`);
if (firstBlocked) console.log(`429 pertama muncul di permintaan ke-${firstBlocked}. Retry-After: ${retryAfter} detik.`);
const pass = blocked > 0 && firstBlocked === limit + 1;
console.log(pass ? "LULUS: tepat setelah batas terlampaui, permintaan ditolak."
  : "TIDAK SESUAI: periksa REDIS_ADDR, RATE_LIMIT_PER_MIN, dan apakah permintaan lain ikut terhitung dalam menit yang sama.");
process.exit(pass ? 0 : 1);
