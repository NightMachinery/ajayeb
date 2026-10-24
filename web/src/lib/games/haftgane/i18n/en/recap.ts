// The reveal's and the turn recap's messages (games/haftgane/table/
// recap-text.ts, RevealTag, CoinDelta, TurnRecap, RecapLayer): what each
// seat did, the coins that moved, the seat and role changes, and where
// hands went, as phone-layout.md section 4 words them.

import type { Messages } from '$lib/i18n';

export const recap = {
  'hg.recap.title': 'The last turn',
  'hg.recap.title.turn': 'Turn {turn}, Age {age}',
  'hg.recap.title.ageEnd': 'Last turn of Age {age}',
  'hg.recap.title.sides': 'Wonder sides',
  'hg.recap.close': 'Tap anywhere to close',
  'hg.recap.open': 'Show the last turn',
  'hg.recap.open.unseen': 'Show the last turn, not seen yet',
  'hg.recap.none': 'No turn has been played yet.',

  'hg.recap.build': '{name} built {card}',
  'hg.recap.you.build': 'You built {card}',
  'hg.recap.buildDiscarded': '{name} built {card} from the discard pile',
  'hg.recap.you.buildDiscarded': 'You built {card} from the discard pile',
  'hg.recap.stage': '{name} built wonder stage {stage}',
  'hg.recap.you.stage': 'You built wonder stage {stage}',
  'hg.recap.you.stageWith': 'You built wonder stage {stage} with {card}',
  'hg.recap.sell': '{name} sold a card',
  'hg.recap.you.sell': 'You sold {card}',
  'hg.recap.you.sellHidden': 'You sold a card',
  'hg.recap.discardNothing': '{name} found nothing to build in the discard pile',
  'hg.recap.you.discardNothing': 'You found nothing to build in the discard pile',
  'hg.recap.side': '{name} chose the {side} side',
  'hg.recap.you.side': 'You chose the {side} side',
  'hg.recap.idle': '{name} was paid for trade',
  'hg.recap.you.idle': 'You were paid for trade',
  'hg.recap.kept': '{name} keeps the last card for an extra turn',
  'hg.recap.you.kept': 'You keep {card} for an extra turn',
  'hg.recap.you.leftover': 'Your last card, {card}, went to the discard pile',

  'hg.recap.tag.stage': 'Stage {stage}',
  'hg.recap.tag.sold': 'Sold',
  'hg.recap.tag.soldFor': 'Sold {coins}',
  'hg.recap.tag.nothing': 'Nothing',

  'hg.recap.coin.bank': {
    one: 'paid {count} coin to the bank',
    other: 'paid {count} coins to the bank'
  },
  'hg.recap.coin.paid': { one: 'paid {name} {count} coin', other: 'paid {name} {count} coins' },
  'hg.recap.coin.received': {
    one: 'received {count} coin from {name}',
    other: 'received {count} coins from {name}'
  },
  'hg.recap.coin.sale': {
    one: 'took {count} coin for the sale',
    other: 'took {count} coins for the sale'
  },
  'hg.recap.coin.card': {
    one: 'took {count} coin from {card}',
    other: 'took {count} coins from {card}'
  },
  'hg.recap.coin.stage': {
    one: 'took {count} coin from wonder stage {stage}',
    other: 'took {count} coins from wonder stage {stage}'
  },

  'hg.recap.pass.left': 'Hands pass left.',
  'hg.recap.pass.right': 'Hands pass right.',
  'hg.recap.pass.hold': {
    one: 'You now hold the {count} card {name} passed you.',
    other: 'You now hold the {count} cards {name} passed you.'
  },
  'hg.recap.ageEnd': 'Age {age} ends: the military conflicts follow.',
  'hg.recap.ageStart': 'Age {age} begins with new hands.',
  'hg.recap.gameOver': 'The game is over: the scores follow.',

  'hg.recap.change.seated': '{by} seated {user} in seat {seat}',
  'hg.recap.change.toBot': '{by} gave seat {seat} to a bot',
  'hg.recap.change.resign': '{by} resigned, and a bot plays seat {seat}',
  'hg.recap.change.promote': '{by} made {user} an admin',
  'hg.recap.change.demote': '{by} removed {user} as admin',
  'hg.recap.change.stepDown': '{user} stopped being an admin',
  'hg.recap.change.temporary': '{user} is temporary admin while the admins are away',
  'hg.recap.someone': 'A player who left'
} as const satisfies Messages;
