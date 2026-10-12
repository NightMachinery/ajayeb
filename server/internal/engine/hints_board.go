package engine

import (
	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
	"github.com/NightMachinery/ajayeb/server/internal/view"
)

// --- The board-level hints (M10.HS4) ---
//
// The shared facts about the board as a whole and about each seat, sent in
// HintsView and PlayerView.hints, each under its family's switch:
//
//   - science: where each seat stands on each of the seven symbols (paired,
//     a live single whose partner may still come, a single whose partner
//     can never come, open, dead), how many symbols it still needs, whether
//     they can still come, whether Law would win, and the public place of
//     both cards of each symbol;
//   - military: the shields still needed to take each capital, whether the
//     attacker can no longer gather them, and what each looting token would
//     take now;
//   - threats: the wins each seat would have with one action if it were
//     its turn now (only in a play turn or a starter choice, the phase gate
//     of the hints design);
//   - wonders: how many more wonders the game allows, and what building
//     each open wonder would do (WonderView.effect);
//   - tokens: what each board token would do for each seat, the tokens in
//     the box named by elimination while a Great Library is open (owner
//     answer Q1 B; the order stays secret), and the Great Library offer's
//     values for the seat choosing from it;
//   - scoring: coins to the next point, who would win on points now, and
//     the points of each owned guild;
//   - counting: the chains' targets and their places, the cards that may
//     lie under each face-down back, and the worth of each unseen guild;
//   - tempo: who takes the age's last card if nobody plays again, and who
//     will choose the next age's starter.
//
// Everything reads public information only: publicPlace for every "where is
// this card", copy-only previews, and, for the threats, WinsNow on a
// normalised copy whose unseen cards are scrubbed (scrubbed), limited to
// the face-up mask. The leak guard checks the whole view.

// boardHints fills the board-level parts of h for s.
func boardHints(s *State, f view.HintSettings, h *Hints, c *simCtx) {
	b := &h.board
	b.Pairs = []view.SciencePairView{}
	b.Looting = []view.LootHint{}
	b.Tokens = []view.TokenValueView{}
	b.BoxTokens = []string{}
	b.GuildPreview = []view.GuildPreviewView{}
	b.Unseen = []view.UnseenView{}
	if s.Phase == PhaseWonderDraft {
		return
	}
	if f.Wonders {
		b.WondersLeft = MaxWondersBuilt - int(s.WondersBuilt)
	}
	if f.Tempo {
		if seat, ok := lastCardTo(s); ok {
			b.LastCardTo = dtoInt(int(seat))
		}
		b.NextStarter = nextStarter(s)
	}
	if f.Scoring {
		b.PreviewWinner = previewWinner(s)
	}
	if f.Science {
		b.Pairs = sciencePairs(s)
	}
	if f.Military {
		b.Looting = lootHints(s)
	}
	if f.Tokens {
		b.Tokens = boardTokenValues(s)
		if greatLibraryOpen(s, 0) || greatLibraryOpen(s, 1) {
			b.BoxTokens = boxTokenNames(s)
		}
	}
	if f.Counting {
		b.Unseen = unseenPools(s)
		if s.Age == NumAges {
			b.GuildPreview = guildPreview(s)
		}
	}
	for seat := range Seat(2) {
		h.players[seat] = playerHints(s, seat, f, c)
	}
	if f.Wonders {
		for seat := range Seat(2) {
			p := &s.Players[seat]
			for k := range p.Wonders {
				if wonderOpen(s, seat, k) {
					e := wonderEffect(s, seat, p.Wonders[k].ID, f)
					h.wonders[seat][k] = &e
				}
			}
		}
	}
}

// --- tempo ---

