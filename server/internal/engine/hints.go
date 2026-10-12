package engine

import (
	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
	"github.com/NightMachinery/ajayeb/server/internal/view"
)

// This file is the hints engine (the hints design, docs/dev/decisions.md
// D23): the facts about a game that the server works out from public
// information and sends in the view, so the client never derives a rule.
//
// # The pieces
//
// ComputeHints computes the shared hints of one state (those that are the
// same for every viewer) once per broadcast, for the families the game's
// hint settings switch on. ViewWithHints (view.go) builds one viewer's view
// and adds them, plus, later, the hints about the decider's own choices.
// ViewFor is ViewWithHints without hints, so the simulator, the tests and
// replays never see any.
//
// The hints are built from a few copy-only simulations, the helpers below:
//
//   - publicPlace: where a card is, as far as both players can know;
//   - previewBuild and previewWonder: what building a card or a wonder
//     would do for a seat, on a copy, without touching the structure;
//   - priceShift and savesInView: how a card would move trade prices and
//     the payments of the cards and wonders in view;
//   - tokenPreview: what taking a progress token would do for a seat;
//   - extraTurnOutcome: whether an action keeps the turn, from the rules
//     alone (no Apply);
//   - WinsNow: whether an action wins at once or through the mover's own
//     follow-up, which the Greedy bot shares.
//
// # Public information only
//
// A hint for viewer v may depend only on what v may see: Observe(s, v)
// hides the rest, and a hint must come out the same on every state
// Determinize can draw from that observation. So a simulation never reads
// a face-down card, the deals, the removed cards, the box guilds, the box
// order or a Great Library offer it may not see. In practice:
//
//   - previewBuild, previewWonder, priceShift and tokenPreview never take a
//     card out of the structure and never finish a turn, so nothing is
//     revealed, no age is dealt and the Great Library never draws (the
//     preview empties the box on its copy first);
//   - every simulation runs on a normalised copy whose Great Library offer
//     and destroy colour are cleared, so the decider's secret offer cannot
//     reach a shared hint;
//   - a nested simulation, a follow-up after a simulated action, only uses
//     cards from the face-up mask: the slots that were face up before the
//     first simulated action (FaceUpSlots), so a card a simulation turns
//     over is never used; and WinsNow stops before any follow-up once the
//     action has revealed a card, dealt an age or drawn the Great Library
//     (sameHidden).
//
// The leak guard in hints_leak_test.go checks the first rule on whole views
// for random states, and FuzzHintsLeak does the same for the nightly fuzz
// runner.
//
// # Cost
//
// The shared hints are a few dozen previews per broadcast. The decider's
// hints (a later slice) Apply each legal action and some follow-ups; they
// run under hintApplyBudget, a count of Applies and not a clock, so a hint
// is a pure function of the state and the settings, and what the budget
// cuts is left out, never guessed.

// SlotMask is a set of structure slots: bit i is slot i, as in Occupied and
// Accessible.
type SlotMask = uint32

// hintApplyBudget is the most Applies the hints of one view may spend. When
// it runs out, the remaining nested checks are omitted.
const hintApplyBudget = 20000

// Hints are the shared hints of one state: what every viewer's view gets,
// computed once (ComputeHints) and passed to ViewWithHints for each
// viewer. It is a value; the views copy it and never change it.
type Hints struct {
	settings view.HintSettings
	board    view.HintsView
}

// Settings returns the hint settings the hints were computed with.
func (h *Hints) Settings() view.HintSettings { return h.settings }

// ComputeHints computes the shared hints of s for the families f switches
// on. A game that is over has none: its views carry no hint at all.
func ComputeHints(s *State, f view.HintSettings) Hints {
	return computeHints(s, f, nil)
}

// computeHints is ComputeHints with a simulation context, which the leak
// tests pass to watch every simulation the hints run.
func computeHints(s *State, f view.HintSettings, _ *simCtx) Hints {
	h := Hints{settings: f}
	if s.Over() {
		return h
	}
	// The slices after M10.HS1 fill h.board here, each part under its
	// family's switch (f.Science, f.Military and so on).
	return h
}

// addHints copies the shared hints h into out, the view of s for v. The
// caller has checked that h is not nil and that the game is not over.
func addHints(_ *State, _ Viewer, h *Hints, out *view.View) {
	board := h.board
	out.Hints = &board
	// The decider's hints (Choice.outcome, under a fresh hintApplyBudget)
	// come with a later slice.
}

