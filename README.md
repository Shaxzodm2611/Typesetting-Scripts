# Typesetting Scripts

Fast, consistent after-lecture summaries for engineering courses. Edit LaTeX
in VS Code and reuse a small formatting package for equations, assumptions,
tables, circuits, and signal plots.

**Current status:** the formatting preview is usable now. The Go CLI and
downloadable executables have not been implemented yet. The CLI installation
and usage sections below describe the agreed interface, not an available release.

## Try the formatting now

Open [the compiled sample](examples/rc-filter/lecture.pdf), or edit
[lecture.tex](examples/rc-filter/lecture.tex). It loads the actual local
[lecturenotes.sty](examples/rc-filter/lecturenotes.sty); the formatting is not
duplicated in the sample source. The sample intentionally contains demonstration
content. New lectures will start blank with only the course/lecture header.

### 1. Install LaTeX

Install [TeX Live](https://tug.org/texlive/quickinstall.html) for Windows or Linux.
The full scheme includes the packages used here. On Linux, follow the installer
instructions to add its binary directory to PATH; the Windows installer handles
PATH setup. Restart VS Code after installation.

Check these commands in a new terminal:

```text
pdflatex --version
latexmk --version
```

The package uses `fontenc`, `lmodern`, `geometry`, `amsmath`, `amssymb`, `xcolor`,
`booktabs`, `tabularx`, `fancyhdr`, `tcolorbox`, `circuitikz`, `pgfplots`, and
`xparse`, and `needspace`. A minimal TeX installation may need additional packages. No shell
escape or external image-conversion program is needed for this sample.

### 2. Set up VS Code

Install the **LaTeX Workshop** extension (`James-Yu.latex-workshop`). It needs a
TeX distribution on PATH; its default build recipe uses `latexmk`. See the
[extension's installation guide](https://github.com/James-Yu/LaTeX-Workshop/wiki/Install).

Open `examples/rc-filter` as a folder in VS Code. Open `lecture.tex`, then run
**LaTeX Workshop: Build LaTeX project** from the Command Palette. Use
**LaTeX Workshop: View LaTeX PDF file** to preview it.

Set this in VS Code settings to keep the PDF next to the source:

```json
{
  "latex-workshop.latex.outDir": "%DIR%"
}
```

Keep the default job name, `lecture`. The planned cleanup command expects the
final file to be `lecture.pdf` in the lecture directory, not in a build subfolder.

Alternatively, compile the sample directly from its folder:

```text
pdflatex -interaction=nonstopmode -halt-on-error lecture.tex
pdflatex -interaction=nonstopmode -halt-on-error lecture.tex
```

### 3. Start a blank lecture manually today

Copy `lecturenotes.sty` into a new lecture folder and create `lecture.tex`:

```latex
\documentclass[a4paper,11pt]{article}
\usepackage{lecturenotes}
\lectureheader{ECE342}{2}
\begin{document}
\null % Ensures even an empty lecture produces its initial page.
\end{document}
```

Add your notes before `\end{document}`. Change `a4paper` to `letterpaper` if
preferred. The header repeats on subsequent pages, and there is no title page
or prefilled section structure.

## Reusable blocks

All blocks use the same muted teal accent, dark text, and subtle backgrounds.
Change `NoteAccent`, `NoteTint`, `NoteInk`, and `NoteMuted` in the local `.sty`
file to adjust the palette.

### Topic divider

Use one divider for each topic or subtopic covered in a lecture. It creates an
unnumbered teal heading and a thin matching rule, with space reserved for the
content that follows. The sample demonstrates two topics and uses matching
comment dividers to make its source easy to scan. Blank lectures do not include
any topic headings until you add them.

```latex
% ==================== Topic: frequency response ====================
\notesection{Frequency response}
% Equations, explanations, tables, or diagrams go here.

% ====================== Topic: step response =======================
\notesection{Step response}
% Next topic's notes go here.
```

### Key equation

```latex
\keyequation{Ohm's law}{V = IR}
```

The second argument is already in display math mode: do not enclose it in `$`.
For several lines, use `\begin{aligned} ... \end{aligned}` inside that argument.

### Assumption

```latex
\assumption{The amplifier operates in its linear region.}
```

### Table

The first argument specifies columns; use an `X` column for text that fills
the available width. The second argument is the header row. End each body row
with `\\`. Header shading, padding, and rules come from the package.

```latex
\begin{notetable}{lXr}{\textbf{Symbol} & \textbf{Meaning} & \textbf{Value}}
$R$ & Resistance & $1\,\mathrm{k}\Omega$ \\
$C$ & Capacitance & $100\,\mathrm{nF}$ \\
\end{notetable}
```

### Circuit

Paste CircuitikZ drawing commands inside the wrapper. Do not paste another
`circuitikz` or `tikzpicture` environment inside it.

```latex
\begin{notecircuit}
  \draw (0,0) to[R,l=$R$] (3,0);
\end{notecircuit}
```

### Signal plot

The wrapper creates both the TikZ picture and PGFPlots axis. Supply axis
options in square brackets and plotting commands in the body.

```latex
\begin{notesignal}[xlabel={$t$},ylabel={$v(t)$},xmin=0,xmax=5]
  \addplot[domain=0:5,samples=100] {1-exp(-x)};
\end{notesignal}
```

For image-assisted diagrams, a useful ChatGPT prompt is:

> Convert this image into LaTeX drawing code. For a circuit, return only the
> inner CircuitikZ commands for a `notecircuit` environment. For a signal plot,
> return PGFPlots axis options and inner plot commands for a `notesignal`
> environment. These wrappers already create their outer environments. Do not
> include a preamble or document environment. Preserve labels and units, and
> flag any values or connections that cannot be read confidently.

If you receive a complete `tikzpicture` or `circuitikz` environment instead,
paste it directly into the document, outside these wrappers. Remove any supplied
document class, preamble, or `document` environment. Check the rendered result
against the image, especially circuit connections and plot scales.

## Planned CLI installation

These instructions will apply once executable builds are available. There are
no release downloads yet. Running a prebuilt executable will require neither
Go nor Python; LaTeX is still required to compile your notes through VS Code.

### Windows

1. Obtain the Windows amd64 build and name it `notes.exe`.
2. Place it in a permanent folder, for example `%LOCALAPPDATA%\Programs\notes`.
3. Open **Edit environment variables for your account**, edit the user `Path`,
   and add that folder.
4. Open a new terminal and run `notes --help`.

### Linux

Obtain the Linux binary for your architecture (amd64 or arm64), name it `notes`,
and install it in your user binary directory:

```sh
mkdir -p "$HOME/.local/bin"
install -m 755 notes "$HOME/.local/bin/notes"
```

If that directory is not already on PATH, add the following to your shell's
startup file (`~/.bashrc` for Bash or `~/.zshrc` for Zsh), then open a new terminal:

```sh
export PATH="$HOME/.local/bin:$PATH"
```

Run `notes --help` to verify installation. Source-build instructions will be
added with the Go implementation; the current repository has no Go module yet.

## Planned CLI workflow

From the directory where you keep your course notes:

```text
notes new ECE342 2
```

This will create `ECE342/lecture-02/` containing `lecture.tex`,
`lecturenotes.sty`, and a `.notes.json` ownership marker. It will refuse to
overwrite an existing lecture folder. Course codes accept letters, digits,
hyphens, and underscores, subject to Windows filename restrictions. The lecture
number is a positive integer.

Open that folder in VS Code, edit `lecture.tex`, and build with LaTeX Workshop.
The command itself will not compile the document.

After reviewing the final PDF:

```text
notes clean ECE342/lecture-02
```

**This is finalization, not ordinary LaTeX auxiliary-file cleanup.** After
confirmation, it will permanently delete everything in that generated lecture
folder except `lecture.pdf`, including source, the `.sty` package, figures,
subdirectories, and its marker. Copy the folder elsewhere first if you want to
retain editable material. Other course/lecture folders will not be removed.

```text
notes clean ECE342/lecture-02 --yes
```

`--yes` will skip the confirmation. Cleanup will require the tool's marker and
a nonempty PDF with a PDF signature. It cannot determine whether your PDF is
up to date or visually correct. It will refuse an unmarked folder, including
this hand-authored example. After successful cleanup, the marker will be gone
and a repeated cleanup will refuse the already-finalized folder.

## Project design

See the [design specification](docs/superpowers/specs/2026-09-21-lecture-notes-design.md).
The repository origin is `https://github.com/Shaxzodm2611/Typesetting-Scripts`.
The present changes are local; configuring an origin does not publish them.
