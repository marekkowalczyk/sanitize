package main

import (
	"bytes"
	"context"
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

func TestCLIStdinNullDelimited(t *testing.T) {
	binary := buildBinary(t)

	tests := []struct {
		name  string
		stdin string
		want  string
	}{
		{"null separated", "Hello World\x00Café\x00", "hello-world\x00cafe\x00"},
		{"skip empty segments", "hello\x00\x00world\x00", "hello\x00world\x00"},
		{"skip segments that sanitize to empty", "!!!\x00hello\x00@@@\x00", "hello\x00"},
		{"filename with newline", "Hello\nWorld\x00foo\x00", "hello-world\x00foo\x00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(binary, "-0")
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

func TestCLIVersion(t *testing.T) {
	binary := buildBinary(t)

	cmd := exec.Command(binary, "--version")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("--version failed: %v", err)
	}
	output := strings.TrimSpace(string(out))
	if !strings.HasPrefix(output, "sanitize ") {
		t.Errorf("--version output should start with 'sanitize ', got: %q", output)
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

func TestCLISanHelp(t *testing.T) {
	binary := buildBinary(t)
	dir := t.TempDir()

	sanLink := filepath.Join(dir, "san")
	os.Symlink(binary, sanLink)

	// "san --help" should show san-specific usage, not "sanitize"
	cmd := exec.Command(sanLink, "--help")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("san --help failed: %v\noutput: %s", err, out)
	}

	output := string(out)
	if !strings.Contains(output, "Usage: san") {
		t.Errorf("san --help should show 'Usage: san', got:\n%s", output)
	}
	if strings.Contains(output, "Usage: sanitize") {
		t.Errorf("san --help should not show 'Usage: sanitize', got:\n%s", output)
	}
}

func TestCLISanVersion(t *testing.T) {
	binary := buildBinary(t)
	dir := t.TempDir()

	sanLink := filepath.Join(dir, "san")
	os.Symlink(binary, sanLink)

	cmd := exec.Command(sanLink, "--version")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("san --version failed: %v", err)
	}
	output := strings.TrimSpace(string(out))
	if !strings.HasPrefix(output, "san ") {
		t.Errorf("san --version should start with 'san ', got: %q", output)
	}
}

// --- Dry-run mode (-n) CLI tests ---

func TestCLIDryRun(t *testing.T) {
	binary := buildBinary(t)
	dir := t.TempDir()

	src := filepath.Join(dir, "Hello World.txt")
	os.WriteFile(src, []byte("test"), 0644)

	cmd := exec.Command(binary, "-f", "-n", src)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("command failed: %v\noutput: %s", err, out)
	}

	// Source should still exist (not renamed)
	if _, err := os.Stat(src); os.IsNotExist(err) {
		t.Error("source file should still exist in dry-run mode")
	}

	// Destination should NOT exist
	dst := filepath.Join(dir, "hello-world.txt")
	if _, err := os.Stat(dst); err == nil {
		t.Error("destination should not exist in dry-run mode")
	}

	// Output should show what would happen
	output := string(out)
	if !strings.Contains(output, "hello-world.txt") {
		t.Errorf("dry-run output should show the target name, got: %q", output)
	}
}

func TestCLIDryRunSkipsClean(t *testing.T) {
	binary := buildBinary(t)
	dir := t.TempDir()

	src := filepath.Join(dir, "hello.txt")
	os.WriteFile(src, []byte("test"), 0644)

	cmd := exec.Command(binary, "-f", "-n", src)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("command failed: %v\noutput: %s", err, out)
	}

	// No output for already-clean files
	if len(strings.TrimSpace(string(out))) > 0 {
		t.Errorf("dry-run should produce no output for clean files, got: %q", string(out))
	}
}

func TestCLIDryRunCombinedFlags(t *testing.T) {
	binary := buildBinary(t)
	dir := t.TempDir()

	src := filepath.Join(dir, "Hello World.txt")
	os.WriteFile(src, []byte("test"), 0644)

	// Test -fn (combined short flags)
	cmd := exec.Command(binary, "-fn", src)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("-fn failed: %v\noutput: %s", err, out)
	}

	// Source should still exist (dry run)
	if _, err := os.Stat(src); os.IsNotExist(err) {
		t.Error("source file should still exist with -fn")
	}

	if !strings.Contains(string(out), "hello-world.txt") {
		t.Errorf("-fn output should show target, got: %q", string(out))
	}
}

