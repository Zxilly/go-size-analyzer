import json
from argparse import ArgumentParser
from html.parser import HTMLParser

import requests
from time import sleep

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


def run_web_test(entry: str):
    print("Running web test...")

    env = os.environ.copy()
    env["GOCOVERDIR"] = get_covdata_integration_dir()

    p = subprocess.Popen(
        args=[entry, "--web", "--listen", "localhost:23371", "--hide-progress", entry],
        text=True, cwd=get_project_root(),
        encoding="utf-8", env=env
    )

    # wait 3 seconds for the server to start
    sleep(3)

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
    print("Web test passed.")


if __name__ == "__main__":
    ap = ArgumentParser()
    ap.add_argument("--cockroachdb", action="store_true", default=False)

    args = ap.parse_args()

    init_dirs()

    gsa = build_gsa()

    # run_unit_tests()
    run_web_test(gsa)

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
