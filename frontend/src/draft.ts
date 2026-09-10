// Whether the unsaved note on screen still has anything in it.
//
// The document view is the only place that can answer this: the form's state
// lives inside its closure and nothing outside can see it. The button that
// would throw that state away — "+ Notă nouă" — lives in the sidebar, which
// cannot import the view (the view already imports escapeHtml from the
// sidebar, so the two would form a cycle). Hence this middle: the view
// publishes the answer, the sidebar asks for it.

let raspunde: () => boolean = () => false;

/**
 * Called by the document view whenever it takes over the screen, with a
 * predicate that reads its own form. The last view to render wins, which is
 * right — only one document view is on screen at a time.
 */
export function publicaStareaCiornei(citeste: () => boolean): void {
  raspunde = citeste;
}

/** True when starting a new note would lose rows the user has typed. */
export function ciornaAreRanduriNesalvate(): boolean {
  return raspunde();
}
