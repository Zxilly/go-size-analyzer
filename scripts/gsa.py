import contextlib

from utils import *


@contextlib.contextmanager
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

    yield temp_binary

    print("Cleaning up...")
    os.remove(temp_binary)
    print("Cleaned up.")


