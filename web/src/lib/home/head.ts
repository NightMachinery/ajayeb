/** Let Home's render-blocking CSS, fonts and hero precede speculative modules.
 * Every module still starts eagerly; its lower network priority leaves paint
 * resources bandwidth. Stylesheet cascade order and body bytes stay intact. */
export function prioritizeHomePaint(html: string): string {
  const end = html.indexOf('</head>');
  if (end < 0) return html;
  const original = html.slice(0, end);
  if (!/<link\b[^>]*\brel=(['"])modulepreload\1/.test(original)) return html;
  const modules: string[] = [];
  const styles: string[] = [];
  let styleAnchor = -1;
  let removed = 0;
  const head = original.replace(/<link\b[^>]*>/g, (tag, offset: number) => {
    if (/\brel=(['"])modulepreload\1/.test(tag)) {
      modules.push(
        /\bfetchpriority=(['"])[^'"]*\1/.test(tag)
          ? tag.replace(/\bfetchpriority=(['"])[^'"]*\1/, 'fetchpriority="low"')
          : tag.replace(/\s*\/?>$/, (close) => ` fetchpriority="low"${close}`)
      );
      removed += tag.length;
      return '';
    }
    // Keep stylesheet order: it is also the CSS cascade order. Discover this
    // group at the first paint-resource link, ahead of font/image preloads.
    const style = /\brel=(['"])stylesheet\1/.test(tag);
    const preload = /\brel=(['"])preload\1/.test(tag) && /\bas=(['"])(?:font|image)\1/.test(tag);
    if (style || preload) {
      if (styleAnchor < 0) styleAnchor = offset - removed;
      if (style) {
        styles.push(tag);
        removed += tag.length;
        return '';
      }
    }
    return tag;
  });
  const anchor = styleAnchor < 0 ? head.length : styleAnchor;
  return (
    head.slice(0, anchor) +
    styles.join('\n') +
    head.slice(anchor) +
    modules.join('\n') +
    '\n' +
    html.slice(end)
  );
}
