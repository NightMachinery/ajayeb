// The wonder draft's Take button (M11.board-layout-fixes) is the move button
// every other move uses (game/ActionButton.svelte), so it follows the one
// act-button rule in a local room and with the confirm step alike: disabled
// while input is closed or the connection is down, and locked, with a
// reason, by the tutorial's rails. Rendered on the server, where the split
// layout's media query is false, so this is the phone's wonder sheet; the
// split layout renders the same snippet, and the e2e driver takes wonders
// through both.
import { render } from 'svelte/server';
import { describe, expect, it } from 'vitest';
import { t } from '$lib/i18n';
import { protocolFixtureNames, protocolFixtures } from '$lib/wire/fixtures.gen';
import type { View } from '$lib/wire';
import { wonderName } from '../card/effect-text';
import Draft from './Draft.svelte';

const view: View = (() => {
  const v = protocolFixtures[protocolFixtureNames.indexOf('draft-round-1-seat1')];
  if (!v) throw new Error('no fixture draft-round-1-seat1');
  return v;
})();
const keyOf = (wonder: string) => `draft:${wonder}`;
const [guided, other] = view.choices.flatMap((c) => (c.kind === 'draft' ? [c.wonder!] : []));

interface Shown {
  /** The sheet's markup, from its title on. */
  html: string;
  /** The sheet's one button tag. */
  tag: string;
  disabled: boolean;
  /** The data-act-group wrapper's tag, when there is a move to send. */
  group: string | null;
}

const isDisabled = (tag: string) => /\sdisabled(?=[\s>=])/.test(tag);

/** The phone's wonder sheet for `wonder`, and its Take button. */
function sheet(wonder: string, props: Record<string, unknown> = {}): Shown {
  const body = render(Draft, {
    props: {
      view,
      canAct: true,
      onact: () => {},
      names: ['Ana', 'Bob'] as const,
      ...props,
      detail: wonder
    }
  }).body;
  const html = body.slice(body.indexOf('id="draft-wonder-title"'));
  const buttons = html.match(/<button[^>]*>/g) ?? [];
  expect(buttons).toHaveLength(1);
  const tag = buttons[0]!;
  return {
    html,
    tag,
    disabled: isDisabled(tag),
    group: html.match(/<div[^>]*data-act-group[^>]*>/)?.[0] ?? null
  };
}

describe('the draft Take button', () => {
  it('has two offered wonders to take', () => {
    expect(guided).toBeDefined();
    expect(other).toBeDefined();
  });

  for (const confirm of [false, true]) {
    describe(confirm ? 'with the confirm step' : 'in a local room', () => {
      it('is enabled while input is open and the connection is up', () => {
        const s = sheet(guided!, { confirm });
        expect(s.tag).toContain(`data-act="${keyOf(guided!)}"`);
        expect(s.disabled).toBe(false);
        expect(s.group).not.toBeNull();
        expect(s.group).not.toContain('data-tutorial-locked');
        expect(s.html).toContain(t('choices.draft.take', { wonder: wonderName(guided!) }));
      });

      it('is disabled and waits for the connection while it is down', () => {
        const s = sheet(guided!, { confirm, connected: false });
        expect(s.disabled).toBe(true);
        expect(s.html).toContain(t('online.waitingConnection'));
      });

      it('is disabled once a move is sent (input closed)', () => {
        expect(sheet(guided!, { confirm, canAct: false }).disabled).toBe(true);
      });

      it('is locked by the rails, with a reason, on every wonder but the guided one', () => {
        const allow = (key: string) => key === keyOf(guided!);
        const rails = { confirm, allow, lockedReason: 'Take the guided wonder.' };
        const locked = sheet(other!, rails);
        expect(locked.disabled).toBe(true);
        expect(locked.group).toContain('data-tutorial-locked');
        expect(locked.html).toContain('data-tutorial-reason');
        expect(locked.html).toContain('Take the guided wonder.');
        const open = sheet(guided!, rails);
        expect(open.disabled).toBe(false);
        expect(open.group).not.toContain('data-tutorial-locked');
        expect(open.html).not.toContain('data-tutorial-reason');
      });
    });
  }

  it('shows a disabled button with no move key on a wonder this client cannot take now', () => {
    const s = sheet(guided!, { view: { ...view, choices: [] } });
    expect(s.disabled).toBe(true);
    expect(s.tag).not.toContain('data-act');
    expect(s.group).toBeNull();
  });
});
