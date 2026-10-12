import { describe, expect, it } from 'vitest';
import { cardSlugs, tokenSlugs, wonderSlugs } from '$lib/catalog';
import { art } from './available.gen';
import {
  artBytes,
  artEntry,
  artId,
  artSource,
  cardArtSizes,
  tokenArtSizes,
  WONDER_ART_SIZES,
  avatarArtId,
  backArtId,
  hasApprovedArt,
  pickArt,
  seatBotLevel,
  victoryArtId,
  type ArtIndex
} from './lookup';

// A small index in the generated module's shape, standing in for approved
// art (the committed index may be empty, so the tests never rely on it).
const index: ArtIndex = {
  'card/lumber-yard': {
    kind: 'card',
    variants: {
      default: { path: '/art/cards/lumber-yard.avif', width: 480, height: 360, bytes: 30000 },
      thumb: { path: '/art/cards/lumber-yard@thumb.avif', width: 240, height: 180, bytes: 9000 }
    }
  },
  'wonder/pyramids': {
    kind: 'wonder',
    variants: {
      default: { path: '/art/wonders/pyramids.avif', width: 960, height: 640, bytes: 0 }
    }
  },
  'icon/app': {
    kind: 'icon',
    variants: {
      '32': { path: '/art/icon/app@32.avif', width: 32, height: 32, bytes: 900 },
      '512': { path: '/art/icon/app@512.avif', width: 512, height: 512, bytes: 50000 },
      '192': { path: '/art/icon/app@192.avif', width: 192, height: 192, bytes: 15000 }
    }
  },
  'token/law': { kind: 'token', variants: {} },
  'token/broken': {
    kind: 'token',
    variants: { default: { path: '', width: 0, height: 0, bytes: 0 } }
  }
};

describe('art ids', () => {
  it('builds "<kind>/<slug>" ids', () => {
    expect(artId('card', 'lumber-yard')).toBe('card/lumber-yard');
    expect(artId('wonder', 'pyramids')).toBe('wonder/pyramids');
    expect(backArtId('guild')).toBe('back/guild');
  });

  it('names the avatars by persona', () => {
    expect(avatarArtId('easy')).toBe('avatar/apprentice');
    expect(avatarArtId('normal')).toBe('avatar/scholar');
    expect(avatarArtId('hard')).toBe('avatar/master');
  });
});

describe('pickArt', () => {
  it('picks the preferred size, with every size in the srcset', () => {
    expect(pickArt('card/lumber-yard', 'thumb', index)).toEqual({
      id: 'card/lumber-yard',
      src: '/art/cards/lumber-yard@thumb.avif',
      width: 240,
      height: 180,
      srcset: '/art/cards/lumber-yard@thumb.avif 240w, /art/cards/lumber-yard.avif 480w'
    });
    expect(pickArt('card/lumber-yard', 'default', index)?.src).toBe('/art/cards/lumber-yard.avif');
    expect(pickArt('card/lumber-yard', undefined, index)?.src).toBe('/art/cards/lumber-yard.avif');
  });

  it('falls back to the default size when the preferred one is missing', () => {
    expect(pickArt('wonder/pyramids', 'thumb', index)?.src).toBe('/art/wonders/pyramids.avif');
  });

  it('falls back to the widest size when there is no default', () => {
    const pick = pickArt('icon/app', 'thumb', index);
    expect(pick?.src).toBe('/art/icon/app@512.avif');
    expect(pick?.srcset).toBe(
      '/art/icon/app@32.avif 32w, /art/icon/app@192.avif 192w, /art/icon/app@512.avif 512w'
    );
    expect(pickArt('icon/app', '192', index)?.src).toBe('/art/icon/app@192.avif');
  });

  it('gives nothing for an id with no approved art, so the placeholder draws', () => {
    expect(pickArt('card/stone-pit', 'thumb', index)).toBeUndefined();
    expect(pickArt('', 'default', index)).toBeUndefined();
    expect(hasApprovedArt('card/stone-pit', index)).toBe(false);
    expect(hasApprovedArt('card/lumber-yard', index)).toBe(true);
  });

  it('gives nothing for an entry with no usable variant', () => {
    expect(pickArt('token/law', 'default', index)).toBeUndefined();
    expect(pickArt('token/broken', 'default', index)).toBeUndefined();
    expect(hasApprovedArt('token/broken', index)).toBe(false);
  });

  it('never mistakes an inherited key for art', () => {
    for (const id of ['constructor', 'toString', '__proto__', 'hasOwnProperty']) {
      expect(artEntry(id, index)).toBeUndefined();
      expect(pickArt(id, 'default', index)).toBeUndefined();
      expect(hasApprovedArt(id, index)).toBe(false);
    }
  });
});

describe('artBytes', () => {
  it('reports a variant’s recorded size, and nothing when it is unknown', () => {
    expect(artBytes('card/lumber-yard', 'thumb', index)).toBe(9000);
    expect(artBytes('wonder/pyramids', 'default', index)).toBeUndefined();
    expect(artBytes('wonder/pyramids', 'thumb', index)).toBeUndefined();
    expect(artBytes('card/stone-pit', 'thumb', index)).toBeUndefined();
  });
});

