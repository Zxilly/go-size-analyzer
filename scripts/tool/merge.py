import os
import shutil
import subprocess
import tempfile

from .utils import dir_is_empty, get_project_root, log, get_covdata_unit_dir, get_covdata_integration_dir, require_go


def require_courtney():
    courtney = shutil.which("courtney")
    if courtney is None:
        raise Exception("Courtney is not installed. Please install Courtney and try again.")
    return courtney


def enhance_coverage(f: str, out: str):
    subprocess.check_call(
        [
            require_courtney(),
            "-l",
            f,
            "-o",
            out,
        ]
    )


def merge_covdata():
    log("Merging coverage data...")

    def merge_covdata_dir(d: str, output: str):
        if os.path.exists(output):
            os.remove(output)

        if dir_is_empty(d):
            log(f"Coverage data directory is empty. Skipping merge {output}")
            return

        with tempfile.NamedTemporaryFile(delete=False) as tmp:
            subprocess.check_call(
                [
                    require_go(),
                    "tool",
                    "covdata",
                    "textfmt",
                    "-i=" + d,
                    "-o=" + tmp.name,
                ],
                cwd=get_project_root(),
            )
            log(f"Merged coverage data from {d}.")
            enhance_coverage(tmp.name, output)
            log(f"Enhanced coverage data from {d}.")

        if not os.path.exists(output):
            raise Exception("Failed to merge coverage data.")

    def abs_path(s: str):
        return os.path.abspath(s)

    merge_covdata_dir(get_covdata_unit_dir(), abs_path("unit.profile"))
    merge_covdata_dir(get_covdata_integration_dir(), abs_path("integration.profile"))

    log("Merged coverage data.")
