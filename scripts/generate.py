import csv
import itertools
from concurrent.futures import ThreadPoolExecutor

import requests

from tool.example import get_example_download_url
from tool.remote import RemoteBinary, RemoteBinaryType, TestType, Target
from tool.utils import get_binaries_source_path


def add_exe(name: str, is_windows: bool) -> str:
    if is_windows:
        return f"{name}.exe"
    return name


def generate_cockroachdb() -> list[RemoteBinary]:
    urls = [
        ("https://binaries.cockroachdb.com/cockroach-v25.3.1.linux-amd64.tgz", "linux-amd64"),
        ("https://binaries.cockroachdb.com/cockroach-v25.3.1.linux-arm64.tgz", "linux-arm64"),
        ("https://binaries.cockroachdb.com/cockroach-v25.3.1.darwin-10.9-amd64.tgz", "darwin-amd64"),
        ("https://binaries.cockroachdb.com/cockroach-v25.3.1.darwin-11.0-arm64.tgz", "darwin-arm64"),
        ("https://binaries.cockroachdb.com/cockroach-v25.3.1.windows-6.2-amd64.zip", "windows-amd64"),
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

    # kubectl 部分
    kubectl_oses = ["windows", "linux", "darwin"]
    kubectl_archs = ["amd64", "arm64", "386"]

    for o, a in itertools.product(kubectl_oses, kubectl_archs):
        if o == "darwin" and a == "386":
            continue

        name = f"kubectl-{o}-{a}"
        url = f"https://dl.k8s.io/v1.34.0/bin/{o}/{a}/kubectl"
        url += ".exe" if o == "windows" else ""

        ret.append(RemoteBinary(
            name,
            url,
            TestType.JSON_TEST,
            RemoteBinaryType.RAW,
            [Target(None, name)]
        ))

    # kube-proxy kube-apiserver
    kube_components = ["kube-proxy", "kube-apiserver"]
    kube_archs = ["amd64", "arm64"]

    for n, a in itertools.product(kube_components, kube_archs):
        name = f"{n}-{a}"
        url = f"https://dl.k8s.io/v1.34.0/bin/linux/{a}/{n}"

        ret.append(RemoteBinary(
            name,
            url,
            TestType.JSON_TEST,
            RemoteBinaryType.RAW,
            [Target(None, name)]
        ))

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
                    f"https://github.com/prometheus/prometheus/releases/download/v3.5.0/prometheus-3.5.0.{o}-{a}.tar.gz",
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
            "https://github.com/vitessio/vitess/releases/download/v22.0.1/vitess-22.0.1-aafd403.tar.gz",
            TestType.JSON_TEST,
            RemoteBinaryType.TAR,
            targets)
    ]


def generate_example() -> list[RemoteBinary]:
    versions = ["1.24", "1.25", "1.26"]
    oses = ["linux", "windows", "darwin"]
    pies = ["-pie", ""]
    cgos = ["-cgo", ""]
    archs = ["amd64", "arm64", "386"]
    strips = ["-strip", "-stripdwarf", ""]

    ret = []

    for v, o, pie, cgo, a, s in itertools.product(versions, oses, pies, cgos, archs, strips):
        if (pie == "-pie" and cgo == "") or \
                (o == "darwin" and a == "386") or \
                (o == "windows" and a == "arm64") or \
                (o == "darwin" and pie == ""):
            continue

        name = f"bin-{o}-{v}-{a}{s}{pie}{cgo}"
        url = get_example_download_url(name)

        if url is None:
            continue

        ret.append(
            RemoteBinary(
                name,
                url,
                TestType.TEXT_TEST | TestType.JSON_TEST | TestType.HTML_TEST | TestType.SVG_TEST,
                RemoteBinaryType.RAW,
            )
        )

    return ret


def generate_big_const() -> list[RemoteBinary]:
    # https://github.com/Zxilly/go-testdata/releases/download/const/const-linux
    # https://github.com/Zxilly/go-testdata/releases/download/const/const-macos
    # https://github.com/Zxilly/go-testdata/releases/download/const/const-windows

    ret = []
    for o in ["windows", "linux", "macos"]:
        name = f"const-{o}"
        url = f"https://github.com/Zxilly/go-testdata/releases/download/const/{name}"
        ret.append(
            RemoteBinary(
                name,
                url,
                TestType.JSON_TEST,
                RemoteBinaryType.RAW,
            )
        )
    return ret


if __name__ == '__main__':
    remotes = []
    remotes.extend(generate_example())
    remotes.extend(generate_big_const())
    remotes.extend(generate_cockroachdb())
    remotes.extend(generate_kubernetes())
    remotes.extend(generate_prometheus())
    remotes.extend(generate_vitess())

    pool = ThreadPoolExecutor(max_workers=16)


    def check_remote(tr: RemoteBinary):
        print(f"Checking {tr.name}...", flush=True)
        resp = requests.head(tr.url)
        resp.raise_for_status()
        resp.close()


    for r in remotes:
        pool.submit(check_remote, r)

    pool.shutdown(wait=True)

    with open(get_binaries_source_path(), "w", newline="", encoding="utf-8") as f:
        writer = csv.writer(f)
        for remote in remotes:
            writer.writerow(remote.to_csv())
