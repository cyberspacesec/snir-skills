# snir report — 报告操作

> **渐进式披露**：[快速上手](#快速上手) → [常用选项](#常用选项) → [高级选项](#高级选项) → [完整参数参考](#完整参数参考)

---

## 快速上手

```bash
# 生成 HTML 报告
./snir report html --input results.jsonl --output report.html

# 合并多个 JSONL 结果文件
./snir report merge --files a.jsonl --files b.jsonl --output merged.jsonl

# 转换格式
./snir report convert --from results.jsonl --to results.csv
```

---

## 子命令概览

| 子命令 | 用法 | 说明 |
|--------|------|------|
| `html` | `snir report html --input <jsonl>` | 从 JSONL 生成 HTML 报告 |
| `merge` | `snir report merge --files <...> --output <path>` | 合并多个 JSONL 文件 |
| `convert` | `snir report convert --from <src> --to <dst>` | 转换报告格式 |

---

## snir report html

### 快速上手

```bash
# 从 JSONL 生成 HTML 报告
./snir report html --input results.jsonl
```

### 常用选项

```bash
# 指定输入和输出路径
./snir report html --input results.jsonl --output my-report.html

# 使用父命令的持久标志
./snir report html --input results.jsonl --output-path ./reports
```

| 标志 | 默认值 | 说明 | 必填 |
|------|--------|------|------|
| `--input` | `""` | JSONL 格式的结果文件路径 | ✅ 是 |
| `--output` | `report.html` | HTML 报告输出路径 | 否 |

### 完整参数参考

| 标志 | 类型 | 默认值 | 说明 | 必填 |
|------|------|--------|------|------|
| `--input` | string | `""` | JSONL 格式的结果文件路径 | ✅ 是 |
| `--output` | string | `report.html` | HTML 报告输出路径 | 否 |

继承 `report` 父命令的持久标志：

| 标志 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `--output-path` | string | `reports` | 报告输出路径 |
| `--format` | string | `html` | 报告格式（`html`/`json`/`csv`） |
| `--host` | string | `127.0.0.1` | Web 服务器主机地址 |
| `--port` | int | `8080` | Web 服务器端口 |

---

## snir report merge

### 快速上手

```bash
# 合并指定文件
./snir report merge --files a.jsonl --files b.jsonl --output merged.jsonl

# 合并整个目录的 JSONL 文件
./snir report merge --path ./results/ --output merged.jsonl
```

### 常用选项

```bash
# 指定多个源文件
./snir report merge \
  --files scan1.jsonl \
  --files scan2.jsonl \
  --files scan3.jsonl \
  --output all-results.jsonl

# 从目录合并
./snir report merge --path ./all-scans/ --output merged.jsonl
```

| 标志 | 默认值 | 说明 | 必填 |
|------|--------|------|------|
| `--files` | `[]` | 源文件路径列表（可多次使用） | ⚠️ 与 `--path` 二选一 |
| `--path` | `""` | 包含源文件的目录路径 | ⚠️ 与 `--files` 二选一 |
| `--output` | `""` | 输出文件路径 | ✅ 是 |

> **注意**：`--files` 和 `--path` 必须指定其中一个，不能都不指定。

### 完整参数参考

| 标志 | 类型 | 默认值 | 说明 | 必填 |
|------|------|--------|------|------|
| `--files` | stringSlice | `[]` | 源文件路径列表（可多次使用） | ⚠️ 二选一 |
| `--path` | string | `""` | 包含源文件的目录路径 | ⚠️ 二选一 |
| `--output` | string | `""` | 输出文件路径 | ✅ 是 |

继承 `report` 父命令的持久标志（同上）。

---

## snir report convert

### 快速上手

```bash
# JSONL → CSV
./snir report convert --from results.jsonl --to results.csv

# JSONL → JSON
./snir report convert --from results.jsonl --to results.json
```

### 常用选项

```bash
# 转换格式
./snir report convert --from results.jsonl --to results.csv
```

| 标志 | 默认值 | 说明 | 必填 |
|------|--------|------|------|
| `--from` | `""` | 源文件路径 | ✅ 是 |
| `--to` | `""` | 目标文件路径 | ✅ 是 |

> **格式自动推断**：输出格式由 `--to` 文件的扩展名决定（`.csv`、`.json`、`.jsonl`）。

### 完整参数参考

| 标志 | 类型 | 默认值 | 说明 | 必填 |
|------|------|--------|------|------|
| `--from` | string | `""` | 源文件路径 | ✅ 是 |
| `--to` | string | `""` | 目标文件路径 | ✅ 是 |

继承 `report` 父命令的持久标志（同上）。

---

## report 父命令持久标志

以下标志被 `html`、`merge`、`convert` 三个子命令共同继承：

| 标志 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `--output-path` | string | `reports` | 报告输出路径 |
| `--format` | string | `html` | 报告格式（`html`/`json`/`csv`） |
| `--host` | string | `127.0.0.1` | Web 服务器主机地址 |
| `--port` | int | `8080` | Web 服务器端口 |

---

## 实战组合示例

```bash
# 典型工作流：扫描 → 合并 → 生成 HTML 报告
./snir scan cidr 192.168.1.0/24 --write-jsonl --jsonl-file subnet1.jsonl
./snir scan file -f urls.txt --write-jsonl --jsonl-file subnet2.jsonl
./snir report merge --files subnet1.jsonl --files subnet2.jsonl --output all.jsonl
./snir report html --input all.jsonl --output final-report.html

# 扫描 → 转换为 CSV → 分析
./snir scan file -f targets.txt --write-jsonl --jsonl-file results.jsonl
./snir report convert --from results.jsonl --to results.csv
```