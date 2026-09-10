import {
  AddProduct,
  DeleteDocument,
  ExportPDF,
  GetDocument,
  GetSettings,
  ListFurnizori,
  ListProducts,
  NewDocumentDraft,
  SaveDocument,
  showError,
} from '../api';
import type { Document, Product, Rand } from '../api';
import { totaluri, valoriRand } from '../calc';
import { showAlert, showConfirm } from '../dialog';
import { cautaProduse } from '../fuzzy';
import { UM_PERMISE, randDinProdus, randGol, validareDocument } from '../nota';
import { formatDateRO, formatLei, formatNumber, formatProcent, parseDateRO, parseNumber } from '../format';
import { navigate } from '../router';
import { escapeHtml } from '../sidebar';
import { showToast } from '../toast';

export async function renderDocumentView(
  outlet: HTMLElement,
  id: string | undefined,
  refreshSidebar: () => Promise<void>,
): Promise<void> {
  let doc: Document;
  let produse: Product[];
  let furnizori: string[];
  let cotaImplicita: number;

  try {
    const setari = await GetSettings();
    cotaImplicita = setari.cotaTva;
    [produse, furnizori] = await Promise.all([ListProducts(), ListFurnizori()]);
    doc = id === undefined ? await NewDocumentDraft() : await GetDocument(Number(id));
  } catch (err) {
    showError('Nu s-a putut încărca nota', err);
    outlet.innerHTML = '<p class="empty">Nota nu a putut fi încărcată.</p>';
    return;
  }

  render();

  function render(): void {
    outlet.innerHTML = `
      <h1>${doc.id === 0 ? 'Notă de recepție nouă' : `Notă de recepție NR ${doc.nr}`}</h1>

      <div class="header-grid">
        <div class="field">
          <label for="f-unitate">Unitatea</label>
          <input id="f-unitate" value="${escapeHtml(doc.unitate)}" readonly />
        </div>
        <div class="field">
          <label for="f-nr">nr.</label>
          <input id="f-nr" class="num" type="number" min="1" step="1" value="${doc.nr}" />
        </div>
        <div class="field">
          <label for="f-data">din data</label>
          <input id="f-data" type="text" inputmode="numeric" maxlength="10"
                 placeholder="ZZ/LL/AAAA" value="${escapeHtml(formatDateRO(doc.data))}" />
        </div>
      </div>

      <h2>Document livrare</h2>
      <div class="header-grid">
        <div class="field">
          <label for="f-livrare">Document livrare</label>
          <input id="f-livrare" placeholder="Factură, Aviz…" value="${escapeHtml(doc.documentLivrare)}" />
        </div>
        <div class="field">
          <label for="f-livrare-nr">Nr.</label>
          <input id="f-livrare-nr" value="${escapeHtml(doc.documentLivrareNr)}" />
        </div>
        <div class="field">
          <label for="f-livrare-data">Data</label>
          <input id="f-livrare-data" type="text" inputmode="numeric" maxlength="10"
                 placeholder="ZZ/LL/AAAA" value="${escapeHtml(formatDateRO(doc.documentLivrareData))}" />
        </div>
        <div class="field">
          <label for="f-furnizor">Furnizorul</label>
          <input id="f-furnizor" list="lista-furnizori" value="${escapeHtml(doc.furnizor)}" />
          <datalist id="lista-furnizori">
            ${furnizori.map((f) => `<option value="${escapeHtml(f)}"></option>`).join('')}
          </datalist>
        </div>
      </div>

      <h2>Produse</h2>
      ${tabelProduse()}
      <div class="table-actions">
        <button class="btn" id="add-rand">+ Adaugă rând</button>
      </div>

      ${panouTotaluri()}

      <div class="btn-row">
        <button class="btn btn-primary" id="save">Salvează</button>
        ${doc.id === 0 ? '' : '<button class="btn" id="print">Printează (PDF)</button>'}
        ${doc.id === 0 ? '' : '<button class="btn btn-danger" id="delete">Șterge</button>'}
      </div>
    `;

    wireEvents();
    recompute();
  }

  /**
   * The product table. Cota T.V.A., Adaos and Adaos % are working columns:
   * they exist here and not on the printed form. The derived cells carry no
   * inputs — they are recomputed from the row on every keystroke.
   */
  function tabelProduse(): string {
    const randuri = doc.randuri
      .map(
        (r, i) => `
        <tr data-rand="${i}">
          <td class="nr-crt">${i + 1}</td>
          <td class="combo">
            <input class="denumire" data-camp="denumire" value="${escapeHtml(r.denumire)}"
                   autocomplete="off" />
          </td>
          <td>
            <select data-camp="um">
              ${UM_PERMISE.map(
                (um) => `<option value="${um}" ${r.um === um ? 'selected' : ''}>${um}</option>`,
              ).join('')}
            </select>
          </td>
          <td><input class="num" data-camp="cantitate" value="${formatNumber(r.cantitate)}" /></td>
          <td><input class="num" data-camp="pretFaraTva" value="${formatNumber(r.pretFaraTva)}" /></td>
          <td><input class="num cota" data-camp="cotaTva" value="${formatNumber(r.cotaTva)}" /></td>
          <td class="derivat" data-derivat="valoareFaraTva"></td>
          <td class="derivat" data-derivat="valoareCuTva"></td>
          <td><input class="num" data-camp="pretVanzare" value="${formatNumber(r.pretVanzare)}" /></td>
          <td class="derivat" data-derivat="valoareVanzare"></td>
          <td class="derivat" data-derivat="adaos"></td>
          <td class="derivat" data-derivat="adaosProcent"></td>
          <td><button class="btn-icon sterge-rand" title="Șterge rândul">×</button></td>
        </tr>`,
      )
      .join('');

    return `
      <table class="tabel-produse">
        <thead>
          <tr>
            <th>Nr. crt.</th>
            <th>Denumirea</th>
            <th>U/M</th>
            <th class="num">Cantitatea</th>
            <th class="num">Preț fără T.V.A.</th>
            <th class="num cota">Cota T.V.A. %</th>
            <th class="num">Valoare fără T.V.A.</th>
            <th class="num">Valoare cu T.V.A.</th>
            <th class="num">Preț de vânzare</th>
            <th class="num">Valoare la preț de vânzare</th>
            <th class="num">Adaos</th>
            <th class="num">Adaos %</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          ${randuri || '<tr><td colspan="13" class="empty">Nicio linie. Apasă „+ Adaugă rând".</td></tr>'}
        </tbody>
      </table>`;
  }

  /**
   * The totals block. It sits under the table as labelled cells rather than as
   * a row inside it: the columns it sums are far apart, and a TOTAL row leaves
   * the reader counting columns to find out which figure is which.
   *
   * There is no quantity total — U/M varies from row to row.
   */
  function panouTotaluri(): string {
    return `
      <div class="totaluri">
        <div class="celula">
          <span class="eticheta">Total valoare fără T.V.A.</span>
          <span class="valoare" id="t-fara-tva">—</span>
        </div>
        <div class="celula">
          <span class="eticheta">Total valoare cu T.V.A.</span>
          <span class="valoare" id="t-cu-tva">—</span>
        </div>
        <div class="celula">
          <span class="eticheta">Total valoare la preț de vânzare</span>
          <span class="valoare" id="t-vanzare">—</span>
        </div>
        <div class="celula">
          <span class="eticheta">Total adaos</span>
          <span class="valoare" id="t-adaos">—</span>
        </div>
        <div class="celula">
          <span class="eticheta">Adaos %</span>
          <span class="valoare" id="t-adaos-procent">—</span>
        </div>
      </div>`;
  }

  /**
   * Recomputes every derived cell and the totals block from `doc`. It runs on
   * every keystroke, so it only writes text into existing cells — it never
   * rebuilds the table, which would take the focus out of the field being
   * typed into.
   */
  function recompute(): void {
    doc.randuri.forEach((r, i) => {
      const rand = outlet.querySelector(`tr[data-rand="${i}"]`);
      if (rand === null) return;
      const v = valoriRand(r);
      scrieDerivat(rand, 'valoareFaraTva', formatLei(v.valoareFaraTva));
      scrieDerivat(rand, 'valoareCuTva', formatLei(v.valoareCuTva));
      scrieDerivat(rand, 'valoareVanzare', formatLei(v.valoareVanzare));
      scrieDerivat(rand, 'adaos', formatLei(v.adaos), v.adaos < 0);
      scrieDerivat(rand, 'adaosProcent', formatProcent(v.adaosProcent),
        v.adaosProcent !== undefined && v.adaosProcent < 0);
    });

    const t = totaluri(doc.randuri);
    setText('t-fara-tva', formatLei(t.valoareFaraTva));
    setText('t-cu-tva', formatLei(t.valoareCuTva));
    setText('t-vanzare', formatLei(t.valoareVanzare));
    setText('t-adaos', formatLei(t.adaos), t.adaos < 0);
    setText('t-adaos-procent', formatProcent(t.adaosProcent),
      t.adaosProcent !== undefined && t.adaosProcent < 0);
  }

  function scrieDerivat(rand: Element, camp: string, text: string, negativ = false): void {
    const celula = rand.querySelector(`[data-derivat="${camp}"]`);
    if (celula === null) return;
    celula.textContent = text;
    celula.classList.toggle('negativ', negativ);
  }

  function setText(id: string, text: string, negativ = false): void {
    const el = outlet.querySelector(`#${id}`);
    if (el === null) return;
    el.textContent = text;
    el.classList.toggle('negativ', negativ);
  }

  function wireEvents(): void {
    outlet.querySelector<HTMLInputElement>('#f-nr')!.addEventListener('input', (e) => {
      doc.nr = Number((e.target as HTMLInputElement).value);
    });
    outlet.querySelector<HTMLInputElement>('#f-data')!.addEventListener('input', (e) => {
      const iso = parseDateRO((e.target as HTMLInputElement).value);
      // A half-typed date is left alone rather than written as garbage; the
      // save-time validation is what refuses it, with a message.
      if (iso !== undefined) doc.data = iso;
    });
    legaText('#f-livrare', (v) => (doc.documentLivrare = v));
    legaText('#f-livrare-nr', (v) => (doc.documentLivrareNr = v));
    legaText('#f-furnizor', (v) => (doc.furnizor = v));
    outlet.querySelector<HTMLInputElement>('#f-livrare-data')!.addEventListener('input', (e) => {
      const text = (e.target as HTMLInputElement).value;
      const iso = parseDateRO(text);
      if (text.trim() === '') doc.documentLivrareData = '';
      else if (iso !== undefined) doc.documentLivrareData = iso;
    });

    outlet.querySelectorAll<HTMLElement>('tr[data-rand]').forEach((rand) => {
      const index = Number(rand.dataset.rand);
      rand.querySelectorAll<HTMLInputElement | HTMLSelectElement>('[data-camp]').forEach((camp) => {
        camp.addEventListener('input', () => {
          aplicaCamp(index, camp.dataset.camp!, camp.value);
          recompute();
        });
      });
      rand.querySelector('.sterge-rand')!.addEventListener('click', () => {
        doc.randuri.splice(index, 1);
        render();
      });

      const denumire = rand.querySelector<HTMLInputElement>('input.denumire')!;
      denumire.addEventListener('input', () => deschideCombo(index, denumire));
      denumire.addEventListener('focus', () => deschideCombo(index, denumire));
      denumire.addEventListener('keydown', (e) => navigheazaCombo(e, index, denumire));
      // Blur closes on the next tick so a click on a suggestion lands before
      // the list is removed; a click removes it itself, so nothing flickers.
      // The close is conditional on comboRand still being this row: focus can
      // move straight from this field into another row's, whose own focus
      // handler has already opened its list by the time this timer fires —
      // an unconditional close would tear that fresh list down instead.
      denumire.addEventListener('blur', () =>
        window.setTimeout(() => {
          if (comboRand === index) inchideCombo();
        }, 150),
      );
    });

    outlet.querySelector<HTMLButtonElement>('#add-rand')!.addEventListener('click', () => {
      doc.randuri.push(randGol(cotaImplicita));
      render();
      // The new row's name field is where typing continues.
      const inputuri = outlet.querySelectorAll<HTMLInputElement>('input.denumire');
      inputuri[inputuri.length - 1]?.focus();
    });

    outlet.querySelector<HTMLButtonElement>('#save')!.addEventListener('click', salveaza);
    outlet.querySelector<HTMLButtonElement>('#print')?.addEventListener('click', printeaza);
    outlet.querySelector<HTMLButtonElement>('#delete')?.addEventListener('click', sterge);
  }

  function legaText(selector: string, seteaza: (v: string) => void): void {
    outlet.querySelector<HTMLInputElement>(selector)!.addEventListener('input', (e) => {
      seteaza((e.target as HTMLInputElement).value);
    });
  }

  /**
   * Writes one typed field back onto its row.
   *
   * Editing the name by hand detaches the row from its product: what is on the
   * row no longer says what the catalogue says, and keeping the link would
   * make the row look like a product it is not. Picking from the dropdown (see
   * the combobox task) re-attaches it.
   */
  function aplicaCamp(index: number, camp: string, valoare: string): void {
    const r = doc.randuri[index] as unknown as Record<string, unknown>;
    switch (camp) {
      case 'denumire':
        r.denumire = valoare;
        r.productId = undefined;
        break;
      case 'um':
        r.um = valoare;
        break;
      default:
        r[camp] = parseNumber(valoare);
    }
  }

  // The index of the row whose suggestion list is open, and which entry is
  // highlighted in it. They live here rather than on the DOM because the list
  // is rebuilt on every keystroke and would lose the highlight otherwise.
  let comboRand: number | undefined;
  let comboEvidentiat = 0;

  /** Draws (or redraws) the suggestion list under one row's name field. */
  function deschideCombo(index: number, input: HTMLInputElement): void {
    inchideCombo();
    comboRand = index;
    // Rebuilding the list (any keystroke, or opening it fresh on focus)
    // always resets the highlight to the first entry. Arrow navigation moves
    // it afterwards by mutating classes directly in navigheazaCombo, without
    // going back through here — so this reset never fights arrow-key use; it
    // only stops a re-typed search from keeping an old index that now points
    // at a different, re-sorted product.
    comboEvidentiat = 0;

    const potriviri = cautaProduse(produse, input.value);
    const exact = produse.some(
      (p) => p.denumire.trim().toLowerCase() === input.value.trim().toLowerCase(),
    );
    const poateFiSalvat = input.value.trim() !== '' && !exact;

    if (potriviri.length === 0 && !poateFiSalvat) return;

    const lista = document.createElement('ul');
    lista.className = 'combo-lista';
    lista.innerHTML =
      potriviri
        .map(
          (p, i) => `
        <li data-produs="${p.id}" class="${i === comboEvidentiat ? 'evidentiat' : ''}">
          ${escapeHtml(p.denumire)}<span class="produs-um">${escapeHtml(p.um)}</span>
        </li>`,
        )
        .join('') +
      (poateFiSalvat
        ? `<li class="combo-nou" data-nou="1">„${escapeHtml(input.value.trim())}" nu este în produse —
             <strong>salvează-l</strong></li>`
        : '');

    // mousedown, not click: the field's blur fires first on a click, and the
    // handler would then be running against a list already being torn down.
    lista.addEventListener('mousedown', (e) => {
      e.preventDefault();
      const li = (e.target as HTMLElement).closest('li');
      if (li === null) return;
      if (li.dataset.nou === '1') {
        void salveazaProdusNou(index, input);
        return;
      }
      const produs = produse.find((p) => p.id === Number(li.dataset.produs));
      if (produs !== undefined) alegeProdus(index, produs);
    });

    input.closest('td')!.appendChild(lista);
  }

  function inchideCombo(): void {
    outlet.querySelectorAll('.combo-lista').forEach((el) => el.remove());
    comboRand = undefined;
  }

  /** Arrow keys move the highlight, Enter takes it, Escape closes the list. */
  function navigheazaCombo(e: KeyboardEvent, index: number, input: HTMLInputElement): void {
    const lista = outlet.querySelector<HTMLUListElement>('.combo-lista');
    if (lista === null || comboRand !== index) return;
    const optiuni = lista.querySelectorAll<HTMLLIElement>('li[data-produs]');

    if (e.key === 'Escape') {
      inchideCombo();
      return;
    }
    if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
      e.preventDefault();
      if (optiuni.length === 0) return;
      comboEvidentiat =
        (comboEvidentiat + (e.key === 'ArrowDown' ? 1 : optiuni.length - 1)) % optiuni.length;
      optiuni.forEach((li, i) => li.classList.toggle('evidentiat', i === comboEvidentiat));
      return;
    }
    if (e.key === 'Enter' && optiuni.length > 0) {
      e.preventDefault();
      const produs = produse.find(
        (p) => p.id === Number(optiuni[comboEvidentiat].dataset.produs),
      );
      if (produs !== undefined) alegeProdus(index, produs);
    }
  }

  /**
   * Replaces a row with one started from the chosen product, keeping whatever
   * has already been typed into it. The quantity and the purchase price belong
   * to this delivery, not to the catalogue, so picking a product must not wipe
   * figures the user has already entered for them.
   */
  function alegeProdus(index: number, produs: Product): void {
    const vechi = doc.randuri[index];
    const nou = randDinProdus(produs);
    nou.cantitate = vechi.cantitate;
    nou.pretFaraTva = vechi.pretFaraTva;
    doc.randuri[index] = nou;
    inchideCombo();
    render();
    // Typing continues in the field that was just filled in.
    outlet.querySelectorAll<HTMLInputElement>('input.denumire')[index]?.focus();
  }

  /**
   * Files the typed name as a new product and attaches the row to it.
   *
   * The unit, the selling price and the rate come from the row as it stands:
   * the user has just typed them, and asking for them again in Setări would be
   * asking twice for the same answer.
   */
  async function salveazaProdusNou(index: number, input: HTMLInputElement): Promise<void> {
    const r = doc.randuri[index];
    try {
      const produs = await AddProduct({
        id: 0,
        denumire: input.value.trim(),
        um: r.um,
        pretVanzare: r.pretVanzare,
        cotaTva: r.cotaTva,
        ordine: 0,
      } as unknown as Product);
      produse = await ListProducts();
      doc.randuri[index] = { ...r, productId: produs.id, denumire: produs.denumire } as Rand;
      inchideCombo();
      render();
      showToast(`„${produs.denumire}" a fost adăugat în produse.`);
    } catch (err) {
      showError('Nu s-a putut salva produsul', err);
    }
  }

  async function salveaza(): Promise<void> {
    const dataTastata = outlet.querySelector<HTMLInputElement>('#f-data')!.value;
    const problema = validareDocument(doc, dataTastata);
    if (problema !== undefined) {
      await showAlert(problema);
      return;
    }
    try {
      doc = await SaveDocument(doc);
    } catch (err) {
      showError('Nu s-a putut salva nota', err);
      return;
    }
    showToast('Nota a fost salvată.');
    await refreshSidebar();
    navigate(`#/document/${doc.id}`);
  }

  async function printeaza(): Promise<void> {
    try {
      const path = await ExportPDF(doc.id);
      // An empty path is the user cancelling the save dialog, which needs no
      // confirmation of its own.
      if (path !== '') showToast('PDF salvat.');
    } catch (err) {
      showError('Nu s-a putut genera PDF-ul', err);
    }
  }

  async function sterge(): Promise<void> {
    if (!(await showConfirm(`Ștergi nota de recepție NR ${doc.nr}?`))) return;
    try {
      await DeleteDocument(doc.id);
    } catch (err) {
      showError('Nu s-a putut șterge nota', err);
      return;
    }
    await refreshSidebar();
    navigate('#/document/new');
  }
}
