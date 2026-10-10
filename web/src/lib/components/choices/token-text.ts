// A progress token's name and printed effect, in words, for the token
// choice and the token sheet. Display only: it restates the catalog.

import { tokenBySlug } from '$lib/catalog';
import { t } from '$lib/i18n';
import { effectsText } from '../card/effect-text';

/** A token's name, or the slug itself for an unknown one. */
export function tokenName(slug: string): string {
  const token = tokenBySlug(slug);
  return token ? t(`token.${token.slug}`) : slug;
}

/** A token's printed effect as sentences, or an empty string for an unknown one. */
export function tokenEffect(slug: string): string {
  const token = tokenBySlug(slug);
  return token ? `${effectsText(token.effects)}.` : '';
}
