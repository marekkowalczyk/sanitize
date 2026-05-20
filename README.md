# sanitize

A Go CLI tool that sanitizes/normalizes strings for safe use as filenames.

It lowercases, strips diacritics, replaces non-alphanumeric characters with hyphens, deduplicates hyphens, and trims leading/trailing non-alphanumeric characters. Output is restricted to Latin-script characters, digits, and hyphens.

## Installation

```bash
go install
```

This installs the `sanitize` binary to `$GOPATH/bin` (typically `~/go/bin`).

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

### Rename files (`-f` or `san`)

```bash
sanitize -f "My Document.PDF"         # renames to my-document.pdf
sanitize -f *.txt                      # rename multiple files
san "My Document.PDF"                  # same as sanitize -f
san *.txt                              # same as sanitize -f *.txt
```

File rename mode splits the filename from its extension, sanitizes each part separately, and renames the file. It will not overwrite existing files. Renames are printed to stderr.

When the binary is invoked as `san` (via symlink), file rename mode is enabled automatically without needing `-f`.

## Transformation pipeline

```
input -> removeIllFormed -> toLower -> removeAccents -> replaceNonAlphaNum -> dedupHyp -> trimEnds -> output
```

1. **removeIllFormed** -- replace ill-formed UTF-8 sequences
2. **toLower** -- lowercase the entire string
3. **removeAccents** -- NFD decomposition + strip combining marks (unicode.Mn), plus special-case replacements for standalone characters that don't decompose (`l` -> `l`, `ss` -> `ss`)
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

- `l`/`L` -> `l`/`L` (Polish barred L)
- `ss` -> `ss` (German eszett)

These are handled with direct string replacement.

### Non-Latin scripts

Characters from non-Latin scripts (Chinese, Cyrillic, Arabic, etc.) are replaced with hyphens and then cleaned up by deduplication and trimming:

```
Hello你好World   ->   hello-world
```

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

After migration, `san.sh` and `san.sh.bak` can be removed when you're confident everything works.

### Behavioral differences from san.sh

- Dotfiles (e.g., `.gitignore`) are preserved as-is (san.sh would strip the dot)
- Already-clean files are skipped silently (san.sh would call `mv` anyway)
- Full path support (san.sh only worked with bare filenames)
- Case-only renames work on macOS (san.sh's `mv -n` would block them)

## Caution

Different input strings can produce identical output. This is by design -- the tool is lossy.

## DEVONthink integration

`DEVONthink-Sanitize-Filenames.applescript` sanitizes names of selected DEVONthink records, setting the `Finder Comment` field to the original filename. Note: the existing Finder Comment is overwritten.

## Testing

```bash
go test -v
```

The test suite includes 150+ cases covering individual pipeline stages, full integration, pipeline ordering, idempotency, file renaming, stdin processing, and CLI behavior.
