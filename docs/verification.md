# Verification record

## Implemented and verified locally

- `notes new COURSE NUMBER` creates an exclusive, portable lecture directory
  with a blank source, the approved local package, and an ownership marker.
- The blank generated source compiles to one A4 page containing only course
  and lecture metadata. An underscore in a course code renders literally.
- The complete sample compiles to three pages without LaTeX warnings or overfull/
  underfull boxes. Every page was rendered and visually inspected.
- The v0.1.2 sample embeds NewTX text and mathematics. Derivation equations use
  a common left edge with a compact, separately aligned annotation column;
  circuits and signal plots carry concise figure titles.
- The CLI does not invoke a TeX compiler or create PDFs.
- Cleanup preserves only `lecture.pdf`; the original PDF SHA-256 remains
  unchanged in actual compiled-PDF checks. A real terminal cancellation keeps
  all files; a real terminal affirmative answer and noninteractive `--yes`
  both finalize correctly.
- 89 Linux test cases pass, including subtests, with no skips. The complete
  suite passes with Go 1.25.14 and Go 1.27.1. Race-detector tests and `go vet`
  pass on Linux.
- Windows test packages cross-compile. Executables build for Windows amd64,
  Linux amd64, and Linux arm64 with `CGO_ENABLED=0`; the Linux binaries are
  statically linked. The Linux amd64 binary was executed locally.
- GitHub Actions is configured for native Windows/Linux tests at the Go 1.25
  floor and current stable Go, LaTeX compilation, and downloadable artifacts.
  Remote CI and native Windows execution have not occurred in this environment.

## Review and corrections

An independent read-only code review covered creation, cleanup, the CLI,
platform differences, tests, CI, and documentation. Two findings were fixed:

1. Raw path components are now checked before lexical normalization. A path
   such as `alias/../ECE342/lecture-02` cannot hide a linked component and select
   a different lecture. Reproducing tests failed before the fix and passed
   after it; safe parent traversal through ordinary directories still works.
2. Lecture-directory names, as well as course-directory names, are checked for
   case-only conflicts. Linux can no longer create a second case-colliding
   lecture directory that would be ambiguous on Windows. The preservation test
   failed before the fix and passed afterward.

The whole suite, race detector, vet, minimum-Go test run, Windows test-package
compilation, and all three executable builds were repeated after these fixes.
There are no deferred review findings.

## Implementation decisions and limits

- Case-only course spellings are refused on Linux as well as Windows to keep
  folders portable. Reuse an existing course's exact spelling. Lecture-folder
  case conflicts are also refused because moving both to Windows would be
  ambiguous.
- Cleanup refuses links/reparse points anywhere in the lecture tree. Remove a
  link entry before finalizing; the target is not deleted. This conservative
  behavior avoids platform-specific traversal guesses.
- Cleanup confines deletion to an opened root and revalidates the PDF and
  marker; it is not a filesystem transaction against hostile concurrent
  writers. Stop automated build/watch processes first. A concurrent writer can
  recreate files or interrupt finalization.
- Full PDF validity and freshness remain the user's review responsibility.
  A signature check does not prevent a user from confirming a stale PDF.
- Hard-linked PDFs count as ordinary files. Cleanup does not change their
  bytes, but another owner of the same inode could later change those bytes.
- Valid marked lecture folders may be moved or renamed; metadata need not
  match the enclosing folder's current name. Copying the marker into an
  unrelated folder effectively asserts tool ownership of that folder.

## Reproduce

```sh
go test -count=1 ./...
go test -race ./...  # Requires a supported race-detector platform and C toolchain.
go vet ./...
go build -trimpath -o bin/notes ./cmd/notes
```

Use a disposable directory for cleanup checks. Native Windows verification is
provided by the checked-in CI job when that workflow is published and run.
