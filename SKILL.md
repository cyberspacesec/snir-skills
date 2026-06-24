---
name: snir-web-intelligence
description: Capture web screenshots and collect page intelligence with the snir CLI, HTTP API, Go SDK, and CDP provider. Use when you need browser-based screenshots, HTML/headers/cookies/network evidence, batch URL or host scanning, device emulation, proxy rotation, or SQLite/JSONL/CSV scan outputs.
---

# snir Web Intelligence

Use `snir` as a Chrome DevTools Protocol based web screenshot and intelligence collector. It can capture single pages, batch URL lists, CIDR-expanded targets, and host/IP lists expanded across common web ports.

## Start Here

Prefer the bundled install helper when available:

```bash
./scripts/install-snir.sh
snir version
```

Or install the latest prebuilt binary manually:

```bash
LATEST=$(curl -s https://api.github.com/repos/cyberspacesec/snir-skills/releases/latest | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
OS=$(uname -s | sed 's/Linux/Linux/;s/Darwin/Darwin/;s/FreeBSD/Freebsd/;s/OpenBSD/Openbsd/;s/NetBSD/Netbsd/')
ARCH=$(uname -m | sed 's/x86_64/x86_64/;s/aarch64/arm64/;s/arm64/arm64/')
curl -L -o snir.tar.gz "https://github.com/cyberspacesec/snir-skills/releases/download/${LATEST}/snir-skills_${OS}_${ARCH}.tar.gz"
tar xzf snir.tar.gz snir
chmod +x snir
sudo mv snir /usr/local/bin/
snir version
```

If working from this repository:

```bash
make build
./snir version
```

## Common Tasks

Single page screenshot:

```bash
snir scan example.com
```

Collect page evidence:

```bash
snir scan example.com --full-page --save-html --save-headers --save-cookies --save-console --save-network
```

Batch scan URLs from a file:

```bash
snir scan file -f urls.txt --threads 10 --write-jsonl --db
```

Expand bare hosts/IPs across common web ports:

```bash
snir scan file -f hosts.txt --ports 80,443,8080,8443 --write-jsonl --db
```

Run the HTTP API:

```bash
snir api --host 127.0.0.1 --port 8080 --api-key secret
```

## Progressive Documentation

Open these files only when the task needs the extra detail:

- `references/README.md` - skill bundle structure and when to open each resource.
- `references/scan-workflows.md` - task-oriented CLI scan patterns.
- `references/api-and-sdk.md` - HTTP API, Go SDK, and CDP provider integration notes.
- `references/outputs-and-evidence.md` - result fields, persistence formats, and evidence collection.
- `scripts/install-snir.sh` - portable install helper for Linux, macOS, FreeBSD, OpenBSD, and NetBSD.
- `evals/evals.json` - realistic skill evaluation prompts and expectations.
- `docs/superpowers/SKILLS.md` - full skill index, installation paths, command map, and flag overview.
- `docs/superpowers/scan.md` - CLI screenshot, batch scan, ports, devices, proxies, evidence, and output options.
- `docs/superpowers/api.md` - HTTP API server, auth, endpoints, request and response schema.
- `docs/superpowers/provider.md` - shared Chrome/CDP provider for reuse across processes.
- `docs/superpowers/report.md` - report generation and conversion workflows.
- `docs/superpowers/webserve.md` - local web serving for generated outputs.
- `docs/skills.md` - broader CLI, Go SDK, HTTP API, and provider reference.

## Operating Notes

- Screenshot capture requires Chrome or Chromium unless `--wss` points to a remote Chrome instance.
- `--ports` expands URL candidates for web capture; it is not TCP/UDP port discovery.
- Use `--db --db-path <file>` when downstream analysis needs structured endpoint metadata and captured evidence.
- Use `--write-jsonl` for streaming or append-friendly pipelines.
- Stay within authorized scope when scanning third-party assets.
