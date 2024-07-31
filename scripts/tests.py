import os.path
import platform
import subprocess
import time
from argparse import ArgumentParser

import requests
from markdown_strings import code_block

from tool.gsa import build_gsa, GSAInstance
from tool.html import assert_html_valid
from tool.junit import generate_junit
from tool.merge import merge_covdata
from tool.remote import load_remote_binaries_as_test, load_remote_for_unit_test, TestType, get_flag_str
from tool.utils import log, get_project_root, ensure_dir, format_time, load_skip, init_dirs, write_github_summary, \
    require_go

unit_path = os.path.join(get_project_root(), "covdata", "unit")
unit_output_dir = os.path.join(get_project_root(), "results", "unit")
ensure_dir(unit_path)
ensure_dir(unit_output_dir)


def run_unit(name: str, env: dict[str, str], pargs: list[str], timeout: int):
    log(f"Running unit test {name}...")
    start = time.time()
    stdout = subprocess.run(
        args=pargs,
        text=True,
        env=env,
        cwd=get_project_root(),
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        timeout=timeout
    )
    stdout.check_returncode()

    with open(os.path.join(unit_output_dir, f"{name}.txt"), "w", encoding="utf-8") as f:
        f.write(stdout.stdout)

    generate_junit(stdout.stdout, os.path.join(get_project_root(), f"{name}.xml"))

    log(f"Unit test {name} passed in {format_time(time.time() - start)}.")


def build_wasm_env():
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
                or k.startswith("ANDROID_")
                or k.startswith("DOTNET")
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

    env["GOOS"] = "js"
    env["GOARCH"] = "wasm"
    return env


def run_unit_tests(full: bool, wasm: bool, no_embed: bool):
    if not full and not wasm and not no_embed:
        return

    log("Running unit tests...")
    load_remote_for_unit_test()

    ensure_dir(unit_output_dir)

    go = require_go()

    if full:
        run_unit("unit_embed", os.environ.copy(),
                 [go,
                  "test",
                  "-v",
                  "-covermode=atomic",
                  "-cover",
                  "-tags=embed",
                  "./...",
                  "-args",
                  f"-test.gocoverdir={unit_path}"],
                 600)

    if no_embed:
        run_unit("unit", os.environ.copy(),
                 [go,
                  "test",
                  "-v",
                  "-covermode=atomic",
                  "-cover",
                  "./internal/webui",
                  "-args",
                  f"-test.gocoverdir={unit_path}"],
                 600)

    if wasm:
        run_unit("unit_wasm", build_wasm_env(),
                 [go,
                  "test",
                  "-v",
                  "-covermode=atomic",
                  "-cover",
                  "-coverpkg=../../...",
                  f"-test.gocoverdir={unit_path}"],
                 600)

    log("Unit tests passed.")


global_failed = 0


def run_integration_tests(typ: str, entry: GSAInstance):
    scope_failed_count = 0

    log(f"Running integration tests {typ}...")

    if typ == "example":
        targets = load_remote_binaries_as_test(lambda x: x.startswith("bin-"))
    elif typ == "real":
        targets = load_remote_binaries_as_test(lambda x: not x.startswith("bin-"))
    else:
        raise Exception(f"Unknown integration test type: {typ}")

    if typ == "example":
        timeout = 10
    else:
        timeout = 60

    all_tests = len(targets)
    completed_tests = 0

    skips = load_skip()

    for target in targets:
        head = f"[{completed_tests + 1}/{all_tests}] Test {os.path.basename(target.path)}"
        log(f"{head} start")

        if target.name in skips:
            log(f"{head} is skipped.")
            continue

        try:
            base = time.time()

            def report_typ(rtyp: TestType):
                log(f"{head} {get_flag_str(rtyp)} passed in {format_time(time.time() - base)}.")

            target.run_test(entry, report_typ, timeout=timeout)
            log(f"{head} passed in {format_time(time.time() - base)}.")
        except Exception as e:
            log(f"{head} failed")

            write_github_summary(code_block(str(e)))

            scope_failed_count += 1

        completed_tests += 1

    if scope_failed_count == 0:
        log(f"Integration tests {typ} passed.")
    else:
        log(f"{scope_failed_count} {typ} integration tests failed.")
        global global_failed
        global_failed += scope_failed_count


def run_version_and_help_test(entry: GSAInstance):
    log("Running flag test...")

    def get_file(n: str):
        d = os.path.join(get_project_root(), "results", n)
        ensure_dir(d)
        return os.path.join(d, f"{n}.output.txt")

    entry.run("--version",
              output=get_file("version"),
              profiler_dir=os.path.join(get_project_root(), "results", "version", "profiler"))
    entry.run("--help",
              output=get_file("help"),
              profiler_dir=os.path.join(get_project_root(), "results", "help", "profiler"))

    log("Flag test passed.")


def run_web_test(entry: GSAInstance):
    log("Running web test...")

    profiler_dir = os.path.join(get_project_root(), "results", "web", "profiler")
    ensure_dir(profiler_dir)

    output_file = os.path.join(get_project_root(), "results", "web", "web.output.txt")

    def check(proc: subprocess.Popen):
        log("Web server started.")

        ret = requests.get(f"http://127.0.0.1:59347").text

        assert_html_valid(ret)

        proc.terminate()
        proc.wait()

    log(f"Waiting for the server to start on port 59347...")
    entry.expect("--verbose", "--web", "--listen", "127.0.0.1:59347", entry.binary,
                 output=output_file, profiler_dir=profiler_dir, expect="localhost",
                 callback=check)

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

    if args.integration_example or args.integration_real:
        with build_gsa() as gsa:
            if args.integration_example:
                run_web_test(gsa)
                run_version_and_help_test(gsa)

                run_integration_tests("example", gsa)
            if args.integration_real:
                run_integration_tests("real", gsa)

    merge_covdata()

    if global_failed == 0:
        log("All tests passed.")
    else:
        log(f"{global_failed} tests failed.")
        exit(1)
