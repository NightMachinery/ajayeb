// The imperative FX layer uses the same original shapes as Icon.svelte.
import { iconMarkup, iconSources, type IconName } from './sources';

let nextPaint = 0;

export function drawIcon(svg: SVGSVGElement, name: IconName): void {
  svg.setAttribute('viewBox', iconSources[name].viewBox);
  svg.setAttribute('data-icon', name);
  // Trusted build-time art only. Each drawn or copied icon owns its paint ids.
  svg.innerHTML = iconMarkup(name, `drawn-${++nextPaint}`);
}

/** makeInert strips element ids. Restore local clipping in its detached copy
 *  so a flying icon never depends on an original that the move removed. */
export function restoreIconPaint(root: Element): void {
  const icons = [root, ...root.querySelectorAll('svg[data-icon]')];
  for (const svg of icons) {
    const name = svg.getAttribute('data-icon') as IconName | null;
    if (name && Object.hasOwn(iconSources, name) && iconSources[name].inner.includes('id="')) {
      drawIcon(svg as SVGSVGElement, name);
    }
  }
}
