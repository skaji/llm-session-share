import { serve } from '@hono/node-server';
import { createApp } from './app.js';
import { gcsStore } from './storage/gcs.js';
import { localStore } from './storage/local.js';

const mode = process.env.STORAGE ?? 'local';
if (mode !== 'local' && mode !== 'gcs') throw new Error('STORAGE must be local or gcs');
if (mode === 'gcs' && !process.env.GCS_BUCKET) throw new Error('Set GCS_BUCKET');
const store = mode === 'gcs' ? gcsStore(process.env.GCS_BUCKET!) : localStore(process.env.DATA_DIR ?? '.data');
const port = Number(process.env.PORT ?? '8080');
const server = serve({ fetch: createApp(store).fetch, port, hostname: process.env.HOST ?? '0.0.0.0' }, () => {
  console.log(`llm-session-share listening on port ${port} (${mode})`);
});
for (const signal of ['SIGTERM', 'SIGINT']) process.on(signal, () => server.close());
