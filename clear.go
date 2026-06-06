// Package clear provides a passthrough Device implementation for unencrypted
// block devices. It satisfies the same interface as the LUKS and APFS backends,
// allowing the fde dispatcher to treat plaintext devices uniformly.
package clear

import (
	"fmt"
	"io"
	"os"
)

// RW is the minimal read-write-close interface accepted by OpenFrom.
type RW interface {
	io.ReaderAt
	WriteAt(p []byte, off int64) (int, error)
	io.Closer
}

// Device is a passthrough block device that applies no encryption or
// decryption. ReadAt and WriteAt forward directly to the underlying device.
type Device struct {
	f    RW
	size int64
}

// Open opens the file at path as a passthrough block device.
// The file must already exist; it is opened for reading and writing.
func Open(path string) (*Device, error) {
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("clear: open %s: %w", path, err)
	}
	// f.Stat() can in theory fail after a successful OpenFile but
	// in practice never does on any modern OS — fall back to size=0
	// if it ever happens.
	var size int64
	if info, err := f.Stat(); err == nil {
		size = info.Size()
	}
	return &Device{f: f, size: size}, nil
}

// OpenFrom wraps rw as a passthrough Device.
// If rw is an *os.File its current size is recorded for Size(); otherwise
// Size returns 0.
func OpenFrom(rw RW) (*Device, error) {
	return &Device{f: rw, size: sizeOf(rw)}, nil
}

// sizeOf returns the size of rw when it is an *os.File, otherwise 0.
// A Stat failure on a live os.File is essentially impossible on any
// modern OS; we silently treat it as size 0 rather than propagating.
func sizeOf(rw RW) int64 {
	if f, ok := rw.(*os.File); ok {
		info, _ := f.Stat()
		if info != nil {
			return info.Size()
		}
	}
	return 0
}

// ReadAt reads from the underlying device at byte offset off.
func (d *Device) ReadAt(p []byte, off int64) (int, error) { return d.f.ReadAt(p, off) }

// WriteAt writes to the underlying device at byte offset off.
func (d *Device) WriteAt(p []byte, off int64) (int, error) { return d.f.WriteAt(p, off) }

// Size returns the byte length of the device as recorded at open time.
// Returns 0 when the size is not known (e.g. OpenFrom with a non-file backend).
func (d *Device) Size() int64 { return d.size }

// Close releases all resources held by the device.
func (d *Device) Close() error { return d.f.Close() }
