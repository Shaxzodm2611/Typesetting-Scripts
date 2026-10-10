# Block reference

Open the [illustrated block reference](block-reference.pdf). It covers the full
`lecturenotes` v0.2.0 API with exact signatures, defaults, copyable calls, rendered
results, and School-Notes editor shortcuts. Use the PDF's first-page index or
bookmarks to jump directly to a block.

| Need | Reference page | Copyable example |
| --- | --- | --- |
| Topic headings | 2 | [Five levels](reference/examples/headings.tex) |
| Contents and PDF bookmarks | 3 | Settings and explanation in the reference |
| Notes, definitions, assumptions | 4 | [Note box](reference/examples/box.tex), [definition](reference/examples/definition.tex), [assumption](reference/examples/assumption.tex) |
| Key equations | 5 | [Equation with explanation](reference/examples/equation.tex), [multiple lines](reference/examples/equation-multiline.tex) |
| Wrapping derivations | 6 | [Below](reference/examples/derivation.tex), [beside](reference/examples/derivation-beside.tex) |
| Compact or multiline derivations | 7 | [Original layout](reference/examples/derivation-legacy.tex), [multiline step](reference/examples/derivation-multiline.tex) |
| Tables and symbol lists | 8 | [Table](reference/examples/table.tex), [notation](reference/examples/notation.tex) |
| Two or three panels | 9 | [Weighted widths](reference/examples/panels.tex), [three columns](reference/examples/panels-three.tex) |
| Images and captions | 10 | [Figure with label](reference/examples/figure.tex) |
| Circuits and annotations | 11 | [Orthogonal MOS connection](reference/examples/circuit.tex) |
| Signal plots | 12 | [Charging response](reference/examples/signal.tex) |
| Source code | 13 | [C](reference/examples/code-c.tex), [ARM](reference/examples/code-arm.tex) |
| School-Notes summary extension | 14 | [Summary](reference/examples/summary.tex) |
| Checking and updating styles | 15 | Commands and options in the reference |
| VS Code shortcuts and common mistakes | 16 | Lookup table in the reference |

The example files are fragments to paste inside a lecture's document body.
The figure example uses the included reference image path: in a lecture, copy
the image into `figures/` and change the path accordingly. `notesummary` is a
local School-Notes extension, not a command included by `notes new`.

To edit or rebuild the PDF, change [its LaTeX source](reference/block-reference.tex)
or the example files, then run from the repository root:

```sh
sh docs/reference/build.sh
```

The build needs the same TeX packages as a lecture, plus `standalone` for the
example image. It compiles the current canonical style and uses the same example
files for both the printed calls and rendered output. The separate
[School-Notes extension](reference/school-notes-extension.sty) is loaded only
for that command's reference page. Build files go under `tmp/pdfs/block-reference/`;
the finished reference is written to `docs/block-reference.pdf` and
`output/pdf/typesetting-block-reference.pdf`.
