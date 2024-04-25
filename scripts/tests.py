import io
import os
import shutil
import subprocess
import tarfile
import tempfile
import zipfile
from argparse import ArgumentParser
from typing import List

import requests
from tqdm import tqdm

from download import ensure_exist as ensure_example_bin_exist


def require_go() -> str:
    go = shutil.which("go")
    if go is None:
        raise Exception("Go is not installed. Please install Go and try again.")
    return go


def get_new_temp_binary() -> str:
    suffix = ".exe" if os.name == "nt" else ""

    return tempfile.mktemp(suffix=suffix)


def get_project_root() -> str:
    return os.path.abspath(os.path.join(os.path.dirname(__file__), os.pardir))


def build_gsa():
    print("Building gsa...")

    go = require_go()
    temp_binary = get_new_temp_binary()
    project_root = get_project_root()

    ret = subprocess.run(
        [
            go,
            "build",
            "-cover",
            "-covermode=atomic",
            "-tags",
            "embed",
            "-o",
            temp_binary,
            f"{project_root}/cmd/gsa",
        ],
        text=True,
        capture_output=True,
        cwd=get_project_root(),
        encoding="utf-8",
    )

    if ret.returncode != 0:
        output = extract_output(ret)
        raise Exception(f"Failed to build gsa. Output: {output}")

    print("Built gsa.")

    return temp_binary


def ensure_dir(path: str):
    os.makedirs(path, exist_ok=True)
    return path


def get_covdata_integration_dir():
    return os.path.join(get_project_root(), "covdata", "integration")


def get_covdata_unit_dir():
    return os.path.join(get_project_root(), "covdata", "unit")


def get_result_dir() -> str:
    return os.path.join(get_project_root(), "results")


def get_result_file(name: str) -> str:
    return os.path.join(get_result_dir(), name)


def init_dirs():
    paths: List[str] = [
        get_result_dir(),
        get_covdata_integration_dir(),
        get_covdata_unit_dir(),
    ]

    for p in paths:
        ensure_dir(p)
        for f in os.listdir(p):
            os.remove(os.path.join(p, f))


def ensure_example_data() -> list[tuple[str, str]]:
    print("Getting example data...")
    test = []
    for v in ["1.18", "1.19", "1.20", "1.21"]:
        for o in ["linux", "windows", "darwin"]:
            for pie in ["-pie", ""]:
                for cgo in ["-cgo", ""]:
                    name = f"bin-{o}-{v}-amd64{pie}{cgo}"
                    p = ensure_example_bin_exist(name)
                    test.append((name, p))
    print("Got example data.")
    return test


def extract_output(p: subprocess.CompletedProcess) -> str:
    ret = ""

    if len(p.stdout) > 0:
        ret += "stdout:\n"
        ret += p.stdout

    if len(p.stderr) > 0:
        ret += "\nstderr:\n"
        ret += p.stderr

    return ret


def eval_test(gsa: str, name: str, path: str):
    env = os.environ.copy()
    env["GOCOVERDIR"] = get_covdata_integration_dir()

    ret = subprocess.run(
        [gsa, "-f", "text", path],
        env=env, text=True, capture_output=True, cwd=get_project_root(),
        encoding="utf-8",
    )
    output_name = get_result_file(f"{name}.txt")

    with open(output_name, "w", encoding="utf-8") as f:
        f.write(extract_output(ret))

    if ret.returncode != 0:
        raise Exception(f"Failed to run gsa on {name}. Check {output_name}.")

    ret = subprocess.run(
        [gsa, "-f", "json", path, "-o", get_result_file(f"{name}.json"), "--hide-progress"],
        env=env, text=True, capture_output=True, cwd=get_project_root(),
        encoding="utf-8",
    )
    output_name = get_result_file(f"{name}.json.txt")

    with open(output_name, "w", encoding="utf-8") as f:
        f.write(extract_output(ret))

    if ret.returncode != 0:
        raise Exception(f"Failed to run gsa on {name}. Check {output_name}.")


def run_unit_tests():
    print("Running unit tests...")
    unit_path = os.path.join(get_project_root(), "covdata", "unit")

    subprocess.run(
        [
            "go",
            "test",
            "-v",
            "-race",
            "-covermode=atomic",
            "-cover",
            "-tags",
            "embed",
            "./...",
            f"-test.gocoverdir={unit_path}",
        ],
        check=True,
        cwd=get_project_root(),
        encoding="utf-8",
    )

    subprocess.run(
        [
            "go",
            "test",
            "-v",
            "-race",
            "-covermode=atomic",
            "-cover",
            "./...",
            f"-test.gocoverdir={unit_path}",
        ],
        check=True,
        cwd=get_project_root(),
        encoding="utf-8",
    )
    print("Unit tests passed.")


def merge_covdata():
    print("Merging coverage data...")

    subprocess.run(
        [
            "go",
            "tool",
            "covdata",
            "textfmt",
            "-i=./covdata/unit,./covdata/integration",
            "-o",
            "coverage.profile",
        ],
        check=True,
        cwd=get_project_root(),
        encoding="utf-8",
    )

    print("Merged coverage data.")


def run_integration_tests(gsa: str, tests: list[tuple[str, str]]):
    all_tests = len(tests)
    for i, t in enumerate(tests):
        count = int(i) + 1
        cur = str(count) + "/" + str(all_tests)
        try:
            name, path = t
            eval_test(gsa, name, path)
            print(f"[{cur}]Test {t[0]} passed.")
        except Exception as e:
            print(f"[{cur}]Test {t[0]} failed: {e}")
            exit(1)

    os.remove(gsa)


def load_files_from_tar(tar: bytes, target_name: str) -> bytes:
    with io.BytesIO(tar) as f:
        with tarfile.open(fileobj=f) as tar:
            for member in tar.getmembers():
                real_name = os.path.basename(member.name)
                if real_name == target_name:
                    return tar.extractfile(member).read()
    raise Exception(f"File {target_name} not found in tar.")


def load_files_from_zip(zb: bytes, target_name: str) -> bytes:
    with io.BytesIO(zb) as f:
        with zipfile.ZipFile(f) as z:
            for name in z.namelist():
                real_name = os.path.basename(name)
                if real_name == target_name:
                    return z.read(name)
    raise Exception(f"File {target_name} not found in zip.")


def get_bin_path(name: str) -> str:
    return os.path.join(get_project_root(), "scripts", "bins", name)


def ensure_cockroachdb_data() -> list[tuple[str, str]]:
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
            print(f"Downloading {url[0]}...")
            file_byte = dne(url[0])
            with open(file_path, "wb") as f:
                f.write(file_byte)
            print(f"Downloaded {url[0]}.")
        else:
            print(f"File {file_path} already exists.")
        ret.append((file_name, file_path))

    return ret


if __name__ == "__main__":
    ap = ArgumentParser()
    ap.add_argument("--cockroachdb", action="store_true", default=False)

    args = ap.parse_args()

    init_dirs()

    gsa = build_gsa()

    print("Running tests...")
    run_unit_tests()
    print("Unit tests passed.")

    print("Downloading example data...")
    tests = ensure_example_data()
    print("Downloaded example data.")

    if args.cockroachdb:
        print("Downloading CockroachDB data...")
        tests.extend(ensure_cockroachdb_data())
        print("Downloaded CockroachDB data.")

    run_integration_tests(gsa, tests)

    merge_covdata()

    print("All tests passed.")
