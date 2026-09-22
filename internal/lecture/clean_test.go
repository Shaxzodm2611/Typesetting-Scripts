package lecture

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

var testPDF = []byte("%PDF-1.7\nfixture for signature validation\n%%EOF\n")

func fixture(t *testing.T) string {
	t.Helper()
	dir, err := New(t.TempDir(), "ECE342", "2")
	if err != nil {
		t.Fatal(err)
	}
	put(t, filepath.Join(dir, "lecture.pdf"), testPDF)
	return dir
}
func put(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
}
func unchangedSource(t *testing.T, dir string) {
	t.Helper()
	if b, err := os.ReadFile(filepath.Join(dir, "lecture.tex")); err != nil || !bytes.Contains(b, []byte(`\begin{document}`)) {
		t.Fatal("source changed", err)
	}
}
func snapshot(t *testing.T, dir string) map[string]string {
	t.Helper()
	result := map[string]string{}
	err := filepath.WalkDir(dir, func(p string, e os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		r, _ := filepath.Rel(dir, p)
		if e.Type()&os.ModeSymlink != 0 {
			v, err := os.Readlink(p)
			result[r] = "link:" + v
			return err
		}
		if e.IsDir() {
			result[r] = "directory"
			return nil
		}
		b, err := os.ReadFile(p)
		result[r] = string(b)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}
func TestFinalizeKeepsOnlyUnchangedPDF(t *testing.T) {
	dir := fixture(t)
	if err := os.Mkdir(filepath.Join(dir, "figures"), 0755); err != nil {
		t.Fatal(err)
	}
	put(t, filepath.Join(dir, "figures", "sketch.txt"), []byte("figure"))
	put(t, filepath.Join(dir, "another.pdf"), []byte("draft"))
	f, err := Inspect(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if f.Directory() != dir {
		t.Fatal(f.Directory())
	}
	if err = f.Apply(); err != nil {
		t.Fatal(err)
	}
	e, err := os.ReadDir(dir)
	if err != nil || len(e) != 1 || e[0].Name() != "lecture.pdf" {
		t.Fatal(e, err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "lecture.pdf"))
	if err != nil || !bytes.Equal(b, testPDF) {
		t.Fatal("PDF changed", err)
	}
	if f, err := Inspect(dir); err == nil {
		f.Close()
		t.Fatal("accepted finalized directory")
	}
}
func TestInspectRefusalsPreserveEverything(t *testing.T) {
	cases := map[string]func(*testing.T, string){
		"missing PDF": func(t *testing.T, d string) { os.Remove(filepath.Join(d, "lecture.pdf")) },
		"empty PDF":   func(t *testing.T, d string) { put(t, filepath.Join(d, "lecture.pdf"), nil) },
		"invalid PDF": func(t *testing.T, d string) { put(t, filepath.Join(d, "lecture.pdf"), []byte("not PDF")) },
		"PDF directory": func(t *testing.T, d string) {
			os.Remove(filepath.Join(d, "lecture.pdf"))
			os.Mkdir(filepath.Join(d, "lecture.pdf"), 0755)
		},
		"missing marker": func(t *testing.T, d string) { os.Remove(filepath.Join(d, ".notes.json")) },
		"bad JSON":       func(t *testing.T, d string) { put(t, filepath.Join(d, ".notes.json"), []byte("{")) },
		"oversize marker": func(t *testing.T, d string) {
			put(t, filepath.Join(d, ".notes.json"), []byte(strings.Repeat(" ", 17000)))
		},
		"trailing JSON": func(t *testing.T, d string) {
			b, _ := os.ReadFile(filepath.Join(d, ".notes.json"))
			put(t, filepath.Join(d, ".notes.json"), append(b, []byte("{}")...))
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			d := fixture(t)
			mutate(t, d)
			before := snapshot(t, d)
			f, err := Inspect(d)
			if err == nil {
				f.Close()
				t.Fatal("accepted invalid lecture")
			}
			if !reflect.DeepEqual(before, snapshot(t, d)) {
				t.Fatal("refusal modified files")
			}
		})
	}
	for name, m := range map[string]Metadata{"unknown version": {2, "ECE342", 2, "lecture.pdf"}, "path escape": {1, "ECE342", 2, "../keep.pdf"}, "invalid course": {1, "../ECE", 2, "lecture.pdf"}, "invalid lecture": {1, "ECE342", 0, "lecture.pdf"}} {
		t.Run(name, func(t *testing.T) {
			d := fixture(t)
			b, _ := json.Marshal(m)
			put(t, filepath.Join(d, ".notes.json"), b)
			before := snapshot(t, d)
			f, err := Inspect(d)
			if err == nil {
				f.Close()
				t.Fatal("accepted bad metadata")
			}
			if !reflect.DeepEqual(before, snapshot(t, d)) {
				t.Fatal("modified on refusal")
			}
		})
	}
	t.Run("unknown field", func(t *testing.T) {
		d := fixture(t)
		put(t, filepath.Join(d, ".notes.json"), []byte(`{"version":1,"course":"ECE342","lecture":2,"pdf":"lecture.pdf","extra":true}`))
		if f, e := Inspect(d); e == nil {
			f.Close()
			t.Fatal("accepted unknown field")
		}
		unchangedSource(t, d)
	})
}
func TestInspectLinks(t *testing.T) {
	for _, name := range []string{"lecture.pdf", ".notes.json", "figures"} {
		t.Run(name, func(t *testing.T) {
			d := fixture(t)
			out := t.TempDir()
			put(t, filepath.Join(out, "sentinel"), []byte("keep"))
			target := out
			if name != "figures" {
				b, _ := os.ReadFile(filepath.Join(d, name))
				target = filepath.Join(out, "external")
				put(t, target, b)
				os.Remove(filepath.Join(d, name))
			}
			if err := os.Symlink(target, filepath.Join(d, name)); err != nil {
				t.Skipf("symlink privilege unavailable: %v", err)
			}
			f, err := Inspect(d)
			if err == nil {
				f.Close()
				t.Fatal("accepted linked entry")
			}
			unchangedSource(t, d)
			b, _ := os.ReadFile(filepath.Join(out, "sentinel"))
			if string(b) != "keep" {
				t.Fatal("changed outside file")
			}
		})
	}
	t.Run("lecture path", func(t *testing.T) {
		d := fixture(t)
		link := filepath.Join(t.TempDir(), "linked")
		if err := os.Symlink(d, link); err != nil {
			t.Skipf("symlink privilege unavailable: %v", err)
		}
		if f, e := Inspect(link); e == nil {
			f.Close()
			t.Fatal("followed link")
		}
		unchangedSource(t, d)
	})
	t.Run("ancestor path", func(t *testing.T) {
		d := fixture(t)
		link := filepath.Join(t.TempDir(), "parent")
		if err := os.Symlink(filepath.Dir(d), link); err != nil {
			t.Skipf("symlink privilege unavailable: %v", err)
		}
		if f, e := Inspect(filepath.Join(link, filepath.Base(d))); e == nil {
			f.Close()
			t.Fatal("followed ancestor link")
		}
		unchangedSource(t, d)
	})
}
func TestFinalizeRevalidatesAfterPrompt(t *testing.T) {
	for _, name := range []string{"PDF replaced", "PDF changed", "marker changed", "new link"} {
		t.Run(name, func(t *testing.T) {
			d := fixture(t)
			f, err := Inspect(d)
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			switch name {
			case "PDF replaced":
				os.Rename(filepath.Join(d, "lecture.pdf"), filepath.Join(d, "old.pdf"))
				put(t, filepath.Join(d, "lecture.pdf"), testPDF)
			case "PDF changed":
				put(t, filepath.Join(d, "lecture.pdf"), append(append([]byte{}, testPDF...), []byte("changed")...))
			case "marker changed":
				put(t, filepath.Join(d, ".notes.json"), []byte("{}"))
			case "new link":
				if err := os.Symlink(t.TempDir(), filepath.Join(d, "newlink")); err != nil {
					t.Skipf("symlink privilege unavailable: %v", err)
				}
			}
			before := snapshot(t, d)
			if err := f.Apply(); err == nil {
				t.Fatal("ignored concurrent change")
			}
			if !reflect.DeepEqual(before, snapshot(t, d)) {
				t.Fatal("deleted after changed preconditions")
			}
		})
	}
}
func TestFinalizePartialFailureKeepsMarkerAndPDF(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX directory permissions; Windows locked-file case is separate")
	}
	d := fixture(t)
	sub := filepath.Join(d, "locked")
	if err := os.Mkdir(sub, 0755); err != nil {
		t.Fatal(err)
	}
	put(t, filepath.Join(sub, "keep"), []byte("locked"))
	f, err := Inspect(d)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := os.Chmod(sub, 0555); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(sub, 0755)
	if p, err := os.Create(filepath.Join(sub, "probe")); err == nil {
		p.Close()
		os.Remove(filepath.Join(sub, "probe"))
		t.Skip("running with permission bypass")
	}
	if err = f.Apply(); err == nil {
		t.Fatal("expected deletion failure")
	}
	b, _ := os.ReadFile(filepath.Join(d, "lecture.pdf"))
	if !bytes.Equal(b, testPDF) {
		t.Fatal("lost PDF")
	}
	if _, err := os.Stat(filepath.Join(d, ".notes.json")); err != nil {
		t.Fatal("lost retry marker")
	}
	os.Chmod(sub, 0755)
	if err = f.Apply(); err != nil {
		t.Fatal("retry failed", err)
	}
}
func TestInspectRootRefused(t *testing.T) {
	root := filepath.VolumeName(t.TempDir()) + string(os.PathSeparator)
	if f, err := Inspect(root); err == nil {
		f.Close()
		t.Fatal("accepted filesystem root")
	}
}

func TestInspectRequiresCanonicalFilenames(t *testing.T) {
	for _, name := range []string{"lecture.pdf", ".notes.json"} {
		t.Run(name, func(t *testing.T) {
			d := fixture(t)
			if err := os.Rename(filepath.Join(d, name), filepath.Join(d, strings.ToUpper(name))); err != nil {
				t.Fatal(err)
			}
			before := snapshot(t, d)
			if f, err := Inspect(d); err == nil {
				f.Close()
				t.Fatal("accepted ambiguous filename casing")
			}
			if !reflect.DeepEqual(before, snapshot(t, d)) {
				t.Fatal("changed files")
			}
		})
	}
}
