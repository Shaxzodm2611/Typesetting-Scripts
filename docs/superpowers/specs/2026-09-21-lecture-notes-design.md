# Lecture notes framework design

## Purpose and agreed requirements

Create short lecture summaries after class with minimal time spent on formatting. This is a lecture-summary workflow; problem sets will never be typeset. The user studies analog and digital integrated circuits, ARM-focused embedded systems, and digital communications. All courses use the same blank template and formatting package.

The implementation language is Go. Prebuilt Windows and Linux executables require neither Go nor Python on the user's machine. The `notes` command works from any directory after installation on PATH. The user edits LaTeX in VS Code; the IDE owns compilation and PDF preview. This tool does not run a TeX compiler.

Each lecture produces a separate PDF. Its initial document is one otherwise blank page with a compact header containing only the course code and lecture number. There is no title page, date, prefilled section, or example content. A local LaTeX package supplies reusable blocks with consistent, subtle color accents. Once the user accepts the PDF, a cleanup command deletes everything else in that lecture directory, including editable source, figures, and the local package. Cleanup prompts by default and supports `--yes`.

## Commands and files

### Create

```text
notes new ECE342 2
```

Creates the following relative to the current working directory:

```text
ECE342/lecture-02/
  lecture.tex
  lecturenotes.sty
  .notes.json
```

The lecture number is a positive integer. Numbers are padded to at least two digits in folder names. Course codes permit ASCII letters, digits, hyphens, and underscores and must begin with a letter or digit. Preserve the supplied case, reject Windows reserved device names, and escape underscores when inserting the code into LaTeX. No initial course registry is required.

Creation refuses to overwrite an existing lecture directory. Embedded template assets are shipped inside the executable. The local package copy makes the lecture portable before cleanup. The JSON marker records a format version, course code, lecture number, and fixed final-PDF filename. No global configuration is needed.

`lecture.tex` uses the local package and sets course and lecture metadata. An invisible body element ensures the initial document actually emits a page when compiled. Building with the normal job name yields `lecture.pdf` beside `lecture.tex`.

### Clean

```text
notes clean ECE342/lecture-02
notes clean ECE342/lecture-02 --yes
```

The command operates only on a directory with a valid tool marker. It requires `lecture.pdf` at the directory root to be a nonempty regular file with a PDF signature. This is a basic file check, not proof that the PDF is visually correct or up to date: the user's review remains the acceptance step. The README explains that VS Code must output the final PDF beside the source using the default job name.

Before deletion, show the resolved lecture directory, PDF to preserve, and that all remaining contents, including sources and figures, will be permanently deleted. Require an explicit affirmative answer unless `--yes` is provided. Noninteractive use without `--yes` fails without deleting anything.

Preserve exactly the root `lecture.pdf`; remove every other entry in that lecture folder, including nested build folders, local images, the `.sty`, `.tex`, and marker. Remove the marker last so partial failures remain diagnosable and retryable. Do not traverse symbolic links or Windows reparse points into other locations; reject ambiguous linked lecture paths and linked final PDFs. Never use marker contents as arbitrary deletion or preservation paths. Invalid metadata, missing PDF, unknown marker versions, or unreadable directories fail before deletion starts. If deletion later fails, report the remaining failure, return nonzero, and preserve the PDF.

A successful cleanup leaves only `lecture.pdf`. A later cleanup of the same folder refuses because the marker is gone; it does not guess ownership from the folder name.

### Help and errors

Provide `notes --help`, command-specific help, and `notes --version`. Explain invalid arguments with an actionable message and nonzero exit status. Creation and cleanup success messages name their output paths. No network access is required during normal use.

## LaTeX package

`lecturenotes.sty` owns the shared page layout, header, palette, spacing, and reusable commands. Proposed defaults are A4 paper, 22 mm margins, 11 pt body text, and a muted teal accent with a very pale matching background. Keep text dark and avoid heavy frames. Paper size can be changed in the document class without changing the tool.

Use established LaTeX packages for mathematics, tables, boxes, headers, CircuitikZ, and PGFPlots. Aim for pdfLaTeX compatibility, with no shell escape, external plotting tools, or custom fonts required. Document the required TeX packages and a working VS Code/LaTeX Workshop setup for both operating systems.