// simCtx counts the Applies of one hint computation against a budget and,
// in tests, reports every simulation. The nil context has no budget and
// reports nothing.
type simCtx struct {
	budget  int // Applies allowed; 0 means no limit
	applies int // Applies spent
	trace   func(simNote)
}

// simNote describes one simulated action, for the leak tests. It holds a
// copy of the state, so that no simulation's scratch state escapes to the
// heap through the trace.
type simNote struct {
	State  State    // the state the action is applied to
	For    Seat     // the seat the simulation is for (the mover)
	Action Action   // the action simulated
	Depth  int      // 0 for the action asked about, 1 for a follow-up
	Mask   SlotMask // the face-up mask the simulation is limited to
}

// apply spends one Apply of the budget on action a of the seat to move in
// s, at the given depth, and reports it to the trace. It returns false,
// and the caller must not simulate, when the budget is spent.
func (c *simCtx) apply(s *State, a Action, depth int, mask SlotMask) bool {
	if c == nil {
		return true
	}
	if c.budget > 0 && c.applies >= c.budget {
		return false
	}
	c.applies++
	if c.trace != nil {
		c.trace(simNote{State: *s, For: s.ToMove, Action: a, Depth: depth, Mask: mask})
	}
	return true
}

// FaceUpSlots returns the face-up mask of s: the slots that hold a face-up
// card.
func FaceUpSlots(s *State) SlotMask {
	var m SlotMask
	for i := range s.Slots {
		if s.Slots[i].Card != 0 && s.Slots[i].FaceUp {
			m |= 1 << i
		}
	}
	return m
}

// normalised returns a copy of s set up for seat to act in a play turn of
// its own: PhasePlay with seat active and to move, no extra turn, and no
// open choice (the destroy colour and the Great Library offer cleared, so
// a secret offer never reaches a simulation). It asks "what could seat do
// if it were its turn now".
func normalised(s *State, seat Seat) State {
	x := *s
	x.Phase = PhasePlay
	x.Active, x.ToMove = seat, seat
	x.ExtraTurn = false
	x.Destroy = catalog.NoColor
	x.Library = [LibraryDraw]catalog.TokenID{}
	return x
}

// PublicPlace is where a card is, as far as both players can know. The
// view's names are view.PublicPlace.
type PublicPlace uint8

// The public places.
const (
	PlaceFaceUp         PublicPlace = iota // face up in a structure slot
	PlaceCity                              // in a player's city
	PlaceTucked                            // under a built wonder
	PlaceDiscard                           // in the discard pile
	PlaceUnseenThisAge                     // a card of the current age not seen: face down or set aside, unknown which
	PlaceUnseenLaterAge                    // a card of an age still to come: dealt or set aside, unknown which
	PlaceRemovedKnown                      // a card of an age that is over and never seen, so certainly set aside (or a scenario's card out of play)
	numPublicPlaces
)

var publicPlaceNames = [numPublicPlaces]string{
	"faceUp", "city", "tucked", "discard", "unseenThisAge", "unseenLaterAge", "removedKnown",
}

// String returns the wire name, for example "unseenThisAge".
func (p PublicPlace) String() string { return enumString(publicPlaceNames[:], uint8(p), "place") }

// cardPlace is a card's public place: the place, and the seat (for a city
// or a tucked card) or the slot (for a face-up card) it is in.
type cardPlace struct {
	Place PublicPlace
	Seat  Seat // PlaceCity and PlaceTucked
	Slot  int  // PlaceFaceUp; -1 otherwise
}

