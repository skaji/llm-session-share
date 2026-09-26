import { test } from 'node:test';
import assert from 'node:assert/strict';
import { mkdtemp, readFile, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { createApp } from '../src/app.js';
import { localStore } from '../src/storage/local.js';
import { sessionKey } from '../src/storage/store.js';

const id = '01a0de0e-f326-7ec1-bc06-0aee7bbdf319';
const snapshot = {
  id, source: 'codex', name: 'skaji', title: 'Session title',
  messages: [{ role: 'user', text: 'Hello 日本語 <script>bad()</script>' }],
};

test('snapshot upload, durable reload, conditional fetch, and replacement', async t => {
  const directory = await mkdtemp(join(tmpdir(), 'llm-session-share-'));
  t.after(() => rm(directory, { recursive: true, force: true }));
  const app = createApp(localStore(directory));
  const endpoint = '/api/sessions/' + id;
  assert.equal((await app.request(endpoint)).status, 404);
  const put = (data: unknown) => app.request(endpoint, {
    method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(data),
  });
  assert.equal((await put({ ...snapshot, raw_tools: 'excluded', messages: [{ ...snapshot.messages[0], secret_metadata: 'excluded' }] })).status, 204);
  const disk = await readFile(join(directory, sessionKey(id)), 'utf8');
  assert(!disk.includes('excluded'));
  assert.equal(JSON.parse(disk).name, 'skaji');
  const restarted = createApp(localStore(directory));
  const response = await restarted.request(endpoint);
  assert.equal(response.status, 200);
  assert.equal(response.headers.get('Cache-Control'), 'private, no-cache');
  const tag = response.headers.get('ETag')!;
  assert(tag);
  const first = await response.json() as typeof snapshot;
  assert.deepEqual(first.messages, snapshot.messages);
  assert.equal(first.title, snapshot.title);
  assert.equal((await restarted.request(endpoint, { headers: { 'If-None-Match': tag } })).status, 304);
  assert.equal((await put({ ...snapshot, title: 'Renamed' })).status, 204);
  const renamed = await restarted.request(endpoint, { headers: { 'If-None-Match': tag } });
  assert.equal(renamed.status, 200);
  assert.equal((await renamed.json() as typeof snapshot).title, 'Renamed');
  const updated = { ...snapshot, source: 'claude', name: 'friend', messages: [...snapshot.messages, { role: 'assistant', text: 'Answer' }] };
  assert.equal((await put(updated)).status, 204);
  const changed = await restarted.request(endpoint, { headers: { 'If-None-Match': tag } });
  assert.equal(changed.status, 200);
  assert.equal((await changed.json() as typeof snapshot).messages.length, 2);
  assert.equal((await put(updated)).status, 204);
  assert.equal((await (await app.request(endpoint)).json() as typeof snapshot).messages.length, 2);
  // The HTML shell does not interpolate untrusted message/name content.
  const page = await (await app.request('/session/' + id)).text();
  assert(!page.includes('<script>bad()'));
  assert(page.includes('/app.js'));
  const latestTag = (await app.request(endpoint)).headers.get('ETag')!;
  assert.equal((await app.request(endpoint, { method: 'DELETE' })).status, 204);
  assert.equal((await restarted.request(endpoint, { headers: { 'If-None-Match': latestTag } })).status, 404);
  await assert.rejects(readFile(join(directory, sessionKey(id))), { code: 'ENOENT' });
  assert.equal((await app.request(endpoint, { method: 'DELETE' })).status, 204);
  assert.equal((await put(snapshot)).status, 204);
  assert.equal((await restarted.request(endpoint)).status, 200);
});

test('reject invalid updates before touching storage', async () => {
  let writes = 0;
  const app = createApp({ get: async () => null, put: async () => { writes++; }, delete: async () => { writes++; } });
  for (const data of [null, {}, { ...snapshot, id: 'different' }, { ...snapshot, name: '' }, { ...snapshot, title: 123 }, { ...snapshot, messages: [{ role: 'tool', text: 'no' }] }]) {
    const response = await app.request('/api/sessions/' + id, { method: 'PUT', body: JSON.stringify(data) });
    assert.equal(response.status, 400);
  }
  assert.equal((await app.request('/api/sessions/invalid')).status, 400);
  assert.equal((await app.request('/api/sessions/invalid', { method: 'DELETE' })).status, 400);
  assert.throws(() => sessionKey('../anything'));
  assert.equal(writes, 0);
});
