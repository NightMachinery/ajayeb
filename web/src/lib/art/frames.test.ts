// The art frames, rendered on the server with a stand-in art index: approved
// ids draw a lazy <img>, every other id draws its procedural placeholder,
// and every card tier but the colour band draws its art.
import { createRawSnippet } from 'svelte';
import { render } from 'svelte/server';
import { describe, expect, it, vi } from 'vitest';
import CardArt from '$lib/components/card/CardArt.svelte';
import CardBack from '$lib/components/card/CardBack.svelte';
import CardFace from '$lib/components/card/CardFace.svelte';
import ProgressToken from '$lib/components/token/ProgressToken.svelte';
import WonderTile from '$lib/components/wonder/WonderTile.svelte';
import ArtImage from './ArtImage.svelte';
import BotAvatar from './BotAvatar.svelte';

vi.mock('./available.gen', () => {
  const variant = (path: string, width: number, height: number) => ({
    path,
    width,
    height,
    bytes: 1000
  });
  return {
    art: {
      'card/lumber-yard': {
        kind: 'card',
        variants: {
          default: variant('/art/cards/lumber-yard.avif', 480, 360),
          thumb: variant('/art/cards/lumber-yard@thumb.avif', 240, 180)
        }
      },
      'wonder/pyramids': {
        kind: 'wonder',
        variants: {
          default: variant('/art/wonders/pyramids.avif', 960, 640),
          thumb: variant('/art/wonders/pyramids@thumb.avif', 480, 320)
        }
      },
      'token/law': {
        kind: 'token',
        variants: {
          default: variant('/art/tokens/law.avif', 256, 256),
          thumb: variant('/art/tokens/law@thumb.avif', 96, 96)
        }
      },
      'back/age2': {
        kind: 'back',
        variants: { default: variant('/art/backs/age2.avif', 320, 495) }
      },
      'wonder/mausoleum': {
        kind: 'wonder',
        variants: {
          default: variant('/art/wonders/mausoleum.avif', 960, 640),
          thumb: variant('/art/wonders/mausoleum@thumb.avif', 480, 320)
        }
      },
      'avatar/scholar': {
        kind: 'avatar',
        variants: { default: variant('/art/avatars/scholar.avif', 192, 192) }
      }
    }
  };
});

function imgs(markup: string): string[] {
  return markup.match(/<img\b[^>]*>/g) ?? [];
}

const fallback = createRawSnippet(() => ({ render: () => '<i data-fallback=""></i>' }));

describe('ArtImage', () => {
  it('draws approved art as a lazy, async, decorative image', () => {
    const [img, ...rest] = imgs(
      render(ArtImage, { props: { id: 'card/lumber-yard', size: 'thumb', fallback } }).body
    );
    expect(rest).toEqual([]);
    expect(img).toContain('src="/art/cards/lumber-yard@thumb.avif"');
    expect(img).toContain('loading="lazy"');
    expect(img).toContain('decoding="async"');
    expect(img).toContain('alt=""');
    expect(img).toContain('data-art="card/lumber-yard"');
    expect(img).not.toContain('srcset');
  });

  it('offers every size in a srcset when given sizes', () => {
    const [img] = imgs(
      render(ArtImage, { props: { id: 'card/lumber-yard', sizes: '140px', fallback } }).body
    );
    expect(img).toContain('src="/art/cards/lumber-yard.avif"');
    expect(img).toContain(
      'srcset="/art/cards/lumber-yard@thumb.avif 240w, /art/cards/lumber-yard.avif 480w"'
    );
    expect(img).toContain('sizes="140px"');
  });

  it('draws the fallback, and no image, for an id with no approved art', () => {
    const markup = render(ArtImage, { props: { id: 'card/stone-pit', fallback } }).body;
    expect(imgs(markup)).toEqual([]);
    expect(markup).toContain('data-fallback');
  });

  it('draws nothing at all with no art and no fallback', () => {
    expect(imgs(render(ArtImage, { props: { id: 'back/age1' } }).body)).toEqual([]);
  });
});

