import contextlib
import shutil
import subprocess

from utils import log, require_go, get_new_temp_binary, get_project_root, extract_output, get_result_file


@contextlib.contextmanager
def build_gsa():
    log("Building gsa...")

    go = require_go()
    temp_binary = get_new_temp_binary()
    project_root = get_project_root()

    ret = subprocess.run(
        [
            go,
            "build",
            "-buildmode=exe",  # since windows use pie by default
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

    log("Built gsa.")

    yield temp_binary

    shutil.move(temp_binary, get_result_file("integration", ".test"))
