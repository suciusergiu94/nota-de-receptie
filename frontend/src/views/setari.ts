import {
  DeleteFurnizor,
  GetSettings,
  ListFurnizori,
  ListProducts,
  SaveProducts,
  SaveSettings,
  showError,
} from '../api';
import type { Product, Settings } from '../api';
import { showAlert, showConfirm } from '../dialog';
import { UM_PERMISE } from '../nota';
import { formatNumber, parseNumber } from '../format';
import { escapeHtml } from '../sidebar';
import { showToast } from '../toast';

export async function renderSetariView(
  outlet: HTMLElement,
  refreshSidebar: () => Promise<void>,
): Promise<void> {
  let setari: Settings;
  let produse: Product[];
  let furnizori: string[];

  try {
    [setari, produse, furnizori] = await Promise.all([
      GetSettings(),
      ListProducts(),
      ListFurnizori(),
    ]);
  } catch (err) {
    showError('Nu s-au putut încărca setările', err);
    outlet.innerHTML = '<p class="empty">Setările nu au putut fi încărcate.</p>';
    return;
  }

  render();

  function render(): void {
    outlet.innerHTML = `
      <h1>Setări</h1>

      <h2>Unitatea</h2>
      <div class="header-grid">
        <div class="field">
          <label for="s-unitate">Numele unității</label>
          <input id="s-unitate" value="${escapeHtml(setari.unitateNume)}" />
        </div>
        <div class="field">
          <label for="s-nr">Numărul următoarei note</label>
          <input id="s-nr" class="num" type="number" min="1" step="1" value="${setari.nextNr}" />
        </div>
        <div class="field">
          <label for="s-cota">Cota T.V.A. implicită (%)</label>
          <input id="s-cota" class="num" value="${formatNumber(setari.cotaTva)}" />
        </div>
      </div>

      <h2>Produse</h2>
      <p class="empty">
        Modificările de aici nu schimbă notele deja salvate: fiecare notă
        păstrează denumirea, U/M, prețul de vânzare și cota pe care produsul
        le avea când a fost salvată.
      </p>
      ${tabelProduse()}
      <div class="table-actions">
        <button class="btn" id="add-produs">+ Adaugă produs</button>
      </div>

      <h2>Furnizori memorați</h2>
      ${listaFurnizori()}

      <div class="btn-row">
        <button class="btn btn-primary" id="save">Salvează</button>
      </div>
    `;
    wireEvents();
  }

  function tabelProduse(): string {
    const randuri = produse
      .map(
        (p, i) => `
        <tr data-produs="${i}">
          <td>${i + 1}</td>
          <td><input data-camp="denumire" value="${escapeHtml(p.denumire)}" /></td>
          <td>
            <select data-camp="um">
              ${UM_PERMISE.map(
                (um) => `<option value="${um}" ${p.um === um ? 'selected' : ''}>${um}</option>`,
              ).join('')}
            </select>
          </td>
          <td><input class="num" data-camp="pretVanzare" value="${formatNumber(p.pretVanzare)}" /></td>
          <td><input class="num cota" data-camp="cotaTva" value="${formatNumber(p.cotaTva)}" /></td>
          <td>
            <button class="btn-icon muta-sus" title="Mută mai sus" ${i === 0 ? 'disabled' : ''}>↑</button>
            <button class="btn-icon muta-jos" title="Mută mai jos"
                    ${i === produse.length - 1 ? 'disabled' : ''}>↓</button>
            <button class="btn-icon sterge-produs" title="Șterge produsul">×</button>
          </td>
        </tr>`,
      )
      .join('');

    return `
      <table>
        <thead>
          <tr>
            <th>Nr.</th>
            <th>Denumirea</th>
            <th>U/M</th>
            <th class="num">Preț de vânzare</th>
            <th class="num cota">Cota T.V.A. %</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          ${randuri || '<tr><td colspan="6" class="empty">Niciun produs.</td></tr>'}
        </tbody>
      </table>`;
  }

  function listaFurnizori(): string {
    if (furnizori.length === 0) {
      return '<p class="empty">Niciun furnizor memorat încă.</p>';
    }
    return `
      <ul class="doc-list">
        ${furnizori
          .map(
            (f) => `
          <li class="furnizor-rand">
            <span>${escapeHtml(f)}</span>
            <button class="btn-icon sterge-furnizor" data-furnizor="${escapeHtml(f)}"
                    title="Șterge sugestia">×</button>
          </li>`,
          )
          .join('')}
      </ul>`;
  }

  function wireEvents(): void {
    outlet.querySelector<HTMLInputElement>('#s-unitate')!.addEventListener('input', (e) => {
      setari.unitateNume = (e.target as HTMLInputElement).value;
    });
    outlet.querySelector<HTMLInputElement>('#s-nr')!.addEventListener('input', (e) => {
      setari.nextNr = Number((e.target as HTMLInputElement).value);
    });
    outlet.querySelector<HTMLInputElement>('#s-cota')!.addEventListener('input', (e) => {
      setari.cotaTva = parseNumber((e.target as HTMLInputElement).value);
    });

    outlet.querySelectorAll<HTMLElement>('tr[data-produs]').forEach((rand) => {
      const i = Number(rand.dataset.produs);
      rand.querySelectorAll<HTMLInputElement | HTMLSelectElement>('[data-camp]').forEach((camp) => {
        camp.addEventListener('input', () => {
          const p = produse[i] as unknown as Record<string, unknown>;
          const nume = camp.dataset.camp!;
          p[nume] = nume === 'denumire' || nume === 'um' ? camp.value : parseNumber(camp.value);
        });
      });
      rand.querySelector('.muta-sus')!.addEventListener('click', () => muta(i, -1));
      rand.querySelector('.muta-jos')!.addEventListener('click', () => muta(i, 1));
      rand.querySelector('.sterge-produs')!.addEventListener('click', () => {
        produse.splice(i, 1);
        render();
      });
    });

    outlet.querySelector<HTMLButtonElement>('#add-produs')!.addEventListener('click', () => {
      produse.push({
        id: 0, denumire: '', um: 'Kg.', pretVanzare: 0, cotaTva: setari.cotaTva, ordine: produse.length,
      } as unknown as Product);
      render();
      const inputuri = outlet.querySelectorAll<HTMLInputElement>('[data-camp="denumire"]');
      inputuri[inputuri.length - 1]?.focus();
    });

    outlet.querySelectorAll<HTMLButtonElement>('.sterge-furnizor').forEach((btn) => {
      btn.addEventListener('click', () => void stergeFurnizor(btn.dataset.furnizor!));
    });

    outlet.querySelector<HTMLButtonElement>('#save')!.addEventListener('click', salveaza);
  }

  function muta(index: number, directie: number): void {
    const tinta = index + directie;
    if (tinta < 0 || tinta >= produse.length) return;
    [produse[index], produse[tinta]] = [produse[tinta], produse[index]];
    render();
  }

  /**
   * Deleting a supplier only forgets a suggestion. Documents keep the name as
   * text, so nothing that has been filed changes — which is why this needs no
   * warning beyond the confirmation.
   */
  async function stergeFurnizor(nume: string): Promise<void> {
    if (!(await showConfirm(`Ștergi sugestia „${nume}"?`))) return;
    try {
      await DeleteFurnizor(nume);
      furnizori = await ListFurnizori();
    } catch (err) {
      showError('Nu s-a putut șterge furnizorul', err);
      return;
    }
    render();
  }

  async function salveaza(): Promise<void> {
    if (setari.nextNr < 1) {
      await showAlert('Numărul următoarei note trebuie să fie cel puțin 1.');
      return;
    }
    const gol = produse.findIndex((p) => p.denumire.trim() === '');
    if (gol !== -1) {
      await showAlert(`Produsul ${gol + 1} nu are denumire.`);
      return;
    }
    const duplicat = primulDuplicat(produse);
    if (duplicat !== undefined) {
      await showAlert(`Denumirea „${duplicat}" apare de două ori în listă.`);
      return;
    }

    try {
      await SaveSettings(setari);
      await SaveProducts(produse);
      produse = await ListProducts();
    } catch (err) {
      showError('Nu s-au putut salva setările', err);
      return;
    }
    showToast('Setările au fost salvate.');
    await refreshSidebar();
    render();
  }
}

/**
 * The first name that appears twice, ignoring case — the store refuses these,
 * and saying which one it is here is more useful than relaying a constraint
 * violation from SQLite.
 */
function primulDuplicat(produse: Product[]): string | undefined {
  const vazute = new Set<string>();
  for (const p of produse) {
    const cheie = p.denumire.trim().toLowerCase();
    if (vazute.has(cheie)) return p.denumire.trim();
    vazute.add(cheie);
  }
  return undefined;
}
