import os.path
import platform
import subprocess
import time
from argparse import ArgumentParser

import requests

from tool.gsa import build_gsa
from tool.junit import generate_junit
from tool.merge import merge_covdata
from tool.remote import load_remote_binaries, load_remote_for_tui_test, TestType, get_flag_str
from tool.utils import log, get_project_root, ensure_dir, format_time, load_skip, get_covdata_integration_dir, \
    find_unused_port, assert_html_valid, init_dirs


def run_unit_tests(full: bool, wasm: bool, no_embed: bool):
    log("Running unit tests...")
    load_remote_for_tui_test()

    unit_path = os.path.join(get_project_root(), "covdata", "unit")

    unit_output_dir = os.path.join(get_project_root(), "results", "unit")
    ensure_dir(unit_output_dir)

    if full:
        try:
            log("Running full unit tests...")
            embed_result = subprocess.run(
                [
                    "go",
                    "test",
                    "-v",
                    "-covermode=atomic",
                    "-cover",
                    "-tags=embed",
                    "./...",
                    f"-test.gocoverdir={unit_path}"
                ],
                text=True,
                cwd=get_project_root(),
                stderr=subprocess.STDOUT,
                stdout=subprocess.PIPE,
                timeout=600
            )
            embed_result.check_returncode()

            with open(os.path.join(unit_output_dir, "unit_embed.txt"), "w", encoding="utf-8") as f:
                f.write(embed_result.stdout)

            generate_junit(embed_result.stdout, os.path.join(get_project_root(), "unit_embed.xml"))

            log("Full unit tests passed.")
        except subprocess.CalledProcessError as e:
            log("Error running embed unit tests:")
            log(f"stdout: {e.stdout}")
            exit(1)

    if no_embed:
        try:
            log("Running normal unit tests for webui...")
            normal_result = subprocess.run(
                [
                    "go",
                    "test",
                    "-v",
                    "-covermode=atomic",
                    "-cover",
                    "./internal/webui",
                    f"-test.gocoverdir={unit_path}"
                ],
                text=True,
                cwd=get_project_root(),
                stderr=subprocess.STDOUT,
                stdout=subprocess.PIPE,
                timeout=600
            )
            normal_result.check_returncode()

            with open(os.path.join(unit_output_dir, "unit.txt"), "w", encoding="utf-8") as f:
                f.write(normal_result.stdout)

            generate_junit(normal_result.stdout, os.path.join(get_project_root(), "unit.xml"))

            log("Normal network unit tests passed.")
        except subprocess.CalledProcessError as e:
            log("Error running normal unit tests:")
            log(f"stdout: {e.stdout}")
            exit(1)

    if wasm:
        try:
            log("Running WebAssembly unit tests...")

            env = os.environ.copy()
            for raw in list(env.keys()):
                k = raw.upper()
                if (k.startswith("GITHUB_")
                        or k.startswith("JAVA_")
                        or k.startswith("PSMODULEPATH")
                        or k.startswith("PYTHONPATH")
                        or k.startswith("STATS_")
                        or k.startswith("RUNNER_")
                        or k.startswith("LIBRARY_")
                        or k == "_OLD_VIRTUAL_PATH"
                ):
                    del env[raw]

                if platform.system() == "Windows":
                    if k == "PATH":
                        parts = env[raw].split(";")
                        new_parts = []
                        for i, part in enumerate(parts):
                            lower = part.lower()
                            if ("go" in lower
                                    or "pip" in lower
                                    or "python" in lower
                                    or "node" in lower):
                                new_parts.append(part)
                        env[raw] = ";".join(new_parts)

            env_size = 0
            for k, v in env.items():
                env_size += len(k) + len(v) + 1
            if env_size > 4000:  # windows use utf-16
                log("Environment size is too large")
                for k, v in env.items():
                    print(f"{k}={v}")
                exit(1)

            env["GOOS"] = "js"
            env["GOARCH"] = "wasm"

            wasm_result = subprocess.run(
                [
                    "go",
                    "test",
                    "-v",
                    "-covermode=atomic",
                    "-cover",
                    "./internal/result",
                    f"-test.gocoverdir={unit_path}"
                ],
                text=True,
                cwd=get_project_root(),
                stderr=subprocess.STDOUT,
                stdout=subprocess.PIPE,
                timeout=600,
                env=env
            )
            wasm_result.check_returncode()

            with open(os.path.join(unit_output_dir, "unit_wasm.txt"), "w", encoding="utf-8") as f:
                f.write(wasm_result.stdout)

            generate_junit(wasm_result.stdout, os.path.join(get_project_root(), "unit_wasm.xml"))
        except subprocess.CalledProcessError as e:
            log("Error running wasm unit tests:")
            log(f"stdout: {e.stdout}")
            exit(1)

    log("Unit tests passed.")


