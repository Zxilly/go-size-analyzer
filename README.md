# gsv

[![Build and publish](https://github.com/Zxilly/go-size-view/actions/workflows/build.yml/badge.svg)](https://github.com/Zxilly/go-size-view/actions/workflows/build.yml)
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2FZxilly%2Fgo-size-view.svg?type=shield)](https://app.fossa.com/projects/git%2Bgithub.com%2FZxilly%2Fgo-size-view?ref=badge_shield)


A simple tool to view the size of a Go compiled binary. 

Build on top of [bloaty](https://github.com/google/bloaty).

## Usage

```bash
Analysis golang compiled binary size

Usage: gsv [OPTIONS] <BINARY>

Arguments:
  <BINARY>  The binary to analysis

Options:
  -p, --port <PORT>  The port to listen on for web mode [default: 8888]
  -w, --web          View the result in the browser
  -h, --help         Print help
```

## Example

### Web mode

```bash
$ gsv --web golang-compiled-binary
```

Will start a web server on port 8888, you can view the result in your browser.

The web page will look like this:

![image](https://user-images.githubusercontent.com/31370133/225002647-1e37e52f-dada-4adb-a33b-e806396621cf.png)


You can click the darker part to see the detail, and click the top bar to return to the previous level.

### Text mode 

```bash
$ gsv golang-compiled-binary
github.com/swaggo/files                             : 8.19MB
.gopclntab                                          : 6.80MB
Debug Section                                       : 6.34MB
github.com/spf13/cobra                              : 4.16MB
ariga.io/atlas/sql                                  : 1.40MB
C                                                   : 1.14MB
net/http                                            : 1.09MB
google.golang.org/protobuf/internal                 : 1.07MB
github.com/ZNotify/server                           : 1.03MB
golang.org/x/net                                    : 965.35KB

...[Collapsed]...

runtime/cgo                                         : 78.00B
internal/race                                       : 46.00B
.comment                                            : 43.00B
.note.gnu.build-id                                  : 36.00B
.note.gnu.property                                  : 32.00B
.interp                                             : 28.00B
.eh_frame_hdr                                       : 28.00B
.init                                               : 27.00B
.rela.plt                                           : 24.00B
.rela.dyn                                           : 24.00B
.fini_array                                         : 8.00B
.plt.got                                            : 8.00B
.init_array                                         : 8.00B
Total                                               : 55.08MB

```

## Limitations

Since lots of workaround for static build is used, currently `gsv` only works on linux platform.

Constant string in Go binary is stored in `.rodata` section, analysis on this section depends on disassembler. Even if a constant is used by multiple packages within the program, the size of the constant will only be attributed to the package where it is first encountered during decompilation. This is a limitation of `bloaty`. For more information, please refer to [this](https://github.com/google/bloaty/blob/main/doc/how-bloaty-works.md)

As for now, `gsv` can only analyze `elf64` binary. PE and Mach-O support will be added in the future.

## Todo

- [ ] Support PE and Mach-O
- [ ] Support Windows and MacOS
- [ ] Remove dependency on `bloaty`

You can find a pure rust version without dependency on `bloaty` at `pure` branch. Since it can not disassemble `.rodata` section, the result is not accurate. The work is still in progress.

## Contribution

Any contribution is welcome, feel free to open an issue or a pull request.

## LICENSE

Published under the [MPL-2.0](https://www.mozilla.org/en-US/MPL/2.0/).

[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2FZxilly%2Fgo-size-view.svg?type=large)](https://app.fossa.com/projects/git%2Bgithub.com%2FZxilly%2Fgo-size-view?ref=badge_large)