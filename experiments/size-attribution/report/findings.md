# 对象证据与函数级差异实验

2026-11-13，Windows/amd64、Go 1.27.0。原型复用生产 `internal.Analyze` 和 `OwnAddressValues`，没有改生产 schema 或物理台账。并行实验期间不报告性能优劣。

## 实现和复现

```powershell
python experiments/size-attribution/report/run.py
go test -mod=readonly -p=2 ./experiments/size-attribution/report
```

产物在 `temp/size-experiments/report/`，包括每个样本的 JSON、HTML、self-diff，跨 Go 版本 diff，以及受控 fixture 的 diff。

- 对象以包、生产模型提供的函数名/receiver/type 或符号名标识，不以地址匹配。模型会规范化部分链接符号名称；重名身份聚合，不猜测一对一对应。生产实现应补存原始链接符号并标记歧义，不能认为本实验已解决所有闭包/生成函数的匹配。
- 保留原有函数/符号关联，显示代码、pcln、引用数据和物理区间并集；默认每对象最多三条证据示例，并输出实际证据条数。
- 记录输入 SHA256、Go build info/模块/设置、分析器二进制 SHA256 与版本控制信息。
- HTML 是最多 300 行的探索预览，JSON 保留全部对象。筛选只作用于显示的行，不伪装成全量搜索。

## 实测

| 样本 | 函数 | 全部对象 | 关联静态引用的对象 | 对象物理并集 / B |
|---|---:|---:|---:|---:|
| Linux Go 1.23 | 2,515 | 4,802 | 1,070 | 1,273,223 |
| Linux Go 1.27 | 2,733 | 5,287 | 1,167 | 1,413,114 |
| Linux Go 1.23 strip | 2,378 | 3,945 | 563 | 1,231,695 |
| Windows Go 1.23 strip CGO | 3,845 | 6,805 | 935 | 2,498,317 |
| Darwin Go 1.27 PIE CGO | 6,871 | 11,927 | 1,060 | 4,147,163 |
| js/wasm Go 1.27 | 2,350 | 3,951 | 未映射 | 未映射 |
| Docker Compose | 74,944 | 171,096 | 16,797 | 53,539,591 |
| analysis-server | 43,314 | 100,107 | 9,381 | 48,671,415 |

八个输入 self-diff 都为零变化。Go 1.23/1.27 example 对比有 3,450 个对象发生添加、删除或分项变化，2,734 个对象不变，文件增长 202,246 B；报告明确提示目标 Go 版本变化，不能视为源码变更归因。

受控 fixture 同一源文件路径、工具链和构建选项，交换 `alpha`/`beta` 字符串长度。两产物均为 1,873,920 B，文件及已归属总大小不变。对象 diff 精确报告 alpha 的引用数据 +6,144 B、beta -6,144 B，代码和 pcln 不变。自动断言验证这两个差异；真实浏览器中验证了筛选和 Compose 对象证据展开。

Compose 可展开 `schema.Schema.data` 的 29,499 B，显示 `static_header / reference / heuristic` 和物理范围，而不是把它改称为精确的定义证据。

## 真实项目功能 A/B

`feature.py` 用相同源代码、工具链和平台分别构建当前 GSA 的默认版与 `-tags=embed` 版。使用已存在的 `internal/webui/index.html`，其 SHA256 为 `a503040dccf12238b182b255811e90560045725015742d4aa19e59aff5beede3`、大小 282,356 B。报告识别出 `webui.tmpl.string` 新增的 **282,356 B**；独立读取产物偏移 **13,122,490** 的相同长度字节，与 HTML 文件逐字节一致。

完整文件变化却为 **-411,136 B**，并有 2,179 个对象发生新增、删除或分项变化。报告可看到 `download`、`readTemplate`、`getCacheFilePath` 等函数移除；由于对象范围可共享，不能把这些行相加声称精确解释了全部文件差值。构建 manifest 正确标出 `-tags` 变化。这是真实项目的 feature A/B，不能视为默认与 embed 两种产品行为完全等价的优化建议。

结果是 `temp/size-experiments/report/gsa-feature-diff.html`、`gsa-feature-diff.json`、`feature-summary.json`。`run.py` 自动运行该实验；也可在已有报告工具上单独运行 `python experiments/size-attribution/report/feature.py`。本次没有执行两份可执行文件。

## 边界与成本

对象级并集不包含目前只在物理台账中保存的符号表附属记录，所以 Compose 的对象并集比总 attributed 少 7,331,456 B。不能累加对象大小替代文件计量，也不能将引用量当成删除收益。源码字段路径、指令 PC 和完整保留图尚未穿透生产 API，本实验如实保留缺失。

WASM 的函数/数据编码映射未在这个投影原型实现；JSON 中 physical_bytes 为 0 表示未映射，不能解读为实际没有磁盘占用。已有生产 coverage 和函数指标仍可查看。

即使只保留三条证据样例，Compose JSON 约 108 MB，analysis-server 约 61 MB。因此生产 UI 应采用分包/分对象加载或紧凑引用表，而不把这个完整 JSON 一次性塞进页面。HTML 预览保持轻量，但还不是生产 Web 功能。

针对测试覆盖 BSS/虚拟区间排除、文件映射裁剪、重叠证据去重、地址变化不影响身份、相消变化、分项变化、构建上下文提示和 HTML 转义。
