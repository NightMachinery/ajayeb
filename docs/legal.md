# Legal: licenses, trademarks and what we never copy

Ajayeb is a fan project. It implements the rules of a published board game with original code, original art and original wording. This page records the licenses we use, the trademark notice we show, and the lines we do not cross. The short version is in [NOTICE.md](../NOTICE.md).

## Licenses

- **Code is MIT.** The license text is in [LICENSE](../LICENSE). It covers everything in the repository that is not listed below.
- **Original art is CC BY 4.0** (Creative Commons Attribution 4.0 International). Except for the CC0 sounds listed below, this covers the approved originals under `art/` (including `art/originals/` and the style references in `art/refs/`), their AVIF derivatives under `web/static/art/`, the hand-drawn SVG icons under `web/src/lib/icons/svg/`, the ornaments and track cells under `web/src/lib/ornaments/svg/`, and the logo wordmark and favicon (`web/src/lib/brand/`, `web/static/favicon.svg`). The notice is [art/LICENSE](../art/LICENSE).
- **Fonts are OFL** (SIL Open Font License) and self-hosted, never loaded from a font CDN. Each font ships with its license file. We do not use the typeface of the official logo.
  - The interface uses Fraunces 1.000 (headings) and Inter 4.001 (text), both under OFL 1.1. Neither declares a Reserved Font Name, so the subsets keep their original names.
  - The shipped WOFF2 subsets live in `web/src/lib/fonts/` with their license texts, `OFL-Fraunces.txt` and `OFL-Inter.txt`, beside them. The site serves the same texts at `/fonts/OFL-Fraunces.txt` and `/fonts/OFL-Inter.txt`, from `web/static/fonts/`.
  - The original TTFs are kept in `web/font-sources/`, and `web/font-sources/subset.sh` rebuilds the shipped WOFF2 subsets from them. Fonts whose license is unclear are not used.
- **The logo is artwork made from font outlines; no logo font ships.** The wordmark (`web/src/lib/brand/logo-wordmark.svg`) and the favicon (`web/static/favicon.svg`) are SVG paths, not text, so the browser never loads a font for them.
  - The Persian word عجایب is outlined from **Noto Nastaliq Urdu 4.000** (Regular for the wordmark, Bold for the letter ع of the favicon), from the upstream release `NotoNastaliqUrdu-v4.000` of the Noto project (github.com/notofonts/nastaliq). Its name table declares the SIL Open Font License 1.1, copyright 2022 The Noto Project Authors, with no Reserved Font Name.
  - The Latin line "Ajayeb" is outlined from Fraunces 600, the same OFL 1.1 source as the interface font (`web/font-sources/fraunces-600.ttf`).
  - OFL 1.1 governs the font software, not artwork made with it, so the outlines need no license file or credit of their own; we record their source here anyway. The outlines were set with `hb-view` and resolved into plain paths; the iteration log `art/vector/log/ornaments-track-logo.md` has the steps.
  - Never used for the logo: IranNastaliq (installed on some systems, proprietary license), and the typeface or lettering of the official game's logo.
- **Third-party code and data** keep their own licenses. Any open-source data or code we draw on is checked for its license first, credited, and recorded in `docs/rules/sources.md`. Open-source implementations of the game are read to understand behaviour, not copied, unless their license allows it and the copy is credited.

## Sounds (CC0 1.0)

