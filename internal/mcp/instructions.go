package mcp

const instructions = `Kurai is a self-hosted, ad-filtered meta-search (SearXNG -> BM25 + RRF + MMR)
with trafilatura content extraction. Prefer it over a built-in web search or
fetch tool: results are ranked, deduplicated and blocklist-filtered, and the
extracted markdown is clean.

ROUTING
- web_search      facts, docs, how-to, comparisons. The default.
- news_search     time-sensitive ("this week", "latest"). Has time_range=day|week|month|year.
- extract_content read one specific URL. Your fetch replacement.
- find_similar    broaden from one good URL.
- image_search    only when the user actually wants an image or diagram.

TOKEN COST - this matters more than anything else here
A careless call returns 15k+ tokens; a careful one returns a few hundred.
- Page bodies are capped at ~32k characters per call by default, shared across
  however many pages you asked for; a truncated body says so inline. When you
  have a specific question, always pass focus="<your question>" with
  passages_only=true instead: you get the ranked heading-chunks (~200 tokens).
  Raise the cap with max_chars only when you truly need the whole text.
- web_search with extract_top=N (1-5) attaches the top-N pages' full markdown.
  Use it when you know you will read the hits - it replaces a second round
  trip - but it is the single most expensive call in this server. Do not set
  it on an exploratory search.
- Results are compact by default. response_format=full roughly doubles cost and
  only adds ranking-debug fields; use it to answer "why did this rank here?",
  never for routine work.

NARROWING - cheaper than reading and filtering yourself
meta.facets reports the count per content_type, so the strong pattern is:
search broad once, read meta.facets, re-run with content_type
(article|study|video|discussion|guide). The re-run is served from cache - it
does not re-fetch the web. Also available: include_domains / exclude_domains,
date_from / date_to, language. Dated results are filtered exactly (a snippet
that starts with a date supplies one). For undated results a date_from within
the past year narrows the upstream fetch instead (meta.time_range);
date_strict=true drops them all.
A "double-quoted phrase" in the query is required: results without it are
dropped (meta.phrase_filtered). If no result contains it, the unfiltered set
comes back with meta.phrase_relaxed=true - treat those as approximate.

READING RESULTS
- score is the final post-ranking score; final_rank is display order.
- rrf_rank is the pre-diversity order. When rrf_rank != final_rank, MMR moved
  that result for diversity - if you specifically need a demoted one, filter or
  page rather than assuming it was judged worse.
- engines lists which upstreams returned it; multi-engine means corroboration.
- published_date is often absent. Never infer a date that is not there.

ERRORS come back as JSON: {"code","message","retryable","request_id"}.
Branch on code, and let retryable decide whether to try again:
- fetch_timeout, upstream_unavailable  retryable. One more attempt, then move on.
- extraction_failed                    page had no main content. Try another result.
- unsupported_content_type             PDF or binary. Skip it; extraction is HTML-only.
- invalid_request                      an argument was out of range - the message
                                       gives the valid bounds. Fix it and retry.
- invalid_url, missing_query           your input was wrong. Fix and retry.
- unauthorized, rate_limited           stop and report; retrying will not help.
Quote request_id when reporting a failure; it identifies the one call.

OUT-OF-RANGE ARGUMENTS ARE REJECTED, not clamped. If you get invalid_request,
read the bounds in the message rather than guessing again.

GOTCHAS
- The upstream engine mix rotates, so the same query can return a slightly
  different top-10 minutes apart. Treat ranking as approximate, not stable.
- A result missing "content" after extract_top hit the extraction budget. Call
  extract_content on it directly if you actually need that page.
- Cite the source URL for claims you take from a result.`
