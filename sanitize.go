//Inspired by:
//https://go.googlesource.com/text/+/master/runes/example_test.go
//https://www.dotnetperls.com/replace-go

package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	flag "github.com/spf13/pflag"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

var dedupHypRe = regexp.MustCompile("-{2,}")

func removeIllFormed(input string) string {
	s, _, _ := transform.String(runes.ReplaceIllFormed(), input)
	return s
}

func toLower(input string) string {
	return strings.ToLower(input)
}

func replaceNonAlphaNum(input string) string {
	mapper := runes.Map(func(r rune) rune {
		if !unicode.Is(unicode.Latin, r) && !unicode.IsDigit(r) {
			return '-'
		}
		return r
	})
	s, _, _ := transform.String(mapper, input)
	return s
}

func removeAccents(input string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	s, _, _ := transform.String(t, input)
	return strings.NewReplacer("ł", "l", "Ł", "L", "ß", "ss").Replace(s)
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

func renameFiles(paths []string, dryRun bool) int {
	exitCode := 0
	for _, path := range paths {
		dir := filepath.Dir(path)
		oldName := filepath.Base(path)
		newName := sanitizeFilename(oldName)

		if newName == oldName {
			continue
		}

		dst := filepath.Join(dir, newName)
		if _, err := os.Stat(dst); err == nil && !sameFile(path, dst) {
			fmt.Fprintf(os.Stderr, "sanitize: %s → %s: target already exists\n", path, dst)
			exitCode = 1
			continue
		}

		if dryRun {
			fmt.Fprintf(os.Stderr, "%s → %s\n", path, dst)
			continue
		}

		if err := os.Rename(path, dst); err != nil {
			fmt.Fprintf(os.Stderr, "sanitize: %s: %v\n", path, err)
			exitCode = 1
			continue
		}
		fmt.Fprintf(os.Stderr, "%s → %s\n", path, dst)
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
	dryRun := flag.BoolP("dry-run", "n", false, "dry run: show what would be renamed without renaming")
	nullDelim := flag.BoolP("null", "0", false, "use null byte as delimiter instead of newline (for stdin mode)")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: sanitize [flags] <text>...
       sanitize -f [-n] <file>...
       san [-n] <file>...
       command | sanitize [-0]

Sanitize strings for safe use as filenames. Lowercases, strips diacritics,
replaces non-alphanumeric characters with hyphens, deduplicates hyphens,
and trims leading/trailing non-alphanumeric characters.

Multiple arguments are joined with hyphens. With no arguments, reads
lines from stdin (one input per line, one output per line).

Flags:
  -f, --file      rename files instead of sanitizing text
  -n, --dry-run   show what would be renamed without renaming
  -0, --null      use null byte as delimiter (for stdin mode)
      --version   print version and exit
  -h, --help      print this help

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
  san "My File.PDF"                 → my-file.pdf
  sanitize -- -hello                → treats -hello as text
`)
	}
	flag.Parse()

	if *showVersion {
		fmt.Printf("sanitize %s\n", version)
		return
	}

	if *fileMode || invokedAsSan() {
		if flag.NArg() == 0 {
			fmt.Fprintf(os.Stderr, "sanitize: -f requires at least one file argument\n")
			os.Exit(1)
		}
		os.Exit(renameFiles(flag.Args(), *dryRun))
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
