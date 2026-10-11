package lecture

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

type ExportOptions struct {
	Vault, Term string
	// Optional discovery roots make discovery independent of process globals.
	Home, ConfigDir string
}

type fileSnapshot struct {
	info os.FileInfo
	hash [32]byte
}

type Export struct {
	Source, Destination, Markdown string
	source, target                *os.Root
	pdf                           fileSnapshot
	previousPDF, note             *fileSnapshot
	name, markdownName            string
}

func (e *Export) Close() error { return errors.Join(e.source.Close(), e.target.Close()) }

var lectureDirectory = regexp.MustCompile(`(?i)^lecture-([0-9]+)$`)
var lectureFilename = regexp.MustCompile(`(?i)^lecture[ _-]*([0-9]+)(?:$|[ ._-])`)

// PlanExport resolves and inspects every affected file without changing any.
// It also works after notes clean, using COURSE/lecture-NN when the marker is gone.
func PlanExport(directory string, options ExportOptions) (*Export, error) {
	if err := rejectLinkedPath(directory); err != nil {
		return nil, err
	}
	directory, err := filepath.Abs(directory)
	if err != nil {
		return nil, err
	}
	match := lectureDirectory.FindStringSubmatch(filepath.Base(directory))
	if match == nil {
		return nil, fmt.Errorf("expected a COURSE/lecture-NN directory")
	}
	number, err := strconv.Atoi(match[1])
	if err != nil || number < 1 {
		return nil, fmt.Errorf("invalid lecture number")
	}
	course := filepath.Base(filepath.Dir(directory))
	if err := validateCourse(course); err != nil {
		return nil, err
	}
	source, err := os.OpenRoot(directory)
	if err != nil {
		return nil, err
	}
	good := false
	defer func() {
		if !good {
			source.Close()
		}
	}()
	if _, err := source.Lstat(".notes.json"); err == nil {
		raw, err := readMarker(source)
		if err != nil {
			return nil, err
		}
		var m Metadata
		if err := json.Unmarshal(raw, &m); err != nil {
			return nil, err
		}
		if m.Course != course || m.Lecture != number {
			return nil, fmt.Errorf("lecture folder disagrees with its ownership marker")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	info, hash, err := inspectPDF(source)
	if err != nil {
		return nil, err
	}
	if options.Term == "." || options.Term == ".." || strings.ContainsAny(options.Term, `/\`) {
		return nil, fmt.Errorf("--term must be a single semester folder name")
	}
	vaults, err := findVaults(options)
	if err != nil {
		return nil, err
	}
	destination, err := findLectureFolder(vaults, course, number, options.Term)
	if err != nil {
		return nil, err
	}
	if err := rejectLinkedPath(destination); err != nil {
		return nil, err
	}
	target, err := os.OpenRoot(destination)
	if err != nil {
		return nil, err
	}
	defer func() {
		if !good {
			target.Close()
		}
	}()
	e := &Export{Source: filepath.Join(directory, "lecture.pdf"), source: source, target: target,
		pdf: fileSnapshot{info, hash}, name: fmt.Sprintf("Lecture %d.pdf", number)}
	entries, err := rootEntries(target)
	if err != nil {
		return nil, err
	}
	var pdfName string
	for _, entry := range entries {
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if (ext != ".md" && ext != ".pdf") || !matchesLecture(entry.Name(), number) {
			continue
		}
		if ext == ".md" {
			if e.markdownName != "" {
				return nil, fmt.Errorf("multiple Markdown notes match lecture %d in %s", number, destination)
			}
			e.markdownName = entry.Name()
			e.note, err = snapshotExportFile(target, entry.Name())
			if err != nil {
				return nil, err
			}
		} else {
			if pdfName != "" {
				return nil, fmt.Errorf("multiple PDFs match lecture %d in %s", number, destination)
			}
			pdfName = entry.Name()
			e.previousPDF, err = snapshotExportFile(target, entry.Name())
			if err != nil {
				return nil, err
			}
		}
	}
	if e.markdownName != "" {
		e.name = strings.TrimSuffix(e.markdownName, filepath.Ext(e.markdownName)) + ".pdf"
	} else if pdfName != "" {
		// Preserve the title on repeat exports after the Markdown was removed.
		e.name = pdfName
	}
	if pdfName != "" && pdfName != e.name {
		return nil, fmt.Errorf("existing PDF title conflicts with Markdown title: %s", filepath.Join(destination, pdfName))
	}
	e.Destination = filepath.Join(destination, e.name)
	if e.markdownName != "" {
		e.Markdown = filepath.Join(destination, e.markdownName)
	}
	good = true
	return e, nil
}

func matchesLecture(name string, number int) bool {
	base := strings.TrimSuffix(name, filepath.Ext(name))
	m := lectureFilename.FindStringSubmatch(base)
	if m == nil {
		return false
	}
	n, err := strconv.Atoi(m[1])
	return err == nil && n == number
}

func snapshotExportFile(root *os.Root, name string) (*fileSnapshot, error) {
	f, err := ordinaryFile(root, name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return nil, err
	}
	var sum [32]byte
	copy(sum[:], h.Sum(nil))
	return &fileSnapshot{info, sum}, nil
}

func unchanged(root *os.Root, name string, expected *fileSnapshot) error {
	if expected == nil {
		if _, err := root.Lstat(name); errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("%s appeared during export; retry", name)
	}
	actual, err := snapshotExportFile(root, name)
	if err != nil {
		return err
	}
	if !os.SameFile(expected.info, actual.info) || expected.hash != actual.hash {
		return fmt.Errorf("%s changed during export; retry", name)
	}
	return nil
}

// Apply stages and verifies a complete PDF before replacing the destination.
// Markdown removal happens last, and only if that exact note is unchanged.
func (e *Export) Apply() error {
	input, err := ordinaryFile(e.source, "lecture.pdf")
	if err != nil {
		return err
	}
	defer input.Close()
	temp := ".notes-export-" + rand.Text() + ".tmp"
	output, err := e.target.OpenFile(temp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	defer e.target.Remove(temp)
	h := sha256.New()
	_, copyErr := io.Copy(io.MultiWriter(output, h), input)
	syncErr := output.Sync()
	closeErr := output.Close()
	if err := errors.Join(copyErr, syncErr, closeErr); err != nil {
		return err
	}
	if string(h.Sum(nil)) != string(e.pdf.hash[:]) {
		return fmt.Errorf("source PDF changed during export; nothing replaced")
	}
	if err := unchanged(e.source, "lecture.pdf", &e.pdf); err != nil {
		return err
	}
	if err := unchanged(e.target, e.name, e.previousPDF); err != nil {
		return err
	}
	if e.note != nil {
		if err := unchanged(e.target, e.markdownName, e.note); err != nil {
			return err
		}
	}
	if err := e.target.Rename(temp, e.name); err != nil {
		return fmt.Errorf("install PDF: %w", err)
	}
	copied, err := snapshotExportFile(e.target, e.name)
	if err != nil {
		return fmt.Errorf("cannot verify copied PDF; Markdown retained: %w", err)
	}
	if copied.hash != e.pdf.hash {
		return fmt.Errorf("copied PDF changed before verification; Markdown retained")
	}
	if e.note != nil {
		if err := unchanged(e.target, e.markdownName, e.note); err != nil {
			return fmt.Errorf("PDF copied to %s; Markdown retained: %w", e.Destination, err)
		}
		if err := e.target.Remove(e.markdownName); err != nil {
			return fmt.Errorf("PDF copied to %s; cannot remove Markdown: %w", e.Destination, err)
		}
	}
	return nil
}

func findVaults(options ExportOptions) ([]string, error) {
	if options.Vault != "" {
		if err := rejectLinkedPath(options.Vault); err != nil {
			return nil, err
		}
		path, err := filepath.Abs(options.Vault)
		if err != nil {
			return nil, err
		}
		if err := rejectLinkedPath(path); err != nil {
			return nil, err
		}
		if !isDirectory(filepath.Join(path, "School")) {
			return nil, fmt.Errorf("vault has no School directory: %s", path)
		}
		return []string{path}, nil
	}
	home := options.Home
	if home == "" {
		var err error
		home, err = os.UserHomeDir()
		if err != nil {
			return nil, err
		}
	}
	config := options.ConfigDir
	if config == "" {
		config, _ = os.UserConfigDir()
	}
	seen := make(map[string]bool)
	var found []string
	add := func(path string) {
		absolute, err := filepath.Abs(path)
		key := absolute
		if runtime.GOOS == "windows" {
			key = strings.ToLower(key)
		}
		if err != nil || seen[key] {
			return
		}
		if !isDirectory(filepath.Join(absolute, ".obsidian")) || !isDirectory(filepath.Join(absolute, "School")) {
			return
		}
		if rejectLinkedPath(absolute) != nil {
			return
		}
		seen[key] = true
		found = append(found, absolute)
	}
	// Obsidian's local registry supports vaults on another drive or mount point.
	if config != "" {
		var registry struct {
			Vaults map[string]struct {
				Path string `json:"path"`
			} `json:"vaults"`
		}
		if data, err := os.ReadFile(filepath.Join(config, "obsidian", "obsidian.json")); err == nil && json.Unmarshal(data, &registry) == nil {
			for _, v := range registry.Vaults {
				if v.Path != "" {
					add(v.Path)
				}
			}
		}
	}
	for _, parent := range []string{"", "Documents", "OneDrive", filepath.Join("OneDrive", "Documents"), "Sync", "Syncthing"} {
		add(filepath.Join(home, parent, "Personal Obsidian Vault"))
	}
	// Fallback for different Syncthing locations; do not traverse linked trees,
	// caches, dependency directories, or the contents of a discovered vault.
	if len(found) == 0 {
		if err := rejectLinkedPath(home); err != nil {
			return nil, err
		}
		err := filepath.WalkDir(home, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return filepath.SkipDir
			}
			if !entry.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(home, path)
			if err != nil {
				return filepath.SkipDir
			}
			if rel != "." && (strings.HasPrefix(entry.Name(), ".") || entry.Name() == "node_modules" || entry.Name() == "AppData" || len(strings.Split(rel, string(os.PathSeparator))) > 6) {
				return filepath.SkipDir
			}
			info, err := entry.Info()
			if err != nil || linked(info) {
				return filepath.SkipDir
			}
			before := len(found)
			add(path)
			if len(found) != before {
				return filepath.SkipDir
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Strings(found)
	if len(found) == 0 {
		return nil, fmt.Errorf("cannot find a school Obsidian vault; use --vault PATH or NOTES_OBSIDIAN_VAULT")
	}
	return found, nil
}

func isDirectory(path string) bool { info, err := os.Stat(path); return err == nil && info.IsDir() }

func findLectureFolder(vaults []string, course string, number int, term string) (string, error) {
	var folders, matching []string
	for _, vault := range vaults {
		school := filepath.Join(vault, "School")
		err := filepath.WalkDir(school, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if !entry.IsDir() {
				return nil
			}
			info, err := entry.Info()
			if err != nil {
				return err
			}
			if linked(info) {
				return filepath.SkipDir
			}
			rel, err := filepath.Rel(school, path)
			if err != nil {
				return err
			}
			parts := strings.Split(rel, string(os.PathSeparator))
			if rel != "." && (strings.HasPrefix(entry.Name(), ".") || len(parts) > 4 || (term != "" && !strings.EqualFold(parts[0], term))) {
				return filepath.SkipDir
			}
			if !strings.EqualFold(entry.Name(), course) {
				return nil
			}
			entries, err := os.ReadDir(path)
			if err != nil {
				return err
			}
			for _, child := range entries {
				if !strings.EqualFold(child.Name(), "Lectures") || !child.IsDir() {
					continue
				}
				folder := filepath.Join(path, child.Name())
				if err := rejectLinkedPath(folder); err != nil {
					return err
				}
				files, err := os.ReadDir(folder)
				if err != nil {
					return err
				}
				folders = append(folders, folder)
				for _, file := range files {
					ext := strings.ToLower(filepath.Ext(file.Name()))
					if (ext == ".md" || ext == ".pdf") && matchesLecture(file.Name(), number) {
						matching = append(matching, folder)
						break
					}
				}
			}
			return filepath.SkipDir
		})
		if err != nil {
			return "", err
		}
	}
	if len(matching) == 1 {
		return matching[0], nil
	}
	if len(matching) > 1 {
		folders = matching
	}
	if len(folders) == 1 {
		return folders[0], nil
	}
	if len(folders) == 0 {
		return "", fmt.Errorf("no %s/Lectures folder found in the school vault(s)", course)
	}
	return "", fmt.Errorf("multiple lecture destinations match; use --vault and/or --term:\n  %s", strings.Join(folders, "\n  "))
}
