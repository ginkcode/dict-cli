# Local Release Builds

`make release` cross-builds `dict` and `gram` for Linux and macOS (amd64 and arm64) and packages portable tarballs in `dist/`. It builds everywhere Go runs — no packaging tools, no publishing, no GitHub access needed.

## Requirements

- Go 1.26.2+ (see `go.mod`)
- `make`, `tar`, `install` (standard on Linux and macOS)
- A configured LLM backend at runtime — releases embed no keys or model config; both tools read your existing config file and environment

## Usage

```sh
make release          # build all platforms into dist/ and build/
make VERSION=1.2.3 release   # override the version (defaults to git describe)
make dist             # alias for release
make clean            # remove build/ and dist/
```

Binaries land at `build/<os>-<arch>/{dict,gram}`; archives at `dist/dict-cli_<VERSION>_<os>_<arch>.tar.gz` (same layout as CI releases).

## Choosing an archive

| Platform | Archive |
| --- | --- |
| Linux x86_64 | `dict-cli_<VERSION>_linux_amd64.tar.gz` |
| Linux ARM64 | `dict-cli_<VERSION>_linux_arm64.tar.gz` |
| macOS Intel | `dict-cli_<VERSION>_darwin_amd64.tar.gz` |
| macOS Apple Silicon | `dict-cli_<VERSION>_darwin_arm64.tar.gz` |

All binaries are built with `CGO_ENABLED=0`. Linux binaries are statically linked; macOS binaries are Mach-O executables for the selected architecture. Actual macOS execution must be tested on a Mac.

## Installing

Linux archives unpack a `usr/bin` tree:

```sh
tar -xzf dict-cli_<VERSION>_linux_amd64.tar.gz   # creates usr/bin/dict and usr/bin/gram
sudo install -m 755 usr/bin/dict usr/bin/gram /usr/local/bin/
```

macOS archives contain both binaries at the root:

```sh
tar -xzf dict-cli_<VERSION>_darwin_arm64.tar.gz
sudo install -m 755 dict gram /usr/local/bin/
```

Both tools are standalone — install both, either one alone works fine.

Verify with:

```sh
dict -version
gram -version
```

## Notes

- Clipboard copying in `gram` shells out to OS clipboard utilities (`xclip`/`xsel`/`wl-copy` on Linux, `pbcopy` on macOS) at runtime. Disabling CGO in these builds removes compile-time library dependencies only — the clipboard utilities must still be present on the target machine.
- CI (`.github/workflows/release.yml`) still builds and publishes official releases on tag push; `make release` is for local, unpublished builds only.