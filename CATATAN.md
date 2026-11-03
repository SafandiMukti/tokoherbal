# CATATAN

Catat setiap kendala dan cara memecahkannya. Saat wawancara, cerita dari catatan ini sangat berharga.
Format: tanggal, gejala (pesan error), penyebab, solusi, apa yang dipelajari.

A. Masalah yang saya alami di laptop

A1. 2026-10-18: Docker daemon tidak terhubung

Gejala: docker version menampilkan Client, lalu error failed to connect to the docker API at npipe:////./pipe/dockerDesktopLinuxEngine.
Penyebab: Docker Desktop belum menyala, jadi tidak ada daemon yang melayani perintah.
Solusi: buka Docker Desktop, tunggu statusnya "Engine running", cek ulang dengan docker info.
Dipelajari: docker hanyalah klien. Yang menjalankan container adalah daemon. Docker Desktop yang ditutup juga tidak menghapus data: volume pgdata dan uploads tetap ada, cukup nyalakan lagi dan jalankan docker compose up -d.

A2. 2026-10-20: auth-service unhealthy, "password authentication failed"

Gejala: sehari setelah git add (belum di-commit), saat proyek dibuka lagi, docker compose ps menunjukkan auth-service unhealthy. Log (docker compose logs auth-service --tail 30) berisi password authentication failed for user "postgres" (kode 28P01).
Penyebab: POSTGRES_PASSWORD dari .env hanya dipakai saat volume pgdata pertama kali dibuat. Mengubah .env sesudahnya tidak mengubah password di database, sehingga password di .env dan di database berbeda.
Solusi: docker compose exec postgres psql -U postgres -c "ALTER USER postgres PASSWORD '...';" (data tetap aman). Alternatif: docker compose down -v, yang menghapus semua data.
Dipelajari: ada konfigurasi yang hanya dibaca sekali; kenali mana yang begitu (init.sql juga begitu). down -v menghapus data, jadi jangan dipakai sembarangan. Membaca log dulu menghemat waktu dibanding menebak.

A3. 2026-10-21: REST Client menjawab 400 "body JSON tidak valid"

- Gejala: login (admin maupun pelanggan) dan POST lain di berkas `.http` selalu 400.
- Penyebab: baris `@variabel = nilai` yang ditulis di bawah body request ikut menjadi bagian body, sehingga JSON-nya rusak.
- Solusi: pindahkan definisi variabel ke bagian atas berkas, atau tempatkan setelah baris `###`.
- Dipelajari: periksa alat ujinya sendiri sebelum menyalahkan API. Pesan error API-nya benar; yang salah adalah isi yang terkirim.

A4. 2026-10-23: Redis dimatikan, aplikasi menggantung 33 detik ("Empty reply from server")

- Gejala: setelah `docker compose stop redis`, `curl http://localhost:8082/products` mengembalikan `curl: (52) Empty reply from server`. Diukur: status 000 setelah 33,27 detik. `/health` lambat dan container `product-service` berstatus `unhealthy`. Log penuh `redis: connection pool: failed to dial ... lookup redis: i/o timeout`.
- Penyebab: nama host `redis` gagal di-resolve dan tiap operasi Redis menunggu sekitar 11 detik. Satu request memanggil Redis tiga kali (cache, rate limit, set cache), jadi totalnya sekitar 33 detik, melewati `WriteTimeout` server 30 detik. Server menutup koneksi sebelum sempat menjawab.
- Solusi: circuit breaker (`internal/breaker`) dan batas waktu keras 300 ms per operasi Redis. Saat Redis mati, request pertama sekitar 1 detik, berikutnya milidetik, dan data tetap diambil dari database. Cek ulang dengan `node scripts/redis-down-check.mjs`.
- Dipelajari: "fallback ke database" harus dibuktikan dengan benar-benar mematikan Redis dan mengukur waktunya. Timeout bawaan pustaka sering terlalu panjang. Prediksi awal saya (1 sampai 3 detik) salah, dan itu ketahuan karena saya mengujinya, bukan menebak.

A5. 2026-10-23: `git push` gagal, "src refspec main does not match any"

