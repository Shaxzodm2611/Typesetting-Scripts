package lecture

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const StyleVersion = "0.2.0"

type StyleUpdate struct {
	Directory, From, To, Status string
	CustomEdits                 int
	Conflicts                   []string
	Migration                   bool
	before, after               []byte
	mode                        os.FileMode
}

type lineEdit struct {
	start, end int
	lines      []string
}

var packageVersion = regexp.MustCompile(`\\ProvidesPackage\{lecturenotes\}\[[^\]\n]*\bv([0-9]+\.[0-9]+\.[0-9]+)\b`)
var legacyHyperref = regexp.MustCompile(`(?s)% Load after the other packages so contents entries also become PDF bookmarks\.\n\\RequirePackage\[([^]]+)\]\{hyperref\}\n|\\RequirePackage\[([^]]+)\]\{hyperref\}\n`)
var declaredResource = regexp.MustCompile(`\\(?:NewDocumentCommand|RenewDocumentCommand|NewDocumentEnvironment|RenewDocumentEnvironment|newcommand|renewcommand|providecommand|newenvironment|definecolor|RequirePackage)\s*\{([^}]+)\}`)
var newDefinition = regexp.MustCompile(`\\(?:NewDocumentCommand|NewDocumentEnvironment|newcommand|newenvironment)\s*\{([^}]+)\}`)

func validateMergedStyle(source string) error {
	masked := maskTeX(source)
	depth := 0
	for i := 0; i < len(masked); i++ {
		if masked[i] == '\\' {
			i++
			continue
		}
		if masked[i] == '{' {
			depth++
		}
		if masked[i] == '}' {
			depth--
			if depth < 0 {
				return fmt.Errorf("merged definitions have unmatched braces; merge manually")
			}
		}
	}
	if depth != 0 {
		return fmt.Errorf("merged definitions have unmatched braces; merge manually")
	}
	seen := map[string]bool{}
	for _, m := range newDefinition.FindAllStringSubmatch(masked, -1) {
		if seen[m[1]] {
			return fmt.Errorf("merged definitions declare %s twice; merge this customization manually", m[1])
		}
		seen[m[1]] = true
	}
	return nil
}

func styleVersion(data []byte) string {
	m := packageVersion.FindSubmatch(data)
	if m == nil {
		return "unknown"
	}
	return string(m[1])
}

// These legacy additions are now implemented by the package itself. Convert
// the old hyperref load options into late settings instead of double-loading
// it with incompatible options. Other custom edits still go through the merge.
func migrateLegacy(data string) (string, bool) {
	var settings []string
	data = legacyHyperref.ReplaceAllStringFunc(data, func(s string) string {
		m := legacyHyperref.FindStringSubmatch(s)
		options := m[1]
		if options == "" {
			options = m[2]
		}
		settings = append(settings, `\hypersetup{`+strings.TrimSpace(options)+`}`)
		return ""
	})
	migrated := len(settings) > 0
	data = strings.ReplaceAll(data, "% Record five unnumbered heading levels in the contents and PDF outline.", "% Record all five note-heading levels without numbering their printed titles.")
	for _, prefix := range []string{
		"\\NewDocumentCommand{\\notesection}{m}{%\n  \\par\\addvspace{15pt}\\Needspace{5\\baselineskip}%\n",
		"\\NewDocumentCommand{\\subnotesection}{m}{%\n  \\par\\addvspace{8pt}\\Needspace{3\\baselineskip}%\n",
		"\\newcommand{\\note@minorheading}[7]{%\n  \\par\\addvspace{#1}\\Needspace{3\\baselineskip}%\n",
	} {
		data = strings.ReplaceAll(data, prefix+"  % Anchor after Needspace so links land on the heading if it moves to a new page.\n  \\phantomsection\n", prefix)
		data = strings.ReplaceAll(data, prefix+"  \\phantomsection\n", prefix)
	}
	if len(settings) > 0 {
		data = strings.TrimRight(data, "\n") + "\n" + strings.Join(settings, "\n") + "\n"
	}
	return data, migrated
}

