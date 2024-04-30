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


class IntegrationTest:
    def __init__(self, name: str, path: str, typ: TestType):
        self.name = name
        self.path = path
        self.type = typ


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
