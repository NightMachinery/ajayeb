import type { Page } from '@playwright/test';

// Shared by the captured-view matrix and sheets over a live table.
export async function probeCardSheet(page: Page) {
  const sheet = page.locator('[data-card-sheet]');
  return sheet.evaluate((root) => {
    const dialog = root.closest('dialog')!;
    const r = dialog.getBoundingClientRect();
    const small = [...root.querySelectorAll<HTMLElement>('button:not(:disabled), [role="radio"]')]
      .filter((e) => !e.closest('[inert]') && e.checkVisibility())
      .map((e) => ({
        key: e.dataset['act'] ?? e.dataset['pick'],
        w: e.getBoundingClientRect().width,
        h: e.getBoundingClientRect().height
      }))
      .filter((e) => e.w < 43.9 || e.h < 43.9);
    const untipped = [...root.querySelectorAll('svg, [role="img"]')]
      .filter(
        (e) =>
          !e.parentElement?.closest('svg') &&
          !e.closest('[data-tooltip]') &&
          !e.closest('[inert]') &&
          e.checkVisibility()
      )
      .map((e) => e.outerHTML.slice(0, 100));
    const blocked = root.querySelector('[data-cannot-build]');
    let lowContrast = false;
    if (blocked) {
      const ctx = document.createElement('canvas').getContext('2d')!;
      const luminance = (colour: string) => {
        ctx.fillStyle = colour;
        ctx.fillRect(0, 0, 1, 1);
        const rgb = [...ctx.getImageData(0, 0, 1, 1).data].slice(0, 3).map((v) => {
          const c = v / 255;
          return c <= 0.04045 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4;
        });
        return rgb[0]! * 0.2126 + rgb[1]! * 0.7152 + rgb[2]! * 0.0722;
      };
      const style = getComputedStyle(blocked);
      const ink = luminance(style.color),
        paper = luminance(style.backgroundColor);
      lowContrast = (Math.max(ink, paper) + 0.05) / (Math.min(ink, paper) + 0.05) < 4.5;
    }
    const footer = root.querySelector('.sheet-footer')!.getBoundingClientRect();
    return {
      lowContrast,
      crampedTitle: root.querySelector('h2')!.getBoundingClientRect().width < 100,
      overflow:
        document.documentElement.scrollWidth > innerWidth + 1 ||
        root.scrollWidth > root.clientWidth + 1,
      small,
      untipped,
      outside: r.x < -1 || r.y < -1 || r.right > innerWidth + 1 || r.bottom > innerHeight + 1,
      footerOutside: footer.bottom > innerHeight + 1 || footer.top < 0,
      animated: document
        .getAnimations()
        .filter(
          (a) => a.playState === 'running' && Number(a.effect?.getComputedTiming().duration) > 1
        ).length
    };
  });
}
