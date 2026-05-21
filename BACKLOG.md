# Backlog

Feature ideas for future versions.

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

## Makefile for dev workflow

**Priority: low.** Encode common commands (`build`, `test`, `bench`, `install`, `lint`, `release-dry-run`) in a Makefile. The main value is encoding `GOTOOLCHAIN=local` and long commands like the bench invocation so they don't need to be remembered. However, Go's built-in tools already cover most of what Make would do — `go build`, `go test`, `go install`, `go clean` all work out of the box, and goreleaser handles releases via CI. The Makefile is really just a shortcut file for 3-4 long commands. If the `GOTOOLCHAIN=local` requirement goes away (macOS upgrade), the value drops further. Inspired by Chapter 9 of *Small, Sharp Software Tools*.

## Richer pipeline examples in README

Show `sanitize` composed with other Unix tools in realistic workflows: `find -print0 | sanitize -0 | xargs -0`, using `tee` to log original names, combining with `sort`/`uniq` to detect would-be collisions before renaming. Inspired by Chapter 5 of *Small, Sharp Software Tools*.

## Shell completions (bash/zsh)

Hand-written completion scripts for bash and zsh. Only 5 flags so it's simple, but improves discoverability. Could be installed manually or shipped with goreleaser. Inspired by Chapter 6 of *Small, Sharp Software Tools*.
