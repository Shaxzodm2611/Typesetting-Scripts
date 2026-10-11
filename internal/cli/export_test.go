package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Shaxzodm2611/Typesetting-Scripts/internal/lecture"
)

func TestExportCLI(t *testing.T) {
	base := t.TempDir()
	dir, err := lecture.New(base, "COE718", "3")
	if err != nil {
		t.Fatal(err)
	}
	pdf := []byte("%PDF-1.7\nfixture\n%%EOF\n")
	if err := os.WriteFile(filepath.Join(dir, "lecture.pdf"), pdf, 0644); err != nil {
		t.Fatal(err)
	}
	vault := filepath.Join(t.TempDir(), "Vault with spaces")
	folder := filepath.Join(vault, "School", "Fall '26", "COE718", "Lectures")
	if err := os.MkdirAll(folder, 0755); err != nil {
		t.Fatal(err)
	}
	note := filepath.Join(folder, "Lecture 3 - ARM7.md")
	if err := os.WriteFile(note, []byte("original"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("NOTES_OBSIDIAN_VAULT", vault)
	var out, errout bytes.Buffer
	streams := IO{Out: &out, Err: &errout, In: strings.NewReader("")}
	if code := Run([]string{"export", filepath.Join("COE718", "lecture-03"), "--dry-run"}, base, streams, "test"); code != 0 {
		t.Fatal(code, errout.String())
	}
	if !strings.Contains(out.String(), note) || !strings.Contains(out.String(), "Dry run") {
		t.Fatal(out.String())
	}
	if _, err := os.Stat(note); err != nil {
		t.Fatal("dry run removed note")
	}
	dest := filepath.Join(folder, "Lecture 3 - ARM7.pdf")
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Fatal("dry run copied PDF")
	}
	// Explicit vault takes priority over a stale per-machine environment setting.
	t.Setenv("NOTES_OBSIDIAN_VAULT", filepath.Join(t.TempDir(), "missing"))
	if code := Run([]string{"export", "--vault", vault, "--term", "Fall '26"}, dir, streams, "test"); code != 0 {
		t.Fatal(code, errout.String())
	}
	if _, err := os.Stat(note); !os.IsNotExist(err) {
		t.Fatal("note not removed")
	}
	if data, err := os.ReadFile(dest); err != nil || !bytes.Equal(data, pdf) {
		t.Fatal("PDF not copied", err)
	}
}

func TestExportCLIArgumentErrors(t *testing.T) {
	for _, args := range [][]string{{"--vault"}, {"--term"}, {"--vault", "--dry-run"}, {"--unknown"}, {"a", "b"}, {"--term", "x", "--term", "y"}} {
		var out, errout bytes.Buffer
		if code := Run(append([]string{"export"}, args...), t.TempDir(), IO{Out: &out, Err: &errout}, "test"); code != 2 {
			t.Fatal(args, code, errout.String())
		}
	}
	var out, errout bytes.Buffer
	if code := Run([]string{"export", "--help"}, t.TempDir(), IO{Out: &out, Err: &errout}, "test"); code != 0 || !strings.Contains(out.String(), "--dry-run") {
		t.Fatal(code, out.String(), errout.String())
	}
}
