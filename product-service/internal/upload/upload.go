// Package upload menyimpan gambar produk di disk (volume Docker).
//
// Aturan keamanan unggahan file:
//  1. Batasi ukuran.
//  2. Tentukan jenis file dari ISI-nya (http.DetectContentType), bukan dari nama
//     file atau header Content-Type yang bisa dipalsukan klien.
//  3. Nama file dibuat server (acak). Nama dari klien tidak pernah dipakai,
//     sehingga tidak ada path traversal seperti "../../etc/passwd".
package upload

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
)

const MaxBytes = 2 << 20 // 2 MB

var (
	ErrTooLarge = errors.New("ukuran gambar maksimal 2 MB")
	ErrBadType  = errors.New("gambar harus berformat JPEG, PNG, atau WebP")
)

var extByType = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

// validName cocok dengan nama yang dibuat Save: 32 hex + ekstensi.
var validName = regexp.MustCompile(`^[a-f0-9]{32}\.(jpg|png|webp)$`)

// ValidName memeriksa apakah name adalah file hasil Save (dipakai saat menyajikan file).
func ValidName(name string) bool { return validName.MatchString(name) }

type Store struct{ dir string }

// New membuat folder penyimpanan bila belum ada.
func New(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("buat folder upload: %w", err)
	}
	return &Store{dir: dir}, nil
}

func (s *Store) Dir() string { return s.dir }

// Save membaca gambar dari r dan mengembalikan nama file yang tersimpan.
func (s *Store) Save(r io.Reader) (string, error) {
	// baca satu byte lebih banyak dari batas untuk mendeteksi file kebesaran
	data, err := io.ReadAll(io.LimitReader(r, MaxBytes+1))
	if err != nil {
		return "", err
	}
	if len(data) > MaxBytes {
		return "", ErrTooLarge
	}

	// DetectContentType hanya melihat 512 byte pertama
	ext, ok := extByType[detect(data)]
	if !ok {
		return "", ErrBadType
	}

	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	name := hex.EncodeToString(b) + ext

	// O_EXCL: gagal bila file sudah ada, tidak pernah menimpa
	f, err := os.OpenFile(filepath.Join(s.dir, name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(f, bytes.NewReader(data)); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", err
	}
	if err := f.Close(); err != nil {
		os.Remove(f.Name())
		return "", err
	}
	return name, nil
}

// Remove menghapus file. Hanya nama hasil Save yang diterima; selain itu diabaikan.
func (s *Store) Remove(name string) {
	if !ValidName(name) {
		return
	}
	_ = os.Remove(filepath.Join(s.dir, name))
}

// Path mengembalikan lokasi file di disk, atau false bila nama tidak valid.
func (s *Store) Path(name string) (string, bool) {
	if !ValidName(name) {
		return "", false
	}
	return filepath.Join(s.dir, name), true
}
