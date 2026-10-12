// The decider's hints (the hints design, M10.HS3): pure display mappers over
// the server's Choice.outcome and DisabledChoice.wins. The server decides
// every fact (the coins and points after a move, the pawn and its looting,
// a win, the game's end, the turn order, the wins a move gives the opponent
// or sets up, the chance a reveal lets the opponent win); this file only
// words them, for the outcome line under each move button, the card
// sheet's "After this pick" panel, the confirm step's warning and the
// starter and token choices. The only arithmetic is display: the change of
// a seat's coins is its coins after (from the server) minus its coins now
// (from the view).
//
// Each part follows its family (the game's settings minus the local hide):
// scoring for coins, points and the end; military for the pawn; science
// for the symbols; tempo for the turn order and what a move uncovers;
// threats for wins; prices for what a freed card costs the opponent.

import type {
  Choice,
  HintFamily,
  OutcomeView,
  PawnMoveView,
  SlotView,
  ThreatView,
  View
} from '$lib/wire';
import type { IconName } from '$lib/icons/names';
import { cardName, joinList, wonderName } from '$lib/components/card/effect-text';
import { tokenName } from '$lib/components/game/names';
import { formatNumber, t } from '$lib/i18n';
import { freedSplit } from './freed';
import type { HintNames, Seat } from './hints';

function other(seat: Seat): Seat {
  return seat === 0 ? 1 : 0;
}

/** A signed number for display: "+3", "−1", "0". */
export function signed(n: number): string {
  return formatNumber(n, { signDisplay: 'exceptZero' });
}

/**
 * The name of what an action key names, for display only (never to decide
 * anything): the card of construct, discard, revive and destroy keys, the
 * wonder of a wonder key, the token of a token key. Undefined for any other
 * key.
 */
export function keyTargetName(key: string): string | undefined {
  const [verb, a] = key.split(':');
  switch (verb) {
    case 'construct':
    case 'discard':
    case 'revive':
    case 'destroy':
      return a ? cardName(a) : undefined;
    case 'wonder':
      return a ? wonderName(a) : undefined;
    case 'token':
      return a ? tokenName(a) : undefined;
  }
  return undefined;
}

/** The name of a threat's card or wonder. */
export function threatName(th: ThreatView): string {
  if (th.wonder) return wonderName(th.wonder);
  if (th.card) return cardName(th.card);
  return '';
}

/** The names of threats, joined for the locale ("Arsenal or the Colossus"). */
export function threatList(list: readonly ThreatView[]): string {
  const names = [...new Set(list.map(threatName).filter((n) => n !== ''))];
  return joinList(names, 'disjunction');
}

// --- The outcome line ------------------------------------------------------

/** One term of an outcome line. */
export interface OutcomeTerm {
  /** A stable id (data-outcome-term). */
  readonly id: string;
  /** The term with its words (desktop, tooltips, screen readers). */
  readonly text: string;
  /** The short form beside its glyph (phone); the text when it has no glyph. */
  readonly short: string;
  /** The glyph before the term: an icon of the sprite, or the crown. */
  readonly icon?: IconName | 'crown';
  /** How it reads: a win, a gain or loss for the mover, or a warning. */
  readonly tone?: 'win' | 'gain' | 'loss' | 'warn';
}

/** What the outcome line needs besides the outcome itself. */
export interface OutcomeContext {
  /** The seat taking the action (the view's toMove). */
  readonly mover: Seat;
  /** Both seats' coins now, from the view. */
  readonly coins: readonly [number, number];
  /** The current age, for "who starts Age II". */
  readonly age: number;
  readonly families: ReadonlySet<HintFamily>;
  readonly names: HintNames;
}

/** The view's part of an OutcomeContext. */
export function outcomeContext(
  view: Pick<View, 'players' | 'age' | 'toMove' | 'active'>,
  families: ReadonlySet<HintFamily>,
  names: HintNames
): OutcomeContext {
  const mover: Seat = (view.toMove ?? view.active) === 1 ? 1 : 0;
  return {
    mover,
    coins: [view.players[0].coins, view.players[1].coins],
    age: view.age,
    families,
    names
  };
}