The public authoring interface is:

```latex
\usepackage{lecturenotes}
\lectureheader{ECE342}{2}

\notesection{Frequency response}

\keyequation{Ohm's law}{V = IR}

\assumption{The op-amp is ideal.}

\begin{notetable}{lll}{Quantity & Symbol & Unit}
Voltage & $V$ & volts \\
Current & $I$ & amperes \\
\end{notetable}

\begin{notecircuit}[scale=1]
  % CircuitikZ drawing commands
\end{notecircuit}

\begin{notesignal}[xlabel={$t$}, ylabel={$v(t)$}]
  % PGFPlots commands, such as \addplot ...;
\end{notesignal}
```

Optional topic dividers use an unnumbered accent heading and a thin matching rule. The source example separates topics with comment dividers as well. Generated lectures remain blank until the author inserts these headings. Equation titles and assumptions use the same accent treatment. Tables receive consistent column padding and header styling without requiring the author to draw rules manually. Circuit wrappers create a centered CircuitikZ environment. Signal wrappers create a centered TikZ picture and PGFPlots axis with shared defaults; user options override those defaults. For ChatGPT-generated diagrams, document exactly which inner commands belong in a wrapper and show how to use a complete standalone TikZ picture directly without nesting incompatible environments.

Circuit and signal content remains user supplied. The tool does not call ChatGPT, interpret images, or generate diagrams. Include a short copyable prompt in the documentation for requesting compatible drawing code from an image.

The package also supplies `\notedefinition{Term}{Explanation}`, a `notation`
environment with fixed Symbol/Meaning/Unit columns, a `notederivation{Title}`
environment containing unnumbered aligned equations and optional annotations,
and a verbatim `notecode` environment. Code defaults to C and accepts options
for C++, ARM instruction highlighting, plain text, and optional line numbers.
Use `listings`, so code needs no Python or shell escape. Derivations represent
lecture reasoning, and code blocks capture lecture examples; no problem-set
or worked-solution workflow is introduced.

## Repository and distribution

Use a small Go module with focused CLI, filesystem lifecycle, and embedded-asset code. Prefer the standard library unless a concrete portability requirement calls for a dependency. Ship a README, `.gitignore`, examples, and automated checks alongside the source. Examples live separately; generated lecture documents remain blank.

The README covers installation on Windows and Linux, adding the binary directory to PATH, creating a lecture, editing and compiling in VS Code, the package interface, pasting diagram code, and destructive cleanup semantics. Ignore local executable outputs, build directories, and LaTeX intermediates. Do not ignore `.sty` source assets or documentation examples.

Configure GitHub Actions to test on Windows and Linux and build executable artifacts for Windows amd64 and Linux amd64/arm64. Building from source requires Go; running downloaded binaries does not. Linux binaries should not require cgo. The existing Git origin is `https://github.com/Shaxzodm2611/Typesetting-Scripts`. Inspect remote history before integrating or publishing; do not force-push or replace existing remote work.

## Verification and acceptance

- Test CLI argument handling, positive lecture-number validation, safe course codes, generated names, and refusal to overwrite existing work.
- Test template substitution and generation from a working directory containing spaces.
- Compile the blank template and verify exactly one page, header contents, and absence of body sections or examples.
- Compile a separate example exercising equations, assumptions, tables, topic dividers, definitions, notation, derivations, C and ARM code, a circuit, and a signal plot; visually inspect its rendered pages.
- Test cleanup confirmation, cancellation, `--yes`, noninteractive refusal, invalid ownership metadata, missing or invalid PDFs, nested contents, link handling, and preservation of the final PDF's exact bytes.
- Test that cleanup cannot affect files outside the lecture directory and that successful cleanup leaves exactly one file.
- Run Go tests and platform builds locally where supported. Use Windows and Linux CI for native platform checks; distinguish locally verified behavior from CI checks that have only been configured.
- Confirm repository documentation, `.gitignore`, and origin match the deliverable.

## Scope boundaries

No editor extension, GUI, compilation command, combined course PDF, cloud service, Python runtime, or automatic image-to-TikZ conversion is needed. Snippets are unnecessary for the initial version because the shared package provides the requested reusable authoring interface.
