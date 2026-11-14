# Binary-size attribution experiments

Four independent prototypes test six directions: object evidence, function-level
diffs, linker retention witnesses, matching external symbols, inline/generic
source views, and bounded static-object/CFG discovery. The first experiment round
left production code unchanged; selected findings are now integrated as described
in [the production iteration](../../docs/size-attribution-iteration.md). These
prototype commands remain research tools, not a supported CLI API.

## Run

Use the repository's Go version (currently Go 1.27), Python 3, and the existing
`scripts/bins` integration samples. The external-symbol experiment additionally
uses `llvm-objcopy` and GNU `objcopy` on PATH. It records rejected stripping
attempts and validates copies before transferring any symbol information.

```powershell
python experiments/size-attribution/run.py --workers 4
```

The runner first builds current GSA into `temp/size-experiments/gsa.exe`, then
launches the four experiment runners concurrently, with at most four Go threads
and two parallel build packages per runner. It saves exit status and logs, and
combines per-group JSON into `temp/size-experiments/summary.json`. Use
`--workers 1` on a memory-constrained host. Parallel elapsed times are not a
performance benchmark.

To compare the integrated production analyzer with a preserved pre-change GSA:

```powershell
go build -mod=readonly -p=2 -o temp/size-iteration/gsa.exe ./cmd/gsa
python experiments/size-attribution/iteration.py --baseline path/to/baseline-gsa.exe
```

The default baseline is `temp/size-experiments/gsa.exe`. Preserve that executable
before rerunning the original experiment runner, which builds the current source
and would otherwise replace it. `iteration.py` compares eleven existing binaries,
checks input hashes, physical totals, function totals, unknown metrics, and bounded
evidence samples, and writes `temp/size-iteration/matrix/summary.json`.

The second production iteration adds a real-binary symbol diff runner and a
controlled Web payload comparison:

```powershell
go build -mod=readonly -p=2 -o temp/size-iteration2/gsa.exe ./cmd/gsa
python experiments/size-attribution/symbols_run.py
python experiments/size-attribution/webpayload/run.py --baseline-cli temp/size-iteration/gsa-embedded.exe
```

Preserve the first iteration's embedded CLI before running the Web comparison.
See [Web payload checks](webpayload/README.md) for custom inputs and compare-only
validation. `symbols_run.py` compares Compose with itself and, when present, the
saved default/embed GSA binaries from the first experiment. It never runs inputs.
The real compiler fixture in `internal/diff/symbols_compiled_test.go` separately
checks offsetting static string changes with identical package and file totals.
For isolated UI rendering measurements using identical full JSON payloads, see
`ui/scripts/render-benchmark.py`. Results and limits are documented in the
[second iteration report](../../docs/size-attribution-iteration2.md).

Input binaries are only read. They are never executed or overwritten. Controlled
Go/assembly fixtures and stripped copies are generated under `temp/` and are
also analyzed without execution. No dependencies, production code, original
binaries, or installed toolchains are rewritten.

To combine already generated summaries without rerunning the experiments:

```powershell
python experiments/size-attribution/run.py --collect
```

## Components

| Directory | Experiment | Findings |
|---|---|---|
| `report` | Preserve function/symbol evidence; manifest; object diff; standalone HTML | [findings](report/findings.md) |
| `buildassist` | `-dumpdep` witness paths; strict ELF symbol-sidecar matching | [findings](buildassist/findings.md) |
| `inline` | DWARF inline origins; deepest-source projection; generic shape groups | [findings](inline/findings.md) |
| `staticreach` | Typed static pointer/interface traversal; bounded amd64 CFG | [findings](staticreach/findings.md) |

Every directory has a standalone `run.py`. Output JSON includes limitations;
individual summaries retain input hashes. Open the HTML reports directly from
`temp/size-experiments/report/`; they have no external dependencies. The largest
JSON reports exceed 100 MB, so the HTML shows a bounded preview instead of loading
the full dataset.

## Checks

```powershell
go test -mod=readonly -p=2 ./experiments/size-attribution/report ./experiments/size-attribution/staticreach ./experiments/size-attribution/inline ./internal/entity ./internal/knowninfo ./internal/diff
python -m unittest discover -s experiments/size-attribution/buildassist -p test_experiments.py -v
```

The Python acceptance checks use the artifacts from the buildassist runner.
Go tests include real compiler outputs and independent interval/ambiguity cases.

## Interpretation

The production physical ledger is the reference for file totals. Object sizes,
inline source sizes, reference candidates and package totals may overlap. None
of them is an estimate of bytes saved by deleting a package. Typed static data,
runtime-type-backed interface candidates and heuristic CFG pairs are reported
separately. Sidecar gains refer to newly attributable code after controlled
stripping, not a smaller executable or recovery of an unknown historical build.

The symbol matcher currently handles little-endian ELF64 full original binaries;
the static/CFG prototype handles ELF64 amd64; the inline prototype handles thin
ELF/PE/Mach-O; the object report accepts all GSA formats but does not project WASM
object file ranges. Unsupported metadata is explicit in reports.
