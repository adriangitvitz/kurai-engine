# Configuration

Kurai reads settings from `.env`, `config.yaml` and `searxng/settings.yml`.
Apply any change with `make redeploy`.

## Environment variables

`make setup` creates `.env` from `.env.example`. The variables are listed in
the [README](../README.md#environment-variables), and each one is described
in `.env.example`.

Environment variables override the matching keys in `config.yaml`.

## Ranking and timeouts

`config.yaml` holds the ranking weights, penalties, cache lifetime and server
timeouts. Every section is commented in the file. The defaults are tuned for
general web search and do not need changing to get started.

## Blocklists

Kurai removes results from domains that appear in any of these lists.

| List | Source | Tracked in the repository |
| --- | --- | --- |
| EasyList | Downloaded at startup | No |
| StevenBlack hosts | Downloaded at startup | No |
| Tranco top sites | Downloaded at startup, used for domain authority | No |
| `blocklists/custom.txt` | Your own domains, one per line | Yes |
| `blocklists/content-farms.txt` | Known low-quality sites | Yes |

Downloaded lists are cached in the `blocklist-cache` Docker volume and
refreshed every 24 hours, set by `blocklists.refresh_hours`.

If a download fails, Kurai logs a warning and starts with the lists it has.
A stale cache is used when one exists.

To reload the lists without restarting:

```bash
curl -X POST "http://localhost:8080/blocklist/reload"
```

## Liveness probe

Before returning a page of results, Kurai sends a quick request to each
result and drops the ones that are clearly dead: DNS failures, refused
connections, and 404, 410 or 5xx responses.

| Setting | Effect |
| --- | --- |
| `KURAI_PROBE_ENABLED=false` | Turn the probe off |
| `KURAI_PROBE_DROP_UNREACHABLE=false` | Keep dead links and mark them `unreachable` |
| `probe.top_n` in `config.yaml` | How many results per page are probed |

## Search engines

`searxng/settings.yml` lists the upstream engines. Engines that rate-limit or
block the instance are suspended by SearXNG for a while and return on their
own. Remove an engine from the file to stop using it.

## DNS-over-HTTPS

By default the containers resolve names through `8.8.8.8` and `1.1.1.1`.

To route DNS through an encrypted Cloudflare proxy instead:

```bash
make up-doh
```

This adds a `cloudflared` container and makes it the first resolver for the
other services. Switch back with `make down` followed by `make up`.

## Synthesized answers and deep research

Both modes call [OpenRouter](https://openrouter.ai/). Set `OPENROUTER_API_KEY`
in `.env` to enable them. With the key empty, the web UI stays in sources-only
mode and no text leaves your machine except the search queries themselves.

Deep research is capped at 5 steps and 3 sources per step to bound its cost.

## Optional reranker

A cross-encoder reranker can re-score the top results. It is off by default
because the model is about 600 MB.

1. Start the stack with the reranker profile:

   ```bash
   docker compose --profile reranker up -d
   ```

2. Enable it in `config.yaml`:

   ```yaml
   reranker:
     enabled: true
     provider: tei
     url: http://tei:80
   ```

3. Run `make redeploy`.

The first start downloads the model. Allow one to two minutes before the
reranker takes effect.

## Saved board

Saved sources are stored in a SQLite file inside the `bookmark-data` Docker
volume. Each saved item records the query that led to it. The data stays on
your machine and is removed with the volume.
