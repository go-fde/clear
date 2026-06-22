package clear

import (
	"sync"
	"testing"
)

// memDevice is an in-memory ReaderAt/WriterAt/Closer used to benchmark the
// passthrough path without involving real disk IO.
type memDevice struct {
	mu  sync.Mutex
	buf []byte
}

func (m *memDevice) ReadAt(p []byte, off int64) (int, error)  { return copy(p, m.buf[off:]), nil }
func (m *memDevice) WriteAt(p []byte, off int64) (int, error) { return copy(m.buf[off:], p), nil }
func (m *memDevice) Close() error                             { return nil }

// BenchmarkPassthroughWrite measures the no-encryption upper bound: the clear
// backend simply forwards WriteAt to the underlying device (a memcpy here).
func BenchmarkPassthroughWrite(b *testing.B) {
	const sz = 1 << 20
	dev := &memDevice{buf: make([]byte, sz)}
	d, err := OpenFrom(dev)
	if err != nil {
		b.Fatal(err)
	}
	p := make([]byte, sz)
	b.SetBytes(int64(sz))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = d.WriteAt(p, 0)
	}
}

func BenchmarkPassthroughRead(b *testing.B) {
	const sz = 1 << 20
	dev := &memDevice{buf: make([]byte, sz)}
	d, err := OpenFrom(dev)
	if err != nil {
		b.Fatal(err)
	}
	p := make([]byte, sz)
	b.SetBytes(int64(sz))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = d.ReadAt(p, 0)
	}
}
