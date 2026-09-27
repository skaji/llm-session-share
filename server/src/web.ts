export const html = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>LLM Session Share</title>
  <link rel="icon" href="/favicon.svg" type="image/svg+xml">
  <link rel="stylesheet" href="/app.css">
  <script src="/app.js" defer></script>
</head>
<body>
  <main>
    <header class="masthead"><a href="/">LLM Session Share</a><span>Shared conversations</span></header>
    <section id="welcome" hidden>
      <h1>A conversation, shared.</h1>
      <p class="subtle">Read Codex and Claude Code conversations as new messages arrive.</p>
      <p class="subtle">Download the CLI for your OS and architecture from <a href="https://github.com/skaji/llm-session-share/releases">GitHub Releases</a>, extract the archive, and put <code>llm-session-share</code> on your PATH.</p>
      <p class="subtle">Then, to share a conversation, run:</p>
      <pre id="share-command" class="example">llm-session-share -server &lt;server-url&gt; -user &lt;your-name&gt; /path/to/session.jsonl</pre>
    </section>
    <section id="session" hidden>
      <div class="heading"><div><h1 id="title">Shared conversation</h1><div class="meta"><span id="owner"></span><span id="source" class="badge"></span><span id="count"></span></div></div></div>
      <div class="session-info"><code id="identity"></code><span id="updated"></span></div>
      <div class="toolbar"><p id="status" role="status" aria-live="polite">Loading conversation…</p><div><label><input id="markdown" type="checkbox"> Markdown</label><label><input id="live" type="checkbox"> Live updates</label><button id="latest" type="button">Latest ↓</button></div></div>
      <div id="messages"></div>
      <p id="empty" class="empty" hidden>No conversation messages yet.</p>
    </section>
  </main>
