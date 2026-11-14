# 二进制体积定位：并行实验结果

本文记录第一轮原型实验；后续生产接入、真实样本回归与使用方式见[持续迭代记录](size-attribution-iteration.md)。

2026-11-13，在当前仓库工作区、Windows/amd64、Go 1.27.0 上完成。四组独立原型并行运行，覆盖对象证据、函数级 diff、链接保留路径、外部符号、内联与泛型、静态对象和有限 CFG。没有修改生产分析器、Web UI、依赖或原始样本。原型和受控夹具位于 [`experiments/size-attribution`](../experiments/size-attribution/README.md)，生成产物位于被忽略的 `temp/size-experiments/`。

## 判断

| 方向 | 本轮证据 | 建议 |
|---|---|---|
| 对象证据与函数级 diff | 文件大小不变，仍精确找到 +6,144 / -6,144 B 的相消变化；真实 GSA feature A/B 定位 282,356 B 嵌入资源 | 优先接入报告与 UI |
| 匹配的外部符号 | 两个 CGO example 恢复 933,530 / 933,572 B；analysis-server 的受控 stripped 副本恢复 934,910 B 代码归属 | 对 CGO 很有价值，必须严格校验同产物身份 |
| 内联与泛型视图 | analysis-server 解释 445,429 B 已有内联代码；当前 GSA 解释 2,545,974 B；Compose 符号恢复 298 个泛型声明组 | 作为已有字节的另一观察视角，不增加文件总量 |
| 静态指针对象图 | analysis-server 找到 29,287 B Swagger 模板，合计补出 35,358 B 原未分类字节的引用证据 | 优先于通用 CFG，保留定义/引用区别 |
| 链接保留路径 | 当前 GSA 得到 123,741 条 `-dumpdep` witness 边 | 能回答一条保留原因，不能计算完整 retained size |
| 有限 CFG | 已知正例补出 16 B；Compose 仅新增 192 B 启发式候选，analysis-server 0 B | 暂缓通用化，按大块残留区域定向投入 |

所有“恢复/新增”都是分析信息或归属证据，不是二进制变小。没有把 package/object/reference 大小当删除收益。

## 样本和控制

使用 `scripts/bins` 中的 example 与真实 integration 输入：Linux ELF 的 Go 1.19/1.21/1.23/1.27、strip/CGO 对照，Windows PE 的 Go 1.23 普通与 stripped CGO，Darwin Mach-O 的 Go 1.23 普通和 Go 1.27 PIE CGO，以及 Go 1.27 js/wasm。真实产物包括 Docker Compose、analysis-server 和本仓新构建的 GSA。

另外编译了可独立验证的优化 fixture：共享调用者、字符串大小相消、跨基本块字符串、普通指针/静态接口、内联与泛型 shape 共享。对已有二进制只读；分析过程中没有执行输入程序，strip 操作只针对输出目录里的新副本。

原始样本的 attributed 基线：Compose **60,871,047 / 63,083,225 B**，analysis-server **52,303,047 / 60,725,080 B**，Windows Go 1.23 stripped CGO **2,498,317 / 3,909,632 B**。baseline 的 attributed、recognized、unclassified 守恒。

输入和工具的 SHA256、构建配置、完整区间或对象结果、失败尝试以及声明的限制，都保留在各组 JSON 中。不同组重复使用相同样本，是交叉验证，不是不同样本数。

## 主要发现

### 对象证据与变化

原型直接保存生产分析器现有的 function/symbol 关联，额外输出对象证据样例和物理范围。Compose 有 171,096 个对象，其中 16,797 个关联静态数据引用；analysis-server 有 100,107 个对象，其中 9,381 个关联引用。八个样本的 self-diff 均为零。

受控 example 的两产物均为 1,873,920 B。alpha 的字符串增长 6,144 B、beta 缩小 6,144 B，函数代码与 pcln 不变；对象 diff 正确显示这两条变化。原包级总量会隐藏这种相消。

真实 GSA 默认构建与 `-tags=embed` 构建相比，报告定位出页面资源 **+282,356 B**，并独立核验目标文件对应物理字节与 HTML 完全相同。但最终文件 **-411,136 B**；报告同时显示下载/缓存相关函数移除，并提示 build tags 变化。不能把嵌入资源大小直接当整体增长，也不能将不同产品行为称为无条件优化。

限制：生产模型会规范化函数名，本原型对重名身份聚合，尚不能解决全部闭包/生成函数匹配；指令 PC 和完整变量字段路径也未全部穿透。WASM 的对象物理映射仍未实现，零值明确表示不可用。符号附属记录只存在于既有台账，不应遗漏后再把对象和当文件总量。

