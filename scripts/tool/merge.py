import os
import subprocess

from .utils import get_project_root, log, get_covdata_integration_dir, require_go


def merge_covdata():
    log("Merging coverage data...")

    def merge_covdata_dir(d: str, output: str):
        if os.path.exists(output):
            os.remove(output)

        subprocess.check_call(
            [
                require_go(),
                "tool",
                "covdata",
                "textfmt",
                "-i=" + d,
                "-o=" + output,
            ],
            cwd=get_project_root(),
        )
        log(f"Merged coverage data from {d}.")

        if not os.path.exists(output):
            raise Exception("Failed to merge coverage data.")
        else:
            log(f"Saved enhanced coverage data to {output}.")

    def abs_path(s: str):
        return os.path.abspath(os.path.join(get_project_root(), s))

    merge_covdata_dir(get_covdata_integration_dir(), abs_path("integration.profile"))

    log("Merged coverage data.")
