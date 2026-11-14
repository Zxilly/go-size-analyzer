"""Run object-evidence/diff experiments against local integration binaries.

From the repository root: python experiments/size-attribution/report/run.py
Only the standard library is required. Artifacts are written under temp/.
"""
import json
import os
from pathlib import Path
import subprocess

ROOT = Path(__file__).resolve().parents[3]
OUT = ROOT / "temp/size-experiments/report"
ENV = dict(os.environ, GOMAXPROCS="4")


def run(*args, log=None):
    p = subprocess.run([str(x) for x in args], cwd=ROOT, env=ENV,
                       stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
    if log:
        Path(log).write_text(p.stdout, encoding="utf-8")
    if p.returncode:
        raise RuntimeError(p.stdout[-5000:])
    print(p.stdout.strip().splitlines()[-1] if p.stdout.strip() else "Completed: " + str(args[0]), flush=True)


def main():
    OUT.mkdir(parents=True, exist_ok=True)
    exe = OUT / ("report.exe" if os.name == "nt" else "report")
    run("go", "build", "-mod=readonly", "-p=2", "-o", exe,
        "./experiments/size-attribution/report")
    samples = [
        "bin-linux-1.23-amd64", "bin-linux-1.27-amd64",
        "bin-linux-1.23-amd64-strip", "bin-windows-1.23-amd64-strip-cgo",
        "bin-darwin-1.27-amd64-pie-cgo", "bin-js-1.27-wasm",
        "docker-compose-linux-x86_64", "analysis-server-linux",
    ]
    summary = []
    for sample in samples:
        binary = ROOT / "scripts/bins" / sample
        if not binary.exists():
            raise FileNotFoundError(binary)
        run(exe, "-out", OUT / f"{sample}.json", "-html", OUT / f"{sample}.html",
            binary, log=OUT / f"{sample}.log")
        r = json.loads((OUT / f"{sample}.json").read_text(encoding="utf-8"))
        assert r["object_physical_union"] <= r["coverage"]["attributed"], sample
        assert sum(r["coverage"][k] for k in ("attributed", "recognized", "unclassified")) == r["size"], sample
        summary.append({"sample": sample, "manifest": r["manifest"], "size": r["size"],
                        "coverage": r["coverage"], "objects": len(r["objects"]),
                        "functions": sum(o["kind"] == "function" for o in r["objects"]),
                        "objects_with_references": sum(o["reference_bytes"] > 0 for o in r["objects"]),
                        "object_physical_union": r["object_physical_union"],
                        "json_size": (OUT / f"{sample}.json").stat().st_size})
        run(exe, "-old", OUT / f"{sample}.json", "-new", OUT / f"{sample}.json",
            "-out", OUT / f"{sample}.self-diff.json")

    run(exe, "-old", OUT / "bin-linux-1.23-amd64.json", "-new", OUT / "bin-linux-1.27-amd64.json",
        "-out", OUT / "go-version-diff.json", "-html", OUT / "go-version-diff.html")
    # Both builds use the same source path and options. Constants exchange size,
    # so the package/file total can conceal large opposing object-level changes.
    source = OUT / "fixture.go"
    for label, alpha, beta in [("old", 2048, 8192), ("new", 8192, 2048)]:
        source.write_text("package main\n//go:noinline\nfunc alpha() string { return " +
                          json.dumps("A" * alpha) + " }\n//go:noinline\nfunc beta() string { return " +
                          json.dumps("B" * beta) + " }\nfunc main(){println(alpha(),beta())}\n", encoding="utf-8")
        binary = OUT / ("fixture-" + label + ".exe")
        run("go", "build", "-mod=readonly", "-p=2", "-o", binary, source)
        run(exe, "-out", OUT / f"fixture-{label}.json", binary,
            log=OUT / f"fixture-{label}.log")
    run(exe, "-old", OUT / "fixture-old.json", "-new", OUT / "fixture-new.json",
        "-out", OUT / "fixture-diff.json", "-html", OUT / "fixture-diff.html")
    d = json.loads((OUT / "fixture-diff.json").read_text(encoding="utf-8"))
    target = [c for c in d["changed"] if c["package"] == "main" and c["name"] in ("alpha", "beta", "main.alpha", "main.beta")]
    assert len(target) == 2, target
    assert sorted(c["reference_delta"] for c in target) == [-6144, 6144], target
    (OUT / "summary.json").write_text(json.dumps({"samples": summary,
        "fixture": {"file_delta": d["file_delta"], "changes": target}}, indent=2), encoding="utf-8")
    from feature import main as run_feature
    run_feature()
    print("Artifacts:", OUT)


if __name__ == "__main__":
    main()
