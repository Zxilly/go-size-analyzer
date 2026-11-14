# Inline source attribution and generic emitted bodies

Measured 2026-11-13 using Go 1.27.0. This prototype is an independent DWARF reader;
it does not modify the production analyzer or add bytes to its physical ledger.

## Reproduce

From the repository root:

```powershell
$env:GOMAXPROCS = '4'
go test -mod=readonly -p=2 ./experiments/size-attribution/inline -v
python experiments/size-attribution/inline/run.py
```

The runner builds this experiment and three optimized fixture binaries with
`-mod=readonly -p=2`, then reads existing example/real binaries and the shared
`temp/size-experiments/gsa.exe` baseline. All generated binaries and reports are
under `temp/size-experiments/inline`. `summary.json` records each input's SHA256,
format, Go version, report path, limitations, and assertions. Each per-input
JSON contains all individual inline occurrences and generic emitted bodies.

## Results

| Input | DWARF | Inline occurrences with ranges | Unique inline file bytes | Generic declaration groups / emitted bodies | Bracketed names not grouped |
|---|---|---:|---:|---:|---:|
| Fixture Linux amd64 | yes | 6,236 | 106,809 | 5 / 10 | 2 |
| Fixture Windows amd64 | yes | 6,579 | 118,160 | 16 / 18 | 4 |
| Fixture Darwin amd64 | yes | 6,398 | 116,612 | 2 / 4 | 2 |
| Fixture Linux, `-w` with symbols retained | no | 0 | 0 | 5 / 10 | 2 |
| Existing Linux Go 1.27 amd64 | yes | 0 | 0 | 10 / 21 | 2 |
| Existing Windows Go 1.23 amd64 | yes | 0 | 0 | 9 / 25 | 6 |
| Existing Darwin Go 1.23 amd64 | yes | 0 | 0 | 9 / 27 | 7 |
| Existing Darwin Go 1.27 amd64 PIE CGO | yes | 15,531 | 627,754 | 92 / 162 | 4 |
| Existing Windows Go 1.23 amd64 stripped CGO | no | 0 | 0 | 0 / 0 | 0 |
| Docker Compose Linux amd64 | no | 0 | 0 | 298 / 657 | 97 |
| analysis-server Linux | yes | 10,421 | 445,429 | 162 / 723 | 92 |
| Current GSA Windows amd64 build | yes | 55,780 | 2,545,974 | 268 / 983 | 56 |

No unresolved inline names, unmapped function/inline range bytes, or equal-depth
source ambiguity occurred in these samples. This is an observation about these
inputs, not a guarantee for arbitrary DWARF. Every report verifies
`sum(deepest_source_bytes) + deepest_ambiguous_bytes == inline_union_bytes`.

The existing simple example binaries provide DWARF but no nonempty inline ranges.
Their recorded build settings explicitly use `-gcflags="all=-N -l"`, disabling
optimization and inlining. They therefore cannot by themselves demonstrate inline recovery. The optimized
fixture, CGO example, analysis-server, and current GSA build do demonstrate it.
The missing DWARF in Compose and stripped PE is reported rather than fabricated.
Compose still has sized ELF function symbols: a narrow native-symbol fallback
recovers 298 generic declaration groups and 657 emitted bodies, with evidence
explicitly marked `native_symbol`. The Linux `-w` fixture exercises this route
and still finds two shared generic bodies without inventing inline information.

The current GSA build exposes 136,523 inline bytes attributed to `errors.New`
under the deepest-source projection. Its largest generic declaration groups
include `slices.pdqsortCmpFunc` (52,478 bytes, 25 emitted bodies) and
`slices.partitionCmpFunc` (49,155 bytes, 25 emitted bodies). These are diagnostic
groupings of existing code; they are not predicted deletion savings.

## Controlled checks

The fixture has an explicitly non-inlined `main.host` containing two calls to
an inlineable helper. DWARF recovers two `main.addOne` occurrences inside this
host, including one nested through `main.doubled`. Physical ranges remain
attached to the host; the additional source view identifies the helper.

The fixture invokes one generic declaration with `int`, `uint8`, and a distinct
named integer type `main.Count`. On all three platforms it produces **three
dictionary symbols but only two shape bodies**, whose union is **50 bytes**.
This directly tests that dictionary count and source instantiation count must
not be treated as independent machine-code copies.

Unit tests independently cover nested interval attribution, ambiguous same-depth
overlap, generic bracket parsing, exclusion of type equality helpers and
dictionaries, physical bounds, and real ELF/PE compilation. The runner additionally
asserts the same fixture properties for Mach-O.

## Accounting and limitations

- Inline records follow `abstract_origin` / `specification`, retain their concrete
  containing function, and map validated address ranges into file ranges. Within
  each origin, size is a range union. Nested origins can overlap; their inclusive
  totals must not be summed. The separate deepest-source projection counts each
  unambiguous file byte once.
- A Go abstract DIE can provide the callee name without a declaration file/line.
  In that case the origin key is its fully qualified name; call-site locations
  remain separate. C DWARF in these samples often includes full declaration paths.
- Generic grouping strips balanced type-argument brackets from DWARF function
  names. Full names and ranges are retained. `go.shape` bodies are explicitly
  labeled as shared; other bodies are labeled symbol-specific without guessing
  whether they are wrappers. `type:` / `type.` helpers and dictionaries are not
  treated as generic declarations. The last table column reports excluded or
  unrecognized bracketed function names, not proven unrecognized generics.
- When DWARF is unavailable, ELF `STT_FUNC` definitions with explicit nonzero
  sizes can supply generic bodies. The complete range must map into executable
  file-backed sections. Existing DWARF and native-symbol views are never added
  together. PE/Mach-O fallback is not attempted because their symbol records in
  this prototype do not supply independently verified sizes.
- Thin ELF/PE/Mach-O and standard-library-supported DWARF are supported. This
  experiment does not read pclntab inline trees, split debug companions, Mach-O
  universal binaries, or WASM. Missing/unsupported DWARF has explicit notes.
- No stripping savings, retained size, callgraph causality, or end-user semantic
  equivalence is inferred. No elapsed-time claims are made from concurrent runs.

## Decision

The experiment supports adding an optional source overlay and generic declaration
grouping above the existing physical ledger. It makes existing bytes easier to
explain without changing package ownership. Its usefulness is gated by the
available metadata; pure-binary stripped inputs need a separate recovery route.
