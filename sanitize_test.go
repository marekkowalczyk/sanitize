package main

import "testing"

// --- Unit tests for individual pipeline stages ---

func TestRemoveIllFormed(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"valid ascii", "hello", "hello"},
		{"valid utf8", "café", "café"},
		{"empty", "", ""},
		{"ill-formed bytes", "hello\x80world", "hello\uFFFDworld"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := removeIllFormed(tt.input)
			if got != tt.want {
				t.Errorf("removeIllFormed(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestToLower(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"all upper", "HELLO", "hello"},
		{"mixed case", "HeLLo WoRLd", "hello world"},
		{"already lower", "hello", "hello"},
		{"digits unchanged", "ABC123", "abc123"},
		{"empty", "", ""},
		{"accented upper", "CAFÉ", "café"},
		{"german sharp s", "straße", "straße"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toLower(tt.input)
			if got != tt.want {
				t.Errorf("toLower(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestRemoveAccents(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"no accents", "hello", "hello"},
		{"empty", "", ""},
		{"acute accent", "café", "cafe"},
		{"grave accent", "à la carte", "a la carte"},
		{"circumflex", "crêpe", "crepe"},
		{"tilde", "señor", "senor"},
		{"umlaut", "über", "uber"},
		{"cedilla", "façade", "facade"},
		{"polish l-stroke lower", "łódź", "lodz"},
		{"polish l-stroke upper", "Łódź", "Lodz"},
		{"polish full set", "ąćęłńóśźż", "acelnoszz"},
		{"czech caron", "háček", "hacek"},
		{"scandinavian ring", "Åland", "Aland"},
		{"multiple accents in one word", "résumé", "resume"},
		{"digits unchanged", "café123", "cafe123"},
		{"mixed accented and plain", "hello café world", "hello cafe world"},
		{"vietnamese tone marks", "Hà Nội", "Ha Noi"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := removeAccents(tt.input)
			if got != tt.want {
				t.Errorf("removeAccents(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestReplaceNonAlphaNum(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain ascii", "hello", "hello"},
		{"empty", "", ""},
		{"space", "hello world", "hello-world"},
		{"multiple spaces", "a  b  c", "a--b--c"},
		{"punctuation", "hello, world!", "hello--world-"},
		{"special chars", "a@b#c$d", "a-b-c-d"},
		{"tabs", "a\tb", "a-b"},
		{"newline", "a\nb", "a-b"},
		{"digits kept", "abc123def", "abc123def"},
		{"latin accented kept", "café", "café"},
		{"non-latin replaced", "hello你好", "hello--"},
		{"cyrillic replaced", "helloмир", "hello---"},
		{"arabic replaced", "helloعالم", "hello----"},
		{"emoji replaced", "hello😀world", "hello-world"},
		{"parentheses", "hello(world)", "hello-world-"},
		{"brackets", "hello[world]", "hello-world-"},
		{"slashes", "path/to/file", "path-to-file"},
		{"backslash", "path\\to\\file", "path-to-file"},
		{"dot", "file.txt", "file-txt"},
		{"hyphen preserved", "already-hyphenated", "already-hyphenated"},
		{"underscore replaced", "under_score", "under-score"},
		{"ampersand", "rock&roll", "rock-roll"},
		{"equals", "a=b", "a-b"},
		{"plus", "a+b", "a-b"},
		{"pipe", "a|b", "a-b"},
		{"colon", "a:b", "a-b"},
		{"semicolon", "a;b", "a-b"},
		{"quotes", `"hello"`, "-hello-"},
		{"single quotes", "'hello'", "-hello-"},
		{"angle brackets", "<hello>", "-hello-"},
		{"curly braces", "{hello}", "-hello-"},
		{"tilde", "~hello", "-hello"},
		{"backtick", "`hello`", "-hello-"},
		{"at sign", "user@host", "user-host"},
		{"percent", "100%done", "100-done"},
		{"caret", "a^b", "a-b"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := replaceNonAlphaNum(tt.input)
			if got != tt.want {
				t.Errorf("replaceNonAlphaNum(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestDedupHyp(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"no hyphens", "hello", "hello"},
		{"empty", "", ""},
		{"single hyphen", "a-b", "a-b"},
		{"double hyphen", "a--b", "a-b"},
		{"triple hyphen", "a---b", "a-b"},
		{"many hyphens", "a----------b", "a-b"},
		{"multiple groups", "a--b--c", "a-b-c"},
		{"leading hyphens", "--hello", "-hello"},
		{"trailing hyphens", "hello--", "hello-"},
		{"only hyphens", "-----", "-"},
		{"mixed content", "a--b-c--d", "a-b-c-d"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dedupHyp(tt.input)
			if got != tt.want {
				t.Errorf("dedupHyp(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestTrimEnds(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"no trimming needed", "hello", "hello"},
		{"empty", "", ""},
		{"leading hyphens", "-hello", "hello"},
		{"trailing hyphens", "hello-", "hello"},
		{"both ends", "-hello-", "hello"},
		{"leading spaces", "  hello", "hello"},
		{"trailing spaces", "hello  ", "hello"},
		{"leading punctuation", "...hello", "hello"},
		{"trailing punctuation", "hello!!!", "hello"},
		{"mixed leading junk", "---...hello", "hello"},
		{"preserves inner hyphens", "-a-b-c-", "a-b-c"},
		{"preserves inner punctuation", "-a.b.c-", "a.b.c"},
		{"digits at ends preserved", "123abc", "123abc"},
		{"digits at ends preserved 2", "abc123", "abc123"},
		{"only punctuation", "---", ""},
		{"single letter", "-a-", "a"},
		{"leading digit", "-1abc", "1abc"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := trimEnds(tt.input)
			if got != tt.want {
				t.Errorf("trimEnds(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// --- Integration tests for the full pipeline ---

func TestSanitize(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		// Basic
		{"simple lowercase", "Hello World", "hello-world"},
		{"already clean", "hello", "hello"},
		{"single word uppercase", "HELLO", "hello"},
		{"digits only", "12345", "12345"},
		{"mixed alpha digits", "abc123def", "abc123def"},

		// Accents and diacritics
		{"french", "Crème Brûlée", "creme-brulee"},
		{"spanish", "El Niño", "el-nino"},
		{"german", "Straße nach München", "strasse-nach-munchen"},
		{"polish phrase", "Zażółć gęślą jaźń", "zazolc-gesla-jazn"},
		{"polish city", "Łódź", "lodz"},
		{"czech", "Příliš žluťoučký kůň", "prilis-zlutoucky-kun"},
		{"scandinavian", "Ångström", "angstrom"},
		{"portuguese", "São Paulo", "sao-paulo"},
		{"turkish dotless i", "İstanbul", "istanbul"},
		{"romanian", "României", "romaniei"},

		// Non-Latin scripts (should be stripped entirely)
		{"chinese", "你好世界", ""},
		{"japanese", "東京タワー", ""},
		{"korean", "서울", ""},
		{"arabic", "مرحبا", ""},
		{"cyrillic", "Москва", ""},
		{"mixed latin and chinese", "Hello你好World", "hello-world"},
		{"mixed latin and cyrillic", "Helloмир", "hello"},
		{"emoji only", "😀🎉🚀", ""},
		{"text with emoji", "Hello 😀 World", "hello-world"},

		// Whitespace and separators
		{"multiple spaces", "hello   world", "hello-world"},
		{"tabs", "hello\tworld", "hello-world"},
		{"newlines", "hello\nworld", "hello-world"},
		{"mixed whitespace", "hello \t\n world", "hello-world"},

		// Punctuation and special characters
		{"comma separated", "one, two, three", "one-two-three"},
		{"period separated", "file.name.here", "file-name-here"},
		{"exclamation", "Hello World!", "hello-world"},
		{"question mark", "What is this?", "what-is-this"},
		{"parentheses", "Hello (World)", "hello-world"},
		{"brackets", "Hello [World]", "hello-world"},
		{"curly braces", "Hello {World}", "hello-world"},
		{"ampersand", "Rock & Roll", "rock-roll"},
		{"at sign", "user@domain", "user-domain"},
		{"hash", "section#anchor", "section-anchor"},
		{"dollar", "price$100", "price-100"},
		{"percent", "100% done", "100-done"},
		{"plus", "a + b", "a-b"},
		{"equals", "a = b", "a-b"},
		{"pipe", "a | b", "a-b"},
		{"colon", "key: value", "key-value"},
		{"semicolon", "a; b", "a-b"},
		{"slash", "path/to/file", "path-to-file"},
		{"backslash", "path\\to\\file", "path-to-file"},
		{"quotes", `"hello world"`, "hello-world"},
		{"single quotes", "'hello world'", "hello-world"},
		{"backticks", "`hello`", "hello"},
		{"tilde", "~user", "user"},
		{"caret", "a^b", "a-b"},

		// Hyphen deduplication
		{"adjacent special chars", "a@#$b", "a-b"},
		{"run of punctuation", "hello!!!world", "hello-world"},
		{"mixed separators", "a - b - c", "a-b-c"},
		{"existing hyphens preserved", "already-hyphenated", "already-hyphenated"},

		// Trimming
		{"leading spaces", "  hello", "hello"},
		{"trailing spaces", "hello  ", "hello"},
		{"leading punctuation", "---hello", "hello"},
		{"trailing punctuation", "hello---", "hello"},
		{"both ends messy", "  ---Hello World!---  ", "hello-world"},
		{"wrapped in parens", "(hello)", "hello"},
		{"wrapped in brackets", "[hello]", "hello"},

		// Edge cases
		{"empty string", "", ""},
		{"single char", "A", "a"},
		{"single digit", "7", "7"},
		{"only spaces", "     ", ""},
		{"only punctuation", "!@#$%^&*()", ""},
		{"only hyphens", "-----", ""},
		{"single hyphen", "-", ""},
		{"very long input", "This Is A Very Long Title That Someone Might Use For A Document Name In The Year 2024",
			"this-is-a-very-long-title-that-someone-might-use-for-a-document-name-in-the-year-2024"},

		// Real-world filename scenarios
		{"document name", "Meeting Notes (2024-03-15).docx", "meeting-notes-2024-03-15-docx"},
		{"photo name", "IMG_20240315_143022.jpg", "img-20240315-143022-jpg"},
		{"download artifact", "report [final] (v2).pdf", "report-final-v2-pdf"},
		{"mac screenshot", "Screenshot 2024-03-15 at 14.30.22.png", "screenshot-2024-03-15-at-14-30-22-png"},
		{"url slug", "How to Cook Pasta — A Guide!", "how-to-cook-pasta-a-guide"},
		{"email subject", "Re: Fwd: Important!!! (Action Required)", "re-fwd-important-action-required"},
		{"code identifier", "myFunction_name.test", "myfunction-name-test"},
		{"version string", "v2.1.0-beta.1", "v2-1-0-beta-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitize(tt.input)
			if got != tt.want {
				t.Errorf("sanitize(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// --- Pipeline ordering tests ---
// Verify that the pipeline handles ordering-sensitive cases correctly.

func TestPipelineOrdering(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		// Lowering must happen before accent removal (É -> é -> e, not É -> E left as-is)
		{"upper accented", "CAFÉ", "cafe"},
		{"upper l-stroke", "ŁÓDŹ", "lodz"},

		// Accent removal must happen before non-alphanumeric replacement
		// (é -> e, not é -> kept as Latin then left accented)
		{"accent then replace", "naïve", "naive"},

		// Non-alpha replacement must happen before dedup
		// (multiple special chars in a row -> multiple hyphens -> single hyphen)
		{"multi special to single hyphen", "a!!b", "a-b"},

		// Dedup must happen before trim
		// (---hello--- -> -hello- -> hello)
		{"dedup then trim", "---hello---", "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitize(tt.input)
			if got != tt.want {
				t.Errorf("sanitize(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// --- Idempotency test ---
// Running sanitize twice should produce the same result as once.

func TestIdempotency(t *testing.T) {
	inputs := []string{
		"Hello, World!",
		"Zażółć gęślą jaźń",
		"  ---test---  ",
		"café résumé",
		"hello",
		"abc123",
		"",
		"!!!",
		"Hello你好World",
		"Meeting Notes (2024-03-15).docx",
	}

	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			once := sanitize(input)
			twice := sanitize(once)
			if once != twice {
				t.Errorf("not idempotent: sanitize(%q) = %q, sanitize(%q) = %q", input, once, once, twice)
			}
		})
	}
}
