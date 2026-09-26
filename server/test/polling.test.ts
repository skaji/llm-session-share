import { test } from 'node:test';
import assert from 'node:assert/strict';
import { runInNewContext } from 'node:vm';
import { setImmediate } from 'node:timers/promises';
import { browserJS } from '../src/web.js';

function browser(search = '') {
  const listeners = new Map<string, () => void>();
  const elements = new Map<string, { checked: boolean; textContent: string; hidden: boolean; classList: { toggle(): void }; replaceChildren(): void; addEventListener(event: string, callback: () => void): void }>();
  const element = (id: string) => {
    if (!elements.has(id)) elements.set(id, {
      checked: false, textContent: '', hidden: false,
      classList: { toggle() {} }, replaceChildren() {},
      addEventListener(event, callback) { listeners.set(id + ':' + event, callback); },
    });
    return elements.get(id)!;
  };
  let url = new URL('https://share.example.com/session/01a0de0e-f326-7ec1-bc06-0aee7bbdf319' + search);
  let requests = 0;
  let timerID = 0;
  const timers = new Map<number, () => void>();
  const document = {
    hidden: false, title: '', getElementById: element,
    addEventListener(event: string, callback: () => void) { listeners.set(event, callback); },
  };
  runInNewContext(browserJS, {
    URL, URLSearchParams, document,
    location: { get href() { return url.href; }, get search() { return url.search; }, get pathname() { return url.pathname; } },
    history: { replaceState(_state: unknown, _title: string, value: URL) { url = new URL(value); } },
    window: { addEventListener(event: string, callback: () => void) { listeners.set(event, callback); } },
    fetch: async () => { requests++; return { status: 404 }; },
    AbortSignal: { timeout: () => undefined },
    setInterval(callback: () => void, delay: number) { assert.equal(delay, 2000); timers.set(++timerID, callback); return timerID; },
    clearInterval(id: number) { timers.delete(id); },
  });
  return { element, listeners, timers, document, requests: () => requests, url: () => url };
}

test('Live updates off loads once; toggling starts and stops API requests', async () => {
  const b = browser();
  await setImmediate();
  assert.equal(b.requests(), 1);
  assert.equal(b.timers.size, 0);
  assert(!b.element('status').textContent.includes('Waiting'));
  b.listeners.get('visibilitychange')!();
  assert.equal(b.requests(), 1);

  b.element('live').checked = true;
  b.listeners.get('live:change')!();
  await setImmediate();
  assert.equal(b.requests(), 2);
  assert.equal(b.url().searchParams.get('live'), '1');
  assert.equal(b.timers.size, 1);
  const tick = [...b.timers.values()][0];
  tick();
  await setImmediate();
  assert.equal(b.requests(), 3);

  b.element('live').checked = false;
  b.listeners.get('live:change')!();
  assert.equal(b.timers.size, 0);
  tick(); // Even an already queued timer must not fetch after disabling Live updates.
  b.listeners.get('visibilitychange')!();
  b.element('markdown').checked = true;
  b.listeners.get('markdown:change')!();
  assert.equal(b.requests(), 3);
  assert.equal(b.url().searchParams.get('live'), '0');
});

test('live=1 starts polling from the URL and skips hidden-tab requests', async () => {
  const b = browser('?live=1');
  await setImmediate();
  assert.equal(b.requests(), 1);
  assert.equal(b.timers.size, 1);
  b.document.hidden = true;
  [...b.timers.values()][0]();
  assert.equal(b.requests(), 1);
  b.document.hidden = false;
  b.listeners.get('visibilitychange')!();
  await setImmediate();
  assert.equal(b.requests(), 2);
});
