package engine

import (
	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
	"github.com/NightMachinery/ajayeb/server/internal/view"
)

// --- The decider's hints (M10.HS3) ---
//
// What each legal action of the seat to move would do (Choice.outcome), and
// what a play action it cannot afford would win (DisabledChoice.wins). Only
// the decider's view carries them, so the waiting seat never sees advice
// about a position it does not decide, and a spectator sees none.
//
// The outcome of an action is read from a copy of the state with the action
// applied: both seats' coins and score preview, the pawn and its looting,
// the mover's different symbols, the choice the action opens, the game's
// end, the trade prices and the affordability of the cards and wonders in
// view. Reading coins after one Apply is safe: a reveal, the next deal and
// the Great Library draw never change coins within that Apply. The turn
// order comes from extraTurnOutcome (no Apply), and wins from winsNow.
//
// The threats then look one turn ahead (gift-check, creates-threat and
// hidden-risk of the hints design): when the action passes the turn to the
// opponent in the same age, which wins would the opponent have, and which
// would the mover have on its next turn, compared with now. Those nested
// simulations run on a scrubbed copy (scrubbed): every card of the
// structure that was not face up before the action, the cards the action
// turned over included, becomes catalog.Unknown, so no simulation can use
// or be told apart by a card the decider has not seen. hidden-risk puts
// each card that may lie under a freed back into one freed slot of that
// copy in turn, and counts those that would let the opponent win.
//
// Everything runs under one simCtx with hintApplyBudget for the view, in a
// fixed order: first every action's own outcome, then the disabled wins
// and the starter threats, then the threats one turn ahead, then the
// reveal risk. What the budget cuts is left out, never guessed: a threat
// list is sent only when both lists it compares were finished.

// deciderFamilies reports whether a family that the decider's hints serve
// is on.
func deciderFamilies(f view.HintSettings) bool {
	return f.Scoring || f.Military || f.Science || f.Tempo || f.Threats || f.Economy
}

// threat is one win a seat would have with one action: Card for building
// or discarding a card, Wonder for building a wonder (with any card).
type threat struct {
	Card   catalog.CardID
	Wonder catalog.WonderID
	Kind   VictoryKind
}

// threatOf names the win of action a.
func threatOf(a Action, kind VictoryKind) threat {
	if a.Type == ActConstructWonder {
		return threat{Wonder: a.Wonder, Kind: kind}
	}
	return threat{Card: a.Card, Kind: kind}
}

// threatList is a list of threats without repeats, in the order found:
// T[:N].
type threatList struct {
	T [MaxLegalActions]threat
	N int
}

// has reports whether the list names the same card or wonder as t.
func (l *threatList) has(t threat) bool {
	for _, u := range l.T[:l.N] {
		if u.Card == t.Card && u.Wonder == t.Wonder {
			return true
		}
	}
	return false
}

// add appends t unless the list already names its card or wonder.
func (l *threatList) add(t threat) {
	if !l.has(t) && l.N < len(l.T) {
		l.T[l.N] = t
		l.N++
	}
}

// minus returns the threats of l that other does not name, in l's order.
func (l *threatList) minus(other *threatList) threatList {
	var out threatList
	for _, t := range l.T[:l.N] {
		if !other.has(t) {
			out.add(t)
		}
	}
	return out
}

// exhausted reports whether the context's budget ran out.
func (c *simCtx) exhausted() bool { return c != nil && c.out }

// threatsFor lists the wins seat would have with one action in x, where
// seat is to move in a play turn, by WinsNow with the cards of mask only
// (and the seat's wonders, which tuck a card of mask). With first set it
// stops at the first win. It reports false when the budget ran out before
// the list was finished.
func threatsFor(x *State, seat Seat, mask SlotMask, c *simCtx, first bool) (threatList, bool) {
	var out threatList
	if x.Over() || x.Phase != PhasePlay || x.ToMove != seat {
		return out, true
	}
	var buf [MaxLegalActions]Action
	for _, a := range LegalActions(x, buf[:0]) {
		w, ok := winsNow(x, a, mask, c)
		if c.exhausted() {
			return out, false
		}
		if ok {
			out.add(threatOf(a, w.Kind))
			if first {
				break
			}
		}
	}
	return out, true
}

