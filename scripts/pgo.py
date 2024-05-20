import os
import shutil
import subprocess

from gsa import build_pgo_gsa
from tests import run_integration_tests
from utils import get_project_root


def merge_profiles():
    # walk result dirs
    # merge profiles
    profiles = []
    for d in os.listdir(os.path.join(get_project_root(), "results")):
        d = os.path.join(get_project_root(), "results", d)
        if not os.path.isdir(d):
            continue

        p = os.path.join(d, "json", "profiler", "cpu.pprof")
        if not os.path.exists(p):
            print(f"Skipping {p}")
            continue
        profiles.append(p)

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
    shutil.rmtree(os.path.join(get_project_root(), "results"), ignore_errors=True)

    with build_pgo_gsa() as gsa:
        run_integration_tests("real")
    merge_profiles()
