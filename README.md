# Typesetting Scripts

Fast, consistent after-lecture summaries for engineering courses. Edit LaTeX
in VS Code and reuse a small formatting package for equations, assumptions,
tables, definitions, notation, derivations, code, circuits, and signal plots.
This is a lecture-summary workflow. Problem sets will not be typeset; the aim
is to keep summarizing lecture material quick and consistent.

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

The package uses `fontenc`, `lmodern`, `mathpazo`, `helvet`, `microtype`,
`geometry`, `amsmath`, `amssymb`, `xcolor`,
`booktabs`, `tabularx`, `fancyhdr`, `tcolorbox`, `circuitikz`, `pgfplots`, and
`xparse`, `needspace`, and `listings`. A minimal TeX installation may need additional packages. No shell
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

The modern-textbook style pairs Palatino body text and mathematics with crisp
sans-serif headings, dark ink, and a muted teal accent. Key equations receive
a pale background; definitions use a slim rule, assumptions stay inline, and
derivations sit on white. Tables use fine horizontal rules and code uses a
neutral background. The fonts are supplied by standard TeX packages and require
no operating-system font installation.

Change `NoteAccent`, `NoteTint`, `NoteInk`, `NoteMuted`, `NoteRule`, and `NoteCode`
in the local `.sty` file to adjust the palette.

### Topic divider

Use one divider for each topic or subtopic covered in a lecture. It creates an
unnumbered dark heading and a short teal rule, with space reserved for the
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
with `\\`. Column padding and fine horizontal rules come from the package.

```latex
\begin{notetable}{lXr}{\textbf{Symbol} & \textbf{Meaning} & \textbf{Value}}
$R$ & Resistance & $1\,\mathrm{k}\Omega$ \\
$C$ & Capacitance & $100\,\mathrm{nF}$ \\
\end{notetable}
```

### Definition and notation

Use a definition block for a concept and the notation table for its symbols.
The notation table supplies the three column headings automatically.

```latex
\notedefinition{Sampling period}{The time $T_s$ between successive samples.}

\begin{notation}
$T_s$ & Sampling period & $\mathrm{s}$ \\
$f_s$ & Sampling frequency & $\mathrm{Hz}$ \\
$n$ & Sample index & dimensionless \\
\end{notation}
```

### Derivation

The body is an unnumbered `align` environment: align on `&`, separate lines
with `\\`, and add explanations using `\text{...}`. Do not wrap it in another
math environment. Keep each derivation short enough to fit a page; split a long
derivation into successive blocks at a logical step.

```latex
\begin{notederivation}{RC transfer function}
  V_{\mathrm{in}} &= RI + V_{\mathrm{out}} && \text{voltage law} \\
  I &= sC V_{\mathrm{out}} && \text{zero initial conditions} \\
  H(s) &= \frac{1}{1+sRC} && \text{collect terms}
\end{notederivation}
```

### Code

`notecode` preserves indentation and uses a subtle background with matching
syntax accents. C is the default; C++, a small ARM instruction vocabulary, and
plain text are also supported. ARM highlighting is not instruction validation.
Write code literally: do not escape `_`, `#`, braces, or `%` as LaTeX.
No Python, Pygments, or shell escape is needed.

```latex
\begin{notecode}[language=C]
float gain(float voltage_in, float scale)
{
    return voltage_in * scale;
}
\end{notecode}

\begin{notecode}[language=ARM]
MOVS r0, #1
LSLS r0, r0, #5
\end{notecode}
```

Use `language=C++` for C++, `language={}` for plain text, or add
`numbers=left` for optional line numbers. Keep code environments directly in
the document rather than inside arguments to other commands.

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
