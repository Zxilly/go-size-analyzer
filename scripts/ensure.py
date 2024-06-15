import os
from argparse import ArgumentParser

from tool.remote import load_remote_binaries

if __name__ == "__main__":
    ap = ArgumentParser()
    ap.add_argument("--example", action="store_true", help="Download example binaries.")
    ap.add_argument("--real", action="store_true", help="Download real binaries.")

    args = ap.parse_args()

    os.environ["FORCE_REFRESH"] = "true"

    cond = ""
    if args.example:
        cond = "example"
    elif args.real:
        cond = "real"

    load_remote_binaries(cond)