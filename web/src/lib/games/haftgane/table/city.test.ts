// The city sheet's and the notebook's grouping over the engine's own view
// fixtures (server/internal/haftgane/rules/testdata/views): the ring tabs'
// order for a seat and a spectator, a city's rows by colour, the point ids
// decoded, and the notebook by Age with its filter.
import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';
import type { Notebook, Table } from '$lib/wire/haftgane.gen';
import { cityRows, itemMatches, matchesFilter, notebookAges, pointParts, ringOrder } from './city';
import { COLOURS, design } from './model';

function fixture(name: string): Table {
  return JSON.parse(
    readFileSync(
      new URL(
        `../../../../../../server/internal/haftgane/rules/testdata/views/${name}.json`,
        import.meta.url
      ),
      'utf8'
    )
  ) as Table;
}

describe('ringOrder', () => {
  it('starts at the viewer and goes round from the left neighbour', () => {
    const view = fixture('discard-seat');
    const order = ringOrder(view.seats);
    expect(order.map((s) => s.relative)).toEqual([0, 1, 2, 3, 4, 5, 6]);
    expect(order[0]!.relation).toBe('self');
    expect(order[1]!.relation).toBe('left');
    expect(order.at(-1)!.relation).toBe('right');
  });

  it('starts at seat 0 for a spectator and goes up the seats', () => {
    const view = fixture('turn-spectator');
    expect(ringOrder(view.seats).map((s) => s.seat)).toEqual([0, 1, 2, 3, 4, 5, 6]);
  });
});

describe('cityRows', () => {
  it('lists every colour once, with exactly the cards the view lists', () => {
    const view = fixture('over-reveal-spectator');
    for (const seat of view.seats) {
      const rows = cityRows(seat.built);
      expect(rows.map((r) => r.colour)).toEqual([...COLOURS]);
      expect(rows.flatMap((r) => r.cards).sort()).toEqual([...seat.built].sort());
      for (const row of rows) {
        for (const slug of row.cards) expect(design(slug)?.colour).toBe(row.colour);
      }
    }
  });
});

describe('pointParts', () => {
  it('decodes the point ids the notebook carries', () => {
    expect(pointParts('age-1-turn-3')).toEqual({ age: 1, turn: 3, kind: 'turn' });
    expect(pointParts('age-2-turn-6-discard-build')).toEqual({
      age: 2,
      turn: 6,
      kind: 'discard-build'
    });
    expect(pointParts('age-3-last-card')).toEqual({ age: 3, turn: 7, kind: 'last-card' });
  });

  it('refuses anything else', () => {
    for (const id of ['sides', '', 'age-4-turn-1', 'age-1-turn-7', 'age-1-turn-1-x']) {
      expect(pointParts(id), id).toBeNull();
    }
  });
});

describe('notebookAges', () => {
  it('puts the current Age first, then the earlier ones, each in turn order', () => {
    const view = fixture('discard-seat');
    const nb = view.you!.notebook;
    const ages = notebookAges(nb, view.age);
    expect(ages.map((a) => a.age)).toEqual([2, 1]);
    for (const a of ages) {
      const turns = a.items.map((i) => i.parts.turn);
      expect(turns).toEqual([...turns].sort((x, y) => x - y));
    }
    // Every entry and look appears once, in the Age its point names.
    const all = ages.flatMap((a) => a.items);
    expect(all.filter((i) => i.kind === 'hand')).toHaveLength(nb.seen.length);
    expect(all.filter((i) => i.kind === 'pile')).toHaveLength(nb.pileLooks.length);
    // The look at the pile follows the hand of its turn.
    const age2 = ages[0]!.items;
    const pile = age2.findIndex((i) => i.kind === 'pile');
    expect(pile).toBeGreaterThan(0);
    expect(age2[pile - 1]!.parts.turn).toBe(age2[pile]!.parts.turn);
  });

  it('lists the current Age even before it holds anything', () => {
    const view = fixture('turn-seat');
    expect(notebookAges(view.you!.notebook, view.age)).toEqual([{ age: 1, items: [] }]);
  });

  it('starts a new Age on its own page', () => {
    const view = fixture('discard-seat');
    const ages = notebookAges(view.you!.notebook, 3);
    expect(ages.map((a) => a.age)).toEqual([3, 2, 1]);
    expect(ages[0]!.items).toEqual([]);
  });
});

describe('the notebook filter', () => {
  const nb: Notebook = fixture('discard-seat').you!.notebook;

  it('matches a colour by the catalog and a card by its slug', () => {
    expect(matchesFilter('stockade', { colour: 'red' })).toBe(true);
    expect(matchesFilter('stockade', { colour: 'blue' })).toBe(false);
    expect(matchesFilter('stockade', { design: 'stockade' })).toBe(true);
    expect(matchesFilter('stockade', null)).toBe(true);
  });

  it('keeps the entries that hold a matching card', () => {
    const items = notebookAges(nb, 2).flatMap((a) => a.items);
    const red = items.filter((i) => itemMatches(i, { colour: 'red' }));
    expect(red.length).toBeGreaterThan(0);
    expect(red.length).toBeLessThan(items.length + 1);
    for (const i of red) {
      const cards = i.kind === 'hand' ? i.seen.hand : i.look.cards.map((c) => c.design);
      expect(cards.some((slug) => design(slug)?.colour === 'red')).toBe(true);
    }
    expect(items.filter((i) => itemMatches(i, null))).toHaveLength(items.length);
  });
});
