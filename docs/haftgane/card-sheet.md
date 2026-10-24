# Card sheet and payments

Tapping a hand card opens its sheet. It shows the printed catalog facts
(card face, category, Age, effects and chains) beside the server's choices.
A chain already present in the player's city is labelled as owned. A free
build names the server's reason and omits the ordinary payment list. A
blocked build shows only the supplied reason: already built, a coin
shortfall with the cheapest quote, or a cost the available production cannot
cover. The screen does not infer a particular missing resource.

Each ordinary build and wonder-stage payment is one radio option, identified
by its offered key. The cheapest server total is selected initially, keeping
server order for ties. A payment witness supplies the resources made by the
player, those bought from each neighbour and their unit prices. The row
shows each neighbour's crest and name, the server's left/right subtotals,
printed bank payment and total. A matching producer and resource in the
neighbour view supplies the discount source. Neither the sheet nor its
payment list calculates a quote, affordability or legal action.

Selecting a radio saves a private, unconfirmed draft through TableSession.
Closing the sheet keeps that draft. Build, Stage and Sell confirm their own
selected offered key. They use the existing two-tap confirmation control
when Confirm moves is Always or Online only; Never sends on the first tap.
Changing the selected payment resets an armed button. Stage payments use
the same radio list, and the sale displays the server's gain. The footer
stays visible while the details scroll. Short landscape and desktop place
the card and choices in separate columns, with the actions in one row.

After confirming, the hand keeps its pick ribbon and the turn bar names the
picked action. Change withdraws confirmation and reopens the card sheet.
Tapping a hand card while confirmed also withdraws before editing. A reload
or reconnect restores the server's pending pick. If the final confirmation
has already resolved the point, the existing Session refusal notice applies.
Wonder-side and discard-pile decisions retain their separate choice panel.

## Development examples and checks

The development gallery contains CardSheet and PaymentList with plain props.
For a focused capture, open `/dev/gallery?hgSheet=trade&seats=5`. The other
variants are `stage`, `sale`, `chain`, `free` and `blocked`; the seat counts
are 3, 5 and 7. `confirm=off` exercises single-tap confirmation. These
examples use seedless ViewFor captures from legal games, filtered to the
selected card's choices and disabled entries, the viewer's city and its two
neighbour trade views. The names are neutral seat labels.

The captures in `web/src/lib/dev/gallery/haftgane-card-cases.json` came from
random-side games, using server Legal keys at each point. In each game the
capture pass preferred the cheapest offered building, except seeds divisible
by three preferred an offered stage during the first four points. The first
matching case was retained, scanning each required seat in order:

- Three seats: blocked seed 1, point 0, seat 0, Stockade; trade seed 1,
  point 7, seat 2, Temple; chain seed 1, point 6, seat 0, Aqueduct; free
  seed 9, point 6, seat 0, Glassworks (Age II).
- Five seats: blocked seed 1, point 0, seat 1, Stockade; trade seed 1,
  point 4, seat 2, Baths; chain seed 1, point 6, seat 0, Courthouse; free
  seed 12, point 6, seat 3, Sawmill.
- Seven seats: blocked seed 1, point 0, seat 0, Baths; trade seed 1,
  point 4, seat 3, Apothecary; chain seed 1, point 6, seat 0, Library;
  free seed 7, point 13, seat 5, Castrum.

Points are zero-based resolved batches. Stage and sale examples reuse the
trade capture and select its offered stage or sale. Captures do not mutate
engine state or invent payment witnesses.

`e2e/ux/haftgane-card-sheet.spec.ts` captures all six variants at all three
seat counts, four viewports, both themes and both directions. It checks
horizontal overflow, 44 px controls, tooltip anchors, on-screen footer,
readable title width, blocked-message text contrast of at least 4.5:1,
Animation off, console errors and failed requests. The confirmation test
checks the two-tap guard and resets it by changing payments.
`e2e/specs/haftgane-card-sheet.full.spec.ts` plays several turns at five
seats through real sheet controls, including a non-default payment, a stage,
a sale and Change. Every received frame still passes the independent
pending-ownership oracle. The existing reload, reconnect and foreign-pending
canaries remain in the table specs. The platform baseline test measures
Home and Duel separately so a shared-runtime dependency cannot hide in
Haftgane's lazy entry.
