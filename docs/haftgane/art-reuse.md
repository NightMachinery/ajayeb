# Haftgane art: reuse and production plan

@OpenAI, 2026-10-21. Design proposal for the second-edition base game. No artwork was generated or changed. Dimensions for the proposed board layout are design targets, not browser measurements.

## Recommendation and scope

Reuse 58 of the 80 card designs, covering 56 names, from 55 distinct approved Duel paintings. Generate 21 new card paintings for the other 22 designs, covering 21 names. Loom shares its new painting across its two designs. Reuse all seven wonder paintings on both board sides, and reuse the existing back paintings without revealing hidden card types. No new crop or bitmap size is required by this proposal.

Plan for 27 additional SVG drawings: 22 for the requested resource, science, chain, conflict and pass sets, plus five ability glyphs needed to present the complete base game. This means 22 new visual concepts and five additional optical-size drawings, not 27 unrelated symbols. Counts and reuse aliases are itemized below.

The visual direction stays “illuminated garden”: original Persian-miniature-influenced gouache, clear central subjects, jewel accents, ivory margins and fine ink outlines. New subjects below are original proposals, not descriptions copied from published illustrations. Existing paintings convey an activity or monument; server-provided text and vectors convey rules. A change in strength, score, chain or scoring scope does not by itself require a new painting.

The audit used `art/style.md`, `art/subjects.json`, `art/manifest.json`, `docs/art/bitmaps.md`, `docs/dev/decisions.md` (especially D07, D18, D20 and D24), `docs/legal.md`, `art/LICENSE`, `art/vector/STYLE.md`, all six vector specs, the shipped asset inventory and icon sources. The sibling `rules.md`, `duel-overlap.md` and `platform-architecture.md`, with the effect inventory ([effects.md](effects.md), held back until M13 and now published), supply the game and platform contracts. Private draft sheets were read for completeness; their costs, values, copy thresholds and provenance rows are not reproduced here.

## Card mapping: reuse unchanged

Each line names Haftgane's card, then its existing art id and why the depicted subject still fits. “Unchanged” means the existing 4:3 AVIF and thumbnail, including their current content-hashed URLs. The presenter supplies the card's Haftgane name, category, effects and chains separately. An art alias never aliases engine data.