// publicPlace returns where card c is, as far as both players can know.
// It reads only public parts of s: the cities, the tucked cards, the
// discard pile, the face-up slots and the age. A card in none of them is
// unseen: of the current age, of an age to come, or of an age that is over
// (removedKnown, which only a past age can give). A face-down card is
// never told apart from a set-aside one. This is the one place that
// decides "unseen" or "removed"; every hint that counts cards uses it.
func publicPlace(s *State, c catalog.CardID) cardPlace {
	for seat := range Seat(2) {
		p := &s.Players[seat]
		if p.City.Has(c) {
			return cardPlace{Place: PlaceCity, Seat: seat, Slot: -1}
		}
		for k := range p.Wonders {
			if p.Wonders[k].Built && p.Wonders[k].Tucked == c {
				return cardPlace{Place: PlaceTucked, Seat: seat, Slot: -1}
			}
		}
	}
	if s.Discard.Has(c) {
		return cardPlace{Place: PlaceDiscard, Slot: -1}
	}
	if s.Age >= 1 && s.Age <= NumAges {
		for i := range s.Slots {
			if sl := &s.Slots[i]; sl.FaceUp && sl.Card == c {
				return cardPlace{Place: PlaceFaceUp, Slot: i}
			}
		}
	}
	if s.OutOfPlay.Has(c) {
		return cardPlace{Place: PlaceRemovedKnown, Slot: -1}
	}
	age := uint8(0)
	if int(c) < len(catalog.T.Age) {
		age = catalog.T.Age[c]
	}
	switch {
	case age > s.Age:
		return cardPlace{Place: PlaceUnseenLaterAge, Slot: -1}
	case age == s.Age:
		return cardPlace{Place: PlaceUnseenThisAge, Slot: -1}
	}
	return cardPlace{Place: PlaceRemovedKnown, Slot: -1}
}

// ScienceEffectKind is what a green card does to its builder's symbols.
// The view's names are view.ScienceEffectKind.
type ScienceEffectKind uint8

// The science effect kinds.
const (
	ScienceEffectNew  ScienceEffectKind = iota // a symbol the seat did not have
	ScienceEffectPair                          // the second card of a symbol
	numScienceEffectKinds
)

var scienceEffectNames = [numScienceEffectKinds]string{"new", "pair"}

// String returns the wire name.
func (k ScienceEffectKind) String() string {
	return enumString(scienceEffectNames[:], uint8(k), "scienceEffect")
}

// scienceEffect is what building a green card does to the builder's
// symbols.
type scienceEffect struct {
	Symbol        catalog.Symbol
	Kind          ScienceEffectKind
	DistinctAfter int  // the builder's different symbols afterwards
	TokenOnBoard  bool // a pair opens the board token choice (the board has a token)
}

// lootTaken is one looting token a pawn move takes.
type lootTaken struct {
	At   int8  // its space: -6, -3, 3 or 6
	Lost int16 // coins the victim actually lost
}

// pawnMove is how a simulated action moves the conflict pawn.
type pawnMove struct {
	From, To int8
	Loot     [2]lootTaken // the tokens taken, in the order the pawn reached them
	NLoot    int
}

// readPawnMove reads the pawn move and the looting of an action's events
// (EvPawn, then EvLoot for each token taken), and whether the pawn moved.
// before is the looting tokens on the track before the action; the tokens
// a move takes are on one side, nearer first, the order lootTokens lists
// them in.
func readPawnMove(ev []Event, before uint8) (pawnMove, bool) {
	var m pawnMove
	moved := false
	for _, e := range ev {
		switch e.Kind {
		case EvPawn:
			m.From, m.To = int8(e.A), int8(e.B)
			moved = true
		case EvLoot:
			for i := range lootTokens {
				t := &lootTokens[i]
				if t.victim == e.Seat && before&t.bit != 0 && m.NLoot < len(m.Loot) {
					m.Loot[m.NLoot] = lootTaken{At: t.at, Lost: e.A}
					m.NLoot++
					before &^= t.bit
					break
				}
			}
		}
	}
	return m, moved
}

// Win is a win an action would bring its mover.
type Win struct {
	Kind VictoryKind
	// Steps is 1 for a win in this turn, at once or through a choice of the
	// mover's own that the action opens, and 2 for a win in the mover's
	// extra turn.
	Steps int
	// Via is the follow-up that wins, or the zero Action for a win at once.
	Via Action
}

// wonBy reports whether x is a finished game won by seat.
func wonBy(x *State, seat Seat) bool {
	return x.Over() && x.Result.Winner == int8(seat)
}

// WinsNow reports whether action a, legal for the seat to move in s, wins
// the game for that seat whatever the hidden information: it ends the game
// with the mover's victory at once (a supremacy, or a civilian win on Age
// III's last card), or it leaves the mover to decide again, through a
// choice of its own or its extra turn, with nothing hidden changed (no card
// revealed, no age dealt, no Great Library draw; see sameHidden), and some
// follow-up wins at once. A play action, and every follow-up, may only use
// a card from mask (normally FaceUpSlots(s)), so a card a simulation
// reveals is never used.
//
// Such a win depends only on public information, so any one
// determinization of the mover's observation proves it. The Greedy bot
// plays a win it finds this way without searching, and the hints show it.
func WinsNow(s *State, a Action, mask SlotMask) (Win, bool) {
	return winsNow(s, a, mask, nil)
}

