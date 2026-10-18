import { afterEach, describe, expect, it } from 'vitest';
import { i18n, stripBidiMarks } from '$lib/i18n';
import type { CardTooltipView } from '$lib/wire';
import { cardTip, backTip, wonderTip } from './tips';

function publicCard(): CardTooltipView {
  return {
    card: 'aqueduct',
    place: 'faceUp',
    accessible: false,
    slot: 2,
    costs: [
      { total: 0, affordable: true, chainFree: true, needsTrade: false, shortfall: 0 },
      { total: 7, affordable: false, chainFree: false, needsTrade: true, shortfall: 3 }
    ]
  };
}
const plain = stripBidiMarks;
afterEach(() => i18n.set('en'));

describe('whole-card summary', () => {
  it('orders identity, effects, printed cost, links and builder without a table', () => {
    const tip = cardTip('baths', { owner: '<Moon>' });
    expect(tip.text).toBe('Baths: Civilian building, Age I.');
    expect(tip.lines).toEqual([
      '3 victory points',
      'Printed cost: 1 stone.',
      'Later builds Aqueduct free.',
      'Built by <Moon>.'
    ]);
    expect(tip.rulesId).toBeUndefined();
    expect(cardTip('missing').text).toBe('missing');
  });

  it('shows both public quotes, including chain and shortfall, even when covered', () => {
    const tip = cardTip('aqueduct', { info: publicCard(), names: ['Sun', 'Rival'], you: 0 });
    expect(tip.lines).toEqual([
      '5 victory points',
      'You now: free through a chain.',
      'Rival now: 7 coins (3 coins short).',
      'Free if you own Baths.'
    ]);
    const named = cardTip('aqueduct', { info: publicCard(), names: ['Sun', 'Moon'] });
    expect(
      cardTip('aqueduct', { info: publicCard(), names: ['Sun', 'Moon'], you: 1 }).lines?.[1]
    ).toBe('You now: 7 coins (3 coins short).');
    expect(named.lines?.slice(1, 3)).toEqual([
      'Sun now: free through a chain.',
      'Moon now: 7 coins (3 coins short).'
    ]);
  });

  it('words a free quote and keeps printed costs for city and discard cards', () => {
    const info = publicCard();
    info.costs[0].chainFree = false;
    expect(cardTip('aqueduct', { info, you: 0 }).lines?.[1]).toBe('You now: free.');
    for (const place of ['city', 'discard'] as const) {
      info.place = place;
      expect(cardTip('aqueduct', { info }).lines?.[1]).toBe('Printed cost: 3 stone.');
    }
    // A stale row for another card must never lend it a price.
    expect(cardTip('baths', { info }).lines?.[1]).toBe('Printed cost: 1 stone.');
  });

  it('uses an actual free revive payment while ordinary discard browsing keeps printed cost', () => {
    const actualCost = {
      affordable: true,
      chainFree: false,
      total: 0,
      shortfall: 0
    } as import('$lib/wire').CostView;
    const named = cardTip('aqueduct', { actualCost, costSeat: 1, names: ['Sun', 'Moon'] });
    expect(named.lines?.[1]).toBe('Moon now: free.');
    expect(cardTip('aqueduct', { actualCost, costSeat: 1, you: 1 }).lines?.[1]).toBe(
      'You now: free.'
    );
    expect(cardTip('aqueduct').lines?.[1]).toBe('Printed cost: 3 stone.');
    expect(
      cardTip('aqueduct', { info: publicCard(), actualCost, costSeat: 1, you: 1 }).lines?.[1]
    ).toBe('You now: free.');
  });

  it('reuses one short line per effect', () => {
    const tip = cardTip('library');
    expect(tip.lines?.[0]).toMatch(/science|symbol/);
    expect(tip.lines?.[1]).toMatch(/victory points/);
  });

  it('translates identity, both quotes, links and builder in Persian', async () => {
    await i18n.set('fa');
    const tip = cardTip('aqueduct', { info: publicCard(), names: ['خورشید', 'Moon'], you: 0 });
    expect(plain(tip.text)).toBe('آبراهه: ساختمان مدنی، دوران دوم.');
    expect(tip.lines?.map(plain)).toEqual([
      '۵ امتیاز',
      'اکنون برای شما: رایگان از راه زنجیره.',
      'اکنون برای Moon: ۷ سکه (۳ سکه کم است).',
      'اگر گرمابه را دارید، رایگان است.'
    ]);
    const free = cardTip('baths', {
      actualCost: {
        total: 0,
        affordable: true,
        chainFree: false,
        shortfall: 0
      } as import('$lib/wire').CostView,
      costSeat: 1,
      names: ['خورشید', 'Moon']
    });
    expect(plain(free.lines?.[1] ?? '')).toBe('اکنون برای Moon: رایگان.');
    const built = cardTip('baths', { owner: 'Moon' });
    expect(plain(built.lines?.at(-1) ?? '')).toBe('ساخته‌ی Moon.');
    expect(backTip('guild').lines?.[0]).toBe(
      'وقتی کارت‌هایی که آن را می‌پوشانند گرفته شوند، رو می‌شود.'
    );
  });
});

describe('whole-wonder summaries for draft cards', () => {
  it('has no age and reports printed effects/cost and public taken/built state', () => {
    const draft = wonderTip('pyramids', undefined, 'Moon');
    expect(draft.text).toBe('Pyramids: wonder.');
    expect(draft.text).not.toContain('Age');
    expect(draft.lines?.[0]).toBe('9 victory points');
    expect(draft.lines?.at(-1)).toBe('Taken by Moon');
    const built = wonderTip('pyramids', {
      wonder: 'pyramids',
      built: true,
      removed: false,
      tucked: 'baths'
    });
    expect(built.lines?.at(-1)).toBe('Built with Baths');
  });

  it('translates the wonder kind and taken state into Persian', async () => {
    await i18n.set('fa');
    const tip = wonderTip('pyramids', undefined, 'Moon');
    expect(plain(tip.text)).toBe('هرم‌های بزرگ: شگفتی.');
    expect(plain(tip.lines?.at(-1) ?? '')).toBe('Moon آن را برداشت');
  });
});
