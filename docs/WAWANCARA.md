# Catatan persiapan wawancara: TokoHerbal

Ini bahan latihan, bukan naskah untuk dihafal. Panduan Bab 12 meminta jawaban "dengan kata-katamu sendiri",
jadi tulis ulang tiap poin dengan bahasamu, lalu latih sambil direkam. Semua angka dan kejadian di bawah
benar-benar terjadi di proyek ini. Jangan menambah klaim yang belum kamu kerjakan (misalnya uji beban atau produksi nyata).

## 1. Penjelasan 5 menit (kerangka Bab 12.1)

1. **Tujuan (30 detik).** Aplikasi katalog dan pemesanan produk herbal untuk belajar arsitektur backend modern.
   Pelanggan bisa mencari produk, memakai keranjang, memesan, dan membatalkan. Admin mengelola produk, gambar, dan status pesanan.
2. **Arsitektur (1 menit).** Buka `docs/arsitektur.svg`. Dua service Go: `auth-service` (akun, token) dan `product-service`
   (produk, keranjang, pesanan). Masing-masing punya database sendiri (`auth_db`, `shop_db`) dan tidak saling membaca.
   Redis untuk cache dan rate limit. Next.js sebagai frontend. Semua jalan dengan satu perintah `docker compose up`.
3. **Keputusan teknis (2 menit).** Pilih tiga dari bagian 2 di bawah. Saran: transaksi stok, Redis mati, dan cookie httpOnly.
4. **Kendala (1 menit).** Satu cerita dari bagian 3. Saran: Redis mati membuat request menggantung 33 detik.
5. **Keterbatasan dan rencana (30 detik).** Jujur: microservices berlebihan untuk skala ini, dipakai untuk belajar polanya.
   Pembayaran disimulasikan. Belum ada message queue, uji beban, atau idempotency key. Rencana: ulasan produk di MongoDB, API gateway.

## 2. Keputusan teknis, alasan, dan bukti di kode

| Keputusan | Alasan | Trade-off yang harus kamu akui | Tunjukkan di |
|---|---|---|---|
| Stok dikurangi dengan `UPDATE ... SET stock = stock - $1 WHERE id = $2 AND stock >= $1` dalam satu transaksi | Dua pembeli yang berebut stok terakhir tidak bisa sama-sama berhasil; tanpa kunci eksplisit | Item diurutkan menurut `product_id` supaya dua pesanan tidak saling mengunci (deadlock) | `internal/store/orders.go` (UPDATE dalam transaksi), `internal/service/order.go` (`normalizeItems` mengurutkan item), `scripts/race-order.mjs` |
| Cache-aside di Redis, TTL 60 detik, semua key `products:*` dihapus saat ada perubahan | Sederhana dan benar; Redis hanya optimasi, bukan sumber kebenaran | Menghapus semua key itu kasar (SCAN + DEL); skala besar butuh versi key atau tag | `internal/service/product.go`, `internal/cache/redis.go` |
| Access token JWT 15 menit + refresh token opaque 7 hari, disimpan sebagai hash SHA-256, berotasi, ada deteksi pemakaian ulang | Access token bisa diverifikasi cepat; refresh token bisa dicabut | JWT tidak bisa dicabut sebelum kedaluwarsa (role lama berlaku sampai 15 menit) | `auth-service/internal/service`, `internal/token` |
| product-service memanggil `GET /verify` di auth-service, tidak memegang `JWT_SECRET` | Rahasia hanya di satu tempat; auth-service pemilik token | Tiap request login menambah satu panggilan jaringan; kalau auth mati, endpoint login ikut 503 | `internal/auth/client.go` |
| Interface didefinisikan di sisi pemakai (`service/ports.go`) dan fakes untuk test | Logika bisnis diuji tanpa database | Lebih banyak berkas | `internal/service/ports.go`, `internal/fakes` |
| Token di cookie httpOnly lewat server Next.js, bukan `localStorage` | Skrip jahat (XSS) tidak bisa membaca token; tidak ada urusan CORS | Frontend jadi punya lapisan server; CSRF dijaga `SameSite=Lax` | `web/lib/session.ts`, `web/app/actions/auth.ts` |
| Pengecualian rate limit lewat `INTERNAL_TOKEN` | Dari sisi product-service semua pengunjung web tampak satu IP; tanpa ini 60 request/menit habis dibagi semua orang | Rahasia bersama harus dijaga; perbandingan memakai `ConstantTimeCompare` | `internal/handler/middleware.go` |
| Image distroless nonroot + healthcheck lewat flag `-healthcheck` pada biner | Image kecil, tanpa shell, proses bukan root | Sulit di-debug dari dalam container | `auth-service/Dockerfile` |