// winsNow is WinsNow under a simulation context: it stops, reporting no
// win, when the context's budget runs out.
func winsNow(s *State, a Action, mask SlotMask, c *simCtx) (Win, bool) {
	mover := s.ToMove
	if !actionInMask(s, a, mask) || !c.apply(s, a, 0, mask) {
		return Win{}, false
	}
	x := *s
	ApplyTrusted(&x, a)
	if wonBy(&x, mover) {
		return Win{Kind: x.Result.Kind, Steps: 1}, true
	}
	if x.Over() || x.ToMove != mover || !sameHidden(s, &x) {
		return Win{}, false
	}
	steps := 1
	if x.Phase == PhasePlay {
		steps = 2 // the mover's extra turn
	}
	var buf [MaxLegalActions]Action
	for _, b := range LegalActions(&x, buf[:0]) {
		if !actionInMask(&x, b, mask) {
			continue
		}
		if !c.apply(&x, b, 1, mask) {
			return Win{}, false
		}
		y := x
		ApplyTrusted(&y, b)
		if wonBy(&y, mover) {
			return Win{Kind: y.Result.Kind, Steps: steps, Via: b}, true
		}
	}
	return Win{}, false
}

// actionInMask reports whether action a uses no structure slot outside
// mask: a play action's card must be in a slot of the mask; every other
// action uses no slot.
func actionInMask(s *State, a Action, mask SlotMask) bool {
	switch a.Type {
	case ActConstructBuilding, ActDiscardBuilding, ActConstructWonder:
		i := SlotOf(s, a.Card)
		return i >= 0 && mask&(1<<i) != 0
	}
	return true
}

// sameHidden reports whether going from state a to state b used no hidden
// information: no card was revealed, no age was dealt and no Great Library
// offer was drawn.
func sameHidden(a, b *State) bool {
	return a.Age == b.Age && faceDownCount(a) == faceDownCount(b) && b.Phase != PhasePickLibraryToken
}

// faceDownCount counts the face-down cards of the structure.
func faceDownCount(s *State) int {
	n := 0
	for i := range s.Slots {
		if s.Slots[i].Card != 0 && !s.Slots[i].FaceUp {
			n++
		}
	}
	return n
}

// buildPreview is what building one card would do for one seat now, as
// the cities stand: the immediate result, before any choice it opens.
type buildPreview struct {
	Paid       bool   // the preview paid the cheapest payment (the seat can afford it); otherwise it paid nothing
	Coins      int    // coins the building gives the seat (its coin effects and Urbanism's chain bonus), the payment not counted
	CoinsAfter [2]int // both seats' coins afterwards, payment, Economy and looting included
	ScoreDelta [2]int // change of each seat's score preview total
	Science    scienceEffect
	HasScience bool
	Pawn       pawnMove
	Moved      bool  // the pawn moved
	Opens      Phase // the choice the building opens (PhasePickBoardToken), or PhasePlay for none
	Win        Win   // a win, at once or through the board token the pair lets the seat take (Law)
	Wins       bool
}