function isYou(seat: Seat, names: HintNames): boolean {
  return names.you === seat;
}

/** The terms of the game's end on points. */
function endTerm(o: OutcomeView, ctx: OutcomeContext): OutcomeTerm | undefined {
  const end = o.end;
  if (!end) return undefined;
  const me = ctx.mover;
  let text: string;
  if (end.shared || end.winner === undefined) {
    text = t('hints.outcome.endShared', { a: end.scores[me].total });
  } else {
    const winner: Seat = end.winner === 1 ? 1 : 0;
    const wa = end.scores[winner].total;
    const wb = end.scores[other(winner)].total;
    const name = ctx.names.names[winner];
    if (wa === wb) {
      text = isYou(winner, ctx.names)
        ? t('hints.outcome.endBlueYou', { a: wa })
        : t('hints.outcome.endBlueNamed', { name, a: wa });
    } else {
      text = isYou(winner, ctx.names)
        ? t('hints.outcome.endYou', { a: wa, b: wb })
        : t('hints.outcome.endNamed', { name, a: wa, b: wb });
    }
  }
  const won = end.winner === me;
  return { id: 'end', text, short: text, icon: 'vp-laurel', tone: won ? 'win' : 'warn' };
}

/** The win term: at once, through a follow-up, or in an extra turn. */
function winTerm(o: OutcomeView): OutcomeTerm | undefined {
  const w = o.wins;
  if (!w) return undefined;
  const kind = t(`hints.victory.${w.kind}`);
  let text: string;
  if (w.via === 'token:law') text = t('hints.outcome.winLaw');
  else if (w.via && w.steps >= 2) {
    text = t('hints.outcome.winTwoStep', { card: keyTargetName(w.via) ?? '' });
  } else if (w.via) text = t('hints.outcome.winVia', { card: keyTargetName(w.via) ?? '' });
  else text = t('hints.outcome.win', { kind });
  return { id: 'win', text, short: text, icon: 'crown', tone: 'win' };
}

/** The pawn and looting terms. */
function pawnTerms(p: PawnMoveView, ctx: OutcomeContext): OutcomeTerm[] {
  const out: OutcomeTerm[] = [];
  const me = ctx.mover;
  const opp = other(me);
  // Positive is toward seat 1's capital: the mover's own direction is
  // toward the other seat's capital.
  const toward = me === 0 ? p.to - p.from : p.from - p.to;
  if (toward !== 0) {
    const spaces = Math.abs(toward);
    const target: Seat = toward > 0 ? opp : me;
    const text = isYou(target, ctx.names)
      ? t('hints.outcome.pawnYou', { count: spaces })
      : t('hints.outcome.pawn', { count: spaces, name: ctx.names.names[target] });
    out.push({ id: 'pawn', text, short: signed(toward), icon: 'shield' });
  }
  const lost = p.loot.reduce((n, l) => n + l.lost, 0);
  if (p.loot.length > 0) {
    // A looting token on the mover's way is always the other seat's.
    const text = isYou(opp, ctx.names)
      ? t('hints.outcome.lootYou', { count: lost })
      : t('hints.outcome.loot', { name: ctx.names.names[opp], count: lost });
    out.push({
      id: 'loot',
      text,
      short: `${ctx.names.names[opp]} ${signed(-lost)}`,
      icon: 'coin-loss',
      tone: 'gain'
    });
  }
  return out;
}

/**
 * The terms of the outcome line under one move button, in a fixed order:
 * a win or the game's end first, then points, the pawn and its looting,
 * coins, symbols, the turn order and the choice the move opens. Terms that
 * say nothing (a zero change) are dropped.
 */
