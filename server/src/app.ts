import { Hono } from 'hono';
import { bodyLimit } from 'hono/body-limit';
import { sessionKey, validID, type Store } from './storage/store.js';
import { html, css, browserJS } from './web.js';
import { renderMarkdown } from './markdown.js';
import { favicon } from './favicon.js';

type Message = { role: 'user' | 'assistant'; text: string; timestamp?: string };
type Snapshot = { id: string; source: 'codex' | 'claude'; name: string; title?: string; messages: Message[] };

function parseSnapshot(value: unknown, id: string): Snapshot | null {
  if (!value || typeof value !== 'object') return null;
  const s = value as Record<string, unknown>;
  if (s.id !== id || (s.source !== 'codex' && s.source !== 'claude') ||
      (s.title !== undefined && typeof s.title !== 'string') ||
      typeof s.name !== 'string' || !s.name.trim() || !Array.isArray(s.messages)) return null;
  const messages: Message[] = [];
  for (const value of s.messages) {
    if (!value || typeof value !== 'object') return null;
    const m = value as Record<string, unknown>;
    if ((m.role !== 'user' && m.role !== 'assistant') || typeof m.text !== 'string' ||
        (m.timestamp !== undefined && typeof m.timestamp !== 'string')) return null;
    messages.push({ role: m.role, text: m.text, ...(m.timestamp ? { timestamp: m.timestamp as string } : {}) });
  }
  // Store only the public conversation schema, even if a client sends raw fields.
  return { id, source: s.source, name: s.name.trim(), ...(typeof s.title === 'string' && s.title.trim() ? { title: s.title.trim() } : {}), messages };
}

async function etag(body: string): Promise<string> {
  const digest = await crypto.subtle.digest('SHA-256', new TextEncoder().encode(body));
  return '"' + Array.from(new Uint8Array(digest), b => b.toString(16).padStart(2, '0')).join('') + '"';
}

export function createApp(store: Store) {
  const app = new Hono();
  app.use('*', async (c, next) => {
    c.header('Cache-Control', 'private, no-cache');
    c.header('X-Content-Type-Options', 'nosniff');
    c.header('Referrer-Policy', 'no-referrer');
    c.header('Content-Security-Policy', "default-src 'self'; script-src 'self'; style-src 'self'; connect-src 'self'; img-src 'self'; base-uri 'none'; frame-ancestors 'none'");
    await next();
  });
  app.onError((error, c) => {
    console.error(error);
    return c.json({ error: 'Storage or server error; please retry.' }, 500);
  });
  app.get('/healthz', c => c.text('ok'));
  app.get('/favicon.svg', c => { c.header('Content-Type', 'image/svg+xml'); return c.body(favicon); });
  app.get('/', c => c.html(html));
  app.get('/app.css', c => { c.header('Content-Type', 'text/css; charset=utf-8'); return c.body(css); });
  app.get('/app.js', c => { c.header('Content-Type', 'text/javascript; charset=utf-8'); return c.body(browserJS); });
  app.get('/session/:id', c => validID(c.req.param('id')) ? c.html(html) : c.notFound());
  app.get('/api/sessions/:id', async c => {
    const id = c.req.param('id');
    if (!validID(id)) return c.json({ error: 'Invalid session ID.' }, 400);
    const body = await store.get(sessionKey(id));
    if (body === null) return c.json({ error: 'Session not shared yet.' }, 404);
    const markdown = c.req.query('render') === 'markdown';
    const tag = await etag((markdown ? 'markdown-v1\n' : '') + body);
    c.header('ETag', tag);
    const candidates = c.req.header('If-None-Match')?.split(',').map(s => s.trim().replace(/^W\//, '')) ?? [];
    if (candidates.includes(tag) || candidates.includes('*')) return c.body(null, 304);
    if (markdown) {
      const snapshot = JSON.parse(body) as Snapshot & { updated_at: string };
      return c.json({
        ...snapshot,
        messages: snapshot.messages.map((message, index) => ({
          ...message, html: renderMarkdown(message.text, index),
        })),
      });
    }
    c.header('Content-Type', 'application/json; charset=utf-8');
    return c.body(body);
  });
  app.put('/api/sessions/:id', bodyLimit({ maxSize: 32 * 1024 * 1024 }), async c => {
    const id = c.req.param('id');
    if (!validID(id)) return c.json({ error: 'Invalid session ID.' }, 400);
    const data = await c.req.json().catch(() => null);
    const snapshot = parseSnapshot(data, id);
    if (!snapshot) return c.json({ error: 'Expected id, source, name, and user/assistant messages.' }, 400);
    await store.put(sessionKey(id), JSON.stringify({ ...snapshot, updated_at: new Date().toISOString() }));
    return c.body(null, 204);
  });
  app.delete('/api/sessions/:id', async c => {
    const id = c.req.param('id');
    if (!validID(id)) return c.json({ error: 'Invalid session ID.' }, 400);
    await store.delete(sessionKey(id));
    return c.body(null, 204);
  });
  return app;
}
