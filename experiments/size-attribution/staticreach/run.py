"""Reproduce bounded static-pointer/interface and CFG experiments; never execute inputs."""
from __future__ import annotations

import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--skip-build", action="store_true")
    parser.add_argument("--out", type=Path)
    args = parser.parse_args()
    root = Path(__file__).resolve().parents[3]
    out = args.out or root / "temp/size-experiments/staticreach"
    out.mkdir(parents=True, exist_ok=True)
    env = os.environ.copy()
    env["GOMAXPROCS"] = "4"
    exe = out / ("staticreach.exe" if os.name == "nt" else "staticreach")
    package = "./experiments/size-attribution/staticreach"

    def run(command: list[str], name: str, custom_env: dict[str, str] | None = None) -> None:
        with (out / f"{name}.log").open("w", encoding="utf-8") as log:
            subprocess.run(command, cwd=root, env=custom_env or env, stdout=log, stderr=subprocess.STDOUT, check=True)

    if not args.skip_build:
        run(["go", "test", "-mod=readonly", "-p=2", package, "-count=1"], "tests")
        run(["go", "build", "-mod=readonly", "-p=2", "-o", str(exe), package], "build")
        target = {**env, "GOOS": "linux", "GOARCH": "amd64", "CGO_ENABLED": "0"}
        for suffix, flags in [("", []), ("-stripped", ["-ldflags=-s -w"])]:
            run(["go", "build", "-mod=readonly", "-p=2", *flags, "-o", str(out / f"fixture{suffix}.elf"), package + "/fixture"], "build-fixture" + suffix, target)

    inputs = [out / "fixture.elf", out / "fixture-stripped.elf"]
    inputs += [root / "scripts/bins" / name for name in [
        "bin-linux-1.27-amd64", "bin-linux-1.23-amd64-strip",
        "docker-compose-linux-x86_64", "analysis-server-linux",
    ]]
    rows = []
    for binary in inputs:
        name = binary.name
        report = out / f"{name}.json"
        run([str(exe), "-out", str(report), str(binary)], name)
        data = json.loads(report.read_text(encoding="utf-8"))
        row = {"input": name, "sha256": hashlib.sha256(binary.read_bytes()).hexdigest(), "report": str(report),
               "size": data["file_size"], "baseline": data["baseline"], "dwarf_roots": data["dwarf_roots"],
               "cycle_or_shared_revisits": data["cycle_or_shared_revisits"], "functions_scanned": data["functions_scanned"],
               "functions_over_budget": data["functions_over_budget"], "notes": data["notes"]}
        for key in ["typed_static_references", "interface_candidates", "cfg_string_candidates", "cfg_cross_block_candidates"]:
            row[key] = {k: data[key][k] for k in ["candidate_count", "unique_file_bytes", "already_attributed_bytes", "new_reference_bytes", "new_from_recognized_bytes", "new_from_unclassified_bytes"]}
        assert sum(data["baseline"].values()) == data["file_size"]
        rows.append(row)
        print(json.dumps(row, ensure_ascii=False), flush=True)
    (out / "summary.json").write_text(json.dumps({"scope": "read-only analysis of input binaries", "results": rows}, indent=2) + "\n", encoding="utf-8")
    stripped = rows[1]
    assert stripped["dwarf_roots"] == 0
    assert stripped["cfg_cross_block_candidates"]["new_reference_bytes"] >= 16, "fixture's known cross-block string not discovered"


if __name__ == "__main__":
    main()
