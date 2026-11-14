# 二进制体积定位：第二轮生产优化

2026-11-13。接续[第一轮生产接入](size-attribution-iteration.md)，并行优化静态对象差异、Web 报告传输与大报告渲染，使用 example、Docker Compose、analysis-server 和保存的 GSA 构建做对照。原始二进制只读，不执行；产物、日志和 SHA256 位于 `temp/size-iteration2/`。

## 静态对象差异

原有包和函数 diff 之外，现在输出 `symbols`，按包、原始符号名和类型匹配。地址仅用于单份报告内去重，不用于跨构建猜测身份。相同已知地址与大小的重复证据合并；多个不同或未知范围使用相同身份时显式跳过。缺失 symbol 列表、类型或大小保持未知，不推断为零或删除。

真实编译 fixture 交换两个全局 string 的长度（2,048 / 8,192 B），文件和包总体积完全不变，仍能找到：

| 对象 | 变化 |
|---|---:|
| `main.Alpha.string` | +6,144 B |
| `main.Beta.string` | −6,144 B |

保存的真实 GSA default/embed 构建对照找到 `internal/webui.tmpl.string` 新增 **282,356 B**，同时保留 717 条函数变化和 773 条符号变化。整文件变化为 −411,136 B；两个构建采用不同的 UI 获取方式，不据此宣称功能等价的体积优化。

Compose 自比的包、函数、符号变化均为零；567 个不唯一符号身份得到诊断。文本报告每类身份冲突最多展示 10 条并提示剩余数量，JSON 保留全部记录。符号大小代表报告中的范围，可能包含共享引用或生成元数据，与其他符号、函数发生重叠，不能相加为物理增长量或删除收益。

## Web 报告传输

HTML 和 WASM 使用专门的 UI 投影：保留完整函数名、代码量、pcln 总量、已检测引用量及所有已采样证据，省去 UI 未读取的函数分项与范围列表、完整 `coverage.regions`。完整 CLI JSON 不变，compact JSON 继续保留用户请求的完整物理覆盖台账。投影不修改共享的分析结果对象。

控制实验先用第一轮 CLI 冻结完整分析结果，再用本轮 serializer 投影相同对象，避免把分析器变化混入传输差异。以下均为嵌入 JSON 的 UTF-8 字节数；旧、新模板不同，不将 HTML 总文件差异当成投影效果。

| 输入 | 模式 | 原 payload | 新 payload | 减少 |
|---|---|---:|---:|---:|
| Go 1.23 Linux example | 默认 | 1,093,677 B | 643,503 B | 41.16% |
| Docker Compose | 默认 | 39,189,377 B | 24,153,174 B | 38.37% |
| analysis-server | 默认 | 20,602,469 B | 13,527,897 B | 34.34% |
| Go 1.23 Linux example | coverage details | 4,261,543 B | 819,702 B | 80.77% |
| Docker Compose | coverage details | 160,179,741 B | 31,163,158 B | 80.54% |
| analysis-server | coverage details | 109,869,826 B | 15,659,976 B | 85.75% |

三个样本分别保留 2,541 / 75,714 / 43,506 个函数；详细模式的 1,361 / 55,287 / 16,089 条函数证据样本逐项一致。六组输出检查了 full JSON roundtrip、compact ledger、函数指标、未知与已知零值、证据内容及 HTML 嵌入数据的一致性。原生与 WASM 桥接一致性另由生产测试验证。

## 大报告渲染

Treemap 根据视口逐层建立层级。只有显示面积足够的节点继续展开；分区仍使用完整子项预算，不把被省略的小矩形面积重新分摊给大对象。文件内函数仍在进入该文件后才展开。选中父项可打开完整子项列表，支持名称过滤和每页 50 项；小对象也能通过保存的 hash 进入。

同一完整 JSON、同一 1,440×900 视口的前后对照：

| 输入 | 原完整层级 | 新保留层级 | 原首屏 DOM 节点 | 新首屏 DOM 节点 |
|---|---:|---:|---:|---:|
| Docker Compose | 79,344 | 858 | 3,882 | 858 |
| analysis-server | 54,961 | 640 | 2,356 | 640 |

这里 DOM 节点指 `.node` 图块，不是全部 SVG 元素。为了保持 squarify 分区，已打开父级的直接子项仍需临时创建：Compose 61,442 个、analysis-server 40,010 个，不能称只处理了 858 / 640 个对象。

隔离其他实验后，每阶段三个独立 Chromium 会话的首个图块提交时间中位数为 Compose **767.9→599.2 ms**、analysis-server **481.8→374.2 ms**。这只是当前机器的观察，包含页面处理且存在明显波动，不构成性能保证。输入使用同一未精简 JSON，未叠加传输节省；Chromium heap 快照受 GC 影响，没有据此宣称内存下降。

