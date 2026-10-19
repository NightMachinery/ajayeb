// Shared engine audio checks, included in the opt-in WebKit matrix too.
// Count native graph starts, not only the existing "would play" cue hook,
// so unsupported decoding and successful synth fallback stay distinct.
import { readdirSync, readFileSync } from 'node:fs';
import { expect, type Page, type TestInfo } from '@playwright/test';
import { test } from '../fixtures/base';
import { openHotseatRoom, playKey } from '../fixtures/scenario';
import { audioSnapshot, audioVisibility, observeWebAudio } from '../fixtures/web-audio';
import { SOUND_FILES } from '../../web/src/lib/fx/sounds.gen';

const cueCount = Object.keys(SOUND_FILES).length;

// Vite names chunks by hash. Locate the metadata chunk by its generated URL
// content so the test catches accidental eager imports without guessing a name.
function metadataPath(): string {
  const dir = new URL('../../web/build/_app/immutable/chunks/', import.meta.url);
  const chunks = readdirSync(dir).filter(
    (file) =>
      file.endsWith('.js') &&
      readFileSync(new URL(file, dir), 'utf8').includes(SOUND_FILES.coin.url)
  );
  expect(chunks, 'one emitted sound metadata chunk').toHaveLength(1);
  return `/_app/immutable/chunks/${chunks[0]!}`;
}

function watchSoundRequests(page: Page) {
  const index = metadataPath();
  const metadata: string[] = [];
  const files: string[] = [];
  const responses: number[] = [];
  page.on('request', (request) => {
    const path = new URL(request.url()).pathname;
    if (path === index) metadata.push(request.url());
    if (path.startsWith('/sfx/')) files.push(request.url());
  });
  page.on('response', (response) => {
    if (new URL(response.url()).pathname.startsWith('/sfx/')) responses.push(response.status());
  });
  return { metadata, files, responses };
}

async function tapToUnlock(page: Page): Promise<void> {
  await page.locator('[data-tut="age"]').click();
  await expect.poll(async () => (await audioSnapshot(page)).states).toEqual(['running']);
  await page.keyboard.press('Escape');
}

async function reportPlayback(page: Page, info: TestInfo): Promise<void> {
  const snapshot = await audioSnapshot(page);
  expect(
    snapshot.fileStarts + snapshot.synthStarts,
    'a native audio source started'
  ).toBeGreaterThan(0);
  const mode =
    snapshot.fileStarts > 0
      ? snapshot.synthStarts > 0
        ? 'AAC buffers + synthesized fallback'
        : 'AAC buffer playback'
      : 'synthesized fallback';
  info.annotations.push({
    type: 'audio-path',
    description: `${info.title}: ${mode} (decoded=${snapshot.decoded.length}, failed=${snapshot.decodeErrors})`
  });
  await info.attach('audio-engine-path', {
    body: JSON.stringify({ project: info.project.name, mode, ...snapshot }),
    contentType: 'application/json'
  });
  console.log(
    `Audio graph (${info.project.name}): ${mode}; decoded=${snapshot.decoded.length}, failed=${snapshot.decodeErrors}`
  );
}

interface FxView {
  cues(): { cue: string }[];
  frames(): { steps: number }[];
  clear(): void;
}

async function clearFx(page: Page): Promise<void> {
  await page.evaluate(() => (window as unknown as { __ajayebFx?: FxView }).__ajayebFx?.clear());
}

async function fxState(page: Page) {
  return page.evaluate(() => {
    const fx = (window as unknown as { __ajayebFx?: FxView }).__ajayebFx;
    return { cues: fx?.cues() ?? [], frames: fx?.frames() ?? [] };
  });
}

