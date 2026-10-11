// The art wired into the app's screens (M10.S01), rendered on the server
// against the generated index: the bot avatars in the new-game sheet, the
// player strip and the city panel, and the victory scene on the result
// screen.
import { render } from 'svelte/server';
import { describe, expect, it } from 'vitest';
import CityPanel from '$lib/components/city/CityPanel.svelte';
import PlayerStrip from '$lib/components/city/PlayerStrip.svelte';
import GameOver from '$lib/components/screens/GameOver.svelte';
import NewGameSheet from '$lib/components/screens/NewGameSheet.svelte';
import { protocolFixtureNames, protocolFixtures } from '$lib/wire/fixtures.gen';
import type { View } from '$lib/wire';

function fixture(name: (typeof protocolFixtureNames)[number]): View {
  const v = protocolFixtures[protocolFixtureNames.indexOf(name)];
  if (!v) throw new Error(`no fixture ${name}`);
  return v;
}

const avatars = (markup: string) =>
  [...markup.matchAll(/data-avatar="([a-z]+)"/g)].map((m) => m[1]);

describe('bot avatars', () => {
  it('shows each level’s avatar in its choice on the new-game sheet', () => {
    const markup = render(NewGameSheet, { props: { open: true, onStart: () => {} } }).body;
    expect(avatars(markup)).toEqual(['easy', 'normal', 'hard']);
  });

  it('shows the avatar by a bot seat’s name in its strip and panel, and none for a human', () => {
    const view = fixture('age-1-start-seat0');
    const seat = (bot?: 'hard') => ({
      name: 'Master',
      seat: 1 as const,
      player: view.players[1],
      military: view.military,
      bot
    });
    for (const Seat of [PlayerStrip, CityPanel]) {
      const withBot = render(Seat, { props: seat('hard') }).body;
      expect(avatars(withBot)).toEqual(['hard']);
      expect(withBot).toContain('/art/avatars/master.avif');
      expect(avatars(render(Seat, { props: seat() }).body)).toEqual([]);
    }
  });
});

describe('victory scenes', () => {
  const scene = (name: (typeof protocolFixtureNames)[number]) => {
    const markup = render(GameOver, {
      props: { view: fixture(name), onhome: () => {}, names: ['Sara', 'Omid'] }
    }).body;
    return /data-victory-scene="([a-z/]+)"/.exec(markup)?.[1];
  };

  it('shows the scene of how the game ended', () => {
    expect(scene('game-over-military-hidden-seat0')).toBe('victory/military');
    expect(scene('game-over-science-seat0')).toBe('victory/science');
    expect(scene('game-over-civilian-seat0')).toBe('victory/civilian');
  });

  it('shows the civilian scene for a draw (no draw scene is approved)', () => {
    expect(scene('game-over-shared-spectator')).toBe('victory/civilian');
  });
});
