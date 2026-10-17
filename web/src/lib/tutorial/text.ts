// The words of a coach step (lib/tutorial/steps.ts) and of the rails'
// reason: the server sends lesson ids, slugs and keys only, and this
// module words them through t() (i18n/en/tutorial.ts). The {resource} of a
// lesson is read from the printed catalog card (what it produces or what
// it costs), never from the game: display only.

import { cardBySlug } from '$lib/catalog';
import { cardName, resourceName, wonderName } from '$lib/components/card/effect-text';
import { t } from '$lib/i18n';
import type { Resource, TutorialState } from '$lib/wire';
import { isLessonId, parseKey, type CoachStep, type LessonId } from './steps';

export interface CoachText {
  /** "Tutorial". */
  readonly title: string;
  /** "Lesson 3 of 10", or null outside the guided part. */
  readonly progress: string | null;
  /** The step's text; empty for a lesson id this client does not know. */
  readonly text: string;
}

const COST_ORDER: readonly Resource[] = ['wood', 'clay', 'stone', 'glass', 'papyrus'];

/** The resource a card produces (buildFree), else the first it costs. */
function resourceOf(card: string | undefined, produced: boolean): string {
  const c = card ? cardBySlug(card) : undefined;
  if (!c) return '';
  if (produced) {
    const made = c.effects.find((e) => e.kind === 'produce' && e.resource)?.resource;
    if (made) return resourceName(made);
  }
  const cost = COST_ORDER.find((r) => c.cost[r] > 0);
  return cost ? resourceName(cost) : '';
}

/** The text of a lesson, by its id. */
function lessonText(
  id: LessonId,
  info: boolean,
  params: { card: string; wonder: string; name: string; resource: string }
): string {
  const { card, wonder, name } = params;
  if (info) {
    switch (id) {
      case 'draftIntro':
        return t('tutorial.lesson.draftIntro', { name });
      case 'draftRound2':
        return t('tutorial.lesson.draftRound2', { name });
      case 'ageOne':
        return t('tutorial.lesson.ageOne');
      case 'turn':
        return t('tutorial.lesson.turn');
      case 'reveal':
        return t('tutorial.lesson.reveal', { card });
      case 'military':
        return t('tutorial.lesson.military', { name });
      case 'coins':
        return t('tutorial.lesson.coins');
      case 'wonders':
        return t('tutorial.lesson.wonders');
      case 'tokens':
        return t('tutorial.lesson.tokens');
      case 'playOn':
        return t('tutorial.lesson.playOn');
      case 'leftPath':
        return t('tutorial.lesson.leftPath');
      default:
        return '';
    }
  }
  switch (id) {
    case 'draftPick':
      return t('tutorial.action.draftPick', { wonder });
    case 'draftPickAgain':
      return t('tutorial.action.draftPickAgain', { wonder });
    case 'buildFree':
      return t('tutorial.action.buildFree', { card, resource: params.resource });
    case 'buildOwn':
      return t('tutorial.action.buildOwn', { card, resource: params.resource });
    case 'trade':
      return t('tutorial.action.trade', { resource: params.resource, name });
    case 'discard':
      return t('tutorial.action.discard', { card });
    case 'buildWonder':
      return t('tutorial.action.buildWonder', { wonder, card });
    case 'chain':
      return t('tutorial.action.chain', { card });
    case 'science':
      return t('tutorial.action.science', { card });
    default:
      return '';
  }
}

/**
 * The words of a step. `name` is the bot's display name (the Apprentice).
 * An unknown lesson id has no text, only the chrome (and Next on an info
 * step); a development build says so in the console.
 */
export function coachText(step: CoachStep, name: string): CoachText {
  const card = step.card ? cardName(step.card) : '';
  const wonder = step.wonder ? wonderName(step.wonder) : '';
  const progress =
    step.turn > 0 && step.turns > 0
      ? t('tutorial.coach.progress', { n: step.turn, total: step.turns })
      : null;
  const title = t('tutorial.coach.title');
  let text = '';
  switch (step.words) {
    case 'tapCard':
      text = t('tutorial.coach.tapCard', { card });
      break;
    case 'openWonders':
      text = t('tutorial.coach.openWonders');
      break;
    case 'takeWonder':
      text = t('tutorial.coach.takeWonder', { wonder });
      break;
    case 'otherCard':
      text = t('tutorial.coach.otherCard', { card });
      break;
    case 'otherWonder':
      text = t('tutorial.coach.otherWonder', { wonder });
      break;
    case 'back':
      text = t('tutorial.coach.back', { card });
      break;
    case 'lesson':
      if (isLessonId(step.id)) {
        const resource = resourceOf(step.card, step.id === 'buildFree');
        text = lessonText(step.id, step.lesson === 'info', { card, wonder, name, resource });
      }
      if (!text && import.meta.env.DEV) {
        console.warn(`tutorial: no text for lesson ${step.id}`);
      }
      break;
  }
  return { title, progress, text };
}

/** The rails' reason on a locked move button: the guided move's card or
 *  wonder (a draft pick names its wonder). */
export function lockedReason(tutorial: TutorialState | undefined): string {
  const key = tutorial?.action?.key ?? '';
  const parts = parseKey(key);
  const card = tutorial?.action?.card ?? parts.card;
  const wonder = tutorial?.action?.wonder ?? parts.wonder;
  if (parts.verb === 'draft' && wonder) {
    return t('tutorial.lockedWonder', { wonder: wonderName(wonder) });
  }
  const name = card ? cardName(card) : '';
  if (parts.verb === 'wonder' && wonder) {
    return t('tutorial.lockedBuildWonder', { wonder: wonderName(wonder), card: name });
  }
  if (parts.verb === 'discard') return t('tutorial.lockedDiscard', { card: name });
  return t('tutorial.locked', { card: name });
}
