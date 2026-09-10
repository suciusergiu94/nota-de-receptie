// Modal dialogs drawn by the page itself.
//
// window.confirm() and window.alert() are dead ends here: the macOS webview
// Wails builds on sets a WKUIDelegate that never implements the JavaScript
// panel methods, so the browser draws nothing and confirm() returns false
// straight away. A "Șterge" guarded by window.confirm therefore looked like a
// button that did nothing at all. These dialogs are ordinary elements, so they
// behave the same on every platform the app is built for.

/**
 * Builds a modal <dialog> around `message` and the given buttons, and resolves
 * with the returnValue of the button pressed. Closing with Esc leaves the
 * returnValue empty, which callers read as "no".
 */
function openModal(
  message: string,
  buttons: HTMLButtonElement[],
  continut?: HTMLElement,
): Promise<string> {
  return new Promise((resolve) => {
    const dialog = document.createElement('dialog');
    dialog.className = 'modal';

    const text = document.createElement('p');
    text.className = 'modal-text';
    // textContent, not innerHTML: the message can carry a document number or
    // an error string straight from Go, neither of which is escaped.
    text.textContent = message;

    const row = document.createElement('div');
    row.className = 'modal-buttons';
    buttons.forEach((button) => row.appendChild(button));

    dialog.appendChild(text);
    if (continut !== undefined) dialog.appendChild(continut);
    dialog.appendChild(row);
    document.body.appendChild(dialog);

    // One dialog per call rather than a reused element, so a dialog opened
    // from another dialog's handler cannot inherit the previous one's state.
    dialog.addEventListener('close', () => {
      const value = dialog.returnValue;
      dialog.remove();
      resolve(value);
    });

    dialog.showModal();
  });
}

/** A button that closes its dialog with `value` when pressed. */
function modalButton(label: string, value: string, className: string): HTMLButtonElement {
  const button = document.createElement('button');
  button.className = className;
  button.textContent = label;
  button.addEventListener('click', () => {
    const dialog = button.closest('dialog');
    dialog?.close(value);
  });
  return button;
}

/**
 * Asks `message` and resolves true only if the user picks "Da".
 *
 * "Nu" comes first in the DOM so showModal() lands the initial focus there:
 * the callers are destructive actions, and Enter on a freshly opened dialog
 * should back out, not go through with it.
 */
export function showConfirm(message: string): Promise<boolean> {
  const no = modalButton('Nu', 'nu', 'btn');
  const yes = modalButton('Da', 'da', 'btn btn-danger');
  return openModal(message, [no, yes]).then((value) => value === 'da');
}

/** Shows `message` with a single dismiss button. */
export function showAlert(message: string): Promise<void> {
  const ok = modalButton('OK', 'ok', 'btn btn-primary');
  return openModal(message, [ok]).then(() => undefined);
}

/** One choice in a picker: what it returns, and the two lines it shows. */
export interface OptiunePicker {
  valoare: string;
  eticheta: string;
  detaliu: string;
}

/**
 * A picker that searches instead of listing everything.
 *
 * `cauta` is asked for the options each time the text changes, including once
 * with "" for the list the dialog opens on — so what "no search yet" shows is
 * the caller's decision, not a rule baked in here.
 */
export interface CautarePicker {
  placeholder: string;
  cauta(text: string): OptiunePicker[];
  /** Shown in place of the list when the search matches nothing. */
  faraRezultate: string;
}

/** One option, as the button that closes the dialog with its value. */
function optiuneButton(o: OptiunePicker): HTMLButtonElement {
  const button = modalButton('', o.valoare, 'btn modal-optiune');
  const eticheta = document.createElement('span');
  eticheta.className = 'modal-optiune-eticheta';
  eticheta.textContent = o.eticheta;
  const detaliu = document.createElement('span');
  detaliu.className = 'modal-optiune-detaliu';
  detaliu.textContent = o.detaliu;
  button.append(eticheta, detaliu);
  return button;
}

/** Fills `lista` with `optiuni`, or with the empty-handed sentence. */
function deseneazaOptiuni(lista: HTMLElement, optiuni: OptiunePicker[], gol: string): void {
  lista.textContent = '';
  if (optiuni.length === 0) {
    const nimic = document.createElement('p');
    nimic.className = 'modal-fara-rezultate';
    nimic.textContent = gol;
    lista.appendChild(nimic);
    return;
  }
  optiuni.forEach((o) => lista.appendChild(optiuneButton(o)));
}

/**
 * Asks the user to pick one of `optiuni`, and resolves with its `valoare`, or
 * undefined if they backed out. Esc and "Renunță" both close with an empty
 * returnValue, which is the same answer either way.
 *
 * Passing a CautarePicker instead of a fixed list puts a search box above the
 * options and redraws them as the user types.
 */
export function showPicker(
  message: string,
  optiuni: OptiunePicker[] | CautarePicker,
): Promise<string | undefined> {
  const cautare = Array.isArray(optiuni) ? undefined : optiuni;

  const continut = document.createElement('div');
  continut.className = 'modal-picker';

  const lista = document.createElement('div');
  lista.className = 'modal-lista';

  if (cautare === undefined) {
    deseneazaOptiuni(lista, optiuni as OptiunePicker[], '');
  } else {
    const camp = document.createElement('input');
    camp.type = 'search';
    camp.className = 'modal-cautare';
    camp.placeholder = cautare.placeholder;
    // The dialog opens with the caret here, so a number can be typed straight
    // away without reaching for the mouse first.
    camp.autofocus = true;

    const redeseneaza = (): void =>
      deseneazaOptiuni(lista, cautare.cauta(camp.value), cautare.faraRezultate);

    camp.addEventListener('input', redeseneaza);
    camp.addEventListener('keydown', (e) => {
      // Enter takes the first option: after typing a number, the one wanted is
      // almost always the only one left, and reaching for it with the mouse to
      // confirm that is a step for nothing.
      if (e.key !== 'Enter') return;
      e.preventDefault();
      lista.querySelector<HTMLButtonElement>('button')?.click();
    });

    redeseneaza();
    continut.appendChild(camp);
  }

  continut.appendChild(lista);

  const renunta = modalButton('Renunță', '', 'btn');
  return openModal(message, [renunta], continut).then((v) => (v === '' ? undefined : v));
}
