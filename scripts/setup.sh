#!/bin/sh
set -eu

cd "$(dirname "$0")/.."

if [ ! -f .env ]; then
  cp .env.example .env
  echo "created .env from .env.example"
fi

ensure_secret() {
  key=$1
  value=$2
  if grep -Eq "^${key}=[^[:space:]#]" .env; then
    return
  fi
  if grep -Eq "^${key}=" .env; then
    awk -v key="$key" -v value="$value" 'index($0, key "=") == 1 { print key "=" value; next } { print }' .env > .env.tmp
    mv .env.tmp .env
  else
    printf '%s=%s\n' "$key" "$value" >> .env
  fi
  echo "generated ${key}"
}

ensure_secret KURAI_AUTH_SECRET "$(openssl rand -base64 32)"
ensure_secret SEARXNG_SECRET "$(openssl rand -hex 32)"

if [ ! -f users.yaml ]; then
  printf 'users: []\n' > users.yaml
  echo "created an empty users.yaml"
fi
