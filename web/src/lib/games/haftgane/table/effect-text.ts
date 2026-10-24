import type { HgEffect } from '$lib/wire/haftgane-catalog.gen';
import { ht } from '../i18n/messages.svelte';

const res = (r: string) => ht(`hg.resource.${r}` as 'hg.resource.wood');
function orList(rs: readonly string[] | undefined): string {
  const names = (rs ?? []).map(res);
  return names.length === 0 ? '' : names.reduce((a, b) => ht('hg.screen.trade.or', { a, b }));
}
function where(scope: readonly string[] | undefined): string {
  const s = scope ?? [];
  const self = s.includes('self');
  const others = s.includes('left') || s.includes('right');
  return self && others
    ? ht('hg.screen.effect.where.all')
    : others
      ? ht('hg.screen.effect.where.neighbours')
      : ht('hg.screen.effect.where.self');
}
function colours(cs: readonly string[] | undefined): string {
  const names = (cs ?? []).map((c) =>
    ht(`hg.screen.colourName.${c}` as 'hg.screen.colourName.red')
  );
  return names.length === 0 ? '' : names.reduce((a, b) => ht('hg.screen.trade.or', { a, b }));
}

export function effectText(effect: HgEffect): string {
  const n = effect.n ?? 1;
  switch (effect.kind) {
    case 'produce':
      return ht('hg.screen.effect.produce', { count: n, resource: res(effect.res?.[0] ?? '') });
    case 'produce-choice':
      return ht('hg.screen.effect.produce-choice', { options: orList(effect.res) });
    case 'trade-discount': {
      const s = effect.scope ?? [];
      const who =
        s.includes('left') && s.includes('right')
          ? ht('hg.screen.effect.who.both')
          : s.includes('left')
            ? ht('hg.screen.effect.who.left')
            : ht('hg.screen.effect.who.right');
      return ht('hg.screen.effect.trade-discount', { options: orList(effect.res), who });
    }
    case 'shields':
      return ht('hg.screen.effect.shields', { count: n });
    case 'science':
      return ht('hg.screen.effect.science', {
        symbol: ht(`hg.science.${effect.symbol ?? 'compass'}`)
      });
    case 'science-choice':
      return ht('hg.screen.effect.science-choice');
    case 'coins':
      return ht('hg.screen.effect.coins', { count: n });
    case 'coins-per-card':
      return ht('hg.screen.effect.coins-per-card', {
        count: n,
        colour: colours(effect.colours),
        where: where(effect.scope)
      });
    case 'coins-per-stage':
      return ht('hg.screen.effect.coins-per-stage', { count: n, where: where(effect.scope) });
    case 'points':
      return ht('hg.screen.effect.points', { count: n });
    case 'points-per-card':
      return ht('hg.screen.effect.points-per-card', {
        count: n,
        colour: colours(effect.colours),
        where: where(effect.scope)
      });
    case 'points-per-stage':
      return ht('hg.screen.effect.points-per-stage', { count: n, where: where(effect.scope) });
    case 'points-if-wonder-complete':
      return ht('hg.screen.effect.points-if-wonder-complete', { count: n });
    case 'play-last-card':
      return ht('hg.screen.effect.play-last-card');
    case 'build-from-discard':
      return ht('hg.screen.effect.build-from-discard');
    case 'free-build':
      return effect.when === 'first-colour'
        ? ht('hg.screen.effect.free-build.first-colour')
        : effect.when === 'first-turn'
          ? ht('hg.screen.effect.free-build.first-turn')
          : ht('hg.screen.effect.free-build.last-turn');
  }
  return '';
}
