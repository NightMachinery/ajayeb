// Sound and haptics (M10.S04): the Settings switches, and the cues a
// hot-seat game's moves make. CI has no audio device (and the config mutes
// Chrome), so the spec reads what would have played from the page's fx
// test hook, window.__ajayebFx (web/src/lib/fx/index.ts), never from audio.
//
// Each case opens a named scenario (server/internal/engine/
// scenarios_named.go) just before its trigger and plays the trigger with
// the clicks a player makes. The cue mapper itself is unit-tested
// (web/src/lib/fx/cues.test.ts); this checks the wiring: the setting, the
// Presenter's call site, and that an opened room's first snapshot is silent.

import { expect, type Page } from '@playwright/test';
import { test } from '../fixtures/base';
import { watchHygiene } from '../fixtures/hygiene';
import { openHotseatRoom, playKey } from '../fixtures/scenario';
import type { GameDriver } from '../fixtures/driver';

interface RecordedCue {
  cue: string;
  delayMs: number;
  version: number;
}

interface FxHook {
  cues(): RecordedCue[];
  haptics(): number[][];
  clear(): void;
}

async function recorded(page: Page): Promise<{ cues: RecordedCue[]; haptics: number[][] }> {
  return page.evaluate(() => {
    const fx = (window as unknown as { __ajayebFx?: FxHook }).__ajayebFx;
    return { cues: fx?.cues() ?? [], haptics: fx?.haptics() ?? [] };
  });
}

async function clearRecorded(page: Page): Promise<void> {
  await page.evaluate(() => (window as unknown as { __ajayebFx?: FxHook }).__ajayebFx?.clear());
}

async function canVibrate(page: Page): Promise<boolean> {
  return page.evaluate(() => typeof navigator.vibrate === 'function');
}

/** Turns Sound on in the /settings page and checks it sticks across a reload. */
async function turnSoundOn(page: Page): Promise<void> {
  await page.goto('/settings');
  const sound = page.locator('input[data-setting="sound"]');
  await expect(sound).toBeChecked();
  await expect(page.locator('[data-setting="sound-volume"]')).toBeVisible();
  await page.reload();
  await expect(page.locator('input[data-setting="sound"]')).toBeChecked();
}

/** The cue names of the frames recorded since the last clear, in order. */
async function cueNames(page: Page): Promise<string[]> {
  return (await recorded(page)).cues.map((c) => c.cue);
}

/** Opens a scenario and checks that its first snapshot played nothing. */
async function openSilently(page: Page, scenario: string): Promise<GameDriver> {
  const driver = await openHotseatRoom(page, scenario);
  expect(await recorded(page), `${scenario}: the opening snapshot is silent`).toEqual({
    cues: [],
    haptics: []
  });
  return driver;
}

