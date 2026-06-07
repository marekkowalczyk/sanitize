package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"testing"
)

// TestGenerateTransliterations generates references/transliterations.csv
// from the specialCases table, showing the final (lowercased) output.
// Run with: GENERATE=1 go test -run TestGenerateTransliterations
//
// Without GENERATE=1, verifies the CSV is up to date.

func buildCSVRows() [][]string {
	rows := [][]string{{"character", "codepoint", "name", "output"}}

	for _, sc := range specialCases {
		r := []rune(sc.from)
		if len(r) == 0 {
			continue
		}
		char := sc.from
		codepoint := fmt.Sprintf("U+%04X", r[0])
		name := unicodeName(r[0])
		output, err := sanitize(sc.from)
		if err != nil {
			// Character sanitizes to empty (stripped entirely)
			output = ""
		}

		rows = append(rows, []string{char, codepoint, name, output})
	}
	return rows
}

func unicodeName(r rune) string {
	// Map of codepoints to human-readable names for the special cases table.
	names := map[rune]string{
		'ł': "Latin Small Letter L with Stroke",
		'Ł': "Latin Capital Letter L with Stroke",
		'ß': "Latin Small Letter Sharp S",
		'ẞ': "Latin Capital Letter Sharp S",
		'đ': "Latin Small Letter D with Stroke",
		'Đ': "Latin Capital Letter D with Stroke",
		'ø': "Latin Small Letter O with Stroke",
		'Ø': "Latin Capital Letter O with Stroke",
		'æ': "Latin Small Letter Ae",
		'Æ': "Latin Capital Letter Ae",
		'œ': "Latin Small Ligature Oe",
		'Œ': "Latin Capital Ligature Oe",
		'ħ': "Latin Small Letter H with Stroke",
		'Ħ': "Latin Capital Letter H with Stroke",
		'ı': "Latin Small Letter Dotless I",
		'þ': "Latin Small Letter Thorn",
		'Þ': "Latin Capital Letter Thorn",
		'ð': "Latin Small Letter Eth",
		'Ð': "Latin Capital Letter Eth",
		'ŋ': "Latin Small Letter Eng",
		'Ŋ': "Latin Capital Letter Eng",
		'ŧ': "Latin Small Letter T with Stroke",
		'Ŧ': "Latin Capital Letter T with Stroke",
		'ĳ': "Latin Small Ligature Ij",
		'Ĳ': "Latin Capital Ligature Ij",
		'ŀ': "Latin Small Letter L with Middle Dot",
		'Ŀ': "Latin Capital Letter L with Middle Dot",
		'ĸ': "Latin Small Letter Kra",
		'ſ': "Latin Small Letter Long S",
		'ŉ': "Latin Small Letter N Preceded by Apostrophe",
		'ƕ': "Latin Small Letter Hv",
		'Ƕ': "Latin Capital Letter Hwair",
		'ⱥ': "Latin Small Letter A with Stroke",
		'Ⱥ': "Latin Capital Letter A with Stroke",
		'ə': "Latin Small Letter Schwa",
		'Ə': "Latin Capital Letter Schwa",
		'ɛ': "Latin Small Letter Open E",
		'Ɛ': "Latin Capital Letter Open E",
		'ɔ': "Latin Small Letter Open O",
		'Ɔ': "Latin Capital Letter Open O",
		'ɓ': "Latin Small Letter B with Hook",
		'Ɓ': "Latin Capital Letter B with Hook",
		'ɗ': "Latin Small Letter D with Hook",
		'Ɗ': "Latin Capital Letter D with Hook",
		'ɖ': "Latin Small Letter D with Tail",
		'Ɖ': "Latin Capital Letter D with Tail",
		'ƙ': "Latin Small Letter K with Hook",
		'Ƙ': "Latin Capital Letter K with Hook",
		'ƒ': "Latin Small Letter F with Hook",
		'Ƒ': "Latin Capital Letter F with Hook",
		'ɲ': "Latin Small Letter N with Left Hook",
		'Ɲ': "Latin Capital Letter N with Left Hook",
		'ɨ': "Latin Small Letter I with Stroke",
		'Ɨ': "Latin Capital Letter I with Stroke",
		'ʉ': "Latin Small Letter U Bar",
		'Ʉ': "Latin Capital Letter U Bar",
		'ʊ': "Latin Small Letter Upsilon",
		'Ʊ': "Latin Capital Letter Upsilon",
		'ʋ': "Latin Small Letter V with Hook",
		'Ʋ': "Latin Capital Letter V with Hook",
		'ƴ': "Latin Small Letter Y with Hook",
		'Ƴ': "Latin Capital Letter Y with Hook",
		'ƶ': "Latin Small Letter Z with Stroke",
		'Ƶ': "Latin Capital Letter Z with Stroke",
		'ʃ': "Latin Small Letter Esh",
		'Ʃ': "Latin Capital Letter Esh",
		'ʒ': "Latin Small Letter Ezh",
		'Ʒ': "Latin Capital Letter Ezh",
		'ǝ': "Latin Small Letter Turned E",
		'Ǝ': "Latin Capital Letter Reversed E",
		'Ǆ': "Latin Capital Letter Dz with Caron",
		'ǅ': "Latin Capital Letter D with Small Letter Z with Caron",
		'ǆ': "Latin Small Letter Dz with Caron",
		'Ǉ': "Latin Capital Letter Lj",
		'ǈ': "Latin Capital Letter L with Small Letter J",
		'ǉ': "Latin Small Letter Lj",
		'Ǌ': "Latin Capital Letter Nj",
		'ǋ': "Latin Capital Letter N with Small Letter J",
		'ǌ': "Latin Small Letter Nj",
		'Ǳ': "Latin Capital Letter Dz",
		'ǲ': "Latin Capital Letter D with Small Letter Z",
		'ǳ': "Latin Small Letter Dz",
		0xFB00: "Latin Small Ligature Ff",
		0xFB01: "Latin Small Ligature Fi",
		0xFB02: "Latin Small Ligature Fl",
		0xFB03: "Latin Small Ligature Ffi",
		0xFB04: "Latin Small Ligature Ffl",
		0xFB05: "Latin Small Ligature Long S T",
		0xFB06: "Latin Small Ligature St",
		// Roman numerals
		'Ⅰ': "Roman Numeral One", 'Ⅱ': "Roman Numeral Two", 'Ⅲ': "Roman Numeral Three",
		'Ⅳ': "Roman Numeral Four", 'Ⅴ': "Roman Numeral Five", 'Ⅵ': "Roman Numeral Six",
		'Ⅶ': "Roman Numeral Seven", 'Ⅷ': "Roman Numeral Eight", 'Ⅸ': "Roman Numeral Nine",
		'Ⅹ': "Roman Numeral Ten", 'Ⅺ': "Roman Numeral Eleven", 'Ⅻ': "Roman Numeral Twelve",
		'Ⅼ': "Roman Numeral Fifty", 'Ⅽ': "Roman Numeral One Hundred",
		'Ⅾ': "Roman Numeral Five Hundred", 'Ⅿ': "Roman Numeral One Thousand",
		'ⅰ': "Small Roman Numeral One", 'ⅱ': "Small Roman Numeral Two",
		'ⅲ': "Small Roman Numeral Three", 'ⅳ': "Small Roman Numeral Four",
		'ⅴ': "Small Roman Numeral Five", 'ⅵ': "Small Roman Numeral Six",
		'ⅶ': "Small Roman Numeral Seven", 'ⅷ': "Small Roman Numeral Eight",
		'ⅸ': "Small Roman Numeral Nine", 'ⅹ': "Small Roman Numeral Ten",
		'ⅺ': "Small Roman Numeral Eleven", 'ⅻ': "Small Roman Numeral Twelve",
		'ⅼ': "Small Roman Numeral Fifty", 'ⅽ': "Small Roman Numeral One Hundred",
		'ⅾ': "Small Roman Numeral Five Hundred", 'ⅿ': "Small Roman Numeral One Thousand",
		// Ordinal indicators
		'ª': "Feminine Ordinal Indicator", 'º': "Masculine Ordinal Indicator",
		// Superscript letters
		'ⁱ': "Superscript Latin Small Letter I", 'ⁿ': "Superscript Latin Small Letter N",
		// Superscript digits
		'⁰': "Superscript Zero", '¹': "Superscript One", '²': "Superscript Two",
		'³': "Superscript Three", '⁴': "Superscript Four", '⁵': "Superscript Five",
		'⁶': "Superscript Six", '⁷': "Superscript Seven", '⁸': "Superscript Eight",
		'⁹': "Superscript Nine",
		// Subscript digits
		'₀': "Subscript Zero", '₁': "Subscript One", '₂': "Subscript Two",
		'₃': "Subscript Three", '₄': "Subscript Four", '₅': "Subscript Five",
		'₆': "Subscript Six", '₇': "Subscript Seven", '₈': "Subscript Eight",
		'₉': "Subscript Nine",
		// Vulgar fractions
		'¼': "Vulgar Fraction One Quarter", '½': "Vulgar Fraction One Half",
		'¾': "Vulgar Fraction Three Quarters", '⅓': "Vulgar Fraction One Third",
		'⅔': "Vulgar Fraction Two Thirds", '⅕': "Vulgar Fraction One Fifth",
		'⅖': "Vulgar Fraction Two Fifths", '⅗': "Vulgar Fraction Three Fifths",
		'⅘': "Vulgar Fraction Four Fifths", '⅙': "Vulgar Fraction One Sixth",
		'⅚': "Vulgar Fraction Five Sixths", '⅛': "Vulgar Fraction One Eighth",
		'⅜': "Vulgar Fraction Three Eighths", '⅝': "Vulgar Fraction Five Eighths",
		'⅞': "Vulgar Fraction Seven Eighths",
		// Letterlike symbols
		'№': "Numero Sign", '™': "Trade Mark Sign", '℠': "Service Mark",
		'℃': "Degree Celsius", '℉': "Degree Fahrenheit",
		'ℓ': "Script Small L", 'µ': "Micro Sign",
		// Common symbols
		'©': "Copyright Sign", '®': "Registered Sign", '§': "Section Sign",
		'°': "Degree Sign", '¶': "Pilcrow Sign", '×': "Multiplication Sign",
		'÷': "Division Sign", '±': "Plus-Minus Sign",
		// Currency symbols
		'¢': "Cent Sign", '£': "Pound Sign", '¥': "Yen Sign", '€': "Euro Sign",
		'₹': "Indian Rupee Sign", '₽': "Ruble Sign", '₩': "Won Sign",
		'₦': "Naira Sign", '₺': "Turkish Lira Sign", '₿': "Bitcoin Sign",
		// ASCII symbols
		'$': "Dollar Sign", '&': "Ampersand", '@': "Commercial At",
		'%': "Percent Sign", '+': "Plus Sign",
	}
	if name, ok := names[r]; ok {
		return name
	}
	return fmt.Sprintf("U+%04X", r)
}

