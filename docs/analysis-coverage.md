# 二进制字节归属：实现、编译器证据与实测

本文记录 2026-09-05 完成的覆盖率扩展。原方案的五个方向均已有实现：物理文件区间台账、ELF 可写数据、外部函数定义、原生字符串模式，以及 WASM 指令与数据映射。额外实现了具名变量静态头、GC stackmap 和经过验证的写屏障路径分析。

## 统计口径

JSON 的 `coverage` 按物理文件偏移统计：

- `attributed`：有包归属的唯一文件字节。
- `recognized`：已识别、尚未归属到包的独立结构，例如文件头、调试信息和符号表。
- `unclassified`：余下未分类的文件字节。
- `shared`：被多个包认领的字节，是 `attributed` 的子集。
- `by_source`：每种证据来源覆盖的唯一字节。来源之间可能重叠，不能相加作为总覆盖量。
- `notes`：分析被关闭、缺少 DWARF/符号或不支持指令集等限制。

前三项互不重叠，严格满足 `attributed + recognized + unclassified == size`。`--coverage-details` 额外输出按偏移排序的 `regions`，包含大小、类别、包归属和证据来源；默认省略这些明细以控制报告体积。

原生虚拟地址通过实际段的文件偏移和文件大小映射；BSS 不计入磁盘字节。WASM 线性地址通过活动数据段映射回文件，多次初始化同一区间时以后写入的数据为准。未映射的文件间隙保留为未分类，不把任意空隙推断成填充。

函数的 pclntab 计量改为真实区间：函数表记录、名称、`_func`、PC 程序、附加 PC/funcdata 记录及可验证的参数/局部变量 GC 位图。共享 PC 程序和位图通过并集去重。解析器支持 Go 1.2、1.16、1.18、1.20+ 的格式标识，识别大小端；越界记录不能贡献覆盖量。JSON 中 `pcln_bytes` 保存精确值，旧数据缺少该字段时仍可读取。

Treemap 保留独立段面积，对共享包面积进行归一化，根对象提示信息显示四项覆盖统计与分析限制。面积分配是展示规则；物理覆盖指标应读取 `coverage`，不能累加各包大小。

## 当前实测

下表为仓库 `scripts/bins` 样本的实际运行结果，单位为字节。每行三类字节之和等于文件大小。

| 样本 | 文件大小 | 已归属 | 独立结构 | 未分类 | 多包共享 |
|---|---:|---:|---:|---:|---:|
| docker-compose-linux-x86_64 / Go 1.21.11 | 63,083,225 | 51,596,241 | 8,189,734 | 3,297,250 | 942,369 |
| analysis-server-linux / Go 1.22.1 | 60,725,080 | 48,370,178 | 11,394,757 | 960,145 | 1,009,137 |
| bin-js-1.27-wasm | 2,644,952 | 2,129,690 | 370,434 | 144,828 | 2,329 |
| bin-wasip1-1.27-wasm | 2,554,276 | 2,055,548 | 356,050 | 142,678 | 2,156 |

同一实现、同一样本，比较默认分析和 `--skip-disasm`，得到指令分析带来的净新增唯一文件字节。静态变量头和编译器元数据在两组中都启用。

| 样本 | 关闭指令分析时已归属 | 启用后净增加 |
|---|---:|---:|
| docker-compose-linux-x86_64 | 50,594,294 | 1,001,947 |
| analysis-server-linux | 47,944,883 | 425,295 |
| bin-js-1.27-wasm | 2,098,739 | 30,951 |
| bin-wasip1-1.27-wasm | 2,025,375 | 30,173 |

Docker Compose 另有 137,812 字节以 `static_header` 为证据，analysis-server 为 28,442 字节。这是来源覆盖量，不能未经消重就再加到上述增量中。

旧方案记录的已归属大小依次为 50,407,268、46,589,675、1,968,736、1,899,637 字节，但包含 pclntab 估算。新旧口径不同，不能把两组差值称为严格的覆盖提升。此次默认/关闭指令分析对照使用同一物理台账，可以直接比较。

## 字符串的实际编译模式

证据来自本机 Go 1.27 源码和真实交叉编译产物。永久回归用例位于 `internal/disasm/compiled_test.go` 和 `wasm_compiled_test.go`：编译同一份小程序，并核验提取出的实际字节，而不是只核验候选地址数量。