// lastCardTo returns the seat that takes the age's last card if nobody
// plays again, and false where no one can say: in the draft, while a new
// age's starter is chosen (the starter is not known yet), or when the
// structure is empty. In a play turn the next taker is the seat to move;
// during a choice a card opened, it is the active seat when it plays
// again, else the other seat. The takers then alternate, so the last of R
// cards goes to the next taker when R is odd (turn.md "Whose turn it is",
// "Extra turns"; ages-scoring.md "The end of an age").
func lastCardTo(s *State) (Seat, bool) {
	if s.Over() || s.Age < 1 || s.Age > NumAges {
		return 0, false
	}
	var next Seat
	switch s.Phase {
	case PhasePlay:
		next = s.ToMove
	case PhaseDestroyCard, PhaseBuildFromDiscard, PhasePickBoardToken, PhasePickLibraryToken:
		next = s.Active.Opp()
		if s.ExtraTurn {
			next = s.Active
		}
	default:
		return 0, false
	}
	r := 0
	for occ := Occupied(s); occ != 0; occ &= occ - 1 {
		r++
	}
	switch {
	case r == 0:
		return 0, false
	case r%2 == 1:
		return next, true
	}
	return next.Opp(), true
}

// nextStarter says who will choose the next age's starter as the pawn
// stands (ages-scoring.md "Who starts the next age"): the weaker seat, or
// with a centred pawn the player who takes the age's last card. Nil in Age
// III and outside the ages.
func nextStarter(s *State) *view.NextStarterView {
	if s.Age < 1 || s.Age >= NumAges || s.Over() {
		return nil
	}
	out := &view.NextStarterView{}
	switch {
	case s.Military > 0:
		out.Chooser = dtoInt(1)
	case s.Military < 0:
		out.Chooser = dtoInt(0)
	default:
		out.ByLastCard = true
		out.LastActiveStarts = s.Opts.CenterPawn == LastActiveStarts
	}
	return out
}

// --- scoring ---

// previewWinner says who would win on points if the game ended now, by the
// end-of-game comparison and tie-break (ages-scoring.md "Tie-break and
// shared victory").
func previewWinner(s *State) *view.PreviewWinnerView {
	sc := Scores(s)
	w := civilianWinner(&sc)
	out := &view.PreviewWinnerView{By: view.TieBreakTotal}
	d := int(sc[0].Total) - int(sc[1].Total)
	if d == 0 {
		out.By = view.TieBreakBlue
		d = int(sc[0].Blue) - int(sc[1].Blue)
		if d == 0 {
			out.By = view.TieBreakShared
		}
	}
	out.Margin = max(d, -d)
	if w != SharedVictory {
		out.Winner = dtoInt(int(w))
	}
	return out
}

// coinsToNextVp is how many more coins seat needs for its next point from
// coins (one point per full three).
func coinsToNextVp(p *Player) int {
	return coinsPerPoint - int(p.Coins)%coinsPerPoint
}

// guildPoints lists the points each guild of seat's city is worth now and
// the city it counts.
func guildPoints(s *State, seat Seat) []view.GuildPointsView {
	out := []view.GuildPointsView{}
	p, o := &s.Players[seat], &s.Players[seat.Opp()]
	for c, ok := p.City.Next(1); ok; c, ok = p.City.Next(c + 1) {
		if cardColor(c) != catalog.Purple {
			continue
		}
		slug, ok := dtoCard(c)
		if !ok {
			continue
		}
		counts := seat
		if d := catalog.Card(c); d != nil {
		effects:
			for _, e := range d.Effects {
				var own, opp int
				switch e.Kind {
				case catalog.EffPointsPerCard:
					own, opp = colorCount(p, e.Colors), colorCount(o, e.Colors)
				case catalog.EffPointsPerWonder:
					own, opp = builtWonders(p), builtWonders(o)
				case catalog.EffPointsPerCoins:
					own, opp = int(p.Coins), int(o.Coins)
				default:
					continue
				}
				if e.Scope == catalog.ScopeMaxCity && opp > own {
					counts = seat.Opp()
				}
				break effects // the first effect that counts names the city
			}
		}
		vp := int(cardEndPoints(p, o, c))
		if int(c) < len(catalog.T.VP) {
			vp += int(catalog.T.VP[c])
		}
		out = append(out, view.GuildPointsView{Card: slug, VP: vp, CountsSeat: int(counts)})
	}
	return out
}

// --- science ---

// pairCards returns the two green cards of a card symbol, in ascending
// order; the scales (Law only) have none.
func pairCards(sym catalog.Symbol) (cards [2]catalog.CardID, n int) {
	set := catalog.T.ByColor[catalog.Green]
	for c, ok := set.Next(1); ok; c, ok = set.Next(c + 1) {
		if catalog.T.Sym[c] == sym && n < len(cards) {
			cards[n] = c
			n++
		}
	}
	return cards, n
}

