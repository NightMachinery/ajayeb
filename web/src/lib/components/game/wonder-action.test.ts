// The card sheet's Build a wonder button (WonderAction.svelte,
// M18.card-sheet-signals): it stands out and names the wonder exactly when
// the view's choices let the card go under one now.
import { describe, expect, it } from 'vitest';
import { render } from 'svelte/server';
import { protocolFixtureNames, protocolFixtures } from '$lib/wire/fixtures.gen';
import type { View } from '$lib/wire';
import { cardActions, readyWonders } from './card-actions';
import WonderAction from './WonderAction.svelte';

function html(ready: readonly string[]): string {
  return render(WonderAction, { props: { ready, onclick: () => undefined } }).body;
}

describe('the Build a wonder button', () => {
  it('stands out only while a wonder is ready, and names it', () => {
    const plain = html([]);
    expect(plain).toContain('Build a wonder');
    expect(plain).not.toContain('data-wonder-ready');
    expect(plain).not.toContain('border-accent');
    const one = html(['pyramids']);
    expect(one).toContain('data-wonder-ready="pyramids"');
    expect(one).toContain('border-accent');
    expect(one).toMatch(/Ready: .*Pyramids/);
    expect(html(['pyramids', 'colossus'])).toContain('Ready: 2 wonders');
  });

  it('follows the choices of a real view', () => {
    const costs: View = protocolFixtures[protocolFixtureNames.indexOf('costs-seat0')]!;
    expect(html(readyWonders(cardActions(costs, 'courthouse')))).toContain(
      'data-wonder-ready="pyramids temple-of-artemis"'
    );
    const none = { ...costs, choices: costs.choices.filter((c) => c.kind !== 'wonder') };
    expect(html(readyWonders(cardActions(none, 'courthouse')))).not.toContain('data-wonder-ready');
  });
});
