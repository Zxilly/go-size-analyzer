import argparse
import os

import requests

TARGET_TAG = "20240117143504"
BIN_REPO = "Zxilly/go-testdata"
versions = ["1.13", "1.17", "1.21"]

full_versions = [f"1.{i}" for i in range(13, 22)]


def get_bin_path(filename: str):
    return os.path.join(os.getcwd(), "bins", filename)


def download(filename: str):
    response = requests.get(f'https://api.github.com/repos/{BIN_REPO}/releases/tags/{TARGET_TAG}')
    response.raise_for_status()
    release_info = response.json()

    # 查找指定的文件
    file_info = None
    for asset in release_info['assets']:
        if asset['name'] == filename:
            file_info = asset
            break

    if file_info is None:
        print(f'File {filename} not found.')
        return

    response = requests.get(file_info['browser_download_url'])
    response.raise_for_status()

    with open(get_bin_path(filename), 'wb') as f:
        f.write(response.content)

    print(f"Downloaded {filename}")


def ensure_exist(filename: str):
    p = get_bin_path(filename)
    e = os.path.exists(p)
    if not e:
        download(filename)
    else:
        print(f"{filename} exists.")


parser = argparse.ArgumentParser()
parser.add_argument('-a', '--arch', choices=['amd64', 'arm64', '386'], nargs="+", default=['amd64'])
parser.add_argument('-o', '--os', choices=['linux', 'windows', 'darwin'], nargs="+", default=['linux'])
parser.add_argument('-v', '--version', default=["1.21"], choices=full_versions)
parser.add_argument('-s', '--strip', action='store_true', default=False)

if __name__ == '__main__':
    args = parser.parse_args()
    for arch in args.arch:
        for pos in args.os:
            for version in args.version:
                name = f"bin-{pos}-{version}-{arch}" + ("_strip" if args.strip else "")
                ensure_exist(name)