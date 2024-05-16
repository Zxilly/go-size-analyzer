import os.path
import subprocess

from utils import log, get_project_root, dir_is_empty

def merge_covdata():
    log("Merging coverage data...")

    unit_path = os.path.join(get_project_root(), "covdata", "unit")
    if not dir_is_empty(unit_path):
        subprocess.run(
            [
                "go",
                "tool",
                "covdata",
                "textfmt",
                f"-i={unit_path}",
                "-o",
                "unit.profile",
            ],
            check=True,
            cwd=get_project_root(),
            encoding="utf-8",
        )
        log("Merged unit coverage data.")

    integration_path = os.path.join(get_project_root(), "covdata", "integration")
    if not dir_is_empty(integration_path):
        subprocess.run(
            [
                "go",
                "tool",
                "covdata",
                "textfmt",
                f"-i={integration_path}",
                "-o",
                "integration.profile",
            ],
            check=True,
            cwd=get_project_root(),
            encoding="utf-8",
        )
        log("Merged integration coverage data.")

    log("Merged coverage data.")
