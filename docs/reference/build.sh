#!/bin/sh
# Run from any directory. Build the reference using the canonical package.
set -eu
repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
build_dir="$repo_root/tmp/pdfs/block-reference"
cd "$repo_root"
mkdir -p "$build_dir" "$repo_root/output/pdf"
export TEXINPUTS="$repo_root/internal/lecture/assets:$repo_root/docs/reference:${TEXINPUTS:-}:"
pdflatex -no-shell-escape -interaction=nonstopmode -halt-on-error \
  -output-directory="$build_dir" docs/reference/figures/rc-response.tex \
  > "$build_dir/figure-build.txt" 2>&1 || {
    cat "$build_dir/figure-build.txt"
    exit 1
  }
cp "$build_dir/rc-response.pdf" docs/reference/figures/rc-response.pdf
for pass in 1 2; do
  pdflatex -no-shell-escape -interaction=nonstopmode -halt-on-error \
    -output-directory="$build_dir" docs/reference/block-reference.tex \
    > "$build_dir/build-$pass.txt" 2>&1 || {
      cat "$build_dir/build-$pass.txt"
      exit 1
    }
done
cp "$build_dir/block-reference.pdf" output/pdf/typesetting-block-reference.pdf
cp "$build_dir/block-reference.pdf" docs/block-reference.pdf
printf 'Built docs/block-reference.pdf\n'