// previewBuild builds card c for seat on a normalised copy of s, without
// touching the structure: it pays the cheapest payment if seat can afford
// it, adds c to the city and resolves the building's effects in the rules'
// order (the Urbanism bonus, coins, shields with looting, science). With
// rebuild set it is the Mausoleum's free rebuild of a discarded card: no
// payment and no Urbanism bonus. No card leaves a slot, so nothing is
// revealed and no age is dealt; c may be face up in the structure, in the
// discard pile or unseen.
//
// When the card completes a pair, the preview also tries each board token
// the seat would then pick, and reports a win through one (Law as the sixth
// symbol).
func previewBuild(s *State, seat Seat, c catalog.CardID, rebuild bool) buildPreview {
	var out buildPreview
	if catalog.Card(c) == nil || s.Players[seat].City.Has(c) {
		return out
	}
	x := normalised(s, seat)
	via := viaDiscard
	if !rebuild {
		via = viaPurchase
		q := BuildingQuote(&x, seat, c)
		if q.Affordable {
			pay(&x, seat, q, nil)
			out.Paid = true
			if q.ChainFree {
				via = viaChain
			}
		}
	}
	addCard(&x, seat, c)
	var buf [16]Event
	ev := buf[:0]
	o := resolveBuildingEffects(&x, seat, c, via, &ev)

	for _, e := range ev {
		if e.Kind == EvCoins && e.Seat == seat && e.A > 0 {
			out.Coins += int(e.A)
		}
	}
	out.Pawn, out.Moved = readPawnMove(ev, s.Looting)
	if sym := catalog.T.Sym[c]; sym != catalog.NoSymbol {
		out.HasScience = true
		out.Science = scienceEffect{Symbol: sym, Kind: ScienceEffectNew, DistinctAfter: DistinctSymbols(&x.Players[seat])}
		if x.Players[seat].Sci[sym] >= 2 {
			out.Science.Kind = ScienceEffectPair
			out.Science.TokenOnBoard = !s.BoardTokens.Empty()
		}
	}
	out.Opens = PhasePlay
	if o == outcomeChoice {
		out.Opens = x.Phase
	}
	out.fill(s, &x)
	if wonBy(&x, seat) {
		out.Win, out.Wins = Win{Kind: x.Result.Kind, Steps: 1}, true
	} else if o == outcomeChoice && x.Phase == PhasePickBoardToken {
		for t, ok := x.BoardTokens.Next(1); ok; t, ok = x.BoardTokens.Next(t + 1) {
			y := x
			takeBoardToken(&y, seat, t, nil)
			if wonBy(&y, seat) {
				out.Win, out.Wins = Win{Kind: y.Result.Kind, Steps: 1, Via: TokenAction(t)}, true
				break
			}
		}
	}
	return out
}

// fill sets the coins and score changes of a preview from the state before
// (s) and after (x).
func (p *buildPreview) fill(s, x *State) {
	before, after := Scores(s), Scores(x)
	for i := range p.CoinsAfter {
		p.CoinsAfter[i] = int(x.Players[i].Coins)
		p.ScoreDelta[i] = int(after[i].Total) - int(before[i].Total)
	}
}

// wonderPreview is what building one of a seat's wonders would do now: the
// effects up to its special choice, which it names without making it.
type wonderPreview struct {
	Paid       bool   // the preview paid the cheapest payment (the seat can afford it)
	CoinsDelta [2]int // change of both seats' coins: payment, Economy, the wonder's coins, looting and the Appian Way
	ScoreDelta [2]int // change of each seat's score preview total
	Pawn       pawnMove
	Moved      bool
	Win        Win // a win at once (the Colossus, the Statue of Zeus or Circus Maximus taking a capital) or through a rebuild (the Mausoleum)
	Wins       bool
	ExtraTurn  ExtraTurnSource
	Removes    catalog.WonderID // the wonder the seven-wonder rule would send back to the box, or 0
	Opens      Phase            // the special choice it opens, or PhasePlay for none
	Pool       int              // its options: destroy candidates, discarded cards, or the tokens the Great Library would draw
}

