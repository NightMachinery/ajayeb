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

function getActionTag(html: string, key: string): string {
  const match = html.match(new RegExp(`<button\\b[^>]*\\bdata-act="${key}"[^>]*>`, 'i'));
  if (!match) throw new Error(`expected button with data-act="${key}"`);
  return match[0];
}

function isTagDisabled(tag: string): boolean {
  return /(?:^|\s)disabled(?:\s|=|>)/i.test(tag);
}

describe('Draft screen Take button states', () => {
  describe('component layout and confirmation branches', () => {
    it('renders standard Button in compact path with confirm: false', () => {
      const { body } = render(Draft, {
        props: {
          view: viewBase,
          split: false,
          detail: 'pyramids',
          confirm: false,
          canAct: true,
          connected: true,
          onact: () => {}
        }
      });
      const tag = getActionTag(body, 'draft:pyramids');
      expect(isTagDisabled(tag)).toBe(false);
      expect(tag).not.toContain('isolate overflow-hidden');
    });

    it('renders ConfirmButton in compact path with confirm: true', () => {
      const { body } = render(Draft, {
        props: {
          view: viewBase,
          split: false,
          detail: 'pyramids',
          confirm: true,
          canAct: true,
          connected: true,
          onact: () => {}
        }
      });
      const tag = getActionTag(body, 'draft:pyramids');
      expect(isTagDisabled(tag)).toBe(false);
      expect(tag).toContain('isolate overflow-hidden');
    });

    it('renders standard Button in split path with confirm: false', () => {
      const { body } = render(Draft, {
        props: {
          view: viewBase,
          split: true,
          confirm: false,
          canAct: true,
          connected: true,
          onact: () => {}
        }
      });
      const tag = getActionTag(body, 'draft:pyramids');
      expect(isTagDisabled(tag)).toBe(false);
      expect(tag).not.toContain('isolate overflow-hidden');
    });

    it('renders ConfirmButton in split path with confirm: true', () => {
      const { body } = render(Draft, {
        props: {
          view: viewBase,
          split: true,
          confirm: true,
          canAct: true,
          connected: true,
          onact: () => {}
        }
      });
      const tag = getActionTag(body, 'draft:pyramids');
      expect(isTagDisabled(tag)).toBe(false);
      expect(tag).toContain('isolate overflow-hidden');
    });
  });

  describe('action button states across paths', () => {
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
      const tag = getActionTag(body, 'draft:pyramids');
      expect(isTagDisabled(tag)).toBe(false);
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
      const tag = getActionTag(body, 'draft:pyramids');
      expect(isTagDisabled(tag)).toBe(true);
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
      const tag = getActionTag(body, 'draft:pyramids');
      expect(isTagDisabled(tag)).toBe(true);
    });

    it('locks non-allowed actions in tutorial readiness and marks them with data-tutorial-locked (compact)', () => {
      const { body } = render(Draft, {
        props: {
          view: viewBase,
          split: false,
          detail: 'mausoleum',
          canAct: true,
          connected: true,
          allow: (key: string) => key === 'draft:pyramids',
          lockedReason: 'Take the Pyramids first.',
          onact: () => {}
        }
      });
      const tag = getActionTag(body, 'draft:mausoleum');
      expect(tag).toContain('data-tutorial-locked');
      expect(isTagDisabled(tag)).toBe(true);
      expect(body).toContain('data-tutorial-reason');
      expect(body).toContain('Take the Pyramids first.');
    });

    it('locks non-allowed actions in tutorial readiness and shows prompt lock banner (split)', () => {
      const { body } = render(Draft, {
        props: {
          view: viewBase,
          split: true,
          canAct: true,
          connected: true,
          allow: (key: string) => key === 'draft:pyramids',
          lockedReason: 'Take the Pyramids first.',
          onact: () => {}
        }
      });
      const lockedTag = getActionTag(body, 'draft:mausoleum');
      expect(lockedTag).toContain('data-tutorial-locked');
      expect(isTagDisabled(lockedTag)).toBe(true);

      const allowedTag = getActionTag(body, 'draft:pyramids');
      expect(allowedTag).not.toContain('data-tutorial-locked');
      expect(isTagDisabled(allowedTag)).toBe(false);

      expect(body).toContain('data-tutorial-reason');
      expect(body).toContain('Take the Pyramids first.');
    });
  });
});
