import csv
import os.path
from enum import Flag, Enum, auto

import requests
from tqdm import tqdm

from utils import *


class TestType(Flag):
    TEXT_TEST = auto()
    JSON_TEST = auto()
    HTML_TEST = auto()
    SVG_TEST = auto()


def get_flag_str(typ: TestType) -> str:
    if typ == TestType.TEXT_TEST:
        return "text"
    elif typ == TestType.JSON_TEST:
        return "json"
    elif typ == TestType.HTML_TEST:
        return "html"
    elif typ == TestType.SVG_TEST:
        return "svg"


class IntegrationTest:
    def __init__(self, name: str, path: str, typ: TestType):
        self.name = name
        self.path = path
        self.type = typ

    @property
    def base_dir(self):
        return os.path.join(
            get_project_root(),
            "results",
            self.name)

    def typed_dir(self, typ: TestType):
        dir_path = os.path.join(self.base_dir, get_flag_str(typ))
        ensure_dir(dir_path)
        return dir_path

    def profiler_dir(self, typ: TestType):
        dir_path = os.path.join(self.typed_dir(typ), "profiler")
        ensure_dir(dir_path)
        return dir_path

    def output_filepath(self, typ: TestType):
        return os.path.join(self.typed_dir(typ), f"{self.name}.{get_flag_str(typ)}.txt")

    def generated_filepath(self, typ: TestType):
        ext = get_flag_str(typ)
        if ext == "text":
            ext = "txt"

        return os.path.join(self.typed_dir(typ), f"{self.name}.{ext}")

    def run_test(self, gsa: str):
        def run(pargs: list[str], typ: TestType):
            o = run_process(pargs, self.name, profiler_dir=self.profiler_dir(typ))
            with open(self.output_filepath(typ), "w") as f:
                f.write(o)

        if TestType.TEXT_TEST in self.type:
            run([gsa, "-f", "text", "--verbose", self.path], TestType.TEXT_TEST)

        if TestType.JSON_TEST in self.type:
            run([gsa,
                 "-f", "json",
                 "--indent", "2",
                 self.path,
                 "-o", self.generated_filepath(TestType.JSON_TEST)],
                TestType.JSON_TEST)

        if TestType.HTML_TEST in self.type:
            run([gsa,
                 "-f", "html",
                 self.path,
                 "-o", self.generated_filepath(TestType.HTML_TEST)],
                TestType.HTML_TEST)

        if TestType.SVG_TEST in self.type:
            run([gsa,
                 "-f", "svg",
                 self.path,
                 "-o", self.generated_filepath(TestType.SVG_TEST)],
                TestType.SVG_TEST)


class RemoteBinaryType(Enum):
    RAW = "raw"
    TAR = "tar"
    ZIP = "zip"


class RemoteBinary:
    def __init__(self, name: str, url: str, test_typ: TestType, typ: RemoteBinaryType, target: str = None):
        self.name = name
        self.url = url
        self.type = typ
        self.test_type = test_typ
        self.target = target

    def to_csv(self) -> [str]:
        return [self.name, self.url, self.test_type.value, self.type.value, self.target]

    @staticmethod
    def from_csv(line: [str]):
        return RemoteBinary(line[0], line[1], TestType(int(line[2])), RemoteBinaryType(line[3]), line[4])

    def ensure_exist(self):
        bin_path = get_bin_path(self.name)
        if os.path.exists(bin_path):
            log(f"{self} already exists.")
            return

        resp = requests.get(self.url, stream=True)
        resp.raise_for_status()

        log(f"Downloading {self}...")

        content = io.BytesIO()
        total = int(resp.headers.get('content-length', 0))
        with tqdm(total=total, unit='B', unit_scale=True, unit_divisor=1024) as bar:
            for data in resp.iter_content(chunk_size=1024):
                content.write(data)
                bar.update(len(data))

        if self.type == RemoteBinaryType.RAW:
            raw = content.getvalue()
        elif self.type == RemoteBinaryType.TAR:
            content.seek(0)
            raw = load_file_from_tar(content, self.target)
        elif self.type == RemoteBinaryType.ZIP:
            raw = load_file_from_zip(content, self.target)
        else:
            raise Exception(f"Unknown binary type {self.type}")

        os.makedirs(os.path.dirname(bin_path), exist_ok=True)
        with open(bin_path, 'wb') as f:
            f.write(raw)

        log(f"Downloaded {self}")

    def __str__(self):
        return f"RemoteBinary({self.name}, {self.url}, {self.type}, {self.target})"

    def to_test(self) -> IntegrationTest:
        self.ensure_exist()
        return IntegrationTest(self.name, get_bin_path(self.name), self.test_type)


def load_remote_binaries() -> list[IntegrationTest]:
    log("Fetching remote binaries...")

    with open(get_binaries_path(), "r") as f:
        reader = csv.reader(f)
        ret = [RemoteBinary.from_csv(line).to_test() for line in reader]

    log("Fetched remote binaries.")
    return ret


def load_remote_for_tui_test():
    (RemoteBinary("bin-linux-1.21-amd64",
                  "https://github.com/Zxilly/go-testdata/releases/download/latest/bin-linux-1.21-amd64",
                  TestType.TEXT_TEST, RemoteBinaryType.RAW)
     .ensure_exist())
