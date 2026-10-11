package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Shaxzodm2611/Typesetting-Scripts/internal/lecture"
)

const exportHelp = `Usage: notes export [DIRECTORY] [--vault PATH] [--term NAME] [--dry-run]
Copies lecture.pdf to the matching Obsidian School/.../COURSE/Lectures folder
using the Markdown title when present, otherwise Lecture NUMBER.pdf.
Removes the corresponding Markdown note only after the PDF copy is verified.
Repeat exports retain the existing PDF title.
DIRECTORY defaults to the current lecture directory (COURSE/lecture-NN).
Finds vaults through Obsidian's registry and home-folder discovery on Windows/Linux.
--vault overrides NOTES_OBSIDIAN_VAULT and automatic discovery.
--term selects a semester folder when a course appears more than once.
--dry-run displays the resolved paths without copying or deleting anything.
The source lecture.pdf and editable lecture directory are preserved.
Example: notes export COE718/lecture-03 --dry-run
`

func runExport(args []string, cwd string, streams IO) int {
	fail := func(code int, err error) int { fmt.Fprintln(streams.Err, "notes:", err); return code }
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		fmt.Fprint(streams.Out, exportHelp)
		return 0
	}
	options := lecture.ExportOptions{Vault: os.Getenv("NOTES_OBSIDIAN_VAULT")}
	path, dry := "", false
	seen := make(map[string]bool)
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--dry-run":
			dry = true
		case "--vault", "--term":
			if seen[arg] || i+1 == len(args) || args[i+1] == "" || strings.HasPrefix(args[i+1], "--") {
				return fail(2, fmt.Errorf("%s requires one value; %s", arg, exportHelp))
			}
			seen[arg] = true
			i++
			if arg == "--vault" {
				options.Vault = args[i]
			} else {
				options.Term = args[i]
			}
		default:
			if strings.HasPrefix(arg, "-") || path != "" {
				return fail(2, fmt.Errorf("unexpected argument %q; %s", arg, exportHelp))
			}
			path = arg
		}
	}
	if path == "" {
		path = "."
	}
	if !filepath.IsAbs(path) {
		if filepath.VolumeName(path) != "" || os.IsPathSeparator(path[0]) {
			return fail(2, fmt.Errorf("use a fully qualified lecture directory"))
		}
		path = cwd + string(os.PathSeparator) + path
	}
	if options.Vault != "" && !filepath.IsAbs(options.Vault) {
		if filepath.VolumeName(options.Vault) != "" || os.IsPathSeparator(options.Vault[0]) {
			return fail(2, fmt.Errorf("use a fully qualified vault path"))
		}
		options.Vault = cwd + string(os.PathSeparator) + options.Vault
	}
	plan, err := lecture.PlanExport(path, options)
	if err != nil {
		return fail(1, err)
	}
	defer plan.Close()
	if _, err = fmt.Fprintf(streams.Out, "Source: %s\nPDF destination: %s\n", plan.Source, plan.Destination); err != nil {
		return fail(1, err)
	}
	if plan.Markdown != "" {
		if _, err = fmt.Fprintf(streams.Out, "Remove after verified copy: %s\n", plan.Markdown); err != nil {
			return fail(1, err)
		}
	}
	if dry {
		fmt.Fprintln(streams.Out, "Dry run. No files changed.")
		return 0
	}
	if err = plan.Apply(); err != nil {
		return fail(1, err)
	}
	fmt.Fprintln(streams.Out, "Exported successfully.")
	return 0
}