| Go 源码形态 | 编译后证据 | 实现策略 |
|---|---|---|
| 返回字符串字面量 | amd64 RIP 相对 `LEAQ` + 长度立即数；ARM64 `ADRP`/`ADD` 或 `ADR` + 长度 | 按内部 ABI 寄存器顺序配对指针/长度 |
| 多个参数中混有字符串 | 字符串占两个连续整数 ABI 槽，前面可能已有整数参数 | 跟踪合法 ABI 槽，不把任意相邻机器寄存器拼成字符串 |
| 返回全局字符串变量 | 从相邻地址加载指针、长度；ARM64 也会使用成对加载 | 验证静态头实际内容，再跟随指向的数据 |
| 字符串装入 `any` | 类型指针 + 静态字符串头地址 | 检查 `internal/abi.Type` 的大小、指针字节数和 String kind，再读取头 |
| `[2]string` 局部聚合初始化 | amd64 `MOVUPS` 复制静态头；ARM64 `FLDPQ` 复制两个向量 | 从静态聚合数据恢复头记录 |
| 向 `*string` 写入常量 | 长度写入、写屏障条件分支、`runtime.gcWriteBarrierN`、指针写入 | 只穿过经过核验的屏障分支，保留支配两条路径的字段存储 |
| WASM 字面量与全局变量 | `i64.const`、地址/字段加载、局部变量转存与线性内存存储 | 在有界符号栈和局部状态中恢复常量与配对头 |
| WASM 聚合初始化 | `memory.copy`，例如从静态地址复制 32 字节字符串头 | 归属被复制的真实字节，并检查其中的字符串头 |
| 具名静态字符串/切片变量 | 符号或 DWARF 给出 2/3 个机器字大小的记录 | 检查指针/长度以及切片 `cap == len`，不依赖函数反汇编 |
| 短字符串比较 | `s == "Z9q!"` 可变为长度比较与 `CMPL (AX), $0x2171395a` | 字面量直接编码在指令中，已计入函数代码，不额外增加 rodata |

短字符串比较说明：源码中有字符串，不代表二进制一定有独立字符串数据。字符串与只读 `[]byte` 转换也可能复用底层数据；同一地址的多个引用只增加共享关系，不重复增加唯一字节。

当前有界跟踪保留最多 12 条指令年龄的证据，遇到普通调用、分支、不明寄存器写入或可能别名的存储会失效。32 位寄存器写入会正确截断，未知高位的 ARM64 `MOVK` 不产生完整地址。屏障特例要求前向分支跨度不超过 128 字节、调用地址精确匹配 `runtime.gcWriteBarrier1..8`，路径内只允许预期 GC 缓冲区操作；任意调用或未知分支仍终止跟踪。

`memory.copy` 可以归属非 UTF-8 字节，因为复制指令已明确指出源区间；对复制数据中的头扫描限制在前 64 KiB。字符串候选仍要求落在文件支持的数据区间、长度有界并通过内容校验。ABI 和头形态提供结构证据，但尚不能保证每条启发式归属都正确；本次没有声称已经得到真实大型程序的全面误报率。

## 其他已落地的定位方法

1. **ELF 可写数据符号。** 现在接纳 `SHF_ALLOC | SHF_WRITE` 的有效文件数据符号，恢复去掉 DWARF 后的全局变量及只有数据的包。BSS 仍不算物理字节。Go 1.27 的 `.go.type`、`.go.func` 等数据段也被正确分类，使类型和静态头能解析到这些段。
2. **普通文本符号与 DWARF 外部定义。** `DW_AT_external` 不再被当成声明；仍排除真正的 declaration。非 Go 文本符号可补充 pclntab 外的函数，函数所有不连续 ranges 分别反汇编，避免跨段拼接。
3. **编译器元数据的实际引用。** 用 `_func` 的 PC/funcdata 偏移定位元数据，并跟随前两类 GC stackmap。真实字节边界替代旧的每函数估计，共享记录保留多个归属者。
4. **WASM 双重视图。** watgo IR 提供指令语义，原始模块提供编码偏移。函数局部变量声明、长度编码、段结构和 `name` 元数据作为独立结构保留，活动数据映射排除零初始化内存。
5. **合法重复自定义段。** 相同名称的 WASM 自定义段使用独立键保存，名称为 `code` 的自定义段不会冒充标准代码段；未知自定义 payload 保留为未分类。声明超过所属段长度的名字会在分配前拒绝。

## 编译器源码依据

下列路径均相对于 Go 源码根目录；本次核对版本为本机 Go 1.27.0。分析依据是源码与本次编译产物，内部 ABI 不是跨版本稳定接口。