func TestCLIDryRunImpliesFileMode(t *testing.T) {
	binary := buildBinary(t)
	dir := t.TempDir()

	src := filepath.Join(dir, "Hello World.txt")
	os.WriteFile(src, []byte("test"), 0644)

	// -n without -f should still enter file mode (dry-run only makes sense for renames)
	cmd := exec.Command(binary, "-n", src)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("-n without -f failed: %v\noutput: %s", err, out)
	}

	// Source should still exist (dry run)
	if _, err := os.Stat(src); os.IsNotExist(err) {
		t.Error("source file should still exist with -n")
	}

	if !strings.Contains(string(out), "hello-world.txt") {
		t.Errorf("-n should imply file mode and show target, got: %q", string(out))
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

// --- Recursive mode (-r) CLI tests ---

func TestCLIRecursive(t *testing.T) {
	binary := buildBinary(t)
	dir := t.TempDir()

	// Create a directory tree:
	// dir/
	//   Hello World.txt
	//   Sub Dir/
	//     Café.pdf
	//     Deep Folder/
	//       My File.doc

	subDir := filepath.Join(dir, "Sub Dir")
	deepDir := filepath.Join(subDir, "Deep Folder")
	os.MkdirAll(deepDir, 0755)

	os.WriteFile(filepath.Join(dir, "Hello World.txt"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(subDir, "Café.pdf"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(deepDir, "My File.doc"), []byte("test"), 0644)

	cmd := exec.Command(binary, "-r", dir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("command failed: %v\noutput: %s", err, out)
	}

	// Files should be renamed
	expected := []string{
		filepath.Join(dir, "hello-world.txt"),
		filepath.Join(dir, "sub-dir", "cafe.pdf"),
		filepath.Join(dir, "sub-dir", "deep-folder", "my-file.doc"),
	}
	for _, path := range expected {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected %q to exist", path)
		}
	}

	// Old paths should not exist
	gone := []string{
		filepath.Join(dir, "Hello World.txt"),
		filepath.Join(dir, "Sub Dir"),
	}
	for _, path := range gone {
		if _, err := os.Stat(path); err == nil {
			t.Errorf("expected %q to no longer exist", path)
		}
	}
}

func TestCLIRecursiveDryRun(t *testing.T) {
	binary := buildBinary(t)
	dir := t.TempDir()

	subDir := filepath.Join(dir, "Sub Dir")
	os.MkdirAll(subDir, 0755)
	os.WriteFile(filepath.Join(subDir, "Café.pdf"), []byte("test"), 0644)

	cmd := exec.Command(binary, "-rn", dir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("command failed: %v\noutput: %s", err, out)
	}

	// Nothing should be renamed
	if _, err := os.Stat(filepath.Join(subDir, "Café.pdf")); os.IsNotExist(err) {
		t.Error("file should still exist in dry-run mode")
	}

	// Output should mention renames
	output := string(out)
	if !strings.Contains(output, "cafe.pdf") {
		t.Errorf("dry-run output should show planned renames, got: %q", output)
	}
}

func TestCLIRecursiveSkipsClean(t *testing.T) {
	binary := buildBinary(t)
	dir := t.TempDir()

	subDir := filepath.Join(dir, "clean-dir")
	os.MkdirAll(subDir, 0755)
	os.WriteFile(filepath.Join(subDir, "hello.txt"), []byte("test"), 0644)

	cmd := exec.Command(binary, "-r", dir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("command failed: %v\noutput: %s", err, out)
	}

	// Everything should still exist at original paths
	if _, err := os.Stat(filepath.Join(subDir, "hello.txt")); os.IsNotExist(err) {
		t.Error("clean file should still exist")
	}

	// No output for clean tree
	if len(strings.TrimSpace(string(out))) > 0 {
		t.Errorf("should produce no output for clean tree, got: %q", string(out))
	}
}

func TestCLIRecursiveNoClobber(t *testing.T) {
	binary := buildBinary(t)
	dir := t.TempDir()

	// Create two files that would collide
	os.WriteFile(filepath.Join(dir, "Hello World.txt"), []byte("source"), 0644)
	os.WriteFile(filepath.Join(dir, "hello-world.txt"), []byte("existing"), 0644)

	cmd := exec.Command(binary, "-r", dir)
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected non-zero exit when target already exists")
	}

	// Existing file should be unchanged
	content, _ := os.ReadFile(filepath.Join(dir, "hello-world.txt"))
	if string(content) != "existing" {
		t.Error("existing file should not have been overwritten")
	}
}

func TestCLIRecursiveSanSymlink(t *testing.T) {
	binary := buildBinary(t)
	dir := t.TempDir()

	sanLink := filepath.Join(dir, "san")
	os.Symlink(binary, sanLink)

	subDir := filepath.Join(dir, "Test Dir")
	os.MkdirAll(subDir, 0755)
	os.WriteFile(filepath.Join(subDir, "My File.txt"), []byte("test"), 0644)

	// san -r should work (san auto-enables file mode, -r adds recursion)
	cmd := exec.Command(sanLink, "-r", dir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("command failed: %v\noutput: %s", err, out)
	}

	if _, err := os.Stat(filepath.Join(dir, "test-dir", "my-file.txt")); os.IsNotExist(err) {
		t.Error("expected recursive rename via san symlink to work")
	}
}

func TestCLIRecursiveImpliesFileMode(t *testing.T) {
	binary := buildBinary(t)
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "Hello World.txt"), []byte("test"), 0644)

	// -r without -f should work (implies file mode)
	cmd := exec.Command(binary, "-r", dir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("-r without -f should work: %v\noutput: %s", err, out)
	}

	if _, err := os.Stat(filepath.Join(dir, "hello-world.txt")); os.IsNotExist(err) {
		t.Error("expected file to be renamed with -r alone")
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
		{"danish slashed o lower", "Ørsted", "Orsted"},
		{"danish slashed o upper", "ØST", "OST"},
		{"danish ae ligature", "Ærø", "AerO"},
		{"french oe ligature", "œuvre", "oeuvre"},
		{"french oe ligature upper", "Œuvre", "OEuvre"},
		{"croatian barred d lower", "đakovo", "dakovo"},
		{"croatian barred d upper", "Đakovo", "Dakovo"},
		{"maltese barred h lower", "ħal", "hal"},
		{"maltese barred h upper", "Ħal", "Hal"},
		{"turkish dotless i", "Diyarbakır", "Diyarbakir"},
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
		{"danish slashed o", "Ørsted", "orsted"},
		{"danish ae ligature", "Ærø", "aero"},
		{"french oe ligature", "Œuvre complète", "oeuvre-complete"},
		{"croatian barred d", "Đakovo", "dakovo"},
		{"maltese barred h", "Ħal Balzan", "hal-balzan"},
		{"turkish dotless i lower", "Diyarbakır", "diyarbakir"},

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

// --- Unit tests for renameOne with io.Writer ---

func TestRenameOneOutput(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "Hello World.txt")
	os.WriteFile(src, []byte("test"), 0644)

	var buf bytes.Buffer
	code := renameOne(src, false, &buf)

	if code != 0 {
		t.Fatalf("renameOne returned %d, want 0", code)
	}

	dst := filepath.Join(dir, "hello-world.txt")
	if _, err := os.Stat(dst); os.IsNotExist(err) {
		t.Error("expected destination file to exist")
	}

	if !strings.Contains(buf.String(), "hello-world.txt") {
		t.Errorf("output should mention target, got: %q", buf.String())
	}
}

