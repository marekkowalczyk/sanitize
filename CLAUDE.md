# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

A Go CLI tool that sanitizes/normalizes strings for safe use as filenames. It lowercases, strips diacritics, replaces non-alphanumeric characters with hyphens, deduplicates hyphens, and trims ends.

## Build, Test, and Run

```bash
go build .                          # build
go test -v                          # run full test suite (250+ cases)
go test -run TestSanitize -v        # run a specific test group
go test -bench=. -benchmem -run=^$  # run benchmarks
./sanitize "input text here"        # sanitize text
./sanitize -f "My File.txt"         # rename a file
echo "text" | ./sanitize            # read from stdin
find . -print0 | ./sanitize -0      # null-delimited stdin
./sanitize --version                # print version
```

Note: on macOS 10.14, use `GOTOOLCHAIN=local` to prevent automatic toolchain upgrades that fail on this OS version.

Dependencies are managed via Go modules (`go.mod`). External dependencies are `golang.org/x/text` for Unicode normalization and rune manipulation, and `github.com/spf13/pflag` for POSIX-style flag parsing.

## Architecture

Single-file Go program (`sanitize.go`) with a transformation pipeline composed via function nesting:

```
input -> removeIllFormed -> toLower -> removeAccents -> replaceNonAlphaNum -> dedupHyp -> trimEnds -> output
```

Key design decisions:
- Diacritics are stripped using Unicode NFD decomposition + removal of `unicode.Mn` (Mark, Nonspacing) category runes
- Polish `ł`/`Ł` and German `ß` are special-cased with direct string replacement because they are standalone characters, not combining sequences
- Both `replaceNonAlphaNum` and `trimEnds` use `unicode.Latin` (not `unicode.Letter`) to restrict output to Latin script characters only
- Multiple CLI arguments are joined with `-` before processing
- Version can be set at build time via `-ldflags "-X main.version=1.0.0"`

## CLI modes

- **Text mode** (default): `sanitize "text"` -- sanitize a string
- **Stdin mode**: `echo "text" | sanitize` -- read lines from stdin when no args and input is piped
- **Null-delimited stdin**: `find . -print0 | sanitize -0` -- use null bytes instead of newlines as delimiters
- **File mode**: `sanitize -f file.txt` or `san file.txt` -- rename files (splits name/extension, sanitizes each part)
- **Recursive mode**: `sanitize -r dir/` or `san -r dir/` -- walk a directory tree depth-first, renaming all files and directories; handles SIGINT gracefully (stops between renames)
- `-n` (dry-run) implies `-f` (file mode), since dry-run only makes sense for renames
- When invoked as `san` (via symlink), file rename mode is automatic
- `--version` prints version and exits
- `--help` / `-h` prints usage and exits

## Test suite

Tests are in `sanitize_test.go` and cover:
- Unit tests for each pipeline stage
- Unit tests for `renameOne`/`renameFiles`/`renameRecursive` (using `io.Writer` for output capture)
- Integration tests for the full `sanitize()` function
- Pipeline ordering verification
- Idempotency tests
- Context cancellation (graceful interrupt)
- CLI integration tests (stdin, args, exit codes, `-f` mode, `-r` recursive, `san` symlink, `-0` null mode, `-n` dry run, `--version`)
- Benchmarks for each pipeline stage and full `sanitize()`/`sanitizeFilename()`

## Supporting Files

- `legacy/san.sh` -- Legacy bash wrapper, superseded by `sanitize -f` / `san` symlink
- `legacy/config.yml` -- Old Super-Linter config, replaced by `.github/workflows/test.yml`
- `contrib/DEVONthink-Sanitize-Filenames.applescript` -- AppleScript for sanitizing DEVONthink record names (saves originals in Finder Comment field). See README for installation instructions.
- `docs/AAR.md` -- After Action Review / continuous improvement log
- `docs/BACKLOG.md` -- Feature backlog and planned improvements
- `docs/CODE-REVIEW.md` -- Code review with issue tracker (all issues resolved)
- `references/` -- Book summary YAMLs (reference material, not part of the tool)
- `.github/workflows/test.yml` -- CI: runs `go build` and `go test` on push/PR
- `.github/workflows/release.yml` -- Release: runs goreleaser on `v*` tags to build cross-platform binaries
- `.goreleaser.yml` -- Goreleaser config: builds for linux/darwin/windows (amd64/arm64)
- `sanitize.1` -- Man page in troff format (`nroff -man sanitize.1` to preview)
