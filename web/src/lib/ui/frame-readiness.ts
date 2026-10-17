// The ready getter can synchronously resolve style/layout. Reuse its promise
// only when every registered face is already loaded (or failed), so no new
// glyph demand can start another load. A changed locale or face set rechecks.
export function createFrameReadiness() {
  let cached:
    | {
        fonts: FontFaceSet;
        locale: string;
        faces: FontFace[];
        promise: Promise<FontFaceSet>;
      }
    | undefined;

  function ready(fonts: FontFaceSet, locale: string): Promise<FontFaceSet> {
    const faces = [...fonts.values()];
    const reusable =
      cached?.fonts === fonts &&
      cached.locale === locale &&
      fonts.status === 'loaded' &&
      faces.length === cached.faces.length &&
      faces.every(
        (face, index) =>
          face === cached?.faces[index] && (face.status === 'loaded' || face.status === 'error')
      );
    if (reusable && cached) return cached.promise;
    cached = { fonts, locale, faces, promise: fonts.ready };
    return cached.promise;
  }

  return (root: HTMLElement, fonts: FontFaceSet, locale: string): (() => void) => {
    let live = true;
    let generation = 0;
    let frame: number | undefined;
    function wait(): void {
      const current = ++generation;
      if (frame !== undefined) cancelAnimationFrame(frame);
      delete root.dataset['ready'];
      void ready(fonts, locale).then(() => {
        if (!live || generation !== current) return;
        frame = requestAnimationFrame(() => {
          frame = undefined;
          if (!live || generation !== current) return;
          if (fonts.status === 'loading') wait();
          else root.dataset['ready'] = '1';
        });
      });
    }
    fonts.addEventListener('loading', wait);
    wait();
    return () => {
      live = false;
      generation++;
      fonts.removeEventListener('loading', wait);
      if (frame !== undefined) cancelAnimationFrame(frame);
      delete root.dataset['ready'];
    };
  };
}