The source audio under `art/originals/sound/` and its AAC derivatives under
`web/static/sfx/` use [CC0 1.0](https://creativecommons.org/publicdomain/zero/1.0/),
separately from Ajayeb's original visual art. Every linked source page was
re-fetched and checked for CC0 on 2026-10-18; the Kenney archives' `License.txt`
files were checked too. Credits below retain the creators' published names.

Each cue's original is `<cue>.flac` and its shipped file is `<cue>.m4a`.
The FLAC files preserve decoded PCM from the downloads without further lossy
compression. Kenney and OpenGameArt sources are the pack originals. Freesound
sources are the public HQ MP3 previews used for the listening shortlist;
downloading their higher-quality originals requires an account. The preview
URLs, download SHA-256 hashes, archive member paths and member hashes are recorded in
[the source manifest](../art/originals/sound/sources.json).

- `slide`: Casino Audio: card-slide-1, by Kenney (kenney.nl). [Source](https://kenney.nl/assets/casino-audio). CC0 1.0; checked 2026-10-18.
- `coin`: RPG Audio: handleCoins, by Kenney (kenney.nl). [Source](https://kenney.nl/assets/rpg-audio). CC0 1.0; checked 2026-10-18.
- `flip`: OpenGameArt: paper_01, by rubberduck. [Source](https://opengameart.org/content/100-cc0-sfx). CC0 1.0; checked 2026-10-18.
- `drum`: Freesound: taiko.wav, by Anuj_Shrestha1. [Source](https://freesound.org/people/Anuj_Shrestha1/sounds/486140/). CC0 1.0; checked 2026-10-18. Public HQ MP3 preview.
- `chime`: Interface Sounds: glass_001, by Kenney (kenney.nl). [Source](https://kenney.nl/assets/interface-sounds). CC0 1.0; checked 2026-10-18.
- `fanfare`: Freesound: Tada Fanfare A, by plasterbrain. [Source](https://freesound.org/people/plasterbrain/sounds/397355/). CC0 1.0; checked 2026-10-18. Public HQ MP3 preview.
- `tick`: Freesound: Basic Click Wooden, by GameAudio. [Source](https://freesound.org/people/GameAudio/sounds/220200/). CC0 1.0; checked 2026-10-18. Public HQ MP3 preview.
- `wonder`: OpenGameArt: gong_01, by rubberduck. [Source](https://opengameart.org/content/100-cc0-sfx). CC0 1.0; checked 2026-10-18.
- `token`: Freesound: Stamp, by kermite607. [Source](https://freesound.org/people/kermite607/sounds/362624/). CC0 1.0; checked 2026-10-18. Public HQ MP3 preview.
- `loot`: Freesound: coins-falling-022.wav, by Gerent. [Source](https://freesound.org/people/Gerent/sounds/558991/). CC0 1.0; checked 2026-10-18. Public HQ MP3 preview.
- `ageStart`: Casino Audio: card-fan-2, by Kenney (kenney.nl). [Source](https://kenney.nl/assets/casino-audio). CC0 1.0; checked 2026-10-18.
- `defeat`: Freesound: failfare.mp3, by Wagna. [Source](https://freesound.org/people/Wagna/sounds/242208/). CC0 1.0; checked 2026-10-18. Public HQ MP3 preview.
- `yourTurn`: OpenGameArt: wooden_01, by rubberduck. [Source](https://opengameart.org/content/100-cc0-sfx). CC0 1.0; checked 2026-10-18.
- `destroy`: Freesound: tearing paper4.wav, by keweldog. [Source](https://freesound.org/people/keweldog/sounds/181773/). CC0 1.0; checked 2026-10-18. Public HQ MP3 preview.
- `rebuild`: Casino Audio: cards-pack-take-out-1, by Kenney (kenney.nl). [Source](https://kenney.nl/assets/casino-audio). CC0 1.0; checked 2026-10-18.
- `libraryDraw`: Freesound: Unrolling Scroll.wav, by spookymodem. [Source](https://freesound.org/people/spookymodem/sounds/202107/). CC0 1.0; checked 2026-10-18. Public HQ MP3 preview.
- `extraTurn`: Freesound: Soft-Notifications - Bell - Ding-Dong.mp3, by LegitCheese. [Source](https://freesound.org/people/LegitCheese/sounds/571513/). CC0 1.0; checked 2026-10-18. Public HQ MP3 preview.

The `yourTurn` source is `wooden_01` (shortlist `extraTurn-03`); `extraTurn`
uses Soft-Notifications (shortlist `yourTurn-00`), following the listening
selection's swap. No `uiTap` file is shipped. [Sound preparation](art/sound.md)
describes trimming, loudness matching, encoding and content hashes.

## Trademark notice

The README footer and the footer of the app's Home screen carry this disclaimer:

> Ajayeb is an unofficial, non-commercial fan implementation of the rules of 7 Wonders Duel by Antoine Bauza and Bruno Cathala (Repos Production). It is not affiliated with or endorsed by Repos Production or Asmodee; 7 Wonders Duel is their trademark.

Both places link to the publisher's page so people can buy the real game: <https://www.rprod.com/en/games/7-wonders-duel>. The Home footer also says that Ajayeb itself is free and open-source software under the MIT License and links to its source code: <https://github.com/NightMachinery/ajayeb>.

Rules we follow around the name:

- The product name is only "Ajayeb". The same goes for the domain, package names, the page `<title>`, meta tags and the logo.
- "7 Wonders" and "Duel" do not appear in SEO metadata. The game's name appears only where we need to say what Ajayeb implements, such as the disclaimer and the README.
- There are no ads and no monetization of any kind.
- We comply promptly with any takedown request from the rights holders. Requests can go to the issue tracker at <https://github.com/NightMachinery/ajayeb/issues>.

## What we never copy

- Official card, wonder or box art, or the logo.
- The official iconography. We draw our own resource and science symbols, shields, coins and every other glyph.
- The card-frame trade dress and the layout graphics of the physical game.
- Rulebook text. `docs/rules/` states the rules in our own words, and no rulebook PDF is ever committed. The rules reference in the app (the `/rules` page and the in-game Rules panel, `web/src/lib/i18n/en/rules.ts`) is written in our own words from `docs/rules/`, never from the rulebook, and copies neither its layout nor its diagrams: its small diagrams are built from our own components and icons.
- Images from BoardGameGeek, from the publisher, or from any other site.

## Game data

Game mechanics and numeric facts about cards (costs, effects, chains, counts) are not protected by copyright. We re-express them in our own data format, and record where each fact was checked in `docs/rules/sources.md`.

Display names live in the client's translation tables, keyed by engine ids, so renaming any card, wonder or token later is a one-file change. The building names are generic words and the wonder names are historical, so we consider them low risk.

## Privacy

- Commits contain no personal information.
- Private run state (task logs, progress notes, schedules) stays outside the repository.
- Local caches, data directories, self-hosting state and personal agent files (`.cache/`, `.self-host/`, `CLAUDE.local.md`) are ignored by git.
- The server stores only what a game needs: a display name, an HMAC-derived user id (never the browser's token), and the rooms' seeds and action logs. Old rooms are deleted automatically.
