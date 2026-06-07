# Changelog

## v2.0.0 (2026-06-07)

Complete rewrite of the sanitization pipeline with significantly expanded
Unicode coverage, file rename capabilities, and safety guarantees.

### Breaking changes

- **Latin-only output**: `replaceNonAlphaNum` now uses `unicode.Latin` instead
  of `unicode.Letter`, so non-Latin script characters (Chinese, Arabic,
  Cyrillic, etc.) are stripped rather than passed through.
- **Expanded special-cases table**: 80+ entries for characters that don't
  NFD-decompose (ł, ß, đ, ø, æ, œ, ħ, ı, þ, ð, ŋ, ĳ, ə, ɛ, ɔ, and many
  more). Previously stripped characters now transliterate to Latin equivalents,
  changing output for the same input.
- **Exit code 2** for postcondition validation failures (exit code 1 remains
  for operational errors like file collisions).

### New features

- **File rename mode** (`-f`): sanitize and rename files, splitting
  name/extension.
- **Recursive rename** (`-r`): depth-first directory tree rename with SIGINT
  graceful stop.
- **Dry-run mode** (`-n`): preview renames without touching the filesystem.
  Implies `-f`.
- **Stdin support**: read lines from piped input for pipeline composition.
- **Null-delimited stdin** (`-0`): for use with `find -print0`.
- **`san` symlink**: when invoked as `san`, file rename mode is automatic.
  Adaptive help text shows `san`-specific usage.
- **`--version` flag**: prints version; settable at build time via ldflags.
- **`--help` / `-h`**: built-in usage text.
- **POSIX flag combining**: switched to `pflag` for `-nrf` style flag groups.
- **Man page** (`sanitize.1`): troff-format manual page.
- **Goreleaser**: cross-platform binary builds (linux/darwin/windows,
  amd64/arm64) on `v*` tags.

### Safety and correctness

- **Postcondition validation**: every output from `sanitize()` and
  `sanitizeFilename()` is checked by a validation gate before returning.
  `validate` enforces `[a-z0-9-]` only, no leading/trailing/consecutive
  hyphens. `validateFilename` additionally allows one dot or dotfile form.
- **Adversarial test suite**: LLM-generated edge cases covering Unicode
  normalization, unhandled Latin scripts, Go case-folding quirks, path
  traversal, dotfile creation, and malicious payloads.
- **400+ test cases** across unit, integration, idempotency, context
  cancellation, and CLI tests.
- **No-clobber protection**: file renames refuse to overwrite existing files.

### Performance

- **Cached transformer chains**: `golang.org/x/text/transform` chains are
  allocated once and reused.
- **Benchmarks** for each pipeline stage and full `sanitize()`/`sanitizeFilename()`.

### Other

- DEVONthink AppleScript for sanitizing record names (shell injection fix
  included).
- CI workflow (`.github/workflows/test.yml`).
- Repository reorganized: `dev/`, `contrib/`, `references/`.

## v1.0.1

Fixed bug when file had no extension.

## v1.0.0

Initial working version.
