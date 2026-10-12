// Small display rules of the board's components, rendered on the server
// (M10.F1): the bot's avatar in the city sheet, signed coin amounts through
// Intl, word-safe names in the score table, and the Allow undo switch rows.
import { render } from 'svelte/server';
import { describe, expect, it } from 'vitest';
import { formatNumber } from '$lib/i18n';
import { protocolFixtureNames, protocolFixtures } from '$lib/wire/fixtures.gen';
import type { View } from '$lib/wire';
import LootingToken from './board/LootingToken.svelte';
import MoveSummaryLine from './game/MoveSummaryLine.svelte';
import HotseatSheet from './screens/HotseatSheet.svelte';
import NewGameSheet from './screens/NewGameSheet.svelte';
import ScoreTable from './screens/ScoreTable.svelte';
import CitySheet from './sheets/CitySheet.svelte';

function fixture(name: (typeof protocolFixtureNames)[number]): View {
  const v = protocolFixtures[protocolFixtureNames.indexOf(name)];
  if (!v) throw new Error(`no fixture ${name}`);
  return v;
}

const avatars = (markup: string) =>
  [...markup.matchAll(/data-avatar="([a-z]+)"/g)].map((m) => m[1]);

describe('the city sheet', () => {
  const view = fixture('age-1-start-seat0');
  const sheet = (activeSeat: 0 | 1) =>
    render(CitySheet, {
      props: {
        open: true,
        names: ['Sara', 'Master'],
        players: view.players,
        military: view.military,
        bots: [undefined, 'hard'],
        activeSeat
      }
    }).body;

  it("shows the bot's avatar on the bot's tab, and none on the human's", () => {
    expect(avatars(sheet(1))).toEqual(['hard']);
    expect(avatars(sheet(0))).toEqual([]);
  });
});

describe('signed coin amounts come from Intl', () => {
  it('a looting token reads its loss with the locale sign', () => {
    const markup = render(LootingToken, { props: { value: 5, magnitude: 6, present: true } }).body;
    expect(markup).toContain(`>${formatNumber(-5, { signDisplay: 'negative' })}<`);
    expect(markup).not.toContain('−');
  });

  it("a move summary's paid coins read as a loss, with the locale sign", () => {
    const markup = render(MoveSummaryLine, {
      props: {
        seat: 0,
        names: ['Sara', 'Omid'],
        line: {
          ply: 3,
          kind: 'build',
          seat: 0,
          card: 'baths',
          paid: 4,
          toOpponent: 0,
          coins: [0, 0],
          pawn: null,
          loot: [],
          pairs: []
        }
      }
    }).body;
    expect(markup).toContain(formatNumber(-4, { signDisplay: 'negative' }));
    expect(markup).not.toContain('−');
  });
});

describe('the score table', () => {
  it('wraps a name at its spaces, never inside a word', () => {
    const markup = render(ScoreTable, {
      props: { rows: [], names: ['Sara', 'Apprentice'], winner: null }
    }).body;
    expect(markup).toContain('wrap-break-word');
    expect(markup).not.toContain('overflow-wrap:anywhere');
  });
});

describe('the Allow undo switch rows', () => {
  it('show the pointer cursor, like the other switch rows', () => {
    const rowClass = (markup: string) =>
      /<label class="([^"]*)"[^>]*data-allow-undo/.exec(markup)?.[1];
    const ai = render(NewGameSheet, { props: { open: true, onStart: () => {} } }).body;
    const hotseat = render(HotseatSheet, { props: { open: true, onStart: () => {} } }).body;
    expect(rowClass(ai)).toContain('cursor-pointer');
    expect(rowClass(hotseat)).toContain('cursor-pointer');
  });
});
