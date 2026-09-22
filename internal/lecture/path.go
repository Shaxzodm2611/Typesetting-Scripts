package lecture

import (
	"fmt"
	"os"
	"path/filepath"
)

// rejectLinkedPath inspects every existing component, not just the final name.
func rejectLinkedPath(path string) error {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	for p := absolute; ; p = filepath.Dir(p) {
		info, err := os.Lstat(p)
		if err != nil {
			return fmt.Errorf("inspect path %q: %w", p, err)
		}
		if linked(info) {
			return fmt.Errorf("linked or reparse-point path is not supported: %s", p)
		}
		if filepath.Dir(p) == p {
			break
		}
	}
	return nil
}

func rootEntries(root *os.Root) ([]os.DirEntry, error) {
	f, err := root.Open(".")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return f.ReadDir(-1)
}
