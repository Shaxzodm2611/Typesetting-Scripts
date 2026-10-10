package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Shaxzodm2611/Typesetting-Scripts/internal/lecture"
)

const checkHelp = `Usage: notes check [DIRECTORY] [--compile] [--strict]
Scans a lecture, course, or notes root (default: current directory).
Reports duplicate/placeholder labels, undefined references, missing images and
inputs, and unfinished TODO/FIXME/TBD notes. Literal TeX paths are checked;
arbitrary TeX macros are not expanded by the source checker.
--compile: also compile twice with pdfLaTeX into a temporary output directory.
Existing PDFs and build files are untouched. Requires pdflatex on PATH.
--strict: fail for warnings as well as errors. Otherwise only errors fail.
Example: notes check ELE727/lecture-02 --compile
`
const updateHelp = `Usage: notes update-style [DIRECTORY] [--dry-run] [--resolve=keep|replace]
Accepts a lecture, course, or notes root (default: current directory).
--dry-run: report versions, preserved custom edits, and conflicts; write nothing.
Apply: replace lecturenotes.sty after backing it up as lecturenotes.sty.bak
(then .bak.1, .bak.2, ...). Lecture content and figures remain in place.
Compatible custom edits are merged using bundled historical style versions.
Conflicts stop the entire update before writing. Review them, then use
--resolve=keep to favor conflicting local edits, or --resolve=replace to favor
updated definitions. Both choices retain compatible custom edits.
Unknown style versions require a manual merge. Rebuild PDFs after updating.
Example: notes update-style ELE727 --dry-run
`

func featurePath(cwd, path string) (string, error) {
	if path == "" {
		path = "."
	}
	if filepath.IsAbs(path) {
		return path, nil
	}
	if filepath.VolumeName(path) != "" || os.IsPathSeparator(path[0]) {
		return "", fmt.Errorf("use a fully qualified or ordinary relative path")
	}
	return cwd + string(os.PathSeparator) + path, nil
}

func runCheck(args []string, cwd string, streams IO) int {
	fail := func(code int, err error) int { fmt.Fprintln(streams.Err, "notes:", err); return code }
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		fmt.Fprint(streams.Out, checkHelp)
		return 0
	}
	path := ""
	compile, strict := false, false
	for _, arg := range args {
		switch {
		case arg == "--compile":
			compile = true
		case arg == "--strict":
			strict = true
		case strings.HasPrefix(arg, "-"):
			return fail(2, fmt.Errorf("unknown option %q; %s", arg, checkHelp))
		case path != "":
			return fail(2, fmt.Errorf("%s", checkHelp))
		default:
			path = arg
		}
	}
	path, err := featurePath(cwd, path)
	if err != nil {
		return fail(2, err)
	}
	diagnostics, count, err := lecture.Check(path, lecture.CheckOptions{Compile: compile})
	if err != nil {
		return fail(1, err)
	}
	errors, warnings := 0, 0
	for _, d := range diagnostics {
		name := d.File
		if rel, err := filepath.Rel(cwd, name); err == nil {
			name = rel
		}
		fmt.Fprintf(streams.Out, "%s:%d: %s [%s] %s\n", name, d.Line, d.Severity, d.Kind, d.Message)
		if d.Severity == "error" {
			errors++
		} else {
			warnings++
		}
	}
	fmt.Fprintf(streams.Out, "Checked %d lecture(s): %d error(s), %d warning(s).\n", count, errors, warnings)
	if errors > 0 || strict && warnings > 0 {
		return 1
	}
	return 0
}

func runUpdate(args []string, cwd string, streams IO) int {
	fail := func(code int, err error) int { fmt.Fprintln(streams.Err, "notes:", err); return code }
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		fmt.Fprint(streams.Out, updateHelp)
		return 0
	}
	path, resolve := "", ""
	dry := false
	for _, arg := range args {
		switch {
		case arg == "--dry-run":
			dry = true
		case strings.HasPrefix(arg, "--resolve="):
			resolve = strings.TrimPrefix(arg, "--resolve=")
			if resolve != "keep" && resolve != "replace" {
				return fail(2, fmt.Errorf("--resolve must be keep or replace"))
			}
		case strings.HasPrefix(arg, "-"):
			return fail(2, fmt.Errorf("unknown option %q; %s", arg, updateHelp))
		case path != "":
			return fail(2, fmt.Errorf("%s", updateHelp))
		default:
			path = arg
		}
	}
	path, err := featurePath(cwd, path)
	if err != nil {
		return fail(2, err)
	}
	plans, err := lecture.PreviewStyles(path, resolve)
	if err != nil {
		return fail(1, err)
	}
	updates, conflicts := 0, 0
	for _, plan := range plans {
		name := plan.Directory
		if rel, err := filepath.Rel(cwd, name); err == nil {
			name = rel
		}
		customStatus := "retained"
		if plan.Status == "conflict" {
			customStatus = "found"
		}
		fmt.Fprintf(streams.Out, "%s: %s, v%s -> v%s; %d custom edit(s) %s\n", name, plan.Status, plan.From, plan.To, plan.CustomEdits, customStatus)
		if plan.Migration {
			fmt.Fprintln(streams.Out, "  Legacy bookmark settings retained; heading anchors supplied by the new package.")
		}
		for _, conflict := range plan.Conflicts {
			fmt.Fprintln(streams.Out, "  "+conflict)
		}
		if plan.Status == "conflict" {
			conflicts++
		}
		if plan.Status == "update" {
			updates++
		}
	}
	if conflicts > 0 {
		return fail(1, fmt.Errorf("%d style(s) have unresolved conflicts; nothing changed. Review --dry-run and resolve the reported edits", conflicts))
	}
	if dry {
		fmt.Fprintf(streams.Out, "%d editable lecture(s); %d style(s) would update. No files written.\n", len(plans), updates)
		return 0
	}
	for _, plan := range plans {
		backup, err := lecture.ApplyStyle(plan)
		if err != nil {
			return fail(1, err)
		}
		if backup != "" {
			fmt.Fprintf(streams.Out, "Updated %s (backup: %s)\n", filepath.Join(plan.Directory, "lecturenotes.sty"), backup)
		}
	}
	fmt.Fprintf(streams.Out, "%d style(s) updated; %d backup(s) created. Rebuild affected PDFs in VS Code.\n", updates, updates)
	return 0
}