- **Academy:** `card/academy`. Philosophers debating in an academy remain a suitable science scene.
- **Altar:** `card/altar`. Quiet garden architecture and offerings remain a civic subject, without baking in a chain destination.
- **Apothecary:** `card/apothecary`. A healer measuring tinctures still represents learned medicine, whatever the science symbol.
- **Aqueduct:** `card/aqueduct`. The water-carrying structure remains a civic achievement.
- **Archery Range:** `card/archery-range`. Practising archers remain a military subject.
- **Arena:** `card/arena`. Tiered seating and spectators fit a commercial venue; its stage-based income is rendered outside the painting.
- **Arsenal:** `card/arsenal`. Armourers making weapons and shields fit the military role.
- **Barracks:** `card/barracks`. Drilling soldiers remain suitable even though the card appears in a different Age.
- **Baths:** `card/baths`. The tiled pool and bathers still fit civic infrastructure.
- **Brickyard:** `card/brickyard`. Brickmaking fits clay production without requiring the image to count units.
- **Builders Guild:** `card/builders-guild`. Builders reviewing a monument suit stage-based scoring as well as whole-monument scoring.
- **Caravansery:** `card/caravansery`. Resting traders and camels represent access to a raw-resource assortment; adding ore does not invalidate the scene.
- **Chamber of Commerce:** `card/chamber-of-commerce`. Merchants and clerks suit commerce linked to manufactured goods.
- **Circus:** `card/circus`. Chariot racing remains the named military spectacle; shield totals belong in the effect band.
- **Clay Pool:** `card/clay-pool`. Clay extraction is still the activity.
- **Courthouse:** `card/courthouse`. Elders in a civic court suit this card. Do not use Duel's red `card/pretorium`, despite its alternative English name.
- **Dispensary:** `card/dispensary`. Dispensing remedies still fits science. The approved art, rather than the older subject wording about labelled jars, is the source.
- **East Trading Post:** `card/customs-house`. Weighing goods beside cargo fits a trading post. Share the painting with West Trading Post; a persistent directional marker and translated name distinguish the two.
- **Forum:** `card/forum`. Merchants bartering under awnings fit access to manufactured goods, including cloth.
- **Fortifications:** `card/fortifications`. Building battlements remains military construction.
- **Gardens:** `card/gardens`. Palace terraces and fountains remain civic embellishment.
- **Glassworks, Age I:** `card/glassworks`. Glassblowing remains the production activity.
- **Glassworks, Age II:** `card/glassworks`. The same named production activity uses the same image; the Age is separate metadata.
- **Guard Tower:** `card/guard-tower`. A watching sentinel remains suitable.
- **Haven:** `card/port`. Dockworkers loading a ship directly fit the harbour-commerce role. This is a subject match despite the different name.
- **Laboratory:** `card/laboratory`. An alchemist with glass vessels remains a science subject.
- **Library:** `card/library`. Scholars and scroll shelves remain suitable for learning.
- **Lighthouse:** `card/lighthouse`. The harbour beacon fits maritime commerce. Do not substitute the wonder painting.
- **Lumber Yard:** `card/lumber-yard`. Felling and stacking timber still represents wood production.
- **Magistrates Guild:** `card/magistrates-guild`. Civic magistrates remain suitable; which cities score is an effect-band matter.
- **Observatory:** `card/observatory`. Astronomers and instruments fit science without requiring the card's symbol in the painting.
- **Palace:** `card/palace`. The royal hall remains a civic subject.
- **Pantheon:** `card/pantheon`. Monumental columned architecture remains suitable, with no need to depict a rite.
- **Press, Age I:** `card/press`. Papyrus sheet production remains the activity.
- **Press, Age II:** `card/press`. The same production scene serves the second design.
- **Quarry:** `card/quarry`. Stonecutters extracting slabs fit stone production; quantity is never inferred from the painted blocks.
- **Sawmill:** `card/sawmill`. Sawyers cutting a beam still represent timber production.
- **School:** `card/school`. Teacher and pupils remain suitable learning imagery.
- **Scientists Guild:** `card/scientists-guild`. Comparing scholarly instruments fits a choice of science discipline. The painting contains no authoritative Duel science glyph or reward count, so the changed effect does not require repainting.
- **Senate:** `card/senate`. Senators in a civic chamber remain suitable.
- **Shipowners Guild:** `card/shipowners-guild`. Owners and shipwrights inspecting a vessel still fit the guild; the additional scoring category needs no painted change.
- **Siege Workshop:** `card/siege-workshop`. Assembling a siege engine remains military work.
- **Scriptorium:** `card/scriptorium`. Scribe and blank scroll fit learning even though the new science symbol differs.
- **Stables:** `card/stable`. Grooms and saddled horses fit both names; no new image is needed for the plural.
- **Statue:** `card/statue`. Unveiling a stone statue remains a civic subject.
- **Stockade:** `card/palisade`. Raising sharpened timber stakes is precisely the named defensive structure.
- **Stone Pit:** `card/stone-pit`. Quarrymen extracting limestone remain suitable.
- **Study:** `card/study`. Scholar, instruments and blank pages suit learning independently of the scoring symbol.
- **Tavern:** `card/tavern`. Travellers sharing food and drink fit the same commercial venue.
- **Temple:** `card/temple`. Approved columned architecture and quiet gathering remain suitable.
- **Theater:** `card/theater`. Masked performers remain the named civic entertainment.
- **Town Hall:** `card/town-hall`. Elders in a civic hall remain suitable.
- **Training Ground:** `card/parade-ground`. Soldiers marching under instruction fit military training.
- **Traders Guild:** `card/merchants-guild`. Guild merchants with goods and coins fit the trade profession; the alternative name is not a rules dependency.
- **University:** `card/university`. Master and student with a celestial globe remain a learning scene. A painted globe does not promise a particular science glyph.
- **Walls:** `card/walls`. Masons raising a stone wall still fit military infrastructure.
- **West Trading Post:** `card/customs-house`. Same trade scene as East Trading Post; distinguish direction through the frame and label, without mirroring the people or painting.
- **Workshop:** `card/workshop`. An artisan making gears and tools remains an appropriate technical-science subject.

These are 58 designs, not 58 unique source paintings. Glassworks and Press each reuse one source twice; East and West Trading Post also share one source. All other mappings above use distinct paintings. Of the 52 shared names identified in `duel-overlap.md`, 51 reuse their existing painting. Five additional names reuse other Duel subjects: the two trading posts, Stockade, Training Ground and Haven.

## Card mapping: new paintings

Each line gives an asset slug proposal, a reason for new art and a one-line subject with its accent. The eventual ids must live in a game-specific namespace so `clay-pit` does not overwrite Duel's original. All take the existing card kind template, with one or two large figures, clean ivory margins and no writing. Resource pairs should appear as one coherent working site, not two separate miniatures. Which two resources a mixed producer yields is card data, so the subjects below leave the pair open until the data milestone has verified it.

