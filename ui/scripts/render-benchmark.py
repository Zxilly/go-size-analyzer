"""Serial browser measurements; requires the installed agent-browser CLI.

Example (from repository root):
  python ui/scripts/render-benchmark.py --phase baseline --template temp/size-iteration2/ui/baseline/template.html
  python ui/scripts/render-benchmark.py --phase after --template ui/dist/webui/index.html

The same full JSON payload is used in both phases, independently of HTML projection.
Timings and Chromium's approximate heap are observations, not CI thresholds.
"""
import argparse
import functools
import hashlib
import http.server
import json
from pathlib import Path
import shutil
import statistics
import subprocess
import threading

ROOT = Path(__file__).resolve().parents[2]
OUT = ROOT / "temp/size-iteration2/ui"
SAMPLES = ["docker-compose-linux-x86_64", "analysis-server-linux"]
INSTRUMENT = """<script>
new MutationObserver((changes, observer) => {
  if (!document.querySelector('#treemap-svg .node')) return;
  observer.disconnect();
  const committed = performance.now();
  requestAnimationFrame(() => requestAnimationFrame(() => {
    window.__renderMeasurement = {
      commit_ms: committed,
      two_frames_ms: performance.now(),
      node_dom_count: document.querySelectorAll('#treemap-svg .node').length,
      svg_elements: document.querySelectorAll('#treemap-svg *').length,
      heap_used_bytes: performance.memory?.usedJSHeapSize ?? null,
      viewport: [innerWidth, innerHeight],
      user_agent: navigator.userAgent
    };
  }));
}).observe(document, {subtree:true, childList:true});
</script>"""


def summarize():
    before_path = OUT / "baseline-quiet/browser-summary.json"
    after_path = OUT / "after/browser-summary.json"
    if not before_path.exists() or not after_path.exists():
        return
    before = json.loads(before_path.read_text(encoding="utf-8"))
    after = json.loads(after_path.read_text(encoding="utf-8"))
    rows = []
    for sample, stem in zip(SAMPLES, ["compose", "analysis"]):
        old = [row for row in before if row["sample"] == sample]
        new = [row for row in after if row["sample"] == sample]
        if not old or not new:
            continue
        assert {row["sha256"] for row in old + new} == {old[0]["sha256"]}
        row = {"sample": sample, "sha256": old[0]["sha256"], "runs_per_phase": [len(old), len(new)]}
        for field in ["node_dom_count", "svg_elements", "commit_ms", "two_frames_ms", "heap_used_bytes"]:
            row[field] = {"before_median": statistics.median(item[field] for item in old),
                          "after_median": statistics.median(item[field] for item in new)}
        old_h = OUT / f"baseline/{stem}-hierarchy.json"
        new_h = OUT / f"after/{stem}-hierarchy.json"
        if old_h.exists() and new_h.exists():
            row["hierarchy_before"] = json.loads(old_h.read_text(encoding="utf-8"))
            row["hierarchy_after"] = json.loads(new_h.read_text(encoding="utf-8"))
        rows.append(row)
    (OUT / "summary.json").write_text(json.dumps({
        "viewport": [1440, 900],
        "method": "Fresh Chromium session per run; serial navigation; identical full JSON payload; two animation frames after first SVG commit",
        "limits": [
            "Timing observations on one machine, not a performance guarantee.",
            "Browser heap is an approximate sample without forced GC; it does not establish retained-memory reduction.",
            "Retained hierarchy counts exclude temporary sibling nodes used to preserve the complete squarify partition; created_hierarchy_nodes includes them.",
            "Node hierarchy timings are separate one-run observations; viewport hierarchy_ms includes layout, baseline layout_ms is separate.",
        ],
        "samples": rows,
    }, indent=2), encoding="utf-8")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--phase", required=True)
    parser.add_argument("--template", required=True, type=Path)
    parser.add_argument("--runs", type=int, default=3)
    args = parser.parse_args()
    target = OUT / args.phase
    target.mkdir(parents=True, exist_ok=True)
    template = args.template.read_text(encoding="utf-8")
    assert template.count('"GSA_PACKAGE_DATA"') == 1
    binary = shutil.which("agent-browser")
    if not binary:
        raise RuntimeError("agent-browser must be installed")
    # Use the native Windows executable so arguments are not reinterpreted by cmd.
    if binary.endswith(".cmd"):
        native = Path(binary).parent / "node_modules/agent-browser/bin/agent-browser-win32-x64.exe"
        if native.exists():
            binary = str(native)
    server = http.server.ThreadingHTTPServer(("127.0.0.1", 0), functools.partial(
        http.server.SimpleHTTPRequestHandler, directory=str(target)))
    threading.Thread(target=server.serve_forever, daemon=True).start()
    measurements = []
    try:
        for sample in SAMPLES:
            source = ROOT / f"temp/size-iteration/matrix/{sample}.after.json"
            data = source.read_bytes()
            payload = json.dumps(json.loads(data), ensure_ascii=True, separators=(",", ":")).replace("<", "\\u003c")
            html = template.replace('"GSA_PACKAGE_DATA"', payload).replace("<head>", "<head>" + INSTRUMENT, 1)
            (target / f"{sample}.html").write_text(html, encoding="utf-8")
            for run in range(args.runs):
                session = f"gsa-render-{args.phase}-{sample}-{run}"

                def browser(*command):
                    result = subprocess.run([binary, "--session", session, *command], check=True,
                                            capture_output=True, text=True, encoding="utf-8")
                    return result.stdout

                try:
                    browser("open", "about:blank")
                    browser("set", "viewport", "1440", "900")
                    browser("open", f"http://127.0.0.1:{server.server_port}/{sample}.html")
                    browser("wait", "--fn", "Boolean(window.__renderMeasurement)")
                    raw = browser("eval", "--json", "JSON.stringify(window.__renderMeasurement)")
                    result = json.loads(raw)
                    measurement = json.loads(result["data"]["result"])
                    measurement.update(sample=sample, phase=args.phase, run=run + 1,
                                       sha256=hashlib.sha256(data).hexdigest())
                    measurements.append(measurement)
                    print(json.dumps(measurement), flush=True)
                    (target / "browser-summary.json").write_text(json.dumps(measurements, indent=2), encoding="utf-8")
                finally:
                    browser("close")
    finally:
        server.shutdown()
    summarize()


if __name__ == "__main__":
    main()
