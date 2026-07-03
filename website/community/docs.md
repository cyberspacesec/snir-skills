# 文档贡献

<p align="center">📝 如何为 snir 文档站贡献内容。</p>

文档站位于 `website/`，基于 [VitePress](https://vitepress.dev/) 渲染，由 GitHub Actions 构建并部署到 GitHub Pages。

## 目录结构

```
website/
├── .vitepress/config.ts   # 配置（导航/侧边栏/主题）
├── public/                # 静态资源（logo）
├── index.md               # 首页
├── guide/                 # 指南
├── cli/                   # CLI 命令文档
├── sdk/                   # Go SDK 文档
├── api/                   # HTTP API 文档
├── internals/             # 内部模块文档
├── advanced/              # 进阶与运维
├── reference/             # 参考手册
└── community/             # 社区
```

## 本地预览

```bash
cd website
npm install
npm run docs:dev
# 访问 http://localhost:5173
```

## 新增文档

1. 在对应目录创建 `.md` 文件
2. 在 `.vitepress/config.ts` 的 `sidebar` 中添加条目
3. 文档头部用 `<p align="center">...</p>` 引导句
4. 多用 emoji、表格、mermaid 图、代码块
5. 与相关文档互链（相对链接）

## 风格约定

- **语言**：简体中文（代码/标识符/报错原文保持原样）
- **标题**：用 `#` 层级，每页一个 `#`
- **代码块**：标注语言（```bash / ```go / ```json / ```mermaid）
- **链接**：相对路径，如 `../cli/scan`、`./overview`
- **图标**：适度使用 emoji 增强可读性

## 部署

推送到 `main` 分支且 `website/` 有变更时，[docs.yml](https://github.com/cyberspacesec/snir-skills/blob/main/.github/workflows/docs.yml) 自动构建并部署到 GitHub Pages。

## 检查清单

- [ ] 文档基于真实代码，无臆造
- [ ] 侧边栏已添加条目
- [ ] 链接有效
- [ ] 本地 `npm run docs:build` 通过

## 下一步

- [贡献指南](./contributing)
- [VitePress 文档](https://vitepress.dev/)
