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

## Summary

| # | File | Severity | Description |
|---|------|----------|-------------|
| 1 | go.mod | High | Missing `require` for `golang.org/x/text`; run `go mod tidy` |
| 2 | sanitize.go | Low | Regex recompiled on every call in `dedupHyp` |
| 3 | sanitize.go | Low | Named returns add verbosity without benefit |
| 4 | sanitize.go | Medium | Variable shadows function name in `replaceNonAlphaNum` |
| 5 | go.mod | Medium | No `go.sum` committed |
| 6 | sanitize.go | Low | No usage message when called with no arguments |
| 7 | sanitize.go | Low | `trimEnds` and `replaceNonAlphaNum` use different letter predicates |
| 8 | san.sh | Low | Unnecessary `echo` in parameter expansion |
| 9 | AppleScript | Medium | Single quotes in filenames cause shell injection risk |
| 10 | san.sh | Low | Doesn't handle file paths, only bare filenames |