- **Ore Vein** (`ore-vein`): Duel has no ore producer. Subject: a miner lifting dark metallic ore from a narrow exposed seam into a shallow basket; ochre accent.
- **Tree Farm** (`tree-farm`): a mixed producer needs a mixed working site. Subject: two workers tending a small stand of straight saplings at one site that also shows the card's second resource; malachite accent.
- **Excavation** (`excavation`): a mixed dig differs from a pure quarry. Subject: two diggers separating the card's two materials at one terraced excavation; ochre accent.
- **Clay Pit** (`clay-pit`): the approved Duel image is entirely a clay dig, and this card is a mixed producer. Subject: two workers hauling clay baskets beside a clearly exposed seam of the card's second resource in the same pit; ochre accent.
- **Timber Yard** (`timber-yard`): Duel's Logging Camp shows timber alone. Subject: a timber worker and a second worker organizing cedar logs and the card's other material in one small riverside yard; amber accent.
- **Forest Cave** (`forest-cave`): a timber camp alone does not convey this combined site. Subject: a woodcutter passing bundled logs to a second worker at a wooded cave mouth, with the card's other material in a basket; malachite accent.
- **Mine** (`mine`): Duel's stone scenes are open quarries, not a mine. Subject: two miners sorting the card's two materials at a low mine entrance; slate accent.
- **Loom, Age I and Age II** (`loom`, one painting): Duel has no cloth producer. Subject: a weaver drawing a shuttle through a simple upright loom with a broad unfinished cloth and a folded bolt beside it; rose accent.
- **Well** (`well`): no approved water-drawing scene matches. Subject: two residents drawing a water jar from a low stone well beneath a plain timber crossbeam; turquoise accent.
- **Marketplace** (`marketplace`): the Forum already has a distinct card role, and a manufacture-focused stall is clearer. Subject: two traders exchanging a glass flask, a blank papyrus roll and folded cloth at one small market stall; saffron accent.
- **Foundry** (`foundry`): weapon forging is too specifically military. Subject: two metalworkers pouring a glowing melt into plain ingot moulds beside a squat furnace; amber accent.
- **Vineyard** (`vineyard`): a tavern is consumption, not cultivation. Subject: two growers harvesting broad bunches of grapes into a basket beneath a low vine trellis; malachite accent.
- **Bazar** (`bazar`): give this trading venue its own silhouette instead of repeating the Forum. Subject: a merchant and an artisan displaying folded cloth, glass vessels and blank paper beneath one striped awning; saffron accent.
- **Ludus** (`ludus`): a training business should differ from both the public Arena and the military Training Ground. Subject: an instructor receiving a small coin pouch beside a trainee practising with a wooden sword in a simple courtyard; saffron accent.
- **Castrum** (`castrum`): Duel's command tent alone lacks the fortified camp. Subject: two sentries at the timber gate of a compact fortified camp with plain tents visible inside; vermilion accent.
- **Lodge** (`lodge`): keep it distinct from the Academy, University and Study. Subject: two scholars examining a plain drafting instrument on a low table in a quiet timber learning lodge; malachite accent.
- **Workers Guild** (`workers-guild`): merchants in reserve paintings do not depict labourers organizing their craft. Subject: a quarry worker and a timber worker conferring beside a stone block and a bundled log at a guild bench; violet accent.
- **Craftsmens Guild** (`craftsmens-guild`): a military armory would misidentify the profession. Subject: a glass artisan and a weaver examining a small finished vessel and a folded textile at a guild table; violet accent.
- **Spies Guild** (`spies-guild`): officers planning a battle are not enough to signal reconnaissance. Subject: two discreet scouts comparing a blank folded route sheet beside a low wall, with no weapons raised; violet accent.
- **Philosophers Guild** (`philosophers-guild`): distinct from instrument-based Scientists Guild. Subject: two philosophers debating with open hand gestures over a closed blank book in a small garden court; violet accent.
- **Decorators Guild** (`decorators-guild`): builders' scaffolding suggests unfinished work. Subject: two artisans inspecting a completed floral tile panel and its small decorative finial at a monument entrance; violet accent.

This is 21 subjects for 22 designs. There are 20 single-design subjects plus Loom's two designs. No existing source can gain a missing resource, profession or structure through a crop alone. Accordingly, zero cards are assigned “reuse with new crop or size.” Reusing the clay-only Clay Pit would save one generation and remain playable because the effect band is authoritative; the recommendation spends that one painting to keep every mixed producer's scene consistent.

## Wonder boards: fourteen sides, seven paintings

Reuse the seven committed 1536x1024 originals and their existing 3:2 AVIF derivatives: 480x320, 960x640 and 1440x960. Both sides of a board use the same painting; day and night describe the selected rules layout, not a promise of two different sky paintings. Show the side name and a sun or moon marker outside the image. Dark mode changes the surrounding UI, not the painting or the game side.

Direct inspection of all seven originals shows why a full-width panoramic crop is unsuitable. A centered 2:1 crop removes 12.5% of the original height at both top and bottom; 3:1 removes 25% at each edge. Several monuments nearly fill the original height. Stretching to either ratio would distort them.

