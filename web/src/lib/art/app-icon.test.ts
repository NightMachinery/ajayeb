// The app icon PNGs and the web app manifest (static/manifest.webmanifest,
// static/icons/, linked from app.html), made from the approved icon/app
// original by tools/art/process.mjs --app-icons.
import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';

const web = new URL('../../../', import.meta.url);
const read = (path: string) => readFileSync(new URL(path, web));

/** A PNG's pixel size, from its IHDR chunk. */
function pngSize(path: string): { width: number; height: number } {
  const b = read(path);
  expect(b.subarray(1, 4).toString('latin1'), path).toBe('PNG');
  return { width: b.readUInt32BE(16), height: b.readUInt32BE(20) };
}

interface WebManifest {
  id: string;
  name: string;
  short_name: string;
  start_url: string;
  scope: string;
  display: string;
  launch_handler: { client_mode: string };
  theme_color: string;
  background_color: string;
  icons: { src: string; sizes: string; type: string; purpose: string }[];
}

const manifest = JSON.parse(read('static/manifest.webmanifest').toString('utf8')) as WebManifest;
const appHtml = read('src/app.html').toString('utf8');

describe('the web app manifest', () => {
  it('names the app and opens it standalone at the root', () => {
    expect(manifest.id).toBe('/');
    expect(manifest.name).toBe('Ajayeb');
    expect(manifest.short_name).toBe('Ajayeb');
    expect(manifest.start_url).toBe('/');
    expect(manifest.scope).toBe('/');
    expect(manifest.display).toBe('standalone');
  });

  it('opens a room link it catches in the open app window, not a second one', () => {
    expect(manifest.launch_handler).toEqual({ client_mode: 'navigate-existing' });
  });

  it('uses the light table colour of the theme-color meta', () => {
    const light =
      /name="theme-color" content="(#[0-9a-f]{6})" media="\(prefers-color-scheme: light\)"/.exec(
        appHtml
      )?.[1];
    expect(light).toBeDefined();
    expect(manifest.theme_color).toBe(light);
    expect(manifest.background_color).toBe(light);
  });

  it('lists the 192 and 512 px PNG icons and a maskable 512, and they are those sizes', () => {
    const icons = manifest.icons.map((i) => `${i.purpose} ${i.sizes}`).sort();
    expect(icons).toEqual(['any 192x192', 'any 512x512', 'maskable 512x512']);
    for (const icon of manifest.icons) {
      expect(icon.type).toBe('image/png');
      const [w, h] = icon.sizes.split('x').map(Number);
      expect(pngSize(`static${icon.src}`), icon.src).toEqual({ width: w, height: h });
    }
  });
});

describe('the app icon links', () => {
  it('links the manifest and a 180 px apple-touch-icon', () => {
    expect(appHtml).toContain(
      '<link rel="manifest" href="%sveltekit.assets%/manifest.webmanifest" />'
    );
    const touch = /<link rel="apple-touch-icon" href="%sveltekit.assets%(\/[^"]+)"/.exec(
      appHtml
    )?.[1];
    expect(touch).toBe('/icons/apple-touch-icon.png');
    expect(pngSize(`static${touch}`)).toEqual({ width: 180, height: 180 });
  });
});