func TestRenameOneDryRunOutput(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "Hello World.txt")
	os.WriteFile(src, []byte("test"), 0644)

	var buf bytes.Buffer
	code := renameOne(src, true, &buf)

	if code != 0 {
		t.Fatalf("renameOne returned %d, want 0", code)
	}

	// Source should still exist
	if _, err := os.Stat(src); os.IsNotExist(err) {
		t.Error("source should still exist in dry-run")
	}

	if !strings.Contains(buf.String(), "hello-world.txt") {
		t.Errorf("dry-run output should mention target, got: %q", buf.String())
	}
}

func TestRenameOneSkipsClean(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "hello.txt")
	os.WriteFile(src, []byte("test"), 0644)

	var buf bytes.Buffer
	code := renameOne(src, false, &buf)

	if code != 0 {
		t.Fatalf("renameOne returned %d, want 0", code)
	}
	if buf.Len() > 0 {
		t.Errorf("should produce no output for clean file, got: %q", buf.String())
	}
}

func TestRenameOneCollision(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "Hello World.txt")
	dst := filepath.Join(dir, "hello-world.txt")
	os.WriteFile(src, []byte("source"), 0644)
	os.WriteFile(dst, []byte("existing"), 0644)

	var buf bytes.Buffer
	code := renameOne(src, false, &buf)

	if code != 1 {
		t.Fatalf("renameOne returned %d, want 1 for collision", code)
	}
	if !strings.Contains(buf.String(), "target already exists") {
		t.Errorf("should report collision, got: %q", buf.String())
	}
}

