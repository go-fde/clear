# Performance — go-fde/clear (passthrough)  (2026-06-22)

`go-fde/clear` applies **no encryption**; `ReadAt`/`WriteAt` forward straight to
the underlying device. There is no cipher to compare against — the only relevant
question is whether the passthrough adds measurable overhead. It does not: it is
a thin forwarding shim and runs at memory-copy speed.

## Methodology

Apple M4 Max, macOS 26.5, Go 1.26.4 `darwin/arm64`. In-memory backing device
(`memDevice`), 1 MiB transfers, `b.SetBytes`. See `bench_test.go`.

## Results (single core, MB/s — higher is better)

| op | ours | verdict |
|---|---:|---|
| WriteAt (passthrough) | 80 118 MB/s | ✅ memcpy-bound, no overhead |
| ReadAt (passthrough)  | 79 561 MB/s | ✅ memcpy-bound, no overhead |

## Summary

The passthrough is bound only by `copy()`/memory bandwidth and contributes no
overhead beyond a function call and a bounds check; it serves as the
no-encryption upper bound for the encrypted backends (`luks`, `apfs`), whose
bulk AES-XTS throughput is ~585–600 MB/s (see their `BENCHMARKS.md`).
