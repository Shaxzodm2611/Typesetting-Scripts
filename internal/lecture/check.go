package lecture

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Diagnostic struct {
	File, Severity, Kind, Message string
	Line                          int
}

type CheckOptions struct{ Compile bool }

type texCall struct {
	name, argument, options string
	offset                  int
}

// maskTeX preserves byte offsets and newlines while hiding comments and code.
// TeX expansion is intentionally left to the optional real compilation pass.
func maskTeX(source string) string {
	return maskTeXMode(source, true)
}
func maskTeXMode(source string, hideComments bool) string {
	b := []byte(source)
	blank := func(start, end int) {
		for i := start; i < end; i++ {
			if b[i] != '\n' {
				b[i] = ' '
			}
		}
	}
	for i := 0; i < len(b); {
		if b[i] == '%' {
			end := strings.IndexByte(source[i:], '\n')
			if end < 0 {
				end = len(b) - i
			}
			if hideComments {
				blank(i, i+end)
			}
			i += end
		} else if b[i] == '\\' {
			matched := false
			for _, env := range []string{"notecode", "lstlisting", "verbatim", "Verbatim", "minted"} {
				begin, end := `\begin{`+env+`}`, `\end{`+env+`}`
				if strings.HasPrefix(source[i:], begin) {
					j := strings.Index(source[i+len(begin):], end)
					stop := len(b)
					if j >= 0 {
						stop = i + len(begin) + j + len(end)
					}
					blank(i, stop)
					i = stop
					matched = true
					break
				}
			}
			if matched {
				continue
			}
			if strings.HasPrefix(source[i:], `\verb`) {
				j := i + 5
				if j < len(b) && b[j] == '*' {
					j++
				}
				if j < len(b) && !isLetter(b[j]) && b[j] != ' ' && b[j] != '\n' {
					end := strings.IndexByte(source[j+1:], b[j])
					stop := len(b)
					if end >= 0 {
						stop = j + end + 2
					}
					blank(i, stop)
					i = stop
					continue
				}
			}
			i += 2 // In particular, an escaped percent is not a comment.
		} else {
			i++
		}
	}
	return string(b)
}

func isLetter(c byte) bool { return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '@' }
func skipSpace(s string, i int) int {
	for i < len(s) && strings.ContainsRune(" \t\r\n", rune(s[i])) {
		i++
	}
	return i
}
func texGroup(s string, i int, open, close byte) (string, int, bool) {
	if i >= len(s) || s[i] != open {
		return "", i, false
	}
	depth := 1
	for j := i + 1; j < len(s); j++ {
		if s[j] == '\\' {
			j++
			continue
		}
		if s[j] == open {
			depth++
		}
		if s[j] == close {
			depth--
			if depth == 0 {
				return s[i+1 : j], j + 1, true
			}
		}
	}
	return "", i, false
}
func texCalls(source string) []texCall {
	var calls []texCall
	for i := 0; i < len(source); {
		if source[i] != '\\' {
			i++
			continue
		}
		start := i
		i++
		j := i
		for i < len(source) && isLetter(source[i]) {
			i++
		}
		if i == j {
			i++
			continue
		}
		name := source[j:i]
		if i < len(source) && source[i] == '*' {
			i++
		}
		pos := skipSpace(source, i)
		options := ""
		for pos < len(source) && source[pos] == '[' {
			value, end, ok := texGroup(source, pos, '[', ']')
			if !ok {
				break
			}
			options = value
			pos = skipSpace(source, end)
		}
		arg, _, ok := texGroup(source, pos, '{', '}')
		if ok {
			calls = append(calls, texCall{name, arg, options, start})
		}
	}
	return calls
}

var todoPattern = regexp.MustCompile(`(?i)\b(TODO|FIXME|TBD)\b|\bAdd\b[^\n]{0,120}\bhere\b`)
var labelOption = regexp.MustCompile(`(?:^|,)\s*label\s*=\s*(\{[^{}]*\}|[^,]+)`)
var logLinePattern = regexp.MustCompile(`(?:at lines? |on input line |\.tex:)(\d+)`)

