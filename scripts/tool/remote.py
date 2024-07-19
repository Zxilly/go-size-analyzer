import csv
import io
import os.path
import tarfile
import zipfile
from enum import Flag, Enum, auto
from threading import Thread
from typing import Callable
from urllib.parse import urlparse

import requests
from tqdm import tqdm

from .process import run_process
from .utils import get_project_root, ensure_dir, get_bin_path, log, get_binaries_path


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

    raise Exception(f"Unknown test type {typ}")


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
        return os.path.join(self.typed_dir(typ), f"{self.name}.{get_flag_str(typ)}.output.txt")

    def performance_figure_filepath(self, typ: TestType):
        return os.path.join(self.typed_dir(typ), f"{self.name}.{get_flag_str(typ)}.graph.svg")

    def generated_filepath(self, typ: TestType):
        ext = get_flag_str(typ)
        if ext == "text":
            ext = "txt"

        return os.path.join(self.typed_dir(typ), f"{self.name}.{ext}")

    def run_test(self, gsa: str, log_typ: callable(TestType), timeout=240):
        threads = []

        draw = not self.name.startswith("bin-")

        def run(pargs: list[str], typ: TestType):
            [output_data, graph_data] = run_process(
                pargs,
                self.name,
                profiler_dir=self.profiler_dir(typ),
                timeout=timeout,
                draw=draw,
            )
            with open(self.output_filepath(typ), "w", encoding="utf-8") as f:
                f.write(output_data)

            if draw and graph_data is not None:
                with open(self.performance_figure_filepath(typ), "wb") as f:
                    f.write(graph_data.encode("utf-8"))

            log_typ(typ)

        if TestType.TEXT_TEST in self.type:
            threads.append(Thread(target=run, args=([gsa, "-f", "text", "--verbose", self.path], TestType.TEXT_TEST)))

        if TestType.JSON_TEST in self.type:
            threads.append(
                Thread(target=run,
                       args=([gsa,
                              "-f", "json",
                              "--verbose",
                              "--indent", "2",
                              self.path,
                              "-o", self.generated_filepath(TestType.JSON_TEST)],
                             TestType.JSON_TEST)))

        if TestType.HTML_TEST in self.type:
            threads.append(
                Thread(target=run,
                       args=([gsa,
                              "-f", "html",
                              self.path,
                              "-o", self.generated_filepath(TestType.HTML_TEST)],
                             TestType.HTML_TEST)))

        if TestType.SVG_TEST in self.type:
            threads.append(
                Thread(target=run,
                       args=([gsa,
                              "-f", "svg",
                              self.path,
                              "-o", self.generated_filepath(TestType.SVG_TEST)],
                             TestType.SVG_TEST)))

        for t in threads:
            t.start()
        for t in threads:
            t.join()


class RemoteBinaryType(Enum):
    RAW = "raw"
    TAR = "tar"
    ZIP = "zip"


class Target:
    name: str
    path: str
    data: bytes | None

    def __init__(self, name: str | None, path: str):
        """
        :param name: the name of the file in the archive, for raw it is None
        :param path: the path name to save the file
        """
        self.name = name
        self.path = get_bin_path(path)
        self.data = None

    def __str__(self):
        return f"{self.name}:{os.path.basename(self.path)}"

    @staticmethod
    def from_str(s: str):
        name, path = s.split(":")
        return Target(name, path)


