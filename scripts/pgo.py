import os
import subprocess

from gsa import build_pgo_gsa
from utils import get_project_root
from tests import run_integration_tests

def merge_profiles():
    # walk result dirs
    # merge profiles

    profiles = []
    for root, dirs, files in os.walk(os.path.join(get_project_root(), "results")):
        for file in files:
            if file == "cpu.pprof":
                profiles.append(os.path.join(root, file))

    profile = subprocess.check_output(
        args=[
            "go",
            "tool",
            "pprof",
            "-proto",
            *profiles,
        ],
        cwd=get_project_root(),
    )

    with open(os.path.join(get_project_root(), "default.pgo"), "wb") as f:
        f.write(profile)

if __name__ == '__main__':
    with build_pgo_gsa() as gsa:
       run_integration_tests("real")
    merge_profiles()

