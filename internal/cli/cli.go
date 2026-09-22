// Package cli implements the terminal interface independently of process globals.
package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Shaxzodm2611/Typesetting-Scripts/internal/lecture"
)

type IO struct {
	In          io.Reader
	Out         io.Writer
	Err         io.Writer
	Interactive bool
}

const help = `notes — lecture summaries with LaTeX

Usage:
  notes new COURSE NUMBER
  notes clean DIRECTORY [--yes]
  notes --version

new creates blank lecture.tex and the formatting package in COURSE/lecture-NN.
Build and review lecture.pdf in VS Code. This tool does not compile LaTeX.
clean permanently deletes everything in that lecture folder EXCEPT lecture.pdf,
including editable source, the formatting package, figures, and build artifacts.

Use notes new --help or notes clean --help for details.
`
const newHelp = `Usage: notes new COURSE NUMBER
Creates COURSE/lecture-NN in the current directory; refuses existing lectures.
COURSE: ASCII letters/digits, hyphens, underscores; no Windows device names.
NUMBER: a positive integer. Example: notes new ECE342 2
`
const cleanHelp = `Usage: notes clean DIRECTORY [--yes]
Permanently removes ALL contents except the final lecture.pdf.
Requires a tool-created directory and a PDF built beside lecture.tex.
Prompts for confirmation; --yes skips the prompt (required for scripts).
Example: notes clean "ECE342/lecture-02"
`

func Run(args []string, cwd string, streams IO, version string) int {
	fail := func(code int, err error) int { fmt.Fprintln(streams.Err, "notes:", err); return code }
	if len(args) == 0 {
		fmt.Fprint(streams.Out, help)
		return 0
	}
	switch args[0] {
	case "--help", "-h":
		if len(args) != 1 {
			return fail(2, fmt.Errorf("unexpected arguments after help"))
		}
		fmt.Fprint(streams.Out, help)
		return 0
	case "--version":
		if len(args) != 1 {
			return fail(2, fmt.Errorf("unexpected arguments after --version"))
		}
		fmt.Fprintln(streams.Out, "notes", version)
		return 0
	case "new":
		if len(args) == 2 && (args[1] == "--help" || args[1] == "-h") {
			fmt.Fprint(streams.Out, newHelp)
			return 0
		}
		if len(args) != 3 {
			return fail(2, fmt.Errorf("%s", newHelp))
		}
		if strings.HasPrefix(args[1], "-") {
			return fail(2, fmt.Errorf("unknown option %q; %s", args[1], newHelp))
		}
		dir, err := lecture.New(cwd, args[1], args[2])
		if err != nil {
			return fail(1, err)
		}
		fmt.Fprintf(streams.Out, "Created %s\nOpen lecture.tex in VS Code to edit and build.\n", dir)
		return 0
	case "clean":
		if len(args) == 2 && (args[1] == "--help" || args[1] == "-h") {
			fmt.Fprint(streams.Out, cleanHelp)
			return 0
		}
		yes := false
		path := ""
		for _, arg := range args[1:] {
			switch {
			case arg == "--yes":
				yes = true
			case strings.HasPrefix(arg, "-"):
				return fail(2, fmt.Errorf("unknown option %q; %s", arg, cleanHelp))
			case path != "":
				return fail(2, fmt.Errorf("%s", cleanHelp))
			default:
				path = arg
			}
		}
		if path == "" {
			return fail(2, fmt.Errorf("%s", cleanHelp))
		}
		if !yes && !streams.Interactive {
			return fail(1, fmt.Errorf("cleanup requires an interactive terminal; use --yes only after reviewing the PDF"))
		}
		if !filepath.IsAbs(path) {
			if filepath.VolumeName(path) != "" || (len(path) > 0 && os.IsPathSeparator(path[0])) {
				return fail(2, fmt.Errorf("use a fully qualified path such as C:\\Notes\\ECE342\\lecture-02"))
			}
			// Preserve raw components until Inspect has rejected symlinks/reparse points.
			path = cwd + string(os.PathSeparator) + path
		}
		f, err := lecture.Inspect(path)
		if err != nil {
			return fail(1, err)
		}
		defer f.Close()
		_, err = fmt.Fprintf(streams.Out, "Lecture directory: %s\nKeep: %s\nPermanently delete EVERYTHING ELSE, including .tex source, .sty package, figures, subdirectories, and build artifacts.\n", f.Directory(), filepath.Join(f.Directory(), "lecture.pdf"))
		if err != nil {
			return fail(1, fmt.Errorf("cannot display cleanup summary: %w", err))
		}
		if !yes {
			if _, err = fmt.Fprint(streams.Out, "Continue? [y/N] "); err != nil {
				return fail(1, err)
			}
			scanner := bufio.NewScanner(streams.In)
			scanner.Buffer(make([]byte, 256), 4096)
			if !scanner.Scan() {
				if err = scanner.Err(); err == nil {
					err = io.EOF
				}
				return fail(1, fmt.Errorf("no valid confirmation; nothing deleted: %w", err))
			}
			answer := strings.ToLower(strings.TrimSpace(scanner.Text()))
			if answer != "y" && answer != "yes" {
				fmt.Fprintln(streams.Out, "Cancelled. Nothing deleted.")
				return 0
			}
		}
		if err = f.Apply(); err != nil {
			return fail(1, err)
		}
		fmt.Fprintf(streams.Out, "Finalized %s — only lecture.pdf remains.\n", f.Directory())
		return 0
	default:
		return fail(2, fmt.Errorf("unknown command %q; run notes --help", args[0]))
	}
}
