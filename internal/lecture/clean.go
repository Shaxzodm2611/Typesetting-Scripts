package lecture

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

// Finalization holds the directory open through confirmation; no deletion occurs
// during Inspect. Callers must Close it whether they confirm or cancel.
type Finalization struct {
	root      *os.Root
	directory string
	marker    []byte
	pdfInfo   os.FileInfo
	pdfHash   [32]byte
}

func (f *Finalization) Directory() string { return f.directory }
func (f *Finalization) Close() error      { return f.root.Close() }

func Inspect(dir string) (*Finalization, error) {
	if err := rejectLinkedPath(dir); err != nil {
		return nil, err
	}
	absolute, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	if filepath.Dir(absolute) == absolute {
		return nil, fmt.Errorf("cannot clean a filesystem root")
	}
	root, err := os.OpenRoot(absolute)
	if err != nil {
		return nil, err
	}
	f := &Finalization{root: root, directory: absolute}
	good := false
	defer func() {
		if !good {
			root.Close()
		}
	}()
	if err = f.preflight(); err != nil {
		return nil, err
	}
	marker, err := readMarker(root)
	if err != nil {
		return nil, err
	}
	info, hash, err := inspectPDF(root)
	if err != nil {
		return nil, err
	}
	f.marker, f.pdfInfo, f.pdfHash = marker, info, hash
	good = true
	return f, nil
}

// ordinaryFile avoids treating a symlink, device, or directory as either control
// file. Root confinement also prevents an intervening link from escaping the root.
func ordinaryFile(root *os.Root, name string) (*os.File, error) {
	info, err := root.Lstat(name)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	if linked(info) || !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%s must be a regular file, not a link", name)
	}
	file, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	actual, err := file.Stat()
	if err != nil || !os.SameFile(info, actual) {
		file.Close()
		return nil, fmt.Errorf("%s changed during inspection", name)
	}
	return file, nil
}

func readMarker(root *os.Root) ([]byte, error) {
	file, err := ordinaryFile(root, ".notes.json")
	if err != nil {
		return nil, fmt.Errorf("not a tool-created lecture (ownership marker): %w", err)
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, 16*1024+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > 16*1024 {
		return nil, fmt.Errorf("ownership marker exceeds 16 KiB")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var m Metadata
	if err = decoder.Decode(&m); err != nil {
		return nil, fmt.Errorf("invalid ownership marker: %w", err)
	}
	var trailing any
	if decoder.Decode(&trailing) != io.EOF {
		return nil, fmt.Errorf("ownership marker contains trailing data")
	}
	if m.Version != 1 || m.Lecture < 1 || m.PDF != "lecture.pdf" || validateCourse(m.Course) != nil {
		return nil, fmt.Errorf("unsupported or invalid ownership marker")
	}
	return raw, nil
}

func inspectPDF(root *os.Root) (os.FileInfo, [32]byte, error) {
	var hash [32]byte
	file, err := ordinaryFile(root, "lecture.pdf")
	if err != nil {
		return nil, hash, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, hash, err
	}
	prefix := make([]byte, 5)
	if _, err = io.ReadFull(file, prefix); err != nil || string(prefix) != "%PDF-" {
		return nil, hash, fmt.Errorf("lecture.pdf is empty or does not have a PDF signature; build and review it first")
	}
	h := sha256.New()
	_, _ = h.Write(prefix)
	if _, err = io.Copy(h, file); err != nil {
		return nil, hash, err
	}
	copy(hash[:], h.Sum(nil))
	return info, hash, nil
}

func (f *Finalization) preflight() error {
	entries, err := rootEntries(f.root)
	if err != nil {
		return err
	}
	pdf, marker := false, false
	for _, e := range entries {
		pdf = pdf || e.Name() == "lecture.pdf"
		marker = marker || e.Name() == ".notes.json"
	}
	if !pdf || !marker {
		return fmt.Errorf("requires files named exactly lecture.pdf and .notes.json")
	}
	// Conservatively refuse links anywhere in the tree, including Windows junctions.
	// This is preferable to guessing whether a platform will traverse a reparse point.
	var walk func(string) error
	walk = func(name string) error {
		info, err := f.root.Lstat(name)
		if err != nil {
			return err
		}
		if linked(info) {
			return fmt.Errorf("remove the linked/reparse-point entry before cleanup: %s", name)
		}
		if !info.IsDir() {
			return nil
		}
		file, err := f.root.Open(name)
		if err != nil {
			return err
		}
		children, err := file.ReadDir(-1)
		closeErr := file.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
		for _, child := range children {
			if err = walk(filepath.Join(name, child.Name())); err != nil {
				return err
			}
		}
		return nil
	}
	for _, e := range entries {
		if err = walk(e.Name()); err != nil {
			return fmt.Errorf("inspect lecture contents: %w", err)
		}
	}
	return nil
}

func (f *Finalization) verify() error {
	marker, err := readMarker(f.root)
	if err != nil {
		return err
	}
	if !bytes.Equal(marker, f.marker) {
		return fmt.Errorf("ownership marker changed; run clean again")
	}
	info, hash, err := inspectPDF(f.root)
	if err != nil {
		return err
	}
	if !os.SameFile(info, f.pdfInfo) || info.Size() != f.pdfInfo.Size() || !info.ModTime().Equal(f.pdfInfo.ModTime()) || hash != f.pdfHash {
		return fmt.Errorf("PDF changed since inspection; review it and run clean again")
	}
	return nil
}

func (f *Finalization) Apply() error {
	if err := f.preflight(); err != nil {
		return err
	}
	if err := f.verify(); err != nil {
		return err
	}
	entries, err := rootEntries(f.root)
	if err != nil {
		return err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	for _, entry := range entries {
		name := entry.Name()
		if name == "lecture.pdf" || name == ".notes.json" {
			continue
		}
		if err = f.root.RemoveAll(name); err != nil {
			return fmt.Errorf("cleanup stopped at %s; PDF and retry marker retained: %w", name, err)
		}
	}
	if err = f.verify(); err != nil {
		return err
	}
	entries, err = rootEntries(f.root)
	if err != nil {
		return err
	}
	if len(entries) != 2 {
		return fmt.Errorf("lecture contents changed during cleanup; PDF and retry marker retained")
	}
	if err = f.root.Remove(".notes.json"); err != nil {
		return fmt.Errorf("remove ownership marker: %w", err)
	}
	entries, err = rootEntries(f.root)
	if err != nil || len(entries) != 1 || entries[0].Name() != "lecture.pdf" {
		// A concurrent writer may have added a file after the last enumeration.
		// Restore the marker exclusively so the user can retry without claiming success.
		file, restoreErr := f.root.OpenFile(".notes.json", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if restoreErr == nil {
			_, restoreErr = file.Write(f.marker)
			restoreErr = errors.Join(restoreErr, file.Close())
		}
		return errors.Join(fmt.Errorf("contents changed while finalizing; inspect the folder before retrying"), err, restoreErr)
	}
	return nil
}
