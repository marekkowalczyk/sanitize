//Inspired by:
//https://go.googlesource.com/text/+/master/runes/example_test.go
//https://www.dotnetperls.com/replace-go

package main

import (
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
	"log"
	"regexp"
	"strings"
	"unicode"
	"os"
	"fmt"
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
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
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
	r := strings.NewReplacer("ł", "l", "Ł", "L")
	output = r.Replace(s)
	return output
}

/*func replaceNonASCII(input string) (output string) {
	replaceNonASCII := runes.Map(func(r rune) rune {
		if !(r <= unicode.MaxASCII) {
			return '-'
		}
		return r
	})
	output, _, _ = transform.String(replaceNonASCII, input)
	return output
}
*/
/*func replaceSpaces(input string) (output string) {
	replaceSpaces := runes.Map(func(r rune) rune {
		if unicode.Is(unicode.Space, r) {
			return '-'
		}
		return r
	})
	output, _, _ = transform.String(replaceSpaces, input)
	return output
}*/

/*func replacePunct(input string) (output string) {
	replacePunct := runes.Map(func(r rune) rune {
		if unicode.Is(unicode.Punct, r) {
			return '-'
		}
		return r
	})
	output, _, _ = transform.String(replacePunct, input)
	return output
}
*/
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

func sanitize(input string) (output string){
output = trimEnds(dedupHyp(replaceNonAlphaNum(removeAccents(toLower(removeIllFormed(input))))))
return output
}

func main() {

		input := strings.Join(os.Args[1:], "-")
		
/*	const input string = "Golang basics - writing unit tests"*/
	/*	fmt.Println(trimEnds(dedupHyp(replacePunct(replaceSpaces(removeAccents(strings.ToLower(replaceNonASCII(removeIllFormed(input)))))))))*/

	fmt.Println(sanitize(input))

}
