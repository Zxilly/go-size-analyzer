import csv

import requests

from example import get_example_download_url
from remote import RemoteBinary, RemoteBinaryType, TestType, Target
from utils import get_binaries_path


def add_exe(name: str, is_windows: bool) -> str:
    if is_windows:
        return f"{name}.exe"
    return name


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
                         [
                             Target(add_exe("cockroach", is_windows), file_name)
                         ])
        )

    return ret


def generate_kubernetes() -> list[RemoteBinary]:
    ret = []

    for o in ["windows", "linux", "darwin"]:
        for a in ["amd64", "arm64", "386"]:
            if o == "darwin" and a == "386":
                continue

            name = f"kubectl-{o}-{a}"
            url = f"https://dl.k8s.io/v1.30.1/bin/{o}/{a}/kubectl"
            if o == "windows":
                url += ".exe"
            ret.append(
                RemoteBinary(
                    name,
                    url,
                    TestType.JSON_TEST,
                    RemoteBinaryType.RAW,
                    [
                        Target(None, name)
                    ]
                )
            )

    for n in ["kube-proxy", "kube-apiserver"]:
        for a in ["amd64", "arm64"]:
            name = f"{n}-{a}"
            url = f"https://dl.k8s.io/v1.30.1/bin/linux/{a}/{n}"
            ret.append(
                RemoteBinary(
                    name,
                    url,
                    TestType.JSON_TEST,
                    RemoteBinaryType.RAW,
                    [
                        Target(None, name)
                    ]
                )
            )

    return ret


def generate_prometheus() -> list[RemoteBinary]:
    ret = []

    for o in ["windows", "linux", "darwin"]:
        for a in ["amd64", "arm64", "386"]:
            if o == "darwin" and a == "386":
                continue

            targets = [
                Target(add_exe("prometheus", o == "windows"), f"prometheus-{o}-{a}"),
                Target(add_exe("promtool", o == "windows"), f"promtool-{o}-{a}")
            ]

            ret.append(
                RemoteBinary(
                    f"prometheus-{o}-{a}",
                    f"https://github.com/prometheus/prometheus/releases/download/v2.52.0/prometheus-2.52.0.{o}-{a}.tar.gz",
                    TestType.JSON_TEST,
                    RemoteBinaryType.TAR,
                    targets)
            )

    return ret


def generate_vitess() -> list[RemoteBinary]:
    targets = [
        Target("vtctl", "vtctl"),
        Target("vtgate", "vtgate"),
        Target("vttablet", "vttablet"),
        Target("vtcombo", "vtcombo"),
        Target("vtgate", "vtgate"),
        Target("vtorc", "vtorc"),
    ]
    return [
        RemoteBinary(
            "vitess",
            "https://github.com/vitessio/vitess/releases/download/v17.0.7/vitess-17.0.7-7c0245d.tar.gz",
            TestType.JSON_TEST,
            RemoteBinaryType.TAR,
            targets)
    ]


def generate_example() -> list[RemoteBinary]:
    ret = []
    for v in ["1.16", "1.19", "1.22"]:
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
                                continue

                            ret.append(
                                RemoteBinary(
                                    name,
                                    get_example_download_url(name),
                                    TestType.TEXT_TEST | TestType.JSON_TEST | TestType.HTML_TEST | TestType.SVG_TEST,
                                    RemoteBinaryType.RAW,
                                    [
                                        Target(name, name)
                                    ]
                                )
                            )

    return ret


if __name__ == '__main__':
    remotes = []
    remotes.extend(generate_example())
    remotes.extend(generate_cockroachdb())
    remotes.extend(generate_kubernetes())
    remotes.extend(generate_prometheus())
    remotes.extend(generate_vitess())

    for r in remotes:
        print(f"Checking {r.name}...")
        resp = requests.get(r.url, stream=True)
        resp.raise_for_status()
        resp.close()

    with open(get_binaries_path(), "w", newline="") as f:
        writer = csv.writer(f)
        for remote in remotes:
            writer.writerow(remote.to_csv())
