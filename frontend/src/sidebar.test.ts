import { describe, expect, it } from 'vitest';
import { currentHash, escapeHtml } from './sidebar';

describe('escapeHtml', () => {
  it('escapeaza si ghilimelele, nu doar unghiularele', () => {
    // Aproape toate apelurile pun rezultatul intr-un atribut HTML, deci o
    // ghilimea neescapata ar rupe atributul.
    expect(escapeHtml('a "b" <c> & \'d\'')).toBe('a &quot;b&quot; &lt;c&gt; &amp; &#39;d&#39;');
  });

  it('escapeaza ampersandul intai, ca sa nu dubleze secventele', () => {
    expect(escapeHtml('&lt;')).toBe('&amp;lt;');
  });
});

describe('currentHash', () => {
  it('cade pe ruta documentului nou cand hash-ul e gol', () => {
    expect(currentHash('')).toBe('#/document/new');
  });

  it('lasa neatins un hash existent', () => {
    expect(currentHash('#/setari')).toBe('#/setari');
  });
});
