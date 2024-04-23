# go-size-analyzer

[![Go Report Card](https://goreportcard.com/badge/github.com/Zxilly/go-size-analyzer)](https://goreportcard.com/report/github.com/Zxilly/go-size-analyzer)
[![GitHub release](https://img.shields.io/github/v/release/Zxilly/go-size-analyzer)](https://github.com/Zxilly/go-size-analyzer/releases)
[![codebeat badge](https://codebeat.co/badges/1c911d47-6e4d-4f30-becb-939406fd8998)](https://codebeat.co/projects/github-com-zxilly-go-size-analyzer-master)

一个简单的工具，用于分析 Go 编译二进制文件的大小。

## 安装

从[发布页面](https://github.com/Zxilly/go-size-analyzer/releases)下载最新版本。

不推荐使用 `go install` 进行安装，因为它不会包含嵌入的 UI 模板，该模板对于网络模式是必需的。

## 使用

### Example

#### Web mode

```bash
$ gsa --web golang-compiled-binary
```

将在 8080 端口启动一个 web 服务器，您可以在浏览器中查看结果。

网页将如下所示：

![image](https://github.com/Zxilly/go-size-analyzer/assets/31370133/78bb8105-fc5a-4852-8704-8c2fac3bf475)

您可以点击以展开包以查看详细信息。

#### 文本模式

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

### 完整选项

```bash
用法：
  gsa [选项] [文件]

应用选项：
      --verbose                 详细输出
  -f, --format=[text|json|html] 输出格式 (默认: text)
  -o, --output=                 写入文件
      --version                 显示版本

文本选项：
      --hide-sections           隐藏 sections
      --hide-main               隐藏主包
      --hide-std                隐藏标准库

Json 选项：
      --indent=                 Json 输出的缩进

Html 选项：
      --web                     启动用于 html 输出的 web 服务器，此选项
                                会将格式覆盖为 html 并忽略输出
                                选项
      --listen=                 监听地址 (默认: :8080)
      --open                    打开浏览器

帮助选项：
  -h, --help                    显示此帮助消息

参数：
  file:                         要分析的二进制文件
```

> [!CAUTION]
>
> 该工具可以分析剥离 symbol 的二进制文件，但可能导致结果不准确。

## TODO

- [ ] 添加更多用于反汇编二进制文件的模式
- [ ] 从 dwarf 段提取信息
- [ ] 计算符号本身的大小到包中
- [ ] 添加其他图表，如火焰图、饼图等
- [ ] 支持 demangle cgo 中的 C++/Rust 符号

## Contribution

欢迎任何形式的贡献，随时提出问题或拉取请求。

## LICENSE

根据 [AGPL-3.0](./LICENSE) 发布。
