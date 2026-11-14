"""Exercise production symbol diffs on saved real binaries without executing them."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess

ROOT = Path(__file__).resolve().parents[2]


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--analyzer", type=Path, default=ROOT / "temp/size-iteration2/gsa.exe")
    parser.add_argument("--output-dir", type=Path, default=ROOT / "temp/size-iteration2/symbols")
    args = parser.parse_args()
    args.output_dir.mkdir(parents=True, exist_ok=True)
    compose = ROOT / "scripts/bins/docker-compose-linux-x86_64"
    feature = ROOT / "temp/size-experiments/report"
    pairs = [("compose-self", compose, compose)]
    if (feature / "gsa-default.exe").is_file() and (feature / "gsa-embed.exe").is_file():
        pairs.append(("gsa-feature", feature / "gsa-default.exe", feature / "gsa-embed.exe"))
    results = []
    for label, before, after in pairs:
        hashes = {str(p): hashlib.sha256(p.read_bytes()).hexdigest() for p in (before, after)}
        target = args.output_dir / f"{label}.json"
        with (args.output_dir / f"{label}.log").open("w", encoding="utf-8") as log:
            subprocess.run([str(args.analyzer.resolve()), str(before), str(after), "-f", "json", "-o", str(target.resolve())],
                           stdout=log, stderr=log, check=True, cwd=ROOT, env=dict(os.environ, GOMAXPROCS="4"))
        result = json.loads(target.read_text(encoding="utf-8"))
        symbols = result.get("symbols", [])
        if label == "compose-self":
            assert not symbols and not result.get("functions") and not result.get("packages")
        assets = [s for s in symbols if s["name"].startswith("github.com/Zxilly/go-size-analyzer/internal/webui.tmpl.")]
        if label == "gsa-feature":
            assert len(assets) == 1 and assets[0]["size"]["delta"] > 0, assets
        assert all(hashlib.sha256(Path(p).read_bytes()).hexdigest() == value for p, value in hashes.items())
        results.append({"label": label, "inputs": hashes, "file_delta": result["new_size"] - result["old_size"],
                        "function_changes": len(result.get("functions", [])), "symbol_changes": len(symbols),
                        "symbol_collisions": sum(d["code"] == "symbol_identity_collision" for d in result.get("diagnostics", [])),
                        "assets": assets})
    (args.output_dir / "summary.json").write_text(json.dumps(results, indent=2), encoding="utf-8")
    print(json.dumps(results, indent=2))


if __name__ == "__main__":
    main()
