# sanitize

A Go CLI tool that sanitizes/normalizes strings for safe use as filenames.

Inspired by the principles in Brian P. Hogan's *Small, Sharp Software Tools*, `sanitize` aims to be a well-behaved Unix citizen: it does one thing well, works with text streams, uses standard I/O conventions, stays quiet, and composes with other tools via pipes.

It lowercases, strips diacritics, replaces non-alphanumeric characters with hyphens, deduplicates hyphens, and trims leading/trailing non-alphanumeric characters. Output is restricted to Latin-script characters, digits, and hyphens.

Designed for mission-critical use: every output is validated against a strict postcondition (`[a-z0-9-]` for strings, `[a-z0-9.-]` for filenames) before being returned. If the pipeline produces any disallowed character, the tool fails with a diagnostic error rather than silently passing through an unsafe result.

## Installation

### Pre-built binaries

Download from [GitHub Releases](https://github.com/marekkowalczyk/sanitize/releases) -- binaries are available for Linux, macOS, and Windows (amd64 and arm64).

### From source

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

Recursive mode walks a directory tree depth-first, sanitizing all filenames and directory names. Deepest entries are renamed first so that parent renames don't invalidate child paths. The `-r` flag implies file mode (`-f`). Combines with `-n` for dry run. Handles SIGINT gracefully -- if you press Ctrl+C during a recursive rename, it stops cleanly between files rather than mid-rename.

### Dry run (`-n`)

```bash
sanitize -n *.txt                      # show what would be renamed (-n implies -f)
sanitize -f -n *.txt                   # explicit -f also works
san -n *.txt                           # same thing via san
sanitize -fn *.txt                     # combined short flags also work
```

The `-n` flag implies file mode (`-f`), since dry-run only makes sense for renames.

### Other flags

```bash
sanitize --version                     # print version
sanitize --help                        # print usage
```

Short flags can be combined: `-fn` is the same as `-f -n`. Long forms are also available: `--file`, `--recursive`, `--dry-run`, `--null`.

## Transformation pipeline

```
input -> removeIllFormed -> toLower -> removeAccents -> replaceNonAlphaNum -> dedupHyp -> trimEnds -> validate -> output
```

1. **removeIllFormed** -- replace ill-formed UTF-8 sequences
2. **toLower** -- lowercase the entire string
3. **removeAccents** -- NFD decomposition + strip combining marks (unicode.Mn), plus special-case replacements for standalone characters that don't decompose (`ł` -> `l`, `ß` -> `ss`)
4. **replaceNonAlphaNum** -- replace anything outside `unicode.Latin` and digits with `-`
5. **dedupHyp** -- collapse runs of `--` into a single `-`
6. **trimEnds** -- strip leading/trailing non-Latin, non-digit characters
7. **validate** -- postcondition check: verify output contains only `[a-z0-9-]`, no leading/trailing or consecutive hyphens. Returns an error if any disallowed character is present (names the offending character and its Unicode codepoint)

## Handling of non-ASCII characters

All non-ASCII characters are transformed to their ASCII equivalents where possible:

```
Kąt na łące żre źrebię   ->   kat-na-lace-zre-zrebie
```

This is achieved by Unicode NFD decomposition followed by removal of [Mark, Nonspacing](https://www.fileformat.info/info/unicode/category/Mn/index.htm) characters. For example, `ą` is `a` combined with `COMBINING OGONEK` (U+0328) -- removing the combining mark leaves `a`.

### Special cases

Some characters are standalone Latin letters that don't decompose into base + combining mark. These are handled via a `specialCases` table (80+ entries, sourced from Unicode CLDR Latin-ASCII, AnyAscii, and Unidecode) with direct string replacement:

| Character | Replacement | Language/Use |
|---|---|---|
| `ł`/`Ł` | `l`/`L` | Polish barred L |
| `ß`/`ẞ` | `ss`/`SS` | German eszett + capital sharp S |
| `đ`/`Đ` | `d`/`D` | Croatian/Vietnamese barred D |
| `ø`/`Ø` | `o`/`O` | Danish/Norwegian slashed O |
| `æ`/`Æ` | `ae`/`AE` | Danish/Norwegian/Icelandic ligature |
| `œ`/`Œ` | `oe`/`OE` | French ligature |
| `ħ`/`Ħ` | `h`/`H` | Maltese barred H |
| `ı` | `i` | Turkish dotless I |
| `þ`/`Þ` | `th`/`Th` | Icelandic thorn |
| `ð`/`Ð` | `d`/`D` | Icelandic/Faroese eth |
| `ŋ`/`Ŋ` | `ng`/`Ng` | Sami/African eng |
| `ŧ`/`Ŧ` | `t`/`T` | Sami barred T |
| `ĳ`/`Ĳ` | `ij`/`IJ` | Dutch IJ ligature |
| `ŀ`/`Ŀ` | `l`/`L` | Catalan middle-dot L |
| `ĸ` | `k` | Greenlandic kra |
| `ſ` | `s` | Historical long S |
| `ə`/`Ə` | `e`/`E` | Azerbaijani/African schwa |
| `ɛ`/`Ɛ` | `e`/`E` | African open E (Ewe, Akan) |
| `ɔ`/`Ɔ` | `o`/`O` | African open O (Akan, Ewe) |
| `ɓ`/`Ɓ` | `b`/`B` | African hooked B (Hausa, Fula) |
| `ɗ`/`Ɗ` | `d`/`D` | African hooked D (Hausa, Fula) |
| `ƙ`/`Ƙ` | `k`/`K` | African hooked K (Hausa) |
| `ʃ`/`Ʃ` | `sh`/`Sh` | African esh (Pan-Nigerian) |
| `ʒ`/`Ʒ` | `zh`/`Zh` | African ezh (Skolt Sami) |
| `ǆ`/`ǉ`/`ǌ` | `dz`/`lj`/`nj` | Croatian digraphs |
| `ﬀ`/`ﬁ`/`ﬂ`/`ﬃ`/`ﬄ` | `ff`/`fi`/`fl`/`ffi`/`ffl` | Typographic ligatures |
| ... | ... | + 20 more African/historical entries |

### Non-Latin scripts

Characters from non-Latin scripts (Chinese, Cyrillic, Arabic, etc.) are replaced with hyphens and then cleaned up by deduplication and trimming:

```
Hello你好World   ->   hello-world
```

## DEVONthink integration

`contrib/DEVONthink-Sanitize-Filenames.applescript` sanitizes names of selected DEVONthink records, setting the `Finder Comment` field to the original filename. Note: the existing Finder Comment is overwritten.

### Installing the script in DEVONthink

1. Open DEVONthink
2. Go to **DEVONthink > Preferences > Scripts** (or in DEVONthink 3, the Scripts folder is at `~/Library/Application Scripts/com.devon-technologies.think3/Menu`)
3. Copy or symlink the script into the DEVONthink scripts folder:
   ```bash
   cp contrib/DEVONthink-Sanitize-Filenames.applescript \
     ~/Library/Application\ Scripts/com.devon-technologies.think3/Menu/Sanitize\ Filenames.scpt
   ```
   Or compile and copy:
   ```bash
   osacompile -o ~/Library/Application\ Scripts/com.devon-technologies.think3/Menu/Sanitize\ Filenames.scpt \
     contrib/DEVONthink-Sanitize-Filenames.applescript
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

## Building and releasing

### Local build with version tag

```bash
go build -ldflags "-X main.version=1.0.0" .
```

Without `-ldflags`, the version defaults to `dev`.

### Releasing

Releases are automated via [GoReleaser](https://goreleaser.com/) and GitHub Actions. To cut a release:

```bash
git tag v1.0.0
git push --tags
```

This triggers `.github/workflows/release.yml`, which builds cross-platform binaries (linux/darwin/windows, amd64/arm64) and publishes them as a GitHub Release.

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
| Flag implication | `-n` implies `-f`, `-r` implies `-f` |
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
| 2 | Usage error or postcondition failure (unknown flag, missing arguments, disallowed character in output) |

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

**Mitigation:** Always use `-n` (dry run) first on unfamiliar directories to check for collisions before committing to renames. See docs/BACKLOG.md for a planned pre-scan feature that would detect all collisions up front and abort before any renames happen.

## Testing

```bash
go test -v
```

The test suite includes 400+ cases covering individual pipeline stages, postcondition validation, full integration, pipeline ordering, idempotency, file renaming, recursive directory renaming, dry run, null-delimited I/O, stdin processing, combined flags, context cancellation, CLI behavior, and an adversarial suite (`sanitize_adversarial_test.go`) with LLM-generated edge cases targeting Unicode normalization gotchas, unhandled Latin script boundaries, Go case-folding quirks, path traversal, accidental dotfile creation, and malicious payloads (null bytes, control characters, PUA codepoints, Cyrillic homoglyphs).

### Benchmarks

```bash
go test -bench=. -benchmem -run=^$
```

Benchmarks cover each pipeline stage and the full `sanitize()`/`sanitizeFilename()` functions.

### Man page

A man page is included as `sanitize.1`. To install locally:

```bash
cp sanitize.1 /usr/local/share/man/man1/
man sanitize
```

The man page is also included in goreleaser archives.

## Comparison with similar tools

| Tool | Language | What it does | File rename | Recursive | Dry run |
|---|---|---|---|---|---|
| **sanitize** | Go | Opinionated filename sanitizer (lowercase, strip diacritics, Latin-only) | Yes (`-f`, `san`) | Yes (`-r`) | Yes (`-n`) |
| **[detox](https://github.com/dharple/detox)** | C | Configurable filename cleanup via sequence files (`.detoxrc`) | Yes | Yes | Yes |
| **[rename/prename](https://metacpan.org/pod/File::Rename)** | Perl | General-purpose renamer using Perl expressions | Yes | No | Yes |
| **[slugify](https://github.com/benlinton/slugify)** (various) | Bash/Python/Node | String-to-slug conversion | Varies | No | No |
| **[convmv](https://www.j3e.de/linux/convmv/)** | Perl | Filename *encoding* conversion (e.g., ISO-8859-1 to UTF-8) | Yes | Yes | Yes |
| **[mmv](https://github.com/itchyny/mmv)** | C | Batch rename with wildcard patterns | Yes | No | No |
| **[vidir](https://joeyh.name/code/moreutils/)** | Perl | Interactive rename in `$EDITOR` | Yes | No | N/A |
| **go-slugify, gosimple/slug** | Go | URL slug generation | No (library only) | No | No |
| **filenamify, sanitize-filename** (npm) | JS | Strip OS-illegal characters | No (library only) | No | No |
| **python-slugify** (pip) | Python | Transliteration via text-unidecode | Minimal CLI | No | No |

### How sanitize differs

**Closest competitor is detox**, which also cleans filenames, transliterates UTF-8, and has recursive + dry-run modes. detox is more configurable (sequence files), but `sanitize` is zero-config, restricts output to Latin script, and handles special cases (Polish `ł`, German `ß`, Danish `ø`/`æ`, French `œ`, Croatian `đ`, Maltese `ħ`, Turkish `ı`) via NFD decomposition + a `specialCases` replacement table.

**rename/prename** is far more powerful but requires writing Perl expressions -- it's a general renamer, not a sanitizer. `sanitize` trades flexibility for zero-config simplicity.

**slugify scripts** are the closest conceptual match, but are typically string-only transformers with no file operations, recursion, or null-delimited I/O.

### What sanitize offers that others don't

- **Zero-config opinionated pipeline** -- no regex, config files, or flags needed for the common case
- **Latin-script-only output** -- unique among these tools; non-Latin characters (Chinese, Cyrillic, Arabic) are stripped
- **Special-case diacritics** -- 80+ standalone Latin characters that don't NFD-decompose are handled via a dedicated replacement table covering Western/Central European, Icelandic, Sami, Dutch, African languages, Croatian digraphs, and typographic ligatures
- **Single static binary** -- Go, no runtime dependencies, cross-platform
- **Full CLI integration** -- `-f` file rename, `-r` recursive, `-n` dry run, `-0` null-delimited stdin, `san` symlink, POSIX-compliant flags

### What others offer that sanitize doesn't

- **detox** -- configurable transliteration tables and wipeup sequences
- **rename** -- arbitrary transformation logic via Perl expressions
- **vidir** -- interactive editing of filenames in your text editor
- **python-slugify** -- broader transliteration coverage via `text-unidecode` (handles more scripts than NFD decomposition)
