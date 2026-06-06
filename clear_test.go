package clear

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestOpen_NotExist(t *testing.T) {
	_, err := Open(filepath.Join(t.TempDir(), "nofile"))
	if err == nil {
		t.Fatal("expected error for nonexistent path")
	}
}

func TestOpen_ReadSize(t *testing.T) {
	path := filepath.Join(t.TempDir(), "disk.raw")
	data := make([]byte, 512)
	copy(data, []byte("clear passthrough data"))
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}

	dev, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer dev.Close()

	if dev.Size() != 512 {
		t.Fatalf("Size: want 512, got %d", dev.Size())
	}
	got := make([]byte, 512)
	if _, err := dev.ReadAt(got, 0); err != nil {
		t.Fatalf("ReadAt: %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Fatal("ReadAt mismatch")
	}
}

func TestOpen_WriteRead(t *testing.T) {
	path := filepath.Join(t.TempDir(), "disk.raw")
	if err := os.WriteFile(path, make([]byte, 512), 0o600); err != nil {
		t.Fatal(err)
	}

	dev, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	want := make([]byte, 512)
	copy(want, []byte("write roundtrip test"))
	if _, err := dev.WriteAt(want, 0); err != nil {
		t.Fatalf("WriteAt: %v", err)
	}
	if err := dev.Close(); err != nil {
		t.Fatal(err)
	}

	dev2, err := Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer dev2.Close()
	got := make([]byte, 512)
	if _, err := dev2.ReadAt(got, 0); err != nil {
		t.Fatalf("ReadAt: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("roundtrip mismatch")
	}
}

func TestOpenFrom_File_Size(t *testing.T) {
	path := filepath.Join(t.TempDir(), "disk.raw")
	data := make([]byte, 512)
	copy(data, []byte("openFrom test"))
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}

	dev, err := OpenFrom(f)
	if err != nil {
		f.Close()
		t.Fatalf("OpenFrom: %v", err)
	}
	defer dev.Close()

	if dev.Size() != 512 {
		t.Fatalf("Size: want 512, got %d", dev.Size())
	}
	got := make([]byte, 512)
	if _, err := dev.ReadAt(got, 0); err != nil {
		t.Fatalf("ReadAt: %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Fatal("ReadAt mismatch")
	}
}

func TestOpenFrom_NonFile_ZeroSize(t *testing.T) {
	rw := &memRW{data: make([]byte, 512)}
	dev, err := OpenFrom(rw)
	if err != nil {
		t.Fatalf("OpenFrom: %v", err)
	}
	defer dev.Close()
	if dev.Size() != 0 {
		t.Fatalf("Size: want 0 for non-file RW, got %d", dev.Size())
	}
}

func TestOpenFrom_Write(t *testing.T) {
	rw := &memRW{data: make([]byte, 512)}
	dev, err := OpenFrom(rw)
	if err != nil {
		t.Fatalf("OpenFrom: %v", err)
	}
	defer dev.Close()

	want := make([]byte, 16)
	copy(want, []byte("write via OpenFrom"))
	if _, err := dev.WriteAt(want, 0); err != nil {
		t.Fatalf("WriteAt: %v", err)
	}
	got := make([]byte, 16)
	if _, err := dev.ReadAt(got, 0); err != nil {
		t.Fatalf("ReadAt: %v", err)
	}
	if !bytes.Equal(got, want[:16]) {
		t.Fatal("write/read mismatch")
	}
}

// memRW is an in-memory RW used to test OpenFrom with non-file backends.
type memRW struct{ data []byte }

func (r *memRW) ReadAt(p []byte, off int64) (int, error) {
	if off >= int64(len(r.data)) {
		return 0, io.EOF
	}
	n := copy(p, r.data[off:])
	return n, nil
}

func (r *memRW) WriteAt(p []byte, off int64) (int, error) {
	end := int(off) + len(p)
	for len(r.data) < end {
		r.data = append(r.data, 0)
	}
	copy(r.data[off:], p)
	return len(p), nil
}

func (r *memRW) Close() error { return nil }
