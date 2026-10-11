package lecture

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func writeFixture(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
func historicalStyle(t *testing.T) string {
	t.Helper()
	data, err := assets.ReadFile("assets/styles/local-v0.1.7.sty")
	if err != nil {
		t.Fatal(err)
	}
	// Git may check embedded styles out with CRLF on Windows. Build fixtures
	// from LF first so the CRLF case does not introduce doubled carriage returns.
	return strings.ReplaceAll(string(data), "\r\n", "\n")
}

func TestSourceCheckGraphAndLocations(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "lecture.tex", `% TODO: add an example
\graphicspath{{images/}}
\input{part}
\includegraphics{present}
\notefigure[label={fig:standard}]{images/present.png}{Standard}
\ref{fig:standard} \eqref{eq:included}
\label{fig:}
\label{fig:}
\ref{absent}
\includegraphics{missing}
\input{missing-input}
\begin{notecode}
% TODO: this is code, not an unfinished note
\label{fig:} \includegraphics{not-real}
\end{notecode}
\verb|\ref{fake}|
% \label{fig:} \includegraphics{commented}
\newcommand{\myfigure}[1]{\includegraphics{#1}}
Add info for the S-suffix here.
`)
	writeFixture(t, dir, "part.tex", `\label{eq:included}`)
	writeFixture(t, dir, "images/present.png", "exists")
	diagnostics, count, err := Check(dir, CheckOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 || len(diagnostics) != 8 {
		t.Fatalf("count=%d diagnostics=%+v", count, diagnostics)
	}
	want := map[string][]int{"todo": {1, 19}, "placeholder-label": {7, 8}, "duplicate-label": {8}, "undefined-reference": {9}, "missing-image": {10}, "missing-input": {11}}
	for kind, locations := range want {
		var got []int
		for _, d := range diagnostics {
			if d.Kind == kind {
				got = append(got, d.Line)
			}
		}
		if fmtInts(got) != fmtInts(locations) {
			t.Fatalf("%s lines=%v want=%v", kind, got, locations)
		}
	}
}
func fmtInts(a []int) string {
	var b strings.Builder
	for _, v := range a {
		b.WriteString(strconv.Itoa(v) + ",")
	}
	return b.String()
}

func TestSourceCheckScopesAndCycles(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"COURSE/lecture-01", "COURSE/lecture-02"} {
		writeFixture(t, root, name+"/lecture.tex", `\label{same}\ref{same}`)
	}
	writeFixture(t, root, "COURSE/lecture-03/lecture.pdf", "finalized")
	diagnostics, count, err := Check(root, CheckOptions{})
	if err != nil || count != 2 || len(diagnostics) != 0 {
		t.Fatal(count, diagnostics, err)
	}
	writeFixture(t, root, "COURSE/lecture-01/part.tex", `\input{lecture}`)
	writeFixture(t, root, "COURSE/lecture-01/lecture.tex", `\input{part}`)
	diagnostics, _, err = Check(root, CheckOptions{})
	if err != nil || len(diagnostics) != 1 || diagnostics[0].Kind != "input-cycle" {
		t.Fatal(diagnostics, err)
	}
}

