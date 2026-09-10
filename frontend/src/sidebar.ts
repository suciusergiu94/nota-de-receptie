import { DocumentSummary, ListDocuments, showError } from './api';
import { showConfirm } from './dialog';
import { ciornaAreRanduriNesalvate } from './draft';
import { formatDateRO } from './format';
import { navigate } from './router';

/** The route an unsaved document lives at. */
export const DRAFT_HASH = '#/document/new';

let hashchangeListenerRegistered = false;

/**
 * The hash to match sidebar entries against. A freshly launched app has an
 * empty hash and the router falls back to the new-document route, so the same
 * fallback applies here — otherwise no entry would be highlighted on the one
 * screen the app always opens on.
 */
export function currentHash(hash: string = window.location.hash): string {
  return hash || DRAFT_HASH;
}

/** Renders the sidebar: new-document button, history, settings link. */
export async function renderSidebar(el: HTMLElement): Promise<void> {
  let documente: DocumentSummary[];
  try {
    documente = await ListDocuments();
  } catch (err) {
    showError('Nu s-a putut încărca lista de documente', err);
    documente = [];
  }

  const items = documente
    .map(
      (doc) => `
        <li>
          <a class="doc-link" href="#/document/${doc.id}">
            <span class="doc-nr">NR ${doc.nr}</span>
            <span class="doc-meta">${escapeHtml(formatDateRO(doc.data))}${
              doc.furnizor ? ` — ${escapeHtml(doc.furnizor)}` : ''
            }</span>
          </a>
        </li>`,
    )
    .join('');

  // The draft entry is rendered every pass and shown or hidden by markActive,
  // so moving in and out of the draft route only toggles an attribute instead
  // of rebuilding a list that costs a round trip to SQLite. It sits above the
  // saved documents because the list is newest-first and an unsaved document
  // is newer than all of them.
  el.innerHTML = `
    <div class="new-doc-wrap">
      <button class="btn btn-primary" id="new-doc">+ Notă nouă</button>
    </div>
    <ul class="doc-list">
      <li id="draft-item" hidden>
        <a class="doc-link" id="draft-link" href="${DRAFT_HASH}">
          <span class="doc-nr">Notă nouă</span>
          <span class="doc-meta draft-meta">Nesalvată</span>
        </a>
      </li>
      ${items}
    </ul>
    <a class="settings-link" href="#/setari">Setări</a>
  `;

  el.querySelector<HTMLButtonElement>('#new-doc')!.addEventListener('click', () => {
    void notaNoua();
  });

  markActive(el);

  // Registered once, against window, which outlives any single render.
  if (!hashchangeListenerRegistered) {
    hashchangeListenerRegistered = true;
    window.addEventListener('hashchange', () => markActive(el));
  }
}

/**
 * Starts a new note, asking first if that would throw one away.
 *
 * Pressing the button while already on the draft route re-renders it in
 * place, and the draft lives only in the form: a dozen typed rows go with one
 * stray click, and the sidebar entry saying "Nesalvată" reads as though
 * something were being kept. Only the draft route has anything to lose — a
 * saved note is on disk, and from any other route the draft is already gone —
 * so nothing is asked anywhere else, and an untouched empty draft is not
 * worth a question either.
 */
async function notaNoua(): Promise<void> {
  if (currentHash() === DRAFT_HASH && ciornaAreRanduriNesalvate()) {
    const continua = await showConfirm(
      'Nota nouă nu este salvată. Începi alta și pierzi rândurile scrise?',
    );
    if (!continua) return;
  }
  navigate(DRAFT_HASH);
}

function markActive(el: HTMLElement): void {
  const hash = currentHash();
  el.querySelector('#draft-item')?.toggleAttribute('hidden', hash !== DRAFT_HASH);
  el.querySelectorAll('a').forEach((link) => {
    link.classList.toggle('active', link.getAttribute('href') === hash);
  });
}

/**
 * Escapes text that goes into an innerHTML template. Almost every call site
 * places the result inside an HTML attribute (`value="${escapeHtml(...)}"`),
 * so quotes must be escaped too — a quote in user data would otherwise break
 * out of the attribute, corrupting the value or injecting an attribute of its
 * own. Ampersand goes first so the other replacements' `&...;` sequences are
 * not re-escaped.
 */
export function escapeHtml(value: string): string {
  return value
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}
