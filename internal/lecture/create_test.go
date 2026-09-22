package lecture

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewPortableBlankLecture(t *testing.T) {
	base := filepath.Join(t.TempDir(), "My Notes")
	if err := os.Mkdir(base, 0755); err != nil {
		t.Fatal(err)
	}
	dir, err := New(base, "ECE_342", "2")
	if err != nil {
		t.Fatal(err)
	}
	if dir != filepath.Join(base, "ECE_342", "lecture-02") {
		t.Fatal(dir)
	}
	data, err := os.ReadFile(filepath.Join(dir, "lecture.tex"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `\lectureheader{ECE\_342}{2}`) {
		t.Fatal(string(data))
	}
	if !strings.Contains(string(data), `\null`) || strings.Contains(string(data), `\notesection`) {
		t.Fatal("not a blank page")
	}
	if _, err := New(base, "ECE_342", "2"); err == nil {
		t.Fatal("overwrote lecture")
	}
	after, err := os.ReadFile(filepath.Join(dir, "lecture.tex"))
	if err != nil || !bytes.Equal(data, after) {
		t.Fatal("changed source", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 3 {
		t.Fatal(entries, err)
	}
	var m Metadata
	raw, err := os.ReadFile(filepath.Join(dir, ".notes.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if m.Version != 1 || m.Course != "ECE_342" || m.Lecture != 2 || m.PDF != "lecture.pdf" {
		t.Fatal(m)
	}
	sty, err := os.ReadFile(filepath.Join(dir, "lecturenotes.sty"))
	if err != nil {
		t.Fatal(err)
	}
	approved, err := os.ReadFile("../../examples/rc-filter/lecturenotes.sty")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(sty, approved) {
		t.Fatal("generated formatting differs from approved sample")
	}
}
func TestNewInvalidInputs(t *testing.T) {
	for _, c := range []string{"../ECE", "ECE/342", `ECE\342`, "", ".", "CON", "nul", "COM1", "LPT9", "A:B", "NUL.txt", "-ECE"} {
		t.Run("course="+c, func(t *testing.T) {
			base := t.TempDir()
			if _, err := New(base, c, "2"); err == nil {
				t.Fatal("accepted invalid course")
			}
			e, _ := os.ReadDir(base)
			if len(e) != 0 {
				t.Fatal("wrote invalid lecture")
			}
		})
	}
	for _, n := range []string{"0", "-2", "2.5", "+2", "999999999999999999999999999", "", " 2"} {
		t.Run("number="+n, func(t *testing.T) {
			base := t.TempDir()
			if _, err := New(base, "ECE", n); err == nil {
				t.Fatal("accepted invalid number")
			}
		})
	}
}
func TestNewValidInputs(t *testing.T) {
	for _, c := range []string{"ECE-342", "ECE_342", "ece342", "123", "Console"} {
		dir, err := New(t.TempDir(), c, "100")
		if err != nil {
			t.Fatal(c, err)
		}
		if filepath.Base(dir) != "lecture-100" {
			t.Fatal(dir)
		}
	}
}
func TestNewConflicts(t *testing.T) {
	t.Run("course file", func(t *testing.T) {
		base := t.TempDir()
		p := filepath.Join(base, "ECE")
		os.WriteFile(p, []byte("keep"), 0644)
		if _, err := New(base, "ECE", "1"); err == nil {
			t.Fatal("accepted file")
		}
		b, _ := os.ReadFile(p)
		if string(b) != "keep" {
			t.Fatal("changed file")
		}
	})
	t.Run("empty lecture", func(t *testing.T) {
		base := t.TempDir()
		p := filepath.Join(base, "ECE", "lecture-01")
		os.MkdirAll(p, 0755)
		if _, err := New(base, "ECE", "1"); err == nil {
			t.Fatal("overwrote existing directory")
		}
	})
}
func TestNewLinkedCourse(t *testing.T) {
	base, out := t.TempDir(), t.TempDir()
	if err := os.Symlink(out, filepath.Join(base, "ECE")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := New(base, "ECE", "1"); err == nil {
		t.Fatal("followed linked course")
	}
	entries, _ := os.ReadDir(out)
	if len(entries) != 0 {
		t.Fatal("wrote outside base")
	}
}
func TestNewCaseCollision(t *testing.T) {
	base := t.TempDir()
	dir, err := New(base, "ECE342", "1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := New(base, "ece342", "1"); err == nil {
		t.Fatal("case collision should be refused for portability")
	}
	if _, err := os.Stat(dir); err != nil {
		t.Fatal(err)
	}
}
