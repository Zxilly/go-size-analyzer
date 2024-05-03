import csv

from define import RemoteBinary, RemoteBinaryType, TestType
from example_download import get_example_download_url
from utils import get_binaries_path


def generate_cockroachdb() -> list[RemoteBinary]:
    urls = [
        ("https://binaries.cockroachdb.com/cockroach-v24.1.0-beta.2.linux-amd64.tgz", "linux-amd64"),
        ("https://binaries.cockroachdb.com/cockroach-v24.1.0-beta.2.linux-arm64.tgz", "linux-arm64"),
        ("https://binaries.cockroachdb.com/cockroach-v24.1.0-beta.2.darwin-10.9-amd64.tgz", "darwin-amd64"),
        ("https://binaries.cockroachdb.com/cockroach-v24.1.0-beta.2.darwin-11.0-arm64.tgz", "darwin-arm64"),
        ("https://binaries.cockroachdb.com/cockroach-v24.1.0-beta.2.windows-6.2-amd64.zip", "windows-amd64"),
    ]

    ret = []
    for url in urls:
        file_name = f"cockroach-{url[1]}"
        is_windows = url[1].startswith("windows")
        ret.append(
            RemoteBinary(file_name,
                         url[0],
                         TestType.JSON_TEST | TestType.SVG_TEST,
                         RemoteBinaryType.ZIP if is_windows else RemoteBinaryType.TAR,
                         "cockroach.exe" if is_windows else "cockroach")
        )
    return ret


def generate_example() -> list[RemoteBinary]:
    ret = []
    for v in ["1.19", "1.20", "1.21", "1.22"]:
        for o in ["linux", "windows", "darwin"]:
            for pie in ["-pie", ""]:
                for cgo in ["-cgo", ""]:
                    name = f"bin-{o}-{v}-amd64{pie}{cgo}"
                    url = get_example_download_url(name)

                    if url is None:
                        print(f"File {name} not found.")
                        continue

                    ret.append(
                        RemoteBinary(
                            name,
                            get_example_download_url(name),
                            TestType.TEXT_TEST | TestType.JSON_TEST | TestType.HTML_TEST,
                            RemoteBinaryType.RAW
                        )
                    )
    for o in ["linux", "windows", "darwin"]:
        for pie in ["-pie", ""]:
            for cgo in ["-cgo", ""]:
                name = f"bin-{o}-1.22-amd64-strip{pie}{cgo}"
                url = get_example_download_url(name)

                if url is None:
                    print(f"File {name} not found.")
                    continue

                ret.append(
                    RemoteBinary(
                        name,
                        get_example_download_url(name),
                        TestType.TEXT_TEST,
                        RemoteBinaryType.RAW
                    )
                )

    return ret


if __name__ == '__main__':
    tests = []
    tests.extend(generate_example())
    tests.extend(generate_cockroachdb())

    with open(get_binaries_path(), "w", newline="") as f:
        writer = csv.writer(f)
        for test in tests:
            writer.writerow(test.to_csv())
