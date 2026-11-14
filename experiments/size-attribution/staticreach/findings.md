# Static reachability and bounded CFG experiment

Run date: 2026-11-13. Production files and dependency manifests were not changed. The input executables were only read, never executed. Concurrent experiments make this unsuitable for performance comparisons.

## Reproduce

From the repository root:

```powershell
python experiments/size-attribution/staticreach/run.py
```

The standard-library runner runs the targeted Go tests, builds the prototype and two Linux/amd64 fixtures, analyzes four existing samples, and writes `temp/size-experiments/staticreach/summary.json`, per-input JSON, and logs. `--out PATH` selects an independent output directory; `--skip-build` reuses binaries. Builds use `-mod=readonly -p=2`, `GOMAXPROCS=4`. Input SHA-256 hashes are in the summary. The baseline calls the same production `internal.Analyze` with `CoverageDetails: true`, rather than implementing an alternative baseline parser.

## What was actually attempted

1. **Typed static graph.** Start from DWARF globals with a static `DW_OP_addr`, follow pointers and typed aggregate fields, validate file-backed ELF ranges, and visit shared/cyclic objects once. Strings and slice backing arrays use declared layouts and sizes. This gives initialized file references, not exclusive ownership or linker retention. The representative root of a shared object is not its sole owner.
2. **Interfaces.** Resolve static interface values against the parsed Go runtime type set. Nonempty interfaces require an itab with an interface-type first word and concrete-type second word. Box sizes come from the concrete runtime descriptor. Direct interface values are followed only for concrete pointer kinds; direct one-element arrays and single-field structs are deliberately skipped. These results stay in a separate candidate family.
3. **Bounded CFG propagation.** Decode amd64 functions; propagate constant/address register facts across direct branches; retain only identical facts at merges; discard state on calls and unknown instruction effects. Partial register writes clear state; 32-bit LEA truncates/zero-extends. At calls/returns, plausible Go ABI address/length pairs must point to printable UTF-8 file bytes. Those are candidates, not proven string semantics or production attribution.

The CFG has a 128 KiB / 15,000-instruction / 60,000-worklist-step budget per function. The static graph has a 250,000-object budget, depth 24, maximum object size 16 MiB, and at most 4,096 aggregate elements. Pure scalar elements need not be traversed. The final analysis-server run did not hit the static-object budget; three functions exceeded a CFG budget.

## Results

All numbers are unique physical file bytes. Each result family takes its union and subtracts the production baseline's attributed union. The families overlap and **must not be added**. The CFG cross-block family is a subset of all CFG candidates. The prototype does not change baseline attribution.

| Input | Baseline attributed | New typed references | Of these: previously recognized | Of these: previously unclassified | New interface candidates | New CFG candidates (all cross-block here) |
|---|---:|---:|---:|---:|---:|---:|
| Compiled fixture | 1,307,041 | 1,868 | 0 | 1,868 | 496 | 0 |
| Same fixture, stripped | 1,136,946 | 0 | 0 | 0 | 0 | 16 |
| bin-linux-1.27-amd64 | 1,571,766 | 4,053 | 0 | 4,053 | 232 | 0 |
| bin-linux-1.23-amd64-strip | 1,231,695 | 0 | 0 | 0 | 0 | 0 |
| docker-compose-linux-x86_64 | 60,871,047 | 0 | 0 | 0 | 0 | 192 |
| analysis-server-linux | 52,303,047 | 98,222 | 62,864 | 35,358 | 0 | 0 |

The DWARF-free inputs correctly have zero typed roots; zero in those columns means unavailable, not that no static graph exists. Docker Compose has no usable DWARF in this sample. The stripped fixture proves that the CFG path can recover a known 16-byte assembly string split across a branch while the current baseline misses it. The unstripped fixture already attributes that same string through symbols.

Useful concrete discoveries:

- `analysis-server-linux`: `github.com/ZNotify/server/docs.SwaggerInfo*.SwaggerTemplate.bytes` identifies a **29,287-byte Swagger template** at physical offset **26,866,173**. This is an example of ordinary-pointer traversal producing actionable business-data attribution evidence.
- In that same file, `runtime.firstmoduledata.typelinks.backing` (44,800 bytes) and `.itablinks.backing` (18,064 bytes) account for all **62,864 previously recognized bytes** newly referenced by the graph. They improve structural explainability; they do not reduce the unclassified region. They must not be presented as 62,864 new unknown bytes explained, nor as runtime-exclusive deletion savings.
- Docker Compose's only new CFG candidates after conservative instruction handling are two **64-byte** `encoding/base64.init` strings and two **32-byte** `encoding/base32.init` strings. Analysis-server has seven cross-block candidates totaling 161 bytes, but every byte was already attributed. The current straight-line analyzer already covers almost all strings discovered by this prototype.
- The compiled interface fixture exercises an initialized `any` string box and a boxed `Payload` struct; the direct `*Node` path is also available. The extra 496 bytes include runtime boxes as well as fixture boxes. These are candidate references, not a claim that all 496 bytes belong solely to the fixture.

## Verification and limits

Targeted tests passed. They cover physical union/subtraction and recognized/unclassified separation; shared/cyclic typed pointers; unmapped-pointer rejection; a real compiled pointer/interface fixture; cross-block propagation; conflicting merge facts; call clobbers; AL/AX partial writes; XCHG/MUL implicit or multiple writes; 32-bit LEA; and rejection of unsupported direct-interface array handling. The runner checks baseline byte conservation for every sample and checks that the stripped fixture discovers the known 16-byte cross-block string. Full per-input evidence and new ranges remain in JSON.

This prototype accepts little-endian amd64 ELF `ET_EXEC` only. It does not apply PIE/shared-object relocations, reconstruct dynamic initializer state, perform general alias analysis, infer unsafe pointer targets, recover maps/channels/heaps, or follow arbitrary direct-interface aggregate layouts. The fixture's `init` function creates a cycle back-edge at runtime; that back-edge is intentionally absent from static file analysis. Cyclic-file behavior is separately exercised by a synthetic initialized-memory test. DWARF type declarations plus file ranges justify the static references, but cannot establish exclusive package ownership or deletion savings. Missing/unsupported control flow and instruction semantics limit CFG completeness; ABI pair detection remains heuristic even after the state-kill fixes.

## Decision

The experiment supports prioritizing **typed pointer traversal plus visible object paths**: it found a concrete 29 KiB business asset and 35,358 previously unclassified bytes in a real binary, using a bounded type-driven implementation. A production version should keep reference edges distinct from definition ownership and expose shared objects.

Interface recovery is implementable but sample-dependent; retain conservative candidate labels until more compiler-version fixtures validate each layout. The bounded CFG implementation establishes feasibility, but **192 new real-sample bytes** and zero new bytes in the other real sample do not justify starting with a full-program CFG or decompiler. More instruction analysis should be targeted at measured large residual regions rather than enabled indiscriminately.