// mayComeTo reports whether card c, at public place pl, may still come
// into seat's city: face up or unseen (it may be taken), or in the discard
// pile while seat's Mausoleum is open or seat is choosing the card it
// rebuilds (it may be rebuilt).
func mayComeTo(s *State, seat Seat, pl cardPlace) bool {
	switch pl.Place {
	case PlaceFaceUp, PlaceUnseenThisAge, PlaceUnseenLaterAge:
		return true
	case PlaceDiscard:
		return mausoleumOpen(s, seat) || (s.Phase == PhaseBuildFromDiscard && s.ToMove == seat)
	}
	return false
}

// unseenOdds is the chance that a card unseen at place pl is in play: for
// the current age, the face-down slots of its back among the unseen cards
// of that age; for a later age, the cards dealt among the age's cards. Nil
// for a card that is not unseen.
func unseenOdds(s *State, c catalog.CardID, pl PublicPlace) *view.Odds {
	age := catalog.T.Age[c]
	if catalog.T.Color[c] == catalog.Purple || age < 1 || age > NumAges {
		return nil
	}
	switch pl {
	case PlaceUnseenThisAge:
		back := catalog.AgeBack(age)
		f, n := 0, 0
		for i := range s.Slots {
			if sl := &s.Slots[i]; sl.Card != 0 && !sl.FaceUp && sl.Back == back {
				f++
			}
		}
		set := catalog.T.AgeCards[age]
		for d, ok := set.Next(1); ok; d, ok = set.Next(d + 1) {
			if publicPlace(s, d).Place == PlaceUnseenThisAge {
				n++
			}
		}
		if n == 0 {
			return nil
		}
		return &view.Odds{Num: f, Den: n}
	case PlaceUnseenLaterAge:
		n := catalog.T.AgeCards[age].Count()
		return &view.Odds{Num: n - RemovedPerAge, Den: n}
	}
	return nil
}

// sciencePairs lists the six card symbols, each with its two cards and
// their public places.
func sciencePairs(s *State) []view.SciencePairView {
	out := make([]view.SciencePairView, 0, catalog.NumSymbols-2)
	for sym := catalog.Globe; sym < catalog.NumSymbols; sym++ {
		cards, n := pairCards(sym)
		if n == 0 {
			continue
		}
		pv := view.SciencePairView{Symbol: view.ScienceSymbol(sym.String()), Cards: make([]view.PairCardView, 0, n)}
		for _, c := range cards[:n] {
			slug, ok := dtoCard(c)
			if !ok {
				continue
			}
			pl := publicPlace(s, c)
			cv := view.PairCardView{Card: slug, Place: view.PublicPlace(pl.Place.String())}
			switch pl.Place {
			case PlaceCity, PlaceTucked:
				cv.Seat = dtoInt(int(pl.Seat))
			case PlaceFaceUp:
				cv.Slot = dtoInt(pl.Slot)
			case PlaceUnseenThisAge, PlaceUnseenLaterAge:
				cv.InPlay = unseenOdds(s, c, pl.Place)
			}
			pv.Cards = append(pv.Cards, cv)
		}
		out = append(out, pv)
	}
	return out
}

// lawMayCome reports whether seat may still take Law: from the board
// through a pair it may still complete (live: the other card of a single,
// or both cards of a symbol it lacks, may still come), or, when Law is on no
// board and in no city (so in the box or an open offer, by elimination),
// through seat's own open Great Library.
func lawMayCome(s *State, seat Seat, live bool) bool {
	switch {
	case s.Players[0].Tokens.Has(catalog.TokenLaw) || s.Players[1].Tokens.Has(catalog.TokenLaw):
		return false
	case s.BoardTokens.Has(catalog.TokenLaw):
		return live
	}
	return greatLibraryOpen(s, seat)
}

