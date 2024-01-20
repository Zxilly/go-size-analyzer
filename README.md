# gsv


A simple tool to view the size of a Go compiled binary. 

> [!CAUTION]
> The GSV is currently being refactored to be implemented in Golang, and search based on the pclntab table will be implemented. Debug information will no longer be a prerequisite.
> 
> If you want to view the code of the old version, please check the `rust` branch.

## Usage

you can use `gsv` to analyze the binary:

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

### Example

#### Web mode

```bash
$ gsv --web golang-compiled-binary
```

Will start a web server on port 8888, you can view the result in your browser.

The web page will look like this:

![image](https://user-images.githubusercontent.com/31370133/225002647-1e37e52f-dada-4adb-a33b-e806396621cf.png)


You can click the darker part to see the detail, and click the top bar to return to the previous level.

#### Text mode 

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

## Contribution

Any contribution is welcome, feel free to open an issue or a pull request.

## LICENSE

Published under the [AGPL-3.0](./LICENSE).
