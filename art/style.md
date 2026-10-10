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

- **lapis**: deep blue, the night sky and water
- **turquoise**: the Sun player's accent, cool green-blue
- **saffron**: warm yellow-orange, commerce and sunlight
- **vermilion**: warm red, military and fire
- **malachite**: green, growth and science
- **ochre**: warm earth brown, raw materials and ground
- **violet**: the guild colour, used sparingly
- **amber**: warm honey-brown resin tone, for wood, tools and stonework
- **rose**: soft warm pink-red, lighter and calmer than vermilion
- **slate**: cool blue-grey, for stone, technical and civic subjects
- **gold**: leaf accents only: borders of a robe, a dome's finial, a coin,
  a sun disc. Never a flat fill and never the sky or a background: gold
  reads as a highlight, not a ground. A wonder or token whose accent used
  to be "gold" now takes one of the hues above instead; see
  `art/subjects.json`.
- **ivory**: the paper ground everything sits on, never gold leaf

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
patterned ripple, not a realistic reflection. Background buildings read as
palaces, domed garden pavilions and kiosks, not places of worship: see
"Don't" below.

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
- **No inscriptions, no glyph friezes, no writing on scrolls, pages, banners
  or buildings; scrolls and pages are blank.** No text, letters, numerals,
  calligraphy, pseudo-writing or a signature anywhere in the image,
  including as a pedestal frieze, a ruled page or a carved wall. The
  numeral cartouche on a card back is drawn separately, in SVG, never
  baked into the bitmap.
- **A frame, border or watermark.** The processing step feathers the image
  into the card; a hard rectangle or a drawn border fights that.
- **Religious figures or scenes of worship as the subject.** A temple or an
  altar is shown as architecture and a quiet gathering, never a depicted
  deity or a specific rite.
- **Background buildings are palaces, domed garden pavilions and kiosks.**
  No minarets, no mosques, no religious buildings, unless the asset's own
  `subject` names one. A skyline filling the background of a scene is
  secular civic architecture, not a place of worship.
- **Political or national emblems.** No lion-and-sun, no flag, no crest tied
  to a real nation, ancient or modern.
- **No heraldic animals on buildings or banners.** No lions, no winged
  lions, no eagles as carved, painted or bannered emblems, and no lion
  paired with a sun disc or sun orb.
- **No filler animals unless the asset's own `subject` names one.** Deer,
  peacocks and a row of cypress trees are motifs, not a default background;
  reusing them on every card is what makes the set look repetitive.
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

Every prompt is `{style prefix} {kind template}`: the verbatim style
prefix below (which already carries its own negatives, such as "no
inscriptions" and "not photorealistic") followed by one kind template.
The kind template fills in the asset's `subject` and `accent` from
`art/subjects.json`.

- **card**: "`{subject}`. Landscape vignette, subject centered, dominant
  accent `{accent}`. Keep every figure and object inside the central area;
  leave a wide margin of plain ivory paper on all four sides, widest at
  left and right. One or two large figures doing the named activity,
  filling the central third; no crowds, no distant city skyline, no filler
  animals unless the subject names them."
- **wonder**: "Panoramic view of `{subject}` as monumental architecture in
  the Persian-miniature idiom, landscape, dominant accent `{accent}`. The
  monument centered and whole, occupying the middle third, not cut by the
  frame."
- **token**: "Circular medallion filling the square almost edge to edge,
  flat `{accent}` ground, one bold symmetric emblem of `{subject}` filling
  the central 60 percent, a thin gold rim only; no landscape, trees, rocks,
  ground or clouds inside the medallion; no ornamental border band."
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
> dress. The sky and background are plain ivory paper, never gold leaf;
> gold appears only on small details such as finials, rims and a sun disc.
> Background buildings are palaces, domed garden pavilions and kiosks;
> no minarets, no mosques, no religious buildings unless the subject names
> one. One clear central subject readable as a small thumbnail, calm
> uncluttered background of warm ivory paper, soft vignette fading to the
> paper colour at the edges. No inscriptions, no glyph friezes, no writing
> on scrolls, pages, banners or buildings; scrolls and pages are blank. No
> border, no frame, no watermark, no signature. Not photorealistic, not
> 3D, no modern objects.
