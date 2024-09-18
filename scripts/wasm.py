import argparse
import os.path
import shutil
import subprocess
import tempfile

from tool.utils import get_project_root, require_go, log


def wasm_location() -> str:
    return os.path.join(get_project_root(), "ui", "gsa.wasm")


def require_binaryen():
    o = shutil.which("wasm-opt")
    if o is None:
        print("wasm-opt not found in PATH. Please install binaryen.")
        exit(1)
    return o


if __name__ == '__main__':
    ap = argparse.ArgumentParser()
    ap.add_argument("--raw", action="store_true", help="Do not optimize the wasm binary")
    args = ap.parse_args()

    go = require_go()
    opt = require_binaryen()

    env = {
        "GOOS": "js",
        "GOARCH": "wasm",
    }
    env.update(os.environ)

    tmp_dir = tempfile.TemporaryDirectory(prefix="gsa-wasm")
    tmp_file = tempfile.NamedTemporaryFile(dir=tmp_dir.name, delete=False)
    tmp_file.close()

    try:
        log("Building wasm binary")
        result = subprocess.run(
            [
                go,
                "build",
                "-trimpath",
                "-o", tmp_file.name,
                "./cmd/wasm/main_js_wasm.go"
            ],
            text=True,
            cwd=get_project_root(),
            stderr=subprocess.PIPE,
            stdout=subprocess.PIPE,
            timeout=120,
            env=env
        )
        result.check_returncode()
        log("Wasm binary built successfully")
    except subprocess.CalledProcessError as e:
        log("Error building wasm:")
        print(f"stdout: {e.stdout}")
        print(f"stderr: {e.stderr}")
        exit(1)

    if args.raw:
        shutil.copy(tmp_file.name, wasm_location())
    else:
        try:
            log("Optimizing wasm")
            result = subprocess.run(
                [
                    opt,
                    tmp_file.name,
                    "-O3",
                    "--enable-bulk-memory",
                    "-o", wasm_location()
                ],
                text=True,
                stderr=subprocess.PIPE,
                stdout=subprocess.PIPE,
                timeout=300
            )
            result.check_returncode()
            log("Wasm optimized successfully")
        except subprocess.CalledProcessError as e:
            log("Error optimizing wasm:")
            print(f"stdout: {e.stdout}")
            print(f"stderr: {e.stderr}")
            exit(1)

    tmp_dir.cleanup()