import os.path
import shutil
import subprocess
import tempfile

from lib.utils import get_project_root, require_go


def wasm_location() -> str:
    return os.path.join(get_project_root(), "ui", "gsa.wasm")


def require_binaryen():
    o = shutil.which("wasm-opt")
    if o is None:
        print("wasm-opt not found in PATH. Please install binaryen.")
        exit(1)
    return o


if __name__ == '__main__':
    go = require_go()
    opt = require_binaryen()

    env = {
        "GOOS": "js",
        "GOARCH": "wasm",
    }
    env.update(os.environ)

    with tempfile.NamedTemporaryFile(mode="w+b", delete=False) as tmp_file:
        try:
            print("Building wasm binary")

            result = subprocess.run(
                [
                    go,
                    "build",
                    "-o", tmp_file,
                    "./cmd/wasm/main_wasm.go"
                ],
                text=True,
                cwd=get_project_root(),
                stderr=subprocess.PIPE,
                stdout=subprocess.PIPE,
                timeout=120,
                env=env
            )

            result.check_returncode()

            print("Wasm binary built successfully")
        except subprocess.CalledProcessError as e:
            print("Error building wasm:")
            print(f"stdout: {e.stdout}")
            print(f"stderr: {e.stderr}")
            exit(1)

        try:
            print("Optimizing wasm")

            result = subprocess.run(
                [
                    opt,
                    tmp_file,
                    "-O4",
                    "--enable-bulk-memory",
                    "-o", wasm_location()
                ],
                text=True,
                stderr=subprocess.PIPE,
                stdout=subprocess.PIPE,
                timeout=120
            )

            result.check_returncode()

            print("Wasm optimized successfully")
        except subprocess.CalledProcessError as e:
            print("Error optimizing wasm:")
            print(f"stdout: {e.stdout}")
            print(f"stderr: {e.stderr}")
            exit(1)
