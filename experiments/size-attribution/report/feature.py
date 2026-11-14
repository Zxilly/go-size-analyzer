"""A/B build the actual GSA project with its existing embed feature toggle.

Requires report/run.py to have built the experimental report tool. Uses the
existing internal/webui/index.html asset, recording its content identity.
"""
import hashlib
import json
import os
from pathlib import Path
import subprocess

ROOT = Path(__file__).resolve().parents[3]
OUT = ROOT / "temp/size-experiments/report"


def main():
    env = dict(os.environ, GOMAXPROCS="4")
    tool = OUT / ("report.exe" if os.name == "nt" else "report")
    asset = ROOT / "internal/webui/index.html"
    if not tool.exists() or not asset.exists():
        raise SystemExit("Build the report tool and the existing web UI asset first")

    def run(args, label):
        with (OUT / f"{label}.log").open("w", encoding="utf-8") as stream:
            subprocess.run([str(x) for x in args], cwd=ROOT, env=env,
                           stdout=stream, stderr=subprocess.STDOUT, check=True)

    for label, tags in [("default", []), ("embed", ["-tags=embed"])]:
        binary = OUT / f"gsa-{label}.exe"
        run(["go", "build", "-mod=readonly", "-p=2", *tags, "-o", binary, "./cmd/gsa"], f"gsa-{label}-build")
        run([tool, "-out", OUT / f"gsa-{label}.json", binary], f"gsa-{label}-analyze")
    run([tool, "-old", OUT / "gsa-default.json", "-new", OUT / "gsa-embed.json",
         "-out", OUT / "gsa-feature-diff.json", "-html", OUT / "gsa-feature-diff.html"], "gsa-feature-diff")
    diff = json.loads((OUT / "gsa-feature-diff.json").read_text(encoding="utf-8"))
    asset_changes = [c for c in diff["changed"] if c["name"].startswith("github.com/Zxilly/go-size-analyzer/internal/webui.tmpl.")]
    assert len(asset_changes) == 1, asset_changes
    assert asset_changes[0]["delta"] == asset.stat().st_size, asset_changes
    assert "Build setting changed: -tags" in diff["warnings"]
    embedded = json.loads((OUT / "gsa-embed.json").read_text(encoding="utf-8"))
    asset_object = next(o for o in embedded["objects"] if o["id"] == asset_changes[0]["id"])
    physical = asset_object["evidence_sample"][0]["file_ranges"]
    assert len(physical) == 1
    image = (OUT / "gsa-embed.exe").read_bytes()
    assert image[physical[0]["offset"]:physical[0]["offset"]+physical[0]["size"]] == asset.read_bytes()
    result = {"asset_path": str(asset), "asset_sha256": hashlib.sha256(asset.read_bytes()).hexdigest(),
              "asset_size": asset.stat().st_size, "file_delta": diff["file_delta"],
              "changed_objects": len(diff["changed"]), "asset_changes": asset_changes,
              "verified_asset_file_range": physical[0],
              "warnings": diff["warnings"],
              "scope": "Actual project, same source/toolchain; only the existing embed build tag differs. Input executables are never run."}
    (OUT / "feature-summary.json").write_text(json.dumps(result, indent=2), encoding="utf-8")
    summary = json.loads((OUT / "summary.json").read_text(encoding="utf-8"))
    summary["real_project_feature"] = result
    (OUT / "summary.json").write_text(json.dumps(summary, indent=2), encoding="utf-8")
    print(json.dumps(result, indent=2), flush=True)


if __name__ == "__main__":
    main()
