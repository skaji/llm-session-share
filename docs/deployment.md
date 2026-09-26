# Run a sharing server

The app has no login or ownership checks. Put it behind your organization's
existing private ingress or Cloudflare Access. Anyone with access can read,
replace, or delete shared sessions. The CLI needs only the server URL and any
credentials required by that proxy; it does not need storage credentials.

Each session is stored as `llm-session-share/<uuid>.json` inside the chosen
bucket or local directory. The bucket must already exist. There is no database
or migration step.

## Local server

Requires Node.js 24+. From a checkout of this repository:

```sh
npm ci
npm run dev
```

The server listens at `http://localhost:8789` and saves data under `.data/`.
Use that URL with the CLI's `-server` option. Override the development port with
`PORT=9000 npm run dev`. Stop the server with Ctrl-C;
saved sessions remain on disk.

| Environment | Storage | Run |
| --- | --- | --- |
| Local | `.data/llm-session-share/*.json` | `npm run dev` |
| GCP | GCS | Cloud Run / Node.js |
| Cloudflare | R2 binding | Workers |

Node server settings:

| Variable | Default | Purpose |
| --- | --- | --- |
| `STORAGE` | `local` | `local` or `gcs` |
| `DATA_DIR` | `.data` | Local storage root |
| `GCS_BUCKET` | None | Required with `STORAGE=gcs`; bucket name only |
| `PORT` | `8789` with `npm run dev`; otherwise `8080` | Listening port |
| `HOST` | `0.0.0.0` | Listening address |

## GCP (primary deployment)

Use a published image such as `ghcr.io/skaji/llm-session-share:v0.0.1`
(replace the tag with an existing release). The GHCR package must be public for
Cloud Run to pull it directly. Alternatively, build the included Dockerfile and
push the image to your company's registry.

Deploy using your existing Cloud Run deployment/network settings and set:

```text
STORAGE=gcs
GCS_BUCKET=your-existing-bucket
```

The GCS adapter uses Application Default Credentials. Give the Cloud Run service
account permission to read, create, overwrite, and delete objects under
`llm-session-share/` in the bucket. No bucket credential is needed on the sender's
PC. For local GCS testing, use your normal ADC setup and run:

```sh
STORAGE=gcs GCS_BUCKET=your-existing-bucket npm run dev
```

Use the existing private ingress/invocation setup in front of the service; the
app adds no login layer. Merely configuring outbound VPC access on a new Cloud
Run service does not configure its inbound access.

## Cloudflare Workers + R2

1. Set `r2_buckets[0].bucket_name` in `wrangler.jsonc` to your R2 bucket.
2. Add a custom-domain route for your chosen hostname, for example:
   `"routes": [{ "pattern": "share.example.com", "custom_domain": true }]`.
3. Protect that hostname with Cloudflare Access for your intended viewers and
   sender. The app does not implement or validate Access authentication itself.
4. Run `npm run worker:deploy`.

`workers.dev` and preview URLs are disabled in the included config, so the
custom domain is the intended entry point. Keep the R2 bucket private. The
Worker uses its R2 binding without storage credentials in the CLI.

To use your own Cloudflare Access login, install `cloudflared`, log in once,
and enable `-cloudflare-access`:

```sh
cloudflared access login https://share.example.com
llm-session-share -cloudflare-access -server https://share.example.com -user skaji /path/to/session.jsonl
```

Before each upload attempt, the CLI runs
`cloudflared access token --app <server-url>` and sends its output in the
`Cf-Access-Token` header, following the
[Cloudflare CLI authentication flow](https://developers.cloudflare.com/cloudflare-one/tutorials/cli/).
The token is never printed or saved by this app. If it is missing or expired,
log in again with `cloudflared access login`; the running watcher retries and
reads the refreshed token on its next attempt. This option takes precedence
over the service-token environment variables below. GCP/local use does not
invoke `cloudflared` unless the option is enabled.

Alternatively, for an unattended CLI behind Access, configure an Access Service Auth policy
that accepts your service token and set these variables on the sender:

```sh
export CF_ACCESS_CLIENT_ID=...
export CF_ACCESS_CLIENT_SECRET=...
export LLM_SESSION_SHARE_URL=https://share.example.com
export LLM_SESSION_SHARE_USER=skaji
llm-session-share /path/to/session.jsonl
```

These become `CF-Access-Client-Id` and `CF-Access-Client-Secret` headers on uploads.
They are for the upstream Access service, not application authentication, and
are not stored in the shared session. An Access login redirect is reported as a
failed upload. Viewers log in through their browsers normally.

For local Workers/R2 testing, `npm run worker:dev` uses Wrangler's local storage
and does not need a deployed service. For a build check without deployment, run
`npm run worker:check`.

