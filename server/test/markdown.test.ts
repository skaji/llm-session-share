import { test } from 'node:test';
import assert from 'node:assert/strict';
import { renderMarkdown } from '../src/markdown.js';
import { createApp } from '../src/app.js';

test('render headings, GFM tables, lists, and fenced code', () => {
  const html = renderMarkdown('# Heading\n\n| Name | State |\n| --- | --- |\n| skaji | **Ready** |\n\n- One\n- Two\n\n```ts\nconst x = "<tag>";\n```', 0);
  for (const fragment of ['<h1>Heading</h1>', '<table>', '<th>Name</th>', '<strong>Ready</strong>', '<li>One</li>', '<pre><code class="language-ts">', '&lt;tag&gt;']) {
    assert(html.includes(fragment), fragment);
  }
});

test('conversation HTML and dangerous links cannot become executable markup', () => {
  const html = renderMarkdown('<script>alert(1)</script>\n\n<style>body { display: none }</style>\n\n<img src=x onerror=alert(1)>\n\n[bad](javascript:alert%281%29)\n\n[also bad](data:text/html,hello)\n\n[safe](https://example.com)', 0);
  assert(!/<(?:script|style|img)\b/i.test(html));
  assert(html.includes('&lt;script&gt;'));
  assert(!/href="(?:javascript|data):/i.test(html));
  assert(html.includes('href="https://example.com"'));
});

test('footnote IDs do not collide between messages', () => {
  const markdown = 'Text[^1]\n\n[^1]: A note';
  assert(renderMarkdown(markdown, 0).includes('id="message-0-fn-1"'));
  assert(renderMarkdown(markdown, 1).includes('id="message-1-fn-1"'));
});

test('rendered API retains raw text, uses a separate ETag, and ignores uploaded HTML', async () => {
  const id = '01a0de0e-f326-7ec1-bc06-0aee7bbdf319';
  let stored = '';
  const app = createApp({ get: async () => stored, put: async (_, body) => { stored = body; }, delete: async () => { stored = ''; } });
  const rawURL = '/api/sessions/' + id;
  const renderedURL = rawURL + '?render=markdown';
  await app.request(rawURL, {
    method: 'PUT', body: JSON.stringify({
      id, name: 'skaji', source: 'codex', messages: [
        { role: 'user', text: '# Actual text', html: '<script>injected</script>' },
      ],
    }),
  });
  assert(!stored.includes('injected'));
  assert(!stored.includes('<h1>'));
  const raw = await app.request(rawURL);
  const rawTag = raw.headers.get('ETag')!;
  const response = await app.request(renderedURL, { headers: { 'If-None-Match': rawTag } });
  assert.equal(response.status, 200);
  assert.notEqual(response.headers.get('ETag'), rawTag);
  const result = await response.json() as { messages: { text: string; html: string }[] };
  assert.equal(result.messages[0].text, '# Actual text');
  assert.equal(result.messages[0].html, '<h1>Actual text</h1>');
  const unchanged = await app.request(renderedURL, { headers: { 'If-None-Match': response.headers.get('ETag')! } });
  assert.equal(unchanged.status, 304);
});
