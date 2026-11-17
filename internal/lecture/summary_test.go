package lecture

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestSummaryStyleUpdatePreservesLegacyCommand(t *testing.T) {
	dir, err := New(t.TempDir(), "SUMMARY", "1")
	if err != nil {
		t.Fatal(err)
	}
	old, err := assets.ReadFile("assets/styles/8c7f90b.sty")
	if err != nil {
		t.Fatal(err)
	}
	legacy := `\newcommand{\notesummary}[3]{#1\par#2\par#3}`
	old = append(old, []byte("\n"+legacy+"\n")...)
	writeFixture(t, dir, "lecturenotes.sty", string(old))
	plans, err := PreviewStyles(dir, "")
	if err != nil || len(plans) != 1 {
		t.Fatal(plans, err)
	}
	if plans[0].Status != "update" || plans[0].From != "0.2.0" || plans[0].To != StyleVersion {
		t.Fatalf("unexpected migration: %+v", plans[0])
	}
	if _, err := ApplyStyle(plans[0]); err != nil {
		t.Fatal(err)
	}
	updated, err := os.ReadFile(filepath.Join(dir, "lecturenotes.sty"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{legacy, `\NewDocumentEnvironment{notesummarybox}`, "v" + StyleVersion} {
		if !bytes.Contains(updated, []byte(want)) {
			t.Fatalf("migration lost %q", want)
		}
	}
	if bytes.Count(updated, []byte(`\phantomsection`)) != bytes.Count(old, []byte(`\phantomsection`)) {
		t.Fatal("updating a v0.2.0 style changed existing heading anchors")
	}
	plans, err = PreviewStyles(dir, "")
	if err != nil || len(plans) != 1 || plans[0].Status != "up-to-date" {
		t.Fatal(plans, err)
	}
}

func TestSummaryBoxLayout(t *testing.T) {
	if _, err := exec.LookPath("pdftotext"); err != nil {
		t.Skip("pdftotext is not installed")
	}
	dir, err := New(t.TempDir(), "SUMMARY", "1")
	if err != nil {
		t.Fatal(err)
	}
	log := latexBuild(t, dir, `\documentclass{article}
\usepackage{lecturenotes}
\newcommand{\notesummary}[3]{#1\par#2\par#3}
\begin{document}
\begin{notesummarybox}
Default recap paragraph.

Second recap paragraph.
\[
\begin{aligned}
H(f) &= k S^*(f)e^{-j2\pi fT} \\[6pt]
h(t) &= k s(T-t)
\end{aligned}
\]
\begin{itemize}\item A recap can include a list.\end{itemize}
\end{notesummarybox}
\begin{notesummarybox}[Matched filter recap]
A custom title replaces Summary.
\end{notesummarybox}
\notesummary{Legacy title}{Legacy overview}{Legacy rows}
\clearpage
\vspace*{0.82\textheight}
Boundary preface.
\begin{notesummarybox}[Page boundary recap]
SUMMARY-START\par
\rule{0pt}{0.2\textheight}\par
SUMMARY-END
\end{notesummarybox}
\end{document}`)
	for _, bad := range []string{"Overfull", "Underfull", "LaTeX Warning", "Package hyperref Warning"} {
		if strings.Contains(log, bad) {
			t.Fatalf("%s in summary fixture:\n%s", bad, log)
		}
	}
	cmd := exec.Command("pdftotext", "-layout", "lecture.pdf", "-")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, want := range []string{"Summary", "Default recap paragraph.", "Second recap paragraph.", "Matched filter recap", "Legacy overview", "Legacy rows"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in compiled recap:\n%s", want, text)
		}
	}
	var startPage, endPage, prefacePage int = -1, -1, -1
	for page, content := range strings.Split(text, "\f") {
		if strings.Contains(content, "SUMMARY-START") {
			startPage = page
		}
		if strings.Contains(content, "SUMMARY-END") {
			endPage = page
		}
		if strings.Contains(content, "Boundary preface.") {
			prefacePage = page
		}
	}
	if startPage < 0 || startPage != endPage || startPage != prefacePage+1 {
		t.Fatalf("recap did not move intact to the next page: preface=%d start=%d end=%d", prefacePage, startPage, endPage)
	}
}
