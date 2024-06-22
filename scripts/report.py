import os

from tool.utils import get_project_root

# these are the tips that are not considered as errors
tips = [
    "DWARF parsing failed",
    "No symbol table found",
    "Disassembler not supported"
]


def check_line(s: str) -> bool:
    if not "WARN" in s and not "ERROR" in s:
        return False

    for tip in tips:
        if tip in s:
            return False

    return True


def need_report(f: str) -> bool:
    with open(f, "r") as f:
        for line in f.readlines():
            if check_line(line):
                return True
    return False


def filter_output(f: str) -> str:
    with open(f, "r") as f:
        lines = f.readlines()
        return "\n".join([line for line in lines if check_line(line)])


if __name__ == '__main__':
    results = os.path.join(get_project_root(), "results")

    if not os.path.exists(results):
        raise FileNotFoundError(f"Directory {results} does not exist")

    for root, dirs, files in os.walk(results):
        for file in files:
            if file.endswith(".output.txt"):
                output_file_path = str(os.path.join(root, file))
                if need_report(output_file_path):
                    print(f"Found bad case in {output_file_path}:\n\n")
                    print(filter_output(output_file_path))
                    break
