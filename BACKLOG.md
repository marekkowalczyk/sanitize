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
