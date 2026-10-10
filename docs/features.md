# Typesetting features, v0.2.0

All commands below are implemented in the embedded `lecturenotes.sty`. New
lectures receive it automatically. Update existing lecture styles before using
the new commands. The original calls remain supported.

## Panels

```latex
\begin{notepanels}[columns=2,widths={2,1},gap=6mm]
  \begin{notepanel}{Circuit}
    \begin{notecircuit}[american,scale=0.55,transform shape]{RC filter}
      \draw (0,0) to[R=$R$] (2,0) to[C=$C$] (2,-1.5);
    \end{notecircuit}
  \end{notepanel}
  \begin{notepanel}{Interpretation}
    The resistor and capacitor form a low-pass filter.
  \end{notepanel}
\end{notepanels}
```

The output has aligned titled columns. `columns=2` or `3` gives equal widths by
default. `widths={2,1}` allocates twice as much usable width to the first panel;
the gap is excluded before distributing widths. For three columns, give three
positive weights. `align=top|center|bottom` controls vertical alignment.
`keep-together=true` keeps the block on one page; `false` permits breaks between
complete rows. Additional panels form additional rows. Use inline figures inside
panels, since normal LaTeX floats cannot appear inside a minipage.

## Standard images

```latex
\notefigure[width=0.85\linewidth,maxheight=65mm,
  placement=here,label=fig:sampling]
  {figures/sampling.png}{A sampled signal and its spectrum.}

See Figure~\ref{fig:sampling}.
```

The output preserves the image's aspect ratio, fits it within both dimensions,
and adds a numbered caption and reference target. Defaults are the limits shown
above and inline placement. `placement=here` keeps image and caption together;
`placement=float` uses `htbp`, or use placement letters directly, such as `t` or
`htbp!`. The label is optional. Graphics paths and extension lookup use graphicx.

## Wrapping derivations

```latex
\begin{notederivation}{Voltage gain}
  \derivestep{i_d=g_m v_{gs}}
    {Use the small-signal relation for drain current.}
  \derivestep{A_v=-g_mR_D}
    {The drain resistor converts current to voltage. This explanation wraps as
     ordinary prose instead of becoming a long, unbreakable math annotation.}
\end{notederivation}
```

Each step keeps its equation and explanation together; long derivations can
break between steps. Explanations appear below by default. For a wrapping text
column beside each equation (`explanations` is also accepted as an alias):

```latex
\begin{notederivation}[explanation=beside,equation-width=0.52\linewidth,gap=6mm]
  {Voltage gain}
  \derivestep{A_v=-g_mR_D}{Explain the approximation here.}
\end{notederivation}
```

An existing equation-only body still uses the original `alignedat` layout.
Step bodies are detected automatically. Set `mode=steps` when the body is
loaded through `\input`, or to make editor context explicit; `mode=legacy`
selects the original layout explicitly. These modes can coexist in one lecture.

## Circuit presets and annotations

```latex
\notesetup{circuit-preset=compact}
\begin{notecircuit}{Diode-connected NMOS}
  \node[nmos,arrowmos,xscale=-1] (M) at (0,0) {};
  \coordinate (bias) at (0,1.2);
  \draw (M.D)--(bias) node[circ]{}
    --(M.G |- bias)--(M.G) node[circ]{};
  \draw (M.S)--++(0,-0.2) node[ground]{};
  \draw[note-highlight] (bias)--(M.G |- bias)--(M.G);
  \draw[note-current] (-0.8,1.1)--(-0.8,0.5);
\end{notecircuit}
```

`compact` supplies American symbols, scale 0.55, transformed labels, and compact
CircuitikZ current arrows; `standard` uses scale 0.8. `none`, the default, leaves
the drawing's existing options in control. Explicit `notecircuit` options are
applied after the preset, so a diagram can override its scale. `note-highlight`
uses a translucent yellow stroke; `note-current` uses a thin red shaft and small
arrowhead. TikZ styles can be overridden with `\tikzset`.

