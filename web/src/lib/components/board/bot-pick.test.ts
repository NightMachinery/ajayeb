// The bot's pick (M10.animations-events): while the Presenter previews a
// bot's answer, the game screen hands what it takes to the structure (a
// slot index) or the draft (a wonder) as their botPick prop, and the card
// or the offered wonder carries data-bot-pick. Rendered on the server.
import { render } from 'svelte/server';
import { describe, expect, it } from 'vitest';
import Draft from '$lib/components/screens/Draft.svelte';
import StructureSlot from '$lib/components/board/StructureSlot.svelte';
import { protocolFixtureNames, protocolFixtures } from '$lib/wire/fixtures.gen';
import type { View } from '$lib/wire';

function fixture(name: (typeof protocolFixtureNames)[number]): View {
  const v = protocolFixtures[protocolFixtureNames.indexOf(name)];
  if (!v) throw new Error(`no fixture ${name}`);
  return v;
}

const marked = (markup: string) => (markup.match(/data-bot-pick/g) ?? []).length;

describe('the bot pick prop', () => {
  const view = fixture('age-1-start-seat0');
  const slot = view.structure.find((s) => s.state === 'faceUp' && s.accessible)!;
  const props = (botPick?: boolean) => ({
    slot,
    place: { x: 0, y: 0, z: 1 },
    tier: 'mini' as const,
    width: 44,
    label: 'card',
    ...(botPick === undefined ? {} : { botPick })
  });

  it('marks a structure slot only while the bot is about to take it', () => {
    expect(marked(render(StructureSlot, { props: props(true) }).body)).toBe(1);
    expect(marked(render(StructureSlot, { props: props(false) }).body)).toBe(0);
    expect(marked(render(StructureSlot, { props: props() }).body)).toBe(0);
  });

  it('marks the offered wonder the bot is about to draft, and no other', () => {
    const draft = fixture('draft-round-1-seat0');
    const wonder = draft.draft?.offer[1]?.wonder;
    expect(wonder).toBeDefined();
    const base = { view: draft, canAct: false, onact: () => {}, names: ['A', 'B'] as const };
    const markup = render(Draft, { props: { ...base, botPick: wonder } }).body;
    expect(marked(markup)).toBe(1);
    expect(markup).toMatch(new RegExp(`data-wonder-offer="${wonder}"[^>]*data-bot-pick`));
    expect(marked(render(Draft, { props: base }).body)).toBe(0);
  });
});
