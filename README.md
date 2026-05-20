# sanitize

A Go CLI tool that sanitizes/normalizes strings for safe use as filenames.

Inspired by the principles in Brian P. Hogan's *Small, Sharp Software Tools*, `sanitize` aims to be a well-behaved Unix citizen: it does one thing well, works with text streams, uses standard I/O conventions, stays quiet, and composes with other tools via pipes.

It lowercases, strips diacritics, replaces non-alphanumeric characters with hyphens, deduplicates hyphens, and trims leading/trailing non-alphanumeric characters. Output is restricted to Latin-script characters, digits, and hyphens.

## Installation

```bash
go install github.com/marekkowalczyk/sanitize@latest
```

Or from a local checkout:

```bash
go install
```

Both install the `sanitize` binary to `$GOPATH/bin` (typically `~/go/bin`).

To also use the `san` shortcut for file renaming:

```bash
ln -s ~/go/bin/sanitize /usr/local/bin/san
```

## Usage

### Sanitize text

```bash
sanitize "Hello, World!"              # hello-world
sanitize "Zażółć gęślą jaźń"          # zazolc-gesla-jazn
sanitize "Straße nach München"         # strasse-nach-munchen
sanitize foo bar baz                   # foo-bar-baz (multiple args joined)
```

### Read from stdin

```bash
echo "Café Résumé" | sanitize          # cafe-resume
cat filenames.txt | sanitize           # one output per line
```

When no arguments are given and input is piped, `sanitize` reads one line at a time from stdin and outputs one sanitized line per line. Blank lines and lines that sanitize to empty are skipped.

#### Null-delimited mode (`-0`)

For filenames that may contain newlines, use `-0` for null-delimited I/O (like `find -print0` / `xargs -0`):

```bash
find . -print0 | sanitize -0           # null-delimited input and output
find . -print0 | sanitize -0 | xargs -0 echo
```

### Rename files (`-f` or `san`)

```bash
sanitize -f "My Document.PDF"         # renames to my-document.pdf
sanitize -f *.txt                      # rename multiple files (shell expands the glob)
san "My Document.PDF"                  # same as sanitize -f
san *.txt                              # same as sanitize -f *.txt
```

File rename mode splits the filename from its extension, sanitizes each part separately, and renames the file. It will not overwrite existing files. Renames are printed to stderr.

When the binary is invoked as `san` (via symlink), file rename mode is enabled automatically without needing `-f`.

Glob patterns (`*.txt`, `IMG_*.jpg`, etc.) are expanded by the shell before `sanitize` sees them -- this is standard Unix behavior and requires no special handling by the tool.

### Recursive rename (`-r`)

```bash
sanitize -r ~/Downloads/          # recursively rename all files and dirs
sanitize -rn ~/Downloads/         # dry run: show what would be renamed
san -r ~/Downloads/               # same thing via san symlink
```

Recursive mode walks a directory tree depth-first, sanitizing all filenames and directory names. Deepest entries are renamed first so that parent renames don't invalidate child paths. The `-r` flag implies file mode (`-f`). Combines with `-n` for dry run.

### Dry run (`-n`)

```bash
sanitize -f -n *.txt                   # show what would be renamed
san -n *.txt                           # same thing
sanitize -fn *.txt                     # combined short flags also work
```

### Other flags

```bash
sanitize --version                     # print version
sanitize --help                        # print usage
```

Short flags can be combined: `-fn` is the same as `-f -n`. Long forms are also available: `--file`, `--recursive`, `--dry-run`, `--null`.

## Transformation pipeline

```
input -> removeIllFormed -> toLower -> removeAccents -> replaceNonAlphaNum -> dedupHyp -> trimEnds -> output
```

1. **removeIllFormed** -- replace ill-formed UTF-8 sequences
2. **toLower** -- lowercase the entire string
3. **removeAccents** -- NFD decomposition + strip combining marks (unicode.Mn), plus special-case replacements for standalone characters that don't decompose (`ł` -> `l`, `ß` -> `ss`)
4. **replaceNonAlphaNum** -- replace anything outside `unicode.Latin` and digits with `-`
5. **dedupHyp** -- collapse runs of `--` into a single `-`
6. **trimEnds** -- strip leading/trailing non-Latin, non-digit characters

## Handling of non-ASCII characters

All non-ASCII characters are transformed to their ASCII equivalents where possible:

```
Kąt na łące żre źrebię   ->   kat-na-lace-zre-zrebie
```

