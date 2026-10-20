// Direct icon rendering must match the original sprite paint, including local
// clip paths, at both theme inks and with explicit two-tone colours.
import { readFileSync } from 'node:fs';
import { expect } from '@playwright/test';
import { ICON_NAMES } from '../../web/src/lib/icons/names';
import { test } from '../fixtures/base';

// Paint comparison has a fixed pixel scale; touch behavior keeps the project's
// device settings and is covered by the ordinary tooltip and animation specs.
test.use({ deviceScaleFactor: 1 });

const symbols = ICON_NAMES.map((name) => {
  const raw = readFileSync(
    new URL(`../../web/src/lib/icons/svg/${name}.svg`, import.meta.url),
    'utf8'
  );
  const viewBox = /viewBox="([^"]+)"/.exec(raw)?.[1];
  const inner = raw.slice(raw.indexOf('>') + 1, raw.lastIndexOf('</svg>'));
  return `<symbol id="reference-${name}" viewBox="${viewBox}">${inner}</symbol>`;
}).join('');

test('original icons retain their pixels and local clipping @mobile', async ({ page }, info) => {
  await page.addInitScript(() => localStorage.setItem('ajayeb:name', 'Tester'));
  await page.setViewportSize({ width: info.project.name === 'mobile' ? 390 : 1280, height: 3000 });
  await page.goto('/dev/gallery');
  // Compare paint at a common multiple of the 12/24-unit source grids.
  // Fixed cells avoid differing fractional transforms in use shadow trees;
  // suppress the scrolling container's transient overlay scrollbar too.
  await page.addStyleTag({
    content: `
    main { scrollbar-width: none; }
    main > section:first-of-type > div { grid-template-columns: repeat(${info.project.name === 'mobile' ? 3 : 7}, 88px); }
    main > section:first-of-type svg { width: 48px; height: 48px; }
  `
  });
  const grid = page.locator('main > section').first();
  await expect(grid.locator('svg[data-icon]')).toHaveCount(ICON_NAMES.length);
  await page.evaluate(() => document.fonts.ready);
  await page.evaluate((symbols) => {
    const sprite = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
    sprite.style.cssText = 'position:absolute;width:0;height:0;overflow:hidden';
    sprite.innerHTML = symbols;
    document.body.append(sprite);
  }, symbols);

  for (const theme of ['light', 'dark']) {
    for (const twoTone of [false, true]) {
      await page.evaluate(
        ({ theme, twoTone }) => {
          document.documentElement.dataset.theme = theme;
          document.documentElement.classList.toggle('dark', theme === 'dark');
          const grid = document.querySelector<HTMLElement>('main > section')!;
          grid.style.setProperty('--i1', twoTone ? '#92470d' : 'currentColor');
          grid.style.setProperty('--i2', twoTone ? '#87c9d9' : 'transparent');
        },
        { theme, twoTone }
      );
      const actual = await grid.screenshot({ animations: 'disabled' });
      const originals = await grid.locator('svg[data-icon]').evaluateAll((icons) =>
        icons.map((svg) => {
          const original = { markup: svg.innerHTML, viewBox: svg.getAttribute('viewBox')! };
          svg.innerHTML = `<use href="#reference-${svg.getAttribute('data-icon')}"/>`;
          svg.removeAttribute('viewBox');
          return original;
        })
      );
      const reference = await grid.screenshot({ animations: 'disabled' });
      await info.attach(`${theme}-${twoTone ? 'two-tone' : 'ink'}-actual`, {
        body: actual,
        contentType: 'image/png'
      });
      await info.attach(`${theme}-${twoTone ? 'two-tone' : 'ink'}-reference`, {
        body: reference,
        contentType: 'image/png'
      });
      expect(
        actual.equals(reference),
        `all icons match original sprite: ${theme}, two-tone=${twoTone}`
      ).toBe(true);
      await grid.locator('svg[data-icon]').evaluateAll(
        (icons, originals) =>
          icons.forEach((svg, i) => {
            svg.innerHTML = originals[i]!.markup;
            svg.setAttribute('viewBox', originals[i]!.viewBox);
          }),
        originals
      );
    }
  }

  const ids = await grid.locator('[id]').evaluateAll((elements) => elements.map((e) => e.id));
  expect(new Set(ids).size).toBe(ids.length);
  for (const name of ['chain-target', 'hint-pair', 'looting-2', 'looting-5', 'token-frame']) {
    const icon = grid.locator(`svg[data-icon="${name}"]`);
    const valid = await icon.evaluate((svg) =>
      [...svg.querySelectorAll('[clip-path]')].every((node) => {
        const id = /url\(#([^)]+)\)/.exec(node.getAttribute('clip-path') ?? '')?.[1];
        return !!id && [...svg.querySelectorAll('[id]')].some((definition) => definition.id === id);
      })
    );
    expect(valid, `${name} paint references are local`).toBe(true);
  }
});

// Icon's defaults sit in app.css's components layer (M11.icon-size-override):
// above preflight's `svg { display: block; vertical-align: middle }`, below
// the utilities. Copies of a real Icon in plain text flow (a flex or grid
// parent would blockify display) read the cascade the page really loads.
test('an icon sits inline with text by default, and a caller size class wins @mobile', async ({
  page
}) => {
  await page.addInitScript(() => localStorage.setItem('ajayeb:name', 'Tester'));
  await page.goto('/dev/gallery');
  const icon = page.locator('main svg.icon[data-icon]').first();
  await expect(icon).toBeAttached();
  const styles = await icon.evaluate((svg) => {
    const p = document.createElement('p');
    p.style.fontSize = '16px';
    const plain = svg.cloneNode(true) as SVGSVGElement;
    const sized = svg.cloneNode(true) as SVGSVGElement;
    plain.setAttribute('class', 'icon');
    sized.setAttribute('class', 'icon size-3');
    p.append('Text ', plain, ' and ', sized);
    document.querySelector('main')!.append(p);
    const read = (el: Element) => {
      const cs = getComputedStyle(el);
      return {
        display: cs.display,
        verticalAlign: cs.verticalAlign,
        width: cs.width,
        height: cs.height
      };
    };
    return { plain: read(plain), sized: read(sized) };
  });
  expect(styles.plain).toEqual({
    display: 'inline-block',
    verticalAlign: '-2px',
    width: '16px',
    height: '16px'
  });
  expect(styles.sized).toEqual({
    display: 'inline-block',
    verticalAlign: '-2px',
    width: '12px',
    height: '12px'
  });
});
