// English messages for the pending-choice panels (destroy, build from the
// discard pile, progress tokens, the Great Library, the next age's starter),
// the end-of-age interstitial, the wonder draft and the game-over screen
// (components/choices/*, components/screens/{Draft,AgeInterstitial,
// GameOver,ScoreTable}.svelte, components/sheets/{Discard,Token}Sheet.svelte).
// See ../index.ts for the conventions. {name} is a player's display name,
// interpolated as data; card, wonder and token names come from card.ts and
// board.ts.

import type { Messages } from '../types';

export const choices = {
  // Shared by every choice panel.
  'choices.peek': 'Peek at the board',
  'choices.reopen': 'Open',
  'choices.selectCard': 'Choose a card first',
  'choices.selectToken': 'Choose a token first',
  'choices.waiting.destroyCard': 'Waiting for {name} to choose a card to destroy…',
  'choices.waiting.buildFromDiscard': 'Waiting for {name} to choose a card from the discard pile…',
  'choices.waiting.pickBoardToken': 'Waiting for {name} to choose a progress token…',
  'choices.waiting.pickLibraryToken': {
    one: '{name} is choosing from {count} hidden token of the Great Library…',
    other: '{name} is choosing from {count} hidden tokens of the Great Library…'
  },
  'choices.waiting.chooseStarter': 'Waiting for {name} to choose who starts Age {age}…',

  // Destroy (the Statue of Zeus, Circus Maximus).
  'choices.destroy.title': 'Destroy a card',
  'choices.destroy.intro': "Choose one of {name}'s {color} cards. It goes to the discard pile.",
  'choices.destroy.confirm': 'Destroy {card}',

  // Build from the discard pile (the Mausoleum) and the discard pile sheet.
  'choices.discard.title': 'Build from the discard pile',
  'choices.discard.intro': 'Choose a discarded card to build for free.',
  'choices.discard.confirm': 'Build {card} for free',
  'choices.discard.sheetTitle': 'Discard pile',
  'choices.discard.order': 'Oldest first.',
  'choices.discard.filter': 'Show cards by colour',
  'choices.discard.all': 'All',
  'choices.discard.count': {
    '=0': 'No cards',
    one: '{count} card',
    other: '{count} cards'
  },
  'choices.discard.empty': 'The discard pile is empty.',

  // Progress tokens: a science pair, and the Great Library.
  'choices.token.title': 'Choose a progress token',
  'choices.token.intro':
    'A pair of matching science symbols earns one progress token from the board.',
  'choices.token.confirm': 'Take {token}',
  'choices.token.list': 'Progress tokens to choose from',
  'choices.library.title': 'The Great Library',
  'choices.library.intro':
    'Three tokens drawn from the box. Keep one; the other two go back unseen.',
  'choices.library.secret': 'Only you can see these tokens.',

  // Who starts the next age.
  'choices.starter.title': 'Who starts Age {age}?',
  'choices.starter.whyBehind': '{name} is behind on the military track, so {name} chooses.',
  'choices.starter.whyBehindYou': 'You are behind on the military track, so you choose.',
  'choices.starter.whyEven': 'The military track is even, so {name}, who played last, chooses.',
  'choices.starter.whyEvenYou': 'The military track is even, so you choose, having played last.',
  'choices.starter.whyOther': '{name} chooses who starts.',
  'choices.starter.whyOtherYou': 'You choose who starts.',
  'choices.starter.hint': 'The starting player takes the first card of the new age.',
  'choices.starter.option': '{name} starts',
  'choices.starter.optionYou': 'I start',

  // The end-of-age interstitial.
  'choices.age.complete': 'Age {age} complete',
  'choices.age.summary': 'Both players after Age {age}',
  'choices.age.coins': 'Coins',
  'choices.age.points': 'Points',
  'choices.age.you': '(you)',
  'choices.age.militaryLead': '{name} leads the military by {margin}.',
  'choices.age.militaryLeadYou': 'You lead the military by {margin}.',
  'choices.age.militaryEven': 'The military track is even.',
  'choices.age.starts': '{name} starts Age {age}.',
  'choices.age.startsYou': 'You start Age {age}.',
  'choices.age.continue': 'Continue',
  'choices.age.begin': 'Begin Age {age}',
  'choices.age.summaryOpen': 'Summary',

  // The wonder draft.
  'choices.draft.title': 'Wonder draft',
  'choices.draft.round': 'Round {round} of 2',
  'choices.draft.order': 'Pick order',
  'choices.draft.pick': 'Pick {n}: {name}',
  'choices.draft.pickDone': 'Pick {n}: {name}, done',
  'choices.draft.pickNow': 'Pick {n}: {name}, now',
  'choices.draft.pickAuto': 'Pick {n}: {name} gets the last wonder',
  'choices.draft.yourPick': 'Your pick: choose a wonder',
  'choices.draft.namePick': "{name}'s pick: choose a wonder",
  'choices.draft.waiting': 'Waiting for {name} to pick…',
  'choices.draft.offer': 'Wonders on offer',
  'choices.draft.take': 'Take {wonder}',
  'choices.draft.cost': 'Cost: {cost}',
  'choices.draft.picked': "{name}'s wonders",
  'choices.draft.emptySlot': 'Not picked yet',

  // The game-over screen.
  'choices.over.kind.civilian': 'Civilian victory',
  'choices.over.kind.military': 'Military supremacy',
  'choices.over.kind.science': 'Scientific supremacy',
  'choices.over.kind.shared': 'Shared victory',
  'choices.over.kind.forfeit': 'Victory by forfeit',
  'choices.over.wins': '{name} wins',
  'choices.over.youWin': 'You win',
  'choices.over.bothWin': 'Both players win',
  'choices.over.reason.civilian': 'The most points after Age III.',
  'choices.over.reason.military': "{name}'s armies reached the rival capital.",
  'choices.over.reason.science': '{name} gathered six different science symbols.',
  'choices.over.reason.shared': 'Equal points, and equal points from blue cards.',
  'choices.over.reason.forfeit': '{name} forfeited.',
  'choices.over.tieBreak': 'Tied on points, so the most points from blue cards wins.',
  'choices.over.rematch': 'Rematch',
  'choices.over.viewBoard': 'View final board',
  'choices.over.home': 'Home',
  'choices.over.results': 'Results',
  'choices.over.bar': 'Game over',

  // The score table.
  'choices.score.caption': 'Final score',
  'choices.score.category': 'Category',
  'choices.score.blue': 'Blue cards',
  'choices.score.green': 'Green cards',
  'choices.score.yellow': 'Yellow cards',
  'choices.score.purple': 'Guilds',
  'choices.score.wonders': 'Wonders',
  'choices.score.tokens': 'Progress tokens',
  'choices.score.coins': 'Coins',
  'choices.score.military': 'Military',
  'choices.score.total': 'Total',
  'choices.score.winner': 'Winner',

  // The /dev/gallery section for this area.
  'dev.gallery.choices': 'Draft, choices and game over'
} as const satisfies Messages;