- Gejala: `git push -u origin main` menampilkan `error: src refspec main does not match any` dan `failed to push some refs`.
- Penyebab: di lokal belum ada branch bernama `main`. Salah satunya: branch lokal bernama `master`, atau belum ada commit sama sekali (sudah `git add` tetapi belum `git commit`).
- Solusi: cek `git status`, `git branch`, `git log --oneline`. Buat commit pertama bila belum ada, lalu `git branch -M main` dan `git push -u origin main`. Sebelum `git add .`, pastikan `git status` tidak menampilkan `.env`.
- Dipelajari: `git add` hanya menyiapkan, yang menyimpan adalah `git commit`. Nama branch lokal dan remote harus cocok. `.env` berisi rahasia dan tidak boleh ikut ter-push.

A6. 2026-10-23: token JWT yang panjang terpotong saat ditempel di CMD

- Gejala: saat menjalankan `node scripts/auth-down-check.mjs --token <token>`, CMD menampilkan potongan token sebagai "perintah" terpisah (`.mjs --token ...`, lalu sisa token sebagai perintah lain). Perintah yang benar baru jalan setelah diulang.
- Penyebab: token JWT panjang tertempel dengan pemisah baris di tengahnya, sehingga CMD menganggapnya beberapa perintah.
- Solusi: tempel token dalam satu baris tanpa menekan Enter di tengah, lalu ulangi perintah. Hasil yang benar: `GET /cart` menjawab 503 dengan `retry-after=5`, dan setelah tiga kegagalan breaker terbuka (request keempat 7 ms). Skrip menyatakan LULUS.
- Dipelajari: JWT adalah satu string tanpa spasi atau baris baru. Token yang ikut tertempel di percakapan perlu diwaspadai; access token hanya berlaku 15 menit sehingga risikonya kecil, tetapi biasakan tidak membagikan token atau password.

A7. 2026-10-27: panel "Signal Check" di portofolio tidak tampil di laptop, tampil di HP

- Gejala: di laptop (Chrome, layar lebar) panel Signal Check di hero tidak terlihat; di ponsel terlihat.
- Penyebab: panel diberi `opacity: 0` sebagai kondisi awal animasi, tetapi lupa didaftarkan ke animasi pembuka di `script.js`, jadi tidak pernah dimunculkan. Di ponsel kemungkinan pengaturan "kurangi animasi" aktif, sehingga semua elemen ditampilkan langsung tanpa animasi. Tes saya sebelumnya tidak menangkap ini karena GSAP tidak termuat di lingkungan tes.
- Solusi: tambahkan panel ke timeline GSAP, dan beri ruang bawah di hero supaya tulisan SCROLL tidak menempel ke tombol. Diuji ulang dengan GSAP sungguhan (opacity berubah dari 0 ke 1).
- Dipelajari: perilaku bisa beda antara perangkat yang animasinya aktif dan yang dikurangi. Uji dengan pustaka yang benar-benar dimuat, bukan hanya jalur cadangannya.

A8. Bisa terjadi: `docker compose` menolak jalan karena INTERNAL_TOKEN kosong

- Gejala: `required variable INTERNAL_TOKEN is missing a value`.
- Penyebab: compose sengaja mewajibkan variabel itu (`${INTERNAL_TOKEN:?...}`) agar tidak berjalan tanpa rahasia bersama antara frontend dan product-service.
- Solusi: isi `INTERNAL_TOKEN` (minimal 16 karakter) di `.env`.
- Dipelajari: `:?` pada compose membuat konfigurasi yang kurang gagal cepat dengan pesan jelas, bukan gagal diam-diam di tengah jalan.

---

B. Masalah yang ditemukan saat pengembangan (di luar laptop, tapi jadi pelajaran)

B1. Refresh token terpakai dua kali dianggap pencurian

