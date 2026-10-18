/** Keep the Home paint resources ahead of speculative module requests.
 * Only generated modulepreload tags move; body bytes and hydration markers
 * stay intact, and every script is still fetched by the browser. */
export function prioritizeHomePaint(html: string): string {
  const end = html.indexOf('</head>');
  if (end < 0) return html;
  const modules: string[] = [];
  const head = html.slice(0, end).replace(/<link\b[^>]*>/g, (tag) => {
    if (!/\brel=(['"])modulepreload\1/.test(tag)) return tag;
    modules.push(tag);
    return '';
  });
  return modules.length ? head + modules.join('\n') + '\n' + html.slice(end) : html;
}
