import os

import requests

from .utils import log

TARGET_TAG = "latest"
BIN_REPO = "Zxilly/go-testdata"
versions = ["1.16", "1.18", "1.21"]

full_versions = [f"1.{i}" for i in range(11, 22)]

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


def get_example_download_url(filename: str) -> None | str:
    release_info = get_release_info()

    file_info = None
    for asset in release_info['assets']:
        if asset['name'] == filename:
            file_info = asset
            break

    if file_info is None:
        log(f'File {filename} not found.')
        return None

    return file_info['browser_download_url']
