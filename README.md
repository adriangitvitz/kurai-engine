# Kurai

Kurai is a self-hosted web search service. It sends each query to a private
[SearXNG](https://github.com/searxng/searxng) instance, then filters,
deduplicates and re-ranks the results before returning them.

It has three ways in:

- A web UI for people.
- An HTTP API.
- A [Model Context Protocol](https://modelcontextprotocol.io/) server for AI
  agents.

## Contents

- [Features](#features)
- [Requirements](#requirements)
- [Quick start](#quick-start)
- [Configuration](#configuration)
- [Web UI](#web-ui)
- [HTTP API](#http-api)
- [MCP server](#mcp-server)
- [Authentication](#authentication)
- [Architecture](#architecture)
- [Development](#development)
- [Troubleshooting](#troubleshooting)
- [License](#license)

## Features

- **Ranking.** Results from several engines are fused with Reciprocal Rank
  Fusion over text relevance (BM25), engine position, domain authority and
  freshness, then diversified so one site does not fill the page.
- **Filtering.** Ad, tracker and content-farm domains are dropped using
  EasyList, the StevenBlack hosts list and your own blocklist. Dead links are
  detected and removed.
- **Search operators.** Double-quoted phrases are required, `OR` groups
  alternatives, and results can be limited by site and by date range.
- **Page reading.** Any URL can be fetched and returned as clean Markdown,
  optionally reduced to the passages that match a question.
- **Instant answers.** Currency conversions, weather and knowledge panels
  appear above the results when a query calls for them.
- **Optional AI modes.** With an OpenRouter key, the UI can write a
  synthesized answer from the top sources or build a longer research path.
  Without a key, Kurai works as a plain search engine.

## Requirements

- Docker with Docker Compose v2
- `make` and `openssl`
- About 2 GB of free memory

The SearXNG image is built for `linux/amd64`. On Apple Silicon it runs under
emulation, which works but is slower.

## Quick start

```bash
git clone https://github.com/adriangitvitz/kurai.git
cd kurai
make up
```

`make up` runs `make setup` first. That step creates `.env` from
`.env.example`, generates the two secrets it needs, and creates an empty
`users.yaml`. It never overwrites values you have already set.

The first start builds three images and downloads the blocklists, so allow a
few minutes.

| Service | URL | Purpose |
| --- | --- | --- |
| Web UI | `http://localhost:3000` | Search in the browser |
| Kurai API | `http://localhost:8080` | HTTP API and MCP endpoint |
| SearXNG | `http://localhost:8888` | The metasearch engine Kurai queries |

Check that it works:

```bash
curl "http://localhost:8080/health"
curl "http://localhost:8080/search?q=kubernetes+operator+pattern&limit=3"
```

Stop the stack with `make down`.

## Configuration

Settings live in three places.

| File | What it controls |
| --- | --- |
| `.env` | Secrets, feature switches and API keys. Created by `make setup`. |
| `config.yaml` | Ranking weights, timeouts, blocklists, the liveness probe and cache. |
| `searxng/settings.yml` | Which upstream search engines SearXNG uses. |

After changing any of them, apply the change with:

```bash
make redeploy
```

### Environment variables

| Variable | Default | Purpose |
| --- | --- | --- |
| `KURAI_AUTH_SECRET` | generated | Signs session cookies. |
| `SEARXNG_SECRET` | generated | SearXNG instance secret. |
| `KURAI_AUTH_ENABLED` | `false` | Require an API key. See [Authentication](#authentication). |
| `OPENROUTER_API_KEY` | empty | Enables synthesized answers and deep research. |
| `OPENROUTER_MODEL` | `anthropic/claude-sonnet-5` | Model that writes the final answer. |
| `OPENROUTER_MODEL_FAST` | `anthropic/claude-haiku-4.5` | Cheaper model for planning and summaries. |
| `KURAI_TAVILY_ENABLED` | `false` | Use Tavily when SearXNG returns few results. |
| `TAVILY_API_KEY` | empty | Required only when Tavily is enabled. |
| `KURAI_LOG_LEVEL` | `info` | One of `debug`, `info`, `warn`, `error`. |

More detail, including DNS-over-HTTPS and the optional reranker, is in
[docs/configuration.md](docs/configuration.md).

## Web UI

Open `http://localhost:3000`.

- Type a query to search. Paste a link to read that page instead.
- Choose between sources only, a synthesized answer, and deep research. The
  last two need `OPENROUTER_API_KEY`.
- Filter results by site and by date range. Typing `site:example.com` in the
  search box sets the site filter.
- Save sources to a board. Saved items are stored on your machine.

Every view has its own URL, so searches can be bookmarked, shared and scripted.

| URL | Opens |
| --- | --- |
| `/?q=<query>` | Results for the query |
| `/?q=<query>&mode=synth` | Results with a synthesized answer |
| `/?q=<query>&mode=deep` | Deep research on the query |
| `/?q=<query>&site=a.com,b.org` | Results limited to those sites |
| `/?q=<query>&from=2026-01-01&to=2026-06-30` | Results limited to a date range |
| `/?url=<page url>` | The page, read in place |

## HTTP API

The API is described by an OpenAPI 3.1 document, served by the running server
at `/openapi.yaml` and `/openapi.json`. The source is
[internal/server/openapi.yaml](internal/server/openapi.yaml).

### Search

```bash
curl "http://localhost:8080/search?q=react+useeffect&include_domains=react.dev&limit=5"
```

Common parameters:

| Parameter | Meaning |
| --- | --- |
| `q` | The query. A `"quoted phrase"` is required in each result. |
| `page`, `limit` | Pagination and page size. |
| `include_domains`, `exclude_domains` | Comma-separated sites. Subdomains match. |
| `date_from`, `date_to` | Date range, as `YYYY-MM-DD`. |
| `content_type` | One of `article`, `study`, `video`, `discussion`, `guide`. |
| `extract_top` | Also return the full text of the top N results. |
| `response_format` | `compact` returns fewer fields per result. |

How phrases, site limits and dates behave is covered in
[docs/search.md](docs/search.md).

### Read a page

```bash
curl "http://localhost:8080/extract?url=https://go.dev/doc/effective_go&format=markdown"
```

Add `focus=<question>&passages_only=true` to get only the passages relevant
to a question.

### Other endpoints

| Endpoint | Purpose |
| --- | --- |
| `GET /news` | News search with clustering |
| `GET /images` | Image search |
| `GET /bookmarks` | The saved board |
| `GET /health` | Service status |

### Errors

Every error uses the same JSON shape:

```json
{
  "error": {
    "code": "missing_query",
    "message": "a non-empty search query is required"
  },
  "request_id": "req_..."
}
```

The codes are listed in [docs/error-codes.md](docs/error-codes.md).

## MCP server

Kurai exposes five tools to MCP clients such as Claude Code and Cursor.

| Tool | Purpose |
| --- | --- |
| `web_search` | Ranked web search with filters |
| `news_search` | Ranked news search |
| `image_search` | Image search |
| `extract_content` | Read a URL as Markdown |
| `find_similar` | Find pages similar to a URL |

### Over HTTP

The MCP endpoint is `/mcp` on the API port. With Claude Code:

```bash
claude mcp add --transport http kurai http://localhost:8080/mcp
```

When authentication is enabled, pass the key as a header:

```bash
claude mcp add --transport http kurai http://localhost:8080/mcp \
  --header "Authorization: Bearer $KURAI_KEY"
```

### Over stdio

Install the binary and point it at a config file. SearXNG must be reachable
at the URL set in that file.

```bash
go install github.com/adriangitvitz/kurai/cmd/kurai@latest
claude mcp add --transport stdio kurai -- "$(which kurai)" mcp -config "$HOME/.config/kurai/config.yaml"
```

The full guide for agents, with tool inputs, response fields and usage
patterns, is in [docs/AGENTS.md](docs/AGENTS.md).

## Authentication

Authentication is off by default, which suits a single machine. Turn it on
before exposing Kurai to a network.

1. Create a key for each user or agent:

   ```bash
   docker compose exec kurai /app/kurai gen-key --user alice --name "Alice"
   ```

   The raw key is printed once on standard output. A ready-to-paste entry is
   printed on standard error.

2. Paste that entry into `users.yaml`. See
   [users.example.yaml](users.example.yaml) for the format.

3. Set `KURAI_AUTH_ENABLED=true` in `.env`.

4. Apply the change:

   ```bash
   make redeploy
   ```

With authentication on, the web UI asks for the key at `/login`, and API
clients send it as `Authorization: Bearer <key>`.

To use a session cookie from the command line instead:

```bash
curl -c cookies.txt -X POST http://localhost:8080/auth \
  -H 'Content-Type: application/json' \
  -d '{"key": "your-api-key"}'

curl -b cookies.txt "http://localhost:8080/search?q=test"
```

Kurai stores only a hash of each key. Sessions last 8 hours by default,
set by `auth.session_ttl` in `config.yaml`.

## Architecture

```text
Browser ----> web (Next.js, :3000) ----+
                                       |
MCP client / curl ---------------------+--> kurai (Go, :8080) --> searxng (:8888) --> search engines
                                                  |
                                                  +--> page extraction, blocklists, ranking, cache
```

| Path | Contents |
| --- | --- |
| `cmd/kurai` | Server entry point and the `gen-key`, `gen-secret` and `mcp` subcommands |
| `internal/server` | HTTP handlers, search pipeline, filters |
| `internal/ranker` | Scoring, rank fusion, diversity |
| `internal/extract` | Page fetching and content extraction |
| `internal/mcp` | MCP tools |
| `internal/blocklist` | Blocklist download, cache and lookup |
| `internal/auth` | API keys and sessions |
| `web` | The Next.js web UI |
| `searxng` | SearXNG image and settings |
| `docs` | Reference documentation |

## Development

Run the backend tests:

```bash
go build ./...
go vet ./...
go test ./...
```

Run the web UI checks:

```bash
cd web
pnpm install --ignore-scripts
pnpm lint
pnpm build
pnpm exec playwright install chromium
pnpm exec playwright test
```

The end-to-end tests use fixtures for every backend call, so they need
neither the Go server nor network access.

See [CONTRIBUTING.md](CONTRIBUTING.md) before opening a pull request.

### Make targets

| Target | Action |
| --- | --- |
| `make setup` | Create `.env` and `users.yaml`, generate secrets |
| `make up` | Start the stack |
| `make redeploy` | Rebuild images and restart |
| `make up-doh` | Start with DNS-over-HTTPS |
| `make down` | Stop the stack |
| `make build` | Build the Go binary into `bin/` |
| `make test` | Run Go tests with the race detector |

## Troubleshooting

### `SEARXNG_SECRET is not set`

Docker Compose was started before the setup step. Run:

```bash
make setup
```

### Searches return few or no results

Upstream engines rate-limit and occasionally block automated traffic.

1. Look at which engines are failing:

   ```bash
   docker compose logs --tail 50 searxng
   ```

2. Check that the container can reach the internet:

   ```bash
   docker compose exec searxng wget -q --spider https://duckduckgo.com && echo ok
   ```

3. If you started with `make up-doh`, try `make down` followed by `make up`.

### The first start is slow

Kurai downloads about 15 MB of blocklists on first start and caches them in a
Docker volume. Later starts reuse the cache.

## License

Kurai is licensed under the [Apache License 2.0](LICENSE).

The files under `searxng/` that are derived from SearXNG keep their
AGPL-3.0-or-later license. See [NOTICE](NOTICE) for the list and for
third-party fonts and data.
