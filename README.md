# go-size-analyzer

English | [简体中文](./README_zh-CN.md)

[![Go Report Card](https://goreportcard.com/badge/github.com/Zxilly/go-size-analyzer)](https://goreportcard.com/report/github.com/Zxilly/go-size-analyzer)
[![GitHub release](https://img.shields.io/github/v/release/Zxilly/go-size-analyzer)](https://github.com/Zxilly/go-size-analyzer/releases)

A simple tool to analyze the size of a Go compiled binary. 

## Installation

Download the latest release from the [release page](https://github.com/Zxilly/go-size-analyzer/releases)

Use `go install` is not recommended, because it won't include the embed ui template, which is required for the web mode.

## Usage

### Example

#### Web mode

```bash
$ gsa --web golang-compiled-binary
```

Will start a web server on port 8080, you can view the result in your browser.

The web page will look like this:

![image](https://github.com/Zxilly/go-size-analyzer/assets/31370133/78bb8105-fc5a-4852-8704-8c2fac3bf475)


You can click to expand the package to see the details.

#### Text mode 

```bash
$ gsa docker-compose-linux-x86_64
+------------------------------------------------------------------------------+
| docker-compose-linux-x86_64                                                  |
+---------+-----------------------------------------------+--------+-----------+
| PERCENT | NAME                                          | SIZE   | TYPE      |
+---------+-----------------------------------------------+--------+-----------+
| 27.76%  | .gopclntab                                    | 17 MB  | section   |
| 15.17%  | .rodata                                       | 9.5 MB | section   |
| 11.63%  | k8s.io/api                                    | 7.3 MB | vendor    |
| 6.69%   | .strtab                                       | 4.2 MB | section   |
| 3.47%   | k8s.io/client-go                              | 2.2 MB | vendor    |
| 3.37%   | .symtab                                       | 2.1 MB | section   |
| 2.28%   | github.com/moby/buildkit                      | 1.4 MB | vendor    |
| 1.54%   | github.com/gogo/protobuf                      | 968 kB | vendor    |
| 1.53%   | github.com/google/gnostic-models              | 958 kB | vendor    |
| 1.33%   | github.com/aws/aws-sdk-go-v2                  | 836 kB | vendor    |
| 1.26%   | crypto                                        | 790 kB | std       |
| 1.25%   | google.golang.org/protobuf                    | 782 kB | vendor    |
| 1.24%   | k8s.io/apimachinery                           | 779 kB | vendor    |
| 1.24%   | net                                           | 777 kB | std       |
| 1.20%   | github.com/docker/compose/v2                  | 752 kB | main      |
| 0.95%   | .noptrdata                                    | 596 kB | section   |
| 0.93%   | go.opentelemetry.io/otel                      | 582 kB | vendor    |
| 0.85%   | google.golang.org/grpc                        | 533 kB | vendor    |
| 0.71%   | runtime                                       | 442 kB | std       |
| 0.59%   | github.com/docker/buildx                      | 371 kB | vendor    |
| 0.55%   | github.com/docker/docker                      | 347 kB | vendor    |
| 0.53%   |                                               | 331 kB | generated |
| 0.52%   | golang.org/x/net                              | 326 kB | vendor    |
| 0.47%   | github.com/theupdateframework/notary          | 294 kB | vendor    |

...[Collapsed]...

| 0.00%   | database/sql/driver                           | 128 B  | std       |
| 0.00%   | .note.go.buildid                              | 100 B  | section   |
| 0.00%   | hash/fnv                                      | 96 B   | std       |
| 0.00%   | maps                                          | 96 B   | std       |
| 0.00%   | github.com/moby/sys/sequential                | 64 B   | vendor    |
| 0.00%   | .text                                         | 1 B    | section   |
+---------+-----------------------------------------------+--------+-----------+
| 97.65%  | KNOWN                                         | 61 MB  |           |
| 100%    | TOTAL                                         | 63 MB  |           |
+---------+-----------------------------------------------+--------+-----------+

```

### Full options

```bash
Usage:
  gsa [OPTIONS] [file]

Application Options:
      --verbose                 Verbose output
  -f, --format=[text|json|html] Output format (default: text)
  -o, --output=                 Write to file
      --version                 Show version

Text Options:
      --hide-sections           Hide sections
      --hide-main               Hide main package
      --hide-std                Hide standard library

Json Options:
      --indent=                 Indentation for json output

Html Options:
      --web                     Start web server for html output, this option
                                will override format to html and ignore output
                                option
      --listen=                 Listen address (default: :8080)
      --open                    Open browser

Help Options:
  -h, --help                    Show this help message

Arguments:
  file:                         Binary file to analyze
```

> [!CAUTION]
>
> The tool can work with stripped binaries, but it may lead to inaccurate results.

## TODO

- [ ] Add more pattern for disassembling the binary
- [ ] Extract the information from the dwarf section
- [ ] Count the symbol size itself to package
- [ ] Add other chart like flame graph, pie chart etc.
- [ ] Support C++/Rust symbol demangling in cgo

## Contribution

Any contribution is welcome, feel free to open an issue or a pull request.

## LICENSE

Published under the [AGPL-3.0](./LICENSE).
