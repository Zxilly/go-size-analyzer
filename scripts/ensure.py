import io
import os

import requests
from tqdm import tqdm

from define import TestType
from download import ensure_exist as ensure_example_bin_exist
from define import IntegrationTest
from utils import load_files_from_tar, load_files_from_zip, get_bin_path, log


def ensure_example_data() -> list[IntegrationTest]:
    log("Getting example data...")
    test = []
    for v in ["1.18", "1.19", "1.20", "1.21"]:
        for o in ["linux", "windows", "darwin"]:
            for pie in ["-pie", ""]:
                for cgo in ["-cgo", ""]:
                    name = f"bin-{o}-{v}-amd64{pie}{cgo}"
                    p = ensure_example_bin_exist(name)
                    test.append(
                        IntegrationTest(name, p, TestType.TEXT_TEST | TestType.JSON_TEST | TestType.HTML_TEST)
                    )
    log("Got example data.")
    return test


def ensure_cockroachdb_data() -> list[IntegrationTest]:
    # from https://www.cockroachlabs.com/docs/releases/
    # https://binaries.cockroachdb.com/cockroach-v24.1.0-beta.2.linux-amd64.tgz
    # https://binaries.cockroachdb.com/cockroach-v24.1.0-beta.2.linux-arm64.tgz
    # https://binaries.cockroachdb.com/cockroach-v24.1.0-beta.2.darwin-10.9-amd64.tgz
    # https://binaries.cockroachdb.com/cockroach-v24.1.0-beta.2.darwin-11.0-arm64.tgz
    # https://binaries.cockroachdb.com/cockroach-v24.1.0-beta.2.windows-6.2-amd64.zip

    def dne(u: str):
        response = requests.get(u, stream=True)
        response.raise_for_status()

        total_size = int(response.headers.get("content-length", 0))

        out = io.BytesIO()

        with tqdm(total=total_size, unit="B", unit_scale=True) as progress_bar:
            for data in response.iter_content(8196):
                progress_bar.update(len(data))
                out.write(data)
        content = out.getvalue()

        if u.endswith(".tgz"):
            ucf = load_files_from_tar(content, "cockroach")
        else:
            ucf = load_files_from_zip(content, "cockroach.exe")
        return ucf

    urls = [
        ("https://binaries.cockroachdb.com/cockroach-v24.1.0-beta.2.linux-amd64.tgz", "linux-amd64"),
        ("https://binaries.cockroachdb.com/cockroach-v24.1.0-beta.2.linux-arm64.tgz", "linux-arm64"),
        ("https://binaries.cockroachdb.com/cockroach-v24.1.0-beta.2.darwin-10.9-amd64.tgz", "darwin-amd64"),
        ("https://binaries.cockroachdb.com/cockroach-v24.1.0-beta.2.darwin-11.0-arm64.tgz", "darwin-arm64"),
        ("https://binaries.cockroachdb.com/cockroach-v24.1.0-beta.2.windows-6.2-amd64.zip", "windows-amd64"),
    ]

    ret = []

    for url in urls:
        file_name = f"cockroach-{url[1]}"
        file_path = get_bin_path(file_name)
        if not os.path.exists(file_path):
            log(f"Downloading {url[0]}...")
            file_byte = dne(url[0])
            with open(file_path, "wb") as f:
                f.write(file_byte)
            log(f"Downloaded {url[0]}.")
        else:
            log(f"File {file_path} already exists.")
        ret.append(IntegrationTest(
            file_name, file_path,
            TestType.JSON_TEST | TestType.SVG_TEST
        ))

    return ret
