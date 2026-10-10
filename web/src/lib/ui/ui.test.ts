import { describe, expect, it } from 'vitest';
import { stateFrameFromView } from '$lib/session/fixture-transport';
import { protocolFixtureNames, protocolFixtures } from '$lib/wire/fixtures.gen';
import type { GameEvent, State, View } from '$lib/wire';
import { GameUi, LOG_LIMIT } from './ui.svelte';

function view(name: (typeof protocolFixtureNames)[number]): View {
  const v = protocolFixtures[protocolFixtureNames.indexOf(name)];
  if (!v) throw new Error(`no protocol fixture named ${name}`);
  return v;
}

function frame(v: View, events: GameEvent[] = []): State {
  return { ...stateFrameFromView(v), events };
}

const build = (card: string, seat = 0): GameEvent => ({
  kind: 'build',
  private: false,
  seat,
  card
});

describe('GameUi: sheets and selection', () => {
  it('opens the card sheet on a slot and steps to the wonders and back', () => {
    const ui = new GameUi();
    expect(ui.sheet).toBeNull();
    ui.selectSlot(4);
    expect(ui.isOpen('card')).toBe(true);
    expect(ui.selected).toBe(4);
    expect(ui.step).toBe('card');
    ui.showWonders();
    expect(ui.step).toBe('wonders');
    ui.backToCard();
    expect(ui.step).toBe('card');
    ui.showWonders();
    ui.selectSlot(2); // a new slot starts at the card again
    expect(ui.step).toBe('card');
    ui.close();
    expect(ui.sheet).toBeNull();
    expect(ui.selected).toBeNull();
    // The wonder step only exists inside the card sheet.
    ui.showWonders();
    expect(ui.step).toBe('card');
  });

  it('opens the other sheets, one at a time', () => {
    const ui = new GameUi();
    ui.openCity(1);
    expect(ui.isOpen('city')).toBe(true);
    expect(ui.citySeat).toBe(1);
    ui.openLog();
    expect(ui.isOpen('log')).toBe(true);
    expect(ui.isOpen('city')).toBe(false);
    ui.openSettings();
    expect(ui.sheet).toBe('settings');
    ui.openWonder('pyramids');
    expect(ui.sheet).toBe('wonder');
    expect(ui.wonder).toBe('pyramids');
  });

  it('keeps wonder-first mode across closing sheets until a move or cancel', () => {
    const ui = new GameUi();
    ui.openWonder('pyramids');
    ui.startWonderPick('pyramids');
    expect(ui.pickFor).toBe('pyramids');
    expect(ui.sheet).toBeNull();
    ui.selectSlot(3);
    ui.close();
    expect(ui.pickFor).toBe('pyramids');
    ui.cancelWonderPick();
    expect(ui.pickFor).toBeNull();
    ui.startWonderPick('pyramids');
    ui.selectSlot(3);
    ui.sent();
    expect(ui.pickFor).toBeNull();
    expect(ui.sheet).toBeNull();
  });
});

describe('GameUi: frames', () => {
  it('hands the device to the first seat to move, then only on ready', () => {
    const ui = new GameUi();
    const v = view('costs-seat0'); // seat 0 to move
    ui.record(frame(v), null, true);
    expect(ui.holder).toBe(0);
    ui.record(frame(view('costs-seat1')), 'act', true); // still seat 0 to move
    ui.record(frame({ ...v, toMove: 1 }), 'act', true);
    expect(ui.holder).toBe(0); // the pass screen shows until seat 1 is ready
    ui.ready(1);
    expect(ui.holder).toBe(1);
    ui.ready(undefined);
    expect(ui.holder).toBe(1);
  });

  it('follows the seat to move when the pass screen is off', () => {
    const ui = new GameUi();
    const v = view('costs-seat0');
    ui.record(frame(v), null, false);
    ui.record(frame({ ...v, toMove: 1 }), 'act', false);
    expect(ui.holder).toBe(1);
  });

  it('grows the log and takes an undone move’s events back out', () => {
    const ui = new GameUi();
    const v = view('costs-seat0');
    ui.record(frame(v, [build('baths')]), null, true);
    ui.record(frame(v, [build('walls'), build('quarry', 1)]), 'act', true);
    expect(ui.log.map((e) => e.card)).toEqual(['baths', 'walls', 'quarry']);
    ui.record(frame(v, [build('aqueduct', 1)]), null, true); // a bot's answer
    ui.record(frame(v, []), 'undo', true);
    expect(ui.log.map((e) => e.card)).toEqual(['baths']);
    // An undo with nothing recorded to take back leaves the log alone.
    ui.record(frame(v, []), 'undo', true);
    expect(ui.log.map((e) => e.card)).toEqual(['baths']);
  });

  it('caps the log at its limit', () => {
    const ui = new GameUi();
    const v = view('costs-seat0');
    const many = Array.from({ length: LOG_LIMIT + 5 }, (_, i) => build(`c${i}`));
    ui.record(frame(v, many), null, true);
    expect(ui.log).toHaveLength(LOG_LIMIT);
    expect(ui.log[0]?.card).toBe('c5');
  });

  it('closes the card sheet when its slot empties', () => {
    const ui = new GameUi();
    const v = view('costs-seat0');
    ui.selectSlot(4);
    ui.record(frame(v), null, true);
    expect(ui.sheet).toBe('card');
    const emptied: View = {
      ...v,
      structure: v.structure.map((s) =>
        s.index === 4
          ? { index: 4, row: s.row, col: s.col, state: 'empty', accessible: false, coveredBy: [] }
          : s
      )
    };
    ui.record(frame(emptied), null, true);
    expect(ui.sheet).toBeNull();
    expect(ui.selected).toBeNull();
  });

  it('leaves wonder-first mode once the wonder is no longer a choice', () => {
    const ui = new GameUi();
    ui.startWonderPick('pyramids');
    ui.record(frame(view('costs-seat0')), null, true);
    expect(ui.pickFor).toBe('pyramids');
    ui.record(frame(view('costs-seat1')), null, true); // the waiting seat has no choices
    expect(ui.pickFor).toBeNull();
  });

  it('resets for a new room', () => {
    const ui = new GameUi();
    ui.record(frame(view('costs-seat0'), [build('baths')]), 'act', true);
    ui.startWonderPick('pyramids');
    ui.selectSlot(1);
    ui.reset();
    expect(ui.log).toEqual([]);
    expect(ui.holder).toBeNull();
    expect(ui.pickFor).toBeNull();
    expect(ui.sheet).toBeNull();
  });
});
