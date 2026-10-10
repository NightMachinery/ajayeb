// Named recorded game states for the /dev/fixture pages, from
// src/lib/wire/fixtures.gen.ts:
//
//   - the recorded state frames, by file name (state-play, state-draft ...);
//   - every recorded View, by scenario and viewer (age-1-start-seat0 ...),
//     wrapped in the state frame its viewer would receive.
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

/** Every fixture name a page can show, recorded frames first. */
export function fixtureList(): string[] {
  const frames = fixtureNames.filter((_, i) => fixtures[i]?.t === 'state');
  return [...frames, ...protocolFixtureNames];
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
  return undefined;
}
