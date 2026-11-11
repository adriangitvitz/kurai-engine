# Security

## Reporting a vulnerability

Report vulnerabilities privately through GitHub: open the **Security** tab of
this repository and choose **Report a vulnerability**. Do not open a public
issue for a security problem.

Include what you found, how to reproduce it, and which version or commit you
tested. You will get a reply once the report has been reviewed.

## Deployment notes

Kurai is built to run on your own machine or private network.

- Authentication is off by default. Set `KURAI_AUTH_ENABLED=true` and add users
  to `users.yaml` before exposing any port beyond localhost.
- `make setup` generates `KURAI_AUTH_SECRET` and `SEARXNG_SECRET` for each
  install. Never commit `.env` or `users.yaml`.
- The extractor refuses private and loopback addresses, so a search result
  cannot be used to read services on your network.