// Check scans each lecture's input/include graph and optionally compiles twice
// into a temporary output directory. Existing PDFs, logs and aux files are untouched.
func Check(path string, options CheckOptions) ([]Diagnostic, int, error) {
	dirs, err := FindEditable(path)
	if err != nil {
		return nil, 0, err
	}
	var all []Diagnostic
	for _, dir := range dirs {
		labels := map[string]Diagnostic{}
		var refs []Diagnostic
		visited, active := map[string]bool{}, map[string]bool{}
		graphics := []string{""}
		add := func(file string, line int, severity, kind, message string) {
			all = append(all, Diagnostic{File: file, Line: line, Severity: severity, Kind: kind, Message: message})
		}
		resolve := func(name string, image bool) (string, bool) {
			prefixes := []string{""}
			exts := []string{"", ".tex"}
			if image {
				prefixes = graphics
				exts = []string{"", ".pdf", ".png", ".jpg", ".jpeg", ".mps", ".eps"}
			}
			for _, prefix := range prefixes {
				for _, ext := range exts {
					candidate := name
					if !filepath.IsAbs(candidate) {
						candidate = filepath.Join(dir, prefix, name)
					}
					if filepath.Ext(name) != "" && ext != "" {
						continue
					}
					candidate += ext
					if info, err := os.Stat(candidate); err == nil && info.Mode().IsRegular() {
						return candidate, true
					}
				}
			}
			return name, false
		}
		var visit func(string) error
		visit = func(file string) error {
			file = filepath.Clean(file)
			if active[file] {
				add(file, 1, "error", "input-cycle", "recursive input/include cycle")
				return nil
			}
			if visited[file] {
				return nil
			}
			visited[file] = true
			active[file] = true
			defer delete(active, file)
			raw, _, err := readOrdinary(file)
			if err != nil {
				return err
			}
			source := string(raw)
			masked := maskTeX(source)
			for _, index := range todoPattern.FindAllStringIndex(maskTeXMode(source, false), -1) {
				line := 1 + strings.Count(source[:index[0]], "\n")
				add(file, line, "warning", "todo", strings.TrimSpace(source[index[0]:index[1]]))
			}
			for _, call := range texCalls(masked) {
				line := 1 + strings.Count(masked[:call.offset], "\n")
				arg := strings.TrimSpace(call.argument)
				literal := !strings.ContainsAny(arg, `\#$`)
				switch call.name {
				case "graphicspath":
					for i := 0; i < len(arg); {
						i = skipSpace(arg, i)
						value, end, ok := texGroup(arg, i, '{', '}')
						if !ok {
							break
						}
						graphics = append(graphics, value)
						i = end
					}
				case "label":
					if literal {
						recordLabel(file, line, arg, labels, add)
					}
				case "ref", "eqref", "pageref", "autoref", "cref", "Cref":
					if literal {
						for _, label := range strings.Split(arg, ",") {
							refs = append(refs, Diagnostic{File: file, Line: line, Message: strings.TrimSpace(label)})
						}
					}
				case "input", "include":
					if !literal {
						continue
					}
					included, ok := resolve(arg, false)
					if !ok {
						add(file, line, "error", "missing-input", fmt.Sprintf("input file %q was not found", arg))
						continue
					}
					if err := visit(included); err != nil {
						return err
					}
				case "includegraphics", "notefigure", "sourcefigure":
					if call.name == "notefigure" {
						if match := labelOption.FindStringSubmatch(call.options); match != nil {
							label := strings.Trim(strings.TrimSpace(match[1]), "{}")
							if !strings.ContainsAny(label, `\#$`) {
								recordLabel(file, line, label, labels, add)
							}
						}
					}
					if !literal {
						continue
					}
					if call.name == "sourcefigure" {
						arg = filepath.Join("figures", arg)
					}
					if _, ok := resolve(arg, true); !ok {
						add(file, line, "error", "missing-image", fmt.Sprintf("image %q was not found", arg))
					}
				}
			}
			return nil
		}
		if err := visit(filepath.Join(dir, "lecture.tex")); err != nil {
			return nil, 0, err
		}
		for _, ref := range refs {
			if _, ok := labels[ref.Message]; !ok {
				add(ref.File, ref.Line, "error", "undefined-reference", fmt.Sprintf("reference %q has no label", ref.Message))
			}
		}
		if options.Compile {
			diagnostics, err := compileCheck(dir)
			if err != nil {
				return nil, 0, err
			}
			all = append(all, diagnostics...)
		}
	}
	sort.SliceStable(all, func(i, j int) bool {
		if all[i].File != all[j].File {
			return all[i].File < all[j].File
		}
		return all[i].Line < all[j].Line
	})
	return all, len(dirs), nil
}

