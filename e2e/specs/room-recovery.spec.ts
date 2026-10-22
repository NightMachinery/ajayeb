// A real reconnect over a scripted socket must replace an already-visible board
// with RoomError, forget the dead Home entry, and keep the Home action usable.
import { expect, type WebSocketRoute } from '@playwright/test';
import { test } from '../fixtures/base';
import { watchHygiene } from '../fixtures/hygiene';
import { fixtureNames, fixtures } from '../../web/src/lib/wire/fixtures.gen';
import type { State } from '../../web/src/lib/wire';
import { session as english } from '../../web/src/lib/i18n/en/session';
import { session as persian } from '../../web/src/lib/i18n/fa/session';

const CODE = 'ARCTST';

for (const locale of ['en', 'fa'] as const) {
  test.describe(`unavailable room recovery in ${locale}`, () => {
    test.use({ appLocale: locale });

    test('an archived reconnect replaces the board and returns Home @mobile', async ({ page }) => {
      const problems = watchHygiene(page);
      await page.addInitScript(() => localStorage.setItem('ajayeb:name', 'Player'));
      const first = structuredClone(
        fixtures[(fixtureNames as readonly string[]).indexOf('state-play')]
      ) as State;
      first.code = CODE;
      first.version = 10;
      first.events = [];
      delete first.ack;
      const sockets: WebSocketRoute[] = [];
      await page.routeWebSocket('**/ws', (socket) => {
        let connection = 0;
        let version = 0;
        socket.onMessage((raw) => {
          const message = JSON.parse(String(raw)) as { t: string; v?: number };
          if (message.t === 'hello') {
            version = message.v ?? 0;
            if (version === 1) connection = sockets.push(socket);
            socket.send(
              JSON.stringify({
                t: 'welcome',
                v: version,
                server: {
                  version: connection === 1 ? 'before' : 'after',
                  rules: connection === 1 ? 1 : 2
                }
              })
            );
          }
          if (message.t === 'join') {
            socket.send(
              JSON.stringify(
                version === 2 || connection === 1
                  ? first
                  : {
                      t: 'error',
                      code: 'room-archived',
                      msg: locale === 'en' ? 'rules-version' : 'replay-failed'
                    }
              )
            );
          }
        });
      });
      await page.goto(`/play/${CODE}`);
      await expect(page.locator('.card-structure')).toBeVisible();
      await expect
        .poll(() => page.evaluate(() => localStorage.getItem('ajayeb:rooms')))
        .toContain(CODE);
      const initialSocket = sockets[0];
      if (!initialSocket) throw new Error('initial socket did not open');
      await initialSocket.close({ code: 1012, reason: 'test server restart' });
      const error = page.locator('[data-room-error="room-archived"]');
      await expect(error).toBeVisible();
      await expect(error).toContainText(
        (locale === 'en' ? english : persian)['session.error.roomArchived']
      );
      await expect(page.locator('.card-structure')).toHaveCount(0);
      expect(await page.evaluate(() => localStorage.getItem('ajayeb:rooms'))).not.toContain(CODE);
      await error.locator('[data-control="error-home"]').click();
      await expect(page).toHaveURL(/\/$/);
      expect(await page.evaluate(() => localStorage.getItem('ajayeb:rooms'))).not.toContain(CODE);
      expect(problems).toEqual([]);
    });
  });
}
