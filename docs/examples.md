# Examples

## First setup

```bash
govman init
govman install latest
govman use latest --default
go version
gofmt -h
```

Reload the shell profile printed by `govman init` before expecting wrapper-managed session changes.

## Exact and partial versions

```bash
govman install 1.25       # newest available 1.25 patch
govman install 1.25.4     # exact release
govman install 1.26rc1    # exact prerelease

govman use 1.25           # highest installed 1.25 patch
govman use 1.25.4         # exact installed release
```

## Project pinning

```bash
cd my-project
govman install 1.25.4
govman use 1.25.4 --local
git add .govman-goversion
git commit -m "build: pin Go 1.25.4"
```

The generated shell hook reads `.govman-goversion` from the current directory. Run `govman refresh` after creating or editing it if a directory-change hook has not fired.

## Batch operations

Quote wildcard patterns so the shell does not expand them:

```bash
govman install '1.24.*' --yes
govman install '1.26*' --unstable --yes
govman uninstall '1.23.*' --yes
```

## Testing multiple installed versions

In an initialized interactive shell:

```bash
for version in 1.25.4 1.24.10; do
  govman use "$version"
  go test ./...
done
govman use default
```

In a non-interactive POSIX shell, evaluate the validated PATH command explicitly:

```bash
for version in 1.25.4 1.24.10; do
  eval "$(govman use "$version")"
  go test ./...
done
```

## GitHub Actions consumer example

```yaml
name: test
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6
      - name: Install govman and selected Go
        shell: bash
        run: |
          set -euo pipefail
          curl -fsSL https://raw.githubusercontent.com/justjundana/govman/main/scripts/install.sh | bash
          export PATH="$HOME/.govman/bin:$PATH"
          version=$(tr -d '[:space:]' < .govman-goversion)
          govman install "$version"
          eval "$(govman use "$version")"
          go version
          go test ./...
```

Environment changes do not persist automatically between GitHub Actions steps; keep activation and dependent commands in one shell block or write the selected toolchain path to `GITHUB_PATH`.

## Corporate proxy

```bash
export HTTPS_PROXY=http://proxy.corp.example:8080
export HTTP_PROXY=http://proxy.corp.example:8080
export NO_PROXY=localhost,127.0.0.1
govman install latest
```

## Controlled release endpoint

The reserved `mirror.*` fields do not route traffic. A compatible internal service must provide Go release metadata and archives:

```yaml
go_releases:
  api_url: https://go-releases.corp.example/api?include=all
  download_url: https://go-releases.corp.example/dl/%s
  cache_expiry: 10m
```

Metadata must include the exact archive filename, OS, architecture, size, kind, and SHA-256 expected by govman.

## Maintenance

```bash
govman list
govman prune --yes
govman clean
```

`prune` keeps active, default, and current project-local versions. `clean` removes downloaded archives but not installed toolchains.

## Self-update

```bash
govman selfupdate --check
govman selfupdate
```

The release must contain the exact platform binary and `checksums.txt`. Development builds must be updated from source or reinstalled.