</body>
</html>`;

export const css = `
:root { color-scheme: light; font-family: ui-sans-serif, system-ui, sans-serif; color: #20201e; background: #fff; }
* { box-sizing: border-box; }
body { margin: 0; }
main { width: 100%; max-width: 1028px; margin: 0 auto; padding: 26px 24px 72px; }
a { color: #2056ab; text-decoration: none; }
a:hover { text-decoration: underline; }
.masthead { display: flex; justify-content: space-between; gap: 16px; padding-bottom: 22px; margin-bottom: 34px; border-bottom: 1px solid #dde3eb; font-size: 14px; }
.masthead a { color: #20201e; font-weight: 650; }
.masthead span, .subtle { color: #66645f; font-size: 14px; }
#welcome h1, .heading h1 { margin: 0 0 12px; font-size: 25px; font-weight: 650; line-height: 1.4; overflow-wrap: anywhere; }
.heading { display: flex; align-items: flex-start; justify-content: space-between; gap: 24px; }
.heading > div { min-width: 0; }
button { flex-shrink: 0; padding: 8px 12px; border: 1px solid #c8c5bd; border-radius: 6px; background: #fff; color: #20201e; font: inherit; font-size: 13px; cursor: pointer; }
button:hover { background: #f5f6f8; }
button:focus-visible, input:focus-visible, a:focus-visible { outline: 2px solid #2056ab; outline-offset: 3px; }
.meta { display: flex; align-items: center; flex-wrap: wrap; gap: 8px 14px; color: #66645f; font-size: 13px; }
#owner { font-weight: 650; color: #20201e; overflow-wrap: anywhere; }
.badge { padding: 2px 7px; border-radius: 999px; background: #ebe9e3; color: #5d5a55; font-size: 12px; }
.badge-codex { background: #e2efff; color: #185787; }
.badge-claude { background: #f8e6df; color: #85452f; }
.session-info { display: flex; flex-wrap: wrap; gap: 8px 20px; margin: 12px 0 28px; color: #77736c; font-size: 12px; overflow-wrap: anywhere; }
.toolbar { display: flex; justify-content: space-between; align-items: center; gap: 12px; padding: 12px 0; color: #66645f; font-size: 12px; }
.toolbar p { margin: 0; }
.toolbar p:empty { display: none; }
.toolbar > div { display: flex; flex-wrap: wrap; justify-content: flex-end; align-items: center; margin-left: auto; gap: 10px 14px; }
.toolbar label { display: flex; align-items: center; gap: 5px; cursor: pointer; }
.toolbar input { accent-color: #2056ab; }
.toolbar button { padding: 5px 9px; font-size: 12px; }
.message { min-width: 0; margin: 24px 0; padding: 16px 0; }
.message-user { width: fit-content; max-width: 85%; margin-left: auto; padding: 18px 22px; border-radius: 20px; background: #e8f3ff; }
.message-source { margin: 13px 0 0; color: #24231f; white-space: pre-wrap; overflow-wrap: anywhere; font: 14px/1.75 ui-monospace, SFMono-Regular, Menlo, monospace; tab-size: 4; }
#messages:not(.raw-view) .message-source, #messages.raw-view .markdown { display: none; }
.markdown { min-width: 0; margin-top: 13px; color: #24231f; font-size: 14px; line-height: 1.75; overflow-wrap: anywhere; }
.markdown > :first-child { margin-top: 0; }
.markdown > :last-child { margin-bottom: 0; }
.markdown :is(h1, h2, h3, h4, h5, h6) { margin: 24px 0 10px; font-weight: 650; line-height: 1.45; }
.markdown h1 { font-size: 21px; }
.markdown h2 { font-size: 19px; }
.markdown h3 { font-size: 17px; }
.markdown :is(h4, h5, h6) { font-size: 15px; }
.markdown p { margin: 12px 0; }
.markdown :is(ul, ol) { margin: 12px 0; padding-left: 25px; }
.markdown li + li { margin-top: 4px; }
.markdown li > :is(ul, ol) { margin: 4px 0; }
.markdown code { padding: 2px 5px; border-radius: 4px; background: #f0f2f5; font: 0.92em/1.6 ui-monospace, SFMono-Regular, Menlo, monospace; }
.markdown pre { max-width: 100%; margin: 14px 0; padding: 14px; overflow-x: auto; white-space: pre; overflow-wrap: normal; background: #f6f7f9; border: 1px solid #e6e9ee; border-radius: 6px; tab-size: 4; }
.markdown pre code { padding: 0; background: none; border-radius: 0; }
.markdown blockquote { margin: 14px 0; padding: 1px 16px; border-left: 3px solid #cbd5e1; color: #66645f; }
.markdown hr { margin: 22px 0; border: 0; border-top: 1px solid #dde3eb; }
.markdown .table-scroll { max-width: 100%; margin: 14px 0; overflow-x: auto; }
.markdown table { min-width: 100%; width: max-content; border-collapse: collapse; font-size: 14px; }
.markdown :is(th, td) { min-width: 100px; max-width: 340px; padding: 8px 12px; border: 1px solid #dde3eb; text-align: left; vertical-align: top; }
.markdown th { background: #f6f7f9; font-weight: 650; }
.markdown [align="center"] { text-align: center; }
.markdown [align="right"] { text-align: right; }
.markdown img { max-width: 100%; height: auto; }
.empty { padding: 40px 0; border-top: 1px solid #dde3eb; color: #77736c; text-align: center; }
#welcome { max-width: 760px; margin: 60px auto; }
.example { white-space: pre-wrap; overflow-wrap: anywhere; padding: 16px; background: #f6f7f9; border-radius: 6px; font-size: 13px; line-height: 1.7; }
[hidden] { display: none !important; }
@media (max-width: 700px) {
  main { padding-left: 16px; padding-right: 16px; }
}
@media (max-width: 340px) {
  main { padding-left: 12px; padding-right: 12px; }
}
@media (max-width: 640px) {
  main { padding-top: 20px; }
  .masthead { margin-bottom: 26px; }
  .masthead span { display: none; }
  #welcome h1, .heading h1 { font-size: 21px; }
  .heading { gap: 12px; }
  .toolbar { align-items: flex-start; flex-direction: column; }
  .message-source { font-size: 13px; }
  .message-user { max-width: 95%; padding: 14px 16px; border-radius: 16px; }
  #welcome { margin-top: 32px; }
}
`;

export const browserJS = String.raw`
(() => {
  const $ = id => document.getElementById(id);
  const match = location.pathname.match(/^\/session\/([0-9a-f-]+)$/);
  if (!match) {
    $('welcome').hidden = false;
    $('share-command').textContent = 'llm-session-share -server ' + location.origin + ' -user <your-name> /path/to/session.jsonl';
    return;
  }
  const id = match[1];
  $('session').hidden = false;
  $('identity').textContent = id;
  let previous = [];
  let tag = '';
  let loaded = false;
  let busy = false;
  let pollTimer;
  const setFormat = () => $('messages').classList.toggle('raw-view', !$('markdown').checked);
  const readOptions = () => {
    const params = new URLSearchParams(location.search);
    for (const key of ['markdown', 'live']) $(key).checked = params.get(key) === '1';
    setFormat();
  };
  readOptions();
  window.addEventListener('popstate', () => { readOptions(); syncLive(true); });
  for (const key of ['markdown', 'live']) {
    $(key).addEventListener('change', () => {
      const url = new URL(location.href);
      for (const option of ['markdown', 'live']) url.searchParams.set(option, $(option).checked ? '1' : '0');
      history.replaceState(null, '', url);
      setFormat();
      if (key === 'live') syncLive(true);
    });
  }
  const formatDate = value => {
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) return '';
    const pad = value => String(value).padStart(2, '0');
    const offset = -date.getTimezoneOffset();
    const zone = (offset < 0 ? '-' : '+') + pad(Math.floor(Math.abs(offset) / 60)) + ':' + pad(Math.abs(offset) % 60);
    return date.getFullYear() + '-' + pad(date.getMonth() + 1) + '-' + pad(date.getDate())
      + ' ' + pad(date.getHours()) + ':' + pad(date.getMinutes()) + ':' + pad(date.getSeconds()) + ' ' + zone;
  };
  const latest = () => window.scrollTo({ top: document.documentElement.scrollHeight, behavior: 'smooth' });
  $('latest').addEventListener('click', latest);
  function render(snapshot) {
    const nearBottom = document.documentElement.scrollHeight - (window.scrollY + window.innerHeight) < 180;
    const wasLoaded = loaded;
    const messages = snapshot.messages;
    const first = messages.find(m => m.role === 'user');
    const title = snapshot.title || (first ? Array.from(first.text.trim().split('\n')[0]).slice(0, 100).join('') : 'Shared conversation');
    $('title').textContent = title;
    document.title = title + ' · LLM Session Share';
    $('owner').textContent = snapshot.name;
    $('source').textContent = snapshot.source === 'codex' ? 'Codex' : 'Claude Code';
    $('source').className = 'badge badge-' + snapshot.source;
    $('count').textContent = messages.length + (messages.length === 1 ? ' message' : ' messages');
    $('updated').textContent = 'Last shared ' + formatDate(snapshot.updated_at);
    let common = 0;
    while (common < previous.length && common < messages.length && JSON.stringify(previous[common]) === JSON.stringify(messages[common])) common++;
    const container = $('messages');
    while (container.children.length > common) container.lastElementChild.remove();
    for (let i = common; i < messages.length; i++) {
      const message = messages[i];
      const article = document.createElement('article');
      article.className = 'message' + (message.role === 'user' ? ' message-user' : '');
      const meta = document.createElement('div');
      meta.className = 'meta';
      const timestamp = document.createElement('time');
      timestamp.textContent = message.timestamp ? formatDate(message.timestamp) : '';
      if (message.timestamp) timestamp.dateTime = message.timestamp;
      meta.append(timestamp);
      const text = document.createElement('pre');
      text.className = 'message-source';
      text.textContent = message.text;
      const formatted = document.createElement('div');
      formatted.className = 'markdown';
      // HTML is generated by the server's safe Markdown renderer, never accepted from uploaders.
      if (typeof message.html === 'string') formatted.innerHTML = message.html;
      else formatted.textContent = message.text;
      for (const table of formatted.querySelectorAll('table')) {
        const wrapper = document.createElement('div');
        wrapper.className = 'table-scroll';
        wrapper.tabIndex = 0;
        wrapper.setAttribute('role', 'region');
        wrapper.setAttribute('aria-label', 'Table (scroll horizontally)');
        table.replaceWith(wrapper);
        wrapper.append(table);
      }
      article.append(meta, text, formatted);
      container.append(article);
    }
    const changed = common !== previous.length || common !== messages.length;
    previous = messages;
    loaded = true;
    $('empty').hidden = messages.length !== 0;
    if (wasLoaded && changed && nearBottom && $('live').checked) latest();
  }
  async function poll(initial = false) {
    if (busy || (!initial && (document.hidden || !$('live').checked))) return;
    busy = true;
    try {
      const response = await fetch('/api/sessions/' + id + '?render=markdown', {
        headers: tag ? { 'If-None-Match': tag } : {},
        cache: 'no-store',
        signal: AbortSignal.timeout(15000)
      });
      if (response.status === 304) {
        $('status').textContent = '';
        return;
      }
      if (response.status === 404) {
        tag = '';
        previous = [];
        $('messages').replaceChildren();
        $('title').textContent = 'Shared conversation';
        document.title = 'LLM Session Share';
        for (const id of ['owner', 'source', 'count', 'updated']) $(id).textContent = '';
        $('empty').hidden = true;
        $('status').textContent = (loaded ? 'Session removed.' : 'Session not shared yet.')
          + ($('live').checked ? ' Waiting for an upload…' : ' Reload or enable Live updates to check again.');
        return;
      }
      if (!response.ok || !response.headers.get('Content-Type')?.includes('application/json')) throw new Error('Unable to check for updates. Retrying; reload if sign-in is needed.');
      const snapshot = await response.json();
      render(snapshot);
      tag = response.headers.get('ETag') || '';
      $('status').textContent = '';
    } catch (error) {
      $('status').textContent = $('live').checked ? 'Updates disconnected. Retrying…' : 'Unable to load the session. Reload to retry.';
    } finally {
      busy = false;
    }
  }
  function syncLive(fetchNow = false) {
    clearInterval(pollTimer);
    if ($('live').checked) {
      pollTimer = setInterval(() => poll(), 2000);
      if (fetchNow) poll();
    }
    if (fetchNow) $('status').textContent = '';
  }
  poll(true);
  syncLive();
  document.addEventListener('visibilitychange', () => poll());
})();
`;
