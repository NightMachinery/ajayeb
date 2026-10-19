// Unit tests for the wonder draft screen (Draft.svelte): Take buttons use
// actButtonState consistently across connection state, sent state, canAct,
// confirmation, and tutorial readiness. A locked tutorial action key remains
// non-actionable.
import { render } from 'svelte/server';
import { describe, expect, it } from 'vitest';
import { t } from '$lib/i18n';
import { protocolFixtures } from '$lib/wire/fixtures.gen';
import type { View } from '$lib/wire';
import Draft from './Draft.svelte';

const draftFixture = protocolFixtures.find((view) => view.phase === 'wonderDraft');
if (!draftFixture) throw new Error('expected a recorded wonder draft fixture');

const viewBase: View = {
  ...draftFixture,
  viewer: 0,
  toMove: 0,
  choices: [
    { key: 'draft:pyramids', kind: 'draft', wonder: 'pyramids' },
    { key: 'draft:mausoleum', kind: 'draft', wonder: 'mausoleum' }
  ],
  draft: {
    round: 1,
    order: [0, 1, 1, 0],
    offer: [{ wonder: 'pyramids' }, { wonder: 'mausoleum' }]
  }
};

describe('Draft screen Take button states', () => {
  it('renders Take buttons enabled when canAct and connected', () => {
    const { body } = render(Draft, {
      props: {
        view: viewBase,
        detail: 'pyramids',
        canAct: true,
        connected: true,
        onact: () => {}
      }
    });
    expect(body).toContain('data-act="draft:pyramids"');
    expect(body).not.toMatch(/data-act="draft:pyramids"[^>]*disabled/);
  });

  it('disables Take buttons and shows connection waiting text when disconnected', () => {
    const { body } = render(Draft, {
      props: {
        view: viewBase,
        detail: 'pyramids',
        canAct: true,
        connected: false,
        onact: () => {}
      }
    });
    expect(body).toContain(t('online.waitingConnection'));
    expect(body).toMatch(/data-act="draft:pyramids"[^>]*disabled/);
  });

  it('disables Take buttons when canAct is false (sent state)', () => {
    const { body } = render(Draft, {
      props: {
        view: viewBase,
        detail: 'pyramids',
        canAct: false,
        connected: true,
        onact: () => {}
      }
    });
    expect(body).toContain('data-act="draft:pyramids"');
    expect(body).toMatch(/data-act="draft:pyramids"[^>]*disabled/);
  });

  it('locks non-allowed actions in tutorial readiness and marks them with data-tutorial-locked', () => {
    const { body } = render(Draft, {
      props: {
        view: viewBase,
        detail: 'mausoleum',
        canAct: true,
        connected: true,
        allow: (key: string) => key === 'draft:pyramids',
        lockedReason: 'Take the Pyramids first.',
        onact: () => {}
      }
    });
    // mausoleum is locked
    expect(body).toContain('data-act="draft:mausoleum"');
    expect(body).toContain('data-tutorial-locked');
    expect(body).toContain('data-tutorial-reason');
    expect(body).toContain('Take the Pyramids first.');
    expect(body).toMatch(/data-act="draft:mausoleum"[^>]*disabled/);
  });
});
