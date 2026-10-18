// Show the bottom fade only while a footer sheet has more details below.
// DOM-only attachment: scrolling, disclosure changes and resizing update one
// attribute, without adding nodes or writing component state.
export function attachSheetScroll(body: HTMLElement): () => void {
  function update(): void {
    body.toggleAttribute(
      'data-scroll-more',
      body.scrollHeight - body.clientHeight - body.scrollTop > 1
    );
  }
  const resize = new ResizeObserver(update);
  resize.observe(body);
  function contentsChanged(): void {
    for (const child of body.children) resize.observe(child);
    update();
  }
  const changes = new MutationObserver(contentsChanged);
  changes.observe(body, { childList: true, subtree: true, characterData: true });
  body.addEventListener('scroll', update, { passive: true });
  contentsChanged();
  return () => {
    resize.disconnect();
    changes.disconnect();
    body.removeEventListener('scroll', update);
  };
}
