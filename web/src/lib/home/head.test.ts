import { describe, expect, it } from 'vitest';
import { prioritizeHomePaint } from './head';

describe('prerendered Home resource discovery', () => {
  it('moves all module preloads after styles, fonts and hero without changing resources or body', () => {
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
    for (const resource of [...modules, ...paint]) expect(result.split(resource)).toHaveLength(2);
    for (const resource of paint)
      expect(result.indexOf(resource)).toBeLessThan(result.indexOf(modules[0]!));
    expect(result.indexOf(modules[0]!)).toBeLessThan(result.indexOf(modules[1]!));
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
