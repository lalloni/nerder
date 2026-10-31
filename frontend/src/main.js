import './style.css';
import {Install, Licenses, List, Preview, Snippet, Uninstall} from '../wailsjs/go/app/App';

const $ = (sel) => document.querySelector(sel);
const cards = $('#cards');
const query = {text: '', installedOnly: false, mono: '', license: ''};
let snippetHTML = '';
let seq = 0; // guards against out-of-order List responses

const esc = (s) => s.replace(/[&<>"]/g, (c) => ({'&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;'}[c]));

const store = {
    get(key) { try { return localStorage.getItem(key); } catch { return null; } },
    set(key, value) { try { localStorage.setItem(key, value); } catch { /* storage unavailable */ } },
};

// --- layout mode ---------------------------------------------------------
function setMode(mode) {
    cards.className = mode;
    for (const b of document.querySelectorAll('.segmented button')) {
        b.setAttribute('aria-pressed', String(b.dataset.mode === mode));
    }
    store.set('mode', mode);
}

// --- font loading (lazy, one request per family) --------------------------
const loaded = new Map(); // folder -> Promise<PreviewInfo>

function ensureFont(folder) {
    if (!loaded.has(folder)) {
        const p = (async () => {
            const info = await Preview(folder);
            const faces = [new FontFace(info.fontFamily, `url("${info.regularUrl}")`, {weight: '400'})];
            if (info.boldUrl) faces.push(new FontFace(info.fontFamily, `url("${info.boldUrl}")`, {weight: '700'}));
            await Promise.all(faces.map((f) => f.load()));
            faces.forEach((f) => document.fonts.add(f));
            return info;
        })();
        loaded.set(folder, p);
        p.catch(() => loaded.delete(folder)); // allow Retry
    }
    return loaded.get(folder);
}

async function loadPreview(card) {
    const pre = card.querySelector('.snippet');
    card.querySelector('.card-error')?.remove();
    pre.dataset.state = 'loading';
    try {
        const info = await ensureFont(card.dataset.folder);
        pre.style.fontFamily = `"${info.fontFamily}", monospace`;
        pre.dataset.state = 'ready';
    } catch (err) {
        pre.dataset.state = 'error';
        const msg = document.createElement('div');
        msg.className = 'card-error';
        msg.innerHTML = `${esc(String(err))} <button class="btn" type="button">Retry</button>`;
        msg.querySelector('button').addEventListener('click', () => loadPreview(card));
        pre.before(msg);
    }
}

const observer = new IntersectionObserver((entries) => {
    for (const e of entries) {
        if (!e.isIntersecting) continue;
        observer.unobserve(e.target);
        loadPreview(e.target);
    }
}, {rootMargin: '200px'});

// --- cards ---------------------------------------------------------------
function cardHTML(f) {
    const spacing = f.monospaced ? 'monospaced' : 'proportional';
    return `
    <header class="card-head">
        <div class="titles">
            <h2>${esc(f.name)}</h2>
            <div class="card-meta">${esc(f.license)} · v${esc(f.version)} · ${spacing}</div>
        </div>
        <span class="badges"></span>
        <span class="actions"></span>
    </header>
    <p class="card-desc">${esc(f.description)}</p>
    <pre class="snippet" data-state="idle" title="Click to expand"><code>${snippetHTML}</code></pre>`;
}

function setStatus(card, status) {
    card.dataset.status = status;
    const badges = card.querySelector('.badges');
    const actions = card.querySelector('.actions');
    badges.innerHTML = status === 'user' ? '<span class="badge">Installed</span>'
        : status === 'system' ? '<span class="badge">Installed (system)</span>' : '';
    actions.innerHTML = status === 'none' ? '<button class="btn primary" type="button">Install</button>'
        : status === 'user' ? '<button class="btn" type="button">Uninstall</button>' : '';
    actions.querySelector('button')?.addEventListener('click', (e) => act(card, e.currentTarget));
}

function showNote(text) {
    const note = $('#note');
    note.textContent = text;
    note.hidden = false;
    clearTimeout(showNote.timer);
    showNote.timer = setTimeout(() => { note.hidden = true; }, 12000);
}

async function act(card, button) {
    const installing = card.dataset.status === 'none';
    const label = button.textContent;
    button.disabled = true; // a double-click cannot fire twice
    button.textContent = installing ? 'Installing…' : 'Removing…';
    card.querySelector('.card-error')?.remove();
    try {
        const view = await (installing ? Install : Uninstall)(card.dataset.folder);
        setStatus(card, view.status);
        if (installing) showNote('Installed. Some terminals and editors need a restart to pick up the new font.');
        if (!installing && query.installedOnly && view.status === 'none') card.remove(), updateCount();
    } catch (err) {
        const text = String(err);
        const msg = document.createElement('div');
        msg.className = 'card-error';
        if (text.includes('cache refresh')) {
            // the files changed; only fc-cache failed
            msg.textContent = `${text}. Run "fc-cache -f" in a terminal to finish.`;
            await syncStatuses();
        } else {
            msg.textContent = text;
            button.disabled = false;
            button.textContent = label;
        }
        card.querySelector('.card-head').after(msg);
    }
}

async function syncStatuses() {
    const all = await List({text: '', installedOnly: false, mono: '', license: ''});
    for (const f of all) {
        const card = cards.querySelector(`[data-folder="${CSS.escape(f.folder)}"]`);
        if (card) setStatus(card, f.status);
    }
}

function updateCount() {
    const n = cards.children.length;
    $('#count').textContent = `${n} ${n === 1 ? 'font' : 'fonts'}`;
    $('#empty').hidden = n > 0;
}

async function refresh() {
    const mine = ++seq;
    const fams = await List(query);
    if (mine !== seq) return;
    cards.replaceChildren(...fams.map((f) => {
        const card = document.createElement('article');
        card.className = 'card';
        card.dataset.folder = f.folder;
        card.innerHTML = cardHTML(f);
        setStatus(card, f.status);
        card.querySelector('.snippet').addEventListener('click', (e) => e.currentTarget.classList.toggle('expanded'));
        observer.observe(card);
        // families already loaded earlier render at once
        if (loaded.has(f.folder)) loadPreview(card);
        return card;
    }));
    updateCount();
}

// --- startup -------------------------------------------------------------
async function main() {
    const spans = await Snippet();
    snippetHTML = spans.map((s) => `<span class="tok-${s.class}">${esc(s.text)}</span>`).join('');

    for (const l of await Licenses()) {
        $('#license').append(new Option(l, l));
    }
    let timer;
    $('#search').addEventListener('input', (e) => {
        clearTimeout(timer);
        timer = setTimeout(() => { query.text = e.target.value.trim(); refresh(); }, 120);
    });
    $('#installed-only').addEventListener('click', (e) => {
        query.installedOnly = !query.installedOnly;
        e.currentTarget.setAttribute('aria-pressed', String(query.installedOnly));
        refresh();
    });
    $('#mono').addEventListener('change', (e) => { query.mono = e.target.value; refresh(); });
    $('#license').addEventListener('change', (e) => { query.license = e.target.value; refresh(); });
    for (const b of document.querySelectorAll('.segmented button')) {
        b.addEventListener('click', () => setMode(b.dataset.mode));
    }
    setMode(store.get('mode') === 'grid' ? 'grid' : 'list');
    await refresh();
}

main();