## 3. Cerita kendala (urutan: gejala, penyebab, solusi, pelajaran)

Bagian A, B, dan E adalah kendala yang benar-benar kamu alami di laptopmu. Bagian C dan D adalah masalah yang **diantisipasi dari desain**
dan diuji dengan simulasi, bukan insiden yang menimpamu. Ceritakan keduanya apa adanya ("saya mengantisipasi", bukan "saya mengalami"):
pewawancara yang bertanya lebih dalam akan membedakannya, dan jawaban jujur justru lebih kuat.

**A. Redis mati membuat aplikasi menggantung (cerita terbaik).**
Saat Redis dihentikan, `curl` ke produk mengembalikan `Empty reply` setelah 33 detik dan `/health` jadi lambat sampai container `unhealthy`.
Penyebab: nama host `redis` gagal di-resolve, tiap operasi menunggu sekitar 11 detik, satu request memanggil Redis tiga kali,
dan totalnya melewati `WriteTimeout` server 30 detik. Kamu menemukannya dengan mematikan Redis sungguhan dan mengukur waktunya
(`curl` menunjukkan 33 detik), bukan dengan menebak. Solusi: circuit breaker dan batas waktu keras 300 ms per operasi (`internal/breaker`).
Hasil: request pertama sekitar 1 detik, berikutnya milidetik, `/health` tetap 200. Pelajaran: "fallback ke database" harus
dibuktikan dengan Redis benar-benar mati, dan timeout default sering tidak cukup.

**B. Password Postgres tidak cocok setelah `.env` diubah.**
Gejala: `auth-service` unhealthy, log `password authentication failed` (28P01). Penyebab: `POSTGRES_PASSWORD` hanya berlaku saat volume
pertama kali dibuat. Solusi: `ALTER USER postgres PASSWORD '...'` di dalam container, tanpa menghapus data. Pelajaran: konfigurasi
yang "hanya dibaca sekali" harus dikenali; `docker compose down -v` menghapus data.

**C. Dua refresh sekaligus bisa mengeluarkan pengguna (diantisipasi dari desain, diuji dengan simulasi).**
Refresh token berotasi dan auth-service menganggap pemakaian ulang sebagai pencurian (semua sesi dicabut). Browser sering mengirim beberapa
request bersamaan, jadi tanpa penjagaan request kedua memakai token lama dan pengguna terlempar. Solusi: refresh satu kali per token
(single-flight) di `web/proxy.ts`, hasilnya disimpan 15 detik. Diuji dengan server auth tiruan yang meniru rotasi dan deteksi pemakaian ulang:
10 request paralel menghasilkan satu panggilan refresh dan nol pemakaian ulang.

**D. Batas body 1 MB pada Server Action (diantisipasi, bukan insiden).**
Server Action membatasi body 1 MB secara bawaan (dokumentasi Next.js), sedangkan batas aplikasi untuk gambar 2 MB. Solusi: `serverActions.bodySizeLimit` di `next.config.ts`;
unggahan PNG 1,5 MB berhasil di pengujian.
Pelajaran: batas di setiap lapisan harus selaras (browser, Next.js, Go, volume).

**E. Variabel `@id` di REST Client menjadi bagian body.**
Baris `@var = ...` di bawah body request ikut terkirim sehingga API menjawab 400 "body JSON tidak valid". Solusi: definisikan variabel di atas berkas.
Cerita singkat yang menunjukkan kamu memeriksa dari ujung ke ujung (alat uji sendiri bisa salah).

## 4. Pertanyaan sulit dan arah jawaban jujur

- **Kenapa microservices untuk aplikasi sekecil ini?** Untuk belajar polanya (database per service, komunikasi antar service, kegagalan sebagian).
  Untuk produk nyata dengan tim kecil, monolith terstruktur lebih masuk akal. Biaya yang saya rasakan: panggilan `/verify` di setiap request, dan harus menangani auth mati.
- **Apa kelemahan proyekmu?** Pembayaran simulasi; invalidasi cache kasar; tidak ada idempotency key (tombol "Pesan" dinonaktifkan saat diproses dan keranjang
  yang sudah kosong menolak pesanan kedua, tetapi itu bukan pengganti idempotency key); belum ada uji beban; logout tidak mencabut access token yang masih berlaku.
