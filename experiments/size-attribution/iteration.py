"""Compare the production iteration against the preserved experiment baseline.

Never runs or rewrites input binaries. Outputs are ignored under temp/size-iteration.
Concurrent wall times are diagnostic only, not performance benchmarks.
"""
from concurrent.futures import ThreadPoolExecutor
import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess
import time

ROOT = Path(__file__).resolve().parents[2]
OUT = ROOT / "temp/size-iteration/matrix"
SAMPLES = [
    "bin-linux-1.27-amd64", "bin-linux-1.23-amd64-strip",
    "bin-linux-1.23-amd64-cgo", "bin-windows-1.23-amd64",
    "bin-darwin-1.23-amd64", "bin-darwin-1.27-amd64-pie-cgo",
    "bin-js-1.27-wasm", "bin-wasip1-1.27-wasm",
    "analysis-server-linux", "docker-compose-linux-x86_64",
    "bin-windows-1.23-amd64-strip-cgo",
]


def functions(report):
    def walk(packages):
        for package in packages.values():
            for file in package.get("files", []):
                for function in file.get("functions", []):
                    yield package["name"], function
            yield from walk(package.get("subPackages", {}))
    return list(walk(report["packages"]))


def run(analyzer, sample, label, flags=()):
    output = OUT / f"{sample}.{label}.json"
    start = time.monotonic()
    with (OUT / f"{sample}.{label}.log").open("w", encoding="utf-8") as log:
        subprocess.run([str(analyzer), *flags, "-f", "json", "-o", str(output),
                        str(ROOT / "scripts/bins" / sample)], cwd=ROOT,
                       env=dict(os.environ, GOMAXPROCS="4"), stdout=log, stderr=log,
                       check=True)
    report = json.loads(output.read_text(encoding="utf-8"))
    coverage = report["coverage"]
    assert sum(coverage[k] for k in ("attributed", "recognized", "unclassified")) == report["size"]
    return report, time.monotonic() - start, output.stat().st_size


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--analyzer", type=Path, default=ROOT / "temp/size-iteration/gsa.exe")
    parser.add_argument("--baseline", type=Path, default=ROOT / "temp/size-experiments/gsa.exe")
    parser.add_argument("--workers", type=int, default=3)
    args = parser.parse_args()
    OUT.mkdir(parents=True, exist_ok=True)
    for sample in SAMPLES:
        if not (ROOT / "scripts/bins" / sample).is_file():
            raise SystemExit(f"Missing test binary: {sample}")

    def compare(sample):
        binary = ROOT / "scripts/bins" / sample
        before_hash = hashlib.sha256(binary.read_bytes()).hexdigest()
        old, old_seconds, old_size = run(args.baseline, sample, "before")
        new, new_seconds, new_size = run(args.analyzer, sample, "after")
        assert hashlib.sha256(binary.read_bytes()).hexdigest() == before_hash
        assert old["size"] == new["size"]
        old_functions, new_functions = functions(old), functions(new)
        assert len(old_functions) == len(new_functions), sample
        assert sum(f["code_size"] for _, f in old_functions) == sum(f["code_size"] for _, f in new_functions)
        assert all(f.get("full_name") for _, f in new_functions), sample
        if "wasm" in sample:
            assert all("referenced_bytes" not in f for _, f in new_functions)
        else:
            assert all("referenced_bytes" in f for _, f in new_functions)
        output = {
            "sample": sample, "sha256": before_hash, "size": new["size"],
            "old_coverage": {k: old["coverage"][k] for k in ("attributed", "recognized", "unclassified")},
            "new_coverage": {k: new["coverage"][k] for k in ("attributed", "recognized", "unclassified")},
            "attributed_gain": new["coverage"]["attributed"] - old["coverage"]["attributed"],
            "unclassified_reduction": old["coverage"]["unclassified"] - new["coverage"]["unclassified"],
            "functions": len(new_functions),
            "functions_with_references": sum(bool(f.get("referenced_bytes")) for _, f in new_functions),
            "build_info_available": bool(new.get("build_info")),
            "old_json_bytes": old_size, "new_json_bytes": new_size,
            "old_seconds_concurrent": old_seconds, "new_seconds_concurrent": new_seconds,
        }
        print(json.dumps(output), flush=True)
        return output

    with ThreadPoolExecutor(max_workers=args.workers) as pool:
        results = list(pool.map(compare, SAMPLES))

    detailed, _, _ = run(args.analyzer, SAMPLES[0], "details", ["--coverage-details"])
    for _, function in functions(detailed):
        evidence = function.get("reference_evidence", [])
        assert len(evidence) <= 8
        assert len(evidence) <= function.get("reference_evidence_count", 0)
        for claim in evidence:
            assert claim["role"] == "reference"
            assert claim["offset"] + claim["size"] <= detailed["size"]
    unavailable, _, _ = run(args.analyzer, SAMPLES[0], "no-disasm", ["--no-disasm"])
    assert all("referenced_bytes" not in f for _, f in functions(unavailable))
    summary = {"performance_benchmark": False, "analyzer_sha256": hashlib.sha256(args.analyzer.read_bytes()).hexdigest(),
               "baseline_sha256": hashlib.sha256(args.baseline.read_bytes()).hexdigest(), "samples": results}
    (OUT / "summary.json").write_text(json.dumps(summary, indent=2), encoding="utf-8")


if __name__ == "__main__":
    main()