The diode connection uses the gate anchor's horizontal position and the bias
node's vertical position. Its wire bends remain orthogonal when the MOS symbol
is mirrored or resized; no guessed gate coordinate is needed.

## Check lectures

```text
notes check ELE727/lecture-02
notes check ELE727 --compile
notes check . --strict
```

The checker visits editable lectures and their literal `\input`/`\include`
files. It reports duplicate and placeholder labels, undefined references,
missing images/inputs, TODO/FIXME/TBD notes, and unfinished "Add ... here" text.
Labels are scoped to each lecture. Comments and verbatim/code blocks do not
create false labels or references; TODOs in comments remain visible. It also
recognizes `\notefigure` labels, graphics search paths, and the School-Notes
`\sourcefigure` wrapper. Arbitrary TeX expansion and external package-generated
labels are beyond the static scan; use the real build to assess such documents.

Diagnostics include filename, line, severity, and kind. Errors return exit 1;
warnings return 0 unless `--strict` is given. Invalid syntax returns 2.
`--compile` runs pdfLaTeX twice without shell escape, using a temporary output
directory and a separate job name. It reports compiler errors and layout
warnings without changing existing PDFs, logs, or aux files. Each pass has a
45-second timeout. Build processes that require EPS conversion or other external
tools should use their normal build recipe.

## Update lecture styles

```text
notes update-style ELE727 --dry-run
notes update-style ELE727
```

Either command accepts a lecture, course, or notes root; omission means the
current directory. The preview reports versions, retained custom edits, and
conflicts. Applying merges compatible custom edits against bundled historical
styles, writes each updated style atomically, and creates an exact backup as
`lecturenotes.sty.bak`, then `.bak.1`, `.bak.2`, and so on. Existing backups are
never overwritten. Local colors, added commands such as `\notesummary`, and
legacy bookmark settings are preserved when compatible. Source and figures are
unchanged. An up-to-date lecture gets no new backup.

All lectures are checked for conflicts before any style is replaced. If a
customization overlaps an updated definition, inspect the reported base lines.
Choose `--resolve=keep` to retain conflicting local edits or
`--resolve=replace` to use the updated definition; compatible additions remain
with either choice. Complex or structurally invalid merges still require a
manual merge. Keeping an old definition can keep its old behavior. Unknown
versions require a manual merge as well. Rebuild and review affected PDFs.

## Heading navigation

```latex
\notesetup{toc-depth=2,bookmark-depth=5}
\tableofcontents
\clearpage
\noteheading{1}{Current mirrors}
\noteheading{2}{Cascode mirrors}
\noteheading[bookmark={Output resistance ro}]{3}{Output resistance $r_o$}
\noteheading{4}{Dominant term}
\noteheading{5}{Approximation condition}
```

Contents is the list printed in the PDF. Bookmarks are the tree in the PDF
viewer's sidebar. Clicking an entry jumps to its heading. Depth 2 lists only
levels 1 and 2; depth 5 includes all five. All headings still appear in the
lecture body. Both depths default to 5; 0 hides all heading entries in the
corresponding list. The optional `bookmark` title avoids putting TeX mathematics
into the sidebar title while retaining it on the printed page. Compile twice
after changing headings to refresh the contents and bookmarks.

## School-Notes editor shortcuts

HyperSnips shortcuts are available at the start of a line:

| Shortcut | Result |
| --- | --- |
| `;pan2`, `;pan3` | Two or three titled panels |
| `;panel` | Additional panel inside a row |
| `;fig` | Standard inline figure with dimensions, caption, and label |
| `;drv` | Derivation using wrapping steps |
| `;step` | Additional equation/explanation step |
| `;cset` | Select a circuit preset |
| `;cir` | Circuit block |
| `;cur`, `;mark` | Current arrow and connection highlight |
| `;nav` | Independent contents and bookmark depths |
| `;hbm` | Heading with a plain bookmark title |
| `;h1` through `;h5` | Existing numbered heading shortcuts |

`;der` retains the original equation-column
derivation. Run **HyperSnips: Reload Snippets** after updating the workspace.
