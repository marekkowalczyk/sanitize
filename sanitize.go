//Inspired by:
//https://go.googlesource.com/text/+/master/runes/example_test.go
//https://www.dotnetperls.com/replace-go

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

func removeIllFormed(input string) (output string) {
	output, _, _ = transform.String(runes.ReplaceIllFormed(), input)
	return output
}

func toLower(input string) (output string) {
	output = strings.ToLower(input)
	return output
}

func replaceNonAlphaNum(input string) (output string) {
	replaceNonAlphaNum := runes.Map(func(r rune) rune {
		if !unicode.Is(unicode.Latin, r) && !unicode.IsDigit(r) {
			return '-'
		}
		return r
	})
	output, _, _ = transform.String(replaceNonAlphaNum, input)
	return output
}

func removeAccents(input string) (output string) {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	s, _, _ := transform.String(t, input)
	r := strings.NewReplacer("ł", "l", "Ł", "L", "ß", "ss")
	output = r.Replace(s)
	return output
}

func dedupHyp(input string) (output string) {
	reg, err := regexp.Compile("-{2,}")
	if err != nil {
		log.Fatal(err)
	}
	output = reg.ReplaceAllString(input, "-")
	return output
}

func trimEnds(input string) (output string) {
	output = strings.TrimFunc(input, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	return output
}

func sanitize(input string) (output string) {
	output = trimEnds(dedupHyp(replaceNonAlphaNum(removeAccents(toLower(removeIllFormed(input))))))
	return output
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: sanitize <text>...

Sanitize strings for safe use as filenames. Lowercases, strips diacritics,
replaces non-alphanumeric characters with hyphens, deduplicates hyphens,
and trims leading/trailing non-alphanumeric characters.

Multiple arguments are joined with hyphens.

Examples:
  sanitize "Hello, World!"          → hello-world
  sanitize "Zażółć gęślą jaźń"     → zazolc-gesla-jazn
  sanitize foo bar baz              → foo-bar-baz
`)
	}
	flag.Parse()

	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(1)
	}

	input := strings.Join(flag.Args(), "-")
	fmt.Println(sanitize(input))
}
