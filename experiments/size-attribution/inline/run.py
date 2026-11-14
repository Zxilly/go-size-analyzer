"""Reproduce the inline/source and generic emitted-body experiment.

Run from any directory: python experiments/size-attribution/inline/run.py
Only the experiment output directory is written. Requires Go and Python 3.
"""

from __future__ import annotations

import json
import os
from pathlib import Path
import subprocess


ROOT = Path(__file__).resolve().parents[3]
OUT = ROOT / "temp" / "size-experiments" / "inline"
ENV = dict(os.environ, GOMAXPROCS="4")


def run(args: list[str], env: dict[str, str] = ENV) -> None:
    subprocess.run(args, cwd=ROOT, env=env, check=True)


def main() -> None:
    OUT.mkdir(parents=True, exist_ok=True)
    tool = OUT / ("inline.exe" if os.name == "nt" else "inline")
    run(["go", "build", "-mod=readonly", "-p=2", "-o", str(tool),
         "./experiments/size-attribution/inline"])
    samples: list[tuple[str, Path]] = []
    for target in ("linux", "windows", "darwin"):
        fixture = OUT / f"fixture-{target}-amd64"
        run(["go", "build", "-mod=readonly", "-p=2", "-o", str(fixture),
             "./experiments/size-attribution/inline/testdata/fixture.go"],
            dict(ENV, GOOS=target, GOARCH="amd64", CGO_ENABLED="0"))
        samples.append((f"fixture-{target}", fixture))
    stripped_fixture = OUT / "fixture-linux-symbols"
    run(["go", "build", "-mod=readonly", "-p=2", "-ldflags=-w", "-o", str(stripped_fixture),
         "./experiments/size-attribution/inline/testdata/fixture.go"],
        dict(ENV, GOOS="linux", GOARCH="amd64", CGO_ENABLED="0"))
    samples.append(("fixture-linux-symbols", stripped_fixture))
    for name in ("bin-linux-1.27-amd64", "bin-windows-1.23-amd64",
                 "bin-darwin-1.23-amd64", "bin-darwin-1.27-amd64-pie-cgo",
                 "bin-windows-1.23-amd64-strip-cgo",
                 "docker-compose-linux-x86_64", "analysis-server-linux"):
        samples.append((name, ROOT / "scripts" / "bins" / name))
    samples.append(("gsa-current", ROOT / "temp" / "size-experiments" / "gsa.exe"))
    summary = {"experiment": "DWARF inline source overlay and generic emitted bodies",
               "performance_benchmark": False, "samples": [], "missing_samples": []}
    for label, path in samples:
        if not path.exists():
            summary["missing_samples"].append(str(path))
            continue
        print(f"inline: {label}", flush=True)
        report_path = OUT / f"{label}.json"
        run([str(tool), "-output", str(report_path), str(path)])
        with report_path.open(encoding="utf-8") as stream:
            report = json.load(stream)[0]
        groups = report["generic_groups"] or []
        inlines = report["inlines"] or []
        fields = ("file", "sha256", "file_size", "format", "go_version",
                  "dwarf_available", "concrete_functions", "inline_occurrences",
                  "unresolved_inline_names", "unmapped_range_bytes",
                  "inline_union_bytes", "deepest_ambiguous_bytes", "notes")
        item = {key: report.get(key) for key in fields}
        item.update(label=label, report=str(report_path), generic_source=report['generic_source'], generic_declarations=len(groups),
                    generic_bodies=sum(len(g["emitted_bodies"]) for g in groups),
                    generic_unrecognized_names=len(report["generic_unrecognized_names"] or []),
                    dictionary_symbols=len(report["dictionary_symbols"] or []),
                    top_sources=(report["sources"] or [])[:5],
                    top_generic_declarations=groups[:5])
        deepest_total = sum(g["deepest_source_bytes"] for g in (report["sources"] or []))
        item["deepest_projection_conserved"] = (
            deepest_total + report["deepest_ambiguous_bytes"] == report["inline_union_bytes"])
        assert item["deepest_projection_conserved"], label
        if label.startswith("fixture-"):
            fixture_inline = [i for i in inlines
                              if i["host"] == "main.host" and i["name"] == "main.addOne"]
            generic = [g for g in groups if g["declaration"] == "main.genericFirst"]
            if label != "fixture-linux-symbols":
                assert fixture_inline, f"{label}: missing known inline source"
            else:
                assert not report["dwarf_available"] and report["generic_source"] == "native_symbol"
            assert generic and len(generic[0]["emitted_bodies"]) == 2, label
            dictionaries = [name for name in report["dictionary_symbols"]
                            if name.startswith("main..dict.genericFirst[")]
            assert len(dictionaries) == 3, label
            item["fixture_inline_occurrences"] = fixture_inline
            item["fixture_generic_group"] = generic[0]
            item["fixture_dictionaries"] = dictionaries
        summary["samples"].append(item)
    (OUT / "summary.json").write_text(json.dumps(summary, indent=2), encoding="utf-8")
    print(f"Wrote {OUT / 'summary.json'}", flush=True)


if __name__ == "__main__":
    main()
