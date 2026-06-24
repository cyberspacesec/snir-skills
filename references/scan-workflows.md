# snir Scan Workflows

Use this reference when a task involves CLI screenshot capture, batch scanning, endpoint expansion, proxies, devices, or local outputs.

## Single Target

```bash
snir scan example.com
snir scan https://example.com --full-page
```

`snir` accepts bare hostnames and URLs. If no scheme is provided, it tries enabled web schemes according to scan options.

## Evidence Collection

```bash
snir scan example.com \
  --full-page \
  --save-html \
  --save-headers \
  --save-cookies \
  --save-console \
  --save-network
```

Use this when the output needs more than an image: page source, response headers, cookies, console logs, and network activity.

## Batch URL List

```bash
snir scan file -f urls.txt --threads 10 --write-jsonl --jsonl-file results.jsonl
```

Use JSONL for streaming pipelines and post-processing.

## Host/IP Lists Across Web Ports

```bash
snir scan file -f hosts.txt --ports 80,443,8080,8443 --write-jsonl --db --db-path results.db
```

`--ports` expands bare hosts or IP addresses into candidate web URLs. It is not TCP/UDP port discovery. Existing URLs with `http://` or `https://` are preserved.

## CIDR Input

```bash
snir scan cidr 192.168.1.0/24 --ports 80,443,8080,8443
```

Use CIDR mode for local or explicitly authorized ranges. Keep scope boundaries explicit.

## Device Emulation

```bash
snir scan example.com --device iphone-15
snir scan example.com --device pixel-8-pro
snir scan --list-devices
```

Device presets set viewport, DPR, mobile/touch behavior, and User-Agent before navigation.

## Proxy Rotation

```bash
snir scan file -f urls.txt --proxy-file proxies.txt --proxy-strategy random --threads 10
```

Use proxy rotation for distributed capture only when it is authorized and operationally necessary.

## Full Reference

Open `docs/superpowers/scan.md` for the full scan flag table and examples.
