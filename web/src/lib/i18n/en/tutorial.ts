// English messages for the guided first game (M10.tutorial-client): the
// coach marks and notes (components/tutorial/), the lessons the server
// names by id (lib/tutorial/text.ts words them), the rails' reasons, Home's
// offer and the completion card. The tutorial's few keys on other screens
// (Home's row, the vs-AI sheet's link, the Settings reset) live here too,
// under those screens' prefixes, so that one file per language holds the
// whole tutorial. See ../index.ts for the conventions.
//
// Parameters: {card} and {wonder} are catalog names, {resource} a
// lower-case resource name, {name} the bot's display name, {n} and {total}
// the guided turn and the number of guided turns. Every lesson is plain
// and at most two sentences.

import type { Messages } from '../types';

export const tutorial = {
  // The coach mark's and the coach note's chrome.
  'tutorial.coach.title': 'Tutorial',
  'tutorial.coach.progress': 'Lesson {n} of {total}',
  'tutorial.coach.next': 'Next',
  'tutorial.coach.pause': 'Play on my own',
  'tutorial.coach.region': 'Tutorial step',
  'tutorial.coach.tapCard': 'Tap {card}.',
  'tutorial.coach.openWonders': 'Choose Build a wonder.',
  'tutorial.coach.takeWonder': 'Tap {wonder}, then Take.',
  'tutorial.coach.otherCard': 'This lesson takes {card}. Close this and tap it.',
  'tutorial.coach.otherWonder': 'This lesson takes {wonder}. Close this and tap it.',
  'tutorial.coach.back': 'This lesson builds {card} itself. Go back to the card.',

  // The rails: why a sheet's move buttons are locked, by the guided move.
  'tutorial.locked': 'Tutorial: this lesson builds {card}.',
  'tutorial.lockedDiscard': 'Tutorial: this lesson discards {card}.',
  'tutorial.lockedBuildWonder': 'Tutorial: this lesson builds {wonder} with {card}.',
  'tutorial.lockedWonder': 'Tutorial: this lesson takes {wonder}.',

  // Pause and resume.
  'tutorial.paused.toast': 'Tutorial paused. Resume it from the menu.',
  'tutorial.menu.pause': 'Pause tutorial',
  'tutorial.menu.resume': 'Resume tutorial',

  // The info lessons (Next ends them).
  'tutorial.lesson.draftIntro':
    'First, you and the {name} pick wonders: four are on offer in each round, and you take turns choosing.',
  'tutorial.lesson.draftRound2':
    'The last wonder of round 1 came to you. In round 2, the {name} picks first and you take the next two.',
  'tutorial.lesson.ageOne':
    'This is the card structure of Age I. You may take a lit card: face up and not covered by another.',
  'tutorial.lesson.turn':
    'Each turn you take one card. You build it, sell it for coins, or tuck it under a wonder to build that wonder.',
  'tutorial.lesson.reveal':
    'Taking a card uncovers the cards below it. {card} has just been uncovered, so it turned face up.',
  'tutorial.lesson.military':
    'Your shield moved the pawn one space toward the capital of the {name}. Reach a capital and you win at once.',
  'tutorial.lesson.coins':
    'Coins pay for cards and for trading. At the end of the game, every 3 coins are worth 1 point.',
  'tutorial.lesson.wonders':
    'These are your wonders. You build one by paying its cost and tucking a card from the structure under it.',
  'tutorial.lesson.tokens':
    'These are the progress tokens. You take one each time you make a pair of the same science symbol.',
  'tutorial.lesson.playOn':
    'That is the heart of the game. Finish Age I on your own: new things are explained as they come up.',
  'tutorial.lesson.leftPath':
    'You went your own way, so the guided lessons stop here. Keep playing, or Undo to go back to the lesson.',

  // The action lessons (the guided move ends them).
  'tutorial.action.draftPick': 'Take {wonder}. You build your wonders later, by paying their cost.',
  'tutorial.action.draftPickAgain': 'Now take {wonder} as well.',
  'tutorial.action.buildFree':
    '{card} costs nothing. Build it: from now on it makes {resource} for you every turn.',
  'tutorial.action.buildOwn':
    '{card} costs {resource}, which you make, so it costs no coins. The cost line shows it.',
  'tutorial.action.trade':
    'You make no {resource}, so you buy it from the bank. The cost line shows the price: 2 coins, plus 1 for each {resource} the {name} makes.',
  'tutorial.action.discard':
    'Sell {card} for coins: 2, plus 1 for each yellow card you own. It goes to the discard pile.',
  'tutorial.action.buildWonder':
    'Build {wonder} with {card}. The card goes face down under the wonder, and its own effect is lost.',
  'tutorial.action.chain':
    '{card} shows a chain symbol, drawn on the card. In a later age, the card with the same symbol is free for you.',
  'tutorial.action.science':
    '{card} shows a science symbol. A pair of the same symbol takes a progress token, and six different ones win at once.',

  // Home's first-launch offer.
  'tutorial.offer.title': 'New to the game?',
  'tutorial.offer.text': 'Learn by playing a guided game against the {name}.',
  'tutorial.offer.start': 'Play the tutorial',
  'tutorial.offer.decline': 'No thanks',

  // The completion card (Age II in play).
  'tutorial.done.title': 'Tutorial complete',
  'tutorial.done.text': 'Age II begins, and the rest is yours. There are three ways to win:',
  'tutorial.done.military': 'Military: push the pawn into the other capital.',
  'tutorial.done.science': 'Science: collect six different science symbols.',
  'tutorial.done.civilian': 'Civilian: have the most points after Age III.',
  'tutorial.done.keep': 'Keep playing',
  'tutorial.done.new': 'New game',
  'tutorial.done.home': 'Home',

  // The tutorial on other screens.
  'home.mode.tutorial': 'Tutorial',
  'home.tutorial': 'Learn to play',
  'home.tutorialHint': 'A guided first game against {name}',
  'home.tutorialDone': 'Tutorial done: play it again',
  'home.ai.tutorialLink': 'First game? Play the tutorial instead.',
  'hints.settings.tutorialReset': 'Show the tutorial offer again',
  'hints.settings.tutorialResetDone': 'Home will offer the tutorial again.'
} as const satisfies Messages;