// previewWonder builds seat's wonder w on a normalised copy of s without
// tucking a card (the tucked card has no effect, and leaving it in the
// structure reveals nothing): it pays the cheapest payment if seat can
// afford it, then applies the wonder's effects in the rules' order (the
// seven-wonder rule, coins, shields, the extra turn, the Appian Way) and
// stops before the special choice, reporting only its kind and size. The
// Great Library never draws: the copy's box is emptied first, so its draw
// finds nothing, and the size is the number of tokens the public box count
// says it would draw. For the Mausoleum, the preview also tries each
// discarded card and reports a win through its rebuild.
func previewWonder(s *State, seat Seat, w catalog.WonderID) wonderPreview {
	var out wonderPreview
	k := wonderSlot(&s.Players[seat], w)
	if k < 0 || !wonderOpen(s, seat, k) {
		return out
	}
	x := normalised(s, seat)
	q := WonderQuote(&x, seat, w)
	if q.Affordable {
		pay(&x, seat, q, nil)
		out.Paid = true
	}
	buildWonder(&x, seat, k, 0)
	boxCount := 0
	for _, t := range x.BoxTokens {
		if t != 0 {
			boxCount++
		}
	}
	x.BoxTokens = [NumBoxTokens]catalog.TokenID{}
	var buf [16]Event
	ev := buf[:0]
	o := resolveWonderEffects(&x, seat, w, &ev)

	for _, e := range ev {
		if e.Kind == EvWonderRemoved {
			out.Removes = e.Wonder
		}
	}
	out.Pawn, out.Moved = readPawnMove(ev, s.Looting)
	out.ExtraTurn = extraTurnSource(&s.Players[seat], w)
	out.Opens = PhasePlay
	t := &catalog.T
	switch {
	case o == outcomeChoice:
		out.Opens = x.Phase
		switch x.Phase {
		case PhaseDestroyCard:
			out.Pool = int(x.Players[seat.Opp()].ColorN[x.Destroy])
		case PhaseBuildFromDiscard:
			out.Pool = x.Discard.Count()
		}
	case o == outcomeFinish && int(w) < len(t.WDraw) && t.WDraw[w] > 0 && boxCount > 0:
		out.Opens = PhasePickLibraryToken
		out.Pool = min(int(t.WDraw[w]), LibraryDraw, boxCount)
	}
	before, after := Scores(s), Scores(&x)
	for i := range out.CoinsDelta {
		out.CoinsDelta[i] = int(x.Players[i].Coins) - int(s.Players[i].Coins)
		out.ScoreDelta[i] = int(after[i].Total) - int(before[i].Total)
	}
	if wonBy(&x, seat) {
		out.Win, out.Wins = Win{Kind: x.Result.Kind, Steps: 1}, true
	} else if out.Opens == PhaseBuildFromDiscard {
		for d, ok := x.Discard.Next(1); ok; d, ok = x.Discard.Next(d + 1) {
			y := x
			reviveCard(&y, seat, d, nil)
			if wonBy(&y, seat) {
				out.Win, out.Wins = Win{Kind: y.Result.Kind, Steps: 1, Via: ReviveAction(d)}, true
				break
			}
		}
	}
	return out
}

// priceShiftEntry is one trade price a card would change: seat's price of
// one unit of resource Res, from From to To.
type priceShiftEntry struct {
	Seat     Seat
	Res      catalog.Resource
	From, To uint8
}

// priceShifts lists the trade prices a card would change, by seat and
// then in resource order: E[:N].
type priceShifts struct {
	E [2 * catalog.NumResources]priceShiftEntry
	N int
}

// priceShift returns the trade prices that change, for either seat, if seat
// adds card c to its city: the other seat's prices rise with c's fixed
// production (brown and grey cards; flexible producers never raise them),
// and seat's own fall with the prices c fixes (the Reserves, the Customs
// House). No Apply: a copy with c added to the city.
func priceShift(s *State, seat Seat, c catalog.CardID) priceShifts {
	var out priceShifts
	if catalog.Card(c) == nil || s.Players[seat].City.Has(c) {
		return out
	}
	x := *s
	addCard(&x, seat, c)
	for who := range Seat(2) {
		from, to := TradePrices(s, who), TradePrices(&x, who)
		for r := range catalog.NumResources {
			if from[r] != to[r] {
				out.E[out.N] = priceShiftEntry{Seat: who, Res: r, From: from[r], To: to[r]}
				out.N++
			}
		}
	}
	return out
}

// faceUpTargets lists the cards of the face-up slots of s, in slot order:
// ids[:n]. A hint that prices the cards in view captures them with this
// before it changes anything.
func faceUpTargets(s *State) (ids [SlotsPerAge]catalog.CardID, n int) {
	if s.Age < 1 || s.Age > NumAges {
		return ids, 0
	}
	for i := range s.Slots {
		if sl := &s.Slots[i]; sl.Card != 0 && sl.FaceUp {
			ids[n] = sl.Card
			n++
		}
	}
	return ids, n
}

// savesInView returns how many coins seat would save, in total, on the
// cards targets and on its open wonders if it added card c to its city:
// for each target other than c, its cheapest payment now minus its
// cheapest payment afterwards, and the same for each open wonder. A target
// free through a chain saves nothing. It is an indicator, not a sum anyone
// can collect, since the targets compete for the same turns. No Apply.
func savesInView(s *State, seat Seat, c catalog.CardID, targets []catalog.CardID) int {
	if catalog.Card(c) == nil || s.Players[seat].City.Has(c) {
		return 0
	}
	x := *s
	addCard(&x, seat, c)
	saved := 0
	for _, t := range targets {
		if t == c || s.Players[seat].City.Has(t) {
			continue
		}
		saved += int(BuildingQuote(s, seat, t).Total) - int(BuildingQuote(&x, seat, t).Total)
	}
	for k := range s.Players[seat].Wonders {
		if wonderOpen(s, seat, k) {
			w := s.Players[seat].Wonders[k].ID
			saved += int(WonderQuote(s, seat, w).Total) - int(WonderQuote(&x, seat, w).Total)
		}
	}
	return saved
}