func lines(s string) []string { return strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n") }

// LCS yields ordered, non-overlapping changes measured against a shared base.
// Style packages are small; no external diff/merge executable is required.
func diffLines(base, changed []string) []lineEdit {
	dp := make([][]int, len(base)+1)
	for i := range dp {
		dp[i] = make([]int, len(changed)+1)
	}
	for i := len(base) - 1; i >= 0; i-- {
		for j := len(changed) - 1; j >= 0; j-- {
			if base[i] == changed[j] {
				dp[i][j] = dp[i+1][j+1] + 1
			} else if dp[i+1][j] > dp[i][j+1] {
				dp[i][j] = dp[i+1][j]
			} else {
				dp[i][j] = dp[i][j+1]
			}
		}
	}
	var edits []lineEdit
	i, j := 0, 0
	for i < len(base) || j < len(changed) {
		if i < len(base) && j < len(changed) && base[i] == changed[j] {
			i++
			j++
			continue
		}
		e := lineEdit{start: i}
		for i < len(base) || j < len(changed) {
			if i < len(base) && j < len(changed) && base[i] == changed[j] {
				break
			}
			if j < len(changed) && (i == len(base) || dp[i][j+1] >= dp[i+1][j]) {
				e.lines = append(e.lines, changed[j])
				j++
			} else {
				i++
			}
		}
		e.end = i
		edits = append(edits, e)
	}
	return edits
}
func editCost(edits []lineEdit) int {
	n := 0
	for _, e := range edits {
		n += e.end - e.start + len(e.lines)
	}
	return n
}
func editsOverlap(a, b lineEdit) bool {
	if a.start == a.end && b.start == b.end {
		return a.start == b.start
	}
	if a.start == a.end {
		return a.start > b.start && a.start < b.end
	}
	if b.start == b.end {
		return b.start > a.start && b.start < a.end
	}
	return a.start < b.end && b.start < a.end
}
func sameEdit(a, b lineEdit) bool {
	return a.start == b.start && a.end == b.end && strings.Join(a.lines, "\n") == strings.Join(b.lines, "\n")
}
func disjointDeclarations(a, b []string) bool {
	resources := map[string]bool{}
	for _, m := range declaredResource.FindAllStringSubmatch(strings.Join(a, "\n"), -1) {
		for _, name := range strings.Split(m[1], ",") {
			resources[name] = true
		}
	}
	for _, m := range declaredResource.FindAllStringSubmatch(strings.Join(b, "\n"), -1) {
		for _, name := range strings.Split(m[1], ",") {
			if resources[name] {
				return false
			}
		}
	}
	return true
}

func mergeStyle(base, local, incoming string, resolution string) (string, int, []string) {
	b := lines(base)
	ours, theirs := diffLines(b, lines(local)), diffLines(b, lines(incoming))
	keepOurs, keepTheirs := make([]bool, len(ours)), make([]bool, len(theirs))
	for i := range keepOurs {
		keepOurs[i] = true
	}
	for i := range keepTheirs {
		keepTheirs[i] = true
	}
	var conflicts []string
	for i, o := range ours {
		for j, n := range theirs {
			if sameEdit(o, n) {
				keepOurs[i] = false
				continue
			}
			if !editsOverlap(o, n) {
				continue
			}
			if o.start == o.end && n.start == n.end && disjointDeclarations(o.lines, n.lines) {
				continue
			}
			conflicts = append(conflicts, fmt.Sprintf("base lines %d-%d: local customization overlaps updated formatting", o.start+1, max(o.start+1, o.end)))
			switch resolution {
			case "keep":
				keepTheirs[j] = false
			case "replace":
				keepOurs[i] = false
			}
		}
	}
	if len(conflicts) > 0 && resolution == "" {
		return "", len(ours), conflicts
	}
	var edits []lineEdit
	retained := 0
	for i, e := range theirs {
		if keepTheirs[i] {
			edits = append(edits, e)
		}
	}
	for i, e := range ours {
		if keepOurs[i] {
			edits = append(edits, e)
			retained++
		}
	}
	// Add incoming insertions before local insertions at the same position.
	sort.SliceStable(edits, func(i, j int) bool {
		if edits[i].start != edits[j].start {
			return edits[i].start < edits[j].start
		}
		return edits[i].end < edits[j].end
	})
	var result []string
	pos := 0
	for _, e := range edits {
		if e.start < pos {
			return "", len(ours), append(conflicts, "overlapping changes require a manual merge")
		}
		result = append(result, b[pos:e.start]...)
		result = append(result, e.lines...)
		pos = e.end
	}
	result = append(result, b[pos:]...)
	return strings.Join(result, "\n"), retained, conflicts
}

// PreviewStyles performs no writes. Resolve selects only conflicting hunks;
// compatible local additions and settings are preserved with either choice.
func PreviewStyles(path, resolve string) ([]StyleUpdate, error) {
	if resolve != "" && resolve != "keep" && resolve != "replace" {
		return nil, fmt.Errorf("resolve must be keep or replace")
	}
	dirs, err := FindEditable(path)
	if err != nil {
		return nil, err
	}
	current, err := assets.ReadFile("assets/lecturenotes.sty")
	if err != nil {
		return nil, err
	}
	entries, err := assets.ReadDir("assets/styles")
	if err != nil {
		return nil, err
	}
	baselines := [][]byte{current}
	for _, e := range entries {
		data, err := assets.ReadFile("assets/styles/" + e.Name())
		if err != nil {
			return nil, err
		}
		baselines = append(baselines, data)
	}
	var plans []StyleUpdate
	for _, dir := range dirs {
		local, info, err := readOrdinary(filepath.Join(dir, "lecturenotes.sty"))
		if err != nil {
			return nil, err
		}
		plan := StyleUpdate{Directory: dir, From: styleVersion(local), To: StyleVersion, Status: "update", before: local, mode: info.Mode().Perm()}
		normal := strings.ReplaceAll(string(local), "\r\n", "\n")
		if plan.From != "unknown" && plan.From != StyleVersion {
			normal, plan.Migration = migrateLegacy(normal)
		}
		base := ""
		cost := int(^uint(0) >> 1)
		for _, candidate := range baselines {
			if styleVersion(candidate) != plan.From {
				continue
			}
			candidateCost := editCost(diffLines(lines(string(candidate)), lines(normal)))
			if candidateCost < cost {
				cost = candidateCost
				base = string(candidate)
			}
		}
		if base == "" {
			plan.Status = "conflict"
			plan.Conflicts = []string{"no bundled baseline for this style version; merge manually"}
			plans = append(plans, plan)
			continue
		}
		merged, custom, conflicts := mergeStyle(base, normal, string(current), resolve)
		if merged != "" {
			if err := validateMergedStyle(merged); err != nil {
				conflicts = append(conflicts, err.Error())
				merged = ""
			}
		}
		plan.CustomEdits = custom
		plan.Conflicts = conflicts
		if merged == "" {
			plan.Status = "conflict"
		} else {
			if bytes.Contains(local, []byte("\r\n")) {
				merged = strings.ReplaceAll(merged, "\n", "\r\n")
			}
			plan.after = []byte(merged)
			if bytes.Equal(local, plan.after) {
				plan.Status = "up-to-date"
			}
		}
		plans = append(plans, plan)
	}
	return plans, nil
}

// ApplyStyle verifies the preview bytes again and atomically replaces one
// style after creating an exclusive, synced backup. Existing backups are kept.
func ApplyStyle(plan StyleUpdate) (string, error) {
	if plan.Status == "up-to-date" {
		return "", nil
	}
	if plan.Status != "update" {
		return "", fmt.Errorf("unresolved style conflicts in %s", plan.Directory)
	}
	if err := rejectLinkedPath(plan.Directory); err != nil {
		return "", err
	}
	root, err := os.OpenRoot(plan.Directory)
	if err != nil {
		return "", err
	}
	defer root.Close()
	info, err := root.Lstat("lecturenotes.sty")
	if err != nil {
		return "", err
	}
	if linked(info) || !info.Mode().IsRegular() {
		return "", fmt.Errorf("style must remain an ordinary file")
	}
	actual, err := root.ReadFile("lecturenotes.sty")
	if err != nil {
		return "", err
	}
	if !bytes.Equal(actual, plan.before) {
		return "", fmt.Errorf("style changed since preview: %s; preview again", plan.Directory)
	}
	backup := ""
	for n := 0; ; n++ {
		name := "lecturenotes.sty.bak"
		if n > 0 {
			name = fmt.Sprintf("lecturenotes.sty.bak.%d", n)
		}
		f, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, plan.mode)
		if os.IsExist(err) {
			continue
		}
		if err != nil {
			return "", err
		}
		_, err = f.Write(plan.before)
		if err == nil {
			err = f.Sync()
		}
		closeErr := f.Close()
		if err == nil {
			err = closeErr
		}
		if err != nil {
			_ = root.Remove(name)
			return "", err
		}
		backup = filepath.Join(plan.Directory, name)
		break
	}
	random := make([]byte, 8)
	if _, err = rand.Read(random); err != nil {
		return backup, err
	}
	temp := fmt.Sprintf(".lecturenotes-update-%x", random)
	f, err := root.OpenFile(temp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return backup, err
	}
	defer root.Remove(temp)
	_, err = f.Write(plan.after)
	if err == nil {
		err = f.Chmod(plan.mode)
	}
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return backup, err
	}
	if err = root.Rename(temp, "lecturenotes.sty"); err != nil {
		return backup, err
	}
	return backup, nil
}