- **Alexandria:** `wonder/great-lighthouse`. Reuse intact. The lantern finial is close to the top edge and the stepped island reaches low in the frame; both would suffer a wide crop. A generic supply-related wonder ability does not require a new lighthouse scene.
- **Babylon:** `wonder/hanging-gardens`. Reuse intact. Preserve the dome finial and lower pool. The science and last-card abilities are stage overlays, so the different Duel effect is irrelevant to the monument.
- **Ephesos:** `wonder/temple-of-artemis`. Reuse intact. The temple and wooded hillside form one tall composition; a crop would lose the setting. Coin-related stages fit the existing monument without depicting money in the architecture.
- **Gizah:** `wonder/pyramids`. Reuse intact. This is the widest monument group, but a crop would still sacrifice river and foreground. Its point-based role matches naturally.
- **Halikarnassos:** `wonder/mausoleum`. Reuse intact. Preserve the roof finial, stairs and foreground pool. Its recovery ability remains an external stage glyph.
- **Olympia:** `wonder/statue-of-zeus`. Reuse intact. The columned hall, seated statue and tiny attendant supply the scale; losing the base or roof weakens it. Free construction is shown in the stages, without carrying over Duel's destruction effect.
- **Rhodos:** `wonder/colossus`. Reuse intact. The raised hand needs the upper margin; the harbour and feet need the lower image. The statue still fits the military-themed board.

### Proposed board composition

Use a responsive landscape panel, approximately 2:1 when space permits. For a concrete layout target, a 720x360 panel has an upper 240-pixel area with the intact painting at 360x240 in its centre, and paper flanks for the translated board name, side, starting-resource tile and status. The lower 120 pixels contain the stage strip. This is one framed UI composition, not one flattened bitmap. Smaller panels may rearrange the name and side above the artwork; allow height to grow for translation or larger text.

At a 360-pixel panel width the same proportions leave a 180x120 painting and a 60-pixel stage strip. Four stages have about 90 pixels each before gutters, enough to prototype a cost line, effect line and a target of at least 44 pixels. This is arithmetic only; phone, RTL, large-text and actual effect-glyph acceptance remain implementation work. The detail sheet can show the painting wider using the existing 960 and 1440 variants.

