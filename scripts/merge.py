from utils import *


def merge_covdata():
    log("Merging coverage data...")

    unit_path = get_covdata_unit_dir()
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

    integration_path = get_covdata_integration_dir()
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
