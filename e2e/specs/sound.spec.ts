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
  await expect(sound).not.toBeChecked();
  await expect(page.locator('[data-setting="sound-volume"]')).toHaveCount(0);
  await sound.check();
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
  test('Settings: sound off by default, haptics only where the device vibrates', async ({
    page
  }) => {
    const problems = watchHygiene(page);
    await page.goto('/settings');
    await expect(page.locator('input[data-setting="sound"]')).not.toBeChecked();
    const haptics = page.locator('input[data-setting="haptics"]');
    if (await canVibrate(page)) {
      await expect(haptics).toBeChecked();
    } else {
      await expect(haptics).toHaveCount(0);
    }
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
    expect(await cueNames(page)).toEqual(['slide', 'coin', 'drum', 'flip']);
    const { cues, haptics } = await recorded(page);
    expect(cues.map((c) => c.delayMs)).toEqual([0, 90, 180, 270]);
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
    expect(await cueNames(page)).toContain('chime');

    // The walls win by military: the drum, then the fanfare.
    driver = await openSilently(page, 'military-supremacy');
    await playKey(driver, 'construct:walls');
    const end = await cueNames(page);
    expect(end).toContain('drum');
    expect(end.at(-1)).toBe('fanfare');
    expect(problems, 'console errors, page errors or failed requests').toEqual([]);
  });
});