func TestRenameFilesOutput(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "Hello.txt"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(dir, "World.pdf"), []byte("test"), 0644)

	var buf bytes.Buffer
	code := renameFiles([]string{
		filepath.Join(dir, "Hello.txt"),
		filepath.Join(dir, "World.pdf"),
	}, false, &buf)

	if code != 0 {
		t.Fatalf("renameFiles returned %d, want 0", code)
	}

	output := buf.String()
	if !strings.Contains(output, "hello.txt") || !strings.Contains(output, "world.pdf") {
		t.Errorf("should mention both targets, got: %q", output)
	}
}

func TestRenameRecursiveOutput(t *testing.T) {
	dir := t.TempDir()
	subDir := filepath.Join(dir, "Sub Dir")
	os.MkdirAll(subDir, 0755)
	os.WriteFile(filepath.Join(subDir, "My File.txt"), []byte("test"), 0644)

	var buf bytes.Buffer
	code := renameRecursive(context.Background(), dir, false, &buf)

	if code != 0 {
		t.Fatalf("renameRecursive returned %d, want 0", code)
	}

	if _, err := os.Stat(filepath.Join(dir, "sub-dir", "my-file.txt")); os.IsNotExist(err) {
		t.Error("expected recursive rename to work")
	}

	output := buf.String()
	if !strings.Contains(output, "my-file.txt") || !strings.Contains(output, "sub-dir") {
		t.Errorf("should mention renames, got: %q", output)
	}
}

// --- Benchmarks ---

func BenchmarkRemoveIllFormed(b *testing.B) {
	for b.Loop() {
		removeIllFormed("Hello\x80World café")
	}
}

func BenchmarkToLower(b *testing.B) {
	for b.Loop() {
		toLower("Hello World CAFÉ")
	}
}

func BenchmarkRemoveAccents(b *testing.B) {
	for b.Loop() {
		removeAccents("Zażółć gęślą jaźń")
	}
}

func BenchmarkReplaceNonAlphaNum(b *testing.B) {
	for b.Loop() {
		replaceNonAlphaNum("hello, world! (2024)")
	}
}

func BenchmarkDedupHyp(b *testing.B) {
	for b.Loop() {
		dedupHyp("a--b---c----d")
	}
}

func BenchmarkTrimEnds(b *testing.B) {
	for b.Loop() {
		trimEnds("---hello-world---")
	}
}

func BenchmarkSanitize(b *testing.B) {
	for b.Loop() {
		sanitize("Meeting Notes (2024-03-15) — Zażółć gęślą jaźń!")
	}
}

func BenchmarkSanitizeFilename(b *testing.B) {
	for b.Loop() {
		sanitizeFilename("Meeting Notes (2024-03-15).PDF")
	}
}

func TestRenameRecursiveCancellation(t *testing.T) {
	dir := t.TempDir()

	// Create many files so we can observe partial rename
	for i := 0; i < 10; i++ {
		name := string(rune('A'+i)) + " File.txt"
		os.WriteFile(filepath.Join(dir, name), []byte("test"), 0644)
	}

	// Cancel immediately — should stop before renaming all files
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before starting

	var buf bytes.Buffer
	code := renameRecursive(ctx, dir, false, &buf)

	if code != 1 {
		t.Fatalf("cancelled renameRecursive should return 1, got %d", code)
	}

	if !strings.Contains(buf.String(), "interrupted") {
		t.Errorf("should report interruption, got: %q", buf.String())
	}
}