- **Bagaimana kalau lalu lintas naik 100 kali?** Ukur dulu. Kandidat: indeks dan query lambat, cache lebih halus, replika baca PostgreSQL,
  beberapa instance product-service di belakang load balancer (rate limit sudah di Redis sehingga bisa dibagi), dan antrean untuk pekerjaan yang tidak perlu sinkron.
- **Kenapa tidak simpan JWT di `localStorage`?** Skrip pihak ketiga atau XSS bisa membacanya. Cookie httpOnly tidak terbaca JavaScript.
  Konsekuensinya: perlu `SameSite` dan lapisan server.
- **Apa risiko `INTERNAL_TOKEN`?** Bila bocor, pihak lain terbebas dari rate limit (bukan dari autentikasi). Mitigasi: minimal 16 karakter, perbandingan constant-time,
  fitur mati bila kosong, jangan di-commit.
- **Bagaimana mencegah SQL injection?** Query berparameter; kolom `ORDER BY` memakai whitelist karena tidak bisa diparameterkan; input `LIKE` di-escape.
- **Apa itu N+1 dan di mana kamu menghindarinya?** Daftar pesanan dan item diambil dengan `order_id = ANY($1)`, bukan satu query per pesanan.
- **Kalau PostgreSQL mati?** `/health` melaporkan tidak sehat (503), container dinilai `unhealthy`, endpoint yang butuh database menjawab galat tanpa membocorkan detail.
  Tidak ada fallback, karena database adalah sumber kebenaran.
- **Kenapa dua database dalam satu container Postgres?** Efisien di laptop; yang penting batasnya logis: tidak ada foreign key lintas database dan tiap service hanya tahu database-nya.
  Di produksi bisa dipisah tanpa mengubah kode selain `DATABASE_URL`.

## 5. Dari pertanyaan Bab 12 ke bukti di proyekmu

| Pertanyaan umum | Tunjukkan |
|---|---|
| Goroutine, channel, race condition | `internal/breaker/breaker.go`: goroutine + `select` + channel berpenyangga; `go test -race` di CI |
| Error di Go | `errors.Is` / `errors.As` di `handler.fail`, `fmt.Errorf("%w")` di klien auth |
| Interface | `internal/service/ports.go` |
| `context` | timeout 4 detik di `handler.authed`, `ctx` diteruskan ke tiap query (`tx.QueryRow(ctx, ...)`), pembatalan di `breaker.Do` |
| ACID | pembuatan pesanan: kurangi stok, buat pesanan, buat item, kosongkan keranjang dalam satu transaksi |
| Index | indeks GIN trigram untuk pencarian nama (`db/init.sql`) |
| N+1, SQL injection | `order_id = ANY($1)`, query berparameter |
| Multi-stage build | `auth-service/Dockerfile`, `web/Dockerfile` |
| Volume | `uploads` (gambar bertahan setelah `docker compose down`) |
| Cache-aside, Redis mati | `internal/cache`, `internal/breaker`, `scripts/redis-down-check.mjs` |
| JWT tidak terenkripsi | token berisi `sub`, `role`, `exp`; `web/lib/session.ts` hanya membaca isinya untuk tampilan |
| CORS | awalnya diperlukan; sekarang dihindari karena browser hanya bicara dengan server Next.js |
| Server vs Client Component | `AddToCart.tsx` (server) membungkus `ActionForm.tsx` (client, `useActionState`) |
| Idempotent | `PUT /cart/items/{id}` menetapkan jumlah mutlak; `POST /orders` tidak idempotent |

## 6. Pertanyaan untuk pewawancara (Bab 12.5)

- Seperti apa arsitektur sistem yang sekarang dipakai, dan bagian mana yang akan saya kerjakan?
- Bagaimana proses code review dan deployment di tim?
- Apa yang diharapkan dari saya dalam 3 bulan pertama?
- Bagaimana kesempatan belajar dan berkembang di tim teknis?

## 7. Cara berlatih

1. Rekam dirimu menjelaskan 5 menit tanpa membaca. Dengar ulang, catat bagian yang tersendat.
2. Buka kode untuk tiap "Tunjukkan di" dan jelaskan lima baris pertama dengan suara keras.
3. Ucapkan satu kalimat jujur untuk tiap batas pengetahuan: "Belum pernah di produksi, tetapi di proyek saya begini dan ini yang saya pelajari."
4. Latih tes teknis kecil: buat endpoint CRUD Go dari nol dalam 30 menit; tulis query JOIN dengan agregasi; jelaskan `docker-compose.yml`.