Compose 的完整对象 JSON 约 108 MB，analysis-server 约 61 MB。HTML 是最多 300 行的独立预览，真实浏览器验证了筛选与证据展开。生产集成应使用紧凑对象表和按需加载。详见 [report findings](../experiments/size-attribution/report/findings.md)。

### 外部符号与保留路径

匹配校验覆盖所有 allocated section 内容/地址/布局、加载映射和权限、架构、入口；允许不参与符号计量的尾部 padding 差异。六个同产物 strip 副本通过，四个仓库里分别构建的 strip/non-strip 配对全部被拒绝，错 Go 版本和代码/只读数据/架构/权限突变也被拒绝。

| 同产物副本实验 | 额外可归属代码 / B |
|---|---:|
| Go 1.23 pure Go example | 0 |
| Go 1.23 CGO example | 933,530 |
| Go 1.27 CGO example | 933,572 |
| Docker Compose | 0 |
| analysis-server | 934,910 |
| 当前 GSA Linux 构建 | 2 |

这是用原始 ELF 作为对应 stripped 副本的 sidecar，不是找回缺失的历史生产符号。暂未实现 PDB/dSYM 或 detached debug 文件发现协议。LLVM objcopy 对 Compose 的一次损坏输出被拒绝，GNU fallback 成功；失败日志保留。

两调用者 fixture 的反汇编有两条 `CALL shared`，`-dumpdep` 只有一条父边，实证其为首次保留路径。因此用该输出直接计算 dominator 或删包节省量会出错。详见 [buildassist findings](../experiments/size-attribution/buildassist/findings.md)。

### 内联与泛型

分析器区分具体宿主函数与内联源码来源，使用物理并集，另外构造最深来源的互斥投影。所有样本均满足投影守恒。

| 产物 | 有范围的内联实例 | 内联物理并集 / B | 泛型声明组 / 实际函数体 |
|---|---:|---:|---:|
| Go 1.27 Darwin PIE CGO example | 15,531 | 627,754 | 92 / 162 |
| analysis-server | 10,421 | 445,429 | 162 / 723 |
| 当前 GSA Windows 构建 | 55,780 | 2,545,974 | 268 / 983 |
| Docker Compose | DWARF 不可用 | 不可用 | 298 / 657（ELF symbol） |

简单 ELF/PE/Mach-O examples 以 `-gcflags="all=-N -l"` 构建，内联结果为零符合输入；优化 fixture 和真实项目验证了非零结果。三平台 fixture 用三个具体类型调用同一泛型，实际只有 **2 个共享 shape body、50 B 代码**，而有 **3 个字典**。不能按字典个数算代码副本。详见 [inline findings](../experiments/size-attribution/inline/findings.md)。

### 静态对象与 CFG

普通类型化指针遍历在 analysis-server 找到 **98,222 B** 尚未归属的静态引用，其中：

- **35,358 B** 来自原 unclassified；包含 **29,287 B SwaggerTemplate**，物理偏移 26,866,173。
- **62,864 B** 原本已经 recognized，是 firstmoduledata 引用的 typelinks/itablinks。它们不能被算作未知区间减少，也不代表 runtime 独占。

Go 1.27 example 另有 4,053 B 新 typed 引用与 232 B interface 候选。无 DWARF 的输入没有静态根，该路线明确不可用。

有限 CFG 的分支、汇合与寄存器清理在受控 assembly fixture 上找到 baseline 漏掉的 16 B。真实 Compose 仅新增两个 64 B base64、两个 32 B base32 候选，共 **192 B**；analysis-server 候选全部已归属。ABI 指针/长度配对仍是启发式，不混入已验证归属。

原型只支持 ELF64 amd64 ET_EXEC，不处理 PIE 重定位、运行时初始化和通用别名。共享/循环对象、partial-register 写入、隐式寄存器写入、接口 direct-value 边界都有针对回归。详见 [staticreach findings](../experiments/size-attribution/staticreach/findings.md)。

## 验收与复现

四组总入口并行执行均 exit 0；另完成真实项目 embed A/B。20 个 Go 实验测试（含真实编译和子用例）、10 个 Python 匹配/恢复验收测试通过；现有 `internal/entity`、`internal/knowninfo`、`internal/diff` 测试通过。未把这轮并发耗时当性能基准，未宣称完整生产回归或实际删除功能后的语义等价。

```powershell
python experiments/size-attribution/run.py --workers 4
```

总报告为 `temp/size-experiments/summary.json`；各组的 `findings.md` 保留可长期阅读的结果。完整运行说明见 [实验 README](../experiments/size-attribution/README.md)。
