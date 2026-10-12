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
// hints (hints_decider.go) Apply each legal action and some follow-ups; they
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
	// slots holds the per-card hints of each face-up slot, by seat, for the
	// slots in cards; nil when no per-card family is on.
	slots *[SlotsPerAge][2]view.CardHints
	cards SlotMask
	// pending is the destroy choice's per-candidate hints, or nil.
	pending *[]view.PendingCardHint
	// players is the board-level hints about each seat (hints_board.go),
	// valid when hasPlayers is set; wonders is what building each open
	// wonder would do, by seat and wonder slot, nil where none.
	players    [2]view.PlayerHintsView
	hasPlayers bool
	wonders    [2][WondersPerPlayer]*view.WonderEffectView
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
func computeHints(s *State, f view.HintSettings, c *simCtx) Hints {
	h := Hints{settings: f, board: view.HintsView{Discard: []view.DiscardHintView{}}}
	if s.Over() {
		return h
	}
	// The board-level hints (hints_board.go). The threats among them spend
	// one hintApplyBudget of their own; a traced context (the leak tests)
	// sees their simulations.
	if c == nil {
		c = &simCtx{budget: hintApplyBudget}
	}
	boardHints(s, f, &h, c)
	h.hasPlayers = s.Phase != PhaseWonderDraft
	// Each part under its family's switch; a field of a family that is off
	// stays zero or empty.
	if perCardFamilies(f) {
		h.slots, h.cards = slotHints(s, f)
	}
	if f.Wonders {
		h.board.Discard = discardHints(s, f)
	}
	if f.Economy && s.Phase == PhaseDestroyCard {
		p := destroyHints(s)
		h.pending = &p
	}
	return h
}

// addHints copies the shared hints h into out, the view of s for v, and
// adds the decider's hints when v is the seat to move. The caller has
// checked that h is not nil and that the game is not over.
func addHints(s *State, v Viewer, h *Hints, out *view.View) {
	board := h.board
	out.Hints = &board
	// The per-card hints are the same for every viewer; the views share the
	// slices, which nobody changes once built.
	if h.slots != nil {
		for i := range out.Structure {
			if sv := &out.Structure[i]; sv.Index < SlotsPerAge && h.cards&(1<<sv.Index) != 0 {
				ch := h.slots[sv.Index]
				sv.Hints = &ch
			}
		}
	}
	if h.pending != nil && out.Pending != nil && out.Pending.Kind == view.PendingKindDestroyCard {
		p := *h.pending
		out.Pending.CardHints = &p
	}
	if h.hasPlayers {
		for seat := range out.Players {
			ph := h.players[seat]
			out.Players[seat].Hints = &ph
			for k := range out.Players[seat].Wonders {
				if k < WondersPerPlayer && h.wonders[seat][k] != nil && out.Players[seat].Wonders[k].Cost != nil {
					out.Players[seat].Wonders[k].Effect = h.wonders[seat][k]
				}
			}
		}
	}
	// The Great Library offer's values, only for the seat choosing from it,
	// whose view alone lists the offer.
	seat, seated := v.Seat()
	decider := seated && seat == s.ToMove
	if decider && h.settings.Tokens && s.Phase == PhasePickLibraryToken && out.Pending != nil {
		vals := libraryTokenValues(s)
		out.Pending.TokenValues = &vals
	}
	// The decider's hints (Choice.outcome and DisabledChoice.wins), under
	// a fresh hintApplyBudget for this view (hints_decider.go).
	if decider && deciderFamilies(h.settings) {
		deciderHints(s, h.settings, out, &simCtx{budget: hintApplyBudget})
	}
}

// simCtx counts the Applies of one hint computation against a budget and,
// in tests, reports every simulation. The nil context has no budget and
// reports nothing.
type simCtx struct {
	budget  int  // Applies allowed; 0 means no limit
	applies int  // Applies spent
	out     bool // an Apply was refused: the budget ran out
	// nest is added to the depth of every simulation reported: 2 while the
	// decider's hints simulate the next turn on a copy whose unseen cards
	// are scrubbed (hints_decider.go), so the leak tests can tell those
	// simulations apart.
	nest  int
	trace func(simNote)
}