// scienceHints fills the science part of seat's hints: each symbol's
// state, the symbols still needed, whether enough may still come, and
// whether Law would be the sixth symbol (military-science.md "Science
// symbols", "Pairs", "Scientific supremacy"; wonders-tokens.md "The
// Mausoleum", "The Great Library"). A missing card symbol is open while one
// of its cards may still come to the seat; the scales while Law may.
func scienceHints(s *State, seat Seat, ph *view.PlayerHintsView) {
	p := &s.Players[seat]
	ph.Symbols = make([]view.SymbolHint, 0, catalog.NumSymbols-1)
	live := false
	var state [catalog.NumSymbols]view.SymbolState
	var open [catalog.NumSymbols]int
	for sym := catalog.Globe; sym < catalog.NumSymbols; sym++ {
		if sym == catalog.Scales {
			continue
		}
		cards, n := pairCards(sym)
		for _, c := range cards[:n] {
			if !p.City.Has(c) && mayComeTo(s, seat, publicPlace(s, c)) {
				open[sym]++
			}
		}
		// A pair the seat may still complete: the other card of a single,
		// or both cards of a symbol it lacks.
		if int(p.Sci[sym])+open[sym] >= 2 && p.Sci[sym] < 2 {
			live = true
		}
		switch have := p.Sci[sym]; {
		case have >= 2:
			state[sym] = view.SymbolStatePaired
		case have == 1 && open[sym] > 0:
			state[sym] = view.SymbolStateLive
		case have == 1:
			state[sym] = view.SymbolStateOwned
		case open[sym] > 0:
			state[sym] = view.SymbolStateOpen
		default:
			state[sym] = view.SymbolStateDead
		}
	}
	switch {
	case p.Sci[catalog.Scales] > 0:
		state[catalog.Scales] = view.SymbolStateOwned
	case lawMayCome(s, seat, live):
		state[catalog.Scales] = view.SymbolStateOpen
		open[catalog.Scales] = 1
	default:
		state[catalog.Scales] = view.SymbolStateDead
	}
	reach := 0
	for sym := catalog.Globe; sym < catalog.NumSymbols; sym++ {
		ph.Symbols = append(ph.Symbols, view.SymbolHint{Symbol: view.ScienceSymbol(sym.String()), State: state[sym], OpenCards: open[sym]})
		if state[sym] == view.SymbolStateOpen {
			reach++
		}
	}
	distinct := DistinctSymbols(p)
	ph.Need = max(ScienceSupremacy-distinct, 0)
	ph.Reachable = ph.Need > 0 && reach >= ph.Need
	ph.LawWins = distinct == ScienceSupremacy-1 && p.Sci[catalog.Scales] == 0 && live &&
		s.BoardTokens.Has(catalog.TokenLaw)
}

// --- military ---

// toCapital is how many more shields seat's opponent needs to take seat's
// capital: 9 minus the pawn's progress toward it.
func toCapital(s *State, seat Seat) int {
	toward := int(s.Military) // toward seat 1's capital
	if seat == 0 {
		toward = -toward
	}
	return max(MilitaryCapital-toward, 0)
}

// shieldsStillPossible is an upper bound of the shields seat may still
// place (military-science.md "Where shields come from"): its open shield
// wonders, and each red building that may still come to it, plus the
// extra shield of Strategy unless the other seat owns it. It reads public
// places only and never names the box.
func shieldsStillPossible(s *State, seat Seat) int {
	n := 0
	p := &s.Players[seat]
	for k := range p.Wonders {
		if wonderOpen(s, seat, k) && int(p.Wonders[k].ID) < len(catalog.T.WShields) {
			n += int(catalog.T.WShields[p.Wonders[k].ID])
		}
	}
	extra := 0
	for t := catalog.TokenID(1); t <= catalog.NumTokens; t++ {
		d := catalog.Token(t)
		if d == nil || s.Players[seat.Opp()].Tokens.Has(t) {
			continue
		}
		if e, ok := catalog.FindEffect(d.Effects, catalog.EffExtraShield); ok {
			extra = max(extra, int(e.N))
		}
	}
	set := catalog.T.ByColor[catalog.Red]
	for c, ok := set.Next(1); ok; c, ok = set.Next(c + 1) {
		if catalog.T.Shields[c] == 0 || !mayComeTo(s, seat, publicPlace(s, c)) {
			continue
		}
		n += int(catalog.T.Shields[c]) + extra
	}
	return n
}

