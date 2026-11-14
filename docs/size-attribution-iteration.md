# 二进制体积定位：第一轮生产接入

2026-11-13。接续[并行实验](size-attribution-experiments.md)，将静态指针归属、函数级差异和 Web 函数证据下钻接入生产代码。原始输入只读，旧分析器保存在 `temp/size-experiments/gsa.exe`；新产物、JSON、日志和输入 SHA256 在 `temp/size-iteration/`。没有把归属改进当成二进制缩小。

后续[第二轮优化](size-attribution-iteration2.md)已加入符号级差异、精简 Web 传输和按视口展开，并用新版工具完成之前未结束的全量 lint。本文保留第一轮当时的测量和验证状态。

## 已接入

- **静态指针**：DWARF 普通指针沿文件支撑的对象继续遍历结构体、数组、string 和 slice。使用地址与类型去重，限制深度与访问量，拒绝越界、union、不完整类型以及 interface/map/chan/func/unsafe 等运行时容器。所有新增目标保留 `reference` 角色。ELF、ELF PIE、PE、Mach-O 的真实编译 fixture 已验证。
- **函数计量与证据**：保留完整 linker/DWARF 名 `full_name`；`referenced_bytes` 是已检测引用范围的物理文件并集，不计入原有函数面积。未分析或不支持时省略，已分析但未发现时为 0。`--coverage-details` 增加每函数最多 8 条去重证据样本和总记录数，包含文件 offset/size、section、source、role、boundary；没有伪造指令 PC 或完整保留链。
- **函数级 diff**：保留包和 section 的原有输出，增加代码、pcln、引用数据各自的 from/to/delta，提示构建上下文变化。按包、函数类型和原始名称匹配，不按地址猜测。旧 schema 和 compact 中缺失的字段不当成零或删除；冲突身份显式跳过。
- **Web 下钻**：文件节点可展开函数，函数详情显示计量和证据表。只展开当前文件，引用范围不增加 treemap 面积。HTML 和 WASM 桥接传递函数；`--compact` 继续隐藏函数。

## 真实样本结果

下表的增加量全部来自原来未分类的字节，recognized 没有被转移给包。原有函数数量及代码总量保持不变。

| 输入 | 新增 attributed | 原 unclassified | 新 unclassified |
|---|---:|---:|---:|
| Linux Go 1.27 amd64 example | 4,053 B | 16,908 B | 12,855 B |
| Linux Go 1.23 CGO example | 572 B | 112,921 B | 112,349 B |
| Windows Go 1.23 example | 352 B | 5,705 B | 5,353 B |
| Darwin Go 1.23 example | 352 B | 10,138 B | 9,786 B |
| analysis-server | 34,734 B | 808,004 B | 773,270 B |
| Docker Compose | 0 | 1,699,596 B | 1,699,596 B |
| Windows Go 1.23 stripped CGO | 0 | 1,287,411 B | 1,287,411 B |

矩阵另覆盖 Linux stripped example、Darwin Go 1.27 PIE CGO、js/wasm 和 wasip1，共 11 个输入。所有输入 SHA256 保持不变，所有输出满足 `attributed + recognized + unclassified = file size`。WASM 没有套用原生 section 地址映射，新增函数引用指标保持未知。

analysis-server 中 `github.com/ZNotify/server/docs` 的 Swagger 模板 **29,287 B** 已进入生产台账：offset **26,866,173**，证据为 `dwarf / reference / declared`。生产映射比原型更保守，不把 runtime.moduledata 引用的 typelinks/itablinks 已识别元数据强行归包，因此不采用原型的 98,222 B 候选量作为生产增益。

受控等体积 fixture 的文件和包变化均为 0，生产 diff 仍找到 `main.alpha` 引用 **+6,144 B** 和 `main.beta` **−6,144 B**。Go 1.23→1.27 example 得到 2,098 条函数变化并显示构建上下文差异。Compose 自比没有变化；35 组重复原始身份明确提示，不进行不确定匹配。

## 使用与复现

```powershell
gsa --coverage-details -o report.html app.exe
gsa --coverage-details -o before.json old.exe
gsa --coverage-details -o after.json new.exe
gsa before.json after.json -f json -o changes.json

go build -mod=readonly -p=2 -o temp/size-iteration/gsa.exe ./cmd/gsa
python experiments/size-attribution/iteration.py --workers 3
```

HTML 查看需要本轮构建的 UI。本地已生成 `temp/size-iteration/gsa-embedded.exe`，通过 Go overlay 嵌入新模板，没有覆盖第一轮实验的旧模板；它生成了 `temp/size-iteration/ui/example.html`。普通非 embed 构建仍使用项目原有 UI 下载/缓存机制。

## 计量与开销边界

物理台账仍是文件总量的依据；函数、共享引用和内联源码视角不能相加为删除收益。普通指针遍历要求 DWARF 和文件内静态初始化数据，stripped 输入不会凭空恢复这些类型信息。

完整 JSON 因增加名称和引用指标而变大：Compose 由 31,654,047 B 增至 39,056,412 B，analysis-server 由 16,632,307 B 增至 20,515,761 B。证据样本只在 details 模式输出，完整物理 regions 仍由原有 details 开关控制。Web 首屏不构造全部函数节点，文件下钻再创建对应函数。

并发实验记录的时间仅用于诊断，不能作为性能基准。外部符号匹配、inline/generic overlay、链接保留 witness 和 CFG 仍保留在实验目录：它们的身份校验、共享计量和启发式边界尚未被推广成默认生产行为。

## 验证记录

- 原生 Go：根包、`internal/...`、`cmd/...` 和三个 Go 实验包测试通过；DWARF 与 knowninfo 的 race 测试通过。
- WASM：entity、marshaler、result 的全部普通测试通过，包含真实 `JSON.stringify` 与原生 JSON 的一致性。Windows Node 的 fuzz seed 目录读取遇到 `O_DIRECTORY` 不支持，因此 WASM 使用 `-run '^Test'`；原生同包测试包含 fuzz seed 回归。
- UI：85 项测试通过；浏览器实测发现并修复 Tooltip 遮挡详情后，相关 10 项测试再次通过。ESLint、TypeScript、WebUI、使用最新 WASM 的 Explorer 构建通过。
- 真实浏览器：实际 embedded CLI 输出的 HTML 可以下钻到 `embed.split`，恢复函数 hash、显示两个 `.rodata` 引用范围并返回文件；没有页面脚本错误。截图和摘要在 `temp/size-iteration/ui/`。
- Go 静态检查：生产包全量 `go vet` 通过，快速 golangci-lint 报告 0 issues，`git diff --check` 通过。完整 golangci-lint 超过设定的 3 分钟仍未退出，在约 4 分钟时终止，未产生诊断，**未记为通过**。

交叉审查还修复了泛型身份碰撞的漏报：一个实例有重复原始名称时，其他具有唯一完整名称的实例仍应正常比较。新增用例先复现漏报，再验证独立实例的 `+20 B` 变化得到保留。
