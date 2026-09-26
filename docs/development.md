# Development

Requires Go 1.27+ and Node.js 24+. Run `npm ci` before the checks below.
Build the CLI with `go build -o llm-session-share .`.

- `internal/session`: JSONL extraction, titles, and local session-link lookup.
- `internal/share`: file watching, uploads/deletion, and Cloudflare Access tokens.
- `server/src/app.ts`: HTTP API and snapshot validation.
- `server/src/web.ts`: HTML, CSS, and browser behavior.
- `server/src/storage`: local filesystem, GCS, and R2 adapters.

## Checks

```sh
gofumpt -w main.go internal
goimports -w main.go internal
go build ./...
go test -race ./...
golangci-lint run
npm run check
npm test
npm run build
npm run worker:check
```

## Releases

Pushing a tag triggers `.github/workflows/release.yml`. GoReleaser runs after
Go and server checks pass, and publishes a GitHub Release with CLI archives for Linux and macOS
(amd64 and arm64), plus `checksums.txt`.

The same workflow builds the server Docker image for `linux/amd64` and pushes
it to `ghcr.io/skaji/llm-session-share:<tag>` using `GITHUB_TOKEN`; no extra secret
is required. Git tags should also be valid Docker tags, such as `v0.1.0`.
Both publication jobs depend on the shared checks (Go tests, TypeScript checks,
server tests, and Node/Workers builds). After those pass, the publication jobs
run independently. The workflow does not deploy to Cloud Run. After the first image push, set the package visibility to public in
GitHub Packages if you want Cloud Run to pull it directly from GHCR.

For example, after the release workflow is on the commit you want to release:

```sh
git tag v0.1.0
git push origin v0.1.0
```

Creating a local tag alone does not trigger the workflow.

## HTTP API

- `PUT /api/sessions/<uuid>`: replace a snapshot; returns `204` after storage succeeds.
- `DELETE /api/sessions/<uuid>`: delete the shared snapshot; returns `204` even if it is already absent.
- `GET /api/sessions/<uuid>`: snapshot JSON, or `304` for a matching `If-None-Match`.
- Add `?render=markdown` to include server-generated `html` alongside each message's original `text`.
- `GET /session/<uuid>`: conversation page; with `live=1`, it waits for an unshared session.
- `GET /healthz`: process health check.

Upload body:

```json
{
  "id": "01a0de0e-f326-7ec1-bc06-0aee7bbdf319",
  "source": "codex",
  "name": "skaji",
  "title": "Explain this change",
  "messages": [
    { "role": "user", "text": "Please explain this change.", "timestamp": "2026-09-27T00:00:00Z" },
    { "role": "assistant", "text": "Here is what changed." }
  ]
}
```

`source` is `codex` or `claude`. `title` and message timestamps are optional. The API field `name` holds the CLI
`-user` value. The server adds
`updated_at` on each successful upload. There is no sender heartbeat: "Last
shared" describes the last upload, not whether the source PC is currently online.

