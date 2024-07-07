# go-size-analyzer

[![Go Report Card](https://goreportcard.com/badge/github.com/Zxilly/go-size-analyzer)](https://goreportcard.com/report/github.com/Zxilly/go-size-analyzer)
[![Tests](https://github.com/Zxilly/go-size-analyzer/actions/workflows/built-tests.yml/badge.svg)](https://github.com/Zxilly/go-size-analyzer/actions/workflows/built-tests.yml)
[![Codecov](https://img.shields.io/codecov/c/gh/Zxilly/go-size-analyzer)](https://codecov.io/github/Zxilly/go-size-analyzer)
[![GitHub release](https://img.shields.io/github/v/release/Zxilly/go-size-analyzer)](https://github.com/Zxilly/go-size-analyzer/releases)
[![go-recipes](https://raw.githubusercontent.com/nikolaydubina/go-recipes/main/badge.svg?raw=true)](https://github.com/nikolaydubina/go-recipes?tab=readme-ov-file#-visualise-dependencies-size-in-compiled-binaries-with-go-size-analyzer)
[![OpenSSF Scorecard](https://api.scorecard.dev/projects/github.com/Zxilly/go-size-analyzer/badge)](https://scorecard.dev/viewer/?uri=github.com/Zxilly/go-size-analyzer)

English | [简体中文](./README_zh_CN.md)

A simple tool to analyze the size of a Go compiled binary.

- [x] Cross-platform support for analyzing `ELF`, `Mach-O`, and `PE` binary formats
- [x] Detailed size breakdown by packages and sections
- [x] Support multiple output formats: `text`, `json`, `html`, `svg`
- [x] Interactive exploration via web interface and terminal UI
- [x] Binary comparison with diff mode (supports `json` and `text` output)

## Installation

[![Packaging status](https://repology.org/badge/vertical-allrepos/go-size-analyzer.svg)](https://repology.org/project/go-size-analyzer/versions)

### [Download the latest binary](https://github.com/Zxilly/go-size-analyzer/releases)

### MacOS / Linux via Homebrew:

Using [Homebrew](https://brew.sh/)
```
brew install go-size-analyzer
```

### Windows:

Using [scoop](https://scoop.sh/)
```
scoop install go-size-analyzer
```

### Go Install:
```
go install github.com/Zxilly/go-size-analyzer/cmd/gsa@latest
```

## Usage

### Example

#### Web mode

```bash
$ gsa --web golang-compiled-binary
```

Will start a web server on port 8080, you can view the result in your browser.

Or you can use the WASM version in the browser: [GSA Treemap](https://gsa.zxilly.dev)

> [!NOTE]  
> Due to the limitation of the browser, the wasm version is much slower than the native version.
> Normally costs 10x time to analyze the same binary.
> 
> Only recommended for analysing small applications (less than 30 MB in size)

The web page will look like this:

![image](https://github.com/Zxilly/go-size-analyzer/assets/31370133/e69583ce-b189-4a0d-b108-c3b7d5c33a82)

You can click to expand the package to see the details.

#### Terminal UI

```bash
$ gsa --tui golang-compiled-binary
```

![demo](https://github.com/Zxilly/go-size-analyzer/assets/31370133/9f38989e-ab9f-4098-a939-26ca23fef407)

#### Text mode 

```bash
$ gsa docker-compose-linux-x86_64
┌─────────────────────────────────────────────────────────────────────────────────┐
│ docker-compose-linux-x86_64                                                     │
├─────────┬──────────────────────────────────────────────────┬────────┬───────────┤
│ PERCENT │ NAME                                             │ SIZE   │ TYPE      │
├─────────┼──────────────────────────────────────────────────┼────────┼───────────┤
│ 17.37%  │ k8s.io/api                                       │ 11 MB  │ vendor    │
│ 15.52%  │ .rodata                                          │ 9.8 MB │ section   │
│ 8.92%   │ .gopclntab                                       │ 5.6 MB │ section   │
│ 7.51%   │ .strtab                                          │ 4.7 MB │ section   │
│ 5.13%   │ k8s.io/client-go                                 │ 3.2 MB │ vendor    │
│ 3.36%   │ .symtab                                          │ 2.1 MB │ section   │
│ 3.29%   │ github.com/moby/buildkit                         │ 2.1 MB │ vendor    │
│ 2.02%   │ google.golang.org/protobuf                       │ 1.3 MB │ vendor    │
│ 1.96%   │ github.com/google/gnostic-models                 │ 1.2 MB │ vendor    │
│ 1.82%   │ k8s.io/apimachinery                              │ 1.1 MB │ vendor    │
│ 1.73%   │ net                                              │ 1.1 MB │ std       │
│ 1.72%   │ github.com/aws/aws-sdk-go-v2                     │ 1.1 MB │ vendor    │
│ 1.57%   │ crypto                                           │ 991 kB │ std       │
│ 1.53%   │ github.com/docker/compose/v2                     │ 964 kB │ vendor    │
│ 1.48%   │ github.com/gogo/protobuf                         │ 931 kB │ vendor    │
│ 1.40%   │ runtime                                          │ 884 kB │ std       │
│ 1.32%   │ go.opentelemetry.io/otel                         │ 833 kB │ vendor    │
│ 1.28%   │ .text                                            │ 809 kB │ section   │
│ 1.18%   │ google.golang.org/grpc                           │ 742 kB │ vendor    │

...[Collapsed]...

│ 0.00%   │ github.com/google/shlex                          │ 0 B    │ vendor    │
│ 0.00%   │ github.com/pmezard/go-difflib                    │ 0 B    │ vendor    │
│ 0.00%   │ go.uber.org/mock                                 │ 0 B    │ vendor    │
│ 0.00%   │ github.com/kballard/go-shellquote                │ 0 B    │ vendor    │
│ 0.00%   │ tags.cncf.io/container-device-interface          │ 0 B    │ vendor    │
│ 0.00%   │ github.com/josharian/intern                      │ 0 B    │ vendor    │
│ 0.00%   │ github.com/shibumi/go-pathspec                   │ 0 B    │ vendor    │
│ 0.00%   │ dario.cat/mergo                                  │ 0 B    │ vendor    │
│ 0.00%   │ github.com/mattn/go-colorable                    │ 0 B    │ vendor    │
│ 0.00%   │ github.com/secure-systems-lab/go-securesystemslib│ 0 B    │ vendor    │
├─────────┼──────────────────────────────────────────────────┼────────┼───────────┤
│ 100%    │ KNOWN                                            │ 63 MB  │           │
│ 100%    │ TOTAL                                            │ 63 MB  │           │
└─────────┴──────────────────────────────────────────────────┴────────┴───────────┘

```

#### Diff Mode

```bash
$ gsa bin-linux-1.21-amd64 bin-linux-1.22-amd64
┌────────────────────────────────────────────────────────────────┐
│ Diff between bin-linux-1.21-amd64 and bin-linux-1.22-amd64     │
├─────────┬──────────────────────┬──────────┬──────────┬─────────┤
│ PERCENT │ NAME                 │ OLD SIZE │ NEW SIZE │ DIFF    │
├─────────┼──────────────────────┼──────────┼──────────┼─────────┤
│ +29.34% │ runtime              │ 782 kB   │ 1.0 MB   │ +229 kB │
│ add     │ internal/chacha8rand │          │ 3.1 kB   │ +3.1 kB │
│ +5.70%  │ <autogenerated>      │ 18 kB    │ 19 kB    │ +1.0 kB │
│ +8.70%  │ internal/abi         │ 6.0 kB   │ 6.6 kB   │ +525 B  │
│ +10.88% │ internal/cpu         │ 4.7 kB   │ 5.2 kB   │ +515 B  │
│ +4.45%  │ internal/reflectlite │ 3.9 kB   │ 4.1 kB   │ +173 B  │
│ +2.64%  │ internal/bytealg     │ 1.5 kB   │ 1.5 kB   │ +39 B   │
│ +0.81%  │ strconv              │ 4.0 kB   │ 4.0 kB   │ +32 B   │
│ +0.19%  │ syscall              │ 13 kB    │ 13 kB    │ +24 B   │
│ -0.17%  │ main                 │ 19 kB    │ 19 kB    │ -32 B   │
│ -0.37%  │ embed                │ 8.6 kB   │ 8.6 kB   │ -32 B   │
│ -0.38%  │ reflect              │ 25 kB    │ 25 kB    │ -96 B   │
│ -0.26%  │ time                 │ 87 kB    │ 86 kB    │ -224 B  │
│ -8.09%  │ sync                 │ 9.3 kB   │ 8.6 kB   │ -755 B  │
├─────────┼──────────────────────┼──────────┼──────────┼─────────┤
│ +7.59%  │ .rodata              │ 143 kB   │ 153 kB   │ +11 kB  │
│ +5.04%  │ .gopclntab           │ 144 kB   │ 152 kB   │ +7.3 kB │
│ +3.61%  │ .debug_info          │ 168 kB   │ 174 kB   │ +6.1 kB │
│ +3.52%  │ .debug_loc           │ 81 kB    │ 84 kB    │ +2.9 kB │
│ +3.03%  │ .debug_line          │ 80 kB    │ 82 kB    │ +2.4 kB │
│ +3.39%  │ .symtab              │ 59 kB    │ 61 kB    │ +2.0 kB │
│ +4.29%  │ .debug_frame         │ 29 kB    │ 30 kB    │ +1.2 kB │
│ +1.31%  │ .strtab              │ 61 kB    │ 62 kB    │ +795 B  │
│ +3.28%  │ .debug_ranges        │ 13 kB    │ 13 kB    │ +415 B  │
│ +5.13%  │ .data                │ 5.0 kB   │ 5.2 kB   │ +256 B  │
│ +7.32%  │ .typelink            │ 1.3 kB   │ 1.3 kB   │ +92 B   │
│ +27.78% │ .go.buildinfo        │ 288 B    │ 368 B    │ +80 B   │
│ -1.56%  │ .debug_gdb_scripts   │ 64 B     │ 63 B     │ -1 B    │
│ -0.63%  │ .noptrdata           │ 2.5 kB   │ 2.5 kB   │ -16 B   │
│ -6.48%  │ .shstrtab            │ 247 B    │ 231 B    │ -16 B   │
├─────────┼──────────────────────┼──────────┼──────────┼─────────┤
│ +3.86%  │ bin-linux-1.21-amd64 │ 1.6 MB   │ 1.6 MB   │ +61 kB  │
│         │ bin-linux-1.22-amd64 │          │          │         │
└─────────┴──────────────────────┴──────────┴──────────┴─────────┘
```

#### Svg Mode

```bash
$ gsa cockroach-darwin-amd64 -f svg -o data.svg --hide-sections
```

![image](./assets/example.svg)

### Full options

```bash
Usage: 
	gsa <file> [flags]
	gsa <old file> <new file> [flags]

A tool for determining the extent to which dependencies contribute to the
bloated size of compiled Go binaries.

Arguments:
  <file>           Binary file to analyze or result json file for diff
  [<diff file>]    New binary file or result json file to compare, optional

Flags:
  -h, --help             Show context-sensitive help.
      --verbose          Verbose output
  -f, --format="text"    Output format, possible values: text,json,html,svg
      --no-disasm        Skip disassembly pass
      --no-symbol        Skip symbol pass
      --no-dwarf         Skip dwarf pass
  -o, --output=STRING    Write to file
      --version          Show version

Text output options
  --hide-sections    Hide sections
  --hide-main        Hide main package
  --hide-std         Hide standard library

Json output options
  --indent=INDENT    Indentation for json output
  --compact          Hide function details, replacement with size

Svg output options
  --width=1028         Width of the svg treemap
  --height=640         Height of the svg treemap
  --margin-box=4       Margin between boxes
  --padding-box=4      Padding between box border and content
  --padding-root=32    Padding around root content

Web interface options
  --web               use web interface to explore the details
  --listen=":8080"    listen address
  --open              Open browser

Terminal interface options
  --tui    use terminal interface to explore the details

```

> [!CAUTION]
>
> The tool can work with stripped binaries, but it may lead to inaccurate results.

## TODO

- [ ] Add more pattern for disassembling the binary
- [x] Extract the information from the DWARF section
- [x] Count the symbol size itself to package
- [ ] Add other charts like flame graph, pie chart, etc.
- [ ] Support C++/Rust symbol demangling in cgo
- [x] Add a TUI mode for exploring details
- [x] Compile to wasm, create a ui to analyze the binary in the browser

## Contribution

Any contribution is welcome, feel free to open an issue or a pull request.

For development, see the [Development Guide](./DEVELOPMENT.md) for more details.

## LICENSE

Published under the [AGPL-3.0](./LICENSE).
