// The tutorial's coach steps at every viewport of the UX plan
// (M10.tutorial-client): a tutorial room from the start, walked to the
// last lesson (playOn) with Animation off and the shared policy
// (fixtures/tutorial.ts). At every coach step: the target on screen and
// unobscured at its centre, the bubble on screen, off an element target
// and over at most 40% of a region target, its arrow towards the target,
// its buttons at least 44 px tall, and no horizontal scroll. No console
// errors, page errors or failed requests on the way.
//
// The desktop project walks all ten viewports with a mouse; the mobile
// project walks the touch widths with touch, like layout.spec.ts.

import { expect } from '@playwright/test';
import { test } from '../fixtures/base';
import { watchHygiene } from '../fixtures/hygiene';
import { freshDevice, openTutorialRoom, walkTutorial } from '../fixtures/tutorial';

const VIEWPORTS = [
  { width: 360, height: 740 },
  { width: 390, height: 844 },
  { width: 412, height: 915 },
  { width: 430, height: 932 },
  { width: 844, height: 390 },
  { width: 740, height: 360 },
  { width: 768, height: 1024 },
  { width: 1024, height: 768 },
  { width: 1280, height: 800 },
  { width: 1920, height: 1080 }
] as const;

const TOUCH_WIDTHS = new Set([360, 390, 412, 430, 844, 740, 768, 1024]);

for (const viewport of VIEWPORTS) {
  const name = `${viewport.width}x${viewport.height}`;
  test.describe(`the tutorial at ${name}`, { tag: '@mobile' }, () => {
    test.use({ viewport });

    test('every coach step fits and points at its target', async ({ page, baseURL, isMobile }) => {
      test.skip(isMobile && !TOUCH_WIDTHS.has(viewport.width), 'desktop-only viewport');
      test.setTimeout(90_000);
      if (!baseURL) throw new Error('no baseURL');
      const problems = watchHygiene(page);
      await freshDevice(page, { tips: false, summary: false, speed: 'off' });
      await openTutorialRoom(page, baseURL);
      const walk = await walkTutorial(page, { touch: isMobile, until: 'playOn' });
      console.log(
        `${name}: ${walk.steps.length} coach steps, placements ${[
          ...new Set(walk.steps.map((s) => s.placement))
        ].join(' ')}`
      );
      expect(walk.observed.at(-1)).toBe('playOn');
      expect(problems).toEqual([]);
    });
  });
}
