# snir webserve — Web 查看服务器

> **渐进式披露**：[快速上手](#快速上手) → [完整参数参考](#完整参数参考)

---

## 快速上手

```bash
# 启动 Web 服务器（默认监听 0.0.0.0:8080）
./snir webserve

# 也可以使用别名
./snir serve

# 指定端口
./snir serve --port 9090
```

启动后访问 `http://localhost:8080` 即可在浏览器中查看截图结果。

---

## 常用选项

### 监听配置

```bash
# 指定监听地址和端口
./snir webserve --host 127.0.0.1 --port 9090

# 仅本地访问
./snir serve --host 127.0.0.1 --port 8080
```

| 标志 | 默认值 | 说明 |
|------|--------|------|
| `--host` | `0.0.0.0` | Web 服务器监听地址 |
| `--port` | `8080` | Web 服务器监听端口 |

---

## 完整参数参考

| 标志 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `--host` | string | `0.0.0.0` | Web 服务器监听地址 |
| `--port` | int | `8080` | Web 服务器监听端口 |

> **提示**：`webserve` 是主命令名，`serve` 是别名，两者完全等价。

---

## 实战组合示例

```bash
# 扫描后启动 Web 查看服务
./snir scan file -f urls.txt --screenshot-path ./screenshots
./snir serve --port 8080

# 本地安全访问
./snir serve --host 127.0.0.1 --port 8080

# 配合 API 一起启动（使用不同端口）
# 终端 1: API 服务
./snir api --port 8080
# 终端 2: Web 查看服务
./snir serve --port 8081
```