package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHelpAndErrors(t *testing.T) {
	cases := []struct {
		args []string
		code int
		want string
	}{
		{nil, 0, "notes new"}, {[]string{"--help"}, 0, "notes clean"}, {[]string{"-h"}, 0, "notes new"},
		{[]string{"new", "--help"}, 0, "COURSE"}, {[]string{"clean", "--help"}, 0, "--yes"},
		{[]string{"--version"}, 0, "test-version"},
		{[]string{"unknown"}, 2, "unknown"}, {[]string{"new"}, 2, "COURSE"},
		{[]string{"new", "ECE", "2", "extra"}, 2, "COURSE"},
		{[]string{"clean"}, 2, "DIRECTORY"}, {[]string{"clean", "x", "--force"}, 2, "--force"},
		{[]string{"--version", "extra"}, 2, "unexpected"}, {[]string{"clean", "a", "b"}, 2, "DIRECTORY"},
		{[]string{"new", "--unknown", "2"}, 2, "--unknown"},
	}
	for _, tc := range cases {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			var out, errout bytes.Buffer
			code := Run(tc.args, t.TempDir(), IO{In: strings.NewReader(""), Out: &out, Err: &errout}, "test-version")
			if code != tc.code {
				t.Fatalf("exit=%d, want %d; %s %s", code, tc.code, out.String(), errout.String())
			}
			if !strings.Contains(out.String()+errout.String(), tc.want) {
				t.Fatal(out.String(), errout.String())
			}
		})
	}
}
func TestCLILifecycle(t *testing.T) {
	for _, tc := range []struct {
		name, input                             string
		interactive, yesBefore, yesAfter, clean bool
	}{
		{name: "cancel", input: "n\n", interactive: true},
		{name: "EOF", interactive: true},
		{name: "redirected yes", input: "yes\n"},
		{name: "oversized answer", input: strings.Repeat("y", 5000) + "\n", interactive: true},
		{name: "affirmative", input: " YES \n", interactive: true, clean: true},
		{name: "short affirmative", input: "y\n", interactive: true, clean: true},
		{name: "yes before", yesBefore: true, clean: true},
		{name: "yes after", yesAfter: true, clean: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			base := filepath.Join(t.TempDir(), "My 'Lecture Notes'")
			if err := os.Mkdir(base, 0755); err != nil {
				t.Fatal(err)
			}
			var out, errout bytes.Buffer
			streams := IO{In: strings.NewReader(tc.input), Out: &out, Err: &errout, Interactive: tc.interactive}
			if code := Run([]string{"new", "ECE342", "2"}, base, streams, "test"); code != 0 {
				t.Fatal(code, errout.String())
			}
			dir := filepath.Join(base, "ECE342", "lecture-02")
			if _, err := os.Stat(filepath.Join(dir, "lecture.pdf")); !os.IsNotExist(err) {
				t.Fatal("new generated a PDF")
			}
			pdf := []byte("%PDF-1.7\nfixture\n%%EOF\n")
			if err := os.WriteFile(filepath.Join(dir, "lecture.pdf"), pdf, 0644); err != nil {
				t.Fatal(err)
			}
			args := []string{"clean"}
			if tc.yesBefore {
				args = append(args, "--yes")
			}
			args = append(args, filepath.Join("ECE342", "lecture-02"))
			if tc.yesAfter {
				args = append(args, "--yes")
			}
			code := Run(args, base, streams, "test")
			e, err := os.ReadDir(dir)
			if err != nil {
				t.Fatal(err)
			}
			if tc.clean {
				if code != 0 || len(e) != 1 {
					t.Fatal(code, e, errout.String())
				}
				if !strings.Contains(out.String(), dir) {
					t.Fatal("missing absolute directory")
				}
			} else {
				if len(e) != 4 {
					t.Fatal("deleted on refusal", e)
				}
				if tc.name != "cancel" && code == 0 {
					t.Fatal("failure reported success")
				}
			}
			b, err := os.ReadFile(filepath.Join(dir, "lecture.pdf"))
			if err != nil || !bytes.Equal(b, pdf) {
				t.Fatal("PDF changed", err)
			}
		})
	}
}
func TestCleanMissingPDFFailsWithoutDeletion(t *testing.T) {
	base := t.TempDir()
	var out, errout bytes.Buffer
	streams := IO{In: strings.NewReader("yes\n"), Out: &out, Err: &errout, Interactive: true}
	if Run([]string{"new", "ECE", "1"}, base, streams, "test") != 0 {
		t.Fatal(errout.String())
	}
	if Run([]string{"clean", "ECE/lecture-01", "--yes"}, base, streams, "test") == 0 {
		t.Fatal("accepted missing PDF")
	}
	if _, err := os.Stat(filepath.Join(base, "ECE", "lecture-01", "lecture.tex")); err != nil {
		t.Fatal(err)
	}
}
func TestNewInvalidCourseIsReported(t *testing.T) {
	var out, errout bytes.Buffer
	if code := Run([]string{"new", "../outside", "1"}, t.TempDir(), IO{In: strings.NewReader(""), Out: &out, Err: &errout}, "test"); code == 0 || errout.Len() == 0 {
		t.Fatal("invalid course not reported")
	}
}

func TestCleanLinkedParentTraversalPreservesBothCandidates(t *testing.T) {
	base, outside := t.TempDir(), t.TempDir()
	var out, errout bytes.Buffer
	streams := IO{In: strings.NewReader(""), Out: &out, Err: &errout}
	for _, root := range []string{base, outside} {
		if Run([]string{"new", "ECE342", "2"}, root, streams, "test") != 0 {
			t.Fatal(errout.String())
		}
		if err := os.WriteFile(filepath.Join(root, "ECE342", "lecture-02", "lecture.pdf"), []byte("%PDF-1.7\nfixture\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	child := filepath.Join(outside, "child")
	if err := os.Mkdir(child, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(child, filepath.Join(base, "alias")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	raw := "alias" + string(os.PathSeparator) + ".." + string(os.PathSeparator) + filepath.Join("ECE342", "lecture-02")
	code := Run([]string{"clean", raw, "--yes"}, base, streams, "test")
	for _, root := range []string{base, outside} {
		e, err := os.ReadDir(filepath.Join(root, "ECE342", "lecture-02"))
		if err != nil || len(e) != 4 {
			t.Fatal("changed a candidate directory", root, e, err)
		}
	}
	if code == 0 {
		t.Fatal("accepted ambiguous linked traversal")
	}
}
