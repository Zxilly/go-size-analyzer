# go-size-analyzer

[![Go Report Card](https://goreportcard.com/badge/github.com/Zxilly/go-size-analyzer)](https://goreportcard.com/report/github.com/Zxilly/go-size-analyzer)
[![Tests](https://github.com/Zxilly/go-size-analyzer/actions/workflows/built-tests.yml/badge.svg)](https://github.com/Zxilly/go-size-analyzer/actions/workflows/built-tests.yml)
[![Codecov](https://img.shields.io/codecov/c/gh/Zxilly/go-size-analyzer)](https://codecov.io/github/Zxilly/go-size-analyzer)
[![GitHub release](https://img.shields.io/github/v/release/Zxilly/go-size-analyzer)](https://github.com/Zxilly/go-size-analyzer/releases)
[![Featured｜HelloGitHub](https://abroad.hellogithub.com/v1/widgets/recommend.svg?rid=c31575032ae144c6b22c553399a42742&claim_uid=QksOfglJ0njwLKa&theme=small)](https://hellogithub.com/repository/c31575032ae144c6b22c553399a42742)
[![OpenSSF Scorecard](https://api.scorecard.dev/projects/github.com/Zxilly/go-size-analyzer/badge)](https://scorecard.dev/viewer/?uri=github.com/Zxilly/go-size-analyzer)

一个简单的工具，用于分析 Go 编译二进制文件的大小。

[![Packaging status](https://repology.org/badge/vertical-allrepos/go-size-analyzer.svg)](https://repology.org/project/go-size-analyzer/versions)

## 安装

### [下载最新二进制文件](https://github.com/Zxilly/go-size-analyzer/releases)

### MacOS / Linux 通过 Homebrew 安装：

使用 [Homebrew](https://brew.sh/)
```
brew install go-size-analyzer
```

### Windows：

使用 [scoop](https://scoop.sh/)
```
scoop install go-size-analyzer
```

### Go 安装：
```
go install github.com/Zxilly/go-size-analyzer/cmd/gsa@latest
```

## 使用

### Example

#### Web mode

```bash
$ gsa --web golang-compiled-binary
```

将在 8080 端口启动一个 web 服务器，您可以在浏览器中查看结果。

或者您可以在浏览器中使用 WASM 版本：[GSA Treemap](https://gsa.zxilly.dev)

> [!NOTE]
> 由于浏览器的限制，wasm 版本比原生版本慢得多。
> 通常分析相同二进制文件需要 10 倍的时间。
> 
> 仅建议用于分析小型应用程序（大小小于 30 MB）

网页将如下所示：

![image](https://github.com/Zxilly/go-size-analyzer/assets/31370133/e69583ce-b189-4a0d-b108-c3b7d5c33a82)

您可以点击以展开包以查看详细信息。

#### 终端用户界面

```bash
$ gsa --tui golang-compiled-binary
```

![demo](https://github.com/Zxilly/go-size-analyzer/assets/31370133/9f38989e-ab9f-4098-a939-26ca23fef407)

#### 文本模式

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

#### Svg 模式

```bash
gsa cockroach-darwin-amd64 -f svg -o data.svg --hide-sections
```

![image](./assets/example.svg)

### 完整选项

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
> 该工具可以分析剥离 symbol 的二进制文件，但可能导致结果不准确。

## TODO

- [ ] 添加更多用于反汇编二进制文件的模式
- [x] 从 DWARF 段提取信息
- [x] 计算符号本身的大小到包中
- [ ] 添加其他图表，如火焰图、饼图等
- [ ] 支持 demangle cgo 中的 C++/Rust 符号
- [x] 添加 TUI 模式以探索
- [x] 编译为 wasm，创建一个在浏览器中分析二进制文件的用户界面

## Contribution

欢迎任何形式的贡献，随时提出问题或拉取请求。

## LICENSE

根据 [AGPL-3.0](./LICENSE) 发布。
