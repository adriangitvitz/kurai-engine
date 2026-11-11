# Search behavior

This page describes how Kurai interprets a query and what the `meta` block of
a `/search` response reports. The same rules apply to the web UI and to the
`web_search` MCP tool.

## Quoted phrases

A phrase in double quotes must appear in the title, snippet or URL of each
result.

```text
"northern trust" custody
```

- `meta.phrase_filtered` is the number of results dropped for lacking the
  phrase.
- If no result contains the phrase, Kurai returns the unfiltered results and
  sets `meta.phrase_relaxed` to `true`. Treat those results as approximate.
- An apostrophe is not a quote. `don't` and `company's` do not form a phrase.

## OR

`OR` in capitals groups the terms on either side as alternatives.

```text
"hardware security module" OR "HSM" custody
```

A group that contains a quoted phrase is satisfied by any of its members.

## Limiting to sites

`include_domains` keeps only results from the listed sites, and
`exclude_domains` drops them. Both match subdomains, so `example.com` covers
`docs.example.com`.

```bash
curl "http://localhost:8080/search?q=custody&include_domains=example.com"
```

- The sites are also sent to the search engines as `site:` operators, so the
  engines search inside them.
- Within a site, a result must match at least one query term other than the
  site's own name. `meta.off_topic_filtered` counts the results dropped. If
  nothing matches, the filter is skipped.

In the web UI, typing `site:example.com` in the search box does the same.

## Date ranges

`date_from` and `date_to` take dates as `YYYY-MM-DD`.

- A result with a known date is kept only if it falls inside the range.
- When an engine sends no date, Kurai uses the date at the start of the
  snippet if there is one, such as `Apr 17, 2026` or `3 days ago`.
- A result with no date at all cannot be checked and is kept. When
  `date_from` is within the past year, Kurai also asks the engines for recent
  results only, and reports the window in `meta.time_range` as `day`, `week`,
  `month` or `year`.
- `date_strict=true` drops every result without a date.
  `meta.undated_filtered` counts them.

## Duplicates

Results with the same URL are merged, and the engines that returned them are
combined.

Results from the same site with the same title are also merged, keeping the
best-ranked one. This applies to titles of four words or more. The same story
on two different sites is kept as two results, because they are separate
sources.

## Counts

`meta.total` and `meta.facets` describe the results after every filter,
including dead links removed from the current page. The total can therefore
change slightly between pages.

## Content types

Each result is classified as `article`, `study`, `video`, `discussion` or
`guide`. `meta.facets` gives the count of each, and `content_type` limits the
results to one of them.
