import subprocess

from .utils import dir_is_empty, get_project_root, log, get_covdata_unit_dir, get_covdata_integration_dir


def merge_covdata():
    log("Merging coverage data...")

    def merge_covdata_dir(d: str, output: str):
        if not dir_is_empty(d):
            subprocess.check_call(
                [
                    "go",
                    "tool",
                    "covdata",
                    "textfmt",
                    "-i=" + d,
                    "-o=" + output,
                ],
                cwd=get_project_root(),
            )
            log(f"Merged coverage data from {d}.")

    merge_covdata_dir(get_covdata_unit_dir(), "unit.profile")
    merge_covdata_dir(get_covdata_integration_dir(), "integration.profile")

    log("Merged coverage data.")