回归检查覆盖物理面积和坐标与完整 D3 分区的一致性、小区域不展开、隐藏对象 hash 恢复、零尺寸视口、500 层路径、列表分页和搜索，以及切换新报告时清除旧选择。

## 完整 lint

原本未结束的 golangci-lint 已通过有界对照定位到旧工具的 SA4023 相关分析路径：2.13.0（Go 1.27.0 构建、honnef `v0.8.0-rc.1`）单独运行 SA4023、关闭测试仍挂起。debug 日志最后停在 `nilness@github.com/eliben/watgo/internal/validate`，未进一步确定具体函数或上游缺陷。

本地从缓存的官方模块源码构建 golangci-lint 2.15.0（同一 Go 1.27.0、honnef `v0.8.2`），保持原配置和全部 18 个 linter，对根包、`cmd/...`、`internal/...` 运行结果为 **exit 0、0 issues**。外部总时长 16.24 秒仅用作正常终止的证据。未关闭规则、未修改生产代码绕过、未替换全局工具；完整探针与工具 hash 保存在 `lint/findings.md`、`lint/summary.json`。

## 复现与产物

提交前将 lint 范围扩展到 `experiments/size-attribution/...`，修正实验工具的格式、排序 API、类型断言和错误处理后，生产包与实验包合并检查为 **0 issues**。实验 Go 测试、buildassist 的 10 项 Python 测试及文档拼写检查通过。受控 fixture 保留内置打印和原始 marker 字节，分别使用局部 lint 说明和精确拼写白名单，避免引入 `fmt` 改变样本依赖；既有测量数据仍对应各轮保存的二进制。

```powershell
$env:GOFLAGS='-mod=readonly -p=2'
$env:GOMAXPROCS='4'
go build -o temp/size-iteration2/gsa.exe ./cmd/gsa
python experiments/size-attribution/symbols_run.py
python experiments/size-attribution/webpayload/run.py --baseline-cli temp/size-iteration/gsa-embedded.exe

# 同一完整 JSON 的 UI 对照；先保存上一轮模板，再构建新模板
python ui/scripts/render-benchmark.py --phase baseline-quiet --template temp/size-iteration2/ui/baseline/template.html
python ui/scripts/render-benchmark.py --phase after --template ui/dist/webui/index.html

go test . ./internal/... ./cmd/...
./temp/size-iteration2/lint/golangci-lint-2.15.exe run -v --timeout 60s . ./cmd/... ./internal/...
```

`symbols_run.py` 会记录输入 hash，检查 Compose 自比无差异，并在存在保存构建时检查 GSA 模板符号。Web 对照脚本支持自选 baseline、重复 `--sample LABEL=PATH`、输出目录及 `--compare-only`，见[说明](../experiments/size-attribution/webpayload/README.md)。保留第一轮 CLI 和模板，避免覆盖基线。

实际 CLI：`temp/size-iteration2/gsa.exe` 与 `gsa-embedded.exe`。后者通过 `ui/embed-overlay.json` 嵌入本轮 UI，未覆盖先前模板，生成了 `ui/example.html`（Go 1.27 Linux）和 `ui/compose.html`，均开启 `--coverage-details`。传输控制实验的 example 为 Go 1.23，两个样本不可混作同一组。

原生生产测试通过；WASM entity、marshaler、result 的全部 `Test*` 通过，新的 WASM 已用于 Explorer 构建。Windows Node 下默认 WASM fuzz seed 读取仍受 `O_DIRECTORY` 限制，未把它计作通过。UI 的 91 项测试、ESLint、TypeScript、WebUI 与 Explorer 构建通过。两个真实报告用例默认跳过；另行启用后，渲染测量在 Compose 和 analysis-server 上通过，函数/schema 用例在新的 Compose Web 投影上通过。

最终真实浏览器检查使用上述 native printer 生成的 HTML。Compose 的包→文件→函数下钻、列表搜索、返回文件和 hash 刷新恢复均通过；`runtime.moduledataverify1` 显示 **315 B** 引用及 **8/22** 条证据，保留 `.rodata / disasm / reference / heuristic` 和物理范围。example 页面亦正常加载，控制台无错误。截图和检查结果在 `ui/browser-checks.json`，浏览器与本地 server 已关闭。更多测量说明见 [UI 复现文档](../ui/scripts/render-benchmark.md)。

符号身份仍有无法安全比较的场景；外部符号、inline/generic overlay、链接 witness、CFG 等上一轮原型没有被扩大为默认生产行为。物理 ledger 继续作为文件总量的唯一合计依据。
