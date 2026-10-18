import { describe, expect, it } from 'vitest';
import { protocolFixtureNames, protocolFixtures } from '$lib/wire/fixtures.gen';
import type { State } from '$lib/wire';
import { stateFrameFromView } from '$lib/session/fixture-transport';
import { AgeSummary } from './age-summary.svelte';

function boundary(): State {
  const view = structuredClone(
    protocolFixtures[protocolFixtureNames.indexOf('choose-starter-seat0')]
  )!;
  return stateFrameFromView(view);
}
function chosen(before: State): State {
  const next = structuredClone(before);
  next.view.ply++;
  next.view.phase = 'play';
  delete next.view.pending;
  next.view.active = 1;
  next.events = [{ kind: 'starterChosen', seat: 0, starter: 1, private: false }];
  return next;
}

describe('completed-age presentation memory', () => {
  it('keeps the boundary totals through starter selection and later moves, even before presentation', () => {
    const summary = new AgeSummary();
    const end = boundary();
    summary.record(end);
    const start = chosen(end);
    summary.record(start);
    const moved = structuredClone(start);
    moved.view.ply++;
    moved.view.active = 0;
    moved.view.players[1].coins += 3;
    moved.view.scorePreview[1].total++;
    moved.events = [{ kind: 'discard', seat: 1, card: 'theater', private: false }];
    summary.record(moved);
    expect(summary.view).toBe(end.view);
    expect(summary.starter).toBe(1);
    expect(summary.dismissed).toBe(false);
    summary.markShown();
    summary.record({ ...moved, events: [] });
    expect(summary.shown).toBe(true);
    expect(summary.view?.players[1].coins).toBe(end.view.players[1].coins);
  });

  it('keeps the summary while the next age opens another pending choice', () => {
    const summary = new AgeSummary();
    const end = boundary();
    summary.record(end);
    const start = chosen(end);
    summary.record(start);
    summary.record({
      ...start,
      view: {
        ...start.view,
        phase: 'pickBoardToken',
        ply: start.view.ply + 1,
        pending: { kind: 'pickBoardToken', seat: 1, cards: [], tokens: [], hiddenTokens: 0 }
      },
      events: []
    });
    expect(summary.view).toBe(end.view);
    expect(summary.isDismissed()).toBe(false);
  });

  it('does not reopen on presence, duplicate frames or a same-position resync after explicit close', () => {
    const summary = new AgeSummary();
    const end = boundary();
    summary.record(end);
    summary.close();
    summary.record(end);
    const start = chosen(end);
    summary.record(start);
    summary.record({ ...start, events: [], history: [] });
    expect(summary.dismissed).toBe(true);
    expect(summary.view).toBe(end.view);
    summary.reopen();
    expect(summary.dismissed).toBe(false);
  });

  it('resets on undo and captures a freshly reached pending choice again', () => {
    const summary = new AgeSummary();
    const end = boundary();
    summary.record(end);
    summary.record(chosen(end));
    summary.close();
    summary.record({ ...end, history: [], events: [] });
    expect(summary.dismissed).toBe(false);
    expect(summary.shown).toBe(false);
    expect(summary.starter).toBeNull();
    const before = structuredClone(end);
    before.view.age--;
    before.view.ply--;
    delete before.view.pending;
    before.history = [];
    before.events = [];
    summary.record(before);
    expect(summary.view).toBeNull();
    summary.record(end);
    expect(summary.view).toBe(end.view);
  });

  it('preserves an unread summary when undo rewinds a later move within the next age', () => {
    const summary = new AgeSummary();
    const end = boundary();
    summary.record(end);
    const start = chosen(end);
    summary.record({ ...start, view: { ...start.view, ply: start.view.ply + 2 } });
    summary.record({ ...start, history: [] });
    expect(summary.view).toBe(end.view);
    expect(summary.starter).toBe(1);
  });

  it('isolates rooms and does not invent an age summary on a mid-age join', () => {
    const summary = new AgeSummary();
    const end = boundary();
    summary.record(end);
    summary.close();
    const other = { ...chosen(end), code: 'OTHER1', events: [] };
    summary.record(other);
    expect(summary.view).toBeNull();
    summary.record({ ...end, code: 'OTHER1' });
    expect(summary.dismissed).toBe(false);
    summary.record({ ...other, view: { ...other.view, phase: 'wonderDraft' } });
    expect(summary.view).toBeNull();
  });

  it('keeps an unread completed age even when the opponent ends the game', () => {
    const summary = new AgeSummary();
    const end = boundary();
    summary.record(end);
    const start = chosen(end);
    summary.record(start);
    summary.markShown();
    summary.record({
      ...start,
      view: { ...start.view, phase: 'gameOver', ply: start.view.ply + 1 }
    });
    expect(summary.view).toBe(end.view);
    expect(summary.isDismissed()).toBe(false);
    expect(summary.shown).toBe(true);
    summary.close();
    expect(summary.isDismissed()).toBe(true);
  });

  it('keeps a separate unread summary for each pass-device holder', () => {
    const summary = new AgeSummary();
    const end = boundary();
    summary.record(end);
    summary.markShown(0);
    expect(summary.isShown(0)).toBe(true);
    expect(summary.isShown(1)).toBe(false);
    expect(summary.isShown()).toBe(false);
    summary.close(0);
    expect(summary.isDismissed(0)).toBe(true);
    expect(summary.isDismissed(1)).toBe(false);
    expect(summary.isDismissed()).toBe(false);
    summary.close(1);
    summary.reopen(0);
    expect(summary.isDismissed(0)).toBe(false);
    expect(summary.isDismissed(1)).toBe(true);
    summary.record(chosen(end));
    summary.record({ ...end, history: [], events: [] });
    expect(summary.isShown(0)).toBe(false);
    expect(summary.isShown(1)).toBe(false);
    expect(summary.isDismissed(0)).toBe(false);
    expect(summary.isDismissed(1)).toBe(false);
  });

  it('queues both ended ages until each holder closes their own recap', () => {
    const summary = new AgeSummary();
    const first = boundary();
    const second = structuredClone(first);
    second.view.age++;
    second.view.ply += 21;
    second.view.players[0].coins += 7;
    summary.record(first);
    summary.markShown(0);
    summary.close(1);
    summary.record(chosen(first));
    summary.record(second);
    const selected = chosen(second);
    selected.view.active = 0;
    selected.events = [{ kind: 'starterChosen', seat: 0, starter: 0, private: false }];
    summary.record(selected);
    expect(summary.viewFor(0)).toBe(first.view);
    expect(summary.starterFor(0)).toBe(1);
    expect(summary.isShown(0)).toBe(true);
    expect(summary.viewFor(1)).toBe(second.view);
    expect(summary.isShown(1)).toBe(false);
    summary.markShown(1);
    summary.close(0);
    expect(summary.viewFor(0)).toBe(second.view);
    expect(summary.starterFor(0)).toBe(0);
    expect(summary.isShown(0)).toBe(false);
    expect(summary.isShown(1)).toBe(true);
    summary.close(1);
    expect(summary.isDismissed(1)).toBe(true);
    expect(summary.isDismissed(0)).toBe(false);
  });

  it('undo removes only obsolete queued boundaries and room changes reset all', () => {
    const summary = new AgeSummary();
    const first = boundary();
    const second = structuredClone(first);
    second.view.age++;
    second.view.ply += 21;
    summary.record(first);
    summary.markShown();
    summary.record(chosen(first));
    summary.record(second);
    summary.record(chosen(second));
    summary.record({ ...second, history: [], events: [] });
    expect(summary.view).toBe(first.view);
    expect(summary.shown).toBe(true);
    summary.close();
    expect(summary.view).toBe(second.view);
    expect(summary.starter).toBeNull();
    expect(summary.shown).toBe(false);
    const earlier = chosen(first);
    earlier.history = [];
    earlier.events = [];
    summary.record(earlier);
    expect(summary.view).toBe(first.view);
    expect(summary.dismissed).toBe(true);
    summary.record({ ...earlier, code: 'OTHER1' });
    expect(summary.view).toBeNull();
  });

  it('never labels post-move totals as the completed age when only a late starter event arrives', () => {
    const summary = new AgeSummary();
    const late = chosen(boundary());
    late.view.ply++;
    late.view.structure[0]!.state = 'empty';
    summary.record(late);
    expect(summary.view).toBeNull();
  });

  it('can present a starter-chosen first frame without needing a pending frame', () => {
    const summary = new AgeSummary();
    const start = chosen(boundary());
    summary.record(start);
    expect(summary.view).toBe(start.view);
    expect(summary.starter).toBe(1);
  });
});