// scrubbed returns a copy of x in which every card of the structure outside
// keep is catalog.Unknown, so that a simulation on it can neither use nor
// be told apart by a card that was not face up in keep. The slots keep
// their place, face and back, so the structure works as before; an Unknown
// card is never affordable, and winsNow's mask filters every other action
// on one.
func scrubbed(x *State, keep SlotMask) State {
	y := *x
	for i := range y.Slots {
		if y.Slots[i].Card != 0 && keep&(1<<i) == 0 {
			y.Slots[i].Card = catalog.Unknown
		}
	}
	return y
}

// deciderCalc is the work of one view's decider hints.
type deciderCalc struct {
	s     *State
	f     view.HintSettings
	c     *simCtx
	mover Seat
	mask  SlotMask // the face-up mask of s

	scores  [2]Score                       // before
	prices  [2][catalog.NumResources]uint8 // trade prices before
	cards   [SlotsPerAge]catalog.CardID    // the accessible face-up cards before: cards[:ncard]
	ncard   int                            //
	afford  [2][SlotsPerAge]bool           // by seat, whether each of cards was affordable
	wafford [2][WondersPerPlayer]bool      // by seat and wonder slot, whether the open wonder was affordable
	wopen   [2][WondersPerPlayer]bool      // by seat and wonder slot, whether the wonder was open

	// The wins each seat would have if it were its turn now (normalised
	// copies), computed when first needed.
	before   [2]threatList
	beforeOK [2]bool
	beforeAt [2]bool // computed

	// The unseen cards of the current age, for the reveal risk, computed
	// when first needed.
	pool   [SlotsPerAge + RemovedPerAge]catalog.CardID
	npool  int
	poolAt bool
}

// pendingAct is one choice whose outcome was computed: the choice's index in
// the view, its action, and the state after it.
type pendingAct struct {
	i int
	a Action
	x State
}

// deciderHints adds the decider's hints to out, the decider's view of s,
// for the families f switches on, spending the budget of c.
func deciderHints(s *State, f view.HintSettings, out *view.View, c *simCtx) {
	d := deciderCalc{s: s, f: f, c: c, mover: s.ToMove, mask: FaceUpSlots(s)}
	d.measureBefore()

	// Each action's own outcome.
	acts := make([]pendingAct, 0, len(out.Choices))
	var evbuf [32]Event
	for i := range out.Choices {
		ch := &out.Choices[i]
		a, err := ParseKey(ch.Key)
		if err != nil {
			continue
		}
		switch a.Type {
		case ActNone, ActDraftWonder, ActChooseStarter, ActResign:
			continue
		}
		if !c.apply(s, a, 0, d.mask) {
			break
		}
		x := *s
		ev := evbuf[:0]
		apply(&x, a, &ev)
		ch.Outcome = d.outcome(a, &x, ev)
		acts = append(acts, pendingAct{i: i, a: a, x: x})
	}
	if !f.Threats {
		return
	}

	// A play action the seat cannot afford: would it win with the coins?
	if s.Phase == PhasePlay {
		for i := range out.Disabled {
			dc := &out.Disabled[i]
			if dc.Reason != view.Reason(ReasonUnaffordable.String()) {
				continue
			}
			a, err := ParseKey(dc.Key)
			if err != nil {
				continue
			}
			if w, ok := d.winsWithCoins(a); ok {
				dc.Wins = viewWin(w)
			}
		}
	}

	// Who starts the next age: the wins the starting seat would have at once.
	if s.Phase == PhaseChooseStarter {
		for i := range out.Choices {
			ch := &out.Choices[i]
			if ch.Kind != view.ChoiceKindStarter || ch.Starter == nil {
				continue
			}
			seat := Seat(*ch.Starter & 1)
			y := normalised(s, seat)
			y = scrubbed(&y, d.mask)
			list, ok := d.nested(func() (threatList, bool) { return threatsFor(&y, seat, d.mask, c, false) })
			if ok && list.N > 0 {
				ch.Outcome = &view.OutcomeView{Threatens: viewThreats(seat, &list)}
			}
		}
	}

	// One turn ahead: the opponent's wins it gives or blocks, and the
	// mover's own wins it sets up.
	opp := d.mover.Opp()
	afterOpp := make([]threatList, len(acts))
	afterOK := make([]bool, len(acts))
	for k := range acts {
		p := &acts[k]
		if !d.passes(&p.x) {
			continue
		}
		o := out.Choices[p.i].Outcome
		y := scrubbed(&p.x, d.mask)
		after, ok := d.nested(func() (threatList, bool) { return threatsFor(&y, opp, d.mask, c, false) })
		afterOpp[k], afterOK[k] = after, ok
		if before, bok := d.beforeFor(opp); ok && bok {
			gives, blocks := after.minus(&before), before.minus(&after)
			o.Gives = viewThreatsPtr(opp, &gives)
			o.Blocks = viewThreatsPtr(opp, &blocks)
		}
		z := normalised(&y, d.mover)
		own, ok := d.nested(func() (threatList, bool) { return threatsFor(&z, d.mover, d.mask, c, false) })
		if before, bok := d.beforeFor(d.mover); ok && bok {
			mine := own.minus(&before)
			o.Threatens = viewThreatsPtr(d.mover, &mine)
		}
	}

	// The reveal risk: only where the opponent has no win already.
	for k := range acts {
		p := &acts[k]
		if !afterOK[k] || afterOpp[k].N > 0 || !d.passes(&p.x) {
			continue
		}
		if r, ok := d.revealRisk(&p.x); ok {
			out.Choices[p.i].Outcome.RevealRisk = r
		}
	}
}

