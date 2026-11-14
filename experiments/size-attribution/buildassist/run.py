"""Run build-assisted retention and strict ELF sidecar experiments.

From repository root: python experiments/size-attribution/buildassist/run.py
Requires go, llvm-objcopy, and temp/size-experiments/gsa.exe.
All generated binaries/reports stay under temp/size-experiments/buildassist.
"""
from __future__ import annotations

import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import subprocess

from elfmatch import ELF, compare, recovery

ROOT = Path(__file__).resolve().parents[3]
OUT = ROOT / "temp/size-experiments/buildassist"
GSA = ROOT / "temp/size-experiments/gsa.exe"
ENV = dict(os.environ, GOMAXPROCS="4", GOOS="linux", GOARCH="amd64", CGO_ENABLED="0")


def command(args, name, env=None):
    p = subprocess.run([str(x) for x in args], cwd=ROOT, env=env or ENV, text=True, encoding="utf-8", errors="replace", capture_output=True)
    (OUT / (name + ".stdout.txt")).write_text(p.stdout, encoding="utf-8")
    (OUT / (name + ".stderr.txt")).write_text(p.stderr, encoding="utf-8")
    if p.returncode:
        raise RuntimeError(f"{args[0]} failed ({p.returncode}): {p.stderr[-3000:]}")
    return p.stdout + p.stderr


def manifest(path, build_command=None):
    return dict(path=str(path), sha256=hashlib.sha256(path.read_bytes()).hexdigest(),
                build_info=command(["go", "version", "-m", path], path.name+".buildinfo"),
                build_command=build_command, inspector_go_version=command(["go", "version"], "go-version").strip(),
                build_environment={k: ENV[k] for k in ("GOMAXPROCS", "GOOS", "GOARCH", "CGO_ENABLED")} if build_command else None)


def analyze(path, label):
    result = OUT / (label+".report.json")
    command([GSA, path, "-f", "json", "--compact", "--coverage-details", "-o", result], label+".gsa")
    return json.loads(result.read_text(encoding="utf-8"))


def witness_experiment():
    fixture = OUT / "fixture.elf"
    args = ["go", "build", "-mod=readonly", "-p=2", "-ldflags=-dumpdep", "-o", fixture,
            "./experiments/size-attribution/buildassist/fixture"]
    dump = command(args, "fixture.build")
    edges = [line.split(" -> ", 1) for line in dump.splitlines() if " -> " in line]
    witnesses = [a for a, b in edges if b == "main.shared"]
    disasm = command(["go", "tool", "objdump", "-s", r"main\.caller[AB]", fixture], "fixture.objdump")
    actual = re.findall(r"TEXT (main\.caller[AB])\(SB\)[\s\S]*?(?=TEXT |\Z)", disasm)
    calls = disasm.count("CALL main.shared(SB)")
    assert sorted(actual) == ["main.callerA", "main.callerB"], actual
    assert calls == 2, disasm
    assert len(witnesses) == 1, witnesses
    result = dict(manifest=manifest(fixture, [str(x) for x in args]),
                  dumpdep_edges=len(edges), shared_witness_parents=witnesses,
                  disassembled_callers=actual, disassembled_calls_to_shared=calls,
                  conclusion="dumpdep is a first-mark witness forest, not the complete retention/reference graph; do not compute dominator savings from it")
    gsa = OUT / "gsa-linux.elf"
    args = ["go", "build", "-mod=readonly", "-p=2", "-ldflags=-dumpdep", "-o", gsa, "./cmd/gsa"]
    dump = command(args, "gsa.build")
    edges = [line.split(" -> ", 1) for line in dump.splitlines() if " -> " in line]
    parents = {b: a for a, b in edges}
    targets = [b for a, b in edges if b.startswith("github.com/go-delve/delve") and "." in b]
    chains = []
    for target in targets[:3]:
        chain, seen = [target], {target}
        while chain[-1] in parents and parents[chain[-1]] not in seen:
            chain.append(parents[chain[-1]])
            seen.add(chain[-1])
        chains.append(list(reversed(chain)))
    result["real_gsa"] = dict(manifest=manifest(gsa, [str(x) for x in args]),
                              witness_edges=len(edges), sample_witness_paths=chains,
                              witness_paths_are_non_exclusive=True)
    return result


