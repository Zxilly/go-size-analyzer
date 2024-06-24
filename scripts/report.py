import base64
import os

import zstd
import requests
from markdown_strings import header, code_block

from tool.utils import get_project_root, write_github_summary, details

# these are the tips that are not considered as errors
tips = [
    "DWARF parsing failed",
    "No symbol table found",
    "Disassembler not supported"
]


def check_line(line: str) -> bool:
    if "level=WARN" not in line and "level=ERROR" not in line:
        return False

    for tip in tips:
        if tip in line:
            return False

    return True


def need_report(f: str) -> bool:
    with open(f, "r") as f:
        for line in f.readlines():
            if check_line(line):
                return True
    return False


def filter_output(f: str) -> str:
    ret = []
    with open(f, "r") as f:
        lines = f.readlines()
        for line in lines:
            if check_line(line):
                ret.append(line)

    # truncate the output if it's more than 50 lines
    if len(ret) > 50:
        ret = ret[:50]
        ret.append("truncated output...")

    return "".join(ret)


def generate_image_url(p: str) -> str:
    with open(p, "rb") as f:
        data = f.read()

    # compress the image
    data = zstd.compress(data, 22)

    # encode base64
    data = base64.b64encode(data).decode("utf-8")

    req = requests.Request("GET", "https://bin2image.zxilly.dev", params={
        "type": "zstd",
        "data": data
    })

    return req.prepare().url


is_ci = os.getenv("CI", False)

if __name__ == '__main__':
    results = os.path.join(get_project_root(), "results")

    if not os.path.exists(results):
        raise FileNotFoundError(f"Directory {results} does not exist")

    for root, dirs, files in os.walk(results):
        for file in files:
            if file.endswith(".output.txt"):
                output_file_path = str(os.path.join(root, file))
                if need_report(output_file_path):
                    write_github_summary(header(f"Found bad case in {output_file_path}", header_level=3)+'\n')
                    write_github_summary(details(code_block(filter_output(output_file_path)))+'\n')
                    break

            # camo limit the url length
            # if file.endswith(".graph.svg"):
            #     image_url = generate_image_url(str(os.path.join(root, file)))
            #     write_github_summary(image(file, image_url) + '\n')
