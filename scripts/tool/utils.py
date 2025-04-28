import os
import shutil
import tempfile
import time


def details(s: str) -> str:
    return f"<details><summary>Details</summary>\n\n{s}\n</details>"


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
    use_cached_bin = os.getenv("TESTDATA_PATH")
    if use_cached_bin is not None:
        return os.path.abspath(use_cached_bin)

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


def load_skip() -> list[str]:
    with open(os.path.join(get_project_root(), "scripts", "skip.csv"), "r", encoding="utf-8") as f:
        return [line.strip() for line in f.readlines()]


process_timeout = 360 if os.name == "nt" else 180


def get_binaries_source_path():
    return os.path.join(get_project_root(), "scripts", "binaries.csv")


def dir_is_empty(p: str) -> bool:
    return len(os.listdir(p)) == 0
