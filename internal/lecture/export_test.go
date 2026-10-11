package lecture

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func exportFixture(t *testing.T, note string) (string, string, string) {
	t.Helper()
	base := t.TempDir()
	dir, err := New(base, "COE718", "3")
	if err != nil {
		t.Fatal(err)
	}
	put(t, filepath.Join(dir, "lecture.pdf"), testPDF)
	vault := filepath.Join(t.TempDir(), "Personal Obsidian Vault")
	folder := filepath.Join(vault, "School", "Fall '26", "COE718", "Lectures")
	if err := os.MkdirAll(folder, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(vault, ".obsidian"), 0755); err != nil {
		t.Fatal(err)
	}
	if note != "" {
		put(t, filepath.Join(folder, note), []byte("original Markdown"))
	}
	return dir, vault, folder
}

func TestExportCopiesBeforeRemovingMatchingMarkdown(t *testing.T) {
	dir, vault, folder := exportFixture(t, "Lecture 3 - ARM7 Programming.md")
	for _, name := range []string{"Lecture 13 - Keep.md", "Lecture 30.md", "Lecture 4.md", "unrelated.md"} {
		put(t, filepath.Join(folder, name), []byte("keep"))
	}
	plan, err := PlanExport(dir, ExportOptions{Vault: vault})
	if err != nil {
		t.Fatal(err)
	}
	defer plan.Close()
	if filepath.Base(plan.Destination) != "Lecture 3 - ARM7 Programming.pdf" {
		t.Fatal(plan.Destination)
	}
	if _, err := os.Stat(plan.Markdown); err != nil {
		t.Fatal("planning changed Markdown", err)
	}
	if _, err := os.Stat(plan.Destination); !os.IsNotExist(err) {
		t.Fatal("planning created PDF", err)
	}
	if err := plan.Apply(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(plan.Markdown); !os.IsNotExist(err) {
		t.Fatal("Markdown remains", err)
	}
	for _, path := range []string{plan.Destination, filepath.Join(dir, "lecture.pdf")} {
		data, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(data, testPDF) {
			t.Fatal(path, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "lecture.tex")); err != nil {
		t.Fatal("source removed", err)
	}
	for _, name := range []string{"Lecture 13 - Keep.md", "Lecture 30.md", "Lecture 4.md", "unrelated.md"} {
		if data, err := os.ReadFile(filepath.Join(folder, name)); err != nil || string(data) != "keep" {
			t.Fatal("unrelated note changed", name, err)
		}
	}
	// Updating the PDF after Markdown removal retains its previous title.
	repeat, err := PlanExport(dir, ExportOptions{Vault: vault})
	if err != nil {
		t.Fatal(err)
	}
	defer repeat.Close()
	if repeat.Markdown != "" || repeat.Destination != plan.Destination {
		t.Fatal(repeat.Destination, repeat.Markdown)
	}
	if err := repeat.Apply(); err != nil {
		t.Fatal("repeat export", err)
	}
}

func TestExportWithoutMarkdownAndAfterClean(t *testing.T) {
	dir, vault, folder := exportFixture(t, "")
	final, err := Inspect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := final.Apply(); err != nil {
		t.Fatal(err)
	}
	final.Close()
	plan, err := PlanExport(dir, ExportOptions{Vault: vault})
	if err != nil {
		t.Fatal(err)
	}
	defer plan.Close()
	if plan.Destination != filepath.Join(folder, "Lecture 3.pdf") || plan.Markdown != "" {
		t.Fatal(plan.Destination)
	}
	if err := plan.Apply(); err != nil {
		t.Fatal(err)
	}
}

func TestExportRefusesAmbiguousAndInvalidFiles(t *testing.T) {
	for _, kind := range []string{"duplicate Markdown", "duplicate PDF", "conflicting PDF", "invalid PDF", "marker mismatch", "invalid term"} {
		t.Run(kind, func(t *testing.T) {
			dir, vault, folder := exportFixture(t, "Lecture 3 - Original.md")
			options := ExportOptions{Vault: vault}
			switch kind {
			case "duplicate Markdown":
				put(t, filepath.Join(folder, "lecture-03 Other.md"), []byte("keep"))
			case "duplicate PDF":
				put(t, filepath.Join(folder, "Lecture 3.pdf"), testPDF)
				put(t, filepath.Join(folder, "Lecture 3 - Other.pdf"), testPDF)
			case "conflicting PDF":
				put(t, filepath.Join(folder, "Lecture 3 - Other.pdf"), testPDF)
			case "invalid PDF":
				put(t, filepath.Join(dir, "lecture.pdf"), []byte("not a PDF"))
			case "marker mismatch":
				put(t, filepath.Join(dir, ".notes.json"), []byte(`{"version":1,"course":"COE718","lecture":4,"pdf":"lecture.pdf"}`))
			case "invalid term":
				options.Term = "../Fall '26"
			}
			if plan, err := PlanExport(dir, options); err == nil {
				plan.Close()
				t.Fatal("accepted", kind)
			}
			if data, err := os.ReadFile(filepath.Join(folder, "Lecture 3 - Original.md")); err != nil || string(data) != "original Markdown" {
				t.Fatal("Markdown changed", err)
			}
		})
	}
}

func TestExportDetectsChangesBeforeCommit(t *testing.T) {
	for _, kind := range []string{"source", "Markdown", "existing PDF", "new PDF", "copy failure"} {
		t.Run(kind, func(t *testing.T) {
			dir, vault, folder := exportFixture(t, "Lecture 3 - Original.md")
			pdf := filepath.Join(folder, "Lecture 3 - Original.pdf")
			if kind == "existing PDF" {
				put(t, pdf, []byte("previous PDF"))
			}
			plan, err := PlanExport(dir, ExportOptions{Vault: vault})
			if err != nil {
				t.Fatal(err)
			}
			defer plan.Close()
			switch kind {
			case "source":
				put(t, filepath.Join(dir, "lecture.pdf"), []byte("%PDF-1.7\nchanged"))
			case "Markdown":
				put(t, plan.Markdown, []byte("new Markdown"))
			case "existing PDF", "new PDF":
				put(t, pdf, []byte("new PDF"))
			case "copy failure":
				plan.target.Close()
			}
			if err := plan.Apply(); err == nil {
				t.Fatal("accepted a changed or unwritable export")
			}
			if _, err := os.Stat(plan.Markdown); err != nil {
				t.Fatal("removed Markdown on failure", err)
			}
			if kind == "existing PDF" || kind == "new PDF" {
				if data, _ := os.ReadFile(pdf); string(data) != "new PDF" {
					t.Fatal("overwrote changed PDF")
				}
			} else if _, err := os.Stat(pdf); !os.IsNotExist(err) {
				t.Fatal("created destination on failure", err)
			}
			entries, _ := os.ReadDir(folder)
			for _, e := range entries {
				if strings.HasPrefix(e.Name(), ".notes-export-") {
					t.Fatal("temporary copy remains")
				}
			}
		})
	}
}

func TestExportDiscoveryAcrossHomeLayouts(t *testing.T) {
	for _, layout := range []string{filepath.Join("OneDrive", "Documents", "Personal Obsidian Vault"), filepath.Join("Documents", "Personal Obsidian Vault"), filepath.Join("Sync", "school", "notes", "My Vault")} {
		t.Run(layout, func(t *testing.T) {
			dir, _, _ := exportFixture(t, "")
			home := t.TempDir()
			vault := filepath.Join(home, layout)
			folder := filepath.Join(vault, "School", "Fall '26", "COE718", "Lectures")
			if err := os.MkdirAll(folder, 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(filepath.Join(vault, ".obsidian"), 0755); err != nil {
				t.Fatal(err)
			}
			plan, err := PlanExport(dir, ExportOptions{Home: home, ConfigDir: t.TempDir()})
			if err != nil {
				t.Fatal(err)
			}
			defer plan.Close()
			if filepath.Dir(plan.Destination) != folder {
				t.Fatal(plan.Destination)
			}
		})
	}
}

func TestExportDiscoveryUsesObsidianRegistry(t *testing.T) {
	dir, vault, folder := exportFixture(t, "")
	config := t.TempDir()
	if err := os.Mkdir(filepath.Join(config, "obsidian"), 0755); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(map[string]any{"vaults": map[string]any{"id": map[string]string{"path": vault}}})
	if err != nil {
		t.Fatal(err)
	}
	put(t, filepath.Join(config, "obsidian", "obsidian.json"), raw)
	plan, err := PlanExport(dir, ExportOptions{Home: t.TempDir(), ConfigDir: config})
	if err != nil {
		t.Fatal(err)
	}
	defer plan.Close()
	if filepath.Dir(plan.Destination) != folder {
		t.Fatal(plan.Destination)
	}
}

func TestExportSemesterAndVaultAmbiguity(t *testing.T) {
	dir, vault, folder := exportFixture(t, "")
	second := filepath.Join(vault, "School", "Winter '27", "COE718", "Lectures")
	if err := os.MkdirAll(second, 0755); err != nil {
		t.Fatal(err)
	}
	if plan, err := PlanExport(dir, ExportOptions{Vault: vault}); err == nil {
		plan.Close()
		t.Fatal("selected arbitrary semester")
	}
	plan, err := PlanExport(dir, ExportOptions{Vault: vault, Term: "fall '26"})
	if err != nil {
		t.Fatal(err)
	}
	plan.Close()
	if filepath.Dir(plan.Destination) != folder {
		t.Fatal(plan.Destination)
	}
	put(t, filepath.Join(second, "Lecture 3.md"), []byte("matching"))
	plan, err = PlanExport(dir, ExportOptions{Vault: vault})
	if err != nil {
		t.Fatal(err)
	}
	plan.Close()
	if filepath.Dir(plan.Destination) != second {
		t.Fatal("did not prefer matching note")
	}
	put(t, filepath.Join(folder, "Lecture 3.md"), []byte("matching"))
	if plan, err := PlanExport(dir, ExportOptions{Vault: vault}); err == nil {
		plan.Close()
		t.Fatal("selected ambiguous matching notes")
	}
}

func TestExportRefusesLinkedFilesAndVaultPaths(t *testing.T) {
	dir, vault, folder := exportFixture(t, "")
	outside := filepath.Join(t.TempDir(), "note.md")
	put(t, outside, []byte("keep"))
	if err := os.Symlink(outside, filepath.Join(folder, "Lecture 3.md")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if plan, err := PlanExport(dir, ExportOptions{Vault: vault}); err == nil {
		plan.Close()
		t.Fatal("accepted linked note")
	}
	if data, _ := os.ReadFile(outside); string(data) != "keep" {
		t.Fatal("changed linked target")
	}
}
