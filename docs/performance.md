# Performance and resource behavior

This document describes implemented behavior. The project does not publish comparative download or command benchmarks in v1.3.4 because no reproducible benchmark dataset is maintained.

## Downloads

Downloads are sequential and streamed to a unique partial file in the cache directory. The compatibility fields `download.parallel` and `download.max_connections` do not change runtime behavior in v1.3.4.

When a partial archive is available, govman requests the remaining range. It accepts resume only when `Content-Range`, declared length, expected total size, and actual byte count agree. A server that returns a full `200` response causes a restart from byte zero. Transient `408`, `429`, and `5xx` responses are retried within configured limits.

Completed archives remain cached and are reused only when their metadata and SHA-256 checksum validate. `govman clean` removes this cache without removing installed Go versions.

## Release metadata cache

Go release JSON is cached in memory by endpoint and cache policy. Concurrent callers share one fetch, and callers receive cloned data rather than a mutable shared slice. The default lifetime is ten minutes and can be changed with `go_releases.cache_expiry`.

## Extraction

Tar.gz and zip archives are streamed into a sibling staging directory. Extraction limits are:

- 2 GiB per regular file;
- 10 GiB total extracted regular-file data;
- 100,000 archive entries.

The staging tree is validated and atomically renamed into place. Failed extraction removes staging and never exposes a partial final installation.

## Progress output

Interactive downloads render progress to stderr. Updates are throttled, repeated percentages are suppressed, and a resumed transfer calculates speed from bytes transferred in the current process. Quiet mode and non-terminal stderr disable progress rendering.

## Practical tuning

- Increase `download.timeout` for a slow or high-latency connection.
- Adjust `download.retry_count` and `download.retry_delay` for transient server errors.
- Increase `go_releases.cache_expiry` to reduce metadata requests.
- Run `govman clean` to reclaim archive cache space.
- Run `govman prune` to remove toolchains that are not active, default, or project-local.
- Put `cache_dir` on a filesystem with sufficient free space for both a partial and completed archive.

The `mirror.*` compatibility fields do not reroute downloads. A controlled endpoint must be configured explicitly with `go_releases.api_url` and `go_releases.download_url`.

## Development measurement

Use Go's standard tooling when investigating a measured regression:

```bash
go test -bench=. -benchmem ./...
go test -cpuprofile=cpu.prof -memprofile=mem.prof ./...
go tool pprof cpu.prof
```

Do not publish timing or memory claims without recording the commit, Go version, platform, hardware, network source, command, sample count, and raw results.
