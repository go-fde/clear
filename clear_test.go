package clear

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
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

// errRW is a RW whose every operation fails with a distinct sentinel, so a
// test can prove each method forwards the underlying error unchanged and
// that Device adds nothing between caller and backend.
type errRW struct{}

var (
	errRead  = errors.New("read failed")
	errWrite = errors.New("write failed")
	errClose = errors.New("close failed")
)

func (errRW) ReadAt([]byte, int64) (int, error)  { return 3, errRead }
func (errRW) WriteAt([]byte, int64) (int, error) { return 5, errWrite }
func (errRW) Close() error                       { return errClose }

func TestErrorsAndCountsAreForwarded(t *testing.T) {
	dev, err := OpenFrom(errRW{})
	if err != nil {
		t.Fatal(err)
	}
	if n, err := dev.ReadAt(make([]byte, 8), 7); n != 3 || !errors.Is(err, errRead) {
		t.Fatalf("ReadAt = %d, %v; want 3, errRead", n, err)
	}
	if n, err := dev.WriteAt(make([]byte, 8), 7); n != 5 || !errors.Is(err, errWrite) {
		t.Fatalf("WriteAt = %d, %v; want 5, errWrite", n, err)
	}
	if err := dev.Close(); !errors.Is(err, errClose) {
		t.Fatalf("Close = %v; want errClose", err)
	}
}

// offsets records the offset each call received, to prove ReadAt/WriteAt
// pass off through untouched.
type offRW struct{ r, w int64 }

func (o *offRW) ReadAt(p []byte, off int64) (int, error)  { o.r = off; return len(p), nil }
func (o *offRW) WriteAt(p []byte, off int64) (int, error) { o.w = off; return len(p), nil }
func (o *offRW) Close() error                             { return nil }

func TestOffsetsPassThrough(t *testing.T) {
	rw := &offRW{}
	dev, _ := OpenFrom(rw)
	if _, err := dev.ReadAt(make([]byte, 1), 1<<40); err != nil || rw.r != 1<<40 {
		t.Fatalf("read offset %d, err %v", rw.r, err)
	}
	if _, err := dev.WriteAt(make([]byte, 1), 4097); err != nil || rw.w != 4097 {
		t.Fatalf("write offset %d, err %v", rw.w, err)
	}
}

// A closed *os.File makes Stat fail: Size must then be 0, not a stale or
// garbage value, and OpenFrom must still succeed (it documents that
// Size is advisory).
func TestOpenFrom_ClosedFile_SizeZero(t *testing.T) {
	path := filepath.Join(t.TempDir(), "disk.raw")
	if err := os.WriteFile(path, make([]byte, 512), 0o600); err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	dev, err := OpenFrom(f)
	if err != nil {
		t.Fatalf("OpenFrom: %v", err)
	}
	if dev.Size() != 0 {
		t.Fatalf("Size = %d; want 0 for a file whose Stat fails", dev.Size())
	}
}

// Open on a directory fails (O_RDWR on a directory is EISDIR) and the
// error names the path and wraps the cause.
func TestOpen_Directory(t *testing.T) {
	dir := t.TempDir()
	_, err := Open(dir)
	if err == nil {
		t.Fatal("expected error opening a directory")
	}
	if !strings.Contains(err.Error(), dir) {
		t.Fatalf("error %q does not name the path", err)
	}
	var pe *fs.PathError
	if !errors.As(err, &pe) {
		t.Fatalf("error %v does not wrap *fs.PathError", err)
	}
}

// Writes extend nothing: Size is the length at open time and stays so.
func TestSizeIsFrozenAtOpen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "disk.raw")
	if err := os.WriteFile(path, make([]byte, 16), 0o600); err != nil {
		t.Fatal(err)
	}
	dev, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer dev.Close()
	if _, err := dev.WriteAt(make([]byte, 16), 1024); err != nil {
		t.Fatal(err)
	}
	if dev.Size() != 16 {
		t.Fatalf("Size = %d; want 16 (recorded at open)", dev.Size())
	}
}
