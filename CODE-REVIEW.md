# Code Review: sanitize

Reviewed: 2026-05-20

## Overall Assessment

This is a well-structured, focused utility that does one thing and does it correctly. The transformation pipeline is logically ordered and each function has a clear responsibility. For a first Go project, the code demonstrates good instincts: proper use of the `golang.org/x/text` library for Unicode handling, correct use of NFD decomposition for stripping diacritics, and a clean pipeline design. The special-casing of `ł`/`Ł` shows genuine understanding of the Unicode problem being solved.

## sanitize.go

### What works well

- **Pipeline composition** — the `sanitize()` function composes transforms in the right order (e.g., lowercasing before accent removal, accent removal before non-alphanumeric replacement).
- **Unicode correctness** — using `norm.NFD` + `unicode.Mn` removal + `norm.NFC` is the textbook approach for stripping diacritics.
- **`replaceNonAlphaNum` uses `unicode.Latin`** — this correctly restricts output to Latin-script characters, preventing non-Latin scripts from leaking into filenames.

### Issues

**1. `go.mod` is missing the `golang.org/x/text` dependency (high)**

`go.mod` has no `require` block. The module won't build from a clean checkout without first running `go get golang.org/x/text`. Running `go mod tidy` will fix this — it will add the dependency and generate `go.sum`.

**2. Regex recompilation on every call in `dedupHyp` (low)**

`regexp.Compile("-{2,}")` is called every time `dedupHyp` runs. In Go, the idiomatic pattern is to compile the regex once at package level:

```go
var dedupHypRe = regexp.MustCompile("-{2,}")

func dedupHyp(input string) string {
    return dedupHypRe.ReplaceAllString(input, "-")
}
```

`MustCompile` is preferred over `Compile` for constant patterns — it panics on invalid regex at startup rather than returning an error that you'd need to handle on every call. Since `"-{2,}"` is a fixed, known-valid pattern, `MustCompile` is safe here. For a CLI that runs once and exits, the performance difference is negligible, but it's the standard Go pattern.

**3. Named return values used as regular variables (low)**

All functions use named return values (`func foo(input string) (output string)`) but then just assign to `output` and explicitly `return output`. Named returns are a Go feature mainly useful for documentation or for "naked returns" in complex functions. Here they add verbosity without benefit. Simpler:

```go
func toLower(input string) string {
    return strings.ToLower(input)
}
```

This applies to all functions in the file.

**4. Variable shadows function name in `replaceNonAlphaNum` (medium)**

The local variable `replaceNonAlphaNum` inside the function `replaceNonAlphaNum` shadows the function name. This compiles fine but is confusing to read and would be flagged by linters. Rename the variable, e.g., `mapper` or `t`.

**5. No `go.sum` file (medium)**

After fixing `go.mod` with `go mod tidy`, commit the generated `go.sum` file too. It locks dependency versions for reproducible builds.

**6. No input validation (low)**

Running `sanitize` with no arguments produces an empty line. This is harmless but could print a usage message instead:

```go
if len(os.Args) < 2 {
    fmt.Fprintf(os.Stderr, "usage: sanitize <text>\n")
    os.Exit(1)
}
```

**7. `trimEnds` uses `unicode.IsLetter` while `replaceNonAlphaNum` uses `unicode.Latin` (low)**

`trimEnds` trims characters that are not `IsLetter` or `IsDigit`, while `replaceNonAlphaNum` keeps only `unicode.Latin` characters. This is an inconsistency — `trimEnds` would preserve a leading non-Latin letter that `replaceNonAlphaNum` already replaced with a hyphen. In practice this doesn't cause bugs because `replaceNonAlphaNum` runs first in the pipeline, so by the time `trimEnds` runs, no non-Latin letters remain. But if the functions were ever reordered or used independently, the behavior would be surprising. Aligning `trimEnds` to also use `unicode.Latin` would make the functions self-consistent.

## san.sh

**8. `echo` in parameter expansion is unnecessary (low)**

```bash
filen=$(echo "${oname%.*}")
```

The `echo` does nothing here. Simpler: `filen="${oname%.*}"`

**9. Filenames with single quotes will break the AppleScript (medium)**

In `DEVONthink-Sanitize-Filenames.applescript`, the name is interpolated with single quotes:

```applescript
set theNewName to do shell script "~/go/bin/sanitize " & "'" & theName & "'"
```

A filename containing a single quote (e.g., `O'Brien's notes`) will break the shell command or, worse, allow shell injection. Use `quoted form of theName` instead:

```applescript
set theNewName to do shell script "~/go/bin/sanitize " & quoted form of theName
```

**10. `san.sh` doesn't handle paths, only bare filenames (low)**