// tokenValue is what taking one progress token would do for a seat now.
type tokenValue struct {
	Coins int  // coins the seat gains at once (Agriculture, Urbanism)
	VP    int  // change of the seat's score preview total
	Wins  bool // the token wins the game (Law as the sixth symbol)
}

// tokenPreview gives seat token t on a normalised copy of s (addToken, then
// the token's immediate effects) and reports the result. It never Applies:
// applying a token choice finishes the turn, which reveals cards. A token
// the seat already owns is worth nothing.
func tokenPreview(s *State, seat Seat, t catalog.TokenID) tokenValue {
	var out tokenValue
	if catalog.Token(t) == nil || s.Players[seat].Tokens.Has(t) {
		return out
	}
	x := normalised(s, seat)
	x.BoardTokens.Remove(t)
	addToken(&x, seat, t)
	resolveTokenEffects(&x, seat, t, nil)
	out.Coins = int(x.Players[seat].Coins) - int(s.Players[seat].Coins)
	out.VP = int(Scores(&x)[seat].Total) - int(Scores(s)[seat].Total)
	out.Wins = wonBy(&x, seat)
	return out
}

// ExtraTurnState is what an action does to the mover's turn. The view's
// names are view.ExtraTurnState.
type ExtraTurnState uint8

// The extra-turn states.
const (
	ExtraTurnNone ExtraTurnState = iota // the turn passes as usual
	ExtraTurnGain                       // the mover plays again
	ExtraTurnLost                       // an extra turn granted with the age's last card, so lost
	numExtraTurnStates
)

var extraTurnStateNames = [numExtraTurnStates]string{"none", "gain", "lost"}

// String returns the wire name.
func (e ExtraTurnState) String() string {
	return enumString(extraTurnStateNames[:], uint8(e), "extraTurnState")
}

// ExtraTurnSource says why a wonder grants an extra turn. The view's names
// are view.ExtraTurnSource.
type ExtraTurnSource uint8

// The extra-turn sources.
const (
	ExtraTurnSourceNone     ExtraTurnSource = iota // no extra turn
	ExtraTurnSourceWonder                          // the wonder's own effect
	ExtraTurnSourceTheology                        // the builder owns Theology
	numExtraTurnSources
)

var extraTurnSourceNames = [numExtraTurnSources]string{"none", "wonder", "theology"}

// String returns the wire name.
func (e ExtraTurnSource) String() string {
	return enumString(extraTurnSourceNames[:], uint8(e), "extraTurnSource")
}

// extraTurnSource returns why p building wonder w would play again: the
// wonder's own effect first (a wonder never grants two), else Theology.
func extraTurnSource(p *Player, w catalog.WonderID) ExtraTurnSource {
	switch {
	case int(w) < len(catalog.T.WPlayAgain) && catalog.T.WPlayAgain[w]:
		return ExtraTurnSourceWonder
	case wondersPlayAgain(p):
		return ExtraTurnSourceTheology
	}
	return ExtraTurnSourceNone
}

// extraTurnOutcome says, from the rules alone, what play action a does to
// the mover's turn in s: a wonder that grants an extra turn (its own effect
// or Theology) gains one, unless the card tucked under it is the last card
// of the age, which loses it; every other action passes the turn. It does
// not Apply a: finishTurn clears the extra turn when it uses it, so a copy
// would hide the answer. A supremacy that ends the game first is the win
// hints' business.
func extraTurnOutcome(s *State, a Action) (ExtraTurnState, ExtraTurnSource) {
	if s.Phase != PhasePlay || a.Type != ActConstructWonder {
		return ExtraTurnNone, ExtraTurnSourceNone
	}
	src := extraTurnSource(&s.Players[s.ToMove], a.Wonder)
	switch {
	case src == ExtraTurnSourceNone:
		return ExtraTurnNone, ExtraTurnSourceNone
	case Occupied(s)&(Occupied(s)-1) == 0: // the tucked card is the only one left
		return ExtraTurnLost, src
	}
	return ExtraTurnGain, src
}