export function outcomeTerms(o: OutcomeView | undefined, ctx: OutcomeContext): OutcomeTerm[] {
  if (!o) return [];
  const f = ctx.families;
  const me = ctx.mover;
  const opp = other(me);
  const out: OutcomeTerm[] = [];
  // A civilian win is the game's end on points: with scoring on, the end
  // term says it (with the crown), so the win term would only repeat it.
  const endSaysWin = f.has('scoring') && o.end !== undefined && o.wins?.kind === 'civilian';
  if (f.has('threats') && !endSaysWin) {
    const w = winTerm(o);
    if (w) out.push(w);
  }
  if (f.has('scoring')) {
    const e = endTerm(o, ctx);
    if (e) out.push(endSaysWin ? { ...e, icon: 'crown', tone: 'win' } : e);
    if (o.score[me] !== 0) {
      out.push({
        id: 'vp',
        text: t('hints.outcome.vp', { delta: signed(o.score[me]) }),
        short: signed(o.score[me]),
        icon: 'vp-laurel',
        tone: o.score[me] > 0 ? 'gain' : 'loss'
      });
    }
    if (o.score[opp] !== 0) {
      const text = t('hints.outcome.vpNamed', {
        name: ctx.names.names[opp],
        delta: signed(o.score[opp])
      });
      out.push({ id: 'vpOpp', text, short: text, icon: 'vp-laurel' });
    }
  }
  if (f.has('military') && o.pawn) out.push(...pawnTerms(o.pawn, ctx));
  if (f.has('scoring')) {
    const mine = o.coins[me] - ctx.coins[me];
    if (mine !== 0) {
      out.push({
        id: 'coins',
        text: t('hints.outcome.coins', { count: o.coins[me], delta: signed(mine) }),
        short: `${formatNumber(o.coins[me])} (${signed(mine)})`,
        icon: 'coin',
        tone: mine > 0 ? 'gain' : undefined
      });
    }
    const theirs = o.coins[opp] - ctx.coins[opp];
    const looted = o.pawn?.loot.reduce((n, l) => n + l.lost, 0) ?? 0;
    // Looting alone is already said by the loot term.
    if (theirs !== 0 && !(f.has('military') && theirs === -looted)) {
      const text = t('hints.outcome.coinsNamed', {
        name: ctx.names.names[opp],
        count: o.coins[opp],
        delta: signed(theirs)
      });
      out.push({ id: 'coinsOpp', text, short: text, icon: 'coin' });
    }
  }
  if (f.has('science') && o.distinct !== undefined) {
    const text = t('hints.outcome.distinct', { count: o.distinct });
    out.push({ id: 'distinct', text, short: text });
  }
  if (f.has('tempo') && o.tempo) {
    const tp = o.tempo;
    if (tp.extraTurn === 'gain') {
      const text =
        tp.source === 'theology'
          ? t('hints.outcome.playAgainTheology')
          : t('hints.outcome.playAgain');
      out.push({ id: 'playAgain', text, short: text, tone: 'gain' });
    } else if (tp.extraTurn === 'lost') {
      const text = t('hints.outcome.extraLost');
      out.push({ id: 'extraLost', text, short: text, tone: 'warn' });
    }
    if (tp.starterChooser !== undefined) {
      const chooser: Seat = tp.starterChooser === 1 ? 1 : 0;
      const text = isYou(chooser, ctx.names)
        ? t('hints.outcome.starterYou', { age: ctx.age + 1 })
        : t('hints.outcome.starterNamed', { name: ctx.names.names[chooser], age: ctx.age + 1 });
      out.push({ id: 'starter', text, short: text });
    }
  }
  if (o.opens && out.length > 0) {
    const text = t(`hints.outcome.opens.${o.opens}`);
    out.push({ id: 'opens', text, short: text });
  }
  return out;
}

/** The whole line in words, for a screen reader or a tooltip. */
export function outcomeText(terms: readonly OutcomeTerm[]): string {
  return terms.map((x) => x.text).join(t('hints.outcome.separator'));
}

// --- The confirm step's warning (blunder-confirm) -------------------------

/**
 * The warning the confirm step adds to a move (owner Q3, A: only where the
 * player asked for a confirm step): the move hands the opponent a win, or
 * another move wins this turn and this one does not.
 */
