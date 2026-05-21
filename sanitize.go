//Inspired by:
//https://go.googlesource.com/text/+/master/runes/example_test.go
//https://www.dotnetperls.com/replace-go

package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	flag "github.com/spf13/pflag"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// Characters that don't decompose via NFD into base + combining mark.
// Add new entries here when a Latin-script character is not handled
// by the standard NFD accent-stripping pipeline.
var specialCases = []struct{ from, to string }{
	{"ł", "l"}, {"Ł", "L"}, // Polish barred L
	{"ß", "ss"},             // German eszett
	{"đ", "d"}, {"Đ", "D"}, // Croatian/Vietnamese barred D
	{"ø", "o"}, {"Ø", "O"}, // Danish/Norwegian slashed O
	{"æ", "ae"}, {"Æ", "AE"}, // Danish/Norwegian/Icelandic ligature
	{"œ", "oe"}, {"Œ", "OE"}, // French ligature
	{"ħ", "h"}, {"Ħ", "H"}, // Maltese barred H
	{"ı", "i"},              // Turkish dotless I (lowercase)
}

var (
	dedupHypRe           = regexp.MustCompile("-{2,}")
	illFormedTransform   = runes.ReplaceIllFormed()
	nonAlphaNumTransform = runes.Map(func(r rune) rune {
		if !unicode.Is(unicode.Latin, r) && !unicode.IsDigit(r) {
			return '-'
		}
		return r
	})
	accentTransform     = transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	specialCaseReplacer = newSpecialCaseReplacer()
)

func newSpecialCaseReplacer() *strings.Replacer {
	var pairs []string
	for _, sc := range specialCases {
		pairs = append(pairs, sc.from, sc.to)
	}
	return strings.NewReplacer(pairs...)
}

func removeIllFormed(input string) string {
	s, _, _ := transform.String(illFormedTransform, input)
	return s
}

func toLower(input string) string {
	return strings.ToLower(input)
}

func replaceNonAlphaNum(input string) string {
	s, _, _ := transform.String(nonAlphaNumTransform, input)
	return s
}

func removeAccents(input string) string {
	s, _, _ := transform.String(accentTransform, input)
	return specialCaseReplacer.Replace(s)
}

func dedupHyp(input string) string {
	return dedupHypRe.ReplaceAllString(input, "-")
}

func trimEnds(input string) string {
	return strings.TrimFunc(input, func(r rune) bool {
		return !unicode.Is(unicode.Latin, r) && !unicode.IsDigit(r)
	})
}

func sanitize(input string) string {
	return trimEnds(dedupHyp(replaceNonAlphaNum(removeAccents(toLower(removeIllFormed(input))))))
}

// sanitizeFilename sanitizes a filename, treating name and extension separately.
// Dotfiles (e.g., .gitignore) are preserved as-is if already clean.
func sanitizeFilename(name string) string {
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)

	if base == "" {
		// Dotfile like .gitignore — ext is the whole name
		return ext
	}

	newBase := sanitize(base)
	if ext == "" {
		return newBase
	}

	newExt := sanitize(ext[1:]) // strip the dot before sanitizing
	if newExt == "" {
		return newBase
	}
	return newBase + "." + newExt
}

func sameFile(a, b string) bool {
	infoA, errA := os.Stat(a)
	infoB, errB := os.Stat(b)
	if errA != nil || errB != nil {
		return false
	}
	return os.SameFile(infoA, infoB)
}

// renameOne renames a single file or directory. Returns 0 on success or skip, 1 on error.
func renameOne(path string, dryRun bool, w io.Writer) int {
	dir := filepath.Dir(path)
	oldName := filepath.Base(path)
	newName := sanitizeFilename(oldName)

	if newName == oldName {
		return 0
	}

	dst := filepath.Join(dir, newName)
	if _, err := os.Stat(dst); err == nil && !sameFile(path, dst) {
		fmt.Fprintf(w, "sanitize: %s → %s: target already exists\n", path, dst)
		return 1
	}

	if dryRun {
		fmt.Fprintf(w, "%s → %s\n", path, dst)
		return 0
	}

	if err := os.Rename(path, dst); err != nil {
		fmt.Fprintf(w, "sanitize: %s: %v\n", path, err)
		return 1
	}
	fmt.Fprintf(w, "%s → %s\n", path, dst)
	return 0
}

func renameFiles(paths []string, dryRun bool, w io.Writer) int {
	exitCode := 0
	for _, path := range paths {
		if code := renameOne(path, dryRun, w); code != 0 {
			exitCode = code
		}
	}
	return exitCode
}

func renameRecursive(ctx context.Context, root string, dryRun bool, w io.Writer) int {
	exitCode := 0

	// Collect all entries first, then process depth-first.
	// Deepest entries are renamed first so that renaming a child
	// doesn't invalidate a parent path before we process it.
	var paths []string

	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Fprintf(w, "sanitize: %s: %v\n", path, err)
			exitCode = 1
			return nil
		}
		if path != root {
			paths = append(paths, path)
		}
		return nil
	})

	for i := len(paths) - 1; i >= 0; i-- {
		if ctx.Err() != nil {
			fmt.Fprintf(w, "sanitize: interrupted, stopping\n")
			return 1
		}
		if code := renameOne(paths[i], dryRun, w); code != 0 {
			exitCode = code
		}
	}

	return exitCode
}

