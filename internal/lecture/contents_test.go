package lecture

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Exercise the generated template with real TeX, including a second build
// after headings and pagination change. The latex CI job supplies both tools.
func TestTableOfContents(t *testing.T) {
	compiler, err := exec.LookPath("pdflatex")
	if err != nil {
		t.Skip("pdflatex is not installed")
	}
	textTool, err := exec.LookPath("pdftotext")
	if err != nil {
		t.Skip("pdftotext is not installed")
	}
	base := filepath.Join(t.TempDir(), "My Notes")
	if err := os.Mkdir(base, 0755); err != nil {
		t.Fatal(err)
	}
	dir, err := New(base, "ECE_342", "2")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "lecture.tex")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	source := strings.Replace(string(raw), `\end{document}`, `
\noteheading{1}{Sampling \& reconstruction}
A sampled waveform.
\noteheading{2}{Spectrum $X(f)$}
Repeated frequency spectra.
\subsubnotesection{Aliasing \& replicas $f_s$}
The sampling frequency separates replicas.
\noteheading{4}{Replica overlap}
Overlapping spectra cause aliasing.
\noteheading{5}{Boundary case}
Equality at the Nyquist limit.
\clearpage
\notesection{Quantization}
Discrete amplitude levels.
\subnotesection{Error bounds}
The maximum quantization error.
\noteheading{3}{Rounding error}
Round to the nearest level.
\end{document}`, 1)
	build := func() (string, string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(source), 0644); err != nil {
			t.Fatal(err)
		}
		for pass := 0; pass < 2; pass++ {
			cmd := exec.Command(compiler, "-interaction=nonstopmode", "-halt-on-error", "-file-line-error", "lecture.tex")
			cmd.Dir = dir
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("pdflatex: %v\n%s", err, output)
			}
		}
		toc, err := os.ReadFile(filepath.Join(dir, "lecture.toc"))
		if err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command(textTool, "-f", "1", "-l", "1", "-layout", "lecture.pdf", "-")
		cmd.Dir = dir
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("pdftotext: %v\n%s", err, output)
		}
		return string(toc), strings.Join(strings.Fields(string(output)), " ")
	}
	toc, contentsPage := build()
	previous := -1
	for _, entry := range []string{
		`{section}{Sampling \& reconstruction}{2}`,
		`{subsection}{Spectrum $X(f)$}{2}`,
		`{subsubsection}{Aliasing \& replicas $f_s$}{2}`,
		`{paragraph}{Replica overlap}{2}`,
		`{subparagraph}{Boundary case}{2}`,
		`{section}{Quantization}{3}`,
		`{subsection}{Error bounds}{3}`,
		`{subsubsection}{Rounding error}{3}`,
	} {
		index := strings.Index(toc, entry)
		if index < 0 || index <= previous {
			t.Fatalf("missing, misordered, or incorrectly paginated entry %q:\n%s", entry, toc)
		}
		previous = index
	}
	if strings.Count(toc, `\contentsline`) != 8 || strings.Contains(toc, `\numberline`) {
		t.Fatalf("expected eight unnumbered entries:\n%s", toc)
	}
	for _, title := range []string{"Contents", "Sampling & reconstruction", "Spectrum", "Aliasing & replicas", "Replica overlap", "Boundary case", "Quantization", "Error bounds", "Rounding error"} {
		if !strings.Contains(contentsPage, title) {
			t.Fatalf("contents page does not show %q:\n%s", title, contentsPage)
		}
	}

	source = strings.Replace(source, `\notesection{Quantization}`, `\notesection{Amplitude quantization}`, 1)
	source = strings.Replace(source, `\subnotesection{Error bounds}`, "", 1)
	source = strings.Replace(source, `\noteheading{3}{Rounding error}`, "", 1)
	source = strings.Replace(source, `\subsubnotesection{Aliasing \& replicas $f_s$}`, `\noteheading{3}{Aliasing limits $f_s$}`, 1)
	source = strings.Replace(source, `\noteheading{2}{Spectrum $X(f)$}`, "\\clearpage\n"+`\noteheading{2}{Spectrum $X(f)$}`, 1)
	toc, contentsPage = build()
	for _, entry := range []string{
		`{section}{Amplitude quantization}{4}`,
		`{subsection}{Spectrum $X(f)$}{3}`,
		`{subsubsection}{Aliasing limits $f_s$}{3}`,
		`{paragraph}{Replica overlap}{3}`,
		`{subparagraph}{Boundary case}{3}`,
	} {
		if !strings.Contains(toc, entry) {
			t.Fatalf("contents did not update titles and page numbers:\n%s", toc)
		}
	}
	if strings.Count(toc, `\contentsline`) != 6 || strings.Contains(toc, `\numberline`) {
		t.Fatalf("expected six unnumbered entries after edits:\n%s", toc)
	}
	for _, removed := range []string{"Error bounds", "Rounding error", "Aliasing \\& replicas", "{section}{Quantization}"} {
		if strings.Contains(toc, removed) {
			t.Fatalf("contents retains removed or renamed entries:\n%s", toc)
		}
	}
	if !strings.Contains(contentsPage, "Amplitude quantization") || !strings.Contains(contentsPage, "Aliasing limits") {
		t.Fatalf("contents did not update titles and page numbers:\n%s", toc)
	}
	for _, removed := range []string{"Error bounds", "Rounding error", "Aliasing & replicas", "Quantization"} {
		if strings.Contains(contentsPage, removed) {
			t.Fatalf("rendered contents did not update:\n%s", contentsPage)
		}
	}
}
