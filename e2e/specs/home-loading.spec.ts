// Cold Home only loads what it paints; deferred sheets still start real rooms.
import { writeFile } from 'node:fs/promises';
import { expect } from '@playwright/test';
import { test } from '../fixtures/base';
import { GameDriver } from '../fixtures/driver';
import { watchHygiene } from '../fixtures/hygiene';
import { joinByLink, newDeviceContext } from '../fixtures/online';

test.describe('Home loading', { tag: '@mobile' }, () => {
  test('enforced CSP permits Home and a hero failure before hydration shows its fallback', async ({
    page,
    baseURL
  }, info) => {
    if (!baseURL) throw new Error('local test origin required');
    const errors: string[] = [];
    let enforcedPolicy = '';
    page.on('console', (message) => {
      if (message.type() === 'error') errors.push(message.text());
    });
    page.on('pageerror', (error) => errors.push(error.message));
    let release!: () => void;
    const scripts = new Promise<void>((resolve) => {
      release = resolve;
    });
    await page.route('**/_app/immutable/**/*.js', async (route) => {
      await scripts;
      await route.continue();
    });
    await page.route('**/art/ui/hero*.avif*', (route) => route.abort());
    await page.route(baseURL + '/', async (route) => {
      const response = await route.fetch();
      const headers = { ...response.headers() };
      const policy = headers['content-security-policy-report-only'];
      if (!policy) throw new Error('expected the normal Go development CSP');
      enforcedPolicy = policy;
      headers['content-security-policy'] = policy;
      delete headers['content-security-policy-report-only'];
      await route.fulfill({ response, headers });
    });
    try {
      await page.goto('/', { waitUntil: 'commit' });
      await expect(page.locator('[data-hero] img')).toBeVisible();
      await expect
        .poll(() =>
          page
            .locator('[data-hero] img')
            .evaluate((image: HTMLImageElement) => image.complete && image.naturalWidth === 0)
        )
        .toBe(true);
      const html = await page.content();
      expect(html).not.toMatch(/<img[^>]+on(?:load|error)=/);
      release();
      await expect(page.locator('[data-hero] svg')).toBeVisible();
      await expect(page.locator('[data-hero] img')).toHaveCount(0);
      await page.waitForLoadState('networkidle');
      // The intentionally aborted art request is expected; CSP/JS errors are not.
      expect(errors.filter((error) => !error.includes('net::ERR_FAILED'))).toEqual([]);
      const policyPath = info.outputPath('enforced-csp-policy.txt');
      await writeFile(policyPath, enforcedPolicy);
      await info.attach('enforced-csp-policy', { path: policyPath, contentType: 'text/plain' });
      await page.getByRole('button', { name: /Play vs AI/ }).click();
      await expect(page.locator('dialog[open] h2')).toHaveText('Choose your rival');
    } finally {
      release();
    }
  });

  test('returning device state hydrates without a tutorial offer or lost Continue rooms', async ({
    page
  }) => {
    const problems = watchHygiene(page);
    const hydration: string[] = [];
    page.on('console', (message) => {
      if (/hydration/i.test(message.text())) hydration.push(message.text());
    });
    await page.addInitScript(() => {
      localStorage.setItem('ajayeb:theme', 'dark');
      localStorage.setItem(
        'ajayeb:rooms',
        JSON.stringify([{ code: 'ABCDEF', mode: 'ai', age: 2, bot: 'normal', lastSeen: 1 }])
      );
      localStorage.setItem(
        'ajayeb:settings',
        JSON.stringify({ locale: 'fa', animationSpeed: 'off', sound: false })
      );
      localStorage.setItem(
        'ajayeb:tips',
        JSON.stringify({ off: true, seen: [], tutorial: { offer: 'declined', done: true } })
      );
      const state = window as unknown as { __offerPainted?: boolean };
      new MutationObserver(() => {
        if (document.querySelector('[data-tutorial-offer]')) state.__offerPainted = true;
      }).observe(document, { childList: true, subtree: true });
    });
    await page.goto('/');
    await expect(page.locator('html')).toHaveAttribute('lang', 'fa');
    await expect(page.locator('html')).toHaveAttribute('dir', 'rtl');
    await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark');
    await expect(page.locator('html')).toHaveAttribute('data-motion', 'reduce');
    await expect(page.getByRole('button', { name: /ABCDEF/ })).toBeVisible();
    await expect(page.locator('[data-continue-details]')).toContainText('۲');
    await expect(page.locator('[data-tutorial-replay]')).toBeVisible();
    await expect(page.locator('[data-tutorial-offer]')).toHaveCount(0);
    expect(
      await page.evaluate(() => (window as { __offerPainted?: boolean }).__offerPainted)
    ).not.toBe(true);
    expect(hydration).toEqual([]);
    expect(problems).toEqual([]);
  });

  test('the real Home paints without scripts, while deep links receive the generic shell', async ({
    browser,
    baseURL,
    request
  }) => {
    if (!baseURL) throw new Error('local test origin required');
    const context = await browser.newContext({
      javaScriptEnabled: false,
      serviceWorkers: 'block',
      viewport: { width: 390, height: 844 }
    });
    try {
      const page = await context.newPage();
      await page.goto(baseURL);
      await expect(page.getByRole('heading', { name: 'Ajayeb', exact: true })).toBeVisible();
      await expect(page.locator('[data-hero] img')).toBeVisible();
      await expect(page.locator('[data-hero] img')).toHaveCSS('opacity', '1');
      await expect(page.getByRole('button', { name: /Play vs AI/ })).toBeVisible();
      await expect(page.locator('link[rel="preload"][as="image"]')).toHaveCount(1);
    } finally {
      await context.close();
    }
    const home = await (await request.get(baseURL)).text();
    const head = home.slice(0, home.indexOf('</head>'));
    const firstModule = head.indexOf('rel="modulepreload"');
    expect(firstModule).toBeGreaterThan(head.indexOf('imagesrcset='));
    expect(firstModule).toBeGreaterThan(head.lastIndexOf('rel="stylesheet"'));
    expect(firstModule).toBeGreaterThan(head.indexOf('as="font"'));
    for (const route of ['/play/ABCDEF', '/settings', '/rules', '/shell.html']) {
      const response = await request.get(new URL(route, baseURL).href);
      expect(response.ok()).toBe(true);
      expect(await response.text()).not.toContain('data-hero');
    }
  });

  test('priority art, deferred messages and sheets, one live session through Continue', async ({
    page
  }, info) => {
    const problems = watchHygiene(page);
    const scripts: Promise<string>[] = [];
    const sockets: string[] = [];
    page.on('websocket', (socket) => sockets.push(socket.url()));
    page.on('response', (response) => {
      if (response.request().resourceType() === 'script') scripts.push(response.text());
    });
    await page.goto('/');
    await expect(page.locator('h1')).toHaveText('Ajayeb');
    await page.waitForLoadState('networkidle');
    expect((await Promise.all(scripts)).join('\n')).not.toContain('کارتی نمانده');
    expect(sockets).toEqual([]);
    await expect(page.locator('dialog')).toHaveCount(0);
    const hero = page.locator('[data-hero] img');
    await expect(hero).toHaveAttribute('loading', 'eager');
    await expect(hero).toHaveAttribute('fetchpriority', 'high');
    await expect(hero).toHaveCSS('opacity', '1');
    const preload = page.locator('link[rel="preload"][as="image"]');
    await expect(preload).toHaveAttribute('imagesrcset', (await hero.getAttribute('srcset'))!);
    await expect(preload).toHaveAttribute('imagesizes', (await hero.getAttribute('sizes'))!);

    const start = page.getByRole('button', { name: /Play vs AI/ });
    await start.click();
    const sheet = page.locator('dialog[open]');
    await expect(sheet.getByRole('heading', { name: 'Choose your rival' })).toBeVisible();
    await expect(sheet.locator('[data-avatar="easy"] img')).toHaveAttribute('loading', 'lazy');
    await expect(sheet.locator('[data-avatar="easy"] img')).toHaveCSS('opacity', '1');
    await sheet.getByText('Apprentice', { exact: true }).click();
    await page.keyboard.press('Escape');
    await expect(start).toBeFocused();
    await start.click();
    await expect(sheet.getByRole('radio', { name: /Apprentice/ })).toBeChecked();
    await sheet.getByText('Sun', { exact: true }).click();
    await sheet.getByRole('button', { name: 'Start game' }).click();
    await page.waitForURL(/\/play\/[A-Z0-9]+$/);
    const driver = new GameDriver(page);
    await driver.waitReady();
    const code = await page.evaluate(() => window.__ajayeb?.room()?.code);
    expect(await page.evaluate(() => window.__ajayeb?.room()?.mode)).toBe('ai');
    expect(sockets).toHaveLength(1);
    await driver.visible('[data-control="menu"]').click();
    await driver.visible('[data-control="menu-home"]').click();
    await page.waitForURL((url) => url.pathname === '/');
    await page.screenshot({ path: info.outputPath('continue-home.png') });
    await page.getByRole('button', { name: `Continue: vs AI · ${code}` }).click();
    await page.waitForURL((url) => url.pathname === `/play/${code}`);
    await driver.waitReady();
    expect(sockets).toHaveLength(1);
    expect(problems).toEqual([]);
  });

  test('query AI, hot-seat, online and join use the deferred entry points', async ({
    page,
    browser
  }, info) => {
    const problems = watchHygiene(page);
    await page.goto('/?play=ai');
    await expect(page.locator('dialog[open] h2')).toHaveText('Choose your rival');
    await page.waitForURL((url) => url.pathname === '/' && !url.search);
    await page.keyboard.press('Escape');
    await page.getByRole('button', { name: /Pass and play/ }).click();
    const sheet = page.locator('dialog[open]');
    await sheet.getByLabel('Player 1 (Sun crest)').fill('First');
    await sheet.getByLabel('Player 2 (Moon crest)').fill('Second');
    await sheet.getByRole('button', { name: 'Start game' }).click();
    await page.waitForURL(/\/play\/[A-Z0-9]+$/);
    await new GameDriver(page).waitReady();
    expect(await page.evaluate(() => window.__ajayeb?.room()?.mode)).toBe('hotseat');

    await page.goto('/');
    await page.getByRole('button', { name: /Play online/ }).click();
    const online = page.locator('dialog[open]');
    await online.locator('[data-name-input]').fill('First');
    await online.getByText('Sun', { exact: true }).click();
    await page.screenshot({ path: info.outputPath('online-sheet.png') });
    await online.locator('[data-control="online-create"]').click();
    await page.waitForURL(/\/play\/[A-Z0-9]+$/);
    await expect(page.locator('[data-lobby]')).toBeVisible();
    const code = await page.locator('[data-lobby]').getAttribute('data-lobby');
    if (!code) throw new Error('online lobby has no room code');
    const other = await newDeviceContext(browser, info);
    try {
      const joiner = await other.newPage();
      const otherProblems = watchHygiene(joiner);
      await joiner.goto('/');
      await joiner.locator('[data-control="home-join"]').click();
      await joiner.locator('dialog[open] [data-code-input]').fill(code);
      await joiner.locator('dialog[open] [data-control="join-submit"]').click();
      await joiner.waitForURL((url) => url.pathname === `/play/${code}`);
      await joinByLink(joiner, joiner.url(), 'Second');
      await new GameDriver(page).waitReady();
      await new GameDriver(joiner).waitReady();
      expect(await page.evaluate(() => window.__ajayeb?.room()?.status)).toBe('playing');
      expect(otherProblems).toEqual([]);
    } finally {
      await other.close();
    }
    expect(problems).toEqual([]);
  });

  test('a pending Persian import cannot override a later English choice', async ({ page }) => {
    let release!: () => void;
    const held = new Promise<void>((resolve) => {
      release = resolve;
    });
    let requested!: () => void;
    const arrived = new Promise<void>((resolve) => {
      requested = resolve;
    });
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    await page.route('**/_app/immutable/chunks/*.js', async (route) => {
      const response = await route.fetch();
      if ((await response.text()).includes('کارتی نمانده')) {
        requested();
        await held;
      }
      await route.fulfill({ response });
    });
    await page.getByRole('button', { name: 'Settings', exact: true }).click();
    const language = (value: string) =>
      page.locator(`label:has(> input[name="settings-language"][value="${value}"])`);
    await language('fa').click();
    await arrived;
    await expect(page.locator('html')).toHaveAttribute('lang', 'en');
    await expect(page.locator('dialog[open] h2')).toHaveText('Settings');
    await language('en').click();
    release();
    await page.waitForLoadState('networkidle');
    await expect(page.locator('html')).toHaveAttribute('lang', 'en');
    await language('fa').click();
    await expect(page.locator('html')).toHaveAttribute('lang', 'fa');
    await expect(page.locator('html')).toHaveAttribute('dir', 'rtl');
    await expect(page.locator('dialog[open] h2')).toHaveText('تنظیمات');
    await page.reload();
    await expect(page.locator('html')).toHaveAttribute('lang', 'fa');
    await expect(page.getByRole('button', { name: 'تنظیمات', exact: true })).toBeVisible();
  });
});
