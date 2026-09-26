# llm-session-share

Share a local Codex or Claude Code conversation with coworkers or friends.
Run the CLI on your PC, send someone the printed URL, and keep it running to
share new messages. Viewers need only a browser; your PC does not need to accept
incoming connections.

A sharing server must already be running. Ask its operator for the URL, or see
[server setup](docs/deployment.md) to run one locally, on Cloud Run/GCS, or on
Cloudflare Workers/R2.

## Start sharing

1. Download an archive from [GitHub Releases](https://github.com/skaji/llm-session-share/releases).
   Choose `darwin-arm64` for an Apple Silicon Mac, `darwin-amd64` for an Intel Mac,
   or `linux-amd64` / `linux-arm64` for Linux.
2. Extract it and put `llm-session-share` in a directory on your `PATH`.
   You do not need Go or Node.js to run the downloaded CLI.
3. Run the command below, replacing the server URL, display name, and JSONL path:

```sh
llm-session-share -server https://share.example.com -user skaji /path/to/session.jsonl
```

The first upload includes the existing conversation. The CLI prints a link:

```text
Sharing: https://share.example.com/session/<uuid>?markdown=1&live=1
```

Send that link to your viewers. It enables Markdown formatting and automatic
updates every two seconds. Keep the CLI running while sharing; Ctrl-C stops
updates but leaves the shared conversation available. Run the same command
again to resume at the same URL.

To avoid repeating the server and display name:

```sh
export LLM_SESSION_SHARE_URL=https://share.example.com
export LLM_SESSION_SHARE_USER=skaji
llm-session-share /path/to/session.jsonl
```

## Choose a session

You can pass a JSONL file or a session link. Put all options before that argument.
For example, with the environment variables above set:

```sh
llm-session-share /path/to/session.jsonl
llm-session-share 'codex://threads/01a0de4a-2245-7102-b715-58d2ab09ea8f'
llm-session-share 'claude://resume?session=01a0de4a-2245-7102-b715-58d2ab09ea8f'
```

Links resolve to files on **the PC running the CLI**; they do not download remote
conversations. Missing or ambiguous matches produce an error, in which case
specify the JSONL path directly.

| Source | Local JSONL files | Link lookup root |
| --- | --- | --- |
| Codex | `sessions/YYYY/MM/DD/rollout-…-<uuid>.jsonl` or `archived_sessions/…` | `$CODEX_HOME`, default `~/.codex` |
| Claude Code | `projects/<project>/<uuid>.jsonl` | `$CLAUDE_CONFIG_DIR`, default `~/.claude` |

Codex titles come from `<home>/session_index.jsonl` only when the selected file
is under `<home>/sessions/` or `<home>/archived_sessions/`. Claude titles come
from `ai-title` and `custom-title` records in the selected JSONL; manual titles
take priority. Title changes are shared too. Without a title, the page uses the
first user message.

## Cloudflare Access

If your sharing server uses Cloudflare Access, install `cloudflared` and log in:

```sh
cloudflared access login https://share.example.com
llm-session-share -cloudflare-access -server https://share.example.com -user skaji /path/to/session.jsonl
```

The CLI reads a token with `cloudflared access token --app <server-url>` before
each upload or deletion. If login expires, log in again; the watcher retries.
Viewers sign in through their browsers. Operators can also configure
[service-token authentication](docs/deployment.md#cloudflare-workers--r2).

## Remove a shared session

Stop any CLI watching that session, then run:

```sh
llm-session-share -server https://share.example.com -delete /path/to/session.jsonl
```

Session links also work. Add `-cloudflare-access` if needed; `-user` is not
required. This deletes only the server copy and leaves your local JSONL intact.
Viewers see the deletion on their next update or reload. A running watcher can
publish the session again, so stop it first.

## Options

| Option | Default | Purpose |
| --- | --- | --- |
| `-server` | `LLM_SESSION_SHARE_URL` | Sharing server URL |
| `-user` | `LLM_SESSION_SHARE_USER` | Your display name; required for uploads |
| `-cloudflare-access` | `false` | Read your login token using `cloudflared` |
| `-once` | `false` | Upload once and exit |
| `-delete` | `false` | Delete the server copy and exit |
| `-source` | `auto` | Override format detection with `codex` or `claude` |
| `-interval` | `1s` | Local file check and retry interval |

## Reading a conversation

- **Markdown** formats headings, tables, lists, and code. Turn it off for raw text.
- **Live updates** checks every two seconds and follows new messages when you
  are already near the bottom. Turn it off to fetch only once.
- **Latest ↓** scrolls to the end.
- Times use the viewer's browser timezone, including an offset such as `+09:00`.

The URL holds both options: `markdown=1&live=1` enables them. Missing parameters
or `0` turn them off. No browser preferences are saved. Mermaid and math are not
rendered. Images and local file attachments are not uploaded.

## What gets shared

The CLI extracts user and assistant text, including assistant progress messages,
using the filtering rules from
[llm-session-search](https://github.com/skaji/llm-session-search). It excludes tool
calls/results, reasoning, and recognized injected context. It uploads the text,
message timestamps, session ID, source, display name, and title. It does not
upload the original JSONL or separately attach local files. Paths or other
information written in conversation text remain part of that text.

Only the selected session is shared. One CLI should publish a session at a time;
multiple senders with the same UUID overwrite the same server copy. Each change
replaces the full snapshot (up to 32 MiB). Failed uploads retry. There is no
heartbeat: “Last shared” means the last successful upload, not that the sender
is currently online.

This app is for trusted groups. Access control belongs to the infrastructure in
front of the server. There are no in-app accounts or ownership checks, and the
display name is not verified.

See [server setup](docs/deployment.md) for hosting and
[development](docs/development.md) for checks, releases, and the HTTP API.
