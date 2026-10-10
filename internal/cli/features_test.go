package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFeatureHelpAndOptions(t *testing.T) {
	for _, tc := range []struct {
		args []string
		code int
		want string
	}{
		{[]string{"check", "--help"}, 0, "--compile"},
		{[]string{"update-style", "--help"}, 0, "--dry-run"},
		{[]string{"check", "--wat"}, 2, "--wat"},
		{[]string{"check", "a", "b"}, 2, "DIRECTORY"},
		{[]string{"update-style", "--resolve=invalid"}, 2, "keep or replace"},
		{[]string{"update-style", "a", "b"}, 2, "DIRECTORY"},
	} {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			var out, errout bytes.Buffer
			code := Run(tc.args, t.TempDir(), IO{Out: &out, Err: &errout}, "test")
			if code != tc.code || !strings.Contains(out.String()+errout.String(), tc.want) {
				t.Fatal(code, out.String(), errout.String())
			}
		})
	}
}

func TestCheckCLIExitCodesAndDirectoryDefault(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "lecture.tex")
	for _, tc := range []struct {
		source string
		args   []string
		code   int
		want   string
	}{
		{"All complete.", []string{"check"}, 0, "0 error(s), 0 warning(s)"},
		{"% TODO: finish\n", []string{"check"}, 0, "[todo]"},
		{"% TODO: finish\n", []string{"check", "--strict"}, 1, "[todo]"},
		{`\ref{missing}`, []string{"check"}, 1, "[undefined-reference]"},
	} {
		if err := os.WriteFile(source, []byte(tc.source), 0644); err != nil {
			t.Fatal(err)
		}
		var out, errout bytes.Buffer
		code := Run(tc.args, dir, IO{Out: &out, Err: &errout}, "test")
		if code != tc.code || !strings.Contains(out.String(), tc.want) {
			t.Fatal(code, out.String(), errout.String())
		}
	}
}

func TestStyleCLIWholeCourseConflictPreflight(t *testing.T) {
	base := t.TempDir()
	old, err := os.ReadFile(filepath.Join("..", "lecture", "assets", "styles", "local-v0.1.7.sty"))
	if err != nil {
		t.Fatal(err)
	}
	var out, errout bytes.Buffer
	streams := IO{Out: &out, Err: &errout}
	for _, number := range []string{"1", "2"} {
		if code := Run([]string{"new", "COURSE", number}, base, streams, "test"); code != 0 {
			t.Fatal(errout.String())
		}
		if err := os.WriteFile(filepath.Join(base, "COURSE", "lecture-0"+number, "lecturenotes.sty"), old, 0644); err != nil {
			t.Fatal(err)
		}
	}
	out.Reset()
	errout.Reset()
	if code := Run([]string{"update-style", "COURSE", "--dry-run"}, base, streams, "test"); code != 0 {
		t.Fatal(errout.String())
	}
	if !strings.Contains(out.String(), "2 style(s) would update") {
		t.Fatal(out.String())
	}
	first := filepath.Join(base, "COURSE", "lecture-01", "lecturenotes.sty")
	before, _ := os.ReadFile(first)
	if !bytes.Equal(before, old) {
		t.Fatal("dry run wrote a style")
	}
	second := filepath.Join(base, "COURSE", "lecture-02", "lecturenotes.sty")
	conflicting := strings.Replace(string(old), "[color=NoteInk,#1]", "[color=red,#1]", 1)
	if err := os.WriteFile(second, []byte(conflicting), 0644); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	errout.Reset()
	if code := Run([]string{"update-style", "COURSE"}, base, streams, "test"); code != 1 {
		t.Fatal(code, out.String(), errout.String())
	}
	before, _ = os.ReadFile(first)
	if !bytes.Equal(before, old) {
		t.Fatal("updated first style before detecting second conflict")
	}
	if _, err := os.Stat(first + ".bak"); !os.IsNotExist(err) {
		t.Fatal("created a backup during preflight")
	}
	out.Reset()
	errout.Reset()
	if code := Run([]string{"update-style", "COURSE", "--resolve=replace"}, base, streams, "test"); code != 0 {
		t.Fatal(code, out.String(), errout.String())
	}
	if !strings.Contains(out.String(), "2 style(s) updated; 2 backup(s) created") {
		t.Fatal(out.String())
	}
	for _, path := range []string{first, second} {
		data, _ := os.ReadFile(path)
		if !bytes.Contains(data, []byte("v0.2.0")) {
			t.Fatal("not updated", path)
		}
	}
}
