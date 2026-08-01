# Internal troubleshooting

This guide is for maintainers diagnosing implementation and CI failures.

## Start with the release gates

```bash
make validate
make test-race
make test-coverage
make release-snapshot
```

Do not run `go mod tidy` or formatting as a repair step inside CI; `validate` is intentionally read-only.

## Configuration failures

- Unknown YAML key: strict decoding rejected a typo or obsolete field.
- Permission error: config must be a regular non-symlink file and is restricted to `0600`.
- Overlap error: `install_dir` and `cache_dir` cannot contain one another.
- URL/template error: endpoints must be absolute HTTP(S) without credentials; the Go download template needs one `%s`.

Reproduce with a temp HOME and explicit `--config`. Avoid resetting global Viper state; each config owns its decoder.

## Version/path failures

Version operations accept aliases only before filesystem boundaries. A managed path requires a strict concrete Go version and a successful containment check relative to `install_dir`.

When investigating deletion or activation, log the requested version, resolved concrete version, install root, relative candidate, and canonical executable. Never weaken containment to accept an unexpected path.

## Download failures

Use a local HTTP test server to reproduce exact response status and headers. Check:

- metadata filename equals the URL path basename;
- expected size is positive;
- `206 Content-Range` start/end/total and body length agree;
- `200` after a range request resets the partial;
- `408`, `429`, and `5xx` close bodies before retry;
- final bytes equal metadata size;
- checksum mismatch removes the committed corrupt cache and permits one fresh retry.

Downloads are sequential. The parallel compatibility fields are intentionally inactive.

## Extraction failures

Adversarial fixtures should cover tar.gz and zip traversal, backslashes, absolute/volume/UNC paths, links, oversize entries, excessive totals/counts, corrupt streams, and missing `bin/go`.

Extraction occurs through `os.Root` into a unique sibling staging directory. A failure must leave no final directory. Do not reintroduce link extraction without a documented corpus requirement and containment proof.

## Activation failures

Default activation spans toolchain links, config persistence, and PATH output. Local activation spans project-file persistence and PATH output. Tests must assert rollback state, not only an error string.

Check that all regular executables in the selected Go `bin` directory are linked and stale govman-owned links are removed without replacing unrelated files.

## Shell failures

Generated wrappers must:

- locate the command after global flags;
- invoke the binary once;
- accept exactly one validated PATH line;
- keep logs off stdout;
- restore the default after leaving a project;
- honor effective config path, project filename, install root, and auto-switch setting.

Profile updates require one complete marker block and same-directory transactional replacement. Preserve permissions/newline style and reject symlink destinations.

## Self-update failures

Check exact platform asset naming, unique checksum entry, bounded response size, downloaded binary `--version`, and retained backup. On Windows, inspect detached helper arguments and rollback; same-process replacement is not expected to work for a locked executable.

## Platform scripts

- Bash: run syntax checks on Linux and macOS, plus ShellCheck on Linux.
- PowerShell: run PSScriptAnalyzer on Windows.
- Batch: preserve PATH entry text and registry type; never enable delayed expansion while reading a PATH containing `!`.

The entire `test/` directory contains local-only helpers. It is ignored and must not be referenced by tracked CI.

## Coverage diagnosis

`make test-coverage` writes ignored `coverage.out` and `coverage/coverage.html`, then enforces 80% total and 70% CLI statement coverage. Add tests through production boundaries with temp directories and local servers; do not copy production logic into tests.