func TestGenerateTransliterations(t *testing.T) {
	const path = "references/transliterations.csv"
	rows := buildCSVRows()

	if os.Getenv("GENERATE") == "1" {
		f, err := os.Create(path)
		if err != nil {
			t.Fatalf("create %s: %v", path, err)
		}
		defer f.Close()
		w := csv.NewWriter(f)
		for _, row := range rows {
			if err := w.Write(row); err != nil {
				t.Fatalf("write csv: %v", err)
			}
		}
		w.Flush()
		if err := w.Error(); err != nil {
			t.Fatalf("flush csv: %v", err)
		}
		t.Logf("wrote %d rows to %s", len(rows)-1, path)
		return
	}

	// Verify mode: check the CSV is up to date.
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("%s not found — run: go test -run TestGenerateTransliterations -generate", path)
	}
	defer f.Close()

	r := csv.NewReader(f)
	existing, err := r.ReadAll()
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	if len(existing) != len(rows) {
		t.Fatalf("%s is stale: has %d rows, want %d — run: go test -run TestGenerateTransliterations -generate",
			path, len(existing), len(rows))
	}

	for i, row := range rows {
		for j, cell := range row {
			if i >= len(existing) || j >= len(existing[i]) || existing[i][j] != cell {
				t.Fatalf("%s is stale at row %d col %d — run: go test -run TestGenerateTransliterations -generate",
					path, i, j)
			}
		}
	}
}
