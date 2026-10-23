# TokoHerbal

Aplikasi katalog dan pemesanan produk herbal/kosmetik dengan arsitektur microservices.
Proyek belajar dan portofolio: Go, PostgreSQL, Redis, Docker, dan Next.js.

> README ini akan dilengkapi di tahap akhir (diagram, tangkapan layar, daftar endpoint lengkap).

## Status tahap

- [x] Tahap 0: fondasi (Docker Compose, skema database, data contoh)
- [x] Tahap 1: auth-service (register, login, refresh token, logout, verify)
- [x] Tahap 2: product-service (CRUD, pencarian, paginasi, upload gambar)
- [x] Tahap 3: keranjang, pesanan, transaksi stok
- [x] Tahap 4: Redis (cache, rate limit)
- [ ] Tahap 5: komunikasi antar service, 503, slog
- [ ] Tahap 6: test dan CI
- [ ] Tahap 7-8: frontend Next.js dan Docker penuh
- [ ] Tahap 9: ulasan produk di MongoDB (opsional)

## Menjalankan (Windows, Command Prompt)

Prasyarat: Docker Desktop sudah menyala (tunggu sampai statusnya "Engine running").

```bat
copy .env.example .env
```

Edit `.env`: isi `DB_PASSWORD` (huruf dan angka saja) dan `JWT_SECRET`. Buat secret acak dengan:

```bat
node -e "console.log(require('crypto').randomBytes(48).toString('hex'))"
```

Unduh dependensi Go dan jalankan test (sekali saja, ini juga membuat `go.sum` yang dibutuhkan Dockerfile):

```bat
cd auth-service
go mod tidy
go vet ./...
go test ./...
cd ..
```

> `go test -race` di Windows butuh compiler C (cgo) yang biasanya belum terpasang. Pakai `go test ./...` di laptop; `-race` dijalankan di CI (Linux) pada tahap 6.

Lalu:

```bat
docker compose up -d --build
docker compose ps
curl http://localhost:8081/health
```

Isi data contoh produk (12 produk, aman dijalankan berulang):

```bat
docker compose exec postgres psql -U postgres -d shop_db -f /seed/seed.sql
```

Mengulang database dari nol (data hilang): `docker compose down -v`

## Port di laptop

| Layanan | Alamat |
|---|---|
| auth-service | http://localhost:8081 |
| PostgreSQL | localhost:5433 (user `postgres`) |
| product-service | http://localhost:8082 |
| Redis | localhost:6379 |

## Menjadikan akun sebagai admin

Register selalu membuat role `customer`. Ubah lewat SQL:

```bat
docker compose exec postgres psql -U postgres -d auth_db -c "UPDATE users SET role='admin' WHERE email='andi@example.com';"
```

Token lama tetap berisi role lama sampai kedaluwarsa (15 menit). Login ulang untuk mendapat role baru.

## Endpoint auth-service

| Method | Path | Akses | Fungsi |
|---|---|---|---|
| GET | /health | publik | Cek service dan koneksi database |
| POST | /register | publik | Buat akun (role selalu customer) |
| POST | /login | publik | Kembalikan access token dan refresh token |
| POST | /refresh | publik | Tukar refresh token dengan pasangan token baru |
| POST | /logout | publik | Cabut refresh token |
| GET | /verify | Bearer token | Kembalikan user_id dan role |

Contoh request siap pakai ada di `auth-service/requests.http` (ekstensi REST Client di VS Code).

## Endpoint product-service

| Method | Path | Akses | Fungsi |
|---|---|---|---|
| GET | /health | publik | Cek service dan database |
| GET | /products | publik | Daftar produk. Query: `q`, `category`, `sort` (newest, price_asc, price_desc, name), `page`, `limit` |
| GET | /products/{id} | publik | Detail produk |
| GET | /categories | publik | Daftar kategori |
| GET | /uploads/{nama} | publik | Gambar produk |
| POST | /products | admin | Tambah produk |
| PUT | /products/{id} | admin | Ubah produk |
| DELETE | /products/{id} | admin | Hapus produk |
| POST | /products/{id}/image | admin | Unggah gambar (field `image`, JPEG/PNG/WebP, maks 2 MB) |