func TestStyleUpdatePreservesCustomizationsAndBackups(t *testing.T) {
	for _, crlf := range []bool{false, true} {
		t.Run(strconv.FormatBool(crlf), func(t *testing.T) {
			dir := t.TempDir()
			writeFixture(t, dir, "lecture.tex", "unchanged lecture")
			local := strings.Replace(historicalStyle(t), "{246A70}", "{123456}", 1)
			local = strings.Replace(local, `\RequirePackage{listings}`, `\RequirePackage{listings}`+"\n"+`\RequirePackage[hidelinks,bookmarksdepth=3,pdfpagemode=UseOutlines]{hyperref}`, 1)
			local = strings.ReplaceAll(local, "  \\addcontentsline", "  \\phantomsection\n  \\addcontentsline")
			local += "\n\\newcommand{\\notesummary}[1]{Custom summary: #1}\n"
			if crlf {
				local = strings.ReplaceAll(local, "\n", "\r\n")
			}
			writeFixture(t, dir, "lecturenotes.sty", local)
			writeFixture(t, dir, "lecturenotes.sty.bak", "older backup")
			plans, err := PreviewStyles(dir, "")
			if err != nil {
				t.Fatal(err)
			}
			if len(plans) != 1 || plans[0].Status != "update" || !plans[0].Migration {
				t.Fatalf("%+v", plans)
			}
			before, _ := os.ReadFile(filepath.Join(dir, "lecturenotes.sty"))
			if string(before) != local {
				t.Fatal("preview wrote the style")
			}
			backup, err := ApplyStyle(plans[0])
			if err != nil {
				t.Fatal(err)
			}
			if filepath.Base(backup) != "lecturenotes.sty.bak.1" {
				t.Fatal(backup)
			}
			b, _ := os.ReadFile(backup)
			if string(b) != local {
				t.Fatal("backup changed bytes")
			}
			b, _ = os.ReadFile(filepath.Join(dir, "lecturenotes.sty.bak"))
			if string(b) != "older backup" {
				t.Fatal("overwrote earlier backup")
			}
			b, _ = os.ReadFile(filepath.Join(dir, "lecture.tex"))
			if string(b) != "unchanged lecture" {
				t.Fatal("changed lecture content")
			}
			updated, _ := os.ReadFile(filepath.Join(dir, "lecturenotes.sty"))
			for _, preserved := range []string{"{123456}", `\newcommand{\notesummary}`, `\hypersetup{hidelinks,bookmarksdepth=3,pdfpagemode=UseOutlines}`, `\NewDocumentEnvironment{notepanels}`, "v0.2.0"} {
				if !bytes.Contains(updated, []byte(preserved)) {
					t.Fatalf("lost %q", preserved)
				}
			}
			if crlf && strings.Contains(strings.ReplaceAll(string(updated), "\r\n", ""), "\n") {
				t.Fatal("changed CRLF line endings")
			}
			plans, err = PreviewStyles(dir, "")
			if err != nil || plans[0].Status != "up-to-date" {
				t.Fatal(plans, err)
			}
		})
	}
}

func TestStyleUpdateConflictChoicesAndStalePreview(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "lecture.tex", "content")
	local := strings.Replace(historicalStyle(t), "[color=NoteInk,#1]", "[color=red,#1]", 1)
	writeFixture(t, dir, "lecturenotes.sty", local)
	plans, err := PreviewStyles(dir, "")
	if err != nil || plans[0].Status != "conflict" {
		t.Fatal(plans, err)
	}
	if _, err := ApplyStyle(plans[0]); err == nil {
		t.Fatal("applied a conflict")
	}
	for _, choice := range []string{"keep", "replace"} {
		plans, err = PreviewStyles(dir, choice)
		if err != nil || plans[0].Status != "update" {
			t.Fatal(plans, err)
		}
		want := "[color=red,#1]"
		retained := 1
		if choice == "replace" {
			want = "[color=NoteInk,note-circuit,#1]"
			retained = 0
		}
		if plans[0].CustomEdits != retained {
			t.Fatalf("%s: retained %d edits, want %d", choice, plans[0].CustomEdits, retained)
		}
		if !bytes.Contains(plans[0].after, []byte(want)) {
			t.Fatalf("%s did not choose %s", choice, want)
		}
	}
	plans, _ = PreviewStyles(dir, "replace")
	writeFixture(t, dir, "lecturenotes.sty", local+"% later edit\n")
	if _, err := ApplyStyle(plans[0]); err == nil {
		t.Fatal("overwrote a later edit")
	}
	if _, err := os.Stat(filepath.Join(dir, "lecturenotes.sty.bak")); !os.IsNotExist(err) {
		t.Fatal("created a backup after stale preview")
	}
}

