import os
import shutil
import subprocess

import tool.gsa
from tests import run_integration_tests
from tool.gsa import build_pgo_gsa
from tool.utils import get_project_root, require_go


def merge_profiles():
    # walk result dirs
    # merge profiles
    profiles = []
    for d in sorted(os.listdir(os.path.join(get_project_root(), "results"))):
        d = os.path.join(get_project_root(), "results", d)
        if not os.path.isdir(d):
            continue

        profiler_dirs = [
            os.path.join(d, "profiler"),
            os.path.join(d, "json", "profiler"),
        ]
        existing_profiler_dirs = [pd for pd in profiler_dirs if os.path.exists(pd)]
        if not existing_profiler_dirs:
            print(f"Skipping {d}, no profiler dir")
            continue

        for pd in existing_profiler_dirs:
            p = os.path.join(pd, "cpu.pprof")
            if not os.path.exists(p):
                print(f"Skipping {p}", os.listdir(pd))
                continue
            profiles.append(p)

    profile = subprocess.check_output(
        args=[
            require_go(),
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
        run_integration_tests("real", gsa)
    merge_profiles()