- Draw stages in their own opaque themed strip below the painting, never over its architecture. Each stage has its number, server-provided cost and effects, and an explicit built/unbuilt state.
- Use two, three or four equal-width stage cells according to the chosen side. All day sides have three. Rhodos night has two, and so does Babylon night, provisionally: its layout waits for a photo of the printed board (Q1 in [data-rulings.md](data-rulings.md#the-night-side-questions)). Gizah night has four; the other night sides have three. These make 41 cells across all 14 layouts (the total rests on Q1), not 41 illustrations.
- Keep stage order spatially left to right in a deliberately LTR stage region, with translated surrounding UI. Do not reverse stage indexes in RTL.
- A neutral, identical tucked-card tab marks each built stage. It carries no card-specific painting, Age, guild status, DOM id or accessible label. Put the visible stage number above it. The mark stands for “a stage was built,” not for a newly revealed card back.
- Reuse existing ornaments sparingly and draw the rectangular stage cells in CSS. No board-sized raster frame and no new stage-frame SVG are needed.

The tradeoff is less edge-to-edge scenery and a smaller painting in a compact board, in exchange for preserving every monument and avoiding seven or fourteen new generations. If a future approved layout requires a wide full-bleed panorama, budget seven new compositions shared across sides, not fourteen. That is an alternative production scope, excluded from the recommended totals.

## Vector sets

Reuse actual geometry, not Duel's mechanics. Existing source files and current inline rendering take precedence over old spec references to a shared sprite. Do not recreate or extend a shared sprite merely because the original specs use that term. New SVG is authored by Opus workers in the established render, inspect and critique loop; Codex does not draw it. All new concepts must be visibly original, not traces of the physical game's symbols.

### Resources: reuse five, add two

Reuse wood, clay, stone, glass and papyrus, each with its existing md and sm drawing: ten files unchanged. Add ore and cloth at both optical sizes: four new drawings, two new concepts.

- **Ore:** a low trapezoidal metal ingot with one broad bevel. It must differ from irregular stone, rectilinear clay bricks and the anvil category glyph. Do not rely on grey colouring to distinguish it.
- **Cloth:** a folded bolt with one broad hanging fold and a notched fringe. It must differ from the papyrus roll, bundled wood and the tablet science symbol.

Drawn in M19.vectors-resources-and-science: spec and log in `art/vector/specs/haftgane-resources-science.md` and `art/vector/log/haftgane-resources-science.md`, sources in `web/src/lib/games/haftgane/icons/svg/`.

Choice production composes the existing choice marker and resource tiles; seven resources do not require seven new choice symbols. Directional trade discounts compose the price tag, family symbols and a neighbour-direction marker.

### Science: two new concepts, one adapted concept

Use compass, gear and tablet consistently; none inherits Duel's pair/token or instant-victory presentation.

- **Compass:** new md and sm drawings of an open drafting compass, two legs joined at a broad hinge. No magnetic compass rose. Separate it from the lyre, crossed spears and set square.
- **Gear:** reuse the existing `chain-gear` geometry at md under a science semantic alias. Draw a new 12-grid sm version, rather than scaling the md teeth. Do not substitute `science-wheel`, whose smooth rim and spokes convey a different object.
- **Tablet:** new md and sm drawings of a single broad writing slab with rounded shoulders and one heavy bevel, blank face. Separate it from the closed book's spine and the scroll's curled ends.

Drawn in M19.vectors-resources-and-science, in the same set as ore and cloth; the `-sm` gear has eight short teeth like the md, on a larger ring.

Science therefore adds five files: four for compass/tablet and one gear-sm. A science-choice effect combines the choice marker and the three symbols, rather than generating another painting or inventing a fourth science symbol.

### Military: reuse shields, add conflict tokens

Reuse `shield` unchanged for strength. Do not reuse the pawn, military track, looting tokens or the coin-loss glyph as conflict results. Haftgane resolves neighbour comparisons at Age end and has no moving pawn.

Add two vector outlines: **conflict-victory**, an intact broad notched pennant, and **conflict-defeat**, a split pennant with a large open break. Render the signed point value as translated, localized live text in the token component. One victory shape supports all three Age values; one defeat shape supports the negative result. Shape, sign and text distinguish outcomes without colour. Keep them distinct from the shield and the chain helmet. No four-value bitmap medallion set is needed.

Two new files, two concepts. Start with robust 24-grid drawings at 12 pixels and above; add sm variants only if the actual token render fails acceptance, as separately recorded production work.

### Coins and scoring

Reuse `coin`, `vp-laurel`, the existing coin-component treatment and localized numbers. Denominations and treasury totals are data labels, not additional SVG files. A negative conflict score must not look like a coin penalty. Reuse `wonder` for board context, but do not let it ambiguously mean an entire wonder where an effect counts individual stages.

### Chains: twenty-four symbols

Keep game-specific semantic ids separate from file names and wire enums. Reuse the following eleven existing chain drawings, with the plural semantic name mapped where necessary:

- barrel: `chain-barrel`.
- book: `chain-book`.
- columns: `chain-column` (the single-column metaphor remains clear).
- drop: `chain-drop`.
- helmet: `chain-helmet`.
- horseshoe: `chain-horseshoe`.
- lamp: `chain-lamp`.
- lyre: `chain-lyre`.
- mask: `chain-mask`.
- target: `chain-target`.
- tower: `chain-tower`.

Reuse three more existing instrument drawings as chain aliases: mortar from `science-mortar`, scales from `science-scales`, and quill from `science-quill`. Those are not science meanings in this game; the chain chip supplies their role. Keep the alias in the presenter, not in the engine's science enum. Their 12-pixel chip appearance still needs acceptance inspection.

Draw ten new chain concepts, one md file each initially:

- **awning:** a low striped canopy with two short supports, different from the purse category glyph.
- **camel:** a single broad camel silhouette, with legs simplified enough for the chip.
- **hammer:** a heavy transverse head on a short diagonal handle, different from the sword and saw.
- **lighthouse:** a stepped beacon with a broad lantern cap, different from the crenellated tower and column.
- **lightning:** a single thick angular bolt, different from the upright torch.
- **planet:** a large disc crossed by one tilted orbital band, without the stand of Duel's globe.
- **saw:** a broad short blade with three large teeth and a handle opening.
- **scroll:** a tall blank scroll with rolled top and bottom, different from the horizontal resource-papyrus and flat science tablet.
- **star:** a solid five-point star, different from the eight-point category-green and rayed sun.
- **torch:** a straight handle with one large flame, different from the low-spouted lamp and zigzag lightning.

The accounting is eleven existing chain files, three reused instruments and ten new drawings: exactly 24. Do not reuse any Duel card-to-card chain relationship. In/out chips use the same geometry and translated help; Haftgane's public catalog supplies the relationships. Any later sm chain drawings are additional optical variants, not extra chain symbols.

### Categories, passing and backs

Reuse all seven category glyphs and their seven sm variants unchanged. Their meanings still fit raw production, manufactured production, civic, scientific, commercial, military and guild cards. Update help text to mention ore and cloth. Keep D20's separation between a category and its effect or chain glyph; the new five-point chain star deliberately avoids the green category star.

Add one **pass-hand** drawing: two overlapping card silhouettes and one heavy horizontal arrow. Mirror that geometry to show the opposite direction; a mirror is not another file or concept. Associate each direction with the named receiving neighbour in translated text and the server-sent pass state. The stable seat ring, not document reading direction or animation, determines where it points. Animation off must leave an equally clear static arrow.

Reuse `back/age1`, `back/age2`, `back/age3` and `back/guild`. They are bitmap ornaments, not four new vector projects. Keep Age labels and pass arrows as live overlays, outside the baked illumination; no new numeral outlines are needed. The existing backs are portrait 320x495, approximately 1:1.545, independent of the card-art inset's 4:3 ratio. The hidden card frame should preserve that portrait shape and contain the ornament rather than make a destructive landscape crop.

**Hidden-information rule:** every publicly concealed Age III card uses the Age III back, including guilds. A distinct guild back would disclose which cards in a hand or commitment are guilds, contrary to the hidden guild draw. Keep the violet guild ornament for a public catalog/rules preview or other explicitly public category display only, not the deck, hand, commitments, sold cards or tucked stages. Duel's public guild-back policy remains Duel-only. If a pile mixes Ages, use a neutral procedural stack rather than showing per-card Age backs.

### Ability glyphs beyond the requested core sets

Budget five new md glyphs so the art total covers the complete base game rather than silently deferring its unique abilities:

- **wonder-stage:** one stepped construction bay and a lower tuck slot, visibly different from the whole-wonder gateway. Reuse it in stage-count effects and board progress.
- **wonder-complete:** a short row of stage bays capped by a check, for completion-based scoring.
- **play-last-card:** two overlapping cards, one lifted beyond a small end marker. Do not use Duel's circular play-again arrow, which suggests a different timing.
- **free-first-colour:** a single card entering the first empty position in a three-card fan; compose category swatches and translated text for the actual colours.
- **free-age-boundary:** a card beside a heavy end-stop. Mirror for first/last turn, with explicit translated labels. Do not bake turn numbers into the shape.

Reuse `build-from-discard` with new timing/help text. Compose all other counting scopes, science choices, production choices and trade directions from shared glyphs and live text. These five ability drawings are included in the 27-file production total; they are not additional bitmaps.

## Totals and budgets

### Production counts

- Cards: 80 design bindings across 77 names. Reuse 58 designs from 55 existing originals; make 21 new originals for 22 designs. The resulting game uses 76 distinct card paintings because the trading posts share a painting as well as the three same-name pairs.
- Wonders: seven existing originals for fourteen side bindings. Zero new paintings, zero crops and zero new sizes.
- Backs: four existing originals available, three used for public concealed Age cards. Zero new paintings; the guild ornament is restricted to public category contexts.
- Other bitmaps: reuse paper/night textures, the shared site branding and avatars where their UI is used. No game-specific hero or victory scene is required to make the base game playable; show the shared hero and a neutral score-led result. Do not present Duel's military/science victory paintings as Haftgane instant-win outcomes.
- New bitmap total: **21 originals and 42 shipped AVIF files**, two variants per new card. Shared originals and URLs are referenced once, without copying files into another game directory.
- New core vectors: resource 4, science 5, chains 10, conflict 2, passing 1: **22 files**.
- New ability vectors: **5 files**. Complete planned total: **27 SVG drawings**, comprising 22 new concepts and five extra size drawings (ore-sm, cloth-sm, compass-sm, tablet-sm and gear-sm). Geometry aliases, mirrors, number labels, CSS frames and composed effects do not add drawings.
- If the UI requires small versions of new chains, conflict or ability icons after inspection, count them explicitly before production expands. They are not assumed successful at every size by this source-only design.

### Per-image byte limits under D24

Use bytes as the authority; the existing documentation's “KB” budgets are multiples of 1024 in `tools/art/lib.mjs`.

- New and reused card art: generation 1536x1024, processed to 4:3. Main 480x360 at at most 35,840 bytes; thumb 240x180 at at most 10,240 bytes.
- Reused wonders: original 1536x1024, 3:2. 480x320 at at most 20,480 bytes; 960x640 at at most 71,680 bytes; 1440x960 at at most 161,792 bytes. The seven relevant wonders already have all three variants.
- Reused backs: 320x495 at at most 30,720 bytes each. No painted direction or Age numeral.
- Reused textures: 512x512 at at most 61,440 bytes each. Reused avatars: 192x192 at at most 12,288 bytes each.
- Shared hero, if displayed: 1600x900 at at most 122,880 bytes and 800x450 at at most 61,440 bytes. No additional hero generation is in scope.
- New SVG glyphs: at most 1,228 bytes each, following the two-tone 24-grid or 12-grid rules. Existing ornaments retain their 4-KiB limit. Treat the legacy 40-KiB raw / 10-KiB gzip icon-family budget as an acceptance gate on each game's needed icon closure, without eagerly downloading the other game's unique icons. A 27-file count does not prove that byte gate passes; measure it in implementation and share geometry instead of duplicating aliases.

The maximum additional card AVIF payload allowed by those individual limits is 967,680 bytes (21 times 46,080). This is a planning upper bound, not measured output and not a replacement total cap. D24 has no total art-size cap. Report shipped bytes and device-specific prefetch bytes after processing.

The existing 480-pixel card main is a sound starting point, not permission to enlarge a sheet image indefinitely. Measure the new hand, card detail and wonder detail at DPR 3. Add a size only when the measured display fails D24's 1.25-times upscale tolerance, bounded by source resolution. Preserve already-shipped Duel variants byte for byte.

## Generation and review batches

This is a future production plan. No commands below have been run by this slice, and no imagegen or vector-authoring skill is being invoked by this document.

First parameterize the catalog/manifest/index tools for the registered game. `platform-architecture.md` does not plan this step; the [roadmap](../dev/roadmap.md) gives it its own slice in the art milestone (`M19.art-pipeline-game-inputs`). Store semantic-to-art aliases separately from card rules and reuse existing content-hashed paths. New assets need a game-qualified namespace; a naive combined manifest regeneration must not drop Duel assets or overwrite the old Clay Pit. Approved-only lookup and procedural fallback stay in force.

Generate new subjects through `tools/art/generate.mjs` using Codex `image_gen`, **one asset at a time under `heavy.py --kind codex`**. A review batch groups decisions, not simultaneous calls. Use the style prefix and card template from `art/style.md`; do not use official illustrations as references or same-subject reference images that invite near-copies.

1. **Pilot, three images:** Ore Vein, Loom, Clay Pit. These test the new material vocabulary, cloth silhouette and mixed-production composition against the existing set. Done in `M19.bitmaps-pilot`: all three are approved and shipped, Ore Vein and Clay Pit on their first attempt and Loom on its third. Clay Pit shows clay and ore, its pair in [data-rulings.md](data-rulings.md) ruling 14. Each card's accent is its category's colour word, the word Duel's catalog gives every Duel card (brown for Ore Vein and Clay Pit, grey for Loom), rather than the hue named in the mapping above, so a Haftgane painting carries the same dominant hue as the Duel cards of its category; the later batches should do the same. Loom's two rejections were pseudo-writing on a pavilion parapet and inside the cloth's stripes, which led to the cloth and parapet rule in `art/style.md`.
2. **Remaining production sites, six images:** Tree Farm, Excavation, Timber Yard, Forest Cave, Mine, Foundry. Review the entire mixed-site family together for distinct thumbnails and balanced paired subjects. Done in `M19.bitmaps-production-sites`: all six are approved and shipped, Tree Farm and Mine on their first attempt, Excavation, Timber Yard and Forest Cave on their second, and Foundry on its third. Each pair comes from the catalog: Tree Farm wood or clay, Excavation stone or clay, Timber Yard wood or stone, Forest Cave wood or ore, Mine stone or ore. Foundry produces two ore, not a choice, so it shows one material in quantity: two ore heaps feeding a furnace and the ingots it casts. The final review found all eight production sites, the pilot's Clay Pit and Ore Vein included, distinct at the 120 px thumbnail; Mine and Forest Cave are the closest pair, told apart by white stone blocks against brown logs. Three first attempts had script-like marks on a background kiosk, and one retry a robe of fleurs-de-lis, which led to the card template's no-building rule and the robe rule in `art/style.md`.
3. **Civic and commercial places, six images:** Well, Marketplace, Vineyard, Bazar, Ludus, Castrum. Check Marketplace/Bazar/Forum separation and commercial Ludus versus military Training Ground. Done in `M19.bitmaps-civic-and-commercial`: all six are approved and shipped, Well, Marketplace, Ludus and Castrum on their first attempt and Vineyard and Bazar on their second. Each accent is the card's category colour (blue for Well, red for Castrum, yellow for the other four), and the paying cards show what they pay for: Vineyard's grapes are bought by a woodcutter with his logs (brown cards), Bazar's shelves hold glass, paper and cloth (grey cards), and Ludus's trainer, wooden sword in hand, is paid in coins beside a practice post and a rack of wooden swords, with no soldiers. Marketplace (two traders on an open mat), Bazar (an arched brick shop) and the reused Forum (an awning stall) read as three places at 120 px, Ludus reads as no red card, and Castrum, a tented camp seen from above inside an earth rampart, copies none of the reused red cards. Vineyard's first attempt drew a trellis post that formed a lone cross, now named in `art/style.md`; Bazar's had script-like strokes on its cloth bolts.
4. **Scholars and guilds, six images:** Lodge, Workers Guild, Craftsmens Guild, Spies Guild, Philosophers Guild, Decorators Guild. Check profession recognition and separation from every reused science/guild scene.

For each batch, follow the M9 pipeline: objective margin and paper gate; contact sheet plus direct full-resolution inspection; written per-attempt approval/rejection with a specific prompt amendment; regenerate rejected subjects sequentially; compare the next round against accepted siblings. Inspect at thumbnail, shipped size and original resolution for pseudo-writing, misleading objects, unwanted emblems, realism, crowds and inadequate margins. Mixed sites must show both materials clearly without asking a player to count painted units.

Start with one attempt per subject, then at most two rejection-driven retries under the normal three-attempt escalation rule. Three complete attempts per 21 subjects would mean 63 calls, a contingency ceiling before escalation, not a target or approval to generate an unnecessary second pass. If a subject cannot be fixed by a precise prompt amendment, stop that subject for a rewritten brief or coordinator review. Never discard a better earlier attempt merely because a later one exists.

Only an approved attempt becomes a lossless WebP original. Require the existing zero-differing-pixels verification against the generated PNG. Derive AVIFs through the processor, verify each byte budget, rebuild the approved index and calculate every URL hash from shipped bytes. Do not hand-edit a derivative, hash or generated TypeScript. Rejected attempts and review scratch files stay outside the repository.

Vector production has three review groups: resources/science, chains, and conflicts/passing/abilities. Each Opus author follows the vector style and set specs, using fast rendering per edit and a guarded browser specimen once per batch in the later implementation milestone. Check light/dark, greyscale, colour-vision cases and actual 12-to-16-pixel chips, with blind resource/science identification and chain matching at the established acceptance rates. Verify the full family together, especially ore/stone/clay, tablet/book/scroll, star/category-green, lamp/torch/lightning and lighthouse/tower/column. Reused instrument geometry in chain chips needs the same small-size check as new geometry.

## Delivery, privacy and licensing

Follow the platform's lazy game loading: loading the Home or Duel page must not fetch every new card, icon or catalog. That holds for page loads: the service worker's all-build precache still fetches Haftgane's code chunks in the background, its catalog and inline icons among them, while art bitmaps are never precached. Prefetch only after entering this game, one file at a time at idle and low priority. Build the plan from the complete public catalog for the current Age and later Ages, never from the viewer's hand, selected guild subset, discard membership, committed key or tucked identities. Use the frame's exact `srcset` and `sizes`, pause when hidden, cancel on exit and retain D24's Save-Data/slow-connection small plan. Loading a known static art file must not be used as evidence that its card was dealt.

Whitelisted game views supply public board/side/stage and concealed-card presentation kinds. The client must never choose an Age/guild back from a hidden engine id. A shared image alias is a display lookup, not a way around the platform's per-viewer redaction or game-specific renderer boundaries.

Original paintings, derivatives, vectors and ornaments remain **CC BY 4.0**, as stated in `art/LICENSE` and `docs/legal.md`. Preserve the existing “Ajayeb original art by NightMachinery” attribution, license link and source notice. Record new aliases, crops or other adaptations so attribution and change history survive. Code remains under its existing license; no font or audio relicensing follows from this art work. No official painting, icon, frame, logo or rulebook wording is an input to production.

## Draft decisions for coordinator review

The drafts as this plan proposed them. They were accepted on 2026-10-21 as D57 to D59 of [decisions.md](../dev/decisions.md), whose entries are the ones in force.

### Shared paintings with game-specific bindings

**Decision:** share approved original art and shipped URLs through explicit game-specific aliases; commission the 21 subjects above. Rules ids and display ids remain independent.

**Reasons:** semantic reuse saves 55 generations and preserves visual continuity, while dedicated mixed-resource and profession scenes avoid misleading reuse. Day/night and Age variants need different metadata, not automatically different illustrations.

**Rejected:** regenerating all 80 designs; treating every matching name as sufficient; copying all Duel art into a second directory; changing Duel's existing art ids or hashes.

### Intact wonder art inside a wide live board

**Decision:** use seven existing 3:2 paintings on fourteen sides, inside a wider responsive panel with a separate live stage strip.

**Reasons:** the originals and their three sizes are sufficient, and stages remain readable without covering the monuments. One painting per monument avoids redundant day/night scenery.

**Rejected:** cropping or stretching every picture into a wide backdrop; baking costs and effects into pixels; fourteen new paintings. Seven new panoramas remain an explicit alternative only if later layout approval requires full-width scenery.

### Concealed backs follow information visibility

**Decision:** in Haftgane, conceal guilds behind the ordinary Age III back and represent mixed-age piles and tucked stages procedurally. Restrict the violet guild back to public category contexts. Preserve Duel's existing policy.

**Reasons:** a distinct guild or tucked-card back would expose information that Haftgane's rules and platform projection keep hidden.

**Rejected:** applying Duel's public guild-back rule to every game; inspecting hidden card ids in the client; painting four new backs to solve a redaction problem.

## Completion evidence and remaining gates

This proposal enumerates every design, all fourteen board sides and all 24 chain symbols. File inventory confirms approved source art exists for every reuse id and that all seven wonders have all three shipped sizes. All seven wonder originals and the key cross-name or changed-role candidates were inspected directly; the other reuse judgements use approved subjects and manifest critiques. This is an art-reuse design audit, not a new full-set visual certification.

No owner answer is needed to continue the recommended design. Before implementation release, verify card-data licensing/values through the data milestone, approve the board composition in the UI slice, inspect new optical variants in context, measure byte/download gates and complete the generation critique rounds. Those are future acceptance gates; no new artwork, browser render, build, performance result or product-code change is claimed here.

## Owner answers (2026-10-21)

The game is named Haftgane: the title and "the new game" now read Haftgane. Nothing else changed.
