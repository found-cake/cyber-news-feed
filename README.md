# cyber-news-feed

Cyber security RSS feeds harvested into static JSON files.

[한국어 README](README_KR.md)

This repository is intentionally small: it fetches RSS/Atom feeds, normalizes article metadata, and publishes per-source JSON under `data/rss/`. It does not crawl article bodies, summarize with LLMs, classify content, or provide a UI.

## Data

Source JSON files:

| Source | File |
| --- | --- |
| BoanNews | `data/rss/boannews.json` |
| The Hacker News | `data/rss/thehackernews.json` |
| Cyber Security News | `data/rss/cybersecuritynews.json` |
| StepSecurity | `data/rss/stepsecurity.json` |
| Dark Reading | `data/rss/darkreading.json` |
| BleepingComputer | `data/rss/bleepingcomputer.json` |
| SecurityWeek | `data/rss/securityweek.json` |

Raw GitHub URLs follow this form:

```text
https://raw.githubusercontent.com/found-cake/cyber-news-feed/master/data/rss/<source>.json
```

The JSON Schema is available at:

```text
schema/rss-feed.schema.json
```

## JSON Shape

Each source document contains:

```json
{
  "schema_version": 1,
  "source": "cybersecuritynews",
  "updated_at": "2026-06-20T08:37:37Z",
  "retention_days": 10,
  "status": {
    "ok": true,
    "last_success_at": "2026-06-20T08:37:37Z",
    "last_error_at": null,
    "last_error": null
  },
  "articles": []
}
```

Each article includes:

- `id`: stable `sha256:<hex>` ID based on normalized URL
- `url`: normalized article URL
- `title`
- `published_at`: UTC RFC3339 when parsable, otherwise `null`
- `published_raw`: original feed date string
- `categories`: feed categories plus source filter categories where applicable
- `description`: feed-provided description or Atom summary
- `feed_id`: feed GUID or Atom ID when present
- `authors`
- `media`
- `source_metadata`: source-specific metadata preserved from feeds

Cyber Security News `content:encoded` values are stored under `source_metadata.cybersecuritynews.content_encoded`, because this field is source-specific in the current feed set.

SecurityWeek channel `image` values are stored under `source_metadata.securityweek.image` with `url`, `title`, `link`, `width`, and `height`, because that feed exposes its image metadata outside article items instead of Media RSS.

HTML in feed-provided fields is preserved as literal JSON string content. For example, `<p>` remains `<p>` instead of being JSON-escaped as `\u003c`.

## Update Policy

- Sources are processed independently.
- If one source fails, other sources continue.
- A failed source keeps its previous `articles` and only updates `status`.
- New RSS items keep feed order.
- Existing retained items are appended after newly fetched items.
- Deduplication uses a minimal normalized URL:
  - trim whitespace
  - remove URL fragments
  - remove trailing slashes
- Default retention is 10 days.
- `RETENTION_DAYS` can override retention at runtime.

## Workflows

### CI

`.github/workflows/ci.yml` runs `go test ./...` on pushes and pull requests.

### Release

`.github/workflows/release.yml` runs on `v*` tags.

It:

1. Runs tests.
2. Builds a Linux amd64 binary.
3. Creates a GitHub Release with `gh release create`.
4. Uploads `cyber-news-feed-linux-amd64`.

Only official GitHub actions are used:

- `actions/checkout`
- `actions/setup-go`

### Scheduled Harvest

Scheduled harvesting is triggered every 2 hours by a Cloudflare Worker. The GitHub Actions workflow at `.github/workflows/schedule.yml` is kept as the manual `workflow_dispatch` target that the Worker invokes.

The workflow does not build from source. Instead, it downloads the latest Release binary and runs it:

```sh
gh release download --pattern "cyber-news-feed-linux-amd64" --output cyber-news-feed
./cyber-news-feed
```

If generated RSS data changes, the workflow commits and pushes `data/rss`.

## Local Development

Requirements:

- Go 1.26 line

Run tests:

```sh
go test ./...
```

Build:

```sh
go build -o cyber-news-feed .
```

Run:

```sh
./cyber-news-feed
```

Override retention:

```sh
RETENTION_DAYS=30 ./cyber-news-feed
```

Output is written to `data/rss/`.

## Notes

- BoanNews uses EUC-KR RSS; the parser handles charset conversion through Go's XML decoder charset hook.
- The harvester preserves feed-provided content only. It does not fetch article pages.
- Feed HTML may contain HTML entities such as `&amp;` or `&#8217;`; those are preserved from the feed.
