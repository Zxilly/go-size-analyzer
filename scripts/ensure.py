import os
from argparse import ArgumentParser

from tool.remote import load_remote_binaries_as_test

if __name__ == "__main__":
    ap = ArgumentParser()
    ap.add_argument("--example", action="store_true", help="Download example binaries.")
    ap.add_argument("--real", action="store_true", help="Download real binaries.")

    args = ap.parse_args()

    os.environ["FORCE_REFRESH"] = "true"


    def cond(name: str) -> bool:
        if args.example:
            return name.startswith("bin-")
        elif args.real:
            return not name.startswith("bin-")
        return True

    load_remote_binaries_as_test(cond)