- Gejala (yang akan terjadi): halaman dan prefetch-nya mengirim beberapa request paralel dengan refresh token yang sama. Request kedua dianggap pemakaian ulang token (pencurian) dan seluruh sesi dicabut, sehingga pengguna terlempar keluar.
- Penyebab: refresh token berotasi, hanya sah dipakai satu kali; request paralel berebut memakainya.
- Solusi: di `proxy.ts`, satu refresh per token (single-flight). Hasilnya dibagi ke request lain dan disimpan 15 detik untuk yang terlambat. Diuji: 10 request paralel menghasilkan 1 panggilan refresh dan 0 pemakaian ulang.
- Dipelajari: keamanan dan kemudahan bisa bertabrakan; mekanisme anti-pencurian harus memperhitungkan request yang berebut.

B2. Rate limit per IP dan frontend yang berbagi IP

- Masalah: semua panggilan dari server Next.js berasal dari satu IP container, sehingga dihitung sebagai satu klien dan batas 60 request per menit cepat terpakai. Gambar `/uploads/*` juga ikut dihitung.
- Solusi: header `X-Internal-Token` dari frontend melewati rate limit. Dibandingkan dengan `subtle.ConstantTimeCompare`; kosong berarti fitur mati; minimal 16 karakter. Diuji: token benar dilewati, tanpa atau salah token tetap dibatasi.
- Dipelajari: batas per IP salah untuk lalu lintas antar-service; bedakan klien luar dan klien tepercaya.

B3. Stok terakhir diperebutkan dua pembeli

- Masalah: dua pesanan bersamaan untuk stok 1 bisa sama-sama lolos kalau stok dibaca lalu ditulis terpisah.
- Solusi: `UPDATE products SET stock = stock - $1 WHERE id = $2 AND stock >= $1` di dalam transaksi; item diurutkan menurut `product_id` agar tidak deadlock; `SELECT ... FOR UPDATE` saat ganti status; pembatalan mengembalikan stok dalam transaksi yang sama. Dibuktikan dengan `scripts/race-order.mjs` (satu 201, satu 409, stok 0).
- Dipelajari: konsistensi data dijaga oleh database, bukan oleh kode aplikasi yang berharap tidak ada yang berebut.

B4. Beberapa kekurangan kecil yang ditemukan lewat pengujian

- `/akun` mengarahkan ke login saat auth-service mati, padahal sesi masih ada. Perbaikan: pemeriksaan `hasPendingRefresh()` dan pesan "layanan login sedang tidak tersedia".
- Pesan sukses "pesanan dibatalkan" hilang karena formulirnya ikut hilang. Perbaikan: pesan permanen di halaman.
- Parameter `next` pada login hanya menerima path di situs sendiri, supaya `?next=//evil.com` tidak dipakai.

B5. Admin tidak boleh berbelanja

- Aturan: akun admin mengelola toko, bukan pelanggan. Admin tidak punya keranjang dan pesanan pribadi.
- API: `GET/PUT/DELETE /cart...`, `POST /orders`, dan `GET /orders` menjawab 403 untuk admin; diuji dengan `TestAdminCannotShop` (dan terbukti gagal bila pengecekan dimatikan). Admin tetap boleh melihat semua pesanan dan mengubah statusnya.
- UI: menu admin hanya Kelola produk dan Pesanan masuk; di halaman produk tombol beli diganti "Ubah produk"; `/keranjang`, `/pesanan`, dan `/pesanan/{id}` dialihkan ke `/admin/pesanan`; aksi belanja di server Next.js juga menolak admin.
- Dipelajari: aturan bisnis harus ditegakkan di API; UI hanya membantu pengguna. Menyembunyikan tombol saja tidak cukup.

B6. Kesalahan pada proses pemandu

- README lama (91 baris) menimpa README Tahap 4 (172 baris) di zip tahap 5 dan 6. Sudah dipulihkan dan dilengkapi di zip tahap 8. Pelajaran: bandingkan berkas yang akan ditimpa sebelum mengirim paket.
- Petunjuk sempat memakai path `D:\work\tokoherbal`, padahal repo ada di `D:\work\tokoherbal\tokoherbal`. Periksa path dengan `cd` dan `dir` sebelum menjalankan perintah.

## user 
- topandim@gmail.com pass: Topandim2603 (admin)
- plenger@gmail.com pass: plenger123
