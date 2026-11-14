# Reproducing large-report rendering measurements

Run from the repository root. These checks use existing analyzer reports and the installed `agent-browser` CLI; they add no dependencies. Browser runs are serial, start a fresh isolated Chromium session for every sample, fix the viewport at 1440 × 900, and close the browser and local HTTP server afterward.

Freeze the old production template before changing the implementation. For this iteration it is `temp/size-iteration2/ui/baseline/template.html`. Keep the full JSON inputs unchanged in `temp/size-iteration/matrix/` so report projection changes do not contaminate the rendering comparison.

```powershell
pnpm --dir ui run build:ui
python ui/scripts/render-benchmark.py --phase baseline-quiet --template temp/size-iteration2/ui/baseline/template.html
python ui/scripts/render-benchmark.py --phase after --template ui/dist/webui/index.html
```

Each command writes generated reports and `browser-summary.json` under its phase directory in `temp/size-iteration2/ui`. When both phases exist, the script writes `summary.json` with medians, input hashes, counts, and measurement limitations. The recorded initial `baseline` run preceded implementation changes; `baseline-quiet` repeats the frozen old template after other agents finished builds and lints.

The MutationObserver records the first React SVG commit and a subsequent two-frame boundary. `node_dom_count` is the number of actual `.node` SVG groups; `svg_elements` includes rectangles and labels. `performance.memory.usedJSHeapSize` is an approximate sample without forced GC. It is **not** retained memory, and this experiment does not establish a universal memory reduction. Navigation and frame timings are observations from one Windows host, not CI performance gates.

For hierarchy/layout observations, use the opt-in Vitest test, one process per report:

```powershell
$env:GSA_RENDER_REPORT = 'E:/Project/CS_Project/gsv/temp/size-iteration/matrix/docker-compose-linux-x86_64.after.json'
$env:GSA_RENDER_SUMMARY = 'E:/Project/CS_Project/gsv/temp/size-iteration2/ui/after/compose-hierarchy.json'
pnpm --dir ui exec vitest run src/tool/render-performance.test.ts --coverage.enabled=false
```

Repeat with `analysis-server-linux.after.json` and `analysis-hierarchy.json`. `GSA_RENDER_MODE=baseline` selects the full reference hierarchy algorithm still used by accounting tests. The originally captured baseline files predate lazy package creation; use their recorded values for that before/after comparison. In viewport mode, `hierarchy_ms` includes layout; sum `hierarchy_ms + layout_ms` when comparing phases. `hierarchy_nodes` counts retained nodes; `created_hierarchy_nodes` also counts temporary sibling nodes that are required to preserve the complete squarify partition.

The implementation retains exact parent budgets, applies the same original sibling partition, and omits tiny tiles only after their rectangle is known. It does not renormalize visible siblings. Tests compare every retained rectangle and value against the full reference layout. Package entries are lazy, files cache their computed size, and function nodes are created only for the selected file. Very small items remain accessible through the selected parent's searchable, paginated list and direct hash links. Tests also cover file/function navigation, back-to-file, switching reports in Explorer, empty windows, zero sizes, and a 500-level path.

```powershell
pnpm --dir ui run lint
pnpm --dir ui exec vitest run --coverage.enabled=false
pnpm --dir ui run build:ui
pnpm --dir ui run build:explorer
```

For manual integration checks, serve the final native printer reports under `temp/size-iteration2/ui`, open `compose.html`, drill into runtime → symtab.go → a function, test the item filter and back-to-file, and reload its hash URL. The final `--coverage-details` report displays `runtime.moduledataverify1` with 315 referenced bytes and 8 of 22 evidence records, while keeping those references out of its area.
