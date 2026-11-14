# Build-assisted retention and external-symbol experiments

Executed 2026-11-13 on Windows/amd64 using Go 1.27.0, GOMAXPROCS=4,
`go build -mod=readonly -p=2`, and the shared current-GSA baseline.
This directory contains isolated prototypes; production code and go.mod were not changed.
The experiments ran concurrently with other work. Their elapsed times are not performance comparisons.

## Reproduce

From the repository root, after building `temp/size-experiments/gsa.exe`:

```powershell
python experiments/size-attribution/buildassist/run.py
python -m unittest discover -s experiments/size-attribution/buildassist -p test_experiments.py -v
```

Tools required: Go, Python standard library, llvm-objcopy, GNU objcopy.
All binary copies, complete GSA reports, command logs, and machine-readable
`results.json`, `summary.json`, `witness.json` are written under
`temp/size-experiments/buildassist/`. Original integration binaries are read only.

## 1. Linker retention witnesses are useful, but incomplete

The fixture has two non-inlined functions, `main.callerA` and `main.callerB`, both
calling `main.shared`. `go tool objdump` proves two actual `CALL main.shared(SB)`
instructions. `-ldflags=-dumpdep` emits only `main.callerA -> main.shared`.

The installed Go 1.27 source explains the result:
`cmd/link/internal/ld/deadcode.go:359-378` prints a dependency only inside
`symIdx != 0 && !AttrReachable(symIdx)`. It records the first reachability witness,
not every incoming edge. A dominator or "delete callerA and save shared" result
derived from that output would therefore be wrong.

A Linux/amd64 build of the actual GSA project produced **123,741 witness edges**
and a manifest containing the binary hash, actual `go version -m` output, exact
build command, and build environment. One observed path reaches
`github.com/go-delve/delve/pkg/dwarf/op.ExecuteStackProgram` through:

`main.main -> main.entry -> main.run -> internal.Analyze -> internal.analyzeNative
-> KnownInfo.TryLoadDwarf -> KnownInfo.GetDwarfCompileUnitFeeder -> feeder.func1
-> KnownInfo.AddDwarfVariable -> knowninfo.staticVariableAddress -> ExecuteStackProgram`.

This is enough for a useful "one reason this was retained" feature. It cannot
promise an exhaustive reference graph, exclusive retained size, or deletion
savings. Some linker canonical symbol names belong to merged/aliased constants;
their spelling alone is not a source-package dependency proof. The experiment
does not patch the installed linker or attempt to claim complete graphs.

## 2. External symbol matching and measured recovery

Each donor is an existing unstripped ELF, and the target is a new copy stripped
with objcopy. A sidecar is accepted only when machine/type/entry, allocated
section names/types/flags/virtual addresses/sizes/content hashes, load layout
and permissions match. Every allocated file-backed section must map to its
declared virtual address. BSS layout is checked but never counted as file bytes.

Nonallocated symbol/debug sections may differ. PT_LOAD file size can differ in
unallocated tail padding; allocated bytes and their mapping must still match.
The matcher does not claim that ELF headers or padding bytes are identical.
After matching, donor attributed executable-byte ranges are mapped by virtual
address into the target and unioned with target coverage. The reported gain is
the difference of unique interval unions, not a sum of symbols or packages.

| Sample | Target attributed executable bytes | With matched sidecar | Extra attributable code |
|---|---:|---:|---:|
| example Go 1.23, pure Go | 644,069 | 644,069 | **0** |
| example Go 1.23, CGO | 1,421,605 | 2,355,135 | **933,530** |
| example Go 1.27, CGO | 1,583,840 | 2,517,412 | **933,572** |
| Docker Compose real binary | 26,452,280 | 26,452,280 | **0** |
| analysis-server real binary | 21,995,160 | 22,930,070 | **934,910** |
| GSA built from current source | 10,389,984 | 10,389,986 | **2** |

These measurements support external symbols for native/CGO attribution. They
do not support an assumption of large extra code coverage for pure Go, where
pclntab already identifies most executable bytes. Additional names/debug
metadata may still help explanations even when the byte-coverage gain is zero.

The repository has no independently supplied historical production debug
sidecar for Compose or analysis-server. Their rows prove the controlled
same-artifact stripping scenario, not recovery of a missing historical build.
The prototype uses full original ELF files as sidecars. Detached debug ELF,
GNU build-id/debuglink lookup, PDB/dSYM, ELF32, and other endianness are outside
this implementation.

## 3. Negative and tool-compatibility results

Four existing independently built unstripped/stripped pairs were rejected:
Go 1.19, 1.21, 1.23 pure Go and Go 1.23 CGO. All had differing `.text` contents
or layout as well as other allocated differences. "Same example source and Go
version" is not sufficient to transfer addresses. A Go 1.27 donor for the Go
1.23 target was also rejected.

LLVM objcopy 19.1.7 produced a malformed section-name string table for the
Compose copy; llvm-readelf independently warned that the SHT_STRTAB was not
NUL-terminated. The matcher rejected it. GNU objcopy 2.44 produced matching
allocated sections and valid names, but removed 16 bytes of trailing load
padding; the explicit mapping-based match accepted that copy. The rejected
LLVM attempt remains recorded in `results.json` and its logs.

Ten acceptance tests cover real two-parent retention, real-project witnesses,
all six matched copy scenarios, separately built mismatches, observed CGO
coverage gains, wrong version, and refusal after mutations of code, read-only
data, target machine or load permissions. Wrongly matched data is never passed
to the transfer calculation.

## Implementation priority suggested by the evidence

1. Add build-assisted witness paths with an explicit "one observed reason"
   meaning; do not derive removal savings from `-dumpdep` alone.
2. Save a matching debug/symbol artifact at build time, with exact content and
   layout identity. This has measured value for CGO code attribution.
3. Treat complete retention graphs and measured feature-removal builds as a
   separate project. The existing linker witness flag is insufficient evidence
   for complete reachability or exclusive retained-size calculations.
