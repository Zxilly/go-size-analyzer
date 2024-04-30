import concurrent.futures
import csv
import json

from define import IntegrationTest, TestType, RemoteBinary
from gsa import build_gsa
from parser import DataParser
from utils import *


def eval_test(gsa: str, target: IntegrationTest):
    name = target.name
    path = target.path
    test_type = target.type

    if TestType.TEXT_TEST in test_type:
        run_process([gsa, "-f", "text", path], name, ".txt")

    if TestType.JSON_TEST in test_type:
        run_process([gsa, "-f", "json", path, "-o", get_result_file(f"{name}.json")], name, ".json.txt")

    if TestType.HTML_TEST in test_type:
        run_process([gsa, "-f", "html", path, "-o", get_result_file(f"{name}.html")], name, ".html.txt")

    if TestType.SVG_TEST in test_type:
        run_process([gsa, "-f", "svg", path, "-o", get_result_file(f"{name}.svg")], name, ".svg.txt")


def run_unit_tests():
    log("Running unit tests...")
    unit_path = os.path.join(get_project_root(), "covdata", "unit")

    run_process(
        [
            "go",
            "test",
            "-v",
            "-race",
            "-covermode=atomic",
            "-cover",
            "-tags",
            "embed",
            "./...",
            f"-test.gocoverdir={unit_path}",
        ],
        "embed-unit",
        ".txt",
    )

    run_process(
        [
            "go",
            "test",
            "-v",
            "-race",
            "-covermode=atomic",
            "-cover",
            "./...",
            f"-test.gocoverdir={unit_path}",
        ],
        "unit",
        ".txt",
    )

    log("Unit tests passed.")


def merge_covdata():
    log("Merging coverage data...")

    subprocess.run(
        [
            "go",
            "tool",
            "covdata",
            "textfmt",
            "-i=./covdata/unit,./covdata/integration",
            "-o",
            "coverage.profile",
        ],
        check=True,
        cwd=get_project_root(),
        encoding="utf-8",
    )

    log("Merged coverage data.")


def run_integration_tests(targets: list[IntegrationTest]):
    with build_gsa() as gsa:
        run_web_test(gsa)

        all_tests = len(targets)
        completed_tests = 0
        with concurrent.futures.ThreadPoolExecutor() as executor:
            futures = {executor.submit(task, gsa, t): t for i, t in enumerate(targets)}
            for future in concurrent.futures.as_completed(futures):
                test = futures[future]
                try:
                    future.result()  # This will raise an exception if the test failed
                    completed_tests += 1
                    log(f"[{completed_tests}/{all_tests}] Test {test.name} passed.")
                except Exception as e:
                    log(f"[{completed_tests}/{all_tests}] Test {test.name} failed: {e}")
                    exit(1)


def task(entry, test):
    try:
        eval_test(entry, test)
    except Exception as e:
        raise RuntimeError(f"Test {test.name} failed: {e}")


def run_web_test(entry: str):
    log("Running web test...")

    env = os.environ.copy()
    env["GOCOVERDIR"] = get_covdata_integration_dir()

    p = subprocess.Popen(
        args=[entry, "--web", "--listen", "localhost:23371", entry],
        text=True, cwd=get_project_root(),
        encoding="utf-8", env=env, stdout=subprocess.PIPE, stderr=subprocess.PIPE
    )

    for line in iter(p.stdout.readline, ""):
        if "localhost" in line:
            break

    ret = requests.get("http://localhost:23371").text

    # parse html
    parser = DataParser()
    parser.feed(ret)

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

    p.terminate()
    log("Web test passed.")


def load_remote_binaries() -> list[IntegrationTest]:
    with open(get_binaries_path(), "r") as f:
        reader = csv.reader(f)
        return [RemoteBinary.from_csv(line).to_test() for line in reader]


if __name__ == "__main__":
    set_base_time()

    init_dirs()

    run_unit_tests()

    log("Fetching remote binaries...")

    tests = load_remote_binaries()

    run_integration_tests(tests)

    merge_covdata()

    log("All tests passed.")