If called with `san.sh path/to/file.txt`, it will try to rename the whole path. This is fine if it's only meant to be used in the current directory, but worth noting.

## Recommendation: fold san.sh into the Go binary

`san.sh` is a thin wrapper that splits filenames from extensions, sanitizes both parts, and renames. All of this is straightforward in Go (`filepath.Ext()`, `strings.TrimSuffix()`, `os.Rename()`, `os.Stat()` for no-clobber). Folding it into the Go binary would:

- Eliminate the shell-specific bugs (#8, #10) and the dependency on `sanitize` being in `$PATH`
- Produce a single distributable binary
- Allow the AppleScript to call one tool with a flag instead of shelling out separately

Suggested CLI design:

```
sanitize "some text"             # current behavior — sanitize a string
sanitize -f file1.txt file2.PDF  # file rename mode
```

The file rename mode would: split name from extension, sanitize each part, check the target doesn't already exist, and rename (printing old → new like `mv -v`).

## Unix Citizenship

`sanitize` produces correct output but can't participate in Unix pipelines or workflows. Measured against standard CLI conventions:

### Missing (important)

**11. No stdin support (high)**

A proper Unix filter reads from stdin when no args are given. Currently `echo "foo" | sanitize` and `cat list.txt | sanitize` don't work. This is the biggest gap — it prevents composition with pipes.

**12. No meaningful exit codes (medium)**

Always exits 0. Should exit non-zero on error (no input, empty output). `./sanitize` with no args silently succeeds with an empty line.

**13. No `--help` / `-h` (medium)**

No usage message. Every Unix tool should have this.

### Missing (nice-to-have)

**14. No `--version` flag (low)**

Minor but standard for distributed binaries.

**15. Errors go to stdout (low)**

No distinction between output and diagnostics. Errors and usage messages should go to stderr.

**16. No `--` separator support (low)**

Can't distinguish flags from arguments starting with `-`.

**17. No per-line stdin processing (low)**

When reading from stdin, the tool should process one input per line and output one result per line. This enables `xargs`, `parallel`, and other standard composition patterns.

**18. No null-delimiter mode (low)**

For filenames with newlines (rare but real), tools like `find -print0 | xargs -0` expect null-delimited I/O. A `-0` flag would support this.

### What it already does right

- Single-purpose tool (one job, does it well)
- Output to stdout with trailing newline
- No chatty or decorative output
- Deterministic, no side effects

### Suggested priority

Adding stdin support (#11), exit codes (#12), and `--help` (#13) would make `sanitize` a proper Unix citizen that composes with other tools. These pair naturally with the `-f` file rename mode proposed above — a `flag`-based CLI would handle `--help`, `--`, and `-f` together.

## Summary

| # | File | Severity | Status | Description |
|---|------|----------|--------|-------------|
| 1 | go.mod | High | Fixed | Missing `require` for `golang.org/x/text`; run `go mod tidy` |
| 2 | sanitize.go | Low | Fixed | Regex recompiled on every call in `dedupHyp` |
| 3 | sanitize.go | Low | Fixed | Named returns add verbosity without benefit |
| 4 | sanitize.go | Medium | Fixed | Variable shadows function name in `replaceNonAlphaNum` |
| 5 | go.mod | Medium | Fixed | No `go.sum` committed |
| 6 | sanitize.go | Low | Fixed | No usage message when called with no arguments |
| 7 | sanitize.go | Low | Fixed | `trimEnds` and `replaceNonAlphaNum` use different letter predicates |
| 8 | san.sh | Low | Superseded | Unnecessary `echo` in parameter expansion |
| 9 | AppleScript | Medium | Open | Single quotes in filenames cause shell injection risk |
| 10 | san.sh | Low | Superseded | Doesn't handle file paths, only bare filenames |
| 11 | sanitize.go | High | Fixed | No stdin support — can't participate in pipelines |
| 12 | sanitize.go | Medium | Fixed | No meaningful exit codes |
| 13 | sanitize.go | Medium | Fixed | No `--help` / `-h` flag |
| 14 | sanitize.go | Low | Open | No `--version` flag |
| 15 | sanitize.go | Low | Fixed | Errors go to stdout instead of stderr |
| 16 | sanitize.go | Low | Fixed | No `--` separator support (provided by Go `flag` package) |
| 17 | sanitize.go | Low | Fixed | No per-line stdin processing |
| 18 | sanitize.go | Low | Open | No null-delimiter (`-0`) mode |

**Notes:** #8 and #10 are superseded by folding `san.sh` into the Go binary (`sanitize -f` / `san` symlink). #16 comes for free with Go's `flag` package. The AppleScript injection (#9) is still open.
