import { render } from 'svelte/server';
import { describe, expect, it } from 'vitest';
import { WONDER_ART_SIZES, WONDER_SHEET_ART_SIZES } from '$lib/art/lookup';
import WonderSheet from './WonderSheet.svelte';

/** The `sizes` attribute of every <img> in some markup. */
function sizesOf(markup: string): string[] {
  return [...markup.matchAll(/<img[^>]*\ssizes="([^"]*)"/g)].map((m) => m[1]!);
}

describe('the wonder sheet art', () => {
  it('asks for a 280 px tile, not half the viewport', () => {
    const markup = render(WonderSheet, {
      props: {
        open: true,
        slug: 'pyramids',
        buildable: false,
        canAct: false,
        onbuild: () => {},
        onclose: () => {}
      }
    }).body;
    expect(sizesOf(markup)).toEqual([WONDER_SHEET_ART_SIZES]);
    expect(WONDER_SHEET_ART_SIZES).not.toBe(WONDER_ART_SIZES);
    expect(WONDER_SHEET_ART_SIZES.endsWith('280px')).toBe(true);
  });
});
