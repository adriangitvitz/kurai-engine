# Contributing

## Setup

You need Go 1.25 or newer, Node.js 20 or newer with pnpm, and Docker.

```bash
make setup
make up
```

The web UI is then at `http://localhost:3000` and the API at
`http://localhost:8080`.

## Running the checks

Run these before opening a pull request. All of them must pass.

Backend:

```bash
go build ./...
go vet ./...
go test ./...
```

Web UI:

```bash
cd web
pnpm install --ignore-scripts
pnpm lint
pnpm build
pnpm exec playwright install chromium
pnpm exec playwright test
```

The end-to-end tests intercept every backend call with fixtures, so they need
neither the Go server nor network access.

## Pull requests

- Keep each pull request to one change.
- Add or update a test for any change in behavior.
- Use pnpm in `web/`, not npm, and install with `--ignore-scripts`.
- Do not commit `.env`, `users.yaml` or anything under `blocklists/cache/`.

## License

By contributing you agree that your contribution is licensed under the
Apache License, Version 2.0.
