# Backlog

Feature ideas for future versions.

## Special-case replacements: hardcoded, not configurable (decision)

**Status: decided — keep hardcoded.** The special-case replacements (190 entries as of 2026-06-07, covering Western/Central European, Icelandic, Sami, Dutch, African languages, Croatian digraphs, typographic ligatures, Roman numerals, super/subscript digits, vulgar fractions, letterlike symbols, currency symbols, and common signs — sourced from Unicode CLDR Latin-ASCII, AnyAscii, and Unidecode) are linguistic facts, not preferences. A config file was considered and rejected:

- The tool's core differentiator is zero-config, opinionated output. A config file would make output depend on external state, breaking reproducibility and composability.
- The `specialCases` table in `sanitize.go` is visible, self-documenting, and trivial to extend — adding a new entry is one line of code plus a test case.
- These are bug fixes (characters that slip through NFD), not user preferences.
- Users who need custom transliteration tables should use detox, which is designed for that.

## Pre-scan for collisions before renaming (safety)

**Priority: high.** The transformation is lossy, so multiple files can sanitize to the same name (e.g., `Café.txt` and `cafe!.txt` both become `cafe.txt`). The current no-clobber check catches this per-file, but the result is a partial rename -- some files moved, others blocked. With `-r` on a large tree this can leave things in a messy half-state.

Proposed fix: before performing any renames, build the full old→new mapping and check for collisions. If any target name appears more than once, abort with a clear report of the conflicts *before touching the filesystem*. This makes the operation atomic in intent: either everything can be renamed cleanly, or nothing is renamed. The check should apply to both `-f` with multiple arguments and `-r`.

This is the single most important safety improvement before widespread use of `-r`.

## Preserve compound extensions (`-e`)

Treat `.tar.gz`, `.tar.bz2`, etc. as a single extension unit instead of splitting on the last dot only. Could use a known-extensions list or a flag like `-e .tar.gz`.

## Verbose mode (`-v`)

Show all files processed, including those already clean (skipped). Currently only renames are printed. Useful for verifying that `-r` traversed the expected tree.

## Configurable separator

Allow a character other than `-` as the replacement separator. For example, `_` for `snake_case` output. Could be `--sep=_` or `-s _`.

## Undo / backup manifest

Write a manifest of old→new renames (e.g., `.sanitize-manifest.json`) so a batch rename can be reversed. Useful for `-r` on large directory trees where mistakes are costly.

## Automatic `san` symlink on install

Currently `go install` only creates the `sanitize` binary. Users must manually `ln -s` to get `san`. (Author migrated from old san.sh to symlink on 2026-05-21.) Options considered:

- **Homebrew formula** — the natural place. A formula can create the symlink as a post-install step. This is the strongest argument for implementing the Homebrew tap (see below).
- **Makefile `install` target** — `go install && ln -sf $(go env GOPATH)/bin/sanitize /usr/local/bin/san`. Works but requires `make install` instead of `go install`.
- **goreleaser post-install** — not supported for tarballs; only packaging formats (deb/rpm/brew) handle post-install hooks.
- **Ship two binaries** (`cmd/san/main.go` wrapper) — overkill for a symlink.

**Recommendation:** Solve this via the Homebrew tap rather than adding Go-level complexity. The adaptive help text (showing `san`-specific usage when invoked as `san`) is already implemented.

## Homebrew tap

**Priority: medium.** Create a personal Homebrew tap (`homebrew-sanitize` repo on GitHub) so users can `brew tap marekkowalczyk/sanitize && brew install sanitize`. Goreleaser can auto-generate and push the Formula on each release — it's ~10 lines of config in `.goreleaser.yml` plus an empty GitHub repo. Natural next step given goreleaser is already set up. Submitting to homebrew-core (the official tap) is premature — requires meaningful adoption/stars and goes through their review process. MacPorts requires manually submitting a Portfile to their ports tree, has a smaller audience than Homebrew, and has no goreleaser integration — not worth the effort unless someone asks.

## Makefile for dev workflow

