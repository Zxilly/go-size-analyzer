import contextlib
import os.path
import shutil

import sh

from .utils import log, require_go, get_new_temp_binary, get_project_root


def run_go_compiler(*args):
    go = require_go()
    sh.Command(go)(*args, _err_to_out=True, _cwd=get_project_root(), _encoding="utf-8")


@contextlib.contextmanager
def build_gsa():
    log("Building gsa...")

    temp_binary = get_new_temp_binary()
    project_root = get_project_root()

    run_go_compiler("build", "-buildmode=exe", "-cover", "-covermode=atomic", "-tags", "embed,profiler", "-o",
                    temp_binary, f"{project_root}/cmd/gsa")

    log("Built gsa.")

    ext = ".exe" if os.name == "nt" else ""
    shutil.copyfile(temp_binary, os.path.join(get_project_root(), "results", "gsa" + ext))

    yield temp_binary

    shutil.rmtree(os.path.dirname(temp_binary))


@contextlib.contextmanager
def build_pgo_gsa():
    log("Building gsa...")

    temp_binary = get_new_temp_binary()
    project_root = get_project_root()

    run_go_compiler("build", "-tags", "embed,pgo", "-o", temp_binary, f"{project_root}/cmd/gsa")

    log("Built gsa.")

    yield temp_binary

    os.remove(temp_binary)


class GSAInstance:
    def __init__(self, binary: str):
        self.cmd = sh.Command(binary)

    def run(self, *args):
        self.cmd(*args, _err_to_out=True, _cwd=get_project_root(), _encoding="utf-8")
