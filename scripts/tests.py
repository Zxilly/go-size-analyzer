import os
import shutil
import tempfile
from download import ensure_exist


def require_go() -> str:
    go = shutil.which("go")
    if go is None:
        raise Exception("Go is not installed. Please install Go and try again.")
    return go


def get_temp_binary_name() -> str:
    suffix = ".exe" if os.name == "nt" else ""

    return tempfile.mktemp(suffix=suffix)


def get_project_root() -> str:
    return os.path.abspath(os.path.join(os.path.dirname(__file__), os.pardir))


def build_gsa():
    go = require_go()
    temp_binary = get_temp_binary_name()
    project_root = get_project_root()

    ret = os.system(f"{go} build -tags embed -o {temp_binary} {project_root}/cmd/gsa")
    if ret != 0:
        raise Exception("Failed to build gsa.")

    return temp_binary


if __name__ == "__main__":
    binary = build_gsa()

    tests = []

    for v in ["1.18", "1.19", "1.20", "1.21"]:
        for o in ["linux", "windows", "darwin"]:
            for pie in ["-pie", ""]:
                for cgo in ["-cgo", ""]:
                    name = f"bin-{o}-{v}-amd64{pie}{cgo}"
                    p = ensure_exist(name)
                    tests.append((name, p))

    for t in tests:
        # check if exit code is 0
        ret = os.system(f"{binary} -f text {t[1]}")
        if ret != 0:
            print(f"Test {t[0]} failed.")
            exit(1)
        else:
            print(f"Test {t[0]} passed.")

    os.remove(binary)
    print("All tests passed.")
