import argparse
import subprocess

from download import versions, get_bin_path, ensure_exist


def print_asm(arch: str, pos: str, target: str = ""):
    for version in versions:
        filename = f"bin-{pos}-{version}-{arch}"
        ensure_exist(filename)
        cmd_args = ["go", "tool", "objdump"]
        if target:
            cmd_args.extend(["-s", target])
        cmd_args.append(get_bin_path(filename))
        ret = subprocess.run(cmd_args, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        if ret.returncode != 0:
            print(f"## {filename} failed")
            print(ret.stderr.decode("utf-8"))
            continue
        print(f"## {filename}")
        print(ret.stdout.decode("utf-8"))


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('-a', '--arch', choices=['amd64', 'arm64', '386'], nargs="+", default=['amd64'])
    parser.add_argument('-o', '--os', choices=['linux', 'windows', 'darwin'], nargs="+", default=['linux'])
    parser.add_argument('-t', '--target', default="", required=False)

    args = parser.parse_args()
    for a in args.arch:
        for p in args.os:
            print_asm(a, p, args.target)
