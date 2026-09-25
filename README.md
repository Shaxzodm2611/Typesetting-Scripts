# Typesetting Scripts

Fast, consistent after-lecture summaries for engineering courses. Edit LaTeX
in VS Code and reuse a small formatting package for topic headings, note boxes,
equations, assumptions, tables, definitions, notation, derivations, code,
circuits, and signal plots.
This is a lecture-summary workflow. Problem sets will not be typeset; the aim
is to keep summarizing lecture material quick and consistent.

**Current status:** the Go CLI is implemented. Windows amd64, Linux amd64,
and Linux arm64 executables can be built from this repository. Local packaged
builds accompany this delivery; no public GitHub release has been published.

## Quick start

After [installing the executable](#cli-installation) and setting up LaTeX:

```text
notes new ECE342 2
```

Open `ECE342/lecture-02/lecture.tex` in VS Code, type your summary, and compile
with LaTeX Workshop. When you have reviewed the PDF and finished editing:

```text
notes clean ECE342/lecture-02
```

Confirm the prompt to keep only `lecture.pdf`. **This permanently deletes the
editable source, formatting package, figures, and all other contents in that
lecture folder.** Stop any automatic build/watch process before finalizing.
Keep a separate copy beforehand if you want to edit the lecture later.

## Try the formatting now

Open [the compiled sample](examples/rc-filter/lecture.pdf), or edit
[lecture.tex](examples/rc-filter/lecture.tex). It loads the actual local
[lecturenotes.sty](examples/rc-filter/lecturenotes.sty); the formatting is not
duplicated in the sample source. The sample intentionally contains demonstration
content. New lectures start blank with only the course/lecture header.

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

The package uses `fontenc`, `lmodern`, `geometry`, `amsmath`, `helvet`,
`newtxtext`, `newtxmath`, `microtype`, `xcolor`, `booktabs`, `tabularx`,
`fancyhdr`, `tcolorbox`, `circuitikz`, `pgfplots`, `xparse`, `multicol`,
`changepage`, `needspace`, and `listings`. A minimal TeX installation may need
additional packages, including `newtx` (available in `texlive-fonts-extra` on
Debian/Ubuntu). No shell escape or external image-conversion program is needed
for this sample.

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

Keep the default job name, `lecture`. The cleanup command expects the
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

The modern-textbook style pairs NewTX body text and mathematics with crisp
sans-serif headings, dark ink, and a muted teal accent. Key equations receive
a pale background; definitions use a slim rule, assumptions stay inline, and
derivations sit on white. Tables use fine horizontal rules and code uses a
neutral background. The fonts are supplied by standard TeX packages and require
no operating-system font installation.

Change `NoteAccent`, `NoteTint`, `NoteInk`, `NoteMuted`, `NoteRule`, and `NoteCode`
in the local `.sty` file to adjust the palette.

### Recent formatting additions

| Feature | Use |
| --- | --- |
| Subtopics | `\subnotesection{NMOS}` adds a smaller heading under `\notesection{Device Structure}`. |
| General note | `\notebox{Title}{Content}` creates a breakable white box with a teal left rule. |
| Equation note | `\keyequation{Title}{E = mc^2}[note]` adds a smaller, indented block below the equation. The original two-argument call still works. |
| Derivation explanation | On each `notederivation` row, `&& \qquad \text{...}` is optional; omit it when no explanation is needed. |

The style file also loads `multicol` and `changepage`, so `multicols` and
`adjustwidth` are available without separate package imports.

### Topic divider

Use one divider for each overall topic covered in a lecture. It creates an
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

### Subtopic heading

Use `\subnotesection` to separate related topics within a `\notesection`.
Its smaller accent heading has less space around it and no divider rule.

```latex
\notesection{Device Structure}
\subnotesection{NMOS}
% NMOS device structure notes go here.

\subnotesection{CMOS}
% CMOS device structure notes go here.
```

### Note box

Use `\notebox{Title}{Content}` for a general explanation or reminder. It has
a white background, a teal left rule, and a small teal title. The body accepts
paragraphs and lists, and the box can break across pages.

```latex
\notebox{Key idea}{
  The gate controls the channel between source and drain.

  \begin{itemize}
    \item Use the title to name the idea.
    \item Add details or examples in the body.
  \end{itemize}
}
```

### Key equation

```latex
\keyequation{Ohm's law}{V = IR}
```

The second argument is already in display math mode: do not enclose it in `$`.
For several lines, use `\begin{aligned} ... \end{aligned}` inside that argument.
The original two-argument call works unchanged. An optional third argument in
square brackets adds a smaller, indented note below the equation:

```latex
\keyequation{Mass-energy equivalence}{E = mc^2}[
  \begin{description}
    \item[$m$] mass
  \end{description}
]
```

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

The body uses two compact, left-aligned columns. Start each equation with `&`
and separate lines with `\\`. For any line, add an optional explanation with
`&& \qquad \text{...}` or omit that entire part. Do not wrap it in another
math environment. Keep each derivation short enough to fit a page; split a long
derivation into successive blocks at a logical step.

```latex
\begin{notederivation}{RC transfer function}
  & V_{\mathrm{in}} = RI + V_{\mathrm{out}} && \qquad \text{voltage law} \\
  & I = sC V_{\mathrm{out}} \\
  & H(s) = \frac{1}{1+sRC} && \qquad \text{collect terms}
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

Give the circuit a short title in braces, then paste CircuitikZ drawing commands
inside the wrapper. Do not paste another `circuitikz` or `tikzpicture`
environment inside it.

```latex
\begin{notecircuit}{RC low-pass topology}
  \draw (0,0) to[R,l=$R$] (3,0);
\end{notecircuit}
```

### Signal plot

The wrapper creates both the TikZ picture and PGFPlots axis. Supply axis
options in square brackets, a short title in braces, and plotting commands in
the body.

```latex
\begin{notesignal}[xlabel={$t$},ylabel={$v(t)$},xmin=0,xmax=5]{Step response}
  \addplot[domain=0:5,samples=100] {1-exp(-x)};
\end{notesignal}
```

For image-assisted diagrams, a useful ChatGPT prompt is:

> Convert this image into LaTeX drawing code. For a circuit, return only the
> inner CircuitikZ commands for a `notecircuit` environment. For a signal plot,
> return PGFPlots axis options and inner plot commands for a `notesignal`
> environment. Also suggest a concise title for the figure. These wrappers
> already create their outer environments. Do not
> include a preamble or document environment. Preserve labels and units, and
> flag any values or connections that cannot be read confidently.

If you receive a complete `tikzpicture` or `circuitikz` environment instead,
paste it directly into the document, outside these wrappers. Remove any supplied
document class, preamble, or `document` environment. Check the rendered result
against the image, especially circuit connections and plot scales.

## CLI installation

Use the local platform archive supplied with this project. Alternatively, once
the repository is pushed and its workflow has completed successfully, open
GitHub **Actions → Build and verify notes → a successful run → Artifacts** and
download `notes-windows-amd64`, `notes-linux-amd64`, or `notes-linux-arm64`.
Extract the archive first. Each build includes the executable and SHA-256
checksums. CI artifacts require a signed-in GitHub account and expire according
to the repository's retention policy; they are not permanent release downloads.

Running the executable requires neither Go nor Python. LaTeX is still required
to compile your notes through VS Code. On Linux, `uname -m` reports `x86_64`
for amd64 or `aarch64` for arm64. On Windows, use the amd64 build on an x64 PC.

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

Run `notes --help` to verify installation. To build from source, install [Go 1.25 or newer](https://go.dev/dl/) and run
one of these commands from the repository root:

```sh
# Linux
mkdir -p bin
go build -trimpath -o bin/notes ./cmd/notes
```

```powershell
# Windows PowerShell
New-Item -ItemType Directory -Force bin | Out-Null
go build -trimpath -o bin/notes.exe ./cmd/notes
```

Install that binary using the platform instructions above. The first source
build downloads the pinned Go terminal-handling dependency; ordinary CLI use
is entirely offline. `go test ./...` and `go vet ./...` run the code checks.

## CLI workflow

From the directory where you keep your course notes:

```text
notes new ECE342 2
```

This creates `ECE342/lecture-02/` containing `lecture.tex`,
`lecturenotes.sty`, and a `.notes.json` ownership marker. It refuses to
overwrite an existing lecture folder. Course codes accept letters, digits,
hyphens, and underscores, subject to Windows filename restrictions. The lecture
number is a positive integer.

Open that folder in VS Code, edit `lecture.tex`, and build with LaTeX Workshop.
The command itself does not compile the document.

After reviewing the final PDF:

```text
notes clean ECE342/lecture-02
```

**This is finalization, not ordinary LaTeX auxiliary-file cleanup.** After
confirmation, it permanently deletes everything in that generated lecture
folder except `lecture.pdf`, including source, the `.sty` package, figures,
subdirectories, and its marker. Copy the folder elsewhere first if you want to
retain editable material. Other course/lecture folders are not removed.

```text
notes clean ECE342/lecture-02 --yes
```

`--yes` skips the confirmation. Cleanup requires the tool's marker and
a nonempty PDF with a PDF signature. It cannot determine whether your PDF is
up to date or visually correct. It refuses an unmarked folder, including
this hand-authored example. After successful cleanup, the marker is gone
and a repeated cleanup refuses the already-finalized folder.

## Project design

See the [design specification](docs/superpowers/specs/2026-09-21-lecture-notes-design.md).
The repository origin is `https://github.com/Shaxzodm2611/Typesetting-Scripts`.
Configuring an origin does not publish commits or workflow artifacts.

The canonical formatting package is `internal/lecture/assets/lecturenotes.sty`.
The copy beside the sample is kept byte-identical by a regression test. The
executable embeds the package, so moving a generated lecture between computers
does not require copying any globally installed package.

### Cleanup details and troubleshooting

- The PDF and marker must be named exactly `lecture.pdf` and `.notes.json`,
  including letter case, for consistent behavior on both platforms.
- Course codes retain their case. If a course already exists, reuse its exact
  spelling; case-only course or lecture-folder collisions are refused even on Linux.
- Cleanup refuses symbolic links and Windows reparse points/junctions anywhere
  in the lecture path or contents. Remove the link entry before finalizing;
  its external target is not removed by the tool. Path components are checked
  before resolving `..`, so links cannot be hidden by parent traversal. On
  Windows, use an ordinary relative path or a fully qualified drive/UNC path,
  rather than a drive-relative path such as `C:notes` or `\notes`.
- If the PDF or ownership marker changes while confirmation is pending, cleanup
  refuses. Review the latest PDF and run the command again.
- If an ordinary deletion fails (for example, a Windows file is locked), the
  PDF and ownership marker remain so you can fix the problem and retry. Some
  other files may already have been removed.
- A PDF signature check establishes only that the file starts like a PDF; it
  does not establish that the document is complete, valid, or up to date.
- Noninteractive cleanup requires `--yes`; piping `yes` into the command does
  not bypass that policy. An explicit negative answer cancels with exit code 0;
  failures return 1, and invalid command syntax returns 2.

### Verification

GitHub Actions is configured to test natively on Windows and Linux with the Go
1.25 floor and current stable Go, compile the blank and sample LaTeX documents,
and build all three executable targets. A configured workflow is not evidence
that a remote run has completed; inspect the Actions run when published.

See the [verification record](docs/verification.md) for the checks actually run,
review corrections, and platform limitations.

## Updating the formatting

Replace the executable to use the updated formatting in new lectures. For an
existing editable lecture, copy `examples/rc-filter/lecturenotes.sty` into its
folder and rebuild in VS Code. Version 0.1.2 uses NewTX text and maths and adds
required titles to circuit and signal-plot environments; ensure the `newtx` TeX
package is installed. Existing derivations should use the left-aligned column
syntax shown above.