// lootHints lists the looting tokens still on the track, in track order,
// with what each would take from its victim now.
func lootHints(s *State) []view.LootHint {
	out := make([]view.LootHint, 0, len(lootTokens))
	for _, i := range viewLootOrder {
		t := &lootTokens[i]
		if s.Looting&t.bit == 0 {
			continue
		}
		out = append(out, view.LootHint{At: int(t.at), WouldLose: int(min(t.value, s.Players[t.victim].Coins))})
	}
	return out
}

// --- threats ---

// threatsGated reports whether the threats one action ahead are computed in
// s: only in a play turn and while a new age's starter is chosen (the
// hint-phase-gate). In a pending choice the cards a turn freed may be
// accessible but still face down, and in a Great Library offer the legal
// actions are the secret offer.
func threatsGated(s *State) bool {
	return !s.Over() && (s.Phase == PhasePlay || s.Phase == PhaseChooseStarter) && s.Age >= 1
}

// seatThreats lists the wins seat would have with one action if it were its
// turn now: WinsNow on a normalised copy with every card outside the
// face-up mask scrubbed, using only cards of the mask. It reports false
// when the budget ran out.
func seatThreats(s *State, seat Seat, c *simCtx) (threatList, bool) {
	mask := FaceUpSlots(s)
	y := normalised(s, seat)
	y = scrubbed(&y, mask)
	return threatsFor(&y, seat, mask, c, false)
}

// --- the seat's hints ---

// playerHints computes the board-level hints about seat.
func playerHints(s *State, seat Seat, f view.HintSettings, c *simCtx) view.PlayerHintsView {
	ph := view.PlayerHintsView{Symbols: []view.SymbolHint{}, ChainTargets: []view.ChainTargetView{}, GuildPoints: []view.GuildPointsView{}}
	p := &s.Players[seat]
	if f.Scoring {
		ph.CoinsToNextVp = coinsToNextVp(p)
		ph.GuildPoints = guildPoints(s, seat)
	}
	if f.Science {
		scienceHints(s, seat, &ph)
	}
	if f.Military {
		ph.ToCapital = toCapital(s, seat)
		ph.CapitalSafe = shieldsStillPossible(s, seat.Opp()) < ph.ToCapital
	}
	if f.Counting {
		ph.ChainTargets = chainTargets(s, seat)
	}
	if f.Threats && threatsGated(s) {
		if list, ok := seatThreats(s, seat, c); ok {
			ph.Threats = viewThreats(seat, &list)
		}
	}
	return ph
}

// chainTargets lists, for each chain symbol seat's city shows, the card it
// makes free and its public place, with what the card would cost the seat
// without the chain today while it may still come (economy.md "Chains").
// A target in the discard pile is dead for the chain: a Mausoleum rebuild
// is free anyway and is not a chain build.
func chainTargets(s *State, seat Seat) []view.ChainTargetView {
	out := []view.ChainTargetView{}
	p := &s.Players[seat]
	for ch := catalog.Chain(1); ch < catalog.NumChains; ch++ {
		if p.Chains&ch.Bit() == 0 {
			continue
		}
		target := catalog.T.ChainTarget[ch]
		slug, ok := dtoCard(target)
		if !ok {
			continue
		}
		pl := publicPlace(s, target)
		tv := view.ChainTargetView{Chain: view.Chain(ch.String()), Target: slug, Place: view.PublicPlace(pl.Place.String())}
		switch pl.Place {
		case PlaceCity, PlaceTucked:
			tv.Seat = dtoInt(int(pl.Seat))
		case PlaceUnseenThisAge, PlaceUnseenLaterAge:
			tv.InPlay = unseenOdds(s, target, pl.Place)
		}
		if pl.Place == PlaceFaceUp || pl.Place == PlaceUnseenThisAge || pl.Place == PlaceUnseenLaterAge {
			x := *s
			x.Players[seat].Chains &^= ch.Bit()
			tv.Worth = dtoInt(int(BuildingQuote(&x, seat, target).Total))
		}
		out = append(out, tv)
	}
	return out
}

// --- wonders ---