**Priority: low.** Encode common commands (`build`, `test`, `bench`, `install`, `lint`, `release-dry-run`) in a Makefile. The main value is encoding `GOTOOLCHAIN=local` and long commands like the bench invocation so they don't need to be remembered. However, Go's built-in tools already cover most of what Make would do — `go build`, `go test`, `go install`, `go clean` all work out of the box, and goreleaser handles releases via CI. The Makefile is really just a shortcut file for 3-4 long commands. If the `GOTOOLCHAIN=local` requirement goes away (macOS upgrade), the value drops further. Inspired by Chapter 9 of *Small, Sharp Software Tools*.

## Richer pipeline examples in README

Show `sanitize` composed with other Unix tools in realistic workflows: `find -print0 | sanitize -0 | xargs -0`, using `tee` to log original names, combining with `sort`/`uniq` to detect would-be collisions before renaming. Inspired by Chapter 5 of *Small, Sharp Software Tools*.

## Review v3.0.0 transliteration design choices (deliberation)

**Priority: high.** The v3.0.0 transliteration expansion was done in a single session and some choices deserve more deliberation:

- **ASCII symbols** ($→usd, &→and, @→at, %→pct, +→plus): are these the right mappings? `&→and` feels natural, but `$→usd` is USD-centric (could be AUD, CAD, etc.). `%→pct` adds content that wasn't alphabetic.
- **Currency abbreviations** (€→eur, £→gbp, ₿→btc): these are abbreviations, not phonetic transliterations — a different *kind* of special case than ł→l. Are they appropriate for a filename sanitizer?
- **Vulgar fractions** (½→1-2): is `1-2` a reasonable filename representation of "one half"?
- **Rejected candidates** (#, ~, *, ^, =, !, ?, |): is the rejection list correct? `#` in particular is debatable — "Issue #42" losing the `#` is arguably worse than "issue-hash42".

Needs dedicated deliberation, not a quick decision. May result in reverting some choices before wider adoption.

## Classic vs extended transliteration mode (`--classic` / `--extended`)

**Priority: medium. Status: idea — needs design work.** Add a flag to switch between "classic" behavior (pre-v3 — symbols just become hyphens) and "extended" behavior (current — symbols transliterated to words/abbreviations). This would let users opt out of opinionated transliterations like &→and or $→usd.

Open questions:

- What's the default? Extended (current) or classic? Changing the default later is a breaking change.
- Does a mode flag violate the zero-config philosophy that differentiates sanitize from detox?
- Could this be a build-time flag instead of runtime (bake the choice into the binary)?
- Is there a simpler design — e.g., only transliterate Latin-script characters (the original scope) and leave symbols as hyphens, making the "extended" set opt-in?

## Refactor pipeline from nested calls to sequential assignment

**Priority: low.** The current pipeline is a deeply nested function composition:

```go
result := trimEnds(dedupHyp(replaceNonAlphaNum(toLower(removeAccents(removeIllFormed(input))))))
```

This is compact but hard to read at 6 levels deep, hard to debug (can't inspect intermediate values), and will get worse if a 7th stage is added. Replace with sequential assignment:

```go
s := removeIllFormed(input)
s = removeAccents(s)
s = toLower(s)
s = replaceNonAlphaNum(s)
s = dedupHyp(s)
s = trimEnds(s)
```

Alternatives considered and rejected:
- **Slice of functions + loop** (`[]func(string) string`): adds indirection for only 6 steps — not enough to justify the abstraction.
- **`transform.Chain`** from `golang.org/x/text/transform`: only works with `transform.Transformer` interface types; our functions are plain `string→string`, so adapters would be needed. Not worth it.

Sequential assignment is the most Go-idiomatic, trivial to step through in a debugger, and makes it obvious where to insert a debug print.

## Shell completions (bash/zsh)

Hand-written completion scripts for bash and zsh. Only 5 flags so it's simple, but improves discoverability. Could be installed manually or shipped with goreleaser. Inspired by Chapter 6 of *Small, Sharp Software Tools*.
