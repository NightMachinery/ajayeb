import { describe, expect, it } from 'vitest';
import { cardSlugs, wonderSlugs } from '$lib/catalog';
import { protocolFixtureNames, protocolFixtures } from '$lib/wire/fixtures.gen';
import type { View } from '$lib/wire';
import { describeCard, describeSlot, describeWonder } from './describe';

function fixture(name: (typeof protocolFixtureNames)[number]): View {
  const view = protocolFixtures[protocolFixtureNames.indexOf(name)];
  if (!view) throw new Error(`no fixture ${name}`);
  return view as View;
}

function slot(view: View, card: string) {
  const found = view.structure.find((s) => s.card === card);
  if (!found) throw new Error(`no slot for ${card}`);
  return found;
}

describe('describeCard', () => {
  it('names the card, its colour and its effects', () => {
    expect(describeCard('lumber-yard')).toBe('Lumber Yard, brown. Produces 1 wood each turn');
  });

  it('includes both chain ends', () => {
    expect(describeCard('temple')).toBe(
      'Temple, blue. 4 victory points. Free with the moon chain symbol of Altar. ' +
        'Chain symbol sun: makes Pantheon free'
    );
  });

  it('joins resource and colour lists for the locale', () => {
    expect(describeCard('caravansery')).toContain('Produces one of wood, clay, or stone each turn');
    expect(describeCard('shipowners-guild')).toContain(
      'Take 1 coin for each brown and grey card in the city that has the most'
    );
  });

  it('describes every card and wonder of the catalog without a raw key or gap', () => {
    for (const slug of [...cardSlugs, ...wonderSlugs]) {
      const text = cardSlugs.includes(slug as never) ? describeCard(slug) : describeWonder(slug);
      expect(text, slug).not.toMatch(/[{}]|\b(card|wonder)\.[a-z]/);
      expect(text.length).toBeGreaterThan(10);
    }
  });
});

describe('describeSlot, from recorded views', () => {
  it('reads a takeable card with its cost for the viewer', () => {
    const view = fixture('age-1-reveal-seat0');
    expect(describeSlot(slot(view, 'stable'), { seat: 0 })).toBe(
      'Stable, red. 1 shield: moves the conflict pawn 1 space. ' +
        'Chain symbol horseshoe: makes Horse Breeders free. ' +
        'Available, build for 2 coins with trade'
    );
    expect(describeSlot(slot(view, 'quarry'), { seat: 0 })).toMatch(/Available, free to build$/);
  });

  it('says when a chain makes it free and when it cannot be built', () => {
    const view = fixture('costs-seat0');
    expect(describeSlot(slot(view, 'statue'), { seat: 0 })).toMatch(
      /Available, free through Theater$/
    );
    expect(describeSlot(slot(view, 'parade-ground'), { seat: 0 })).toMatch(
      /Available, cannot build: costs 10 coins, 1 more than you have$/
    );
  });

  it('says a covered card is covered, and by how many', () => {
    const view = fixture('age-1-start-seat0');
    expect(describeSlot(slot(view, 'pharmacist'), { seat: 0 })).toMatch(/Covered by 2 cards$/);
  });

  it('marks the cost as planning-only when it is not your turn', () => {
    const view = fixture('age-1-start-seat0');
    const text = describeSlot(slot(view, 'baths'), { seat: 0, waiting: true });
    expect(text).toMatch(/, not your turn$/);
  });

  it('reads face-down and empty slots', () => {
    const view = fixture('looting-seat1');
    const faceDown = view.structure.find((s) => s.state === 'faceDown')!;
    const empty = view.structure.find((s) => s.state === 'empty')!;
    expect(describeSlot(faceDown)).toBe('Face-down Age III card, revealed when uncovered');
    expect(describeSlot(empty)).toBe('Empty place');
  });
});

describe('describeWonder', () => {
  it('reads a wonder with its cost, or its built state', () => {
    const view = fixture('costs-seat0');
    const pyramids = view.players[0].wonders.find((w) => w.wonder === 'pyramids')!;
    expect(describeWonder('pyramids', pyramids)).toBe(
      'Pyramids, wonder. 9 victory points. Build for 3 coins with trade'
    );
    expect(describeWonder('pyramids', { ...pyramids, built: true, tucked: 'baths' })).toMatch(
      /Built with Baths$/
    );
  });
});
