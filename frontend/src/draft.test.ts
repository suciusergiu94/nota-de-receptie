import { describe, expect, it } from 'vitest';
import { ciornaAreRanduriNesalvate, publicaStareaCiornei } from './draft';

describe('starea ciornei', () => {
  it('spune ca nu e nimic de pierdut cat timp nicio vedere n-a publicat nimic', () => {
    // Bara laterala intreaba si inainte sa se fi desenat vreo nota; raspunsul
    // prudent e "nimic de pierdut", ca sa nu apara o intrebare din senin.
    expect(ciornaAreRanduriNesalvate()).toBe(false);
  });

  it('citeste starea formularului aflat pe ecran', () => {
    let randuri = 0;
    publicaStareaCiornei(() => randuri > 0);
    expect(ciornaAreRanduriNesalvate()).toBe(false);
    randuri = 3;
    expect(ciornaAreRanduriNesalvate()).toBe(true);
  });

  it('pastreaza raspunsul ultimei vederi desenate', () => {
    publicaStareaCiornei(() => true);
    publicaStareaCiornei(() => false);
    expect(ciornaAreRanduriNesalvate()).toBe(false);
  });
});
