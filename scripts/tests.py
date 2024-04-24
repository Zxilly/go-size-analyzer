import os
import shutil
import subprocess
import tempfile
from typing import List, Tuple
from download import ensure_exist


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
            "-tags",
            "embed",
            "-o",
            temp_binary,
            f"{project_root}/cmd/gsa",
        ],
        text=True,
        capture_output=True,
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


def get_coverage_dir() -> str:
    return os.path.join(get_project_root(), "coverage")


def init_dirs():
    paths: List[str] = [
        get_result_dir(),
        get_covdata_integration_dir(),
        get_covdata_unit_dir(),
        get_coverage_dir(),
    ]

    for p in paths:
        ensure_dir(p)
        for f in os.listdir(p):
            os.remove(os.path.join(p, f))


def get_example_data() -> Tuple[str, str]:
    print("Getting example data...")
    test = []
    for v in ["1.18", "1.19", "1.20", "1.21"]:
        for o in ["linux", "windows", "darwin"]:
            for pie in ["-pie", ""]:
                for cgo in ["-cgo", ""]:
                    name = f"bin-{o}-{v}-amd64{pie}{cgo}"
                    p = ensure_exist(name)
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


def eval_test(gsa: str, target: Tuple[str, str]):
    name, path = target

    env = os.environ.copy()
    env["GOCOVERDIR"] = get_covdata_integration_dir()

    ret = subprocess.run(
        [gsa, "-f", "text", path], env=env, text=True, capture_output=True
    )

    with open(get_result_file(f"{name}.txt"), "w") as f:
        f.write(extract_output(ret))

    if ret.returncode != 0:
        raise Exception(f"Failed to run gsa on {name}.")


def run_unit_tests():
    print("Running unit tests...")
    unit_path = os.path.join(get_project_root(), "covdata", "unit")

    subprocess.run(
        [
            "go",
            "test",
            "-v",
            "-race",
            "-covermode=atomic" "-cover",
            "-tags",
            "embed",
            "./...",
            f"-test.gocoverdir={unit_path}",
        ],
        check=True,
    )
    print("Unit tests passed.")


def merge_covdata():
    print("Merging coverage data...")

    subprocess.run(
        [
            "go",
            "tool",
            "covdata",
            "merge",
            "-i=./covdata/unit,./covdata/integration",
            "-o",
            "coverage",
        ],
        check=True,
    )

    print("Merged coverage data.")


if __name__ == "__main__":
    init_dirs()

    gsa = build_gsa()

    tests: List[Tuple[str, str]] = []
    tests.extend(get_example_data())

    count = len(tests)

    for i, t in enumerate(tests):
        try:
            eval_test(gsa, t)
            print(f"[{i+1}/{count}]Test {t[0]} passed.")
        except Exception as e:
            print(f"[{i+1}/{count}]Test {t[0]} failed: {e}")
            exit(1)

    os.remove(gsa)

    run_unit_tests()

    merge_covdata()

    print("All tests passed.")