// wonderEffect is what building seat's open wonder w would do now, as a
// view.
func wonderEffect(s *State, seat Seat, w catalog.WonderID, f view.HintSettings) view.WonderEffectView {
	p := previewWonder(s, seat, w)
	e := view.WonderEffectView{Paid: p.Paid, ExtraTurn: view.ExtraTurnSource(p.ExtraTurn.String()), Pool: p.Pool}
	for i := range e.Coins {
		e.Coins[i] = view.Amount(p.CoinsDelta[i])
		e.Score[i] = view.Amount(p.ScoreDelta[i])
	}
	if f.Military && p.Moved {
		m := &view.PawnMoveView{From: int(p.Pawn.From), To: int(p.Pawn.To), Loot: make([]view.LootTaken, 0, p.Pawn.NLoot)}
		for _, l := range p.Pawn.Loot[:p.Pawn.NLoot] {
			m.Loot = append(m.Loot, view.LootTaken{At: int(l.At), Lost: int(l.Lost)})
		}
		e.Pawn = m
	}
	if f.Threats && p.Wins {
		e.Wins = viewWin(p.Win)
	}
	if p.Removes != 0 {
		e.Removes = dtoWonderPtr(p.Removes)
	}
	if p.Opens != PhasePlay {
		if k, ok := pendingKind(p.Opens); ok {
			e.Opens = &k
		}
	}
	return e
}

// greatLibraryOpen reports whether seat has an open wonder that draws
// tokens from the box (the Great Library).
func greatLibraryOpen(s *State, seat Seat) bool {
	p := &s.Players[seat]
	for k := range p.Wonders {
		w := p.Wonders[k].ID
		if int(w) < len(catalog.T.WDraw) && catalog.T.WDraw[w] > 0 && wonderOpen(s, seat, k) {
			return true
		}
	}
	return false
}

// --- tokens ---

// boxTokenNames names the tokens in the box (and in an open Great Library
// offer) by elimination: every token on no board and in no city, in id
// order. The order of the box stays secret (owner answer Q1 B).
func boxTokenNames(s *State) []string {
	out := []string{}
	for t := catalog.TokenID(1); t <= catalog.NumTokens; t++ {
		if s.BoardTokens.Has(t) || s.Players[0].Tokens.Has(t) || s.Players[1].Tokens.Has(t) {
			continue
		}
		if slug, ok := dtoToken(t); ok {
			out = append(out, slug)
		}
	}
	return out
}

// boardTokenValues is what each board token would do for each seat now, by
// token, then seat.
func boardTokenValues(s *State) []view.TokenValueView {
	out := []view.TokenValueView{}
	for t, ok := s.BoardTokens.Next(1); ok; t, ok = s.BoardTokens.Next(t + 1) {
		for seat := range Seat(2) {
			if v, ok := tokenValueView(s, seat, t); ok {
				out = append(out, v)
			}
		}
	}
	return out
}

// tokenValueView is what taking token t would do for seat now
// (token-value-now): the coins, the points and a win from tokenPreview, and
// the token's own measure: what a discount (Masonry, Architecture) saves on
// the cards in view and the seat's open wonders, the trade coins the other
// seat pays now (Economy), the chains still live (Urbanism), the open
// wonders that would gain play again (Theology), and the tokens the seat
// would own (Mathematics).
func tokenValueView(s *State, seat Seat, t catalog.TokenID) (view.TokenValueView, bool) {
	slug, ok := dtoToken(t)
	d := catalog.Token(t)
	if !ok || d == nil || s.Players[seat].Tokens.Has(t) {
		return view.TokenValueView{}, false
	}
	pv := tokenPreview(s, seat, t)
	out := view.TokenValueView{Token: slug, Seat: int(seat), Coins: pv.Coins, VP: pv.VP, Wins: pv.Wins}
	p := &s.Players[seat]
	for _, e := range d.Effects {
		switch e.Kind {
		case catalog.EffDiscount:
			x := normalised(s, seat)
			addToken(&x, seat, t)
			ids, n := faceUpTargets(s)
			for _, c := range ids[:n] {
				if !p.City.Has(c) {
					out.Savings += int(BuildingQuote(s, seat, c).Total) - int(BuildingQuote(&x, seat, c).Total)
				}
			}
			for k := range p.Wonders {
				if wonderOpen(s, seat, k) {
					w := p.Wonders[k].ID
					out.Savings += int(WonderQuote(s, seat, w).Total) - int(WonderQuote(&x, seat, w).Total)
				}
			}
		case catalog.EffTradeIncome:
			opp := seat.Opp()
			acc := Accessible(s) & FaceUpSlots(s)
			for i := range s.Slots {
				if acc&(1<<i) != 0 {
					if q := BuildingPayment(s, opp, s.Slots[i].Card); q.Affordable {
						out.Savings += int(q.TradeTotal)
					}
				}
			}
		case catalog.EffChainBonus:
			for ch := catalog.Chain(1); ch < catalog.NumChains; ch++ {
				if p.Chains&ch.Bit() != 0 {
					if pl := publicPlace(s, catalog.T.ChainTarget[ch]); pl.Place == PlaceFaceUp || pl.Place == PlaceUnseenThisAge || pl.Place == PlaceUnseenLaterAge {
						out.Count++
					}
				}
			}
		case catalog.EffWondersPlayAgain:
			for k := range p.Wonders {
				w := p.Wonders[k].ID
				if wonderOpen(s, seat, k) && extraTurnSource(&Player{}, w) == ExtraTurnSourceNone {
					out.Count++
				}
			}
		case catalog.EffPointsPerToken:
			out.Count = p.Tokens.Count() + 1
		}
	}
	return out, true
}