// measureBefore records what the outcomes compare with: the scores, the
// trade prices, and which accessible face-up cards and open wonders each
// seat can afford now.
func (d *deciderCalc) measureBefore() {
	s := d.s
	d.scores = Scores(s)
	acc := Accessible(s) & d.mask
	for i := range s.Slots {
		if acc&(1<<i) != 0 {
			d.cards[d.ncard] = s.Slots[i].Card
			d.ncard++
		}
	}
	for seat := range Seat(2) {
		d.prices[seat] = TradePrices(s, seat)
		for j, cd := range d.cards[:d.ncard] {
			d.afford[seat][j] = BuildingQuote(s, seat, cd).Affordable
		}
		for k := range s.Players[seat].Wonders {
			if wonderOpen(s, seat, k) {
				d.wopen[seat][k] = true
				d.wafford[seat][k] = WonderQuote(s, seat, s.Players[seat].Wonders[k].ID).Affordable
			}
		}
	}
}

// outcome is the outcome of action a, with x the state after it and ev its
// events.
func (d *deciderCalc) outcome(a Action, x *State, ev []Event) *view.OutcomeView {
	s, f, mover := d.s, d.f, d.mover
	o := &view.OutcomeView{}
	if f.Scoring {
		after := Scores(x)
		for i := range o.Coins {
			o.Coins[i] = view.Amount(x.Players[i].Coins)
			o.Score[i] = view.Amount(after[i].Total) - view.Amount(d.scores[i].Total)
		}
		if x.Over() && x.Result.Kind == VictoryCivilian {
			o.End = viewResult(&x.Result)
		}
	}
	if f.Military {
		if m, moved := readPawnMove(ev, s.Looting); moved {
			pm := &view.PawnMoveView{From: int(m.From), To: int(m.To), Loot: make([]view.LootTaken, 0, m.NLoot)}
			for _, l := range m.Loot[:m.NLoot] {
				pm.Loot = append(pm.Loot, view.LootTaken{At: int(l.At), Lost: int(l.Lost)})
			}
			o.Pawn = pm
		}
	}
	if f.Science {
		if n := DistinctSymbols(&x.Players[mover]); n != DistinctSymbols(&s.Players[mover]) {
			o.Distinct = dtoInt(n)
		}
	}
	if !x.Over() && x.ToMove == mover {
		switch x.Phase {
		case PhaseDestroyCard, PhaseBuildFromDiscard, PhasePickBoardToken, PhasePickLibraryToken:
			if k, ok := pendingKind(x.Phase); ok {
				o.Opens = &k
			}
		}
	}
	if f.Tempo {
		o.Tempo = d.tempo(a, x)
	}
	if f.Threats {
		if w, ok := winsNow(s, a, d.mask, d.c); ok {
			o.Wins = viewWin(w)
		}
	}
	if f.Economy {
		o.Prices = d.priceChanges(x)
		if x.Age == s.Age && !x.Over() {
			o.Flips = d.flips(x)
		}
	}
	return o
}

