"""Run four independent size-attribution experiments concurrently.

Usage: python experiments/size-attribution/run.py [--workers 4]
       python experiments/size-attribution/run.py --collect

Inputs are existing scripts/bins examples and real integration binaries. The
inputs are read, never executed or overwritten. Generated fixtures are compiled,
not executed. Outputs, binaries and logs live in temp/size-experiments.
"""
from concurrent.futures import ThreadPoolExecutor, as_completed
from datetime import datetime, timezone
import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[2]
HERE = Path(__file__).resolve().parent
OUT = ROOT / "temp/size-experiments"
GROUPS = ("report", "buildassist", "inline", "staticreach")


def collect():
    summaries = {}
    missing = []
    for group in GROUPS:
        path = OUT / group / "summary.json"
        if path.exists():
            summaries[group] = json.loads(path.read_text(encoding="utf-8"))
        else:
            missing.append(group)
    result = {"collected_at_utc": datetime.now(timezone.utc).isoformat(),
              "performance_benchmark": False, "missing_groups": missing,
              "summaries": summaries,
              "source_sha256": {str(p.relative_to(ROOT)): hashlib.sha256(p.read_bytes()).hexdigest()
                                for p in HERE.rglob("*") if p.suffix in (".go", ".py", ".s")}}
    (OUT / "summary.json").write_text(json.dumps(result, indent=2), encoding="utf-8")
    print("Combined results:", OUT / "summary.json", flush=True)
    return missing


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--workers", type=int, default=4, choices=range(1, 5))
    parser.add_argument("--collect", action="store_true", help="combine existing results without rerunning them")
    args = parser.parse_args()
    OUT.mkdir(parents=True, exist_ok=True)
    if args.collect:
        if collect():
            raise SystemExit("Some experiment summaries are missing")
        return
    env = dict(os.environ, GOMAXPROCS="4", PYTHONDONTWRITEBYTECODE="1")
    subprocess.run(["go", "build", "-mod=readonly", "-p=2", "-o", str(OUT / "gsa.exe"),
                    "./cmd/gsa"], cwd=ROOT, env=env, check=True)

    def execute(group):
        log = OUT / f"{group}.run.log"
        with log.open("w", encoding="utf-8") as stream:
            p = subprocess.run([sys.executable, str(HERE / group / "run.py")], cwd=ROOT,
                               env=env, stdout=stream, stderr=subprocess.STDOUT)
        return {"group": group, "exit_code": p.returncode, "log": str(log)}

    state = []
    with ThreadPoolExecutor(max_workers=args.workers) as pool:
        futures = {pool.submit(execute, group): group for group in GROUPS}
        for future in as_completed(futures):
            row = future.result()
            state.append(row)
            print(f"{row['group']}: exit {row['exit_code']} ({row['log']})", flush=True)
    (OUT / "run-state.json").write_text(json.dumps(state, indent=2), encoding="utf-8")
    collect()
    if any(r["exit_code"] for r in state):
        raise SystemExit("One or more experiments failed; inspect run-state.json and logs")


if __name__ == "__main__":
    main()
