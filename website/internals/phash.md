# 感知哈希（pHash）

<p align="center">🖼️ `pkg/phash/phash.go` — 视觉去重与相似度。</p>

`pkg/phash` 用感知哈希（perceptual hash）比较截图，发现视觉重复或微小变化，比逐像素对比快得多且对人眼变化敏感。

> 📁 源码：[`pkg/phash/phash.go`](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/phash/phash.go)

## 核心类型与函数

| 符号 | 源码 | 说明 |
|------|------|------|
| `HashResult` | [L15](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/phash/phash.go#L15) | 哈希结果（含距离/相似度） |
| `ComputeHash(path)` | [L23](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/phash/phash.go#L23) | 单图计算 |
| `ComputeHashFromImage(img)` | [L31](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/phash/phash.go#L31) | 从 image.Image 计算 |
| `ComputePerceptionHash(path)` | [L43](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/phash/phash.go#L43) | 别名 |
| `CompareHashes(a, b)` | [L52](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/phash/phash.go#L52) | 比较两哈希 |
| `HashToHex(h)` | [L67](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/phash/phash.go#L67) | 哈希转十六进制 |
| `HexToHash(s)` | [L75](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/phash/phash.go#L75) | 十六进制转回 |

## 算法流程

基于 [goimagehash](https://github.com/corona10/goimagehash)，默认 pHash（DCT）：

```mermaid
flowchart LR
  IMG[原图] --> RS[缩放到 32x32]
  RS --> GS[灰度化]
  GS --> DCT[DCT 变换]
  DCT --> TOP[取左上 8x8]
  TOP --> MED[算中值]
  MED --> BIN[按中值二值化]
  BIN --> HASH[64bit 哈希]
```

## 相似度计算

[`CompareHashes`](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/phash/phash.go#L52) 用汉明距离（不同位数）衡量差异：

```
  相似度 = 1 - hamming_distance / 64

  distance=0  → 完全相同 (相似度 1.0)
  distance<5  → 视觉几乎相同
  distance<10 → 高度相似
  distance>20 → 明显不同
```

## HashResult 字段

| 字段 | 说明 |
|------|------|
| `Hash` | 64bit 感知哈希 |
| `Hex` | 十六进制表示 |
| `Distance` | 与参照的距离 |
| `Similarity` | 0~1 相似度 |

## 应用场景

- **去重**：批量结果中过滤视觉重复页
- **变化监控**：定期截图，距离>阈值即告警
- **克隆检测**：钓鱼站常复刻正版视觉
- **回归**：UI 变更前后对比

见 [感知哈希（进阶）](../advanced/perceptual-hash) 与 [监控](../guide/monitoring)。

## 存储与恢复

哈希可存为十六进制串（SQLite/JSONL），需要时 [`HexToHash`](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/phash/phash.go#L75) 还原再比较，无需保留原图。

## 下一步

- [感知哈希（进阶）](../advanced/perceptual-hash)
- [pkg/models](./models)
- [变化监控](../guide/monitoring)
- [技术检测](./techdetect)