This is achieved by Unicode NFD decomposition followed by removal of [Mark, Nonspacing](https://www.fileformat.info/info/unicode/category/Mn/index.htm) characters. For example, `ą` is `a` combined with `COMBINING OGONEK` (U+0328) -- removing the combining mark leaves `a`.

### Special cases

Some characters are standalone Latin letters that don't decompose into base + combining mark:

- `ł`/`Ł` -> `l`/`L` (Polish barred L)
- `ß` -> `ss` (German eszett)

These are handled with direct string replacement.

### Non-Latin scripts

Characters from non-Latin scripts (Chinese, Cyrillic, Arabic, etc.) are replaced with hyphens and then cleaned up by deduplication and trimming:

```
Hello你好World   ->   hello-world
```

## DEVONthink integration

`DEVONthink-Sanitize-Filenames.applescript` sanitizes names of selected DEVONthink records, setting the `Finder Comment` field to the original filename. Note: the existing Finder Comment is overwritten.

### Installing the script in DEVONthink

1. Open DEVONthink
2. Go to **DEVONthink > Preferences > Scripts** (or in DEVONthink 3, the Scripts folder is at `~/Library/Application Scripts/com.devon-technologies.think3/Menu`)
3. Copy or symlink the script into the DEVONthink scripts folder:
   ```bash
   cp DEVONthink-Sanitize-Filenames.applescript \
     ~/Library/Application\ Scripts/com.devon-technologies.think3/Menu/Sanitize\ Filenames.scpt
   ```
   Or compile and copy:
   ```bash
   osacompile -o ~/Library/Application\ Scripts/com.devon-technologies.think3/Menu/Sanitize\ Filenames.scpt \
     DEVONthink-Sanitize-Filenames.applescript
   ```
4. The script appears in the **Scripts** menu inside DEVONthink
5. Select one or more records, then run the script from the menu

The script requires `sanitize` to be installed at `~/go/bin/sanitize` (the default `go install` location).

### What the script does

For each selected record:
1. Saves the original name to the record's **Finder Comment** field
2. Runs `sanitize` on the name
3. Sets the record name to the sanitized result

## Migrating from san.sh

If you previously used `san.sh` as a bash wrapper for file renaming, the functionality is now built into the Go binary. To migrate:

1. **Build and test locally** (without touching your installed tools):
   ```bash
   go build -o ./sanitize .
   ln -s ./sanitize ./san
   ./san "Test File.txt"              # verify it works
   ```

2. **Install the updated binary**:
   ```bash
   go install                         # updates ~/go/bin/sanitize
   ```

3. **Replace the old san.sh** (back up first):
   ```bash
   cp /usr/local/bin/san /usr/local/bin/san.sh.bak
   ln -sf ~/go/bin/sanitize /usr/local/bin/san
   ```

4. **Verify**:
   ```bash
   which san                          # should show /usr/local/bin/san
   san --help                         # should show usage with -f mode
   ```

After migration, `/usr/local/bin/san.sh.bak` can be removed when you're confident everything works. The original `san.sh` is preserved in `legacy/` for reference.

### Behavioral differences from san.sh

- Dotfiles (e.g., `.gitignore`) are preserved as-is (san.sh would strip the dot)
- Already-clean files are skipped silently (san.sh would call `mv` anyway)
- Full path support (san.sh only worked with bare filenames)
- Case-only renames work on macOS (san.sh's `mv -n` would block them)

## Building with a version tag

```bash
go build -ldflags "-X main.version=1.0.0" .
```

Without `-ldflags`, the version defaults to `dev`.

## Design philosophy

`sanitize` follows the Unix tool conventions described in Brian P. Hogan's *Small, Sharp Software Tools*:

- **Do one thing well** -- sanitize strings for filenames, nothing else
- **Work with text streams** -- reads stdin, writes to stdout, one entry per line
- **Use standard I/O** -- output to stdout, diagnostics to stderr, meaningful exit codes
- **Be quiet** -- no banners, progress messages, or decorative output
- **Be a filter** -- sits in the middle of a pipeline: `cat list.txt | sanitize | xargs ...`
- **Support null delimiters** -- `-0` for filenames containing newlines
- **Dry run** -- `-n` shows what would happen without doing it

### POSIX compliance

Flag handling follows POSIX conventions:

| Convention | Example |
|---|---|
| Short flags | `-f`, `-r`, `-n`, `-0` |
| Combined short flags | `-fn` equals `-f -n`, `-rn` equals `-r -n` |
| Long flags | `--file`, `--recursive`, `--dry-run`, `--null`, `--version`, `--help` |
| `--` separator | `sanitize -- -hello` treats `-hello` as text, not a flag |
| Unknown flag | prints error to stderr, exits 2 |
| `--help` | prints usage to stderr, exits 0 |
| `--version` | prints version to stdout, exits 0 |

Exit codes:

| Code | Meaning |
|---|---|
| 0 | Success |
| 1 | Runtime error (rename failed, target exists, etc.) |
| 2 | Usage error (unknown flag, missing arguments) |

### The `-f` concession

The `-f` file rename mode is a pragmatic concession. Strictly speaking, a pure Unix tool would only transform text, and you'd compose it with `mv`:

```bash
for f in *.txt; do mv "$f" "$(sanitize "$f")"; done
```

The `-f` flag bundles transform + rename into one operation because it's a common workflow that's error-prone to do by hand (splitting extensions, handling no-clobber, case-insensitive filesystems). The `san` symlink makes this even more convenient. This trades Unix purity for daily usability.

## Caution

Different input strings can produce identical output. This is by design -- the tool is lossy.

### Collision risk in file rename mode

Because the transformation is lossy, multiple files in the same directory can sanitize to the same name. For example, `Café.txt`, `cafe!.txt`, and `CAFÉ.txt` all become `cafe.txt`.

**Current protection:** Before each rename, the tool checks whether the target already exists (`os.Stat` + no-clobber). If it does, the rename is skipped with an error. This prevents data loss -- `os.Rename` on Unix silently overwrites, so this check is the sole safeguard.

**Remaining risk -- partial renames:** When renaming multiple files (`-f *.txt`) or recursively (`-r`), the first collision succeeds and subsequent ones are skipped. This leaves you in a half-renamed state: some files moved, others didn't. With `-r` on a deep directory tree this can be especially messy, as some directories may have been renamed while files inside sibling directories were blocked.

**Mitigation:** Always use `-n` (dry run) first on unfamiliar directories to check for collisions before committing to renames. See BACKLOG.md for a planned pre-scan feature that would detect all collisions up front and abort before any renames happen.

## Testing

```bash
go test -v
```

The test suite includes 210+ cases covering individual pipeline stages, full integration, pipeline ordering, idempotency, file renaming, recursive directory renaming, dry run, null-delimited I/O, stdin processing, combined flags, and CLI behavior.