describe('card frames', () => {
  it('shows the art at the mini, medium and full tiers, and never on a band', () => {
    const medium = render(CardFace, {
      props: { slug: 'lumber-yard', tier: 'medium', width: 101 }
    }).body;
    expect(imgs(medium)).toHaveLength(1);
    expect(imgs(medium)[0]).toContain('/art/cards/lumber-yard@thumb.avif');

    const full = render(CardFace, {
      props: { slug: 'lumber-yard', tier: 'full', width: 180 }
    }).body;
    expect(imgs(full)).toHaveLength(1);
    expect(imgs(full)[0]).toContain('src="/art/cards/lumber-yard.avif"');
    expect(imgs(full)[0]).toContain('sizes="180px"');

    // The phone pyramid: the thumbnail below the 40% line, with the
    // narrower edge fade.
    const mini = imgs(
      render(CardFace, { props: { slug: 'lumber-yard', tier: 'mini', width: 54 } }).body
    );
    expect(mini).toHaveLength(1);
    expect(mini[0]).toContain('src="/art/cards/lumber-yard@thumb.avif"');
    expect(mini[0]).toContain('feather-edge');
    expect(mini[0]).not.toContain('srcset');
    expect(imgs(render(CardFace, { props: { slug: 'lumber-yard', tier: 'band' } }).body)).toEqual(
      []
    );
  });

  it('spells out no chain-out line on the full card (the chain chip carries it)', () => {
    // Archery Range chains to Siege Workshop; the sheet beside the card says so.
    const full = render(CardFace, {
      props: { slug: 'archery-range', tier: 'full', width: 140 }
    }).body;
    expect(full).not.toContain('Siege Workshop');
    expect(full).toContain('Military building');
  });

  it('breaks names between words, and shrinks one whose longest word is long', () => {
    const medium = render(CardFace, {
      props: { slug: 'scriptorium', tier: 'medium', width: 101 }
    }).body;
    expect(medium).not.toContain('wrap-anywhere');
    expect(medium).toContain('wrap-normal');
    expect(medium).toContain('--name-chars: 11');
  });

  it('keeps the placeholder for a card with no approved art', () => {
    const markup = render(CardArt, {
      props: { slug: 'stone-pit', color: 'brown', tier: 'full' }
    }).body;
    expect(imgs(markup)).toEqual([]);
    expect(markup).toContain('data-card-art="stone-pit"');
    expect(markup).toContain('i-category-brown');
  });

  it('puts approved back art under the frame and the numeral, else the lattice', () => {
    const age2 = render(CardBack, { props: { back: 'age2' } }).body;
    expect(imgs(age2)).toHaveLength(1);
    expect(imgs(age2)[0]).toContain('/art/backs/age2.avif');
    expect(age2).toContain('<text');
    const age1 = render(CardBack, { props: { back: 'age1' } }).body;
    expect(imgs(age1)).toEqual([]);
    expect(age1).toContain('<text');
  });
});

describe('wonder, token and avatar frames', () => {
  it('shows wonder art in the draft tile and the city row, else the placeholder sky', () => {
    const draft = imgs(render(WonderTile, { props: { slug: 'pyramids', size: 'draft' } }).body);
    expect(draft).toHaveLength(1);
    expect(draft[0]).toContain('srcset=');
    const city = imgs(render(WonderTile, { props: { slug: 'pyramids', size: 'city' } }).body);
    expect(city).toHaveLength(1);
    expect(city[0]).toContain('src="/art/wonders/pyramids@thumb.avif"');
    const other = render(WonderTile, { props: { slug: 'colossus', size: 'draft' } }).body;
    expect(imgs(other)).toEqual([]);
    expect(other).toContain('wonder-art');
  });

  it('anchors the Mausoleum’s crop near the top, in the draft tile and the city row', () => {
    for (const size of ['draft', 'city'] as const) {
      const [img] = imgs(render(WonderTile, { props: { slug: 'mausoleum', size } }).body);
      expect(img, size).toContain('object-position: 50% 12%');
    }
    const [draft] = imgs(render(WonderTile, { props: { slug: 'mausoleum', size: 'draft' } }).body);
    expect(draft).toContain('feather-sky');
    const [pyramids] = imgs(
      render(WonderTile, { props: { slug: 'pyramids', size: 'draft' } }).body
    );
    expect(pyramids).toContain('object-position: 50% 50%');
    expect(pyramids).toContain('feather-soft');
  });

  it('shows a token in a city in full colour, and one gone from the board greyed', () => {
    const owned = render(ProgressToken, { props: { slug: 'law', size: 'sm', owned: true } }).body;
    expect(owned).toContain('data-owned');
    expect(owned).not.toContain('grayscale');
    expect(owned).toContain('Law (owned)');
    const taken = render(ProgressToken, { props: { slug: 'law', taken: true } }).body;
    expect(taken).toContain('data-taken');
    expect(taken).toContain('grayscale');
    expect(taken).toContain('Law (taken)');
  });

  it('shows token art in the medallion, else the frame and initial', () => {
    const law = render(ProgressToken, { props: { slug: 'law' } }).body;
    expect(imgs(law)).toHaveLength(1);
    expect(imgs(law)[0]).toContain('/art/tokens/law@thumb.avif');
    expect(law).not.toContain('i-token-frame');
    const urbanism = render(ProgressToken, {
      props: { slug: 'urbanism', actionKey: 'token:urbanism' }
    }).body;
    expect(imgs(urbanism)).toEqual([]);
    expect(urbanism).toContain('i-token-frame');
    expect(urbanism).toContain('data-act="token:urbanism"');
  });

  it('shows a bot’s portrait, else its initial', () => {
    expect(imgs(render(BotAvatar, { props: { level: 'normal' } }).body)[0]).toContain(
      '/art/avatars/scholar.avif'
    );
    const easy = render(BotAvatar, { props: { level: 'easy' } }).body;
    expect(imgs(easy)).toEqual([]);
    expect(easy).toMatch(/>A<\/span>/);
  });
});
