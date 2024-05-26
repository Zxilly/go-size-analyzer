import os.path
import subprocess

from lib.utils import get_project_root, require_go


def wasm_location() -> str:
    return os.path.join(get_project_root(), "ui", "gsa.wasm")


if __name__ == '__main__':
    go = require_go()

    env = {
        "GOOS": "js",
        "GOARCH": "wasm",
    }
    env.update(os.environ)

    try:
        result = subprocess.run(
            [
                go,
                "build",
                "-o", wasm_location(),
                "-ldflags=-s -w",
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
    except subprocess.CalledProcessError as e:
        print("Error building wasm:")
        print(f"stdout: {e.stdout}")
        print(f"stderr: {e.stderr}")
        exit(1)
