package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// --- CLI integration tests (stdin, args, exit codes) ---

// buildBinary builds the sanitize binary once for CLI tests.
func buildBinary(t *testing.T) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "sanitize")
	cmd := exec.Command("go", "build", "-o", binary, ".")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to build binary: %v", err)
	}
	return binary
}

func TestCLIStdin(t *testing.T) {
	binary := buildBinary(t)

	tests := []struct {
		name  string
		stdin string
		want  string
	}{
		{"single line", "Hello, World!\n", "hello-world\n"},
		{"multiple lines", "Hello World\nCafé Résumé\nŁódź\n", "hello-world\ncafe-resume\nlodz\n"},
		{"blank lines skipped", "hello\n\nworld\n", "hello\nworld\n"},
		{"trailing whitespace trimmed", "hello  \n", "hello\n"},
		{"lines producing empty output skipped", "!!!\nhello\n@@@\n", "hello\n"},
		{"mixed real filenames", "Meeting Notes (2024).docx\nIMG_001.jpg\n", "meeting-notes-2024-docx\nimg-001-jpg\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(binary)
			cmd.Stdin = strings.NewReader(tt.stdin)
			out, err := cmd.Output()
			if err != nil {
				t.Fatalf("command failed: %v", err)
			}
			if string(out) != tt.want {
				t.Errorf("stdin %q:\ngot:  %q\nwant: %q", tt.stdin, string(out), tt.want)
			}
		})
	}
}

func TestCLIArgs(t *testing.T) {
	binary := buildBinary(t)

	tests := []struct {
		name string
		args []string
		want string
	}{
		{"single arg", []string{"Hello, World!"}, "hello-world\n"},
		{"multiple args joined", []string{"foo", "bar", "baz"}, "foo-bar-baz\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(binary, tt.args...)
			out, err := cmd.Output()
			if err != nil {
				t.Fatalf("command failed: %v", err)
			}
			if string(out) != tt.want {
				t.Errorf("args %v:\ngot:  %q\nwant: %q", tt.args, string(out), tt.want)
			}
		})
	}
}

func TestCLIEmptyStdin(t *testing.T) {
	binary := buildBinary(t)

	// Empty piped stdin should succeed with no output (like cat < /dev/null)
	cmd := exec.Command(binary)
	cmd.Stdin = strings.NewReader("")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("empty stdin should exit 0, got: %v", err)
	}
	if string(out) != "" {
		t.Errorf("empty stdin should produce no output, got: %q", string(out))
	}
}

func TestCLIHelpExitCode(t *testing.T) {
	binary := buildBinary(t)

	cmd := exec.Command(binary, "--help")
	err := cmd.Run()
	if err != nil {
		t.Errorf("--help should exit 0, got: %v", err)
	}
}

// --- Args[0] "san" symlink tests ---

func TestCLISanSymlink(t *testing.T) {
	binary := buildBinary(t)
	dir := t.TempDir()

	// Create a "san" symlink to the binary
	sanLink := filepath.Join(dir, "san")
	if err := os.Symlink(binary, sanLink); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	// Create a test file
	src := filepath.Join(dir, "Hello World.txt")
	os.WriteFile(src, []byte("test"), 0644)

	// Invoke via "san" symlink — should auto-enable file rename mode
	cmd := exec.Command(sanLink, src)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("san command failed: %v\noutput: %s", err, out)
	}

	dst := filepath.Join(dir, "hello-world.txt")
	if _, err := os.Stat(dst); os.IsNotExist(err) {
		t.Errorf("expected %q to exist after rename via san symlink", dst)
	}
}

func TestCLISanSymlinkMultiple(t *testing.T) {
	binary := buildBinary(t)
	dir := t.TempDir()

	sanLink := filepath.Join(dir, "san")
	os.Symlink(binary, sanLink)

	files := []struct {
		src  string
		want string
	}{
		{"Café.pdf", "cafe.pdf"},
		{"My Document.txt", "my-document.txt"},
	}

	var args []string
	for _, f := range files {
		src := filepath.Join(dir, f.src)
		os.WriteFile(src, []byte("test"), 0644)
		args = append(args, src)
	}

	cmd := exec.Command(sanLink, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("command failed: %v\noutput: %s", err, out)
	}

	for _, f := range files {
		dst := filepath.Join(dir, f.want)
		if _, err := os.Stat(dst); os.IsNotExist(err) {
			t.Errorf("expected %q to exist", dst)
		}
	}
}

