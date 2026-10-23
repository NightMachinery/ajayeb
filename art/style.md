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
paper. Most kinds also fade into that paper with a soft vignette at the
edges; a progress token instead sits on flat paper inside a crisp rim, and
a texture runs evenly to every edge so it can tile (see the kind templates).

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
"Don't" below. A card shows one only when its subject names one (see the
card template): a small background kiosk was where most pseudo-writing
appeared in the Haftgane cards (M19.bitmaps-production-sites), as a band of
script-like marks on its parapet or eave, and leaving it out removed the
fault.

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
- **No inscriptions, no glyph friezes, no writing on scrolls, pages, banners,
  cloth or buildings; scrolls and pages are blank.** No text, letters,
  numerals, calligraphy, pseudo-writing or a signature anywhere in the
  image, including as a pedestal frieze, a ruled page or a carved wall.
  Woven and dyed cloth (a bolt, a robe, an awning) carries plain stripes,
  checks or small floral sprigs, never a band of script-like marks or a
  cartouche inside a stripe. A background pavilion's parapet, door panels
  and steps carry plain geometric tile bands. Those two places are where
  pseudo-writing slipped into the Haftgane pilot (M19.bitmaps-pilot), and
  it showed only at full resolution, not at thumbnail size. The numeral
  cartouche on a card back is drawn separately, in SVG, never baked into
  the bitmap.
- **A frame, border or watermark.** The processing step feathers the image
  into the card; a hard rectangle or a drawn border fights that.
- **Religious figures or scenes of worship as the subject.** A temple or an
  altar is shown as architecture and a quiet gathering, never a depicted
  deity or a specific rite.
- **Background buildings are palaces, domed garden pavilions and kiosks.**
  No minarets, no mosques, no religious buildings, unless the asset's own
  `subject` names one. A skyline filling the background of a scene is
  secular civic architecture, not a place of worship.
- **Political or national emblems.** No lion-and-sun, no flag, no
  fleur-de-lis, no crest tied to a real nation, ancient or modern. Patterned
  robes use rosettes and leafy sprigs, not a repeated emblem.
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

The edge treatment lives in the kind templates, not in the prefix, because
the kinds disagree about it (M9.B4). Cards, wonders, backs, the hero, the
avatars, the victory scenes and the app icon fade to the paper with a soft
vignette, so a frame can feather into them. A token is a medallion with a
crisp gold rim on flat paper, which `tools/art/process.mjs` finds and masks
to; a vignette there would blur the rim and darken the margin. A texture
must tile, so it has no vignette at all.

- **card**: "`{subject}`. Landscape vignette, subject centered, dominant
  accent `{accent}`. Keep every figure and object inside the central area;
  leave a wide margin of plain ivory paper on all four sides, widest at
  left and right. One or two large figures doing the named activity,
  filling the central third; no crowds, no distant city skyline, no filler
  animals unless the subject names them. No building in the background
  unless the subject names one. Soft vignette fading to the paper colour at
  the edges."
- **wonder**: "Panoramic view of `{subject}` as monumental architecture in
  the Persian-miniature idiom, landscape, dominant accent `{accent}`. The
  monument centered and whole, occupying the middle third, not cut by the
  frame. Soft vignette fading to the paper colour at the edges."
- **token**: "Circular medallion centered on a square of plain ivory paper,
  filling the square except for a narrow, even margin of paper (about 4
  percent of the width) on every side; the paper outside the rim is flat
  and clean, with no vignette, fade, shadow or ornament. Inside: a flat
  `{accent}` ground, one bold symmetric emblem of `{subject}` filling the
  central 60 percent, a thin crisp gold rim only; no landscape, trees,
  rocks, ground or clouds inside the medallion; no ornamental border
  band."
- **back**: "Symmetric ornamental illumination (tazhib / girih strapwork),
  portrait orientation, no figures: `{subject}`. Soft vignette fading to
  the paper colour at the edges."
- **texture**: "Seamless fine paper grain, `{subject}`, no figures, no
  objects, suitable for tiling."
- **hero**: "Wide panoramic scene: `{subject}`, landscape, warm inviting
  light, dominant accent `{accent}`. Soft vignette fading to the paper
  colour at the edges."
- **avatar**: "Portrait bust, shoulders up, one figure, centered:
  `{subject}`. Dominant accent `{accent}`, plain ivory background, no
  scenery. Soft vignette fading to the paper colour at the edges."
- **victory**: "Wide celebratory scene: `{subject}`, landscape, dominant
  accent `{accent}`. Soft vignette fading to the paper colour at the
  edges."
- **icon**: "Simple bold emblem on a plain ivory ground, readable at 32
  pixels: `{subject}`. Dominant accent `{accent}`, no scenery, no
  background detail. Soft vignette fading to the paper colour at the
  edges."

## Style prefix (verbatim)

> Original illustration in the manner of a 16th-century Persian miniature
> painting (Safavid manuscript illumination): flat decorative perspective,
> fine dark ink outlines, opaque gouache in jewel tones (lapis lazuli
> blue, turquoise, saffron, vermilion, malachite green, ochre) with small
> touches of gold leaf; stylized cypress trees, rounded rocky hills with
> pastel edges, patterned tilework, ornamental clouds, tiny figures in
> period dress. The sky and background are plain ivory paper, never gold
> leaf; gold appears only on small details such as finials, rims and a sun
> disc. Background buildings are palaces, domed garden pavilions and
> kiosks; no minarets, no mosques, no religious buildings unless the
> subject names one. Robes are plain or patterned with small rosettes or
> leafy sprigs, never fleur-de-lis or another repeated emblem. One clear
> central subject readable as a small thumbnail, calm uncluttered
> background of warm ivory paper. No inscriptions, no glyph friezes, no
> writing on scrolls, pages, banners, cloth or buildings; scrolls and
> pages are blank, cloth shows only plain stripes or small floral sprigs,
> and parapets show only plain geometric tile bands. No border, no frame,
> no watermark, no signature. Not photorealistic, not 3D, no modern
> objects.
