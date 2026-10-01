# Legal: licenses, trademarks and what we never copy

Ajayeb is a fan project. It implements the rules of a published board game with original code, original art and original wording. This page records the licenses we use, the trademark notice we show, and the lines we do not cross. The short version is in [NOTICE.md](../NOTICE.md).

## Licenses

- **Code is MIT.** The license text is in [LICENSE](../LICENSE). It covers everything in the repository that is not listed below.
- **Original art is CC BY 4.0** (Creative Commons Attribution 4.0 International). This covers the approved originals under `art/` (including `art/originals/` and the style references in `art/refs/`), their AVIF derivatives under `web/static/art/`, and the hand-drawn SVG icons under `web/src/lib/icons/svg/`. The license file will live in `art/` and arrives with the first art commit. The owner may change this license before any art is published.
- **Fonts are OFL** (SIL Open Font License) and self-hosted, never loaded from a font CDN. Each font ships with its license file. We do not use the typeface of the official logo.
  - The interface uses Fraunces 1.000 (headings) and Inter 4.001 (text), both under OFL 1.1. Neither declares a Reserved Font Name, so the subsets keep their original names.
  - The license texts ship next to the fonts as `web/static/fonts/OFL-Fraunces.txt` and `web/static/fonts/OFL-Inter.txt`.
  - The original TTFs are kept in `web/font-sources/`, and `web/font-sources/subset.sh` rebuilds the shipped WOFF2 subsets from them. For Persian lettering in the logo we use an OFL Nastaliq font; fonts whose license is unclear are not used.
- **Third-party code and data** keep their own licenses. Any open-source data or code we draw on is checked for its license first, credited, and recorded in `docs/rules/sources.md`. Open-source implementations of the game are read to understand behaviour, not copied, unless their license allows it and the copy is credited.

## Trademark notice

The README footer and the in-app About page carry this disclaimer:

> Ajayeb is an unofficial, non-commercial fan implementation of the rules of 7 Wonders Duel by Antoine Bauza and Bruno Cathala (Repos Production). It is not affiliated with or endorsed by Repos Production or Asmodee; 7 Wonders Duel is their trademark.

Both places link to the publisher's page so people can buy the real game: <https://www.rprod.com/en/games/7-wonders-duel>.

Rules we follow around the name:

- The product name is only "Ajayeb". The same goes for the domain, package names, the page `<title>`, meta tags and the logo.
- "7 Wonders" and "Duel" do not appear in SEO metadata. The game's name appears only where we need to say what Ajayeb implements, such as the disclaimer and the README.
- There are no ads and no monetization of any kind.
- We comply promptly with any takedown request from the rights holders. Requests can go to the issue tracker at <https://github.com/NightMachinery/ajayeb/issues>.

## What we never copy

- Official card, wonder or box art, or the logo.
- The official iconography. We draw our own resource and science symbols, shields, coins and every other glyph.
- The card-frame trade dress and the layout graphics of the physical game.
- Rulebook text. `docs/rules/` states the rules in our own words, and no rulebook PDF is ever committed.
- Images from BoardGameGeek, from the publisher, or from any other site.

## Game data

Game mechanics and numeric facts about cards (costs, effects, chains, counts) are not protected by copyright. We re-express them in our own data format, and record where each fact was checked in `docs/rules/sources.md`.

Display names live in the client's translation tables, keyed by engine ids, so renaming any card, wonder or token later is a one-file change. The building names are generic words and the wonder names are historical, so we consider them low risk.

## Privacy

- Commits contain no personal information.
- Private run state (task logs, progress notes, schedules) stays outside the repository.
- Local caches, data directories, self-hosting state and personal agent files (`.cache/`, `.self-host/`, `CLAUDE.local.md`) are ignored by git.
- The server stores only what a game needs: a display name, an HMAC-derived user id (never the browser's token), and the rooms' seeds and action logs. Old rooms are deleted automatically.