func TestStyleUpdateUnknownAndLinkedStyles(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "lecture.tex", "source")
	writeFixture(t, dir, "lecturenotes.sty", `\ProvidesPackage{lecturenotes}[2026/10/10 v9.9.9 Unknown]`)
	plans, err := PreviewStyles(dir, "replace")
	if err != nil || plans[0].Status != "conflict" {
		t.Fatal(plans, err)
	}
	target := filepath.Join(t.TempDir(), "style.sty")
	writeFixture(t, filepath.Dir(target), "style.sty", historicalStyle(t))
	if err := os.Remove(filepath.Join(dir, "lecturenotes.sty")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(dir, "lecturenotes.sty")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := PreviewStyles(dir, ""); err == nil {
		t.Fatal("accepted a linked style")
	}
}

func latexBuild(t *testing.T, dir, source string) string {
	t.Helper()
	compiler, err := exec.LookPath("pdflatex")
	if err != nil {
		t.Skip("pdflatex is not installed")
	}
	writeFixture(t, dir, "lecture.tex", source)
	for pass := 0; pass < 2; pass++ {
		cmd := exec.Command(compiler, "-interaction=nonstopmode", "-halt-on-error", "-file-line-error", "lecture.tex")
		cmd.Dir = dir
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("pdflatex: %v\n%s", err, output)
		}
	}
	log, _ := os.ReadFile(filepath.Join(dir, "lecture.log"))
	return string(log)
}