def sidecar_experiment():
    sources = [
        ("example-go1.23", ROOT / "scripts/bins/bin-linux-1.23-amd64"),
        ("example-cgo-go1.23", ROOT / "scripts/bins/bin-linux-1.23-amd64-cgo"),
        ("example-cgo-go1.27", ROOT / "scripts/bins/bin-linux-1.27-amd64-cgo"),
        ("docker-compose", ROOT / "scripts/bins/docker-compose-linux-x86_64"),
        ("analysis-server", ROOT / "scripts/bins/analysis-server-linux"),
        ("gsa-source-build", OUT / "gsa-linux.elf"),
    ]
    rows = []
    for label, source in sources:
        print("sidecar:", label, flush=True)
        stripped = OUT / (label+".stripped.elf")
        command(["llvm-objcopy", "--strip-all", source, stripped], label+".strip")
        donor, target = ELF(source), ELF(stripped)
        match = compare(target, donor)
        rejected_strip_attempts = []
        if not match["accepted"]:
            rejected_strip_attempts.append(dict(tool="llvm-objcopy", match=match))
            command(["objcopy", "--strip-all", source, stripped], label+".gnu-strip")
            target = ELF(stripped)
            match = compare(target, donor)
        assert match["accepted"], match
        donor_report, target_report = analyze(source, label+".donor"), analyze(stripped, label+".target")
        row = dict(label=label, match=match, rejected_strip_attempts=rejected_strip_attempts,
                   recovery=recovery(target, donor, target_report, donor_report),
                   donor_coverage={k: v for k, v in donor_report["coverage"].items() if k != "regions"},
                   target_coverage={k: v for k, v in target_report["coverage"].items() if k != "regions"},
                   donor_manifest=manifest(source),
                   scope="original binary used as matching sidecar for an experimentally stripped copy; not a located historical production sidecar")
        rows.append(row)
        print(row["recovery"], flush=True)
        del donor_report, target_report

    separately_built = []
    for version, suffix in [("1.19", ""), ("1.21", ""), ("1.23", ""), ("1.23", "-cgo")]:
        donor = ELF(ROOT / f"scripts/bins/bin-linux-{version}-amd64{suffix}")
        target = ELF(ROOT / f"scripts/bins/bin-linux-{version}-amd64-strip{suffix}")
        separately_built.append(compare(target, donor))
    wrong = compare(ELF(OUT / "example-go1.23.stripped.elf"), ELF(ROOT / "scripts/bins/bin-linux-1.27-amd64"))
    assert not wrong["accepted"]
    return dict(matched_stripped_copies=rows, separate_build_pairs=separately_built, wrong_version_rejection=wrong,
                limitations=["ELF64 little-endian only", "full unstripped ELF sidecars only; detached debug-file build-id/debuglink protocol not implemented",
                             "matching covers all allocated section content/layout, validated virtual-address mappings, load permissions/layout, architecture and entry; allows changed nonallocated sections and unmapped-symbol tail padding",
                             "experimental sidecar transfer only quantifies executable-byte attribution via existing GSA reports; no production parser integration",
                             "historical Compose/analysis-server artifacts have no independently supplied matching sidecar in this repository"])


def main():
    OUT.mkdir(parents=True, exist_ok=True)
    if not GSA.exists():
        raise SystemExit("build baseline temp/size-experiments/gsa.exe first")
    result = dict(witness=witness_experiment())
    (OUT / "witness.json").write_text(json.dumps(result["witness"], indent=2), encoding="utf-8")
    result["sidecar"] = sidecar_experiment()
    (OUT / "results.json").write_text(json.dumps(result, indent=2), encoding="utf-8")
    summary = dict(witness_shared_parents=result["witness"]["shared_witness_parents"],
                   disassembled_shared_call_count=result["witness"]["disassembled_calls_to_shared"],
                   real_gsa_witness_edges=result["witness"]["real_gsa"]["witness_edges"],
                   sidecar=[dict(label=r["label"], accepted=r["match"]["accepted"], **r["recovery"]) for r in result["sidecar"]["matched_stripped_copies"]],
                   separate_build_pairs=[dict(target=r["target"], accepted=r["accepted"], differences=r["differences"]) for r in result["sidecar"]["separate_build_pairs"]],
                   wrong_version_rejected=not result["sidecar"]["wrong_version_rejection"]["accepted"])
    (OUT / "summary.json").write_text(json.dumps(summary, indent=2), encoding="utf-8")
    print("Results:", OUT / "results.json", flush=True)


if __name__ == "__main__":
    main()
