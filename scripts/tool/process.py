import os
import subprocess
import time
from threading import Thread

import psutil

from .gsa import DISABLE_DRAW
from .matplotlib import draw_usage
from .utils import get_covdata_integration_dir, get_project_root


def run_process(
        pargs: list[str],
        name: str,
        timeout=240,
        profiler_dir: str = None,
        draw: bool = False) -> [str, bytes | None]:
    if DISABLE_DRAW:
        draw = False

    env = os.environ.copy()
    env["GOCOVERDIR"] = get_covdata_integration_dir()
    if profiler_dir is not None:
        env["OUTPUT_DIR"] = profiler_dir

    process = subprocess.Popen(
        args=pargs,
        env=env, cwd=get_project_root(),
        stdout=subprocess.PIPE, stderr=subprocess.STDOUT,
        text=True,
        encoding="utf-8",
    )

    cpu_percentages = []
    memory_usage_mb = []
    timestamps = []
    output = ""
    start_time = time.time()

    def collect_stdout():
        nonlocal output
        for line in iter(process.stdout.readline, ""):
            output += line

    Thread(target=collect_stdout).start()

    try:
        ps_process = psutil.Process(process.pid)

        while process.poll() is None:
            percent = ps_process.cpu_percent(interval=0.1)
            mem = ps_process.memory_info().rss / (1024 * 1024)
            elapsed_time = time.time() - start_time
            if elapsed_time > timeout:
                raise TimeoutError(f"Process {name} timed out after {timeout} seconds.")

            if draw:
                cpu_percentages.append(percent)
                memory_usage_mb.append(mem)
                timestamps.append(elapsed_time)

    except TimeoutError as e:
        process.kill()
        raise e
    except psutil.NoSuchProcess:
        pass

    pic: None | str = None

    if draw and timestamps[-1] >= 2:
        pic = draw_usage(name, cpu_percentages, memory_usage_mb, timestamps)

    return [output, pic]
