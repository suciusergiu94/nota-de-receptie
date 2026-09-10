import './style.css';
import { renderSidebar } from './sidebar';
import { startRouter } from './router';
import { renderDocumentView } from './views/document';
// Task 15 creates views/setari.ts and restores this import along with the
// '#/setari' route below.
// import { renderSetariView } from './views/setari';

document.querySelector<HTMLDivElement>('#app')!.innerHTML = `
  <aside class="sidebar" id="sidebar"></aside>
  <main class="main" id="outlet"></main>
`;

const sidebar = document.getElementById('sidebar') as HTMLElement;
const outlet = document.getElementById('outlet') as HTMLElement;

/** Re-reads the document history; called after any save or delete. */
export function refreshSidebar(): Promise<void> {
  return renderSidebar(sidebar);
}

void refreshSidebar();

startRouter(
  [
    {
      pattern: /^#\/document\/new$/,
      render: (el) => renderDocumentView(el, undefined, refreshSidebar),
    },
    {
      pattern: /^#\/document\/(\d+)$/,
      render: (el, id) => renderDocumentView(el, id, refreshSidebar),
    },
    // Task 15 restores this route alongside the import above.
    // {
    //   pattern: /^#\/setari$/,
    //   render: (el) => renderSetariView(el, refreshSidebar),
    // },
  ],
  outlet,
);
