import shutil
import subprocess


def require_svgo() -> str:
    svgo = shutil.which("svgo")
    if svgo is None:
        raise Exception("svgo is required to optimize SVG files.")
    return svgo


def optimize_svg(d: str) -> str:
    # svgo -i - -o -
    return subprocess.check_output(
        args=[
            require_svgo(),
            "-i",
            "-",
            "-o",
            "-",
            "--multipass",
        ],
        input=d,
        text=True,
    )
