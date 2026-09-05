# Expanding binary analysis coverage

This document concerns attribution of on-disk binary bytes. The observations below were collected on 2026-09-05, after repairing package coverage mutation, function identity, ELF pclntab attribution, DWARF byte order, and standalone section display.

## Measure distinct outcomes

The analyzer should report three separate quantities:

1. **Attributed bytes:** unique file ranges assigned to packages, functions, symbols, or shared runtime data.
2. **Recognized standalone bytes:** understood structures such as DWARF, symbol tables, relocation tables, and binary headers that have not been assigned to packages.
3. **Unclassified bytes:** remaining file ranges without an explanation.

`Section.KnownSize` currently represents bytes attributed elsewhere. Marking a debug section fully known without assigning its bytes to a package made it disappear from every presentation that subtracts KnownSize. Recognized standalone sections now retain their own area.

Package sizes may include shared ranges. Their sum is not a unique coverage measurement. The treemap normalizes shared package areas while reserving the exact area of standalone sections and file headers; original symbol sizes remain available in tooltips. That presentation rule does not establish exclusive ownership of shared bytes.

## Measurements

Values are bytes. "Attributed" is the current sum of section KnownSize values, which still includes estimated pclntab accounting; it is not yet a precise interval-based coverage metric.

| Sample | Go version | File bytes | Current attributed bytes | Remaining regions worth investigating |
|---|---|---:|---:|---|
| docker-compose-linux-x86_64 | 1.21.11 | 63,083,225 | 50,407,268 | .rodata 4,060,962; .noptrdata 599,472; .data 197,936 |
| analysis-server-linux | 1.22.1 | 60,725,080 | 46,589,675 | .rodata 1,626,570; .text 946,454 |
| bin-js-1.27-wasm | 1.27 | 2,644,952 | 1,968,736 | data 586,092; name 67,652; code 14,563 |
| bin-wasip1-1.27-wasm | 1.27 | 2,554,276 | 1,899,637 | data 567,361; name 65,560; code 14,075 |
| Local PE fixture with a global string | 1.27 | 2,506,752 | 1,585,300 | .data 44,730; investigate inferred symbol boundaries and shared type metadata |

Docker Compose additionally contains 5,548,161 bytes of `.strtab` and 2,120,280 bytes of `.symtab`. The analysis-server sample contains 7,122,432 bytes of DWARF. These are recognized standalone structures, not unexplained payloads.

The sum of package sizes for Docker Compose is 56,568,151 bytes, exceeding the section attribution sum by 6,160,883 bytes. This is direct evidence that a package-sum percentage would overstate unique coverage.

## Recommended sequence

### 1. Establish a file-range accounting ledger

Use half-open file-offset intervals as the accounting unit, retaining analyzer provenance and one or more owners. Convert virtual addresses through the actual file mappings. For WASM, retain encoded section/function/data-segment offsets alongside linear-memory addresses.

Keep package-local coverage separate from immutable global coverage. Compute unique covered bytes by interval union and distinguish shared bytes from exclusive ownership. Expose unclassified intervals per section with reasons such as missing symbols, unsupported architecture, unsupported location expression, or unmapped data.

Pclntab particularly needs this: `PclnSymbolSize` currently adds per-function names and PC-data sizes, which can reference shared bytes, and section accounting clamps overestimates. Extend the gosym/gore integration to expose actual byte ranges before advertising an exact coverage percentage.

Acceptance criteria:

- Every counted interval is within a file-backed range; BSS and virtual memory are excluded.
- Attributed, recognized standalone, and unclassified intervals partition the physical file without overlap.
- Analyzer execution order and repeated runs do not change byte totals.
- Shared symbols and type auxiliary records do not increase global unique coverage twice.

### 2. Recover available ELF writable-data symbols

`ElfWrapper.LoadSymbols` currently recognizes read-only allocated data but skips `SHF_ALLOC | SHF_WRITE`. Supporting bounded, file-backed writable symbols can recover global variables even when DWARF was removed but the symbol table remains.

A direct symbol-table scan of Docker Compose found:

| Section | Symbols | Union of symbol ranges |
|---|---:|---:|
| .data | 2,188 | 60,209 B |
| .noptrdata | 952 | 307,761 B |

The approximately 368 KB total is a candidate range, not a promised incremental gain: intersect it with the ledger's current gaps first. Exclude BSS and linker boundary markers. Handle packages containing only data, including Go DWARF compilation units that currently create a Package without inserting it into the dependency trie.

### 3. Recover external C/C++ definitions

`EntryShouldIgnore` treats `DW_AT_external` on a subprogram as grounds to skip it. External linkage can describe a real definition with address ranges; `DW_AT_declaration` and missing ranges should be handled separately.

The analysis-server DWARF contains 357 non-Go external definitions whose ranges union to 55,479 bytes inside `.text`. This is a bounded opportunity within the 946,454-byte text remainder. Further coverage can come from ordinary ELF function symbols for code absent from Go pclntab, including binaries without DWARF.

Preserve non-contiguous function ranges rather than disassembling `[first address, first address + sum of sizes)`. Validate against C/C++ fixtures with external definitions, declarations, aliases, and split functions.

### 4. Add validated string and static-data extraction

The native disassembler currently implements amd64 RIP-relative LEA followed by an immediate length. Negative RIP displacements are now sign-extended correctly. Additional useful patterns are:

- AMD64 loads of a static string header: a pointer and length from adjacent storage, followed through the applicable relocation mapping.
- ARM64 ADRP/ADD or literal loads combined with an immediate length, using register tracking within a bounded basic block.
- Static string and byte-slice headers identified by DWARF or type/relocation information.

Require a file-backed data range, a bounded length, and evidence of a matching pointer/length pair. UTF-8 validity alone is insufficient; machine words and other metadata can resemble strings. Record heuristic provenance and measure false positives as well as recovered bytes.

The multi-megabyte `.rodata` remainders make this more promising than optimizations that merely relabel metadata sections.

### 5. Expand WASM attribution using semantic and encoded information

The largest remaining WASM region is the data section. Explore statically recoverable data references in watgo IR and global pointer/length records, then intersect them with actual active data segments. Do not count zero-initialized memory as file bytes.

For code, distinguish function instructions from local declarations, body-length encodings, section counts, imported functions, and generated wrappers. The current 14–15 KB code remainder provides a small, measurable target. The 65–68 KB name sections can be recognized immediately and optionally attributed using function indices.

Preserve multiple custom sections with identical names when introducing a physical ledger; the current name-keyed section map is not sufficient to represent every valid WASM file.

## Validation and performance

Use fixtures spanning ELF, PE, Mach-O, js/wasm, wasip1/wasm, stripped and DWARF-only-stripped builds, PIE, CGO, and both byte orders. For each new analyzer, compare unique new intervals against the pre-existing uncovered intervals and check that known zero-initialized memory stays excluded.

Keep speed measurements separate from attribution changes. The fixed-window amd64 extractor was compared against the previous full-instruction-slice implementation on 75,714 functions, producing identical 37,696 candidate records. The retained `BenchmarkExtractBinaryFunctions` uses a real Go ELF fixture; three local runs reduced allocation from about 71.0 MB/op to 1.73 MB/op while retaining 1,279 candidates/op. The local median extraction time changed from about 34.1 ms/op to 20.9 ms/op.

Prioritize the accounting ledger and writable-data/C-definition improvements, then implement architecture and WASM extraction. Avoid claiming 100% coverage by marking entire recognizable sections as package-attributed.
