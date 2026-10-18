import { describe, expect, it } from 'vitest';
import { prioritizeHomePaint } from './head';

describe('prerendered Home resource discovery', () => {
  it('moves all module preloads after styles, fonts and hero with low module priority, preserving resources and body', () => {
    const modules = [
      '<link href="/start.hash.js" rel="modulepreload">',
      "<link rel='modulepreload' href='/chunk.hash.js'>"
    ];
    const paint = [
      '<link rel="preload" as="font" href="/font.hash.woff2">',
      '<link rel="preload" as="image" href="/hero.avif?v=123">',
      '<link href="/app.hash.css" rel="stylesheet">'
    ];
    const body =
      '<body><!--[--><main>Home</main><!--]--><script>start()</script><link rel="modulepreload" href="/body.js"></body>';
    const original =
      '<head>' +
      modules[0] +
      '<!-- head marker -->' +
      paint.join('') +
      modules[1] +
      '</head>' +
      body;
    const result = prioritizeHomePaint(original);
    expect(result.slice(result.indexOf('</head>') + 7)).toBe(body);
    expect(result).toContain('<!-- head marker -->');
    for (const resource of paint) expect(result.split(resource)).toHaveLength(2);
    const lowered = modules.map((tag) => tag.replace('>', ' fetchpriority="low">'));
    for (const resource of lowered) expect(result.split(resource)).toHaveLength(2);
    expect(result.match(/href=['"][^'"]+['"]/g)?.sort()).toEqual(
      original.match(/href=['"][^'"]+['"]/g)?.sort()
    );
    expect(result.indexOf(paint[2]!)).toBeLessThan(result.indexOf(paint[0]!));
    for (const resource of paint)
      expect(result.indexOf(resource)).toBeLessThan(result.indexOf(lowered[0]!));
    expect(result.indexOf(lowered[0]!)).toBeLessThan(result.indexOf(lowered[1]!));
  });

  it('keeps cascade order, overrides existing priority and is idempotent', () => {
    const html =
      '<head><link rel="modulepreload" href="/app.js" fetchpriority="high">' +
      '<link rel="preload" as="image" href="/hero.avif">' +
      '<link rel="stylesheet" href="/base.css">' +
      '<link rel="stylesheet" href="/page.css"></head><body>unchanged</body>';
    const result = prioritizeHomePaint(html);
    expect(result.indexOf('/base.css')).toBeLessThan(result.indexOf('/page.css'));
    expect(result.indexOf('/page.css')).toBeLessThan(result.indexOf('/hero.avif'));
    expect(result).toContain('href="/app.js" fetchpriority="low"');
    // Whitespace between moved groups may grow; resources and markup do not.
    expect(prioritizeHomePaint(result).replace(/>\s+</g, '><')).toBe(
      result.replace(/>\s+</g, '><')
    );
  });

  it('leaves a generic shell or an incomplete head unchanged', () => {
    for (const html of [
      '<head><link rel="stylesheet" href="/app.css"></head><body>shell</body>',
      '<head><link rel="modulepreload" href="/app.js">'
    ]) {
      expect(prioritizeHomePaint(html)).toBe(html);
    }
  });
});
