# Draf entri untuk CATATAN.md

Salin yang relevan ke `CATATAN.md`, isi tanggalnya, dan ubah dengan kata-katamu. Entri tentang Docker daemon sudah ada di CATATAN.md.

## (tanggal): auth-service unhealthy, "password authentication failed"

- Gejala: `docker compose ps` menunjukkan `auth-service` unhealthy; log berisi `password authentication failed for user "postgres"` (kode 28P01).
- Penyebab: `POSTGRES_PASSWORD` dari `.env` hanya dipakai saat volume `pgdata` pertama kali dibuat. Mengubah `.env` sesudahnya tidak mengubah password di database.
- Solusi: `docker compose exec postgres psql -U postgres -c "ALTER USER postgres PASSWORD '...';"` (data tetap aman). Alternatif: `docker compose down -v`, yang menghapus semua data.
- Dipelajari: ada konfigurasi yang hanya dibaca sekali; kenali mana yang begitu. `down -v` menghapus data.

## (tanggal): REST Client menjawab 400 "body JSON tidak valid"

- Gejala: login dan POST lain di berkas `.http` selalu 400.
- Penyebab: baris `@variabel = nilai` yang ditulis di bawah body request ikut menjadi bagian body.
- Solusi: pindahkan definisi variabel ke bagian atas berkas, atau setelah baris `###`.
- Dipelajari: periksa alat ujinya sendiri sebelum menyalahkan API.

## (tanggal): Redis dimatikan, aplikasi menggantung 33 detik ("Empty reply from server")

- Gejala: `curl http://localhost:8082/products` mengembalikan `Empty reply from server` setelah 33 detik; `/health` lambat; container `unhealthy`.
- Penyebab: nama host `redis` gagal di-resolve dan tiap operasi menunggu sekitar 11 detik; satu request memanggil Redis tiga kali; total melewati `WriteTimeout` 30 detik.
- Solusi: circuit breaker dan batas waktu keras 300 ms per operasi Redis. Setelah itu request pertama sekitar 1 detik dan berikutnya milidetik.
- Dipelajari: "fallback ke database" harus dibuktikan dengan benar-benar mematikan Redis dan mengukur waktunya; timeout bawaan sering tidak cukup.

## (tanggal): `docker compose` menolak jalan karena INTERNAL_TOKEN kosong

- Gejala: `required variable INTERNAL_TOKEN is missing a value`.
- Penyebab: compose sengaja mewajibkan variabel itu (`${INTERNAL_TOKEN:?...}`) agar tidak berjalan tanpa rahasia bersama.
- Solusi: isi `INTERNAL_TOKEN` (minimal 16 karakter) di `.env`.
- Dipelajari: `:?` pada compose membuat konfigurasi yang kurang gagal cepat dengan pesan jelas, bukan gagal diam-diam di tengah jalan.
