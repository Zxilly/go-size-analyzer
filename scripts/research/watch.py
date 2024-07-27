import os
import subprocess
import time

from watchdog.events import PatternMatchingEventHandler
from watchdog.observers import Observer

p = os.path.abspath(os.path.dirname(__file__))

compiling = False


def compile_go():
    global compiling
    if compiling:
        return
    compiling = True
    print("Compiling...")
    # os.system("go tool compile -S -o main.o main.go")
    out = subprocess.check_output(
        ["go", "tool", "compile", "-S", "-o", "main.o", "-trimpath", p, "main.go"],
        text=True)
    with open("main.s.txt", "w") as f:
        f.write(out)

    out = subprocess.check_output(["go", "tool", "objdump", "-s", "main.", "-S", "-gnu", "main.o"], text=True)
    with open("main.r.txt", "w") as f:
        f.write(out)

    print("Compiled")
    compiling = False


class ReCompileHandler(PatternMatchingEventHandler):
    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self._last_event_time = time.time()

    def on_modified(self, event):
        if time.time() - self._last_event_time < 1:
            return
        compile_go()
        self._last_event_time = time.time()


if __name__ == '__main__':

    event_handler = ReCompileHandler(
        patterns=["main.go"],
        ignore_directories=True,
    )
    observer = Observer()
    observer.schedule(event_handler, p, recursive=False)
    observer.start()
    try:
        while True:
            time.sleep(1)
    finally:
        observer.stop()
        observer.join()
