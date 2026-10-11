// Per-asset framing of bitmap art: where a cover crop is anchored and which
// feather it gets. Most paintings are composed around their middle and use
// the defaults; the ones listed here put their subject near the top edge
// (a statue's raised arm, a tomb's stepped roof and finial), which a
// centred crop or the wide top fade of the 'soft' feather would cut off.
// Pure data: positions are CSS object-position values.

export interface ArtCrop {
  /** CSS object-position of the cover crop. */
  readonly position: string;
  /** The ArtImage feather for the large frames (thumbnails keep 'edge'). */
  readonly feather: 'soft' | 'sky';
}

const DEFAULT_CROP: ArtCrop = { position: '50% 50%', feather: 'soft' };

const WONDER_CROPS: { readonly [slug: string]: ArtCrop } = {
  // The statue's head and raised hand reach the top tenth of the painting.
  colossus: { position: '50% 8%', feather: 'sky' },
  // The stepped roof and its finial rise from the top twentieth.
  mausoleum: { position: '50% 12%', feather: 'sky' }
};

/** The framing of a wonder's art in WonderTile (the draft and the city). */
export function wonderArtCrop(slug: string): ArtCrop {
  return (Object.hasOwn(WONDER_CROPS, slug) ? WONDER_CROPS[slug] : undefined) ?? DEFAULT_CROP;
}
