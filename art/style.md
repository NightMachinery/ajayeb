# Bitmap art direction: "Illuminated garden"

This is the single source the bitmap pipeline draws from. `tools/art/generate.mjs`
prepends the **style prefix** below to every prompt, then the **kind template**
for the asset's kind, then the asset's own `subject` from `art/subjects.json`,
then any `promptAmend` a review round added. Nothing else goes into a prompt.

See [docs/legal.md](../docs/legal.md) for why this has to stay clearly
original, and [docs/art/bitmaps.md](../docs/art/bitmaps.md) for how the
pipeline runs end to end.

## Look

Original, Persian-miniature-influenced illustration (the idiom of a
16th-century Safavid manuscript): flat decorative perspective with no
vanishing-point realism, fine dark ink outlines, opaque gouache in jewel
tones, and small touches of gold leaf. One clear central subject that stays
readable as a small thumbnail. A calm, uncluttered background of warm ivory
paper, with a soft vignette fading to the paper colour at the edges.

## Palette

Jewel tones only, named so every prompt and every reviewer uses the same
words:

- **lapis** — deep blue, the night sky and water
- **turquoise** — the Sun player's accent, cool green-blue
- **saffron** — warm yellow-orange, commerce and sunlight
- **vermilion** — warm red, military and fire
- **malachite** — green, growth and science
- **ochre** — warm earth brown, raw materials and ground
- **violet** — the guild colour, used sparingly
- **gold** — leaf accents only: borders of a robe, a dome's finial, a coin.
  Never a flat fill; gold reads as a highlight, not a background.
- **ivory** — the paper ground everything sits on

These are the same hues as the interface's OKLCH tokens in
`web/src/lib/styles/tokens.css` (`--lapis`, `--turquoise`, `--saffron`,
`--cinnabar` ≈ vermilion, `--malachite`), so a rendered card's frame and its
painted scene read as one object, not two clashing palettes.

## Motifs

Stylized cypress trees; rounded rocky hills with pastel edges; patterned
tilework (girih strapwork, not representational mosaics); ornamental clouds
(the Chinese-cloud motif Safavid painting borrowed); tiny figures in period
dress, shown from the flattened three-quarter angle typical of the idiom;
domes, arches and columns for architecture; calm water rendered as a
patterned ripple, not a realistic reflection.

## Do

- One clear subject, centered, readable at a glance.
- Flat decorative perspective: figures and buildings stack by composition,
  not by a single vanishing point.
- Fine, continuous ink outlines around every shape.
- Opaque colour fields (gouache, not watercolour wash or airbrush gradients).
- A warm ivory paper background that the subject sits on, not a hard-edged
  rectangle behind it.

## Don't

- **Painterly realism, photorealism or 3D rendering.** This is what keeps
  the art at a clear distance from the official game's photographed and
  painted art and from any other illustrated edition.
- **Text, letters, numerals, calligraphy or a signature anywhere in the
  image.** The numeral cartouche on a card back is drawn separately, in SVG,
  never baked into the bitmap.
- **A frame, border or watermark.** The processing step feathers the image
  into the card; a hard rectangle or a drawn border fights that.
- **Religious figures or scenes of worship as the subject.** A temple or an
  altar is shown as architecture and a quiet gathering, never a depicted
  deity or a specific rite.
- **Political or national emblems.** No lion-and-sun, no flag, no crest tied
  to a real nation, ancient or modern.
- **Anachronisms.** Nothing that reads as a specific modern object, uniform
  or technology.
- **Any likeness of the official game's art, card frames or iconography.**
  These prompts are written from scratch per [docs/legal.md](../docs/legal.md)
  and never reference the published game's illustrations.

## Culturally respectful

The Persian-miniature idiom is a painting style, not a costume for a
specific culture's sacred or political symbols. Depict generic period
figures, trades and buildings; avoid identifiable religious iconography,
named historical rulers, and any symbol tied to a living nation or faith.
When a subject from `art/subjects.json` would need one of those to read
correctly, it is written to avoid the need instead (see the `theology`
token, which reads as a lit lamp and olive leaves rather than a national
crescent-and-star).

## Kind templates

Every prompt is `{style prefix} {kind template} No text, no letters, no
calligraphy, no numerals, no border, no frame, no watermark, no signature.
Not photorealistic, not 3D, no modern objects.` The kind template fills in
the asset's `subject` and `accent` from `art/subjects.json`.

- **card**: "`{subject}`. Landscape vignette, subject centered, dominant
  accent `{accent}`."
- **wonder**: "Panoramic view of `{subject}` as monumental architecture in
  the Persian-miniature idiom, landscape, dominant accent `{accent}`."
- **token**: "Circular medallion with one bold symmetric emblem of
  `{subject}` on a `{accent}` ground, high contrast, readable at 40 pixels;
  flat ivory outside the circle."
- **back**: "Symmetric ornamental illumination (tazhib / girih strapwork),
  portrait orientation, no figures: `{subject}`."
- **texture**: "Seamless fine paper grain, `{subject}`, no figures, no
  objects, suitable for tiling."
- **hero**: "Wide panoramic scene: `{subject}`, landscape, warm inviting
  light, dominant accent `{accent}`."
- **avatar**: "Portrait bust, shoulders up, one figure, centered:
  `{subject}`. Dominant accent `{accent}`, plain ivory background, no
  scenery."
- **victory**: "Wide celebratory scene: `{subject}`, landscape, dominant
  accent `{accent}`."
- **icon**: "Simple bold emblem on a plain ivory ground, readable at 32
  pixels: `{subject}`. Dominant accent `{accent}`, no scenery, no
  background detail."

## Style prefix (verbatim)

> Original illustration in the manner of a 16th-century Persian miniature
> painting (Safavid manuscript illumination): flat decorative perspective,
> fine dark ink outlines, opaque gouache in jewel tones — lapis lazuli blue,
> turquoise, saffron, vermilion, malachite green, ochre — with small touches
> of gold leaf; stylized cypress trees, rounded rocky hills with pastel
> edges, patterned tilework, ornamental clouds, tiny figures in period
> dress. One clear central subject readable as a small thumbnail, calm
> uncluttered background of warm ivory paper, soft vignette fading to the
> paper colour at the edges. No text, no letters, no calligraphy, no
> numerals, no border, no frame, no watermark, no signature. Not
> photorealistic, not 3D, no modern objects.