Pemeriksaan token dilakukan dengan memanggil `GET /verify` di auth-service (timeout 3 detik).
Bila auth-service mati, endpoint admin membalas 503, bukan menggantung.

Cache Redis dan rate limit sudah ada di kode tetapi mati selama `REDIS_ADDR` kosong. Dinyalakan di Tahap 4.

Contoh request siap pakai: `product-service/requests.http`.

## Endpoint keranjang dan pesanan (product-service)

| Method | Path | Akses | Fungsi |
|---|---|---|---|
| GET | /cart | login | Isi keranjang beserta subtotal dan total |
| PUT | /cart/items/{product_id} | login | Isi atau ganti jumlah (body `{"qty": 2}`) |
| DELETE | /cart/items/{product_id} | login | Hapus satu produk dari keranjang |
| POST | /orders | login | Buat pesanan. Body `{}` = pesan isi keranjang, atau `{"items":[{"product_id":1,"qty":2}]}` |
| GET | /orders | login | Riwayat pesanan sendiri |
| GET | /orders/{id} | login | Detail pesanan (milik orang lain dijawab 404) |
| PATCH | /orders/{id}/status | login | Ubah status. Pelanggan hanya boleh `cancelled` pada pesanannya sendiri |
| GET | /admin/orders | admin | Semua pesanan, filter `?status=` |

Alur status: `pending` -> `paid` -> `shipped` -> `completed`. Batal boleh saat `pending` atau `paid`
dan mengembalikan stok. Pembayaran disimulasikan (admin yang menandai `paid`).

### Keamanan stok

Membuat pesanan berjalan dalam SATU transaksi database: kurangi stok, buat pesanan, buat item, kosongkan keranjang.
Stok dikurangi dengan `UPDATE ... WHERE stock >= qty`, jadi dua pembeli yang berebut stok terakhir
tidak bisa sama-sama berhasil. Buktikan sendiri (setelah `docker compose up`):

```bat
node scripts/race-order.mjs EMAIL_ADMIN PASSWORD_ADMIN
```

Contoh request siap pakai: `product-service/orders.http`.

## Redis: cache dan rate limit (Tahap 4)

- **Cache-aside** untuk `GET /products`, `GET /products/{id}`, dan `GET /categories`. Key berawalan `products:`, TTL 60 detik.
  Log `cache HIT` / `cache MISS` muncul di `docker compose logs -f product-service`.
- **Invalidasi**: tambah, ubah, hapus produk, unggah gambar, membuat pesanan, dan membatalkan pesanan menghapus semua key `products:*`.
- **Rate limit**: maksimal `RATE_LIMIT_PER_MIN` (bawaan 60) permintaan per IP per menit, di luar `/health`. Lebih dari itu dijawab `429` dengan header `Retry-After`.
- **Redis mati tidak mematikan aplikasi**: request tetap dilayani dari database (cache dilewati) dan rate limit dilewati.
  `/health` menandai `"redis": "down"` tetapi tetap 200, karena Redis hanyalah optimasi.
- **Circuit breaker + batas waktu keras (300 ms) untuk setiap operasi Redis** (`internal/breaker`). Tanpa ini, Redis yang mati
  membuat tiap perintah menggantung belasan detik (nama host `redis` gagal di-resolve), satu request memanggil Redis 2-3 kali,
  dan totalnya melewati `WriteTimeout` server (klien menerima "Empty reply"; `/health` jadi lambat dan container `unhealthy`).
  Sekarang: tiga kegagalan berturut-turut membuka breaker, panggilan ke Redis ditolak instan selama 10 detik, lalu dicoba lagi satu per satu.
  Ukuran: request pertama saat Redis mati sekitar 1 detik, berikutnya milidetik, `/health` tetap 200.
- Redis dibatasi `maxmemory 128mb` dengan kebijakan `allkeys-lru` (key paling lama tak terpakai dibuang saat penuh).

Pembuktian otomatis:

```bat
node scripts/redis-check.mjs EMAIL_ADMIN PASSWORD_ADMIN
node scripts/rate-limit-test.mjs
```

Uji Redis mati secara otomatis:

```bat
docker compose stop redis
node scripts/redis-down-check.mjs
docker compose start redis
node scripts/redis-down-check.mjs --recover
```
