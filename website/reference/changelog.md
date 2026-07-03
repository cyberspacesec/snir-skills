# 更新日志

<p align="center">📝 snir 版本演进记录。</p>

::: tip 提示
完整 release 历史见 [GitHub Releases](https://github.com/cyberspacesec/snir-skills/releases)。
:::

## v1.x — AI 原生重构

- 🤖 **Skill Bundle 化**：仓库重构为 Anthropic 兼容技能包，`SKILL.md` 为入口
- 📚 **references/ 渐进文档**：代理按需加载任务文档
- 🧪 **evals/**：评估代理能否正确使用 snir 的测试提示
- 🧩 **Go SDK**：类型化 `Client`、Builder 模式 `ScreenshotOption`、共享池函数
- 🌐 **HTTP API**：鉴权、并发限流、批量、健康/统计端点
- 🔌 **CDP Provider**：跨进程共享 Chrome
- 🛡️ **统一 Result 模型**：`schema_version` 标记、全量证据、感知哈希聚类
- 🎭 **浏览器控制**：设备预设、指纹伪装、代理轮换、Cookie 持久化、JS 注入、表单交互
- 🔍 **技术检测**：指纹库识别框架/CMS/CDN
- 📊 **报告**：富 HTML 报告、转换、合并、本地 webserve
- 🗄️ **SQLite 持久化**：GORM、索引字段、会话与标签

## 早期 — go-web-screenshot

项目前身为基础网页截图工具，逐步演进为情报采集子系统并改名 `snir`，定位 AI 优先。

## 发布流程

发布由 [release.yml](https://github.com/cyberspacesec/snir-skills/blob/main/.github/workflows/release.yml) GitHub Action 驱动，goreleaser 构建多平台二进制。见 [release-guide](https://github.com/cyberspacesec/snir-skills/blob/main/docs/release-guide.md)。

## 下一步

- [FAQ](./faq)
- [路线图](../community/roadmap)
- [GitHub Releases](https://github.com/cyberspacesec/snir-skills/releases)
