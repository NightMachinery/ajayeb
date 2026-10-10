# art/refs: the style anchors

These three images are the owner's picks from the M9.B1/M9.B2 style pilot.
Each one is the reference a reviewer holds every new attempt of its kind
against before approving it (see the review criteria in
[docs/art/bitmaps.md](../../docs/art/bitmaps.md)). They are **review
references only**: `tools/art/generate.mjs` sends no image to Codex unless
it is run with `--ref`, and the batch never passes `--ref`, because the
M9.B3 pilot showed that a reference of the same subject makes Codex return
a near-copy of it instead of a new illustration.

An anchor fixes the **style**, not the subject. Judge an attempt against
it for line, palette, paper margin and density; its subject, composition
details and any motif the current `art/style.md` now rules out are not part
of what it anchors (each anchor below names those).

## The anchors

- **`card-anchor.jpg`**, from the pilot's `card/scriptorium` attempt 2: a
  scribe in a green robe and white turban kneels at a low lacquered
  writing desk on a blue carpet, writing on a blank page, among flowering
  trees, cypresses, lilac rocks and a winding river, with a domed pavilion
  far behind.
  - Why it is the reference: it is the clearest example of the "illuminated
    garden" look the whole set should share. Fine continuous ink outlines,
    opaque gouache fields with no airbrush gradient, jewel tones used at
    full strength (malachite robe, lapis carpet, vermilion cushion) and
    gold kept to small accents (the clouds' edges, the sun disc, the ewer).
    The ground is a pale, even ivory paper, and the painting thins out into
    it on every side, so a card frame can feather into the margin.
  - Density: one large figure doing the named activity in the middle
    third, readable at the card thumbnail size; the rest is supporting
    garden detail that stays lighter in weight than the figure.
  - Not anchored: the deer and the peacock at the sides. They are exactly
    the filler animals `art/style.md` now forbids unless a subject names
    them, so an attempt is not asked to have them.
- **`wonder-anchor.jpg`**, from the pilot's `wonder/pyramids` attempt 2: a
  wide desert landscape with three monuments, the largest centred and
  whole in the middle third, a river and palms on the right, a camel
  caravan in the foreground, a group of seated onlookers on lilac rocks at
  the left, and a gold sun disc and blue ornamental clouds in a plain
  ivory sky.
  - Why it is the reference: it is the panoramic composition wonders need.
    The monument is centred and uncut, the sky is ivory paper and not gold
    leaf, the palette runs from lapis and turquoise water through saffron
    and ochre sand to the lilac and rose rocks, and the margin of paper on
    every side passes `tools/art/gate.mjs`.
  - Density: busier than a card, as a wonder's larger tile allows, but the
    monument still reads first at the 480 px thumbnail.
  - Not anchored: the architecture itself. Its pyramids are stepped, with
    stairs and a pavilion on top, which is exactly what the current
    `wonder/pyramids` subject rules out (smooth-sided pyramids of Giza,
    never stepped, never a ziggurat). It anchors how a wonder is painted,
    not what the Pyramids look like.
- **`token-anchor.jpg`**, from the pilot's `token/agriculture` attempt 1: a
  sheaf of golden wheat tied with a vermilion cord and tassels, on a flat
  malachite ground scattered with small gold, lapis and vermilion flowers
  and scrolling vines, inside a thin gold rim, centred on the square with
  a narrow, even margin of plain ivory paper all round.
  - Why it is the reference: one bold, symmetric emblem filling the central
    part of the medallion, a single flat accent ground, a thin gold rim and
    no border band. The paper outside the rim is flat and clean, with no
    vignette or shadow, which is what lets `tools/art/process.mjs` find the
    painted rim and mask the token to it.
  - Density: the emblem reads at the 40 px token size; the vines and
    flowers stay small and low in contrast against the ground.
  - Not anchored: the two blue clouds inside the medallion. The token
    template forbids clouds there.

## The files

- Each anchor is a JPEG made from the attempt's raw PNG with ImageMagick:
  scaled to 1024 px on the long side, metadata stripped, 4:2:0 chroma,
  progressive. The card and wonder are at quality 82; the token is at
  quality 80, because at 82 its fine green texture came to 254 KB, over
  the 250 KB budget for an anchor.
  - `card-anchor.jpg`: 1024x683, about 192 KB.
  - `wonder-anchor.jpg`: 1024x683, about 198 KB.
  - `token-anchor.jpg`: 1024x1024, about 239 KB.
- The raw source attempts live outside the repository, under
  `${AJAYEB_RUN_DIR:-~/tmp/agents/ajayeb}/art-work/raw/<kind>/<slug>/<attempt>.png`.
  None of the three is an approved original: an anchor is a style
  reference, and its asset still goes through the batch like any other.

## License

The anchors are original art made for Ajayeb and are licensed under
CC BY 4.0, like the rest of the art in this directory. See
[art/LICENSE](../LICENSE) and [docs/legal.md](../../docs/legal.md).
