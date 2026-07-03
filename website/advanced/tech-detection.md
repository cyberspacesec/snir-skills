# 技术检测

<p align="center">🔍 识别网站使用的技术栈。</p>

`pkg/techdetect` 通过指纹匹配识别框架/CMS/CDN/分析等。

## 指纹维度

`Fingerprint` 从四个维度匹配：

| 维度 | 字段 | 示例 |
|------|------|------|
| HTML | `HTML []string`（正则） | `<script src="jquery` |
| 头 | `Headers map[string]string` | `Server: nginx` |
| Cookie | `Cookies map[string]string` | `PHPSESSID` |
| Meta | `Meta map[string]string` | `generator: WordPress` |

## 流程

```mermaid
flowchart LR
  E[证据 HTML/头/Cookie/Meta] --> D[Detector]
  F[指纹库 fingerprints.go] --> D
  D --> T[[]Technology]
  T --> M[models.Technology]
  M --> R[Result.Technologies]
```

技术检测在证据采集后自动进行，结果存入 `Result.Technologies`。

## Detector

```go
func NewDetector() *Detector
func NewDetectorWithFingerprints(fps []Fingerprint) *Detector
```

`NewDetector` 用内置指纹库；`NewDetectorWithFingerprints` 可自定义。

## 查询

```sql
-- 每站技术栈
SELECT s.host, group_concat(t.name) FROM technologies t
JOIN screenshots s ON t.result_id = s.id GROUP BY s.host;

-- 用某技术的站点
SELECT s.host FROM technologies t
JOIN screenshots s ON t.result_id = s.id WHERE t.name = 'WordPress';
```

## 用途

- 资产技术栈盘点
- 发现特定框架站点（漏洞影响面）
- CMS 识别

## 下一步

- [pkg/techdetect](../internals/techdetect)
- [证据采集](./evidence)
- [安全侦察](../guide/security-recon)