// tempo is what action a, leading to x, does to the turn order, or nil when
// the turn simply passes.
func (d *deciderCalc) tempo(a Action, x *State) *view.TempoView {
	s := d.s
	state, src := extraTurnOutcome(s, a)
	t := view.TempoView{
		ExtraTurn: view.ExtraTurnState(state.String()),
		Source:    view.ExtraTurnSource(src.String()),
	}
	occ := Occupied(s)
	switch a.Type {
	case ActConstructBuilding, ActDiscardBuilding, ActConstructWonder:
		t.LastCard = occ != 0 && occ&(occ-1) == 0
	}
	switch {
	case x.Over():
	case x.Phase == PhaseChooseStarter && s.Phase != PhaseChooseStarter:
		t.StarterChooser = dtoInt(int(x.ToMove))
	case x.Age == s.Age && x.Age < NumAges && Occupied(x) == 0 && x.ToMove == d.mover &&
		(x.Phase == PhaseDestroyCard || x.Phase == PhasePickBoardToken || x.Phase == PhasePickLibraryToken):
		// The age's last card opened a choice that cannot move the pawn:
		// the chooser follows from the pawn as it stands.
		if seat, choose := starterChooser(x); choose {
			t.StarterChooser = dtoInt(int(seat))
		}
	}
	if state == ExtraTurnNone && !t.LastCard && t.StarterChooser == nil {
		return nil
	}
	return &t
}

// priceChanges lists the trade prices that differ between s and x, by seat
// and then in resource order, or nil when none does.
func (d *deciderCalc) priceChanges(x *State) *[]view.PriceShift {
	var ps priceShifts
	for who := range Seat(2) {
		to := TradePrices(x, who)
		for r := range catalog.NumResources {
			if from := d.prices[who][r]; from != to[r] {
				ps.E[ps.N] = priceShiftEntry{Seat: who, Res: r, From: from, To: to[r]}
				ps.N++
			}
		}
	}
	if ps.N == 0 {
		return nil
	}
	out := viewPriceShifts(ps)
	return &out
}

// flips lists the accessible face-up cards (still in the structure) and
// open wonders whose affordability differs between s and x, or nil when
// none does: for the opponent both ways, for the mover only what it can
// newly afford (that a payment leaves the mover short of something else is
// frequent and plain, and would only lengthen the view). Only cards
// captured before the action are priced, so a card the action turns over
// is never named.
func (d *deciderCalc) flips(x *State) *[]view.AffordFlip {
	out := []view.AffordFlip{}
	for seat := range Seat(2) {
		keep := func(now bool) bool { return seat != d.mover || now }
		for j, cd := range d.cards[:d.ncard] {
			if SlotOf(x, cd) < 0 {
				continue
			}
			if now := BuildingQuote(x, seat, cd).Affordable; now != d.afford[seat][j] && keep(now) {
				out = append(out, view.AffordFlip{Seat: int(seat), Card: dtoCardPtr(cd), Affordable: now})
			}
		}
		for k := range x.Players[seat].Wonders {
			if !d.wopen[seat][k] || !wonderOpen(x, seat, k) {
				continue
			}
			w := x.Players[seat].Wonders[k].ID
			if now := WonderQuote(x, seat, w).Affordable; now != d.wafford[seat][k] && keep(now) {
				out = append(out, view.AffordFlip{Seat: int(seat), Wonder: dtoWonderPtr(w), Affordable: now})
			}
		}
	}
	if len(out) == 0 {
		return nil
	}
	return &out
}

// passes reports whether the state x after one of the decider's actions is
// the opponent's play turn in the same age: the only case the threats one
// turn ahead are computed for. An action that leaves the mover to decide
// (a choice it opened, an extra turn) or ends the age is skipped.
func (d *deciderCalc) passes(x *State) bool {
	return !x.Over() && x.Phase == PhasePlay && x.ToMove == d.mover.Opp() && x.Age == d.s.Age
}

// nested runs f as a simulation one turn ahead, reported at depth 2.
func (d *deciderCalc) nested(f func() (threatList, bool)) (threatList, bool) {
	if d.c != nil {
		d.c.nest += 2
		defer func() { d.c.nest -= 2 }()
	}
	return f()
}

// beforeFor returns the wins seat would have if it were its turn now, on a
// normalised, scrubbed copy of s, and whether the list was finished.
func (d *deciderCalc) beforeFor(seat Seat) (threatList, bool) {
	if !d.beforeAt[seat] {
		d.beforeAt[seat] = true
		y := normalised(d.s, seat)
		y = scrubbed(&y, d.mask)
		d.before[seat], d.beforeOK[seat] = d.nested(func() (threatList, bool) {
			return threatsFor(&y, seat, d.mask, d.c, false)
		})
	}
	return d.before[seat], d.beforeOK[seat]
}