func scanNullTerminated(data []byte, atEOF bool) (advance int, token []byte, err error) {
	for i := 0; i < len(data); i++ {
		if data[i] == 0 {
			return i + 1, data[:i], nil
		}
	}
	if atEOF && len(data) > 0 {
		return len(data), data, nil
	}
	return 0, nil, nil
}

func invokedAsSan() bool {
	base := filepath.Base(os.Args[0])
	return base == "san"
}

var version = "dev"

func main() {
	fileMode := flag.BoolP("file", "f", false, "rename files instead of sanitizing text")
	recursive := flag.BoolP("recursive", "r", false, "recursively rename files in directories")
	dryRun := flag.BoolP("dry-run", "n", false, "dry run: show what would be renamed without renaming")
	nullDelim := flag.BoolP("null", "0", false, "use null byte as delimiter instead of newline (for stdin mode)")
	showVersion := flag.Bool("version", false, "print version and exit")
	isSan := invokedAsSan()
	flag.Usage = func() {
		if isSan {
			fmt.Fprintf(os.Stderr, `Usage: san [-rn] <file|dir>...

Sanitize filenames. Lowercases, strips diacritics, replaces non-alphanumeric
characters with hyphens, deduplicates hyphens, and trims leading/trailing
non-alphanumeric characters.

Flags:
  -r, --recursive   recursively rename files in directories
  -n, --dry-run     show what would be renamed without renaming
      --version     print version and exit
  -h, --help        print this help

Short flags can be combined: -rn equals -r -n.

Examples:
  san "My File.PDF"                 → my-file.pdf
  san *.txt                         → rename multiple files
  san -n *.txt                      → dry run (show renames)
  san -r ~/Downloads/               → recursive rename
  san -rn ~/Downloads/              → recursive dry run
`)
		} else {
			fmt.Fprintf(os.Stderr, `Usage: sanitize [flags] <text>...
       sanitize -f [-n] <file>...
       sanitize -r [-n] <dir>...
       san [-rn] <file|dir>...
       command | sanitize [-0]

Sanitize strings for safe use as filenames. Lowercases, strips diacritics,
replaces non-alphanumeric characters with hyphens, deduplicates hyphens,
and trims leading/trailing non-alphanumeric characters.

Multiple arguments are joined with hyphens. With no arguments, reads
lines from stdin (one input per line, one output per line).

Flags:
  -f, --file        rename files instead of sanitizing text
  -r, --recursive   recursively rename files in directories
  -n, --dry-run     show what would be renamed without renaming (implies -f)
  -0, --null        use null byte as delimiter (for stdin mode)
      --version     print version and exit
  -h, --help        print this help

Use -- to separate flags from arguments starting with -.
Short flags can be combined: -fn equals -f -n.

Examples:
  sanitize "Hello, World!"          → hello-world
  sanitize "Zażółć gęślą jaźń"     → zazolc-gesla-jazn
  sanitize foo bar baz              → foo-bar-baz
  echo "Café Résumé" | sanitize     → cafe-resume
  find . -print0 | sanitize -0      → null-delimited I/O
  sanitize -f "My File.PDF"         → my-file.pdf
  sanitize -fn *.txt                → dry run (show renames)
  sanitize -r ~/Downloads/          → recursive rename
  san "My File.PDF"                 → my-file.pdf
  san -r ~/Downloads/               → recursive rename via san
  sanitize -- -hello                → treats -hello as text
`)
		}
	}
	flag.Parse()

	if *showVersion {
		name := "sanitize"
		if isSan {
			name = "san"
		}
		fmt.Printf("%s %s\n", name, version)
		return
	}

	isFileMode := *fileMode || *recursive || *dryRun || isSan

	if isFileMode {
		if flag.NArg() == 0 {
			fmt.Fprintf(os.Stderr, "sanitize: -f requires at least one file argument\n")
			os.Exit(1)
		}
		if *recursive {
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
			defer stop()
			exitCode := 0
			for _, dir := range flag.Args() {
				if code := renameRecursive(ctx, dir, *dryRun, os.Stderr); code != 0 {
					exitCode = code
				}
			}
			os.Exit(exitCode)
		}
		os.Exit(renameFiles(flag.Args(), *dryRun, os.Stderr))
	}

	if flag.NArg() == 0 {
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) != 0 {
			// No args, no piped stdin — show usage
			flag.Usage()
			os.Exit(1)
		}
		scanner := bufio.NewScanner(os.Stdin)
		if *nullDelim {
			scanner.Split(scanNullTerminated)
		}
		delim := "\n"
		if *nullDelim {
			delim = "\x00"
		}
		for scanner.Scan() {
			entry := scanner.Text()
			if entry == "" {
				continue
			}
			result := sanitize(entry)
			if result == "" {
				continue
			}
			fmt.Print(result + delim)
		}
		if err := scanner.Err(); err != nil {
			fmt.Fprintf(os.Stderr, "sanitize: %v\n", err)
			os.Exit(1)
		}
		return
	}

	input := strings.Join(flag.Args(), "-")
	fmt.Println(sanitize(input))
}
