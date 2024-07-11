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

        tmp_out = tempfile.NamedTemporaryFile(delete=False)
        if not dir_is_empty(d):
            subprocess.check_call(
                [
                    require_go(),
                    "tool",
                    "covdata",
                    "textfmt",
                    "-i=" + d,
                    "-o=" + tmp_out.name,
                ],
                cwd=get_project_root(),
            )

            log(f"Merged coverage data from {d}.")

            enhance_coverage(tmp_out.name, output)

            log(f"Enhanced coverage data from {d}.")

            os.remove(tmp_out.name)

            log(f"Merge cleaned up for {d}.")
        else:
            log(f"Coverage data directory is empty. Skipping merge {output}")

    merge_covdata_dir(get_covdata_unit_dir(), "unit.profile")
    merge_covdata_dir(get_covdata_integration_dir(), "integration.profile")

    log("Merged coverage data.")