func TestFormattingFeatures(t *testing.T) {
	if _, err := exec.LookPath("pdflatex"); err != nil {
		t.Skip("pdflatex is not installed")
	}
	if _, err := exec.LookPath("pdftotext"); err != nil {
		t.Skip("pdftotext is not installed")
	}
	dir, err := New(t.TempDir(), "FEATURES", "1")
	if err != nil {
		t.Fatal(err)
	}
	im := image.NewRGBA(image.Rect(0, 0, 120, 40))
	for y := 0; y < 40; y++ {
		for x := 0; x < 120; x++ {
			im.Set(x, y, color.RGBA{36, 106, 112, 255})
		}
	}
	file, err := os.Create(filepath.Join(dir, "image.png"))
	if err != nil {
		t.Fatal(err)
	}
	if err = png.Encode(file, im); err != nil {
		t.Fatal(err)
	}
	file.Close()
	source := `\documentclass{article}
\usepackage{lecturenotes}
\notesetup{toc-depth=2,bookmark-depth=3,circuit-preset=compact}
\begin{document}\tableofcontents\clearpage
\noteheading{1}{Main topic}\noteheading{2}{Subtopic}
\noteheading[bookmark={Output resistance ro}]{3}{Output resistance $r_o$}
\noteheading{4}{Fourth level}\noteheading{5}{Fifth level}
\begin{notepanels}[columns=2,widths={2,1},gap=4mm]
\begin{notepanel}{Wide}\typeout{PANELWIDTH=\the\linewidth} Wide text.\end{notepanel}
\begin{notepanel}{Narrow}\typeout{PANELWIDTH=\the\linewidth} Narrow text.\end{notepanel}
\end{notepanels}
\begin{notepanels}[columns=3,align=center,keep-together=false]
\begin{notepanel}{One}\typeout{PANELWIDTH=\the\linewidth}One\end{notepanel}
\begin{notepanel}{Two}\typeout{PANELWIDTH=\the\linewidth}Two\end{notepanel}
\begin{notepanel}{Three}\typeout{PANELWIDTH=\the\linewidth}Three\end{notepanel}
\end{notepanels}
\begin{notederivation}{Legacy}x &= y && \quad\text{Equality}\end{notederivation}
\begin{notederivation}[explanations=below]{Wrapping}\derivestep{x &= y}{A longer explanatory paragraph wraps as ordinary text below the equation, so it does not create a wide unbreakable math annotation.}\end{notederivation}
\begin{notederivation}[explanation=beside]{Beside}\derivestep{x &= y}{This paragraph wraps beside the equation in a narrower column, with the text and equation aligned at the top.}\end{notederivation}
\notefigure[width=40mm,maxheight=15mm,label=fig:inline]{image}{An inline figure.}
\notefigure[placement=htbp,width=40mm,maxheight=15mm,label=fig:float]{image.png}{A floating figure.}
\begin{notecircuit}[scale=0.8]{Circuit override}\draw (0,0) to[R] (2,0);\draw[note-current] (0,0.5)--(1,0.5);\end{notecircuit}
See Figures~\ref{fig:inline} and~\ref{fig:float}.
\clearpage
`
	for i := 0; i < 35; i++ {
		source += `\begin{notederivation}[mode=steps]{}` + `\derivestep{x=y}{START` + strconv.Itoa(i) + ` A complete step stays together while long derivations can break between steps. This explanatory text should wrap cleanly on the page. END` + strconv.Itoa(i) + `}\end{notederivation}
`
	}
	source += `\end{document}`
	log := latexBuild(t, dir, source)
	for _, bad := range []string{"Overfull", "Underfull", "LaTeX Warning", "Package hyperref Warning", "Package caption Warning"} {
		if strings.Contains(log, bad) {
			t.Fatalf("%s in formatting fixture:\n%s", bad, log)
		}
	}
	matches := regexp.MustCompile(`PANELWIDTH=([0-9.]+)pt`).FindAllStringSubmatch(log, -1)
	if len(matches) != 5 {
		t.Fatal(matches)
	}
	wide, _ := strconv.ParseFloat(matches[0][1], 64)
	narrow, _ := strconv.ParseFloat(matches[1][1], 64)
	if wide/narrow < 1.999 || wide/narrow > 2.001 {
		t.Fatal("unequal panel widths", matches)
	}
	if matches[2][1] != matches[3][1] || matches[3][1] != matches[4][1] {
		t.Fatal("unequal three-column row", matches)
	}
	out, _ := os.ReadFile(filepath.Join(dir, "lecture.out"))
	if !bytes.Contains(out, []byte(`\BOOKMARK [3]`)) || bytes.Contains(out, []byte(`\BOOKMARK [4]`)) || bytes.Contains(out, []byte(`\BOOKMARK [5]`)) {
		t.Fatalf("bookmark depth incorrect: %s", out)
	}
	cmd := exec.Command("pdftotext", "-f", "1", "-l", "1", "lecture.pdf", "-")
	cmd.Dir = dir
	text, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(text, []byte("Main topic")) || bytes.Contains(text, []byte("Output resistance")) {
		t.Fatalf("printed contents depth incorrect: %s", text)
	}
	cmd = exec.Command("pdftotext", "lecture.pdf", "-")
	cmd.Dir = dir
	text, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	// Windows pdftotext writes CRLF; page and step boundaries are equivalent.
	pages := strings.Split(strings.ReplaceAll(string(text), "\r\n", "\n"), "\f")
	if len(pages) < 4 {
		t.Fatal("steps did not paginate")
	}
	for i := 0; i < 35; i++ {
		start, end := "START"+strconv.Itoa(i)+" ", "END"+strconv.Itoa(i)+"\n"
		for _, page := range pages {
			if strings.Contains(page, start) && !strings.Contains(page, end) {
				t.Fatalf("split step %d", i)
			}
		}
	}
}

func TestCompileCheckPreservesExistingOutputs(t *testing.T) {
	if _, err := exec.LookPath("pdflatex"); err != nil {
		t.Skip("pdflatex is not installed")
	}
	dir, err := New(t.TempDir(), "CHECK", "1")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"lecture.pdf", "lecture.aux", "lecture.log"} {
		writeFixture(t, dir, name, "existing "+name)
	}
	diagnostics, _, err := Check(dir, CheckOptions{Compile: true})
	if err != nil || len(diagnostics) != 0 {
		t.Fatal(diagnostics, err)
	}
	for _, name := range []string{"lecture.pdf", "lecture.aux", "lecture.log"} {
		data, _ := os.ReadFile(filepath.Join(dir, name))
		if string(data) != "existing "+name {
			t.Fatal("changed", name)
		}
	}
}