class RemoteBinary:
    def __init__(self, name: str, url: str, test_typ: TestType, typ: RemoteBinaryType, targets: list[Target]):
        self.name = name
        self.url = url
        self.type = typ
        self.test_type = test_typ
        self.targets = targets

    def to_csv(self) -> [str]:
        return [self.name, self.url, self.test_type.value, self.type.value, "@".join([str(t) for t in self.targets])]

    @staticmethod
    def from_csv(line: [str]):
        ret = RemoteBinary(line[0],
                           line[1],
                           TestType(int(line[2])),
                           RemoteBinaryType(line[3]),
                           [Target.from_str(t) for t in line[4].split("@")])
        return ret

    def ensure_exist(self):
        ok = True
        for target in self.targets:
            if not os.path.exists(target.path):
                ok = False
                break
        if ok and os.getenv("FORCE_REFRESH") is None:
            log(f"{self} already exists.")
            return

        header = dict()

        host = urlparse(self.url).hostname

        if host == "github.com":
            token = os.getenv('GITHUB_TOKEN')
            if token:
                header['Authorization'] = f'Bearer {token}'

        resp = requests.get(self.url, stream=True, headers=header)
        resp.raise_for_status()

        log(f"Downloading {self}...")

        content = io.BytesIO()
        total = int(resp.headers.get('content-length', 0))
        with tqdm(total=total, unit='B', unit_scale=True, unit_divisor=1024) as bar:
            for data in resp.iter_content(chunk_size=1024):
                content.write(data)
                bar.update(len(data))

        content.seek(0)

        if self.type == RemoteBinaryType.RAW:
            self.targets[0].data = content.getvalue()
        elif self.type == RemoteBinaryType.TAR:
            self.targets = load_file_from_tar(content, self.targets)
        elif self.type == RemoteBinaryType.ZIP:
            self.targets = load_file_from_zip(content, self.targets)
        else:
            raise Exception(f"Unknown binary type {self.type}")

        for target in self.targets:
            d = os.path.dirname(target.path)
            ensure_dir(d)
            with open(target.path, "wb") as f:
                f.write(target.data)

        log(f"Downloaded {self}")

    def __str__(self):
        return f"RemoteBinary({self.name}, {self.url}, {self.type})"

    def to_test(self) -> list[IntegrationTest]:
        self.ensure_exist()
        ret = []

        def get_name(target_name: str):
            if len(self.targets) == 1:
                return self.name
            return self.name + "-" + target_name

        for target in self.targets:
            ret.append(IntegrationTest(get_name(target.name), get_bin_path(target.path), self.test_type))
        return ret


def load_remote_binaries_as_test(cond: Callable[[str], bool]) -> list[IntegrationTest]:
    log("Fetching remote binaries...")

    with open(get_binaries_path(), "r", encoding="utf-8") as f:
        reader = csv.reader(f)
        ret = [RemoteBinary.from_csv(line) for line in reader]

    def filter_binary(t: RemoteBinary):
        return cond(t.name)

    filtered = list(filter(filter_binary, ret))
    tests = []
    for binary in filtered:
        tests.extend(binary.to_test())

    log("Fetched remote binaries.")
    return tests


def load_remote_for_unit_test():
    (RemoteBinary("bin-linux-1.21-amd64",
                  "https://github.com/Zxilly/go-testdata/releases/download/latest/bin-linux-1.21-amd64",
                  TestType.TEXT_TEST, RemoteBinaryType.RAW, [Target("bin-linux-1.21-amd64", "bin-linux-1.21-amd64")])
     .ensure_exist())
    (RemoteBinary("bin-linux-1.22-amd64",
                  "https://github.com/Zxilly/go-testdata/releases/download/latest/bin-linux-1.21-amd64-cgo",
                  TestType.TEXT_TEST, RemoteBinaryType.RAW,
                  [Target("bin-linux-1.22-amd64", "bin-linux-1.22-amd64")])
     .ensure_exist())


def load_file_from_tar(f: io.BytesIO, targets: list[Target]) -> list[Target]:
    with tarfile.open(fileobj=f) as tar:
        for member in tar.getmembers():
            real_name = os.path.basename(member.name)
            for target in targets:
                if real_name == target.name:
                    f = tar.extractfile(member)
                    if f is None:
                        continue
                    target.data = f.read()

    for target in targets:
        if target.data is None:
            raise Exception(f"File {target.name} not found in tar.")

    return targets


def load_file_from_zip(f: io.BytesIO, targets: list[Target]) -> list[Target]:
    with zipfile.ZipFile(f) as z:
        for name in z.namelist():
            real_name = os.path.basename(name)
            for target in targets:
                if real_name == target.name:
                    target.data = z.read(name)

    for target in targets:
        if target.data is None:
            raise Exception(f"File {target.name} not found in zip.")

    return targets
