import argparse
import os

import requests

TARGET_TAG = "latest"
BIN_REPO = "Zxilly/go-testdata"
versions = ["1.16", "1.18", "1.21"]

full_versions = [f"1.{i}" for i in range(11, 22)]


def get_bin_path(filename: str):
    return os.path.join(os.path.dirname(__file__), "bins", filename)


release_info_cache = None


def get_release_info():
    global release_info_cache
    if release_info_cache is None:
        # read GitHub token if possible
        token = os.getenv('GITHUB_TOKEN')
        headers = {}
        if token:
            headers['Authorization'] = f'Bearer {token}'

        response = requests.get(f'https://api.github.com/repos/{BIN_REPO}/releases/tags/{TARGET_TAG}', headers=headers)
        response.raise_for_status()
        release_info_cache = response.json()
    return release_info_cache


def download(filename: str):
    release_info = get_release_info()

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

    bin_path = get_bin_path(filename)

    os.makedirs(os.path.dirname(bin_path), exist_ok=True)

    with open(bin_path, 'wb') as f:
        f.write(response.content)

    print(f"Downloaded {filename}")


def ensure_exist(filename: str):
    p = get_bin_path(filename)
    e = os.path.exists(p)
    if not e:
        download(filename)
    else:
        print(f"{filename} exists.")
    return p


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('-a', '--arch', choices=['amd64'], nargs="+", default=['amd64'])
    parser.add_argument('-o', '--os', choices=['linux', 'windows', 'darwin'], nargs="+", default=['linux'])
    parser.add_argument('-v', '--version', choices=full_versions, nargs="+", default=["1.21"])
    parser.add_argument("-c", '--cgo', action='store_true', default=False, help="Download CGO version")
    parser.add_argument('-p', '--pie', action='store_true', default=False, help="Download PIE version")
    parser.add_argument('-s', '--strip', action='store_true', default=False, help="Download stripped version")

    args = parser.parse_args()
    for arch in args.arch:
        for pos in args.os:
            for version in args.version:
                name = f"bin-{pos}-{version}-{arch}" + ("-strip" if args.strip else "") + ("-cgo" if args.cgo else "") + ("-pie" if args.pie else "")
                ensure_exist(name)