// winsWithCoins reports the supremacy play action a would win if the mover
// had the coins to pay for it: on a copy where the mover is rich, a must
// be legal and win by WinsNow with a military or scientific supremacy (the
// extra coins would change a civilian count, so those do not count).
func (d *deciderCalc) winsWithCoins(a Action) (Win, bool) {
	x := *d.s
	x.Players[d.mover].Coins += 200
	if legality(&x, a) != ReasonNone {
		return Win{}, false
	}
	w, ok := winsNow(&x, a, d.mask, d.c)
	if !ok || (w.Kind != VictoryMilitary && w.Kind != VictoryScience) {
		return Win{}, false
	}
	return w, true
}

// revealRisk is the chance that the face-down cards the action turned over
// (x is the state after it) let the opponent win at once: n freed slots of
// the age's back, m unseen cards of the age that may lie under them, of
// which k would let the opponent win by WinsNow if one lay face up in a
// freed slot (judged on a scrubbed copy of x, with the coins after the
// action). P = 1 - C(m-k, n) / C(m, n). A guild back never matters (a guild
// has no shield and no symbol). It reports false when no age back was
// freed, k is 0, or the budget ran out.
func (d *deciderCalc) revealRisk(x *State) (*view.Odds, bool) {
	s := d.s
	back := catalog.AgeBack(s.Age)
	n, freed, slot := 0, 0, -1
	for i := range s.Slots {
		if s.Slots[i].Card == 0 || s.Slots[i].FaceUp || x.Slots[i].Card == 0 || !x.Slots[i].FaceUp {
			continue
		}
		freed++
		if s.Slots[i].Back == back {
			n++
			if slot < 0 {
				slot = i
			}
		}
	}
	if n == 0 {
		return nil, false
	}
	pool := d.unseenPool()
	m := len(pool)
	if m < n {
		return nil, false
	}
	opp := d.mover.Opp()
	keep := d.mask | 1<<slot
	k := 0
	for _, cand := range pool {
		y := scrubbed(x, d.mask)
		y.Slots[slot].Card = cand
		list, ok := d.nested(func() (threatList, bool) { return threatsFor(&y, opp, keep, d.c, true) })
		if !ok {
			return nil, false
		}
		if list.N > 0 {
			k++
		}
	}
	if k == 0 {
		return nil, false
	}
	den := binomial(m, n)
	num := den - binomial(m-k, n)
	g := gcd(num, den)
	return &view.Odds{Num: num / g, Den: den / g, Freed: freed}, true
}

// unseenPool returns the cards of the current age (guilds aside) whose
// public place is unseen this age: what may lie under its face-down backs.
func (d *deciderCalc) unseenPool() []catalog.CardID {
	if !d.poolAt {
		d.poolAt = true
		s := d.s
		if s.Age >= 1 && s.Age <= NumAges {
			set := catalog.T.AgeCards[s.Age]
			for c, ok := set.Next(1); ok; c, ok = set.Next(c + 1) {
				if publicPlace(s, c).Place == PlaceUnseenThisAge && d.npool < len(d.pool) {
					d.pool[d.npool] = c
					d.npool++
				}
			}
		}
	}
	return d.pool[:d.npool]
}

// binomial returns C(m, n), 0 when n > m.
func binomial(m, n int) int {
	if n < 0 || n > m {
		return 0
	}
	r := 1
	for i := 1; i <= n; i++ {
		r = r * (m - n + i) / i
	}
	return r
}

// gcd returns the greatest common divisor of a and b (b > 0).
func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

// viewThreats converts the threats of seat to their DTOs.
func viewThreats(seat Seat, l *threatList) *[]view.ThreatView {
	out := make([]view.ThreatView, 0, l.N)
	for _, t := range l.T[:l.N] {
		tv := view.ThreatView{Seat: int(seat), Kind: view.VictoryKind(t.Kind.String())}
		if t.Wonder != 0 {
			tv.Wonder = dtoWonderPtr(t.Wonder)
		} else {
			tv.Card = dtoCardPtr(t.Card)
		}
		out = append(out, tv)
	}
	return &out
}

// viewThreatsPtr is viewThreats, or nil for an empty list.
func viewThreatsPtr(seat Seat, l *threatList) *[]view.ThreatView {
	if l.N == 0 {
		return nil
	}
	return viewThreats(seat, l)
}
