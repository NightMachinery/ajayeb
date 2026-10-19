// The room error screen, rendered on the server (M10.F2): a forbidden join,
// which only a pass-and-play or bot game started in another browser gives,
// says why and points at the device link; other codes keep their messages.
import { render } from 'svelte/server';
import { describe, expect, it } from 'vitest';
import { i18n, t } from '$lib/i18n';
import RoomError from './RoomError.svelte';

const screen = (error: string) =>
  render(RoomError, { props: { error, code: 'ABC234', onhome: () => {} } }).body;

describe('the room error screen', () => {
  it('explains a forbidden join and names the device link', () => {
    const body = screen('forbidden');
    expect(body).toContain('data-room-error="forbidden"');
    expect(body).toContain(t('online.error.forbidden'));
    expect(body).toContain(t('online.error.forbiddenHint'));
    expect(body).not.toContain(t('session.error.forbidden'));
    expect(body).not.toContain(t('online.error.retryHint'));
  });

  it('explains an archived game and offers Home and backup recovery', () => {
    const body = screen('room-archived');
    expect(body).toContain(t('session.error.roomArchived'));
    expect(body).toContain(t('online.error.archivedHint'));
    expect(body).toContain('data-control="error-home"');
    expect(body).not.toContain(t('online.error.retryHint'));
    expect(body).not.toContain('server update');
  });

  it('renders the archived message and recovery guidance in Persian', async () => {
    await i18n.set('fa');
    try {
      const body = screen('room-archived');
      expect(body).toContain(t('session.error.roomArchived'));
      expect(body).toContain(t('online.error.archivedHint'));
      expect(body).not.toContain('به‌روزرسانی');
    } finally {
      await i18n.set('en');
    }
  });

  it('keeps the not-found and retry messages', () => {
    expect(screen('room-not-found')).toContain(t('online.error.notFoundHint'));
    const busy = screen('busy');
    expect(busy).toContain(t('session.error.busy'));
    expect(busy).toContain(t('online.error.retryHint'));
  });
});
