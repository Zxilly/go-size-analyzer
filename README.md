# gsv

A simple tool to view the size of a Go compiled binary. 

Build on top of [bloaty](https://github.com/google/bloaty).

## Usage

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

## LICENSE

Published under the [MPL-2.0](https://www.mozilla.org/en-US/MPL/2.0/).