# snir Outputs and Evidence

Use this reference when the task needs machine-readable results, persistence, or downstream analysis.

## Output Formats

```bash
snir scan file -f urls.txt --write-jsonl --jsonl-file results.jsonl
snir scan file -f urls.txt --write-csv --csv-file results.csv
snir scan file -f urls.txt --db --db-path results.db
```

- JSONL is best for append-friendly pipelines.
- CSV is best for simple inventory exports.
- SQLite is best when queries need endpoint metadata and structured evidence.

## Endpoint Metadata

Results include normalized endpoint fields:

- `schema_version`
- `scheme`
- `host`
- `port`
- `endpoint`
- `url`
- `final_url`

Use `endpoint` for deduplication and grouping when the same host is scanned across multiple schemes or ports.

## Evidence Fields

When enabled, snir can capture:

- screenshot file path or in-memory screenshot bytes
- HTML source
- HTTP headers
- cookies
- console logs
- network requests
- TLS metadata
- technology fingerprints
- perceptual hash

## SQLite Persistence

Use SQLite when downstream tooling needs to join screenshots, endpoints, and evidence:

```bash
snir scan file -f urls.txt \
  --db \
  --db-path results.db \
  --save-html \
  --save-headers \
  --save-cookies \
  --save-network \
  --save-console
```

SQLite stores endpoint fields and JSON evidence so later analysis can avoid reparsing raw JSONL.

## Scope Boundary

snir is a web asset collection and evidence tool. It is not a general TCP/UDP port scanner. For cyberspace mapping, pair it with an authorized discovery layer, then pass candidate web endpoints or host/port lists into snir.
