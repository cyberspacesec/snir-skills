# snir — Web Screenshot & Intelligence Collector

<p align="center">
  <strong>Chrome DevTools Protocol powered screenshot tool with multi-modal integration</strong>
</p>

<p align="center">
  <a href="https://github.com/cyberspacesec/snir-skills/releases/latest"><img src="https://img.shields.io/github/v/release/cyberspacesec/snir-skills?style=flat-square" alt="Release"></a>
  <img src="https://img.shields.io/github/go-mod/go-version/cyberspacesec/snir-skills?style=flat-square" alt="Go Version">
  <img src="https://img.shields.io/github/license/cyberspacesec/snir-skills?style=flat-square" alt="License">
  <img src="https://img.shields.io/github/actions/workflow/status/cyberspacesec/snir-skills/ci.yml?branch=main&style=flat-square" alt="CI">
</p>

---

## Integration Methods

### 1. 🤖 SKILLS (AI Agent Integration) — Recommended

SKILLS provides progressive-disclosure documentation that AI agents can use to autonomously install and operate snir without any prior Go knowledge.

**One-click install for AI agents:**

```bash
# Auto-detect platform and install the latest release
LATEST=$(curl -s https://api.github.com/repos/cyberspacesec/snir-skills/releases/latest | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
OS=$(uname -s | sed 's/Linux/Linux/;s/Darwin/Darwin/;s/FreeBSD/Freebsd/;s/OpenBSD/Openbsd/;s/NetBSD/Netbsd/')
ARCH=$(uname -m | sed 's/x86_64/x86_64/;s/aarch64/arm64/;s/arm64/arm64/')
curl -L -o snir.tar.gz "https://github.com/cyberspacesec/snir-skills/releases/download/${LATEST}/snir-skills_${OS}_${ARCH}.tar.gz"
tar xzf snir.tar.gz snir && chmod +x snir && sudo mv snir /usr/local/bin/
snir version
```

📖 **Anthropic Skills entry:** [`SKILL.md`](SKILL.md)

Canonical skill bundle resources:

| Resource | Purpose |
|----------|---------|
| [`references/`](references/) | On-demand task references loaded by agents only when needed |
| [`scripts/install-snir.sh`](scripts/install-snir.sh) | Portable release installer helper |
| [`evals/evals.json`](evals/evals.json) | Skill evaluation prompts and expectations |

📖 **Full SKILLS documentation:** [`docs/superpowers/SKILLS.md`](docs/superpowers/SKILLS.md)

Per-command docs with progressive disclosure (quick start → common options → advanced → full reference):

| Command | Document |
|---------|----------|
| `scan` | [`scan.md`](docs/superpowers/scan.md) |
| `api` | [`api.md`](docs/superpowers/api.md) |
| `provider` | [`provider.md`](docs/superpowers/provider.md) |
| `report` | [`report.md`](docs/superpowers/report.md) |
| `webserve` | [`webserve.md`](docs/superpowers/webserve.md) |
| `version` | [`version.md`](docs/superpowers/version.md) |

### 2. 🖥️ CLI

```bash
# Single URL screenshot
snir scan example.com

# Batch from file
snir scan file -f urls.txt

# Expand bare hosts/IPs by common Web ports
snir scan file -f hosts.txt --ports 80,443,8080,8443

# CIDR network scan
snir scan cidr 192.168.1.0/24

# Full-page screenshot with data collection
snir scan example.com --full-page --save-html --save-headers --save-cookies
```

### 3. 📦 Go SDK

```go
import "github.com/cyberspacesec/snir-skills/pkg/sdk"

client, _ := sdk.NewClient(sdk.DefaultClientOptions())
defer client.Close()

result, _ := client.Screenshot("https://example.com", nil)
fmt.Println(result.Title)
```

