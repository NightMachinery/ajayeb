/** Observe live image events and an image that completed before attachment.
 * Cleanup also cancels the queued completion check when a frame changes src. */
export function listenForImage(
  image: HTMLImageElement,
  onLoaded: () => void,
  onFailed: () => void
): () => void {
  let active = true;
  const loaded = () => {
    if (active) onLoaded();
  };
  const failed = () => {
    if (active) onFailed();
  };
  image.addEventListener('load', loaded);
  image.addEventListener('error', failed);
  if (image.complete)
    queueMicrotask(() => {
      if (image.naturalWidth > 0) loaded();
      else failed();
    });
  return () => {
    active = false;
    image.removeEventListener('load', loaded);
    image.removeEventListener('error', failed);
  };
}