func recordLabel(file string, line int, label string, labels map[string]Diagnostic, add func(string, int, string, string, string)) {
	if label == "" || strings.HasSuffix(label, ":") {
		add(file, line, "warning", "placeholder-label", fmt.Sprintf("label %q needs a descriptive identifier", label))
	}
	if first, ok := labels[label]; ok {
		add(file, line, "error", "duplicate-label", fmt.Sprintf("label %q is already defined at %s:%d", label, first.File, first.Line))
	} else {
		labels[label] = Diagnostic{File: file, Line: line}
	}
}

func compileCheck(dir string) ([]Diagnostic, error) {
	compiler, err := exec.LookPath("pdflatex")
	if err != nil {
		return nil, fmt.Errorf("--compile requires pdflatex on PATH: %w", err)
	}
	temp, err := os.MkdirTemp("", "notes-check-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(temp)
	var compileErr error
	var output []byte
	for pass := 0; pass < 2; pass++ {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		cmd := exec.CommandContext(ctx, compiler, "-no-shell-escape", "-interaction=nonstopmode", "-halt-on-error", "-file-line-error", "-jobname=notes-check", "-output-directory="+temp, "lecture.tex")
		cmd.Dir = dir
		output, compileErr = cmd.CombinedOutput()
		cancel()
		if compileErr != nil {
			break
		}
	}
	log, _ := os.ReadFile(filepath.Join(temp, "notes-check.log"))
	var diagnostics []Diagnostic
	scanner := bufio.NewScanner(strings.NewReader(string(log)))
	for scanner.Scan() {
		line := scanner.Text()
		// A deliberate -no-shell-escape build cannot run EPS conversion. This
		// configuration notice is not a defect in a PDF/PNG/JPEG lecture.
		if line == "Package epstopdf Warning: Shell escape feature is not enabled." {
			continue
		}
		if strings.HasPrefix(line, "Overfull ") || strings.HasPrefix(line, "Underfull ") || strings.HasPrefix(line, "LaTeX Warning:") || strings.HasPrefix(line, "Package ") && strings.Contains(line, " Warning:") {
			number := 1
			if match := logLinePattern.FindStringSubmatch(line); match != nil {
				number, _ = strconv.Atoi(match[1])
			}
			diagnostics = append(diagnostics, Diagnostic{File: filepath.Join(dir, "lecture.tex"), Line: number, Severity: "warning", Kind: "latex-warning", Message: line})
		}
	}
	if compileErr != nil {
		message := string(output)
		if len(message) > 1800 {
			message = message[len(message)-1800:]
		}
		diagnostics = append(diagnostics, Diagnostic{File: filepath.Join(dir, "lecture.tex"), Line: 1, Severity: "error", Kind: "latex-error", Message: strings.TrimSpace(message)})
	}
	return diagnostics, scanner.Err()
}
