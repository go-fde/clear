<p align="center"><img src="https://raw.githubusercontent.com/go-fde/brand/main/social/go-fde.png" alt="go-fde/clear" width="720"></p>

# clear

Pure-Go passthrough block device with no encryption.

## Overview

The `clear` package exposes a `Device` that forwards all I/O directly to the
underlying file or block device without any encryption or decryption. It
satisfies the same interface as the `luks` and `apfs` backends so that the
`fde` dispatcher can treat plaintext devices uniformly.

## Usage

### Open a plaintext file as a block device

```go
import "github.com/go-fde/clear"

dev, err := clear.Open("/path/to/disk.raw")
if err != nil {
    log.Fatal(err)
}
defer dev.Close()

buf := make([]byte, 512)
_, err = dev.ReadAt(buf, 0)

_, err = dev.WriteAt(buf, 0)

fmt.Println("device size:", dev.Size())
```

### Layer on top of another block device

`OpenFrom` accepts any value satisfying:

```go
interface {
    io.ReaderAt
    WriteAt([]byte, int64) (int, error)
    io.Closer
}
```

```go
dev, err := clear.OpenFrom(someRW)
if err != nil {
    log.Fatal(err)
}
defer dev.Close()
```

`Size()` returns the file size when the underlying value is an `*os.File`, and
`0` for all other backends.

## Device interface

| Method | Description |
|--------|-------------|
| `ReadAt(p, off)` | Forward read to the underlying device |
| `WriteAt(p, off)` | Forward write to the underlying device |
| `Size() int64` | File size at open time (`0` for non-file backends) |
| `Close()` | Close the underlying device |
