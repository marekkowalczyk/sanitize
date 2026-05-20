# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

A Go CLI tool that sanitizes/normalizes strings for safe use as filenames. It lowercases, strips diacritics, replaces non-alphanumeric characters with hyphens, deduplicates hyphens, and trims ends.

## Build, Test, and Run

```bash
go build .                          # build
go test -v                          # run full test suite (155 cases)
go test -run TestSanitize -v        # run a specific test group
./sanitize "input text here"        # sanitize text
./sanitize -f "My File.txt"         # rename a file
echo "text" | ./sanitize            # read from stdin
find . -print0 | ./sanitize -0      # null-delimited stdin
./sanitize --version                # print version
```

Note: on macOS 10.14, use `GOTOOLCHAIN=local` to prevent automatic toolchain upgrades that fail on this OS version.

Dependencies are managed via Go modules (`go.mod`). The main external dependency is `golang.org/x/text` for Unicode normalization and rune manipulation.

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
- When invoked as `san` (via symlink), file rename mode is automatic
- `--version` prints version and exits
- `--help` / `-h` prints usage and exits

## Test suite

Tests are in `sanitize_test.go` and cover:
- Unit tests for each pipeline stage
- Integration tests for the full `sanitize()` function
- Pipeline ordering verification
- Idempotency tests
- CLI integration tests (stdin, args, exit codes, `-f` mode, `san` symlink, `-0` null mode, `--version`)

## Supporting Files

- `san.sh` -- Legacy bash wrapper, superseded by `sanitize -f` / `san` symlink
- `DEVONthink-Sanitize-Filenames.applescript` -- AppleScript for sanitizing DEVONthink record names (saves originals in Finder Comment field). See README for installation instructions.
- `CODE-REVIEW.md` -- Code review with issue tracker (all issues resolved)