test.describe('native browser audio', { tag: ['@mobile'] }, () => {
  test.beforeEach(async ({ page, browserName }) => {
    expect(['chromium', 'webkit']).toContain(browserName);
    await observeWebAudio(page);
  });

  test('first tap unlocks real audio, lazy metadata and AAC decode or a reported fallback', async ({
    page
  }, info) => {
    const requests = watchSoundRequests(page);
    const driver = await openHotseatRoom(page, 'looting-double');
    expect((await audioSnapshot(page)).states).toEqual([]);
    expect(requests.metadata).toEqual([]);
    expect(requests.files).toEqual([]);
    await tapToUnlock(page);
    await expect.poll(() => requests.responses.length).toBe(cueCount);
    expect(requests.responses.every((status) => status === 200)).toBe(true);
    expect(requests.metadata).toHaveLength(1);
    expect(requests.files.every((url) => /\.m4a\?v=[a-f0-9]{12}$/.test(url))).toBe(true);
    await expect
      .poll(async () => {
        const audio = await audioSnapshot(page);
        return audio.decoded.length + audio.decodeErrors;
      })
      .toBe(cueCount);
    const audio = await audioSnapshot(page);
    expect(audio.available).toBe(true);
    for (const buffer of audio.decoded) {
      expect(buffer.channels).toBe(1);
      expect(buffer.duration).toBeGreaterThan(0);
      expect(buffer.duration).toBeLessThan(3.1);
    }
    await playKey(driver, 'construct:arsenal');
    const playback = await audioSnapshot(page);
    if (audio.decodeErrors === 0) expect(playback.fileStarts).toBeGreaterThan(0);
    if (audio.decoded.length === 0) {
      // Linux WebKit may have no AAC decoder in its GStreamer installation.
      // A pass then requires real fallback playback, not just failed decoding.
      expect(audio.decodeErrors).toBe(cueCount);
      expect(playback.fileStarts).toBe(0);
      expect(playback.synthStarts).toBeGreaterThan(0);
    }
    await reportPlayback(page, info);
  });

  test('missing AAC files use real synthesized sources without breaking a move', async ({
    page
  }, info) => {
    await page.route('**/sfx/**', (route) => route.fulfill({ status: 404, body: '' }));
    const requests = watchSoundRequests(page);
    const driver = await openHotseatRoom(page, 'looting-double');
    await tapToUnlock(page);
    await expect.poll(() => requests.responses.length).toBe(cueCount);
    await playKey(driver, 'construct:arsenal');
    const audio = await audioSnapshot(page);
    expect(audio.decoded).toEqual([]);
    expect(audio.fileStarts).toBe(0);
    expect(audio.synthStarts).toBeGreaterThan(0);
    await reportPlayback(page, info);
  });

  test('native AAC decoding rejection uses synthesized fallback', async ({ page }, info) => {
    await page.route('**/sfx/**', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'audio/mp4',
        body: Buffer.from('invalid AAC payload')
      })
    );
    const driver = await openHotseatRoom(page, 'looting-double');
    await tapToUnlock(page);
    await expect.poll(async () => (await audioSnapshot(page)).decodeErrors).toBe(cueCount);
    await playKey(driver, 'construct:arsenal');
    const audio = await audioSnapshot(page);
    expect(audio.decoded).toEqual([]);
    expect(audio.fileStarts).toBe(0);
    expect(audio.synthStarts).toBeGreaterThan(0);
    await reportPlayback(page, info);
  });

  test('a simulated hide/show and real context suspension recover on the next tap', async ({
    page
  }, info) => {
    const driver = await openHotseatRoom(page, 'looting-double');
    await tapToUnlock(page);
    await audioVisibility(page, 'hidden');
    expect(await page.evaluate(() => document.visibilityState)).toBe('hidden');
    expect((await audioSnapshot(page)).states).toEqual(['suspended']);
    await audioVisibility(page, 'visible');
    expect(await page.evaluate(() => document.visibilityState)).toBe('visible');
    await tapToUnlock(page);
    await playKey(driver, 'construct:arsenal');
    await reportPlayback(page, info);
  });

  for (const animationSpeed of ['off', 'normal'] as const) {
    test(`Sound mutes new sources, while Animation ${animationSpeed} remains independent`, async ({
      page
    }, info) => {
      await page.addInitScript((speed) => {
        const settings = JSON.parse(localStorage.getItem('ajayeb:settings') ?? '{}');
        localStorage.setItem(
          'ajayeb:settings',
          JSON.stringify({
            ...settings,
            sound: false,
            animationSpeed: speed,
            hotseatPassDevice: false
          })
        );
      }, animationSpeed);
      const requests = watchSoundRequests(page);
      const driver = await openHotseatRoom(page, 'looting-double');
      await playKey(driver, 'construct:arsenal');
      expect((await audioSnapshot(page)).states).toEqual([]);
      expect(requests.files).toEqual([]);
      expect(requests.metadata).toEqual([]);
      expect((await fxState(page)).cues).toEqual([]);
      const frames = (await fxState(page)).frames;
      expect(frames.length).toBeGreaterThan(0);
      expect(frames.some((frame) => frame.steps > 0)).toBe(animationSpeed === 'normal');
      const toggle = page.locator('[data-control="sound"]');
      await toggle.click();
      await expect(toggle).toHaveAttribute('aria-pressed', 'true');
      await driver.undo();
      await playKey(driver, 'construct:arsenal');
      await reportPlayback(page, info);
      await toggle.click();
      await expect(toggle).toHaveAttribute('aria-pressed', 'false');
      await clearFx(page);
      const before = await audioSnapshot(page);
      await driver.undo();
      await playKey(driver, 'construct:arsenal');
      const after = await audioSnapshot(page);
      expect(after.fileStarts + after.synthStarts).toBe(before.fileStarts + before.synthStarts);
      expect((await fxState(page)).cues).toEqual([]);
    });
  }
});