export function blunderLine(
  choice: Pick<Choice, 'key' | 'outcome'> | undefined,
  choices: readonly Pick<Choice, 'key' | 'outcome'>[],
  ctx: Pick<OutcomeContext, 'mover' | 'families' | 'names'>
): string | undefined {
  if (!choice?.outcome || !ctx.families.has('threats')) return undefined;
  const o = choice.outcome;
  const opp = other(ctx.mover);
  if (o.gives && o.gives.length > 0) {
    return t('hints.blunder.gives', { name: ctx.names.names[opp], list: threatList(o.gives) });
  }
  if (o.wins) return undefined;
  const winners = [
    ...new Set(
      choices
        .filter((c) => c.outcome?.wins)
        .map((c) => keyTargetName(c.key))
        .filter((n): n is string => n !== undefined)
    )
  ];
  if (winners.length === 0) return undefined;
  const list = joinList(winners, 'disjunction');
  return isYou(ctx.mover, ctx.names)
    ? t('hints.blunder.missedYou', { list })
    : t('hints.blunder.missedNamed', { name: ctx.names.names[ctx.mover], list });
}

// --- After this pick -------------------------------------------------------

/** One move of the card sheet, for the "After this pick" panel. */
export interface PickAction {
  /** What the move is called in a line that differs between moves. */
  readonly label: string;
  readonly outcome?: OutcomeView;
}

/** One line of the "After this pick" panel. */
export interface AfterPickLine {
  /** A stable id (data-after-line). */
  readonly id: string;
  readonly text: string;
  /** A threat line: the panel opens by itself when it holds one. */
  readonly threat?: boolean;
}

/** The threat lines of one move: gifts, blocks, wins it sets up, and the reveal risk. */
function threatLines(o: OutcomeView | undefined, ctx: OutcomeContext): AfterPickLine[] {
  if (!o || !ctx.families.has('threats')) return [];
  const me = ctx.mover;
  const opp = other(me);
  const oppName = ctx.names.names[opp];
  const out: AfterPickLine[] = [];
  if (o.gives && o.gives.length > 0) {
    out.push({
      id: 'gives',
      text: t('hints.after.gives', { name: oppName, list: threatList(o.gives) }),
      threat: true
    });
  }
  if (o.blocks && o.blocks.length > 0) {
    out.push({
      id: 'blocks',
      text: t('hints.after.blocks', { name: oppName, list: threatList(o.blocks) }),
      threat: true
    });
  }
  if (o.threatens && o.threatens.length > 0) {
    const list = threatList(o.threatens);
    out.push({
      id: 'threatens',
      text: isYou(me, ctx.names)
        ? t('hints.after.threatensYou', { list, name: oppName })
        : t('hints.after.threatensNamed', { mover: ctx.names.names[me], list, name: oppName }),
      threat: true
    });
  }
  if (o.revealRisk) {
    const r = o.revealRisk;
    out.push({
      id: 'risk',
      text: t('hints.after.risk', { count: r.freed, num: r.num, den: r.den, name: oppName }),
      threat: true
    });
  }
  return out;
}

/**
 * The lines of the card sheet's "After this pick" panel for the card in
 * `slot`: what taking it uncovers (frees-preview), who sees a turned card
 * first (reveal-first-look), what a freed face-up card does for the
 * opponent (gift-to-opponent), and each move's threats one turn ahead
 * (gift-check, creates-threat, hidden-risk). A threat line that differs
 * between the moves is labelled with its move.
 */
