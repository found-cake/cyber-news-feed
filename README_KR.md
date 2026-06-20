# cyber-news-feed

사이버 보안 RSS 피드를 정적 JSON 파일로 수집하는 저장소입니다.

[English README](README.md)

이 저장소는 의도적으로 작게 유지합니다. RSS/Atom 피드를 가져오고, 기사 메타데이터를 정규화한 뒤, 소스별 JSON을 `data/rss/` 아래에 저장합니다. 기사 본문 크롤링, LLM 요약/분류, UI는 포함하지 않습니다.

## 데이터

소스별 JSON 파일:

| 소스 | 파일 |
| --- | --- |
| 보안뉴스 | `data/rss/boannews.json` |
| The Hacker News | `data/rss/thehackernews.json` |
| Cyber Security News | `data/rss/cybersecuritynews.json` |
| StepSecurity | `data/rss/stepsecurity.json` |
| Dark Reading | `data/rss/darkreading.json` |
| BleepingComputer | `data/rss/bleepingcomputer.json` |

Raw GitHub URL은 다음 형식입니다.

```text
https://raw.githubusercontent.com/found-cake/cyber-news-feed/master/data/rss/<source>.json
```

JSON Schema는 다음 위치에 있습니다.

```text
schema/rss-feed.schema.json
```

## JSON 형식

각 소스 문서는 다음 형태입니다.

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

각 기사는 다음 필드를 포함합니다.

- `id`: 정규화된 URL 기반의 안정적인 `sha256:<hex>` ID
- `url`: 정규화된 기사 URL
- `title`
- `published_at`: 파싱 가능한 경우 UTC RFC3339, 실패하면 `null`
- `published_raw`: 피드 원문 날짜 문자열
- `categories`: 피드 카테고리와 필요한 경우 소스 필터 카테고리
- `description`: 피드의 description 또는 Atom summary
- `feed_id`: 피드 GUID 또는 Atom ID
- `authors`
- `media`
- `source_metadata`: 소스별로 보존한 추가 메타데이터

Cyber Security News의 `content:encoded` 값은 현재 수집 소스 중 해당 소스에만 특화된 값이므로 `source_metadata.cybersecuritynews.content_encoded`에 저장합니다.

피드에서 제공한 HTML은 JSON 문자열 안에 가능한 그대로 보존합니다. 예를 들어 `<p>`는 `\u003c`로 escape하지 않고 `<p>`로 저장합니다.

## 업데이트 정책

- 소스는 서로 독립적으로 처리합니다.
- 한 소스가 실패해도 다른 소스는 계속 처리합니다.
- 실패한 소스는 기존 `articles`를 유지하고 `status`만 갱신합니다.
- 새 RSS 항목은 피드 원본 순서를 유지합니다.
- 기존 누적 항목은 새 항목 뒤에 붙입니다.
- 중복 제거 기준은 최소 URL 정규화 결과입니다.
  - 앞뒤 공백 제거
  - fragment 제거
  - trailing slash 제거
- 기본 보관 기간은 10일입니다.
- 실행 시 `RETENTION_DAYS`로 보관 기간을 바꿀 수 있습니다.

## 워크플로우

### CI

`.github/workflows/ci.yml`은 push와 pull request에서 `go test ./...`를 실행합니다.

### Release

`.github/workflows/release.yml`은 `v*` 태그 push에서 실행됩니다.

동작:

1. 테스트를 실행합니다.
2. Linux amd64 바이너리를 빌드합니다.
3. `gh release create`로 GitHub Release를 생성합니다.
4. `cyber-news-feed-linux-amd64`를 업로드합니다.

사용하는 GitHub Actions는 공식 action만 사용합니다.

- `actions/checkout`
- `actions/setup-go`

### Scheduled Harvest

`.github/workflows/schedule.yml`은 매시 15분 UTC에 실행되며, `workflow_dispatch`로 수동 실행할 수도 있습니다.

이 workflow는 소스에서 빌드하지 않습니다. 최신 Release 바이너리를 다운로드해서 실행합니다.

```sh
gh release download --pattern "cyber-news-feed-linux-amd64" --output cyber-news-feed
./cyber-news-feed
```

생성된 RSS 데이터가 바뀌면 workflow가 `data/rss` 변경사항을 commit/push합니다.

## 로컬 개발

요구사항:

- Go 1.26 라인

테스트:

```sh
go test ./...
```

빌드:

```sh
go build -o cyber-news-feed .
```

실행:

```sh
./cyber-news-feed
```

보관 기간 override:

```sh
RETENTION_DAYS=30 ./cyber-news-feed
```

출력은 `data/rss/`에 저장됩니다.

## 참고

- BoanNews RSS는 EUC-KR을 사용합니다. Go XML decoder의 charset hook으로 변환 처리합니다.
- 하베스터는 피드에 포함된 콘텐츠만 보존합니다. 기사 페이지를 별도로 가져오지 않습니다.
- 피드 HTML 안의 `&amp;`, `&#8217;` 같은 HTML entity는 피드 원문에서 온 값이므로 그대로 보존합니다.
