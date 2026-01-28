# go-snir 代码重构计划 (更新版)

## 目标

- 重构优化代码结构，减小单个文件的大小
- 提高代码可维护性和可读性
- 更合理地组织代码模块
- 避免导入循环问题

## 当前问题文件

根据文件行数分析，以下文件需要重点重构：

1. `pkg/api/handlers.go` (587行)
2. `pkg/runner/chromedp.go` (486行) 
3. `pkg/report/html.go` (303行)
4. `pkg/report/server.go` (230行)
5. `pkg/runner/blacklist.go` (243行)
6. `pkg/runner/writer.go` (239行)

## 重构计划

### 重要注意事项：避免导入循环

在进行模块拆分时，必须小心处理包的依赖关系，避免形成导入循环。比如:
- `pkg/api/handlers` 不能导入 `pkg/api`
- 子包不能依赖父包

### 1. pkg/api 模块重构

将大文件拆分为更小的模块，但使用内部函数而不是创建新包：

```
pkg/api/
├── handlers.go      # 主要处理函数，分成多个函数
├── screenshot.go    # 截图相关处理函数
├── batch.go         # 批量处理功能 
├── list.go          # 列表相关功能
├── middleware.go    # 中间件
├── server.go        # 服务器配置
├── types.go         # 类型定义
└── helpers.go       # 辅助函数
```

### 2. pkg/runner 模块重构

将 `chromedp.go` 拆分为多个逻辑相关的文件：

```
pkg/runner/
├── chrome.go       # Chrome基础实现
├── fingerprint.go  # 指纹伪装
├── screenshot.go   # 截图操作
├── network.go      # 网络请求处理
├── actions.go      # 浏览器交互
├── blacklist.go    # 黑名单核心逻辑
├── writer.go       # 写入器接口与基础实现
├── jsonl.go        # JSONL输出
├── csv.go          # CSV输出
├── options.go      # 选项配置
└── runner.go       # 核心运行器
```

### 3. pkg/report 模块重构

将 `html.go` 和 `server.go` 拆分为更小的组件：

```
pkg/report/
├── template.go     # 模板定义
├── renderer.go     # 模板渲染
├── routes.go       # 服务器路由
└── handlers.go     # 请求处理
```

### 4. pkg/scan 重构

合理拆分 `scan.go`：

```
pkg/scan/
├── single.go       # 单URL扫描
├── multi.go        # 多URL扫描
├── file.go         # 文件扫描
└── cidr.go         # 网段扫描
```

## 重构步骤

1. 创建新的文件结构
2. 将大文件的功能按逻辑拆分成多个小文件
3. 确保没有导入循环
4. 测试重构后的代码功能
5. 优化新文件之间的接口

## 注意事项

- 保持对外接口一致，不影响现有功能
- 确保重构过程中不引入新问题
- 添加必要的注释说明各模块功能
- 确保拆分后的文件命名合理，反映其功能 