func TestCLISanSymlinkNoArgs(t *testing.T) {
	binary := buildBinary(t)
	dir := t.TempDir()

	sanLink := filepath.Join(dir, "san")
	os.Symlink(binary, sanLink)

	// "san" with no args should show usage and exit non-zero
	cmd := exec.Command(sanLink)
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected non-zero exit with no args")
	}
}

// --- File rename mode (-f) CLI tests ---

func TestCLIFileRename(t *testing.T) {
	binary := buildBinary(t)
	dir := t.TempDir()

	tests := []struct {
		name     string
		filename string
		want     string
	}{
		{"simple", "Hello World.txt", "hello-world.txt"},
		{"accented", "Café Résumé.pdf", "cafe-resume.pdf"},
		{"uppercase ext", "Document.PDF", "document.pdf"},
		{"no extension", "README", "readme"},
		{"dotfile", ".gitignore", ".gitignore"},
		{"multiple dots", "my.file.name.tar.gz", "my-file-name-tar.gz"},
		{"already clean", "hello.txt", "hello.txt"},
		{"spaces in ext", "file. t x t", "file.t-x-t"},
		{"polish", "Zażółć gęślą.doc", "zazolc-gesla.doc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create the source file
			src := filepath.Join(dir, tt.filename)
			if err := os.WriteFile(src, []byte("test"), 0644); err != nil {
				t.Fatalf("failed to create test file: %v", err)
			}

			cmd := exec.Command(binary, "-f", src)
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("command failed: %v\noutput: %s", err, out)
			}

			// Check that the destination file exists
			dst := filepath.Join(dir, tt.want)
			if _, err := os.Stat(dst); os.IsNotExist(err) {
				t.Errorf("expected %q to exist after rename", dst)
			}

			// Clean up for next test
			os.Remove(dst)
		})
	}
}

func TestCLIFileRenameNoClobber(t *testing.T) {
	binary := buildBinary(t)
	dir := t.TempDir()

	// Create source and a conflicting target
	src := filepath.Join(dir, "Hello World.txt")
	dst := filepath.Join(dir, "hello-world.txt")
	os.WriteFile(src, []byte("source"), 0644)
	os.WriteFile(dst, []byte("existing"), 0644)

	cmd := exec.Command(binary, "-f", src)
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected non-zero exit when target already exists")
	}

	// Source should still exist (not deleted)
	if _, err := os.Stat(src); os.IsNotExist(err) {
		t.Error("source file should not have been removed")
	}

	// Target should be unchanged
	content, _ := os.ReadFile(dst)
	if string(content) != "existing" {
		t.Error("existing target file should not have been overwritten")
	}
}

func TestCLIFileRenameMultiple(t *testing.T) {
	binary := buildBinary(t)
	dir := t.TempDir()

	files := []struct {
		src  string
		want string
	}{
		{"Hello World.txt", "hello-world.txt"},
		{"Café.pdf", "cafe.pdf"},
		{"README", "readme"},
	}

	var args []string
	args = append(args, "-f")
	for _, f := range files {
		src := filepath.Join(dir, f.src)
		os.WriteFile(src, []byte("test"), 0644)
		args = append(args, src)
	}

	cmd := exec.Command(binary, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("command failed: %v\noutput: %s", err, out)
	}

	for _, f := range files {
		dst := filepath.Join(dir, f.want)
		if _, err := os.Stat(dst); os.IsNotExist(err) {
			t.Errorf("expected %q to exist", dst)
		}
	}
}

func TestCLIFileRenameSkipClean(t *testing.T) {
	binary := buildBinary(t)
	dir := t.TempDir()

	// A file that's already clean should be skipped (no error, no rename)
	src := filepath.Join(dir, "hello.txt")
	os.WriteFile(src, []byte("test"), 0644)

	cmd := exec.Command(binary, "-f", src)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("command failed: %v\noutput: %s", err, out)
	}

	// File should still exist at original path
	if _, err := os.Stat(src); os.IsNotExist(err) {
		t.Error("clean file should still exist at original path")
	}
}

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
		// Non-Latin letters should be trimmed (consistent with replaceNonAlphaNum using unicode.Latin)
		{"non-latin leading", "你hello", "hello"},
		{"non-latin trailing", "hello你", "hello"},
		{"non-latin both ends", "你hello你", "hello"},
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
