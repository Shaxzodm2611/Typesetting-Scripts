package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCleanWindowsDriveRelativePathsRefused(t *testing.T) {
	base := t.TempDir()
	var out, errout bytes.Buffer
	streams := IO{In: strings.NewReader(""), Out: &out, Err: &errout}
	if Run([]string{"new", "ECE342", "2"}, base, streams, "test") != 0 {
		t.Fatal(errout.String())
	}
	dir := filepath.Join(base, "ECE342", "lecture-02")
	if err := os.WriteFile(filepath.Join(dir, "lecture.pdf"), []byte("%PDF-1.7\nfixture\n"), 0644); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{`\ECE342\lecture-02`, filepath.VolumeName(base) + `ECE342\lecture-02`} {
		if Run([]string{"clean", path, "--yes"}, base, streams, "test") == 0 {
			t.Fatal("accepted drive-relative path", path)
		}
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) != 4 {
			t.Fatal("changed unrelated directory", entries, err)
		}
	}
}