// simNote describes one simulated action, for the leak tests. It holds a
// copy of the state, so that no simulation's scratch state escapes to the
// heap through the trace.
type simNote struct {
	State  State    // the state the action is applied to
	For    Seat     // the seat the simulation is for (the mover)
	Action Action   // the action simulated
	Depth  int      // 0 for the action asked about, 1 for a follow-up; 2 and 3 for the same a turn later, on a scrubbed copy (hints_decider.go)
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
		c.out = true
		return false
	}
	c.applies++
	if c.trace != nil {
		c.trace(simNote{State: *s, For: s.ToMove, Action: a, Depth: depth + c.nest, Mask: mask})
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

// --- Per-card hints (M10.HS2) ---
//
// What building each face-up card would do for each seat, the board pips
// (win, pair, loot) and the card sheet's lines: the science pair it
// completes, the pawn move and looting, a win, the trade prices it moves,
// the coins a producer saves on the cards in view, whether it is the last
// producer of its resource, and its worth now. Each part is computed only
// while its family is on. The hints are shared: the same for every viewer,
// computed once per broadcast from public information only (previewBuild,
// priceShift, savesInView and publicPlace never touch the structure).

// HintCode is a board pip a card earns for a seat. The view's names are
// view.HintCode.
type HintCode uint8

// The hint codes, in the order a card's codes are listed.
const (
	HintWin  HintCode = iota // building it wins, at once or through the board token its pair lets the seat take
	HintPair                 // building it completes the seat's science pair while a board token remains
	HintLoot                 // building it takes a looting token from the other seat
	numHintCodes
)

var hintCodeNames = [numHintCodes]string{"win", "pair", "loot"}

// String returns the wire name.
func (c HintCode) String() string { return enumString(hintCodeNames[:], uint8(c), "hintCode") }

// LastProducer says that a brown or grey card is the last producer of its
// resource in view. The view's names are view.LastProducer.
type LastProducer uint8

// The last-producer states.
const (
	LastProducerCertain LastProducer = iota // no other producer can still come into play
	LastProducerVisible                     // none in view, but unseen cards may hold one
	numLastProducers
)

var lastProducerNames = [numLastProducers]string{"lastCertain", "lastVisible"}

// String returns the wire name.
func (l LastProducer) String() string {
	return enumString(lastProducerNames[:], uint8(l), "lastProducer")
}

// perCardFamilies reports whether any family that the per-card hints serve
// is on.
func perCardFamilies(f view.HintSettings) bool {
	return f.Science || f.Military || f.Threats || f.Economy || f.Scoring || f.Counting
}

// slotHints computes the per-card hints of every face-up slot of s, for
// both seats, and the set of slots it filled.
func slotHints(s *State, f view.HintSettings) (*[SlotsPerAge][2]view.CardHints, SlotMask) {
	ids, n := faceUpTargets(s)
	if n == 0 {
		return nil, 0
	}
	var out [SlotsPerAge][2]view.CardHints
	var filled SlotMask
	for i := range s.Slots {
		sl := &s.Slots[i]
		if sl.Card == 0 || !sl.FaceUp {
			continue
		}
		for seat := range Seat(2) {
			out[i][seat] = cardHints(s, seat, sl.Card, f, ids[:n], false)
		}
		filled |= 1 << i
	}
	return &out, filled
}

// cardHints is what building card c would do for seat now, with the parts
// of the families f switches on. targets are the face-up cards, for
// savesInView. With rebuild set it is the Mausoleum's free rebuild of a
// discarded card.
func cardHints(s *State, seat Seat, c catalog.CardID, f view.HintSettings, targets []catalog.CardID, rebuild bool) view.CardHints {
	ch := view.CardHints{Codes: []view.HintCode{}, PriceShift: []view.PriceShift{}}
	p := previewBuild(s, seat, c, rebuild)
	e := view.EffectView{Paid: p.Paid}
	said := false
	if f.Scoring && worthCard(c) && (p.Coins != 0 || p.ScoreDelta[seat] != 0) {
		e.Coins = p.Coins
		e.ScoreDelta = p.ScoreDelta[seat]
		e.OppScoreDelta = p.ScoreDelta[seat.Opp()]
		said = true
	}
	if f.Threats && p.Wins {
		e.Wins = viewWin(p.Win)
		said = true
		if p.Paid || rebuild {
			ch.Codes = append(ch.Codes, view.HintCode(HintWin.String()))
		}
	}
	if f.Science && p.HasScience {
		e.Science = &view.ScienceEffect{
			Symbol:        view.ScienceSymbol(p.Science.Symbol.String()),
			Kind:          view.ScienceEffectKind(p.Science.Kind.String()),
			DistinctAfter: p.Science.DistinctAfter,
			TokenOnBoard:  p.Science.TokenOnBoard,
		}
		said = true
		if p.Science.Kind == ScienceEffectPair && p.Science.TokenOnBoard {
			ch.Codes = append(ch.Codes, view.HintCode(HintPair.String()))
		}
	}
	if f.Military && p.Moved {
		m := &view.PawnMoveView{From: int(p.Pawn.From), To: int(p.Pawn.To), Loot: make([]view.LootTaken, 0, p.Pawn.NLoot)}
		for _, l := range p.Pawn.Loot[:p.Pawn.NLoot] {
			m.Loot = append(m.Loot, view.LootTaken{At: int(l.At), Lost: int(l.Lost)})
		}
		e.Military = m
		said = true
		if len(m.Loot) > 0 {
			ch.Codes = append(ch.Codes, view.HintCode(HintLoot.String()))
		}
	}
	if said {
		ch.Effect = &e
	}
	if f.Economy {
		ch.PriceShift = viewPriceShifts(priceShift(s, seat, c))
		if producerCard(c) {
			saved := savesInView(s, seat, c, targets)
			ch.SavesInView = &saved
		}
	}
	if f.Counting {
		if l, ok := lastProducer(s, c); ok {
			v := view.LastProducer(l.String())
			ch.LastProducer = &v
		}
	}
	return ch
}

// worthCard reports whether c is a yellow card or a guild, whose worth now
// (coins and points) the scoring family sends.
func worthCard(c catalog.CardID) bool {
	if int(c) >= len(catalog.T.Color) {
		return false
	}
	col := catalog.T.Color[c]
	return col == catalog.Yellow || col == catalog.Purple
}

// saysSomething reports whether a card's hints carry anything beyond its
// price: a pip, an effect, a price it moves or the last-producer note.
func saysSomething(ch *view.CardHints) bool {
	return len(ch.Codes) > 0 || ch.Effect != nil || len(ch.PriceShift) > 0 || ch.LastProducer != nil ||
		(ch.SavesInView != nil && *ch.SavesInView > 0)
}

// viewWin converts a Win to its DTO.
func viewWin(w Win) *view.WinView {
	out := &view.WinView{Kind: view.VictoryKind(w.Kind.String()), Steps: w.Steps}
	if w.Via.Type != ActNone {
		if k := w.Via.Key(); k != "" {
			out.Via = &k
		}
	}
	return out
}

// viewPriceShifts converts price shifts to their DTOs.
func viewPriceShifts(ps priceShifts) []view.PriceShift {
	out := make([]view.PriceShift, 0, ps.N)
	for _, e := range ps.E[:ps.N] {
		out = append(out, view.PriceShift{Seat: int(e.Seat), Resource: dtoResource(e.Res), From: int(e.From), To: int(e.To)})
	}
	return out
}

// producerCard reports whether c changes what its owner pays for
// resources: a brown or grey card (fixed production), a flexible producer
// (the Forum, the Caravansery) or a card that fixes trade prices (the
// Reserves, the Customs House). savesInView is sent for these only.
func producerCard(c catalog.CardID) bool {
	t := &catalog.T
	if int(c) >= len(t.Color) {
		return false
	}
	col := t.Color[c]
	return col == catalog.Brown || col == catalog.Grey || t.Choice[c] != 0 || t.TradeFix[c] != 0
}

// lastProducer says whether brown or grey card c is the last producer of
// its resources in view (economy.md "Resources"): another brown or grey
// card producing one of them counts while it may still come into play,
// that is face up, unseen in this age or a later one, or in the discard
// pile while some seat's Mausoleum is open. With none in view it is the
// last for certain, unless an unseen card may hold one. Cards in a city,
// tucked, or certainly set aside never come back. The same for both
// seats, from public places only. It reports false when c is not a brown
// or grey card, or another producer is in view.
func lastProducer(s *State, c catalog.CardID) (LastProducer, bool) {
	t := &catalog.T
	if int(c) >= len(t.Color) || (t.Color[c] != catalog.Brown && t.Color[c] != catalog.Grey) {
		return 0, false
	}
	mausoleum := mausoleumOpen(s, 0) || mausoleumOpen(s, 1)
	unseen := false
	for d, ok := t.All.Next(1); ok; d, ok = t.All.Next(d + 1) {
		if d == c || (t.Color[d] != catalog.Brown && t.Color[d] != catalog.Grey) || !sharesProduction(c, d) {
			continue
		}
		switch publicPlace(s, d).Place {
		case PlaceFaceUp:
			return 0, false
		case PlaceDiscard:
			if mausoleum {
				return 0, false
			}
		case PlaceUnseenThisAge, PlaceUnseenLaterAge:
			unseen = true
		}
	}
	if unseen {
		return LastProducerVisible, true
	}
	return LastProducerCertain, true
}

// sharesProduction reports whether cards c and d produce a resource in
// common.
func sharesProduction(c, d catalog.CardID) bool {
	for r := range catalog.NumResources {
		if catalog.T.Prod[c][r] > 0 && catalog.T.Prod[d][r] > 0 {
			return true
		}
	}
	return false
}

// mausoleumOpen reports whether seat has an open wonder that rebuilds from
// the discard pile (the Mausoleum).
func mausoleumOpen(s *State, seat Seat) bool {
	p := &s.Players[seat]
	for k := range p.Wonders {
		w := p.Wonders[k].ID
		if int(w) < len(catalog.T.WRebuild) && catalog.T.WRebuild[w] && wonderOpen(s, seat, k) {
			return true
		}
	}
	return false
}

// discardHints computes what rebuilding each discarded card would do, for
// each seat whose Mausoleum is open and for the seat choosing a card to
// rebuild (mausoleum-pool-hint): by seat, then in ascending card order.
// The discard pile and the cities are public.
func discardHints(s *State, f view.HintSettings) []view.DiscardHintView {
	out := []view.DiscardHintView{}
	if s.Discard.Empty() {
		return out
	}
	ids, n := faceUpTargets(s)
	for seat := range Seat(2) {
		if !mausoleumOpen(s, seat) && (s.Phase != PhaseBuildFromDiscard || s.ToMove != seat) {
			continue
		}
		for d, ok := s.Discard.Next(1); ok; d, ok = s.Discard.Next(d + 1) {
			slug, ok := dtoCard(d)
			if !ok || s.Players[seat].City.Has(d) {
				continue
			}
			ch := cardHints(s, seat, d, f, ids[:n], true)
			if !saysSomething(&ch) {
				continue
			}
			q := BuildingQuote(s, seat, d)
			out = append(out, view.DiscardHintView{
				Card:      slug,
				Seat:      int(seat),
				Hints:     ch,
				Total:     int(q.Total),
				ChainFree: q.ChainFree,
			})
		}
	}
	return out
}

// destroyHints computes, for each candidate of the open destroy choice,
// what its loss would change (destroy-price-drop, opponent-pending-impact):
// the trade prices it moves (the destroyer's fall), the coins the
// destroyer would save on the face-up cards and its open wonders, and
// whether it is the victim's only fixed producer of a resource. The
// candidates are the victim's cards of the destroy colour, which are
// public, in ascending card order as PendingView.cards lists them.
func destroyHints(s *State) []view.PendingCardHint {
	out := []view.PendingCardHint{}
	destroyer := s.ToMove
	victim := destroyer.Opp()
	vp := &s.Players[victim]
	ids, n := faceUpTargets(s)
	for c, ok := vp.City.Next(1); ok; c, ok = vp.City.Next(c + 1) {
		if cardColor(c) != s.Destroy {
			continue
		}
		slug, ok := dtoCard(c)
		if !ok {
			continue
		}
		x := *s
		removeCard(&x, victim, c)
		var ps priceShifts
		for who := range Seat(2) {
			from, to := TradePrices(s, who), TradePrices(&x, who)
			for r := range catalog.NumResources {
				if from[r] != to[r] {
					ps.E[ps.N] = priceShiftEntry{Seat: who, Res: r, From: from[r], To: to[r]}
					ps.N++
				}
			}
		}
		saved := 0
		for _, t := range ids[:n] {
			if s.Players[destroyer].City.Has(t) {
				continue
			}
			saved += int(BuildingQuote(s, destroyer, t).Total) - int(BuildingQuote(&x, destroyer, t).Total)
		}
		for k := range s.Players[destroyer].Wonders {
			if wonderOpen(s, destroyer, k) {
				w := s.Players[destroyer].Wonders[k].ID
				saved += int(WonderQuote(s, destroyer, w).Total) - int(WonderQuote(&x, destroyer, w).Total)
			}
		}
		only := false
		for r := range catalog.NumResources {
			if pr := catalog.T.Prod[c][r]; pr > 0 && vp.Prod[r] == pr {
				only = true
			}
		}
		out = append(out, view.PendingCardHint{Card: slug, PriceShift: viewPriceShifts(ps), SavesInView: saved, OnlySource: only})
	}
	return out
}
