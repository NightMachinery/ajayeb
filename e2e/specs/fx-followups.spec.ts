// Regression for the production client's age deal against the ordinary
// dev/e2e server, which deliberately keeps --bot-delay 0.
import { expect } from '@playwright/test';
import { test } from '../fixtures/base';
import { fxRecords, openPosition, playThrough, useSettings, waitSettled } from '../fixtures/anim';
import { watchHygiene } from '../fixtures/hygiene';

test.describe('age deals with an immediate bot', { tag: '@mobile' }, () => {
  test('the last draft pick still deals Age I', async ({ page, baseURL }) => {
    if (!baseURL) throw new Error('no baseURL');
    const problems = watchHygiene(page);
    await useSettings(page, 'normal');
    const driver = await openPosition(page, baseURL, { mode: 'ai', seed: '11', keys: [] });
    let picks = 0;
    for (; picks < 8; picks++) {
      const snap = await driver.nextTurn();
      if (snap.phase !== 'wonderDraft') break;
      const choice = snap.choices[0];
      if (!choice) throw new Error('no draft choice');
      await playThrough(driver, choice.key);
    }
    expect(picks).toBeGreaterThan(0);
    await waitSettled(page);
    expect((await driver.snapshot()).phase).toBe('play');
    const deal = (await fxRecords(page)).anims.filter((a) => a.kind === 'deal');
    expect(deal.length, 'a deal survives the queued bot response').toBeGreaterThan(0);
    expect(
      deal.some((a) => !a.interrupted && a.endAt !== null),
      'the deal reaches a landing'
    ).toBe(true);
    expect(problems).toEqual([]);
  });

  test('a bot’s starter choice still deals Age II', async ({ page, baseURL }) => {
    if (!baseURL) throw new Error('no baseURL');
    const problems = watchHygiene(page);
    await useSettings(page, 'normal');
    const driver = await openPosition(page, baseURL, {
      mode: 'ai',
      seed: '11',
      scenario: 'age-transition-chooser',
      keys: []
    });
    await driver.nextTurn();
    await playThrough(driver, 'discard:theater');
    await driver.nextTurn();
    await waitSettled(page);
    const snap = await driver.snapshot();
    expect(snap.age).toBe(2);
    expect(snap.phase).toBe('play');
    const deal = (await fxRecords(page)).anims.filter((a) => a.kind === 'deal');
    expect(deal.length, 'a deal survives the queued starter answer').toBeGreaterThan(0);
    expect(
      deal.some((a) => !a.interrupted && a.endAt !== null),
      'the deal reaches a landing'
    ).toBe(true);
    expect(problems).toEqual([]);
  });
});