describe('the generated index', () => {
  it('only lists ids the app can show, each with a usable shipped file', () => {
    const known = new Set([
      ...cardSlugs.map((s) => `card/${s}`),
      ...wonderSlugs.map((s) => `wonder/${s}`),
      ...tokenSlugs.map((s) => `token/${s}`)
    ]);
    for (const id of Object.keys(art)) {
      const kind = id.split('/')[0];
      if (kind === 'card' || kind === 'wonder' || kind === 'token') {
        expect(known.has(id), id).toBe(true);
      }
      const pick = pickArt(id);
      expect(pick, id).toBeDefined();
      expect(pick?.src, id).toMatch(/^\/art\/[a-z0-9-]+\/[a-z0-9@-]+\.avif$/);
    }
  });

  it('is what the lookups read by default', () => {
    for (const id of ['card/lumber-yard', 'wonder/pyramids', 'token/law', 'back/age1']) {
      expect(hasApprovedArt(id)).toBe(Object.hasOwn(art, id));
    }
  });
});

describe('bot avatars by seat', () => {
  const room = {
    seats: [
      { seat: 0, kind: 'human' as const },
      { seat: 1, kind: 'bot' as const, bot: 'hard' as const }
    ]
  };

  it('gives the level of the seat a bot plays', () => {
    expect(seatBotLevel(room, 1)).toBe('hard');
    expect(avatarArtId(seatBotLevel(room, 1)!)).toBe('avatar/master');
  });

  it('gives nothing for a human or open seat, or no room', () => {
    expect(seatBotLevel(room, 0)).toBeUndefined();
    expect(seatBotLevel({ seats: [{ seat: 1, kind: 'open' }] }, 1)).toBeUndefined();
    expect(seatBotLevel(null, 1)).toBeUndefined();
    expect(seatBotLevel(undefined, 0)).toBeUndefined();
  });
});

describe('victory scenes', () => {
  const scene = (id: string) => ({
    [id]: {
      kind: 'victory',
      variants: { default: { path: `/art/${id}.avif`, width: 960, height: 540, bytes: 1 } }
    }
  });
  const three: ArtIndex = {
    ...scene('victory/civilian'),
    ...scene('victory/military'),
    ...scene('victory/science')
  };

  it('matches each supremacy and the civilian victory to its own scene', () => {
    expect(victoryArtId('military', three)).toBe('victory/military');
    expect(victoryArtId('science', three)).toBe('victory/science');
    expect(victoryArtId('civilian', three)).toBe('victory/civilian');
  });

  it('shows the civilian scene for a forfeit', () => {
    expect(victoryArtId('forfeit', three)).toBe('victory/civilian');
  });

  it('shows the civilian scene for a draw until a draw scene is approved', () => {
    expect(victoryArtId('shared', three)).toBe('victory/civilian');
    expect(victoryArtId('shared', { ...three, ...scene('victory/draw') })).toBe('victory/draw');
  });

  it('names the first candidate when nothing is approved (its placeholder shows)', () => {
    expect(victoryArtId('science', {})).toBe('victory/science');
    expect(victoryArtId('shared', {})).toBe('victory/draw');
    expect(pickArt(victoryArtId('military', {}), 'default', {})).toBeUndefined();
  });

  it('finds every scene the result screen can ask for in the generated index', () => {
    for (const kind of ['civilian', 'military', 'science', 'shared', 'forfeit'] as const) {
      expect(hasApprovedArt(victoryArtId(kind)), kind).toBe(true);
    }
  });
});

describe('artSource', () => {
  it('gives a fixed src without sizes, and the srcset with them', () => {
    expect(artSource('card/lumber-yard', 'thumb', undefined, index)).toEqual({
      src: '/art/cards/lumber-yard@thumb.avif'
    });
    expect(artSource('card/lumber-yard', 'default', cardArtSizes(140), index)).toEqual({
      src: '/art/cards/lumber-yard.avif',
      srcset: '/art/cards/lumber-yard@thumb.avif 240w, /art/cards/lumber-yard.avif 480w',
      sizes: '140px'
    });
    expect(artSource('card/stone-pit', 'default', '140px', index)).toBeUndefined();
  });

  it('names the sizes of each frame', () => {
    expect(cardArtSizes(139.6)).toBe('140px');
    expect(tokenArtSizes('md')).toBe('36px');
    expect(tokenArtSizes('sm')).toBe('26px');
    expect(tokenArtSizes('xs')).toBe('20px');
    // Half the viewport for the two-a-row draft, 280 px in the split layout
    // (app.css: a 64rem width, or a short landscape screen).
    expect(WONDER_ART_SIZES).toBe(
      '(min-width: 64rem) 280px, (orientation: landscape) and (max-height: 32rem) 280px, 50vw'
    );
  });
});
