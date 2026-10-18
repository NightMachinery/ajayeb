import { afterEach, describe, expect, it } from 'vitest';
import { i18n, stripBidiMarks } from '$lib/i18n';
import { cardBySlug, cardSlugs } from '$lib/catalog';
import type { Chain } from '$lib/wire';
import { chainSource, chainTarget, chainTooltip } from './chain-links';

const CHAINS = [
  ...new Set(cardSlugs.flatMap((slug) => [cardBySlug(slug)?.chainOut].filter(Boolean)))
] as Chain[];

describe('chain links', () => {
  it('links every chain symbol to the cards the catalog names', () => {
    expect(CHAINS.length).toBeGreaterThan(10);
    for (const chain of CHAINS) {
      const source = chainSource(chain);
      const target = chainTarget(chain);
      expect(source, chain).toBeDefined();
      expect(target, chain).toBeDefined();
      expect(cardBySlug(source!)?.chainTo).toBe(target);
      expect(cardBySlug(target!)?.chainFrom).toBe(source);
    }
  });

  it('Baths (drop) makes Aqueduct free, and Aqueduct stacks Baths', () => {
    const out = chainTooltip('drop', 'out');
    expect(out.text).toContain('Aqueduct');
    expect(out.stack).toEqual([expect.objectContaining({ kind: 'card', slug: 'aqueduct' })]);
    const into = chainTooltip('drop', 'in');
    expect(into.text).toContain('Baths');
    expect(into.stack).toEqual([expect.objectContaining({ kind: 'card', slug: 'baths' })]);
    // Chains are explained in the rules' costs section.
    expect(out.rulesId).toBe('costs');
    expect(into.rulesId).toBe('costs');
  });

  it('carries the status lines it is given', () => {
    expect(chainTooltip('drop', 'out').lines).toEqual([
      'Aqueduct: Civilian building, Age II.',
      '5 victory points'
    ]);
    expect(chainTooltip('drop', 'in', ['You show this symbol.']).lines).toEqual([
      'Baths: Civilian building, Age I.',
      '3 victory points',
      'You show this symbol.'
    ]);
  });
});

function linkedContext(): import('$lib/view/tips').CardTipContext {
  return {
    names: ['Sun', 'Rival'],
    you: 0,
    linkSeat: 0,
    info: {
      card: 'baths',
      place: 'city',
      seat: 0,
      accessible: false,
      costs: [
        { total: 0, affordable: true, chainFree: false, needsTrade: false, shortfall: 0 },
        { total: 0, affordable: true, chainFree: false, needsTrade: false, shortfall: 0 }
      ],
      chainOut: {
        place: 'faceUp',
        accessible: true,
        slot: 4,
        costs: [
          { total: 9, affordable: false, chainFree: false, needsTrade: true, shortfall: 2 },
          { total: 4, affordable: true, chainFree: false, needsTrade: true, shortfall: 0 }
        ]
      }
    }
  };
}

describe('linked public details', () => {
  afterEach(() => i18n.set('en'));

  it('shows identity, effects, the solver quote without chain, and whereabouts', () => {
    const context = linkedContext();
    const tip = chainTooltip('drop', 'out', ['old contradictory status'], context);
    expect(tip.lines).toEqual([
      'Aqueduct: Civilian building, Age II.',
      '5 victory points',
      'Without the chain for you now: 9 coins (2 coins short).',
      'Resource trading needed.',
      'Face up in the structure, available now.'
    ]);
    expect(tip.stack).toBeUndefined();
    expect(tip.avoidAnchor).toBe(true);
    context.linkSeat = 1;
    expect(chainTooltip('drop', 'out', [], context).lines?.[2]).toBe(
      'Without the chain for Rival now: 4 coins.'
    );
    // Planning seat and actual city owner take precedence over the viewer.
    context.linkSeat = undefined;
    context.costSeat = 1;
    expect(chainTooltip('drop', 'out', [], context).lines?.[2]).toContain('Rival');
  });

  it('formats all public locations and a resource-free quote without deriving any values', () => {
    const context = linkedContext();
    const detail = context.info!.chainOut!;
    detail.costs![0] = {
      total: 0,
      affordable: true,
      chainFree: false,
      needsTrade: false,
      shortfall: 0
    };
    expect(chainTooltip('drop', 'out', [], context).lines?.slice(2, 4)).toEqual([
      'Without the chain for you now: free.',
      'No resource trading needed.'
    ]);
    for (const [place, accessible, text] of [
      ['faceUp', false, 'Face up in the structure, still covered.'],
      ['city', false, 'Built by Rival.'],
      ['tucked', false, 'Under a wonder built by Rival.'],
      ['discard', false, 'In the discard pile.'],
      ['unseenThisAge', false, 'Not seen yet.'],
      ['unseenLaterAge', false, 'Not seen yet.'],
      ['removedKnown', false, 'Not in this game.']
    ] as const) {
      Object.assign(detail, { place, accessible, seat: 1 });
      expect(chainTooltip('drop', 'out', [], context).lines?.at(-1)).toBe(text);
    }
  });

  it('previews the reverse source with the same block minus every cost line', () => {
    const context = linkedContext();
    context.info!.chainIn = { place: 'city', accessible: false, seat: 1 };
    const tip = chainTooltip('drop', 'in', [], context);
    expect(tip.lines).toEqual([
      'Baths: Civilian building, Age I.',
      '3 victory points',
      'Built by Rival.'
    ]);
    expect(tip.lines?.some((line) => /coin|trading|chain for/.test(line))).toBe(false);
  });

  it('translates linked effects, asymmetric cost and every whereabouts line into Persian', async () => {
    await i18n.set('fa');
    const context = linkedContext();
    context.linkSeat = 1;
    context.names = ['خورشید', 'Moon'];
    const lines = chainTooltip('drop', 'out', [], context).lines!.map(stripBidiMarks);
    expect(lines).toEqual([
      'آبراهه: ساختمان مدنی، دوران دوم.',
      '۵ امتیاز',
      'اکنون بدون زنجیره برای Moon: ۴ سکه.',
      'به خرید منبع از بانک نیاز دارد.',
      'در ساختار رو است و اکنون در دسترس است.'
    ]);
    const detail = context.info!.chainOut!;
    const places = [
      ['faceUp', 'در ساختار رو است، اما هنوز پوشیده است.'],
      ['city', 'ساخته‌ی Moon.'],
      ['tucked', 'زیر شگفتیِ ساخته‌ی Moon.'],
      ['discard', 'در کارت‌های کنارگذاشته است.'],
      ['unseenThisAge', 'هنوز دیده نشده است.'],
      ['unseenLaterAge', 'هنوز دیده نشده است.'],
      ['removedKnown', 'در این بازی نیست.']
    ] as const;
    for (const [place, text] of places) {
      Object.assign(detail, { place, accessible: false, seat: 1 });
      expect(stripBidiMarks(chainTooltip('drop', 'out', [], context).lines!.at(-1)!)).toBe(text);
    }
    context.linkSeat = 0;
    expect(stripBidiMarks(chainTooltip('drop', 'out', [], context).lines![2]!)).toBe(
      'اکنون بدون زنجیره برای شما: ۹ سکه (۲ سکه کم است).'
    );
    context.info!.chainIn = { place: 'city', accessible: false, seat: 1 };
    expect(chainTooltip('drop', 'in', [], context).lines!.map(stripBidiMarks)).toEqual([
      'گرمابه: ساختمان مدنی، دوران یکم.',
      '۳ امتیاز',
      'ساخته‌ی Moon.'
    ]);
  });
});