export function afterPickLines(
  slot: SlotView,
  structure: readonly SlotView[],
  actions: readonly PickAction[],
  ctx: OutcomeContext
): AfterPickLine[] {
  const f = ctx.families;
  const me = ctx.mover;
  const opp = other(me);
  const oppName = ctx.names.names[opp];
  const out: AfterPickLine[] = [];
  if (f.has('tempo')) {
    const freed = freedSplit(slot.index, structure);
    const items = freed.faceUp.map((s) => cardName(s.card ?? ''));
    if (freed.faceDown > 0) items.push(t('hints.after.faceDown', { count: freed.faceDown }));
    const n = freed.faceUp.length + freed.faceDown;
    out.push(
      n === 0
        ? { id: 'frees', text: t('hints.after.freesNothing') }
        : {
            id: 'frees',
            text: t('hints.after.frees', { count: n, list: joinList(items, 'conjunction') })
          }
    );
    if (freed.faceDown > 0) {
      out.push({
        id: 'reveal',
        text: t('hints.after.reveal', { count: freed.faceDown, name: oppName })
      });
      if (actions.some((a) => a.outcome?.tempo?.extraTurn === 'gain')) {
        out.push({ id: 'revealWonder', text: t('hints.after.revealWonder') });
      }
    }
    for (const s of freed.faceUp) {
      const card = cardName(s.card ?? '');
      const h = s.hints?.[opp];
      const c = s.costs?.[opp];
      let text: string | undefined;
      if (f.has('threats') && h?.codes.includes('win')) {
        text = t('hints.after.giftWin', { card, name: oppName });
      } else if (f.has('science') && h?.codes.includes('pair')) {
        text = t('hints.after.giftPair', { card, name: oppName });
      } else if (f.has('prices') && c?.chainFree) {
        text = t('hints.after.giftChain', { card, name: oppName });
      } else if (f.has('prices') && c?.affordable) {
        text = t('hints.after.giftPays', { card, name: oppName, count: c.total });
      }
      if (text) out.push({ id: `gift-${s.index}`, text });
    }
  }
  // The threats one turn ahead: once when every move agrees, else by move.
  const per = actions
    .filter((a) => a.outcome)
    .map((a) => ({ label: a.label, lines: threatLines(a.outcome, ctx) }));
  const texts = per.map((p) => p.lines.map((l) => l.text).join('\n'));
  if (per.length > 0 && texts.every((x) => x === texts[0])) {
    out.push(...per[0]!.lines);
  } else {
    per.forEach((p, i) => {
      for (const l of p.lines) {
        out.push({
          ...l,
          id: `${l.id}-${i}`,
          text: t('hints.after.byMove', { move: p.label, line: l.text })
        });
      }
    });
  }
  return out;
}

// --- Starter and token choices ---------------------------------------------

/**
 * starter-threat: for each starter choice that names wins, the line "If
 * Moon starts, they can win at once with Arsenal", in seat order.
 */
export function starterThreatLines(
  choices: readonly Pick<Choice, 'kind' | 'starter' | 'outcome'>[],
  families: ReadonlySet<HintFamily>,
  names: HintNames
): { seat: Seat; text: string }[] {
  if (!families.has('threats')) return [];
  const out: { seat: Seat; text: string }[] = [];
  for (const c of choices) {
    const list = c.outcome?.threatens;
    if (c.kind !== 'starter' || !list || list.length === 0) continue;
    const seat: Seat = c.starter === 1 ? 1 : 0;
    out.push({
      seat,
      text: isYou(seat, names)
        ? t('hints.starter.threatYou', { list: threatList(list) })
        : t('hints.starter.threatNamed', { name: names.names[seat], list: threatList(list) })
    });
  }
  return out.sort((a, b) => a.seat - b.seat);
}

/** The cards a new age opens with: its accessible face-up cards (starter-preview). */
export function openingCards(structure: readonly SlotView[]): string[] {
  return structure
    .filter((s) => s.state === 'faceUp' && s.accessible && s.card !== undefined)
    .map((s) => s.card!);
}

// --- Selection ghosts (hud-ghosts, pawn-preview) ---------------------------

/** What a selected move would change on the board: the pawn, and each seat's coins and score after. */
export interface SelectionGhost {
  readonly pawn?: PawnMoveView;
  /** Coins after, by seat (scoring). */
  readonly coins?: readonly [number, number];
  /** Score preview totals after, by seat (scoring). */
  readonly score?: readonly [number, number];
}

/**
 * The ghost of a move with outcome `o`, drawn while it is selected: the
 * pawn's destination with the looting it takes (military), and both seats'
 * coins and score after (scoring). The score after is the view's total
 * plus the server's change.
 */
export function selectionGhost(
  o: OutcomeView | undefined,
  scoreNow: readonly [number, number],
  families: ReadonlySet<HintFamily>
): SelectionGhost | undefined {
  if (!o) return undefined;
  const pawn = families.has('military') ? o.pawn : undefined;
  const scoring = families.has('scoring');
  if (!pawn && !scoring) return undefined;
  return {
    ...(pawn ? { pawn } : {}),
    ...(scoring
      ? {
          coins: [o.coins[0], o.coins[1]] as const,
          score: [scoreNow[0] + o.score[0], scoreNow[1] + o.score[1]] as const
        }
      : {})
  };
}