| 源码 | 核对内容 |
|---|---|
| `src/cmd/compile/internal/staticdata/data.go`，`StringSym` / `InitConst` | 字符串内容符号与指针、长度静态初始化 |
| `src/cmd/compile/abi-internal.md` | 字符串、接口和聚合值在内部调用 ABI 中的分解 |
| `src/internal/abi/abi_amd64.go`、`abi_arm64.go` | 架构整数参数寄存器数量 |
| `src/internal/abi/type.go` | 内建 String 类型布局与 kind 标识 |
| `src/cmd/compile/internal/ssa/_gen/AMD64.rules`、`ARM64.rules` | 地址计算、加载、存储及聚合复制的降级规则 |
| `src/cmd/compile/internal/ssa/_gen/AMD64Ops.go`、`ARM64Ops.go` | `LoweredWB` 寄存器约定 |
| `src/runtime/asm_amd64.s`、`asm_arm64.s` | GC 写屏障助手及寄存器保存约定 |
| `src/cmd/compile/internal/ssa/_gen/Wasm.rules`、`src/cmd/compile/internal/wasm/ssa.go` | WASM 常量、地址、数据加载与内存复制 |
| `src/cmd/compile/internal/walk/compare.go` | 字符串比较展开，短常量可进入比较指令 |
| `src/cmd/link/internal/ld/pcln.go`、`src/runtime/runtime2.go` | pclntab 与 `_func` 编码 |
| `src/runtime/symtab.go`、`stkframe.go` | funcdata 与 GC stackmap 的读取方式 |

## 复现与验证

使用仓库要求的 Go 工具链和构建脚本准备 fixture、UI 后：

```sh
go test -tags embed ./...
go test -race ./internal/knowninfo ./internal/disasm ./internal/entity ./internal/wrapper
go test ./internal/disasm -run 'TestCompilerGenerated' -v
go run ./cmd/gsa scripts/bins/docker-compose-linux-x86_64 --format json --compact --output full.json
go run ./cmd/gsa scripts/bins/docker-compose-linux-x86_64 --format json --compact --skip-disasm --output skip.json
go run ./cmd/gsa scripts/bins/bin-js-1.27-wasm --format json --coverage-details --output wasm-details.json
```

支持旧版构建环境时需按仓库配置设置 `GOEXPERIMENT=jsonv2`。本地忽略的 vendor 副本不完整时用 `-mod=readonly`，CI 使用下载的模块依赖。比较 `full.json` 和 `skip.json` 的 `coverage.attributed`，不要比较包大小之和。

本次已完成完整 Go 测试、含覆盖率的干净源码快照测试、竞态检查，以及 ELF/PE/Mach-O、PIE/CGO、js/wasm、wasip1/wasm 实际样本分析。新增回归覆盖真实编译字符串模式、外部定义、拆分 ranges、无 DWARF 可写变量、物理分区不变式、共享元数据、WASM 重复段/覆盖映射和截断输入。前端验证包括 schema、报告树、上传与 WASM worker 流程。

原生提取保持固定窗口，不再保存完整函数的指令切片。大型样本单次日志中，Docker Compose 的指令分析为 265 ms，analysis-server 为 155 ms；这是新增规则下的一次观测，不是跨机器基准。旧规则的既有基准曾将分配从约 71.0 MB/op 降到 1.73 MB/op，但规则已经扩展，不能把旧数字当作当前实现的性能测量。

浏览器端到端测试还发现了 WASM 稀疏数据映射的分配问题：逐段重建并排序完整列表会耗尽浏览器内存。改为二分定位、原地替换后，`BenchmarkWasmRawLayout` 在同一个 Go 1.27 js/wasm 样本上，单次本地对照从 2.763 秒、18,237,178,088 B/op 降至 10.30 毫秒、8,887,368 B/op。原始文件映射结果不变；真实浏览器重新上传同一文件后约 1 秒完成主要分析步骤并正常显示报告。

Go 1.25/1.26 的 WASM 集成测试又暴露了汇总阶段的全表扫描：每条元数据引用遍历全部活动数据段，且函数元数据大小计算在区间间隙中逐段前进。改用二分定位后，同一 Go 1.25 样本在 CI 相同的 `-cover -covermode=atomic -tags embed,profiler` 构建下，本地耗时从 10.999 秒降至 0.487 秒。规范化无序集合后，修复前后完整 JSON 报告一致。保留 `BenchmarkAnalyzeWasm` 覆盖实际 Go 1.25/1.27 样本的整条分析路径。

## 明确保留的边界

动态拼接、解密、运行时构造的字符串无法从静态文件完整恢复。当前没有通用跨函数常量传播，也不解释任意控制流或所有向量变体；只处理已验证模式。WASM 扩展常量表达式中超出现有解码器支持的活动段偏移会报错，不能猜测映射。

目前未逐一解析所有 Go funcdata、链接重定位关系、DWARF 位置表达式和任意聚合变量类型。这些仍可扩大后续覆盖，但本轮没有把未知元数据整段冒充包归属。`name` 等独立结构已能准确计量，进一步按函数拆分归属属于粒度改善，不会增加物理文件的已识别总量。
