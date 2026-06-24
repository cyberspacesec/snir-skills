# snir API, SDK, and Provider Integration

Use this reference when embedding snir in another system instead of running one-off CLI commands.

## HTTP API

Start a local API server:

```bash
snir api --host 127.0.0.1 --port 8080 --api-key secret
```

Capture one screenshot:

```bash
curl -X POST http://127.0.0.1:8080/screenshot \
  -H "X-API-Key: secret" \
  -H "Content-Type: application/json" \
  -d '{"url":"https://example.com","capture_full_page":true,"save_html":true,"save_headers":true}'
```

Use the API for non-Go callers, service integration, or request/response workflows.

## Go SDK

```go
client, err := sdk.NewClient(sdk.DefaultClientOptions())
if err != nil {
    return err
}
defer client.Close()

result, err := client.Screenshot("https://example.com", nil)
if err != nil {
    return err
}
fmt.Println(result.Endpoint, result.Title, result.ResponseCode)
```

Use the SDK when snir is part of a Go application and the caller needs typed options and results.

For complex screenshot scenarios, use the fluent Capture API:

```go
result, err := client.Capture(
    "https://example.com",
    sdk.WithFullPage(),
    sdk.WithEvidence(),
    sdk.WithDevice("iphone-15"),
    sdk.WithProxyList(runner.ProxyRoundRobin, "http://proxy-a:8080", "http://proxy-b:8080"),
    sdk.WithCookieHeader("session=abc; tenant=demo"),
    sdk.WithIgnoreCertErrors(),
)
```

Common SDK entrypoints:

- `Capture` / `CaptureBytes` for composable functional options.
- `ScreenshotEvidence` / `ScreenshotEvidenceBytes` for HTML, headers, cookies, console, and network evidence.
- `ScreenshotElement`, `ScreenshotXPath`, `ScreenshotElementBytes`, and `ScreenshotXPathBytes` for targeted capture.
- `ScreenshotDevice` and `ScreenshotViewport` for per-request browser profile control.
- `ScreenshotWithJSBefore` and `ScreenshotWithJSFile` for preload or file-based JavaScript injection.
- `WithProxyList`, `WithProxyFile`, `WithProxyURL`, and `WithProxyStrategy` for per-request proxy rotation.
- `WithCookieHeader`, `WithCookieStrings`, `WithCookieImport`, `WithCookieExport`, and `WithCookieWriteBack` for authenticated and stateful captures.
- `NewScreenshotOptions`, `WithFullPage`, `WithEvidence`, `WithDevice`, `WithViewport`, `WithPorts`, `WithJSAfter`, `WithCustomHeaders`, and related `With...` helpers for reusable scenario presets.

## CDP Provider

```bash
snir provider --port 9223 --idle-timeout 5m
```

Use the provider to share Chrome across processes or avoid starting a new browser per worker. Other snir commands can connect with `--wss`.

## Full References

- `docs/superpowers/api.md` for API flags and endpoint schemas.
- `docs/superpowers/provider.md` for CDP provider setup.
- `docs/skills.md` for SDK examples and broader integration notes.
