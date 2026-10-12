# hints: iteration log

The hint pip glyphs (M10.F2). There is no separate spec file: the brief is below. The set is `hints` in [../sets.json](../sets.json).

## Brief

- **hint-crown**: "this card wins the game". It replaces the inline crown path that HintPip.svelte and HintLines.svelte drew since M10.HS2. One sentence: a three-pointed crown with a ball on each point, standing on a separate band.
- **hint-pair**: "this card completes a science pair, so its builder picks a progress token". Before, the pip drew the token frame with the card's own science symbol inside at 56%, which left the symbol about 9 px wide inside the 16 px phone pip: legible but not crisp (the HS2 review). One sentence: the progress-token rosette, solid, with an equals sign cut through it, so it reads as "matches" rather than as one particular symbol; the tooltip line and the card sheet name the symbol in words, and the card's own effect band already shows it.
- Contexts: inside the pip, a 16 px disc on the phone (glyph box 12 px) and a 20 px disc on larger screens (glyph box 16 px), filled gold with ink, the Moon crest with paper, or paper inside a crest-coloured ring; and in the pip's tooltip lines and the card sheet's hint lines at about 12 to 15 px on the panel. Phones run at DPR 2 or 3, so the 12 px box is 24 to 36 device pixels.
- Must differ from: `capital-sun` and `capital-moon` (castles, not crowns), `coin` (a plain disc with a square hole), `crest-sun` (a disc with two eyes, which is why the pair is not two dots), `token-frame` (a ring, where hint-pair is a solid rosette), `badge-chain` (linked rings).

## Status

- hint-crown: done (iteration 2)
- hint-pair: done (iteration 2)

## Iteration 1 (2026-10-13)

Renders: art-work/vector/hints/iter-1/ (render-fast.sh at 12, 16, 24 and 48 px on paper and on night, upscaled 400% with point filtering; and pip mocks: a 16 px disc in gold, Moon and outlined paper, at DPR 1, 2 and 3, with coin-loss beside them for weight).
Lint: clean.

### hint-crown

- Seen: reads as a crown at every size and in every pip. A small diamond cut into the body as a gem is a clear jewel at 48 px but only a one-pixel speck at DPR 1 and 2 inside the pip: noise, not a detail.
- Changed: removed the gem. The crown is now one silhouette (three points with balls, sharp inner valleys) over a band separated by a 1.5 unit gap, all in `--i1`.

### hint-pair

- Seen: the rosette is right, but the equals sign (two bars 2.5 units tall with a 2 unit gap) closes up at 12 px DPR 1 and turns the glyph into a blob with a grey centre; at DPR 2 it just reads.
- Changed: bars 3 units tall, 11 wide, with a 3 unit gap (y 7.5 to 10.5 and 13.5 to 16.5), so each bar and the gap between them are 1.5 px at 12 px DPR 1.

## Iteration 2 (2026-10-13)

Renders: art-work/vector/hints/iter-2/ (the same sheet and pip mocks).
Lint: clean; the sprite is 26210 bytes raw and 5375 gzipped with both icons.

### hint-crown

- Seen: clean at 12, 16, 24 and 48 px on paper and night; inside the gold, Moon and outlined pips at DPR 1 to 3 it is the most legible glyph of the three. Same weight as coin-loss.
- Changed: nothing. Done.

### hint-pair

- Seen: the equals sign now stays open at 12 px DPR 1 (just) and is crisp at DPR 2 and 3 in all three pip fills; at 16 px and up it reads as a scalloped medallion crossed by two bars. The rosette rim is thin beside the bar ends, which at 12 px makes the glyph read as a framed equals sign more than a token; that is the meaning we want.
- Changed: nothing. Done.

## Browser check (2026-10-13)

Renders: `tools/chrome/svg-render.mjs` at 16, 24 and 48 px on paper and on night, DPR 1 and 2, beside coin, coin-loss, crest-sun and token-frame (shots/M10/F2/icons-sheet@1.png and @2.png in the run directory); and the real pips in a test build, on the phone (390x844 at DPR 2) and the desktop, light and dark (shots/M10/F2/after-pip2-*.png).

- Chrome draws both as librsvg did. At 16 px DPR 1 hint-pair is a scalloped disc crossed by two bars, clearly apart from coin (a square hole), token-frame (a ring) and crest-sun (rays and eyes); hint-crown is unmistakable.
- In the app the crown pip (the Moon's solid fill, paper glyph) is crisp on the phone and the desktop. The pair pip in the captured position is a faint, tinted pip on a half-covered card; its equals sign still reads, the rosette edge only just.
- No blind check was run: the reader would be the author, and the two glyphs are always seen with a tooltip line that says what they mean.
