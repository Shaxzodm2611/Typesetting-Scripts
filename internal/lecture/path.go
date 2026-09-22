package lecture

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// rejectLinkedPath inspects every existing component, not just the final name.
func rejectLinkedPath(path string) error {
	// Do not call Clean, Join or Abs until every supplied component has been
	// inspected: link/.. can resolve somewhere different from its lexical form.
	if !filepath.IsAbs(path) {
		if filepath.VolumeName(path) != "" || (len(path) > 0 && os.IsPathSeparator(path[0])) {
			return fmt.Errorf("use a fully qualified or ordinary relative path: %q", path)
		}
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		path = cwd + string(os.PathSeparator) + path
	}
	volume := filepath.VolumeName(path)
	current := volume + string(os.PathSeparator)
	parts := strings.FieldsFunc(path[len(volume):], func(c rune) bool { return c == '/' || c == rune(os.PathSeparator) })
	// Include the root itself (also covers UNC share roots on Windows).
	for i := -1; i < len(parts); i++ {
		if i >= 0 {
			current += parts[i] + string(os.PathSeparator)
		}
		query := current
		if i >= 0 {
			query = strings.TrimSuffix(current, string(os.PathSeparator))
		}
		info, err := os.Lstat(query)
		if err != nil {
			return fmt.Errorf("inspect path %q: %w", current, err)
		}
		if linked(info) {
			return fmt.Errorf("linked or reparse-point path is not supported: %s", current)
		}
		if !info.IsDir() {
			return fmt.Errorf("path component is not a directory: %s", current)
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
