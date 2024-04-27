from argparse import ArgumentParser

from define import IntegrationTest, TestType
from ensure import ensure_example_data, ensure_cockroachdb_data
from utils import *


def build_gsa():
    print("Building gsa...")

    go = require_go()
    temp_binary = get_new_temp_binary()
    project_root = get_project_root()

    ret = subprocess.run(
        [
            go,
            "build",
            "-cover",
            "-covermode=atomic",
            "-tags",
            "embed",
            "-o",
            temp_binary,
            f"{project_root}/cmd/gsa",
        ],
        text=True,
        capture_output=True,
        cwd=get_project_root(),
        encoding="utf-8",
    )

    if ret.returncode != 0:
        output = extract_output(ret)
        raise Exception(f"Failed to build gsa. Output: {output}")

    print("Built gsa.")

    return temp_binary


def eval_test(gsa: str, target: IntegrationTest):
    name = target.name
    path = target.path
    test_type = target.type

    env = os.environ.copy()
    env["GOCOVERDIR"] = get_covdata_integration_dir()

    def run_gsa(args: list[str], suffix: str):
        ret = subprocess.run(
            args=args,
            env=env, text=True, capture_output=True, cwd=get_project_root(),
            encoding="utf-8",
        )
        output_name = get_result_file(f"{name}{suffix}")
        with open(output_name, "w", encoding="utf-8") as f:
            f.write(extract_output(ret))

        if ret.returncode != 0:
            raise Exception(f"Failed to run gsa on {name}. Check {output_name}.")

    if TestType.TEXT_TEST in test_type:
        run_gsa([gsa, "-f", "text", path], ".txt")

    if TestType.JSON_TEST in test_type:
        run_gsa([gsa, "-f", "json", path, "-o", get_result_file(f"{name}.json"), "--hide-progress"], ".json.txt")

    if TestType.HTML_TEST in test_type:
        run_gsa([gsa, "-f", "html", path, "-o", get_result_file(f"{name}.html"), "--hide-progress"], ".html.txt")


def run_unit_tests():
    print("Running unit tests...")
    unit_path = os.path.join(get_project_root(), "covdata", "unit")

    subprocess.run(
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
        check=True,
        cwd=get_project_root(),
        encoding="utf-8",
    )

    subprocess.run(
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
        check=True,
        cwd=get_project_root(),
        encoding="utf-8",
    )
    print("Unit tests passed.")


def merge_covdata():
    print("Merging coverage data...")

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

    print("Merged coverage data.")


def run_integration_tests(entry: str, targets: list[IntegrationTest]):
    all_tests = len(targets)
    for i, t in enumerate(targets):
        count = int(i) + 1
        cur = str(count) + "/" + str(all_tests)
        try:
            eval_test(entry, t)
            print(f"[{cur}]Test {t.name} passed.")
        except Exception as e:
            print(f"[{cur}]Test {t.name} failed: {e}")
            exit(1)

    os.remove(entry)


if __name__ == "__main__":
    ap = ArgumentParser()
    ap.add_argument("--cockroachdb", action="store_true", default=False)

    args = ap.parse_args()

    init_dirs()

    gsa = build_gsa()

    print("Running tests...")
    run_unit_tests()
    print("Unit tests passed.")

    print("Downloading example data...")
    tests = ensure_example_data()
    print("Downloaded example data.")

    if args.cockroachdb:
        print("Downloading CockroachDB data...")
        tests.extend(ensure_cockroachdb_data())
        print("Downloaded CockroachDB data.")

    run_integration_tests(gsa, tests)

    merge_covdata()

    print("All tests passed.")
