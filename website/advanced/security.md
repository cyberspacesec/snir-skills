# 安全注意

<p align="center">🛡️ 安全与合规使用 snir。</p>

::: warning 重要
仅在授权范围内使用 snir 扫描第三方资产。未经授权的扫描可能违法。
:::

## SSRF 防护

snir 默认黑名单屏蔽内网与云元数据地址，防 SSRF：

- 私有网段（10/8、172.16/12、192.168/16、fc00::/7）
- 云元数据（169.254.169.254、metadata.google.internal 等）
- 数据库端口（1433/3306/5432/6379）

请求在进入浏览器前的过滤链路如下：

```mermaid
flowchart TD
    U[用户输入 URL] --> B{黑名单检查}
    B -- 命中私有/元数据/DB 端口 --> R[拒绝并记录]
    B -- 通过 --> N[导航到目标]
    N --> P[页面加载与截图]
    P --> O[产物落盘]
    O --> D[数据处理<br/>脱敏 / 保管 / 清理]

    style B fill:#fff4e6,stroke:#e8a317,color:#1a1a1a
    style R fill:#fde8e8,stroke:#d23a3a
    style N fill:#e6f4ea,stroke:#3aa676
```

**生产环境务必保留默认黑名单。** 仅在授权内网扫描时才 `--enable-blacklist=false`。见 [黑名单](./blacklist)。

## API 鉴权

- `--api-key` 强随机密钥
- 生产前置 HTTPS（反代）
- 内网监听 `--host 127.0.0.1`
- 不提交 key 到代码库

见 [API 鉴权](../api/auth)。

## 远程 Chrome 暴露面

- 远程调试端口勿暴露公网
- `--remote-debugging-address` 谨慎开放
- 限内网或加网络层鉴权

见 [远程 Chrome](./remote-chrome)。

## 数据处理

- 截图/HTML/Cookie 可能含敏感信息
- 妥善保管 SQLite/JSONL 产物
- 按合规要求定期清理
- 分享报告前脱敏

## 代理使用

- 遵守代理服务条款
- WebRTC 可能泄露真实 IP，配合 `WithDisableWebRTC()`
- 代理不等于完全匿名

## 授权范围

- 仅扫描授权资产
- 尊重 robots.txt 与服务条款
- 控制并发与频率，避免影响目标
- 不绕过访问控制

## 合规

- 遵守当地法律法规
- 企业内部使用遵循公司安全策略
- 保留审计日志

安全注意要点按维度分类：

```mermaid
mindmap
  root((安全注意))
    SSRF 防护
      私有网段黑名单
      云元数据屏蔽
      数据库端口拦截
    证书
      生产不忽略
      测试可跳过
      默认 TLS 校验
    API Key
      强随机密钥
      内网监听
      前置 HTTPS
    WebRTC
      禁用防真实 IP 泄露
      WithDisableWebRTC
    代理
      遵守服务条款
      非完全匿名
    数据处理
      截图脱敏
      产物妥善保管
      定期清理
    授权范围
      仅扫授权资产
      尊重 robots
      控制频率
```

## 下一步

- [黑名单](./blacklist)
- [API 鉴权](../api/auth)
- [远程 Chrome](./remote-chrome)
- [FAQ](../reference/faq)
