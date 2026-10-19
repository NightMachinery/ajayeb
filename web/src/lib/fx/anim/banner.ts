// The victory banner (runner.ts, the `victory` step): a ribbon naming how
// the game ended, with the winner's crest, and a burst of PARTICLES small
// diamonds and dots, drawn in the FX layer's top-layer part over the result
// screen. Transform and opacity only; never clickable, so it never blocks
// the result's buttons. Its words are the result screen's own victory kinds
// (choices.over.kind.*), its tint the victory kind's: cinnabar for a
// military supremacy, green for a scientific one, gold for a civilian
// victory, neutral for a shared one or a forfeit.

import { i18n, t } from '$lib/i18n';
import { drawIcon } from '$lib/icons/draw';
import type { IconName } from '$lib/icons/names';
import { drawn } from './dom';

export const PARTICLES = 12;

export interface Banner {
  el: HTMLElement;
  ribbon: HTMLElement;
  particles: HTMLElement[];
}

type BannerKind = 'military' | 'science' | 'civilian' | 'forfeit' | 'shared';

function kindOf(victory: string | undefined, winner: number | undefined): BannerKind {
  if (winner === undefined) return 'shared';
  switch (victory) {
    case 'military':
    case 'science':
    case 'forfeit':
      return victory;
    default:
      return 'civilian';
  }
}

function icon(name: IconName, className: string): SVGSVGElement {
  const ns = 'http://www.w3.org/2000/svg';
  const svg = document.createElementNS(ns, 'svg');
  svg.setAttribute('class', className);
  svg.setAttribute('aria-hidden', 'true');
  drawIcon(svg, name);
  return svg;
}

/** The banner's elements, not yet placed: the caller positions and animates them. */
export function victoryBanner(victory: string | undefined, winner: 0 | 1 | undefined): Banner {
  const kind = kindOf(victory, winner);
  const el = drawn('div', `fx-banner fx-banner-${kind}`);
  const ribbon = drawn('div', 'fx-banner-ribbon');
  if (winner !== undefined) {
    ribbon.append(icon(winner === 0 ? 'crest-sun' : 'crest-moon', 'fx-banner-crest'));
  }
  const label = document.createElement('span');
  label.className = 'fx-banner-text';
  label.dir = i18n.dir;
  label.textContent = t(`choices.over.kind.${kind}`);
  ribbon.append(label);
  const particles: HTMLElement[] = [];
  for (let i = 0; i < PARTICLES; i++) {
    const shape = i % 3 === 0 ? 'fx-particle-dot' : 'fx-particle-gem';
    const p = drawn('span', `fx-particle ${shape} fx-hue-${i % 4}`);
    particles.push(p);
  }
  el.append(...particles, ribbon);
  return { el, ribbon, particles };
}

/** Where particle `i` of PARTICLES flies, in px from the banner's centre. */
export function particleOffset(i: number, width: number): { x: number; y: number; turn: number } {
  const angle = (i / PARTICLES) * Math.PI * 2 + 0.3;
  const reach = Math.max(90, width * 0.4) * (i % 2 === 0 ? 1 : 0.7);
  return {
    x: Math.round(Math.cos(angle) * reach),
    y: Math.round(Math.sin(angle) * reach * 0.7),
    turn: (i % 2 === 0 ? 1 : -1) * (90 + i * 15)
  };
}
