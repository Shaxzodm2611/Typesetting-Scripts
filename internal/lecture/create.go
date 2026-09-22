package lecture

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type Metadata struct {
	Version int    `json:"version"`
	Course  string `json:"course"`
	Lecture int    `json:"lecture"`
	PDF     string `json:"pdf"`
}

var coursePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]*$`)
var numberPattern = regexp.MustCompile(`^[0-9]+$`)

func validateCourse(course string) error {
	if !coursePattern.MatchString(course) {
		return fmt.Errorf("course must start with a letter or digit and contain only ASCII letters, digits, hyphens or underscores")
	}
	upper := strings.ToUpper(course)
	if upper == "CON" || upper == "PRN" || upper == "AUX" || upper == "NUL" || (len(upper) == 4 && (strings.HasPrefix(upper, "COM") || strings.HasPrefix(upper, "LPT")) && upper[3] >= '1' && upper[3] <= '9') {
		return fmt.Errorf("%q is a reserved Windows device name", course)
	}
	return nil
}

// New creates a lecture exclusively. Existing directories are never reused.
func New(base, course, number string) (string, error) {
	if err := validateCourse(course); err != nil {
		return "", err
	}
	n, err := strconv.Atoi(number)
	if !numberPattern.MatchString(number) || err != nil || n < 1 {
		return "", fmt.Errorf("lecture number must be a positive integer")
	}
	if err = rejectLinkedPath(base); err != nil {
		return "", err
	}
	base, err = filepath.Abs(base)
	if err != nil {
		return "", err
	}
	root, err := os.OpenRoot(base)
	if err != nil {
		return "", err
	}
	defer root.Close()
	entries, err := rootEntries(root)
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		if strings.EqualFold(e.Name(), course) && e.Name() != course {
			return "", fmt.Errorf("course %q conflicts with existing %q; use its exact spelling", course, e.Name())
		}
	}
	if err = root.Mkdir(course, 0755); err != nil && !errors.Is(err, os.ErrExist) {
		return "", err
	}
	info, err := root.Lstat(course)
	if err != nil {
		return "", err
	}
	if linked(info) || !info.IsDir() {
		return "", fmt.Errorf("course path must be an ordinary directory")
	}
	courseRoot, err := root.OpenRoot(course)
	if err != nil {
		return "", err
	}
	defer courseRoot.Close()
	name := fmt.Sprintf("lecture-%02d", n)
	lectureEntries, err := rootEntries(courseRoot)
	if err != nil {
		return "", err
	}
	for _, e := range lectureEntries {
		if strings.EqualFold(e.Name(), name) {
			return "", fmt.Errorf("lecture directory %q already exists; existing lectures are never overwritten", e.Name())
		}
	}
	if err = courseRoot.Mkdir(name, 0755); err != nil {
		return "", fmt.Errorf("create %s: %w (existing lectures are never overwritten)", name, err)
	}
	done := false
	defer func() {
		if !done {
			_ = courseRoot.RemoveAll(name)
		}
	}()
	lectureRoot, err := courseRoot.OpenRoot(name)
	if err != nil {
		return "", err
	}
	defer lectureRoot.Close()
	var source bytes.Buffer
	err = sourceTemplate.Execute(&source, struct {
		Course  string
		Lecture int
	}{strings.ReplaceAll(course, "_", `\_`), n})
	if err != nil {
		return "", err
	}
	sty, err := assets.ReadFile("assets/lecturenotes.sty")
	if err != nil {
		return "", err
	}
	marker, err := json.MarshalIndent(Metadata{1, course, n, "lecture.pdf"}, "", "  ")
	if err != nil {
		return "", err
	}
	files := []struct {
		name string
		data []byte
	}{{"lecture.tex", source.Bytes()}, {"lecturenotes.sty", sty}, {".notes.json", append(marker, '\n')}}
	for _, file := range files {
		if err = lectureRoot.WriteFile(file.name, file.data, 0644); err != nil {
			return "", fmt.Errorf("write %s: %w", file.name, err)
		}
	}
	done = true
	return filepath.Join(base, course, name), nil
}