// libraryTokenValues is what each token of the open Great Library offer
// would do for the seat choosing from it. Only that seat's view carries it,
// where the offer itself already appears.
func libraryTokenValues(s *State) []view.TokenValueView {
	out := []view.TokenValueView{}
	for _, t := range s.Library {
		if t == 0 {
			continue
		}
		if v, ok := tokenValueView(s, s.ToMove, t); ok {
			out = append(out, v)
		}
	}
	return out
}

// --- counting ---

// unseenPools lists, for each back face down in the structure, the cards
// that may lie under it (facedown-candidates, guild-pool): for an age's
// back, the age's cards (guilds aside) whose public place is unseen this
// age; for the guild back, the guilds in no public place, which always
// include the guilds never put in the game (setup.md "What is hidden from
// both players"; decisions.md "Guild back").
func unseenPools(s *State) []view.UnseenView {
	out := []view.UnseenView{}
	if s.Age < 1 || s.Age > NumAges {
		return out
	}
	var down [catalog.BackGuild + 1]int
	for i := range s.Slots {
		if sl := &s.Slots[i]; sl.Card != 0 && !sl.FaceUp && int(sl.Back) < len(down) {
			down[sl.Back]++
		}
	}
	add := func(back catalog.Back, set catalog.CardSet) {
		if down[back] == 0 {
			return
		}
		u := view.UnseenView{Back: *dtoBack(back), Cards: []string{}, FaceDown: down[back]}
		for c, ok := set.Next(1); ok; c, ok = set.Next(c + 1) {
			if pl := publicPlace(s, c).Place; pl == PlaceUnseenThisAge {
				if slug, ok := dtoCard(c); ok {
					u.Cards = append(u.Cards, slug)
				}
			}
		}
		u.SetAside = max(len(u.Cards)-u.FaceDown, 0)
		out = append(out, u)
	}
	add(catalog.AgeBack(s.Age), catalog.T.AgeCards[s.Age])
	if s.Age == NumAges {
		add(catalog.GuildBack, catalog.T.Guilds)
	}
	return out
}

// guildPreview is what building each guild not yet seen would bring each
// seat now (guild-pool): a free build on a copy, never a real one. The
// candidates are every guild in no public place, the box guilds among
// them, so the list never tells which guilds are in the game.
func guildPreview(s *State) []view.GuildPreviewView {
	out := []view.GuildPreviewView{}
	set := catalog.T.Guilds
	for c, ok := set.Next(1); ok; c, ok = set.Next(c + 1) {
		if pl := publicPlace(s, c).Place; pl != PlaceUnseenThisAge && pl != PlaceFaceUp {
			continue
		}
		slug, ok := dtoCard(c)
		if !ok {
			continue
		}
		g := view.GuildPreviewView{Guild: slug}
		for seat := range Seat(2) {
			p := previewBuild(s, seat, c, true)
			g.Coins[seat] = view.Amount(p.Coins)
			g.VP[seat] = view.Amount(p.ScoreDelta[seat])
		}
		out = append(out, g)
	}
	return out
}
