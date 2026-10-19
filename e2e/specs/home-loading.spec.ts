// Cold Home only loads what it paints; deferred sheets still start real rooms.
import { writeFile } from 'node:fs/promises';
import { expect } from '@playwright/test';
import { test } from '../fixtures/base';
import { GameDriver } from '../fixtures/driver';
import { watchHygiene } from '../fixtures/hygiene';
import { joinByLink, newDeviceContext } from '../fixtures/online';

test.describe('Home loading', { tag: '@mobile' }, () => {
  for (const action of [
    { name: 'Play vs AI', heading: 'Choose your rival' },
    { name: 'Pass and play', heading: 'Pass and play' },
    { name: 'Play online', heading: 'Play online' },
    { name: 'Learn to play', tutorial: true },
    { name: 'Join with a code', heading: 'Join a game' },
    { name: 'Settings', heading: 'Settings' }
  ]) {
    test(`early Home activation: ${action.name}`, async ({ page, isMobile }, info) => {
      let release!: () => void;
      const held = new Promise<void>((resolve) => {
        release = resolve;
      });
      await page.route('**/_app/immutable/**/*.js', async (route) => {
        // Both the initial app and deferred sheets face the slow network.
        await Promise.all([held, new Promise((resolve) => setTimeout(resolve, 1500))]);
        await route.continue();
      });
      try {
        await page.goto('/', { waitUntil: 'commit' });
        await expect(page.getByRole('heading', { name: 'Ajayeb', exact: true })).toBeVisible();
        const control = page.getByRole('button', { name: new RegExp(action.name) });
        await expect(control).toBeDisabled();
        await expect(control).toHaveCSS('cursor', 'wait');
        await expect(page.getByRole('status')).toHaveText('Getting ready…');
        await expect(page.getByRole('radio', { name: 'Dark', exact: true })).toBeDisabled();
        await control.scrollIntoViewIfNeeded();
        await page.screenshot({ path: info.outputPath('before-start.png') });
        // A real pointer press is harmless while native disabled; it is never queued.
        const bounds = await control.boundingBox();
        if (!bounds) throw new Error('missing Home control');
        if (isMobile)
          await page.touchscreen.tap(bounds.x + bounds.width / 2, bounds.y + bounds.height / 2);
        else await page.mouse.click(bounds.x + bounds.width / 2, bounds.y + bounds.height / 2);
        await expect(page.locator('dialog[open]')).toHaveCount(0);
        release();
        await expect(control).toBeEnabled();
        await expect(page.getByRole('status')).toHaveCount(0);
        if (action.name === 'Settings') {
          await control.focus();
          await page.keyboard.press('Enter');
        } else if (isMobile) await control.tap();
        else await control.click();
        if (action.tutorial) {
          await page.waitForURL(/\/play\/[A-Z0-9]+$/);
          await new GameDriver(page).waitReady();
          expect(await page.evaluate(() => window.__ajayeb?.room()?.mode)).toBe('ai');
        } else {
          await expect(page.locator('dialog[open] h2')).toHaveText(action.heading!);
          await expect(page.locator('dialog[open]')).toHaveCount(1);
        }
        await page.screenshot({ path: info.outputPath('after-start.png') });
      } finally {
        release();
      }
    });
  }

  test('an immediate online click survives delayed startup', async ({ page }) => {
    let release!: () => void;
    const held = new Promise<void>((resolve) => {
      release = resolve;
    });
    await page.route('**/_app/immutable/**/*.js', async (route) => {
      await Promise.all([held, new Promise((resolve) => setTimeout(resolve, 1500))]);
      await route.continue();
    });
    try {
      await page.goto('/', { waitUntil: 'commit' });
      await expect(page.getByRole('heading', { name: 'Ajayeb', exact: true })).toBeVisible();
      // The production helper's immediate click: no startup or hydration wait.
      const opening = page.getByRole('button', { name: /Play online/ }).click();
      // Give that click a chance to land while all app scripts are still held.
      // Native disabled controls keep it pending; the old Home loses it here.
      await Promise.race([opening, page.waitForTimeout(1000)]);
      release();
      await opening;
      await expect(page.locator('dialog[open] h2')).toHaveText('Play online');
    } finally {
      release();
    }
  });

  test('early Home theme controls become available for pointer and keyboard', async ({
    page,
    isMobile
  }) => {
    let release!: () => void;
    const held = new Promise<void>((resolve) => {
      release = resolve;
    });
    await page.route('**/_app/immutable/**/*.js', async (route) => {
      await Promise.all([held, new Promise((resolve) => setTimeout(resolve, 1500))]);
      await route.continue();
    });
    try {
      await page.goto('/', { waitUntil: 'commit' });
      await expect(page.locator('h1')).toBeVisible();
      const dark = page.getByRole('radio', { name: 'Dark', exact: true });
      await expect(dark).toBeDisabled();
      await expect(page.getByRole('radio', { name: 'Light', exact: true })).toBeDisabled();
      await expect(page.getByRole('radio', { name: 'System', exact: true })).toBeDisabled();
      const label = page.locator('label:has(input[name="theme"][value="dark"])');
      const initialTheme = await page.locator('html').getAttribute('data-theme');
      const bounds = await label.boundingBox();
      if (!bounds) throw new Error('missing theme label');
      // Locator actions correctly wait for a label's disabled input. Use a
      // physical press to exercise the unavailable native control itself.
      if (isMobile)
        await page.touchscreen.tap(bounds.x + bounds.width / 2, bounds.y + bounds.height / 2);
      else await page.mouse.click(bounds.x + bounds.width / 2, bounds.y + bounds.height / 2);
      await expect(dark).not.toBeChecked();
      await expect(page.locator('html')).toHaveAttribute('data-theme', initialTheme!);
      release();
      await expect(dark).toBeEnabled();
      if (isMobile) await label.tap();
      else await label.click();
      await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark');
      await dark.focus();
      await page.keyboard.press('ArrowLeft');
      await expect(page.locator('html')).toHaveAttribute('data-theme', 'light');
    } finally {
      release();
    }
  });

  test('loading feedback leaves controls and links uncovered at review sizes', async ({
    page
  }, info) => {
    let release!: () => void;
    const held = new Promise<void>((resolve) => {
      release = resolve;
    });
    await page.route('**/_app/immutable/**/*.js', async (route) => {
      await Promise.all([held, new Promise((resolve) => setTimeout(resolve, 1500))]);
      await route.continue();
    });
    try {
      await page.goto('/', { waitUntil: 'commit' });
      for (const viewport of [
        { width: 360, height: 740 },
        { width: 390, height: 844 },
        { width: 740, height: 360 },
        { width: 1280, height: 800 }
      ]) {
        await page.setViewportSize(viewport);
        await page.locator('h1').scrollIntoViewIfNeeded();
        await expect(page.getByRole('status')).toBeInViewport();
        const overlap = await page.getByRole('status').evaluate((status) => {
          const a = status.getBoundingClientRect();
          return [...document.querySelectorAll('main button, main a, main input')]
            .filter((control) => {
              const b = control.getBoundingClientRect();
              return a.top < b.bottom && a.bottom > b.top && a.left < b.right && a.right > b.left;
            })
            .map((control) => control.textContent?.trim() || control.getAttribute('aria-label'));
        });
        expect(overlap).toEqual([]);
        await page.screenshot({
          path: info.outputPath(`loading-${viewport.width}x${viewport.height}.png`)
        });
      }
      release();
      await expect(page.locator('[data-home-loading]')).toHaveCount(0);
    } finally {
      release();
    }
  });

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
    await expect(page.getByRole('button', { name: /ABCDEF/ })).toBeEnabled();
    await expect
      .poll(async () =>
        ((await page.locator('[data-continue-details]').textContent()) ?? '').replace(
          /[\u200e\u200f\u2066-\u2069]/gu,
          ''
        )
      )
      .toContain('دوران دوم');
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
      await expect(page.getByRole('button', { name: /Play vs AI/ })).toBeDisabled();
      await expect(page.getByRole('status')).toHaveText('Getting ready…');
      await expect(page.locator('link[rel="preload"][as="image"]')).toHaveCount(1);
    } finally {
      await context.close();
    }
    const homeResponse = await request.get(baseURL);
    const home = await homeResponse.text();
    const criticalStyle = homeResponse.headers()['link'];
    expect(criticalStyle).toMatch(
      /^<\/_app\/immutable\/assets\/[^>]+\.css>; rel=preload; as=style$/
    );
    const criticalURL = criticalStyle!.slice(1, criticalStyle!.indexOf('>'));
    expect(home).toContain(criticalURL.slice(1));
    const head = home.slice(0, home.indexOf('</head>'));
    const firstModule = head.indexOf('rel="modulepreload"');
    expect(firstModule).toBeGreaterThan(head.indexOf('imagesrcset='));
    expect(firstModule).toBeGreaterThan(head.lastIndexOf('rel="stylesheet"'));
    expect(firstModule).toBeGreaterThan(head.indexOf('as="font"'));
    expect(head.lastIndexOf('rel="stylesheet"')).toBeLessThan(head.indexOf('as="font"'));
    expect(head.lastIndexOf('rel="stylesheet"')).toBeLessThan(head.indexOf('imagesrcset='));
    const modules = head.match(/<link\b[^>]*rel="modulepreload"[^>]*>/g) ?? [];
    expect(modules.length).toBeGreaterThan(0);
    for (const module of modules) expect(module).toContain('fetchpriority="low"');
    for (const route of ['/play/ABCDEF', '/settings', '/rules', '/shell.html']) {
      const response = await request.get(new URL(route, baseURL).href);
      expect(response.ok()).toBe(true);
      expect(await response.text()).not.toContain('data-hero');
      expect(response.headers()['link']).toBeUndefined();
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