global_failed = 0


def run_integration_tests(typ: str, gsa_path: str):
    scope_failed_count = 0

    log(f"Running integration tests {typ}...")

    targets = load_remote_binaries(typ)

    if typ == "example":
        timeout = 10
    else:
        timeout = 60

    if typ == "example":
        run_web_test(gsa_path)

    all_tests = len(targets)
    completed_tests = 0

    skips = load_skip()

    for target in targets:
        head = f"[{completed_tests + 1}/{all_tests}] Test {os.path.basename(target.path)}"
        log(f"{head} start")

        if target.path in skips:
            log(f"{head} is skipped.")
            continue

        try:
            base = time.time()

            def report_typ(rtyp: TestType):
                log(f"{head} {get_flag_str(rtyp)} passed in {format_time(time.time() - base)}.")

            target.run_test(gsa_path, report_typ, timeout=timeout)
            log(f"{head} passed in {format_time(time.time() - base)}.")
        except Exception as e:
            log(f"{head} failed")

            if os.getenv("CI") is not None:
                with open(os.getenv("GITHUB_STEP_SUMMARY"), "a", encoding="utf-8") as f:
                    f.write(f"```log\n{str(e)}\n```\n")
            else:
                print(e)

            scope_failed_count += 1

        completed_tests += 1

    if scope_failed_count == 0:
        log("Integration tests passed.")
    else:
        log(f"{scope_failed_count} integration tests failed.")
        global global_failed
        global_failed += scope_failed_count


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
        args=[entry, "--web", "--listen", f"127.0.0.1:{port}", entry],
        text=True, cwd=get_project_root(),
        encoding="utf-8", env=env, stdout=subprocess.PIPE, stderr=subprocess.PIPE,
    )

    log(f"Waiting for the server to start on port {port}...")

    for line in iter(p.stdout.readline, ""):
        if "localhost" in line:
            break
        stdout_data += line

    time.sleep(1)

    if p.poll() is not None:
        stdout_data += p.stdout.read()
        stderr_data = p.stderr.read()

        log(f"stdout:\n {stdout_data}\n")
        log(f"stderr:\n {stderr_data}\n")

        raise Exception("Failed to start the server.")

    ret = requests.get(f"http://127.0.0.1:{port}").text

    assert_html_valid(ret)

    p.terminate()
    p.wait()
    log("Web test passed.")


def get_parser() -> ArgumentParser:
    ap = ArgumentParser()

    ap.add_argument("--unit-full", action="store_true", help="Run full unit tests.")
    ap.add_argument("--unit-wasm", action="store_true", help="Run unit tests for wasm.")
    ap.add_argument("--unit-embed", action="store_true", help="Run unit tests for embed")
    ap.add_argument("--unit", action="store_true", help="Run unit tests.")

    ap.add_argument("--integration-example", action="store_true", help="Run integration tests for small binaries.")
    ap.add_argument("--integration-real", action="store_true", help="Run integration tests for large binaries.")
    ap.add_argument("--integration", action="store_true", help="Run all integration tests.")

    return ap


if __name__ == "__main__":
    parser = get_parser()
    args = parser.parse_args()

    init_dirs()

    if args.integration:
        args.integration_example = True
        args.integration_real = True

    if args.unit:
        args.unit_full = True
        args.unit_wasm = True
        args.unit_embed = True

    if (not args.unit and not args.unit_full and not args.unit_wasm and not args.unit_embed
            and not args.integration_example and not args.integration_real):
        if os.getenv("CI") is None:
            args.unit_full = True
            args.unit_wasm = True
            args.unit_embed = True
            args.integration_example = True
            args.integration_real = True
        else:
            raise Exception("Please specify a test type to run.")

    run_unit_tests(args.unit_full, args.unit_wasm, args.unit_embed)

    with build_gsa() as gsa:
        if args.integration_example:
            run_integration_tests("example", gsa)
        if args.integration_real:
            run_integration_tests("real", gsa)

    merge_covdata()

    if global_failed == 0:
        log("All tests passed.")
    else:
        log(f"{global_failed} tests failed.")
        exit(1)
