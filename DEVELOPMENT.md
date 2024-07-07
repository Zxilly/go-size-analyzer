# Development Guide

An overview of the tools and scripts used in the project.

## Golang

The project currently runs with the `Go Toolchain`. Use any Go version that supports it.

The project has some build tags that can be used to build the project with different configurations.

### Build Tags

#### `embed`

This tag is used to embed the static files in the binary. Without this tag, the static files will be downloaded
from the `v1` tag of the release.

#### `js` and `wasm`

These tags are used to build the project to WebAssembly. With this tag enabled, some printers are disabled and the
json marshaler will be changed to the `wasm` marshaler.

#### Profiler tags

##### `pgo`

Enables the `cpu` profiler to collect information for profile-guided optimization.

##### `profiler`

Activate all profilers while running integration tests.

### Tests

#### Unit Tests

Unit tests are divided into several parts.

- `embed` build tag
- `GOOS=js GOARCH=wasm` build constraints

For wasm tests, you need a runner for wasm files.

You can try [Zxilly/go_js_wasm_exec](https://github.com/Zxilly/go_js_wasm_exec).

> This runner requires `node.js` to be installed on your machine. It's a wrapper for the official `misc/wasm/go_js_wasm_exec` runner.

Or you can use [agnivade/wasmbrowsertest](https://github.com/agnivade/wasmbrowsertest),
but it has bugs in the Go toolchain environment, see [issue](https://github.com/agnivade/wasmbrowsertest/issues/61).

#### Integration tests

Integration tests should be executed by the helper scripts described below.
It runs on the binaries from [Zxilly/go-testdata](https://github.com/Zxilly/go-testdata)

### Helper scripts

Helper scripts are managed by `poetry`. Make sure you have it installed.

All scripts are in the `scripts` directory.

#### Download the binary

Download the binary from [Zxilly/go-testdata](https://github.com/Zxilly/go-testdata).

```bash
python scripts/ensure.py --help
usage: ensure.py [-h] [--example] [--real]

options:
  -h, --help  show this help message and exit
  --example   Download example binaries.
  --real      Download real binaries.
```

#### Tests

```bash
python scripts/tests.py --help
usage: tests.py [-h] [--unit-full] [--unit-wasm] [--unit-embed] [--unit] [--integration-example] [--integration-real] [--integration]

options:
  -h, --help            show this help message and exit
  --unit-full           Run full unit tests.
  --unit-wasm           Run unit tests for wasm.
  --unit-embed          Run unit tests for embed
  --unit                Run unit tests.
  --integration-example
                        Run integration tests for small binaries.
  --integration-real    Run integration tests for large binaries.
  --integration         Run all integration tests.
```

##### Integration binary source generator

GeGenerate `scripts/binaries.csv` as source for tests.

```bash
python scripts/generate.py
```

##### Reporter for CI

Collect the test results and generate a report in GitHub Actions format.

Requires `svgo` to be installed to optimize the SVG files.

It uses service [Zxilly/data2image](https://github.com/Zxilly/data2image) to bypass
GitHub data-uri limit. This project was written in `Rust`.

The svg data was optimized with `svgo` then compressed with `zstd` and encoded with `base64`
for safe inclusion in the url.

```bash
python scripts/report.py
```

#### Build

##### Wasm

Requires [WebAssembly/binaryen](https://github.com/WebAssembly/binaryen) installed to optimize the wasm binary.

```bash
python scripts/wasm.py --help
usage: wasm.py [-h] [--raw]

options:
  -h, --help  show this help message and exit
  --raw       Do not optimize the wasm binary
```

##### Profile Guided Optimization

Collect the profile data, then build `./cmd/gsa` with the profile data.

```bash
python scripts/pgo.py
```

### Linter

The project uses `golangci-lint` to lint the code.

> Golangci-lint doesn't officially support `Go 1.23` at the moment. Use the preview version.

```bash
golangci-lint run
```

## TypeScript and React

The project uses `TypeScript` and `React` to build the web interface.
Most of the files are located in the `ui` directory.

### Build

The project uses `pnpm` to manage the dependencies.

```bash
pnpm install
```

#### Explorer

The explorer needs a `wasm` file built from the go part.

Just running `scripts/wasm.py` should put it in the right place.

```bash
pnpm run dev:explorer # Development
pnpm run build:explorer # Production
```

#### Webui

The built file `./ui/dist/index.html` should be placed in `internal/webui` to be embedded in the binary.
Don't forget to set the `embed` build tag.

```bash
pnpm run dev:ui # Development
pnpm run build:ui # Production
```

### Tests

```bash
pnpm run test
```

### Linter

The project uses `eslint` to lint the code.

```bash
pnpm run lint
pnpm run lint:fix
```

## Typos

The project uses [crate-ci/typos](https://github.com/crate-ci/typos) to find typos.