📖 [Go SDK documentation](docs/skills.md#二go-sdk-集成)

### 4. 🌐 HTTP API

```bash
# Start API server
snir api --port 8080 --api-key secret

# Screenshot via API
curl -X POST http://localhost:8080/screenshot \
  -H "X-API-Key: secret" \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com"}'
```

📖 [HTTP API documentation](docs/superpowers/api.md)

### 5. 🔌 CDP Provider (Share Chrome across processes)

```bash
snir provider --port 9223 --idle-timeout 5m
# Other tools connect via: --wss ws://host:9222/devtools/browser/xxx
```

📖 [Provider documentation](docs/superpowers/provider.md)

---

## Features

- **Screenshot** — Full-page, element-level (CSS selector / XPath), PNG/JPEG with quality control, file or in-memory bytes
- **Intelligence** — HTML source, HTTP headers, cookies, console logs, network requests, TLS/final URL/status metadata
- **Browser interaction** — JavaScript execution, form filling, click/scroll/input actions
- **Device and fingerprinting** — CDP device presets, mobile/touch/DPR emulation, custom User-Agent, WebGL, platform, language, WebRTC disable
- **Chrome reuse** — Connection pool, singleton pool, remote connection, auto-discovery
- **Proxy rotation** — Proxy list, proxy file (hot-reload), proxy API, round-robin/random/sequential strategies, local Chrome process isolation per proxy
- **Cookie management** — Persistent JSON cookie jar, Netscape format import/export, inline cookies
- **Library helpers** — Go SDK, HTTP API, pHash, technology detection, streaming/callback batch capture
- **Output** — JSONL, CSV, SQLite database, stdout
- **Cross-platform** — 43 platform combinations (Linux/Windows/macOS/FreeBSD/OpenBSD/NetBSD × amd64/arm64/386/arm/mips/ppc64le/riscv64/s390x)

For cyberspace mapping systems, snir is best treated as the Web asset collection, screenshot, fingerprinting, and page-evidence subsystem. See the Chinese [cyberspace mapping support assessment](docs/cyberspace-mapping-assessment.md) for scope boundaries and gaps.

---

## Installation

### Pre-built binaries (no Go required)

Download from [GitHub Releases](https://github.com/cyberspacesec/snir-skills/releases/latest):

| Platform | Command |
|----------|---------|
| **Linux x86_64** | `curl -L https://github.com/cyberspacesec/snir-skills/releases/latest/download/snir-skills_Linux_x86_64.tar.gz \| tar xz snir` |
| **macOS arm64** | `curl -L https://github.com/cyberspacesec/snir-skills/releases/latest/download/snir-skills_Darwin_arm64.tar.gz \| tar xz snir` |
| **Windows x86_64** | Download `snir-skills_Windows_x86_64.zip` from [Releases](https://github.com/cyberspacesec/snir-skills/releases/latest) |

### Linux packages (deb/rpm/archlinux)

Available in every [Release](https://github.com/cyberspacesec/snir-skills/releases/latest):

```bash
sudo dpkg -i snir_*.deb        # Debian/Ubuntu
sudo rpm -i snir-*.rpm         # RHEL/Fedora
sudo pacman -U snir-*.pkg.tar.zst  # Arch Linux
```

### Docker

```bash
docker pull ghcr.io/cyberspacesec/snir:latest
docker run --rm ghcr.io/cyberspacesec/snir:latest scan example.com
```

### From source (requires Go 1.23+)

```bash
git clone https://github.com/cyberspacesec/snir-skills.git
cd snir-skills && make build
```

### Prerequisite

Screenshot requires Chrome/Chromium. Or use `--wss` to connect to a remote Chrome instance.

```bash
sudo apt install chromium-browser   # Debian/Ubuntu
brew install --cask google-chrome   # macOS
```

---

## Quick Examples

```bash
# Single URL
snir scan example.com

# With timeout and proxy
snir scan example.com --timeout 60 --proxy http://127.0.0.1:8080

# Full-page + collect everything
snir scan example.com --full-page --save-html --save-headers --save-cookies --save-network

# Element screenshot (CSS selector)
snir scan example.com --selector "#dashboard-panel"

# Execute JavaScript before screenshot
snir scan example.com --js "document.querySelectorAll('.popup').forEach(el => el.remove());"

# Batch scan with proxy rotation
snir scan file -f urls.txt --threads 10 --proxy-file proxies.txt --proxy-strategy random

# Output to JSONL + database
snir scan file -f urls.txt --write-jsonl --db --db-path results.db
```

---

## Documentation

| Document | Description |
|----------|-------------|
| [SKILLS Index](docs/superpowers/SKILLS.md) | AI agent integration — install, commands, all 70 CLI flags |
| [Full Capabilities](docs/skills.md) | CLI + Go SDK + HTTP API + Provider complete reference |
| [Cyberspace Mapping Assessment](docs/cyberspace-mapping-assessment.md) | Scope, gaps, and priorities when used as a cyberspace mapping lower-level library |
| [Quick Examples](docs/quick_examples.md) | Copy-paste examples for common scenarios |
| [Usage Examples](docs/usage_examples.md) | Detailed examples with explanations |

---

## License

[MIT](LICENSE)

---

## 简体中文

[点击查看中文文档](README.zh-CN.md)
