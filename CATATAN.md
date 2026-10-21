# CATATAN

Catat setiap kendala dan cara memecahkannya. Saat wawancara, cerita dari catatan ini sangat berharga.
Format: tanggal, gejala (pesan error), penyebab, solusi, apa yang dipelajari.

## 2026-10-18: Docker daemon tidak terhubung

- Gejala: `docker version` menampilkan Client, lalu error `failed to connect to the docker API at npipe:////./pipe/dockerDesktopLinuxEngine`.
- Penyebab: Docker Desktop belum menyala, jadi tidak ada daemon yang melayani perintah.
- Solusi: buka Docker Desktop, tunggu statusnya "Engine running", cek ulang dengan `docker info`.
- Dipelajari: `docker` hanyalah klien. Yang menjalankan container adalah daemon.

## (tambahkan catatan berikutnya di bawah ini)
