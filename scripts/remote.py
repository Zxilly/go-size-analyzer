import csv

from define import IntegrationTest, RemoteBinary, RemoteBinaryType, TestType
from utils import log, get_binaries_path


def load_remote_binaries() -> list[IntegrationTest]:
    log("Fetching remote binaries...")

    with open(get_binaries_path(), "r") as f:
        reader = csv.reader(f)
        ret = [RemoteBinary.from_csv(line).to_test() for line in reader]

    log("Fetched remote binaries.")
    return ret


def load_remote_for_tui_test():
    (RemoteBinary("bin-linux-1.21-amd64",
                  "https://github.com/Zxilly/go-testdata/releases/download/latest/bin-linux-1.21-amd64",
                  TestType.TEXT_TEST, RemoteBinaryType.RAW)
     .ensure_exist())
