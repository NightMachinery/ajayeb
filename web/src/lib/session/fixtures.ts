// Named recorded game states for the /dev/fixture pages, from
// src/lib/wire/fixtures.gen.ts:
//
//   - the recorded state frames, by file name (state-play, state-draft ...);
//   - every recorded View, by scenario and viewer (age-1-start-seat0 ...),
//     wrapped in the state frame its viewer would receive;
//   - a few online room states made from those views (online-...): the
//     lobby, an opponent who dropped, and a game with spectators. Only the
//     room part is changed; the views are the recorded ones. The age-start
//     scene also carries the recorded starterChosen event.
//
// The generated file is large (about 1.4 MB of JSON), so pages import this
// module dynamically and it never reaches the bundle of a normal page.

import type { State } from '$lib/wire';
import {
  fixtureNames,
  fixtures,
  protocolFixtureNames,
  protocolFixtures
} from '$lib/wire/fixtures.gen';
import { stateFrameFromView } from './fixture-transport';

const SEAT_NAMES = ['Sara', 'Omid'] as const;

function viewFrame(name: (typeof protocolFixtureNames)[number]): State | undefined {
  const view = protocolFixtures[protocolFixtureNames.indexOf(name)];
  return view ? stateFrameFromView(view, { code: 'FIXTUR', names: SEAT_NAMES }) : undefined;
}

/** The online room states, by name: each a recorded view in a changed room. */
const ONLINE: Readonly<Record<string, () => State | undefined>> = {
  // The starter-chosen golden fixture's event, restored beside its view.
  'online-age-start-seat0': () => {
    const frame = viewFrame('starter-chosen-seat0');
    if (!frame) return undefined;
    frame.events = [{ kind: 'starterChosen', private: false, seat: 1, starter: 0 }];
    return frame;
  },
  // Seat 0 created the room; seat 1 is still open.
  'online-lobby-seat0': () => {
    const f = viewFrame('draft-round-1-seat0');
    if (!f) return undefined;
    f.room.status = 'lobby';
    f.room.seats[1] = {
      seat: 1,
      name: '',
      kind: 'open',
      connected: false,
      you: false,
      nameNo: 0
    };
    f.you = { seats: [0], canAct: false, admin: true };
    return f;
  },
  // Seat 1 is to move and has dropped; one spectator watches.
  'online-away-seat0': () => {
    const f = viewFrame('card-destroyed-seat0');
    if (!f || !f.room.seats[1]) return undefined;
    f.room.seats[1].connected = false;
    f.room.spectators = 1;
    return f;
  },
  // A spectator's view of a game two more spectators watch.
  'online-watching-spectator': () => {
    const f = viewFrame('age-1-start-spectator');
    if (!f) return undefined;
    f.room.spectators = 3;
    return f;
  }
};

/** Every fixture name a page can show, recorded frames first. */
export function fixtureList(): string[] {
  const frames = fixtureNames.filter((_, i) => fixtures[i]?.t === 'state');
  return [...frames, ...protocolFixtureNames, ...Object.keys(ONLINE)];
}

/** The state frames of a named fixture, or undefined for an unknown name. */
export function fixtureFrames(name: string): State[] | undefined {
  const i = (fixtureNames as readonly string[]).indexOf(name);
  if (i >= 0) {
    const frame = fixtures[i];
    return frame?.t === 'state' ? [frame as State] : undefined;
  }
  const j = (protocolFixtureNames as readonly string[]).indexOf(name);
  if (j >= 0) {
    const view = protocolFixtures[j];
    if (!view) return undefined;
    return [stateFrameFromView(view, { code: 'FIXTUR', names: SEAT_NAMES })];
  }
  if (Object.hasOwn(ONLINE, name)) {
    const frame = ONLINE[name]?.();
    return frame ? [frame] : undefined;
  }
  return undefined;
}