test.describe('sound and haptics', { tag: ['@mobile'] }, () => {
  test('Settings: fresh sound is on, explicit off persists, haptics match the device', async ({
    page
  }, testInfo) => {
    const problems = watchHygiene(page);
    await page.goto('/settings');
    await expect(page.locator('input[data-setting="sound"]')).toBeChecked();
    await page.locator('input[data-setting="sound"]').uncheck();
    await page.reload();
    await expect(page.locator('input[data-setting="sound"]')).not.toBeChecked();
    await expect(page.locator('[data-setting="sound-volume"]')).toHaveCount(0);
    const haptics = page.locator('input[data-setting="haptics"]');
    const touch = await page.evaluate(() => navigator.maxTouchPoints > 0);
    if ((await canVibrate(page)) && touch) {
      await expect(haptics).toBeChecked();
    } else {
      await expect(haptics).toHaveCount(0);
    }
    // Desktop Chrome has navigator.vibrate, but no touch: no switch there.
    if (testInfo.project.name === 'desktop') await expect(haptics).toHaveCount(0);
    if (testInfo.project.name === 'mobile') await expect(haptics).toBeChecked();
    await page.locator('input[data-setting="sound"]').check();
    const volume = page.locator('[data-setting="sound-volume"]');
    await volume.getByText('Loud').click();
    await page.reload();
    await expect(page.locator('[data-setting="sound-volume"] input[value="loud"]')).toBeChecked();
    const stored = await page.evaluate(() =>
      JSON.parse(localStorage.getItem('ajayeb:settings') ?? '{}')
    );
    expect(stored).toMatchObject({ sound: true, soundVolume: 'loud' });
    expect(problems, 'console errors, page errors or failed requests').toEqual([]);
  });

  test('with sound off, moves record no cues', async ({ page }) => {
    const problems = watchHygiene(page);
    const driver = await openSilently(page, 'looting-double');
    const toggle = page.locator('[data-control="sound"]');
    await expect(toggle).toHaveAttribute('aria-pressed', 'true');
    await toggle.click();
    await expect(toggle).toHaveAttribute('aria-pressed', 'false');
    await page.reload();
    await driver.waitReady();
    await expect(toggle).toHaveAttribute('aria-pressed', 'false');
    await playKey(driver, 'construct:arsenal');
    const { cues, haptics } = await recorded(page);
    expect(cues).toEqual([]);
    // Haptics is its own switch, on by default.
    if (await canVibrate(page)) expect(haptics).toEqual([[24, 40, 24, 40, 36]]);
    expect(problems, 'console errors, page errors or failed requests').toEqual([]);
  });

  test('moves play their cues: build, pay, pawn, looting, science and the end', async ({
    page
  }) => {
    const problems = watchHygiene(page);
    await turnSoundOn(page);
    const vibrates = await canVibrate(page);

    // Arsenal pays, pushes the pawn and loots twice, and taking it turns a
    // face-down card up: one cue of each kind, in first-event order, spaced.
    let driver = await openSilently(page, 'looting-double');
    await playKey(driver, 'construct:arsenal');
    expect(await cueNames(page)).toEqual(['slide', 'drum', 'loot']);
    const { cues, haptics } = await recorded(page);
    expect(cues.map((c) => c.delayMs)).toEqual([0, 90, 180]);
    if (vibrates) expect(haptics).toEqual([[24, 40, 24, 40, 36]]);

    // A science card that pairs a symbol, then the progress token it earns.
    driver = await openSilently(page, 'science-pair');
    await playKey(driver, 'construct:library');
    const pair = await cueNames(page);
    expect(pair).toContain('slide');
    expect(pair).toContain('chime');
    expect(pair.filter((c) => c === 'chime')).toHaveLength(1);
    if (vibrates) expect((await recorded(page)).haptics).toEqual([[8]]);
    await clearRecorded(page);
    const choose = await driver.nextTurn();
    const token = choose.choices.find((c) => c.kind === 'token');
    expect(token, 'the pair offers a progress token').toBeTruthy();
    await playKey(driver, token?.key ?? '');
    expect(await cueNames(page)).toContain('token');

    // The walls win by military: the drum, then the fanfare.
    driver = await openSilently(page, 'military-supremacy');
    await playKey(driver, 'construct:walls');
    const end = await cueNames(page);
    expect(end).toContain('drum');
    expect(end.at(-1)).toBe('fanfare');
    expect(problems, 'console errors, page errors or failed requests').toEqual([]);
  });
});

