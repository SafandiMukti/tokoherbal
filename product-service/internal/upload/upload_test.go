package upload

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// tanda tangan PNG yang cukup untuk http.DetectContentType
var pngBytes = append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte{0}, 32)...)

func TestSave(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	name, err := s.Save(bytes.NewReader(pngBytes))
	if err != nil {
		t.Fatal(err)
	}
	if !ValidName(name) || !strings.HasSuffix(name, ".png") {
		t.Fatalf("nama = %q", name)
	}
	if _, err := os.Stat(filepath.Join(s.Dir(), name)); err != nil {
		t.Fatalf("file tidak tersimpan: %v", err)
	}

	s.Remove(name)
	if _, err := os.Stat(filepath.Join(s.Dir(), name)); !os.IsNotExist(err) {
		t.Fatal("file harus terhapus")
	}
}

func TestSaveRejects(t *testing.T) {
	s, _ := New(t.TempDir())

	if _, err := s.Save(strings.NewReader("<script>alert(1)</script>")); !errors.Is(err, ErrBadType) {
		t.Fatalf("HTML harus ditolak sebagai bukan gambar, err = %v", err)
	}
	// file .exe yang "menyamar" tetap ditolak karena yang dibaca isinya
	if _, err := s.Save(strings.NewReader("MZ\x90\x00 program windows")); !errors.Is(err, ErrBadType) {
		t.Fatalf("err = %v", err)
	}

	big := append(append([]byte{}, pngBytes...), bytes.Repeat([]byte{1}, MaxBytes)...)
	if _, err := s.Save(bytes.NewReader(big)); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("file kebesaran harus ditolak, err = %v", err)
	}

	entries, _ := os.ReadDir(s.Dir())
	if len(entries) != 0 {
		t.Fatalf("tidak boleh ada file tersisa: %v", entries)
	}
}

func TestValidNameBlocksTraversal(t *testing.T) {
	for _, bad := range []string{"../../etc/passwd", "..", "a.png", "", "../" + strings.Repeat("a", 32) + ".png", strings.Repeat("A", 32) + ".png", strings.Repeat("a", 32) + ".exe"} {
		if ValidName(bad) {
			t.Errorf("%q seharusnya tidak valid", bad)
		}
	}
	if !ValidName(strings.Repeat("a1", 16) + ".webp") {
		t.Error("nama hasil Save harus valid")
	}
	s, _ := New(t.TempDir())
	if _, ok := s.Path("../x"); ok {
		t.Error("Path harus menolak traversal")
	}
}
