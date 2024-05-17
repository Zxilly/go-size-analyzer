import os.path
import time
from argparse import ArgumentParser

import requests

from gsa import build_gsa
from merge import merge_covdata
from remote import load_remote_binaries, load_remote_for_tui_test
from utils import *


def run_unit_tests():
    log("Running unit tests...")
    load_remote_for_tui_test()

    unit_path = os.path.join(get_project_root(), "covdata", "unit")

    unit_output_dir = os.path.join(get_project_root(), "results", "unit")
    ensure_dir(unit_output_dir)

    embed_out = run_process(
        [
            "go",
            "test",
            "-v",
            "-race",
            "-covermode=atomic",
            "-cover",
            "-tags=embed",
            "./...",
            f"-test.gocoverdir={unit_path}"
        ],
        "unit_embed",
        timeout=600,  # Windows runner is extremely slow
    )

    # test no tag
    normal_out = run_process(
        [
            "go",
            "test",
            "-v",
            "-race",
            "-covermode=atomic",
            "-cover",
            "./internal/webui",
            f"-test.gocoverdir={unit_path}",
        ],
        "unit",
        timeout=600,  # Windows runner is extremely slow
    )

    with open(os.path.join(unit_output_dir, "unit_embed.txt"), "w") as f:
        f.write(embed_out)
    with open(os.path.join(unit_output_dir, "unit.txt"), "w") as f:
        f.write(normal_out)

    log("Unit tests passed.")


def run_integration_tests():
    log("Running integration tests...")

    targets = load_remote_binaries()

    with build_gsa() as gsa:
        run_web_test(gsa)

        all_tests = len(targets)
        completed_tests = 1

        for target in targets:
            try:
                base = time.time()
                target.run_test(gsa)
                log(f"[{completed_tests}/{all_tests}] Test {target.name} passed in {format_time(time.time() - base)}.")
                completed_tests += 1
            except Exception as e:
                log(f"Test {target.name} failed: {e}")
                raise e

    log("Integration tests passed.")


def run_web_test(entry: str):
    log("Running web test...")

    env = os.environ.copy()
    env["GOCOVERDIR"] = get_covdata_integration_dir()
    env["OUTPUT_DIR"] = os.path.join(get_project_root(), "results", "web", "profiler")
    ensure_dir(env["OUTPUT_DIR"])

    port = find_unused_port()
    if port is None:
        raise Exception("Failed to find an unused port.")

    stdout_data, stderr_data = "", ""
    p = subprocess.Popen(
        args=[entry, "--web", "--listen", f"0.0.0.0:{port}", entry],
        text=True, cwd=get_project_root(),
        encoding="utf-8", env=env, stdout=subprocess.PIPE, stderr=subprocess.PIPE
    )

    for line in iter(p.stdout.readline, ""):
        if "localhost" in line:
            break
        stdout_data += line

    time.sleep(1)

    if p.poll() is not None:
        stdout_data += p.stdout.read()
        stderr_data = p.stderr.read()

        print(f"stdout: {stdout_data}\n")
        print(f"stderr: {stderr_data}\n")

        raise Exception("Failed to start the server.")

    ret = requests.get(f"http://127.0.0.1:{port}").text

    assert_html_valid(ret)

    p.terminate()
    log("Web test passed.")


def get_parser() -> ArgumentParser:
    ap = ArgumentParser()

    ap.add_argument("--unit", action="store_true", help="Run unit tests.")
    ap.add_argument("--integration", action="store_true", help="Run integration tests.")

    return ap


if __name__ == "__main__":
    parser = get_parser()
    args = parser.parse_args()

    init_dirs()

    if not args.unit and not args.integration:
        if os.getenv("CI") is None:
            args.unit = True
            args.integration = True
        else:
            raise Exception("Please specify a test type to run.")

    if args.unit:
        run_unit_tests()
    if args.integration:
        run_integration_tests()

    merge_covdata()

    log("All tests passed.")
