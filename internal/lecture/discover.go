package lecture

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// FindEditable accepts a lecture, a course, or a notes root. It never follows
// linked directories and deliberately ignores finalized PDF-only lectures.
func FindEditable(path string) ([]string, error) {
	if err := rejectLinkedPath(path); err != nil {
		return nil, err
	}
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	var dirs []string
	err = filepath.WalkDir(path, func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() && name != path && strings.HasPrefix(entry.Name(), ".") {
			return filepath.SkipDir
		}
		if entry.Name() != "lecture.tex" {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if linked(info) || !info.Mode().IsRegular() {
			return fmt.Errorf("lecture source must be an ordinary file: %s", name)
		}
		dirs = append(dirs, filepath.Dir(name))
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(dirs) == 0 {
		return nil, fmt.Errorf("no editable lecture.tex files found in %s", path)
	}
	sort.Strings(dirs)
	return dirs, nil
}

func readOrdinary(path string) ([]byte, os.FileInfo, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, nil, err
	}
	if linked(info) || !info.Mode().IsRegular() {
		return nil, nil, fmt.Errorf("expected an ordinary file: %s", path)
	}
	data, err := os.ReadFile(path)
	return data, info, err
}
