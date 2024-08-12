import contextlib
import os.path
import shutil
import subprocess
import time
from threading import Thread, Timer
from typing import Callable

import psutil

from .matplotlib import draw_usage
from .utils import log, require_go, get_new_temp_binary, get_project_root, get_covdata_integration_dir


def run_go_compiler(*args):
    go = require_go()
    subprocess.check_call([go, *args], cwd=get_project_root())


@contextlib.contextmanager
def build_gsa():
    log("Building gsa...")

    temp_binary = get_new_temp_binary()
    project_root = get_project_root()

    run_go_compiler("build", "-buildmode=exe", "-cover", "-covermode=atomic", "-tags", "embed,profiler", "-o",
                    temp_binary, f"{project_root}/cmd/gsa")

    log("Built gsa.")

    ext = ".exe" if os.name == "nt" else ""
    shutil.copyfile(temp_binary, os.path.join(get_project_root(), "results", "gsa" + ext))

    yield GSAInstance(temp_binary)

    shutil.rmtree(os.path.dirname(temp_binary))


@contextlib.contextmanager
def build_pgo_gsa():
    log("Building gsa...")

    temp_binary = get_new_temp_binary()
    project_root = get_project_root()

    run_go_compiler("build", "-tags", "embed,pgo", "-o", temp_binary, f"{project_root}/cmd/gsa")

    log("Built gsa.")

    yield GSAInstance(temp_binary)

    os.remove(temp_binary)


class GSAInstance:
    def __init__(self, binary: str):
        self.binary = binary

    @staticmethod
    def getenv(profiler_dir: str):
        env = os.environ.copy()
        env["GOCOVERDIR"] = get_covdata_integration_dir()
        env["OUTPUT_DIR"] = profiler_dir
        return env

    def run(self, *args, output: str, profiler_dir: str):
        out = subprocess.check_output(
            args=[self.binary, *args],
            cwd=get_project_root(),
            text=True,
            encoding="utf-8",
            env=self.getenv(profiler_dir),
            stderr=subprocess.STDOUT)

        with open(output, "w", encoding="utf-8") as f:
            f.write(out)

    def expect(self, *args, output: str, profiler_dir: str, expect: str, callback: Callable[[subprocess.Popen], None],
               timeout: int):
        with open(output, "w", encoding="utf-8") as f:
            with subprocess.Popen(
                    args=[self.binary, *args],
                    cwd=get_project_root(),
                    stdout=subprocess.PIPE,
                    stderr=subprocess.STDOUT,
                    text=True,
                    bufsize=1,
                    env=self.getenv(profiler_dir),
                    encoding="utf-8",

            ) as proc:
                def kill():
                    proc.kill()
                    raise TimeoutError(f"Process timed out after {timeout} seconds.")

                Timer(timeout, kill).start()
                for line in iter(proc.stdout.readline, ""):
                    f.write(line)
                    if expect in line:
                        callback(proc)

    def run_with_figure(self, *args, output: str, profiler_dir: str, timeout: int, figure_name: str,
                        figure_output: str = None):
        cpu_percentages = []
        memory_usage_mb = []
        timestamps = []
        start_time = time.time()

        def collect_stdout(p: subprocess.Popen):
            with open(output, "w", encoding="utf-8") as f:
                for line in p.stdout:
                    f.write(line)

        with subprocess.Popen(
                args=[self.binary, *args],
                cwd=get_project_root(),
                stdout=subprocess.PIPE,
                stderr=subprocess.STDOUT,
                text=True,
                env=self.getenv(profiler_dir),
                encoding="utf-8") as proc:
            Thread(target=collect_stdout, args=(proc,)).start()

            try:
                ps_process = psutil.Process(proc.pid)
                while proc.poll() is None:
                    percent = ps_process.cpu_percent(interval=0.1)
                    mem = ps_process.memory_info().rss / (1024 * 1024)
                    elapsed_time = time.time() - start_time
                    if elapsed_time > timeout:
                        raise TimeoutError(f"Process timed out after {timeout} seconds.")

                    if figure_output is not None:
                        cpu_percentages.append(percent)
                        memory_usage_mb.append(mem)
                        timestamps.append(elapsed_time)
            except TimeoutError as e:
                proc.kill()
                raise e
            except psutil.NoSuchProcess:
                pass

        if figure_output is not None and timestamps[-1] > 2:
            pic = draw_usage(figure_name, cpu_percentages, memory_usage_mb, timestamps)
            with open(figure_output, "w", encoding="utf-8") as f:
                f.write(pic)
