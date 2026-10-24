// Home-created production lobbies, legal offered keys and per-frame privacy.
// The discard search shows the server's reason for each pile card it does
// not offer, and on the sixth turn a card's sheet and the hand show the
// card each choice keeps, both read from the searching or choosing seat's
// own frame.
// Until the parallel lobby control lands, the test adds the already-supported
// sides option to the host's ordinary seating edit, through its real socket.
import { expect, type Page } from '@playwright/test';
import { mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { test } from '../fixtures/base';
import { newDeviceContext } from '../fixtures/online';
import { createTableFromHome, lobbyReady } from '../fixtures/lobby';
import {
  dismissMilitary,
  latest,
  pendingOwnership,
  ready,
  traffic,
  type Traffic
} from '../fixtures/table';
import { dataDir } from '../fixtures/global-setup';
import { watchHygiene } from '../fixtures/hygiene';
import { quiet } from '../fixtures/layout-probes';
import type { Choice } from '../../web/src/lib/wire/haftgane.gen';
import type { TableState } from '../../web/src/lib/wire/platform.gen';
import { hgDesigns, type HgDesign } from '../../web/src/lib/wire/haftgane-catalog.gen';

function chooseKey(f: TableState, olympiaNight: boolean): Choice {
  const cs = f.view.you!.choices;
  const city = f.view.seats[f.you.seat!]!;
  if (f.view.phase === 'sides')
    return cs.find(
      (c) => c.side === (city.board === 'olympia' && !olympiaNight ? 'day' : 'night')
    )!;
  if (f.view.phase !== 'turn') return cs[0]!;
  if (city.board === 'halikarnassos' && f.view.turn < 6 && city.coins < 8)
    return cs.find((c) => c.kind === 'sell')!;
  const stage = cs.find((c) => c.kind === 'stage');
  const needsAbility =
    (city.board === 'olympia' && city.stages < 2) || (city.board === 'babylon' && city.stages < 1);
  if (stage && (needsAbility || (city.board === 'halikarnassos' && f.view.turn === 6)))
    return stage;
  // Bank enough coins to buy the ability stage when resources become available.
  // Merely building the first offered card can spend the purse every turn.
  if (needsAbility && city.coins < 10) return cs.find((c) => c.kind === 'sell')!;
  if (city.board === 'olympia') {
    const free = cs.find((c) => ['first-colour', 'first-turn', 'last-turn'].includes(c.free));
    if (free) return free;
  }
  return (
    cs.find(
      (c) =>
        c.kind === 'build' &&
        ['brown', 'grey'].includes((hgDesigns as Record<string, HgDesign>)[c.design]!.colour)
    ) ??
    cs.find((c) => c.kind === 'build') ??
    cs[0]!
  );
}
/** The phone and desktop captures in both themes, for the screens that show
 *  the view's own new fields, with `focus` scrolled into view. The theme
 *  follows the emulated system (each context stores 'system'). */
async function themedShots(p: Page, out: string, id: string, focus?: string): Promise<void> {
  for (const scheme of ['light', 'dark'] as const) {
    await p.emulateMedia({ colorScheme: scheme });
    for (const [w, h] of [
      [360, 740],
      [1280, 800]
    ] as const) {
      await p.setViewportSize({ width: w, height: h });
      if (focus) await p.locator(focus).first().scrollIntoViewIfNeeded();
      await quiet(p);
      await p.screenshot({ path: path.join(out, `${id}-${scheme}-${w}x${h}.png`) });
    }
  }
  await p.emulateMedia({ colorScheme: 'light' });
}

/**
 * Plays the choice through the screen. A card's sheet names the card each
 * of its choices keeps on the sixth turn (the choices' `kept`), and once
 * the choice is drafted the hand marks that card. Returns whether the
 * sheet showed a kept card.
 */
async function submit(page: Page, c: Choice, choices: readonly Choice[]): Promise<boolean> {
  const special = c.kind === 'side' || c.kind === 'build-discarded';
  let previewed = false;
  if (special) {
    await page.locator(`[data-act="${c.key}"]`).first().click();
    await ready(page);
    await page.locator('[data-special-confirm]').click();
  } else {
    await page.locator(`[data-act="hand:${c.design}"]`).first().click();
    await expect(page.locator('[data-card-sheet]')).toBeVisible();
    const keeps = [
      ...new Set(choices.filter((x) => x.design === c.design && x.kept).map((x) => x.kept!))
    ].sort();
    const shown = await page
      .locator('[data-card-sheet] [data-kept-preview]')
      .evaluateAll((els) => els.map((el) => el.getAttribute('data-kept-preview')!));
    expect([...new Set(shown)].sort()).toEqual(keeps);
    previewed = keeps.length > 0;
    const draft = page.locator(`[data-card-sheet] [data-pick="${c.key}"]`);
    if (await draft.isVisible()) {
      await draft.click();
      await ready(page);
      await expect(page.locator('[data-keeps-preview]')).toHaveCount(c.kept ? 1 : 0);
      if (c.kept) await expect(page.locator(`[data-keeps-preview="${c.kept}"]`)).toHaveCount(1);
    }
    await page.locator(`[data-card-sheet] .sheet-footer [data-act="${c.key}"]`).click();
  }
  await ready(page);
  return previewed;
}

test('special decisions reached from Home keep each frame private @full @mobile @haftgane-special', async ({
  browser
}, info) => {
  test.setTimeout(1_500_000);
  const found = new Set<string>();
  const out = process.env.HG_SPECIAL_LIVE_DIR ?? '../.cache/e2e/hg-special-live';
  mkdirSync(out, { recursive: true });
  const evidence: { id: string; phase: string; point: string; seat: number }[] = [];
  const attempts: { side: string; attempt: number; seed: number; found: string[] }[] = [];
  for (const olympiaNight of [false, true]) {
    const required = olympiaNight
      ? ['first-turn', 'last-turn', 'discard', 'kept', 'kept-preview', 'last', 'side', 'sixth']
      : ['first-colour'];
    // Seeds are assigned by the shared server counter. A deal can lack the
    // production needed for an ability under this test policy, so try another
    // real Home lobby rather than assuming which deal this spec receives.
    for (let attempt = 1; attempt <= 4; attempt++) {
      const contexts = await Promise.all(
        Array.from({ length: 8 }, () => newDeviceContext(browser, info))
      );
      try {
        for (const [i, ctx] of contexts.entries())
          await ctx.addInitScript((n) => {
            localStorage.setItem('ajayeb:name', `Player ${n}`);
            localStorage.setItem('ajayeb:theme', 'system');
            const s = JSON.parse(localStorage.getItem('ajayeb:settings') ?? '{}');
            localStorage.setItem(
              'ajayeb:settings',
              JSON.stringify({ ...s, confirmMoves: 'never' })
            );
          }, i);
        const pages = await Promise.all(contexts.map((ctx) => ctx.newPage()));
        const logs: Traffic[][] = pages.map((p) => traffic(p));
        // Routed sockets do not produce the ordinary page websocket events.
        const hostLog: Traffic[] = [];
        await pages[0]!.routeWebSocket('**/ws', (ws) => {
          const peer = ws.connectToServer();
          ws.onMessage((raw) => {
            const msg = JSON.parse(String(raw)) as Record<string, unknown>;
            if (msg['t'] === 'lobby-edit' && msg['op'] === 'options')
              msg['options'] = { ...(msg['options'] as object), sides: 'choose' };
            hostLog.push({ dir: 'sent', msg });
            peer.send(JSON.stringify(msg));
          });
          peer.onMessage((raw) => {
            hostLog.push({ dir: 'recv', msg: JSON.parse(String(raw)) as Record<string, unknown> });
            ws.send(raw);
          });
        });
        logs[0] = hostLog;
        const hygiene = pages.map(watchHygiene);
        const host = pages[0]!;
        const code = await createTableFromHome(host, 'Player 0');
        await host.locator('[data-act="lobby-seats"]').selectOption('7');
        await lobbyReady(host);
        await host.locator('[data-act="lobby-seating"]').selectOption('lobby');
        await lobbyReady(host);
        for (const p of pages.slice(1, 7)) {
          await p.goto(`/play/${code}`);
          await lobbyReady(p);
        }
        await expect
          .poll(
            async () =>
              (await host.evaluate(() => window.__ajayebTable!.lobby()))?.seats.filter(
                (s) => s.kind === 'human'
              ).length
          )
          .toBe(7);
        await host.locator('[data-act="lobby-start"]').click();
        await latest(host);
        await pages[7]!.goto(`/play/${code}`);
        await latest(pages[7]!);
        for (let step = 0; step < 30; step++) {
          const before = await latest(host);
          if (before.status === 'finished') break;
          for (const p of pages.slice(0, 7)) {
            const f = await latest(p);
            // The end of an Age opens the conflicts over the next decision.
            await dismissMilitary(p);
            const seat = f.you.seat!;
            if (f.view.you?.kept && f.view.phase === 'discard-build') {
              found.add('kept');
              const id = `${olympiaNight}-kept`;
              if (!evidence.some((x) => x.id === id)) {
                await expect(p.locator('[data-kept-card] button')).toHaveAttribute(
                  'aria-disabled',
                  'true'
                );
                for (const [w, h] of [
                  [360, 740],
                  [390, 844],
                  [844, 390],
                  [1280, 800]
                ]) {
                  await p.setViewportSize({ width: w!, height: h! });
                  await quiet(p);
                  await p.screenshot({ path: path.join(out, `${id}-own-${w}x${h}.png`) });
                }
                evidence.push({ id, phase: f.view.phase, point: f.point!.ref.id, seat });
              }
            }
            if (!f.point?.required.includes(seat)) continue;
            const c = chooseKey(f, olympiaNight);
            const kind =
              f.view.phase === 'sides'
                ? 'side'
                : f.view.phase === 'discard-build'
                  ? 'discard'
                  : f.view.phase === 'last-card'
                    ? 'last'
                    : f.view.turn === 6
                      ? 'sixth'
                      : '';
            if (kind) found.add(kind);
            if (kind === 'discard') {
              // The searching seat reads why each pile card it cannot
              // build is not offered: the city already has its name.
              const you = f.view.you!;
              const barred = you.offer.filter((o) => !o.legal).map((o) => o.design);
              expect(you.disabled.map((d) => `${d.kind}:${d.design}:${d.reason}`).sort()).toEqual(
                barred.map((d) => `build-discarded:${d}:already-built`).sort()
              );
              const picker = p.locator('[data-discard-picker]');
              await expect(picker.locator('[data-blocked-reason="already-built"]')).toHaveCount(
                barred.length
              );
              await expect(picker.locator('[data-blocked-reason=""]')).toHaveCount(0);
              const id = `${olympiaNight}-discard-reasons`;
              if (barred.length > 0 && !evidence.some((x) => x.id === id)) {
                await themedShots(p, out, id, '[data-blocked-reason="already-built"]');
                evidence.push({ id, phase: f.view.phase, point: f.point!.ref.id, seat });
              }
            }
            const keepsHere = f.view.you!.choices.some((x) => x.design === c.design && x.kept);
            const keptId = `${olympiaNight}-kept-preview`;
            if (kind === 'sixth' && keepsHere && !evidence.some((x) => x.id === keptId)) {
              await p.locator(`[data-act="hand:${c.design}"]`).first().click();
              await expect(
                p.locator('[data-card-sheet] [data-kept-preview]').first()
              ).toBeVisible();
              await themedShots(p, out, keptId);
              // Drafted, the choice's kept card is tagged in the hand.
              const draft = p.locator(`[data-card-sheet] [data-pick="${c.key}"]`);
              const tagged = c.kept !== undefined && (await draft.isVisible());
              if (tagged) {
                await draft.click();
                await ready(p);
              }
              await p.locator('[data-act="pick-close"]').click();
              if (tagged) {
                await expect(p.locator(`[data-keeps-preview="${c.kept}"]`)).toBeVisible();
                await themedShots(p, out, `${keptId}-hand`, '[data-keeps-preview]');
              }
              evidence.push({ id: keptId, phase: f.view.phase, point: f.point!.ref.id, seat });
            }
            for (const x of f.view.you!.choices)
              if (['first-colour', 'first-turn', 'last-turn'].includes(x.free)) found.add(x.free);
            const capture =
              kind === 'sixth' ||
              kind === 'side' ||
              kind === 'discard' ||
              kind === 'last' ||
              (c.free !== '' && c.free !== 'chain');
            const captureKind = c.free && c.free !== 'chain' ? c.free : kind;
            const captureId = `${olympiaNight}-${captureKind}`;
            let confirmedCapture: string | null = null;
            if (capture && !evidence.some((x) => x.id === captureId)) {
              if (kind === 'last' || (c.free !== '' && c.kind === 'build'))
                await p.locator(`[data-act="hand:${c.design}"]`).first().click();
              for (const [w, h] of [
                [360, 740],
                [390, 844],
                [844, 390],
                [1280, 800]
              ]) {
                await p.setViewportSize({ width: w!, height: h! });
                await quiet(p);
                await p.screenshot({ path: path.join(out, `${captureId}-own-${w}x${h}.png`) });
              }
              if (kind === 'last' || (c.free !== '' && c.kind === 'build'))
                await p.locator('[data-act="pick-close"]').click();
              let waiting: Page | undefined;
              for (const q of pages.slice(0, 7)) {
                if (q === p) continue;
                const other = await latest(q);
                if (
                  !other.point?.required.includes(other.you.seat!) ||
                  other.point.confirmed.includes(other.you.seat!)
                ) {
                  waiting = q;
                  break;
                }
              }
              if (waiting) {
                await dismissMilitary(waiting);
                await waiting.screenshot({ path: path.join(out, `${captureId}-waiting.png`) });
              } else confirmedCapture = captureId;
              evidence.push({ id: captureId, phase: f.view.phase, point: f.point.ref.id, seat });
            }
            if (await submit(p, c, f.view.you!.choices)) found.add('kept-preview');
            if ((await latest(host)).view.phase === 'discard-build' && !found.has('kept')) {
              for (const q of pages.slice(0, 7)) {
                const kept = await latest(q);
                if (!kept.view.you?.kept) continue;
                found.add('kept');
                await expect(q.locator('[data-kept-card] button')).toHaveAttribute(
                  'aria-disabled',
                  'true'
                );
                const id = `${olympiaNight}-kept`;
                for (const [w, h] of [
                  [360, 740],
                  [390, 844],
                  [844, 390],
                  [1280, 800]
                ]) {
                  await q.setViewportSize({ width: w!, height: h! });
                  await quiet(q);
                  await q.screenshot({ path: path.join(out, `${id}-own-${w}x${h}.png`) });
                }
                evidence.push({
                  id,
                  phase: kept.view.phase,
                  point: kept.point!.ref.id,
                  seat: kept.you.seat!
                });
                break;
              }
            }
            if (confirmedCapture) {
              const confirmed = await latest(p);
              expect(confirmed.pending?.confirm).toBe(true);
              await expect(
                p.locator('[data-side-picker], [data-discard-picker], [data-card-sheet]')
              ).toHaveCount(0);
              await p.screenshot({ path: path.join(out, `${confirmedCapture}-waiting.png`) });
            }
          }
          await expect
            .poll(async () => (await latest(host)).point?.ref.id)
            .not.toBe(before.point?.ref.id);
        }
        for (const [i, log] of logs.entries()) {
          expect(pendingOwnership(`player ${i}`, log)).toEqual([]);
          for (const { dir, msg } of log)
            if (dir === 'recv' && msg['t'] === 'table-state') {
              const f = msg as unknown as TableState;
              if (f.you.seat === undefined) expect(f.view.you).toBeUndefined();
              if (f.view.phase === 'discard-build' && !f.point?.required.includes(f.you.seat!)) {
                expect(f.view.you?.offer ?? []).toEqual([]);
                expect(
                  f.view.you?.disabled.filter((d) => d.kind === 'build-discarded') ?? []
                ).toEqual([]);
              }
              // A kept-card preview names a card in the viewer's own hand.
              for (const c of f.view.you?.choices ?? [])
                if (c.kept) expect(f.view.you!.hand).toContain(c.kept);
            }
          expect(hygiene[i]).toEqual([]);
        }
        const record = JSON.parse(
          readFileSync(path.join(dataDir, 'rooms', `${code}.json`), 'utf8')
        ) as { setup: { seed: number } };
        attempts.push({
          side: olympiaNight ? 'night' : 'day',
          attempt,
          seed: record.setup.seed,
          found: [...found].sort()
        });
        console.log('special-decision attempt', JSON.stringify(attempts.at(-1)));
        writeFileSync(
          path.join(out, `frames-${olympiaNight ? 'night' : 'day'}-attempt-${attempt}.json`),
          JSON.stringify({ record, logs })
        );
        writeFileSync(
          path.join(out, `frames-${olympiaNight ? 'night' : 'day'}.json`),
          JSON.stringify({ record, logs })
        );
      } finally {
        for (const ctx of contexts) await ctx.close();
      }
      writeFileSync(path.join(out, 'attempts.json'), JSON.stringify(attempts, null, 2));
      if (required.every((kind) => found.has(kind))) break;
    }
    expect(
      required.filter((kind) => !found.has(kind)),
      `missing decisions after four ${olympiaNight ? 'night' : 'day'} tables; attempts: ${JSON.stringify(attempts)}`
    ).toEqual([]);
  }
  writeFileSync(
    path.join(out, 'evidence.json'),
    JSON.stringify({ found: [...found], attempts, evidence }, null, 2)
  );
  expect([...found].sort()).toEqual([
    'discard',
    'first-colour',
    'first-turn',
    'kept',
    'kept-preview',
    'last',
    'last-turn',
    'side',
    'sixth'
  ]);
});
