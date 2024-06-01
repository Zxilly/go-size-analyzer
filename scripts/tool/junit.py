import shutil
import subprocess


def require_go_junit_report() -> str:
    gjr = shutil.which("go-junit-report")
    if gjr is None:
        raise Exception("go-junit-report is required to generate JUnit reports.")
    return gjr


def generate_junit(stdout: str, target: str):
    gjr = require_go_junit_report()
    with open(target, "w") as f:
        subprocess.run(
            [gjr],
            input=stdout,
            stdout=f,
            text=True,
            check=True,
        )


