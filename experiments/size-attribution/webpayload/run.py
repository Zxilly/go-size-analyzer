"""Compare an old CLI's HTML payload with the current web projection.

All generated reports/binaries stay in --output-dir. Baseline binaries and
input samples are read only. Comparisons use the baseline's full JSON as the
new projector input, rather than rerunning a potentially changed analyzer.
"""
import argparse
import hashlib
from html.parser import HTMLParser
import json
import os
from pathlib import Path
import re
import subprocess

ROOT = Path(__file__).resolve().parents[3]
ENV = dict(os.environ, GOMAXPROCS="4")
DEFAULT_SAMPLES = [
    ("example", "bin-linux-1.23-amd64"),
    ("compose", "docker-compose-linux-x86_64"),
    ("analysis-server", "analysis-server-linux"),
]


class DataScript(HTMLParser):
    def __init__(self):
        super().__init__(convert_charrefs=False)
        self.active = False
        self.payload = ""

    def handle_starttag(self, tag, attrs):
        if tag == "script" and dict(attrs).get("id") == "data":
            self.active = True

    def handle_endtag(self, tag):
        if tag == "script":
            self.active = False

    def handle_data(self, data):
        if self.active:
            self.payload += data


def functions(report):
    result = []

    def visit(packages):
        for package in packages.values():
            for file in package["files"]:
                result.extend(file.get("functions", []))
            visit(package["subPackages"])

    visit(report["packages"])
    return result


def compare_case(out, label):
    oldhtml = (out / (label + ".old.html")).read_text(encoding="utf8")
    newhtml = (out / (label + ".new.html")).read_text(encoding="utf8")
    old, new = DataScript(), DataScript()
    old.feed(oldhtml)
    new.feed(newhtml)
    assert old.payload and new.payload, label
    before, after = json.loads(old.payload), json.loads(new.payload)
    direct = json.loads((out / (label + ".new.web.json")).read_text(encoding="utf8"))
    assert after == direct, "HTML data differs from the web JSON projection"
    old_functions, new_functions = functions(before), functions(after)
    assert len(old_functions) == len(new_functions), label
    for old_fn, new_fn in zip(old_functions, new_functions):
        expected = {k: v for k, v in old_fn.items() if k not in ("pcln_size", "ranges")}
        if "pcln_bytes" not in expected:
            expected["pcln_bytes"] = sum(
                sum(v.values()) if isinstance(v, dict) else v
                for v in old_fn.get("pcln_size", {}).values()
            )
        assert expected == new_fn, (label, old_fn.get("name"))
    old_regions = before["coverage"].pop("regions", [])
    assert before["coverage"] == after["coverage"], label
    assert "regions" not in after["coverage"], label
    original = json.loads((out / (label + ".full.json")).read_text(encoding="utf8"))
    roundtrip = json.loads((out / (label + ".new.full-roundtrip.json")).read_text(encoding="utf8"))
    assert original == roundtrip, "full CLI JSON changed"
    compact = json.loads((out / (label + ".new.compact.json")).read_text(encoding="utf8"))
    assert compact["coverage"] == original["coverage"], "compact lost requested regions"
    assert not functions(compact), "compact unexpectedly contains functions"
    old_bytes, new_bytes = len(old.payload.encode()), len(new.payload.encode())
    result = dict(
        sample=label,
        old_payload_bytes=old_bytes,
        new_payload_bytes=new_bytes,
        reduction_bytes=old_bytes-new_bytes,
        reduction_percent=round((old_bytes-new_bytes)/old_bytes*100, 2),
        old_html_bytes=(out / (label + ".old.html")).stat().st_size,
        new_html_bytes=(out / (label + ".new.html")).stat().st_size,
        template_equal=oldhtml.replace(old.payload, "") == newhtml.replace(new.payload, ""),
        functions=len(new_functions),
        function_evidence_records=sum(len(f.get("reference_evidence", [])) for f in new_functions),
        omitted_regions=len(old_regions),
        full_json_unchanged=True,
        compact_keeps_requested_regions=True,
        new_payload_sha256=hashlib.sha256(new.payload.encode()).hexdigest(),
    )
    (out / (label + ".new.extracted.json")).write_text(new.payload, encoding="utf8")
    return result


def run(args, log):
    process = subprocess.run([str(a) for a in args], cwd=ROOT, env=ENV,
                             capture_output=True, text=True, encoding="utf8", errors="replace")
    log.write_text(process.stdout + process.stderr, encoding="utf8")
    if process.returncode:
        raise RuntimeError(f"Command failed ({process.returncode}); see {log}")


def parse_sample(value):
    label, separator, path = value.partition("=")
    if not separator or not re.fullmatch(r"[A-Za-z0-9_-]+", label) or not path:
        raise argparse.ArgumentTypeError("sample must be LABEL=PATH; labels use letters, digits, _ or -")
    return label, Path(path).resolve()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--baseline-cli", type=Path, default=ROOT / "temp/size-iteration/gsa-embedded.exe")
    parser.add_argument("--sample", action="append", type=parse_sample, help="repeat LABEL=PATH; defaults to three repository integration samples")
    parser.add_argument("--output-dir", type=Path, default=ROOT / "temp/size-iteration2/transport")
    parser.add_argument("--compare-only", action="store_true", help="validate existing artifacts without rebuilding or analyzing inputs")
    args = parser.parse_args()
    out = args.output_dir.resolve()
    out.mkdir(parents=True, exist_ok=True)
    samples = args.sample or [(label, ROOT / "scripts/bins" / name) for label, name in DEFAULT_SAMPLES]
    if len({label for label, _ in samples}) != len(samples):
        parser.error("sample labels must be unique")
    generated_labels = [label + suffix for label, _ in samples for suffix in ("", "-default")]
    if len(set(generated_labels)) != len(generated_labels):
        parser.error("sample labels overlap generated -default artifact names")
    projector = out / ("webpayload.exe" if os.name == "nt" else "webpayload")
    if not args.compare_only:
        baseline = args.baseline_cli.resolve()
        if baseline == projector:
            parser.error("baseline CLI cannot be the new projector output")
        if not baseline.is_file():
            parser.error(f"baseline CLI missing: {baseline}")
        run(["go", "build", "-mod=readonly", "-p=2", "-tags=embed", "-o", projector,
             "./experiments/size-attribution/webpayload"], out / "projector-build.log")
    summary = []
    for details in (True, False):
        for sample_label, sample in samples:
            label = sample_label if details else sample_label + "-default"
            if not args.compare_only:
                flags = ["--coverage-details"] if details else []
                run([baseline, sample, *flags, "-o", f"json={out / (label+'.full.json')}",
                     "-o", f"html={out / (label+'.old.html')}"], out / (label + ".baseline.log"))
                run([projector, "-input", out / (label + ".full.json"), "-output-base", out / label],
                    out / (label + ".projection.log"))
            row = compare_case(out, label)
            summary.append(row)
            print(json.dumps(row), flush=True)
    (out / "summary.json").write_text(json.dumps(summary, indent=2), encoding="utf8")


if __name__ == "__main__":
    main()
