# Demo 5 menit, tangkapan layar, dan CV

## 1. Skenario demo (sekitar 5 menit)

Siapkan sebelum mulai: `docker compose up -d`, semua service `healthy`, data contoh terisi, dua akun (satu admin, satu pelanggan), dua jendela browser (atau satu biasa + satu penyamaran).

| Menit | Lakukan | Katakan |
|---|---|---|
| 0:00 | Tunjukkan diagram di README | "Dua service Go dengan database masing-masing, Redis, Next.js. Semua jalan dengan satu perintah." |
| 0:30 | `docker compose ps`: lima service `healthy` | "Tiap service punya healthcheck; urutan start diatur `depends_on`." |
| 1:00 | Katalog: cari "madu", ganti urutan, buka halaman 2 | "Pencarian dan paginasi di server; hasil di-cache di Redis 60 detik." |
| 1:45 | Masuk sebagai pelanggan, tambah produk, ubah jumlah di keranjang, **Pesan sekarang** | "Pesanan membuat satu transaksi: stok dikurangi dengan `WHERE stock >= qty`, sehingga tidak bisa oversell." |
| 2:45 | Di jendela admin: `/admin/pesanan`, tandai dibayar lalu dikirim | "Aturan alur status ada di satu tempat di backend; tombol di UI hanya mengikuti." |
| 3:15 | Admin: tambah produk, unggah gambar, tampil di katalog | "Jenis file diperiksa dari isinya, bukan ekstensinya; gambar di volume Docker, jadi tidak hilang." |
| 3:45 | `docker compose stop redis`, muat ulang katalog (tetap tampil), lalu `docker compose start redis` | "Redis hanyalah optimasi. Circuit breaker membuat request tetap cepat saat Redis mati." |
| 4:30 | Tunjukkan GitHub Actions hijau | "Test dengan race detector dan build image berjalan di setiap push." |
| 4:50 | Sebut keterbatasan | "Microservices berlebihan untuk skala ini, pembayaran simulasi, belum ada uji beban." |

Tips: latih dua kali dengan stopwatch. Bila ada yang gagal saat demo, jelaskan apa yang kamu perkirakan terjadi dan cek lognya (`docker compose logs product-service`): itu justru menunjukkan cara kerjamu.

## 2. Tangkapan layar untuk README

Ambil di http://localhost:3000 dengan data sungguhan. Simpan di `docs/screenshots/` dengan nama berikut:

1. `01-katalog.png`: halaman utama dengan beberapa produk dan gambar
2. `02-keranjang.png`: keranjang berisi dua produk dengan total
3. `03-pesanan.png`: detail pesanan berstatus "Menunggu pembayaran"
4. `04-admin-produk.png`: tabel kelola produk
5. `05-admin-pesanan.png`: pesanan dengan tombol status
6. `06-docker-ps.png`: terminal `docker compose ps` dengan lima service `healthy`
7. `07-ci-hijau.png`: tab Actions di GitHub dengan centang hijau

Setelah berkasnya ada, di `README.md` hapus tanda komentar (`<!--` dan `-->`) pada bagian "Tangkapan layar".
Jangan menampilkan email atau token pribadi di gambar.

## 3. Bullet CV (bagian Proyek)

Perbarui hanya setelah kamu benar-benar menjalankan dan memahami tiap bagian. Angka di bawah dihitung dari repo; hitung ulang bila berubah.

- Membangun aplikasi katalog dan pemesanan produk herbal dengan arsitektur microservices: dua service Go (autentikasi; produk, keranjang, dan pesanan) dengan database PostgreSQL terpisah, Redis, dan frontend Next.js, dijalankan dengan satu perintah `docker compose up`.
- Menerapkan autentikasi JWT dengan refresh token berotasi dan deteksi pemakaian ulang, password bcrypt, dan penyimpanan token di cookie httpOnly; menangani refresh bersamaan agar pengguna tidak terlempar.
- Menjamin stok tidak terjual berlebih lewat transaksi PostgreSQL dan pengurangan atomik (`WHERE stock >= qty`), dibuktikan dengan skrip dua pembeli yang berebut stok terakhir.
- Menambahkan cache Redis (cache-aside), rate limiting, serta circuit breaker sehingga aplikasi tetap melayani request ketika Redis atau layanan autentikasi mati; kegagalan disimulasikan dan diukur.
- Menulis 67 fungsi test Go (handler, service, dan fake store) dan pipeline GitHub Actions: `gofmt`, `go vet`, `go test -race`, dan build image Docker.

**Keahlian:** Go, PostgreSQL, Redis, Docker dan Docker Compose, Next.js/React (TypeScript), REST API, JWT, GitHub Actions.
Tulis "microservices" sebagai pengalaman proyek, bukan pengalaman produksi.

## 4. Surat lamaran

Ganti kalimat "belum pernah memakai Docker, Redis, maupun Next.js" dengan paragraf seperti ini (sesuaikan dengan kata-katamu):

> Untuk mempersiapkan diri, saya membangun TokoHerbal, aplikasi katalog dan pemesanan dengan dua service Go, PostgreSQL, Redis, dan Next.js yang dijalankan dengan Docker Compose.
> Dalam proyek ini saya menangani masalah nyata seperti stok yang diperebutkan dua pembeli, aplikasi yang menggantung saat Redis mati, dan pembaruan token yang bersamaan.
> Kodenya, README, dan pipeline CI-nya dapat dilihat di [tautan repo].

## 5. GitHub

- **Deskripsi repo:** "Aplikasi katalog dan pemesanan produk herbal: microservices Go, PostgreSQL, Redis, Next.js, Docker Compose."
- **Topics:** `go`, `nextjs`, `postgresql`, `redis`, `docker`, `microservices`, `jwt`, `github-actions`
- **Disematkan (pin)** di profil, dan pastikan riwayat commit rapi dan tidak ada secret (`.env` tidak ada di Git; periksa dengan `git log --all -p -- .env`).