for (const locale of ['en', 'fa'] as const) {
  test(
    `board sound switch is one tap away at review widths (${locale})`,
    { tag: ['@mobile'] },
    async ({ page }, info) => {
      const problems = watchHygiene(page);
      await page.addInitScript((locale) => {
        const current = JSON.parse(localStorage.getItem('ajayeb:settings') ?? '{}');
        localStorage.setItem('ajayeb:settings', JSON.stringify({ ...current, locale }));
      }, locale);
      const driver = await openHotseatRoom(page, 'looting-double');
      // Give the top bar its widest state: undo, summary, log and sound.
      await playKey(driver, 'construct:arsenal');
      await driver.nextTurn();
      for (const { layout, ...viewport } of [
        { width: 360, height: 740, layout: 'phone-portrait' },
        { width: 390, height: 844, layout: 'phone-portrait' },
        { width: 740, height: 360, layout: 'phone-landscape' },
        { width: 1280, height: 800, layout: 'desktop' }
      ]) {
        await page.setViewportSize(viewport);
        // CSS media queries switch before the script's MediaQuery runes do, so
        // measure only once the game screen has taken the layout for this size.
        await expect(page.locator('.game-screen')).toHaveAttribute('data-layout', layout);
        const toggle = page.locator('[data-control="sound"]');
        await expect(toggle).toBeVisible();
        await expect(toggle).toHaveAccessibleName(
          locale === 'en' ? 'Turn sound off' : 'خاموش کردن صدا'
        );
        // Keep the localized ordinal age and full spoken count even when
        // the narrow header shows only the count's localized digits.
        await expect(page.locator('[data-tut="age"]')).toContainText(
          locale === 'en' ? 'III' : 'سوم'
        );
        if (viewport.width < 416) {
          const count = page.locator('.top-bar [aria-hidden="true"][data-must-fit] + .sr-only');
          await expect(count).toHaveText(locale === 'en' ? '19 cards left' : '۱۹ کارت مانده');
          await expect(page.locator('.top-bar [aria-hidden="true"][data-must-fit]')).toHaveText(
            locale === 'en' ? '19' : '۱۹'
          );
        }
        for (const control of ['menu', 'undo', 'move-summary', 'log', 'sound']) {
          const target = await page
            .locator(`.top-bar [data-control="${control}"]`)
            .evaluate((node) => {
              const r = node.getBoundingClientRect();
              return {
                fits: r.width >= 44 && r.height >= 44 && r.x >= 0 && r.right <= innerWidth,
                rect: { x: r.x, right: r.right, width: r.width, height: r.height },
                innerWidth,
                layout: document.querySelector('.game-screen')?.getAttribute('data-layout')
              };
            });
          expect(
            target.fits,
            `${control} keeps its 44px target inside the viewport: ${JSON.stringify(target)}`
          ).toBe(true);
        }
        expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(
          true
        );
        expect(
          await toggle.evaluate((node) => {
            const r = node.getBoundingClientRect();
            return r.width >= 44 && r.height >= 44 && r.x >= 0 && r.right <= innerWidth;
          })
        ).toBe(true);
        const overlaps = await page.locator('.top-bar').evaluate((header) => {
          const problems: string[] = [];
          for (const label of header.querySelectorAll('[data-must-fit]')) {
            const range = document.createRange();
            range.selectNodeContents(label);
            for (const r of range.getClientRects()) {
              if (r.width < 1 || r.height < 1) continue;
              for (const button of header.querySelectorAll('button')) {
                const b = button.getBoundingClientRect();
                if (
                  Math.min(r.right, b.right) - Math.max(r.left, b.left) > 1 &&
                  Math.min(r.bottom, b.bottom) - Math.max(r.top, b.top) > 1
                )
                  problems.push(button.getAttribute('data-control') ?? 'button');
              }
            }
          }
          return problems;
        });
        expect(overlaps, 'top-bar labels must stay clear of the controls').toEqual([]);
        await info.attach(`board-sound-${locale}-${viewport.width}x${viewport.height}`, {
          body: await page.screenshot({
            path: info.outputPath(`board-sound-${locale}-${viewport.width}x${viewport.height}.png`)
          }),
          contentType: 'image/png'
        });
        await toggle.click();
        await expect(toggle).toHaveAttribute('aria-pressed', 'false');
        await expect(toggle).toHaveAccessibleName(
          locale === 'en' ? 'Turn sound on' : 'روشن کردن صدا'
        );
        await toggle.click();
        await expect(toggle).toHaveAttribute('aria-pressed', 'true');
      }
      expect(problems).toEqual([]);
    }
  );
}

test(
  'rare sound cues follow wonder, library, destroy, rebuild and age deal events',
  { tag: ['@mobile'] },
  async ({ page }) => {
    const problems = watchHygiene(page);
    for (const [scenario, keys, expected] of [
      ['appian-economy', ['wonder:appian-way:rostrum'], ['wonder', 'extraTurn']],
      [
        'great-library',
        ['wonder:great-library:rostrum', 'token:law'],
        ['wonder', 'libraryDraw', 'token']
      ],
      ['zeus-destroy', ['wonder:statue-of-zeus:rostrum', 'destroy:quarry'], ['wonder', 'destroy']],
      ['mausoleum', ['wonder:mausoleum:rostrum', 'revive:baths'], ['wonder', 'rebuild']],
      ['age-transition-chooser', ['discard:theater', 'starter:0'], ['ageStart']]
    ] as const) {
      const driver = await openSilently(page, scenario);
      for (const key of keys) await playKey(driver, key);
      const cues = await cueNames(page);
      for (const cue of expected) expect(cues, scenario).toContain(cue);
      expect(cues, scenario).not.toContain('yourTurn');
    }
    expect(problems).toEqual([]);
  }
);
