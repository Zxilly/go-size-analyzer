"""Conservative ELF64 sidecar experiment; no production parser changes.

Match architecture, executable entry and ALL allocated section layouts/content.
File offsets need not match after objcopy. NOBITS layout must still match.
This accepts an original ELF as a symbol sidecar, not arbitrary debug-only ELFs.
"""
from __future__ import annotations

import hashlib
import struct
from pathlib import Path


class ELF:
    def __init__(self, path):
        self.path = str(path)
        self.data = Path(path).read_bytes()
        self._functions = None
        if self.data[:6] != b"\x7fELF\x02\x01":
            raise ValueError("experiment supports little-endian ELF64 only")
        header = struct.unpack_from("<HHIQQQIHHHHHH", self.data, 16)
        self.kind, self.machine, _, self.entry = header[:4]
        phoff, phsize, phcount = header[4], header[8], header[9]
        if phsize != 56:
            raise ValueError("unsupported ELF program headers")
        self.loads = []
        for i in range(phcount):
            p = struct.unpack_from("<IIQQQQQQ", self.data, phoff+i*phsize)
            if p[0] == 1:
                self.loads.append(dict(flags=p[1], addr=p[3], physical_addr=p[4], file_size=p[5], memory_size=p[6], alignment=p[7]))
        shoff, shsize, shcount, shstrndx = header[5], header[10], header[11], header[12]
        if shsize != 64 or not shcount or shstrndx >= shcount:
            raise ValueError("unsupported or missing ELF section table")
        raw = [struct.unpack_from("<IIQQQQIIQQ", self.data, shoff + i * shsize) for i in range(shcount)]
        strings = self.data[raw[shstrndx][4]:raw[shstrndx][4]+raw[shstrndx][5]]
        self.sections = []
        for r in raw:
            name = strings[r[0]:].split(b"\0", 1)[0].decode("utf-8", "replace")
            self.sections.append(dict(name=name, type=r[1], flags=r[2], addr=r[3], offset=r[4], size=r[5], link=r[6], entsize=r[9]))
        # Every allocated file-backed byte must map at its declared address.
        # PT_LOAD file_size may change solely in unallocated tail padding after
        # strip; it is not a symbol's extent, so it is validated, not compared.
        programs = [struct.unpack_from("<IIQQQQQQ", self.data, phoff+i*phsize) for i in range(phcount)]
        for s in self.sections:
            if s["flags"] & 2 and s["type"] != 8 and s["size"]:
                if not any(p[0] == 1 and s["offset"] >= p[2] and s["offset"]+s["size"] <= p[2]+p[5]
                           and s["addr"] == p[3]+s["offset"]-p[2] for p in programs):
                    raise ValueError("allocated section does not map to its declared virtual address: "+s["name"])

    def fingerprint(self):
        allocated = []
        for s in self.sections:
            if not s["flags"] & 2:
                continue
            row = {k: s[k] for k in ("name", "type", "flags", "addr", "size")}
            if s["type"] != 8:
                data = self.data[s["offset"]:s["offset"]+s["size"]]
                if len(data) != s["size"]:
                    raise ValueError("truncated allocated section")
                row["sha256"] = hashlib.sha256(data).hexdigest()
            allocated.append(row)
        loads = [{k: v for k, v in p.items() if k != "file_size"} for p in self.loads]
        return dict(machine=self.machine, kind=self.kind, entry=self.entry, loads=loads, sections=allocated)

    def functions(self):
        if self._functions is not None:
            return self._functions
        result = []
        for s in self.sections:
            if s["type"] != 2 or s["entsize"] != 24:
                continue
            strings = self.sections[s["link"]]
            names = self.data[strings["offset"]:strings["offset"]+strings["size"]]
            for off in range(s["offset"], s["offset"]+s["size"], 24):
                name, info, _, shndx, addr, size = struct.unpack_from("<IBBHQQ", self.data, off)
                if info & 15 == 2 and size and 0 < shndx < len(self.sections):
                    end = names.find(b"\0", name)
                    result.append(dict(name=names[name:end].decode("utf-8", "replace"), addr=addr, size=size))
        self._functions = result
        return result


def compare(target: ELF, donor: ELF):
    a, b = target.fingerprint(), donor.fingerprint()
    differences = []
    for k in ("machine", "kind", "entry", "loads"):
        if a[k] != b[k]:
            differences.append(k)
    if a["sections"] != b["sections"]:
        by_a = {s["name"]: s for s in a["sections"]}
        by_b = {s["name"]: s for s in b["sections"]}
        differences.extend("section:"+name for name in sorted(by_a.keys() | by_b.keys()) if by_a.get(name) != by_b.get(name))
    return dict(accepted=not differences, differences=differences,
                target=target.path, donor=donor.path,
                target_sha256=hashlib.sha256(target.data).hexdigest(),
                donor_sha256=hashlib.sha256(donor.data).hexdigest(),
                target_loads=target.loads, donor_loads=donor.loads,
                target_function_symbols=len(target.functions()), donor_function_symbols=len(donor.functions()))


def merged(ranges):
    result = []
    for start, end in sorted(ranges):
        if end <= start:
            continue
        if result and start <= result[-1][1]:
            result[-1][1] = max(result[-1][1], end)
        else:
            result.append([start, end])
    return result


def length(ranges):
    return sum(b-a for a, b in merged(ranges))


def attributed_code(elf, report):
    """Map attributed physical code bytes to virtual address intervals."""
    ranges = []
    for s in elf.sections:
        if s["type"] == 8 or s["flags"] & 6 != 6:
            continue
        lo, hi = s["offset"], s["offset"]+s["size"]
        for r in report["coverage"]["regions"]:
            if r["class"] != "attributed":
                continue
            start, end = max(lo, r["offset"]), min(hi, r["offset"]+r["size"])
            if start < end:
                ranges.append((s["addr"]+start-lo, s["addr"]+end-lo))
    return merged(ranges)


def recovery(target, donor, target_report, donor_report):
    match = compare(target, donor)
    if not match["accepted"]:
        raise ValueError("refusing symbol transfer: " + ", ".join(match["differences"]))
    current, external = attributed_code(target, target_report), attributed_code(donor, donor_report)
    return dict(target_code_attributed=length(current),
                donor_code_attributed=length(external),
                recovered_additional_code_bytes=length(current+external)-length(current),
                combined_code_attributed=length(current+external))
