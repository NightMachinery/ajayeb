import { describe, expect, it } from 'vitest';
import type { TutorialState } from '$lib/wire';
import { ackKey, CLOSED_UI, coachStep, parseKey, type CoachUi, type StepInput } from './steps';

function tut(over: Partial<TutorialState> = {}): TutorialState {
  return {
    script: 'first-game',
    status: 'guided',
    left: false,
    turn: 4,
    turns: 10,
    lessons: [],
    ...over
  };
}

function input(tutorial: TutorialState | undefined, over: Partial<StepInput> = {}): StepInput {
  return {
    tutorial,
    code: 'ROOM',
    ply: 6,
    paused: false,
    acked: () => false,
    ui: CLOSED_UI,
    ...over
  };
}

const ui = (over: Partial<CoachUi>): CoachUi => ({ ...CLOSED_UI, ...over });

const build = tut({
  lessons: [{ id: 'ageOne' }, { id: 'turn' }],
  action: { id: 'buildFree', key: 'construct:lumber-yard', card: 'lumber-yard' }
});

describe('the coach step machine', () => {
  it('shows nothing outside a tutorial room, while paused, or in the free part with nothing to read', () => {
    expect(coachStep(input(undefined))).toBeNull();
    expect(coachStep(input(build, { paused: true }))).toBeNull();
    expect(coachStep(input(tut({ status: 'free' })))).toBeNull();
    expect(coachStep(input(tut({ status: 'complete' })))).toBeNull();
  });

  it('shows the info lessons first, in order, each until acknowledged', () => {
    const first = coachStep(input(build));
    expect(first).toMatchObject({
      id: 'ageOne',
      kind: 'info',
      surface: 'board',
      target: ['[data-tut="structure"]'],
      region: true,
      turn: 4,
      turns: 10,
      ack: ackKey('ROOM', 6, 0, 'ageOne')
    });
    const acked = new Set([ackKey('ROOM', 6, 0, 'ageOne')]);
    const second = coachStep(input(build, { acked: (k) => acked.has(k) }));
    expect(second).toMatchObject({ id: 'turn', kind: 'info', target: ['[data-tut="turn-bar"]'] });
    expect(second?.key).not.toBe(first?.key);
  });

  it('then taps the card, and acts in its sheet', () => {
    const all = () => true;
    expect(coachStep(input(build, { acked: all }))).toMatchObject({
      id: 'buildFree',
      lesson: 'action',
      kind: 'tap',
      surface: 'board',
      words: 'tapCard',
      card: 'lumber-yard',
      target: ['[data-tut="structure"] [data-card="lumber-yard"]'],
      actionKey: 'construct:lumber-yard'
    });
    expect(
      coachStep(
        input(build, { acked: all, ui: ui({ sheet: { card: 'lumber-yard', step: 'card' } }) })
      )
    ).toMatchObject({
      kind: 'act',
      surface: 'card-sheet',
      words: 'lesson',
      target: ['[data-act="construct:lumber-yard"]']
    });
  });

  it('a sheet on another card, or a face-down one, names the expected card with no target', () => {
    const all = () => true;
    for (const card of ['stone-pit', null]) {
      expect(
        coachStep(input(build, { acked: all, ui: ui({ sheet: { card, step: 'card' } }) }))
      ).toMatchObject({ kind: 'note', surface: 'card-sheet', words: 'otherCard', target: null });
    }
    // The right card, but its wonder step: back to the card.
    expect(
      coachStep(
        input(build, { acked: all, ui: ui({ sheet: { card: 'lumber-yard', step: 'wonders' } }) })
      )
    ).toMatchObject({ kind: 'note', words: 'back', target: null });
  });

  it('a discard acts on its own key', () => {
    const t = tut({ action: { id: 'discard', key: 'discard:stone-pit', card: 'stone-pit' } });
    expect(
      coachStep(input(t, { ui: ui({ sheet: { card: 'stone-pit', step: 'card' } }) }))
    ).toMatchObject({ kind: 'act', target: ['[data-act="discard:stone-pit"]'] });
  });

  it('a wonder: the card, then Build a wonder, then the wonder in its step', () => {
    const t = tut({
      lessons: [],
      action: {
        id: 'buildWonder',
        key: 'wonder:great-lighthouse:glassworks',
        card: 'glassworks',
        wonder: 'great-lighthouse'
      }
    });
    expect(coachStep(input(t))).toMatchObject({
      kind: 'tap',
      surface: 'board',
      target: ['[data-tut="structure"] [data-card="glassworks"]'],
      wonder: 'great-lighthouse'
    });
    expect(
      coachStep(input(t, { ui: ui({ sheet: { card: 'glassworks', step: 'card' } }) }))
    ).toMatchObject({
      kind: 'tap',
      surface: 'card-sheet',
      words: 'openWonders',
      target: ['[data-control="wonder-step"]']
    });
    expect(
      coachStep(input(t, { ui: ui({ sheet: { card: 'glassworks', step: 'wonders' } }) }))
    ).toMatchObject({
      kind: 'act',
      surface: 'card-sheet',
      words: 'lesson',
      target: ['[data-act="wonder:great-lighthouse:glassworks"]']
    });
    expect(
      coachStep(input(t, { ui: ui({ sheet: { card: 'baths', step: 'wonders' } }) }))
    ).toMatchObject({ kind: 'note', words: 'otherCard' });
  });

  it('the draft: Take on the page (split), else the tile, then Take in the phone sheet', () => {
    const t = tut({
      turn: 1,
      lessons: [],
      action: { id: 'draftPick', key: 'draft:temple-of-artemis', wonder: 'temple-of-artemis' }
    });
    expect(coachStep(input(t, { ui: ui({ split: true }) }))).toMatchObject({
      kind: 'act',
      surface: 'board',
      words: 'lesson',
      target: ['[data-act="draft:temple-of-artemis"]']
    });
    expect(coachStep(input(t))).toMatchObject({
      kind: 'tap',
      surface: 'board',
      words: 'takeWonder',
      target: ['[data-pick="draft:temple-of-artemis"]']
    });
    expect(coachStep(input(t, { ui: ui({ draftSheet: 'temple-of-artemis' }) }))).toMatchObject({
      kind: 'act',
      surface: 'draft-sheet',
      target: ['[data-act="draft:temple-of-artemis"]']
    });
    expect(coachStep(input(t, { ui: ui({ draftSheet: 'mausoleum' }) }))).toMatchObject({
      kind: 'note',
      surface: 'draft-sheet',
      words: 'otherWonder',
      target: null
    });
  });

  it('no step while the wonder-first pick bar is up', () => {
    expect(coachStep(input(build, { acked: () => true, ui: ui({ pickBar: true }) }))).toBeNull();
  });

  it('points each info lesson at its anchor', () => {
    const at = (id: string, card?: string) =>
      coachStep(input(tut({ status: 'free', lessons: [card ? { id, card } : { id }] })))?.target;
    expect(at('draftIntro')).toEqual(['[data-tut="draft-offer"]']);
    expect(at('reveal', 'baths')).toEqual(['[data-tut="structure"] [data-card="baths"]']);
    expect(at('military')).toEqual(['[data-tut="track"] [data-pawn]']);
    expect(at('coins')).toEqual(['[data-tut="home"] [data-tally="coins"]']);
    expect(at('wonders')).toEqual(['[data-tut="home"] [data-tut="wonders"]']);
    expect(coachStep(input(tut({ status: 'free', lessons: [{ id: 'wonders' }] })))?.region).toBe(
      true
    );
    expect(at('tokens')).toEqual(['[data-tut="tokens"]']);
    expect(at('playOn')).toEqual(['[data-tut="turn-bar"]']);
    expect(at('leftPath')).toEqual(['[data-tut="turn-bar"]']);
    // A lesson this client does not know: a generic step with no target.
    expect(at('someday')).toBeNull();
  });

  it('the outro and leftPath show in the free part, and the extra lessons after the server', () => {
    const outro = tut({
      status: 'free',
      turn: 0,
      turns: 0,
      lessons: [{ id: 'tokens' }, { id: 'playOn' }]
    });
    expect(coachStep(input(outro))).toMatchObject({ id: 'tokens', kind: 'info', turn: 0 });
    const left = tut({ status: 'free', left: true, turn: 0, turns: 0, lessons: [] });
    expect(coachStep(input(left))).toBeNull();
    expect(coachStep(input(left, { extra: [{ id: 'leftPath' }] }))).toMatchObject({
      id: 'leftPath',
      ack: ackKey('ROOM', 6, 0, 'leftPath')
    });
  });

  it('parses action keys', () => {
    expect(parseKey('construct:baths')).toEqual({ verb: 'construct', card: 'baths' });
    expect(parseKey('wonder:pyramids:baths')).toEqual({
      verb: 'wonder',
      wonder: 'pyramids',
      card: 'baths'
    });
    expect(parseKey('draft:piraeus')).toEqual({ verb: 'draft', wonder: 'piraeus' });
    expect(parseKey('starter:0')).toEqual({ verb: 'starter' });
  });
});
