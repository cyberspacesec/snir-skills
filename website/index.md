---
layout: home

hero:
  name: snir
  text: AI 原生网页情报采集
  tagline: 🖥️ 基于 Chrome DevTools Protocol 的截图与 Web 情报子系统 — 让 AI 代理与自动化系统拥有浏览器级取证能力
  image:
    src: /logo.svg
    alt: snir
  actions:
    - theme: brand
      text: 🚀 快速开始
      link: /guide/quick-start
    - theme: alt
      text: 📖 了解 snir
      link: /guide/what-is-snir
    - theme: alt
      text: 🎮 GitHub
      link: https://github.com/cyberspacesec/snir-skills

features:
  - icon: 📸
    title: 多形态截图
    details: 视口截图、全页面、CSS 选择器、XPath 元素截图，支持 PNG/JPEG，可输出到文件或内存字节。
  - icon: 🔍
    title: 页面情报
    details: 采集 HTML 源码、HTTP 头、Cookie、控制台日志、网络请求、TLS 信息、最终 URL 与状态码。
  - icon: 🧩
    title: Go SDK
    details: 类型化客户端、流式批量、共享浏览器池、Builder 模式选项，让 Go 应用一行集成。
  - icon: 🌐
    title: HTTP API
    details: 语言中立的工具端点，支持鉴权、并发限流、批量采集与流式结果。
  - icon: 🔌
    title: 共享 CDP Provider
    details: 多进程代理与 worker 复用同一个 Chrome/CDP 提供者，降低资源占用。
  - icon: 🎭
    title: 浏览器控制
    details: 设备模拟、浏览器指纹伪装、代理轮换、Cookie 持久化、JS 注入、表单交互。
  - icon: 🛡️
    title: 证据与持久化
    details: JSONL/CSV/SQLite 多格式输出，感知哈希聚类，技术栈识别，可生成富 HTML 报告。
  - icon: 🤖
    title: AI 优先
    details: 作为 Anthropic 兼容 Skill Bundle 提供，代理可自发现入口、安装二进制、选择集成模式。
---
