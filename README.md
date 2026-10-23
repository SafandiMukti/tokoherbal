# TokoHerbal

Aplikasi katalog dan pemesanan produk herbal/kosmetik dengan arsitektur microservices.
Proyek belajar dan portofolio: Go, PostgreSQL, Redis, Docker, dan Next.js.

> README ini akan dilengkapi di tahap akhir (diagram, tangkapan layar, daftar endpoint lengkap).

## Status tahap

- [x] Tahap 0: fondasi (Docker Compose, skema database, data contoh)
- [x] Tahap 1: auth-service (register, login, refresh token, logout, verify)
- [ ] Tahap 2: product-service (CRUD, pencarian, paginasi)
- [ ] Tahap 3: keranjang, pesanan, transaksi stok
- [ ] Tahap 4: Redis (cache, rate limit)
- [x] Tahap 5: komunikasi antar service, 503, slog
- [x] Tahap 6: test dan CI
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

Isi data contoh produk (setelah Tahap 2 ada endpoint-nya):

```bat
docker compose exec postgres psql -U postgres -d shop_db -f /seed/seed.sql
```

Mengulang database dari nol (data hilang): `docker compose down -v`

## Port di laptop

| Layanan | Alamat |
|---|---|
| auth-service | http://localhost:8081 |
| PostgreSQL | localhost:5433 (user `postgres`) |
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
