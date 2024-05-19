import csv

from example_download import get_example_download_url
from remote import RemoteBinary, RemoteBinaryType, TestType
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
                         TestType.JSON_TEST,
                         RemoteBinaryType.ZIP if is_windows else RemoteBinaryType.TAR,
                         "cockroach.exe" if is_windows else "cockroach")
        )

    return ret


def generate_kubernetes() -> list[RemoteBinary]:
    ret = []

    for o in ["windows", "linux", "darwin"]:
        for a in ["amd64", "arm64", "386"]:
            name = f"kubectl-{o}-{a}"
            url = f"https://dl.k8s.io/release/v1.30.1/bin/{o}/{a}/kubectl"
            if o == "windows":
                url += ".exe"
            ret.append(
                RemoteBinary(
                    name,
                    url,
                    TestType.JSON_TEST,
                    RemoteBinaryType.RAW
                )
            )

    for a in ["amd64", "arm64"]:
        name = f"kube-apiserver-{a}"
        url = f"https://dl.k8s.io/release/v1.30.1/bin/linux/{a}/kube-apiserver"
        ret.append(
            RemoteBinary(
                name,
                url,
                TestType.JSON_TEST,
                RemoteBinaryType.RAW
            )
        )

    return ret


def generate_example() -> list[RemoteBinary]:
    ret = []
    for v in ["1.19", "1.20", "1.21", "1.22"]:
        for o in ["linux", "windows", "darwin"]:
            for pie in ["-pie", ""]:
                for cgo in ["-cgo", ""]:
                    for a in ["amd64", "arm64", "386"]:
                        for s in ["-strip", ""]:
                            if pie == "-pie" and cgo == "":
                                continue

                            if o == "darwin" and a == "386":
                                continue

                            if o == "windows" and a == "arm64":
                                continue

                            name = f"bin-{o}-{v}-{a}{s}{pie}{cgo}"
                            url = get_example_download_url(name)

                            if url is None:
                                print(f"File {name} not found.")
                                continue

                            ret.append(
                                RemoteBinary(
                                    name,
                                    get_example_download_url(name),
                                    TestType.TEXT_TEST | TestType.JSON_TEST | TestType.HTML_TEST | TestType.SVG_TEST,
                                    RemoteBinaryType.RAW
                                )
                            )

    return ret


if __name__ == '__main__':
    remotes = []
    remotes.extend(generate_example())
    remotes.extend(generate_cockroachdb())
    remotes.extend(generate_kubernetes())

    with open(get_binaries_path(), "w", newline="") as f:
        writer = csv.writer(f)
        for remote in remotes:
            writer.writerow(remote.to_csv())
