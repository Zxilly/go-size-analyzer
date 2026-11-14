"""Artifact acceptance tests. Run run.py first, then unittest discover here."""
import json
from pathlib import Path
import struct
import tempfile
import unittest

from elfmatch import ELF, compare, recovery

ROOT = Path(__file__).resolve().parents[3]
OUT = ROOT / "temp/size-experiments/buildassist"


class SidecarSafetyTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.fixture = ELF(OUT / "fixture.elf")

    def changed_elf(self, data):
        temporary = tempfile.TemporaryDirectory(dir=OUT)
        self.addCleanup(temporary.cleanup)
        path = Path(temporary.name) / "changed.elf"
        path.write_bytes(data)
        return ELF(path)

    def test_identical_binary_accepted(self):
        self.assertTrue(compare(self.fixture, self.fixture)["accepted"])

    def test_changed_code_rejected_and_transfer_refused(self):
        data = bytearray(self.fixture.data)
        text = next(s for s in self.fixture.sections if s["name"] == ".text")
        data[text["offset"]+8] ^= 1
        wrong = self.changed_elf(data)
        self.assertIn("section:.text", compare(self.fixture, wrong)["differences"])
        with self.assertRaisesRegex(ValueError, "refusing symbol transfer"):
            recovery(self.fixture, wrong, {}, {})

    def test_changed_readonly_data_rejected(self):
        data = bytearray(self.fixture.data)
        rodata = next(s for s in self.fixture.sections if s["name"] == ".rodata")
        data[rodata["offset"]+8] ^= 1
        self.assertFalse(compare(self.fixture, self.changed_elf(data))["accepted"])

    def test_changed_machine_rejected(self):
        data = bytearray(self.fixture.data)
        struct.pack_into("<H", data, 18, 183)
        self.assertIn("machine", compare(self.fixture, self.changed_elf(data))["differences"])

    def test_changed_load_permissions_rejected(self):
        data = bytearray(self.fixture.data)
        phoff = struct.unpack_from("<Q", data, 32)[0]
        phnum = struct.unpack_from("<H", data, 56)[0]
        for i in range(phnum):
            off = phoff+i*56
            if struct.unpack_from("<I", data, off)[0] == 1:
                flags = struct.unpack_from("<I", data, off+4)[0]
                struct.pack_into("<I", data, off+4, flags ^ 2)
                break
        self.assertIn("loads", compare(self.fixture, self.changed_elf(data))["differences"])


class EvidenceTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.result = json.loads((OUT / "results.json").read_text(encoding="utf-8"))

    def test_dumpdep_omits_one_proven_incoming_call(self):
        witness = self.result["witness"]
        self.assertEqual(witness["disassembled_calls_to_shared"], 2)
        self.assertEqual(len(witness["shared_witness_parents"]), 1)

    def test_real_project_has_witness_paths(self):
        gsa = self.result["witness"]["real_gsa"]
        self.assertGreater(gsa["witness_edges"], 1000)
        self.assertTrue(gsa["sample_witness_paths"])

    def test_all_matching_copies_and_wrong_version(self):
        sidecar = self.result["sidecar"]
        self.assertEqual(len(sidecar["matched_stripped_copies"]), 6)
        for row in sidecar["matched_stripped_copies"]:
            self.assertTrue(row["match"]["accepted"])
            self.assertGreaterEqual(row["recovery"]["recovered_additional_code_bytes"], 0)
        self.assertFalse(sidecar["wrong_version_rejection"]["accepted"])

    def test_same_source_version_is_not_a_matching_sidecar(self):
        pairs = self.result["sidecar"]["separate_build_pairs"]
        self.assertEqual(len(pairs), 4)
        self.assertTrue(all(not p["accepted"] for p in pairs))

    def test_cgo_sidecar_recovers_observed_code(self):
        rows = {r["label"]: r for r in self.result["sidecar"]["matched_stripped_copies"]}
        self.assertGreater(rows["example-cgo-go1.23"]["recovery"]["recovered_additional_code_bytes"], 900000)
        self.assertGreater(rows["analysis-server"]["recovery"]["recovered_additional_code_bytes"], 900000)
        self.assertEqual(rows["docker-compose"]["recovery"]["recovered_additional_code_bytes"], 0)


if __name__ == "__main__":
    unittest.main()
