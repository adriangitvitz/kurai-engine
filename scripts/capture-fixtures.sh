#!/usr/bin/env bash

set -euo pipefail

SEARXNG_URL="${SEARXNG_URL:-http://localhost:8888}"
FIXTURE_DIR="$(cd "$(dirname "$0")/.." && pwd)/internal/ranker/testdata/fixtures"
mkdir -p "$FIXTURE_DIR"

declare_queries() {
  cat <<'EOF'
rust_ownership|rust ownership tutorial
kubernetes_pod_terminating|kubernetes pod stuck terminating
react_useeffect_cleanup|react useEffect cleanup function
cbt_overview|what is cognitive behavioral therapy
sqlite_vs_postgres|sqlite vs postgres performance benchmark
EOF
}

if ! curl -sf "${SEARXNG_URL}/healthz" >/dev/null; then
  echo "SearXNG is not responding at ${SEARXNG_URL}/healthz" >&2
  echo "Start it with: docker compose up -d searxng" >&2
  exit 1
fi

while IFS='|' read -r id query; do
  [[ -z "$id" || "$id" == "#"* ]] && continue
  enc=$(jq -rn --arg q "$query" '$q|@uri')
  out="${FIXTURE_DIR}/${id}.json"
  echo "fetching ${id} (\"${query}\") ..."
  curl -fsS "${SEARXNG_URL}/search?q=${enc}&format=json" \
    | jq '{
        query: .query,
        results: [.results[] | {
          url, title,
          content: .content,
          engines: (.engines // []),
          score: (.score // 0),
          publishedDate: (.publishedDate // null)
        }]
      }' > "${out}"
  count=$(jq '.results | length' "$out")
  echo "  saved ${count} results -> ${out#$PWD/}"
done < <(declare_queries)

echo ""
echo "done. Re-run the golden eval:"
echo "  go test ./internal/ranker/ -run TestGolden_NDCGFloor -v"
