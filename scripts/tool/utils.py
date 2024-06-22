import json
import os
import shutil
import socket
import subprocess
import tempfile
import time
from html.parser import HTMLParser
from io import BytesIO
from threading import Thread

import matplotlib
import matplotlib.pyplot as plt
import psutil

matplotlib.use('agg')


def write_github_summary(s: str, fallback: bool = True):
    if os.getenv("CI") is not None:
        with open(os.getenv("GITHUB_STEP_SUMMARY"), "a", encoding="utf-8") as f:
            f.write(f"{s}\n")
    else:
        if fallback:
            print(s)


def get_new_temp_binary() -> str:
    suffix = ".exe" if os.name == "nt" else ""

    parent_dir = os.path.join(get_project_root(), "temp")
    os.makedirs(parent_dir, exist_ok=True)
    temp_dir = tempfile.mkdtemp(prefix="gsa_", dir=parent_dir)

    bin_hash = os.urandom(4).hex()

    return os.path.join(temp_dir, f"gsa-{bin_hash}{suffix}")


def get_project_root() -> str:
    return os.path.abspath(
        os.path.join(os.path.dirname(__file__),
                     os.pardir,
                     os.pardir))


def ensure_dir(path: str) -> str:
    os.makedirs(path, exist_ok=True)
    return path


def get_covdata_integration_dir():
    return os.path.join(get_project_root(), "covdata", "integration")


def get_covdata_unit_dir():
    return os.path.join(get_project_root(), "covdata", "unit")


def init_dirs():
    paths: list[str] = [
        get_covdata_integration_dir(),
        get_covdata_unit_dir(),
    ]

    for p in paths:
        ensure_dir(p)
        for f in os.listdir(p):
            os.remove(os.path.join(p, f))

    results = os.path.join(get_project_root(), "results")
    clear_folder(results)
    ensure_dir(results)


def clear_folder(folder_path: str) -> None:
    if not os.path.exists(folder_path):
        return

    for filename in os.listdir(folder_path):
        file_path = os.path.join(folder_path, filename)

        if os.path.isfile(file_path):
            os.remove(file_path)
        elif os.path.isdir(file_path):
            shutil.rmtree(file_path)


def get_bin_path(name: str) -> str:
    return os.path.join(get_project_root(), "scripts", "bins", name)


def require_go() -> str:
    go = shutil.which("go")
    if go is None:
        raise Exception("Go is not installed. Please install Go and try again.")
    return go


base_time = time.time()


def log(msg: str):
    global base_time
    t = format_time(time.time() - base_time)
    print(f"[{t}] {msg}", flush=True)


def format_time(t: float) -> str:
    return "{:.2f}s".format(t)


def find_unused_port(start_port=20000, end_port=60000):
    for port in range(start_port, end_port + 1):
        try:
            with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
                s.bind(("localhost", port))
                return port
        except OSError:
            pass
    return None


def load_skip() -> list[str]:
    with open(os.path.join(get_project_root(), "scripts", "skip.csv"), "r", encoding="utf-8") as f:
        return [line.strip() for line in f.readlines()]


process_timeout = 360 if os.name == "nt" else 180


def run_process(pargs: list[str], name: str, timeout=240, profiler_dir: str = None, draw: bool = False) -> [str, bytes]:
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
            elapsed_time = time.time() - start_time
            if elapsed_time > timeout:
                raise TimeoutError(f"Process {name} timed out after {timeout} seconds.")

            if draw:
                percent = ps_process.cpu_percent()
                mem = ps_process.memory_info().rss / (1024 * 1024)

                # ensure all data is valid
                cpu_percentages.append(percent)
                memory_usage_mb.append(mem)
                timestamps.append(elapsed_time)

            time.sleep(0.05)
    except TimeoutError as e:
        print(f"TimeoutError occurred: {e}")
        process.kill()
    except psutil.NoSuchProcess:
        pass

    buf = BytesIO()

    if draw and timestamps[-1] >= 2:
        fig, ax1 = plt.subplots(figsize=(14, 5))

        color = 'tab:blue'
        ax1.set_xlabel('Time (seconds)')
        ax1.set_ylabel('CPU %', color=color)
        ax1.plot(timestamps, cpu_percentages, color=color)
        ax1.tick_params(axis='y', labelcolor=color)

        ax1.set_xticks(range(0, int(max(timestamps)) + 1, 1))
        ax1.set_xlim(0, int(max(timestamps)))

        ax2 = ax1.twinx()

        color = 'tab:purple'
        ax2.set_ylabel('Memory (MB)', color=color)
        ax2.plot(timestamps, memory_usage_mb, color=color)
        ax2.tick_params(axis='y', labelcolor=color)

        plt.title('CPU and Memory Usage')

        plt.savefig(buf, format='svg')
        buf.seek(0)

        plt.clf()
        plt.close()

    return [output, buf.getvalue()]


def get_binaries_path():
    return os.path.join(get_project_root(), "scripts", "binaries.csv")


class DataParser(HTMLParser):
    def __init__(self):
        super().__init__()
        self.in_data = False
        self.data = None

    def handle_starttag(self, tag, attrs):
        if tag == "script":
            for attr in attrs:
                if attr[0] == "type" and attr[1] == "application/json":
                    self.in_data = True

    def handle_data(self, data):
        if self.in_data:
            self.data = data

    def handle_endtag(self, tag):
        if self.in_data:
            self.in_data = False

    def get_data(self):
        return self.data


def assert_html_valid(h: str):
    # parse html
    parser = DataParser()
    parser.feed(h)

    json_data = parser.get_data()
    if json_data is None:
        raise Exception("Failed to find data element in the html.")

    # try load value as json
    try:
        content = json.loads(json_data)
    except json.JSONDecodeError:
        raise Exception("Failed to parse data element as json.")

    # check if the data is correct
    keys = ["name", "size", "packages", "sections"]
    for key in keys:
        if key not in content:
            raise Exception(f"Missing key {key} in the data.")


def dir_is_empty(p: str) -> bool:
    return len(os.listdir(p)) == 0
