package engine

import (
	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
	"github.com/NightMachinery/ajayeb/server/internal/view"
)

// This file builds the view a client receives: ViewFor copies into a fresh
// view.View exactly what one viewer may see. It is a whitelist: every field
// of the DTO is filled here from a part of the state that is public, or
// public to that viewer, and nothing is copied wholesale, so a field nobody
// thought about cannot leak. The rules of what is hidden are in the view
// package's documentation; in short:
//
//   - face-down slots show only their back, never their card or its cost;
//   - the removed cards, the later ages' deals, the box guilds, the box
//     wonders, the second draft round (until the first is over) and the box
//     tokens show at most their number, until the game is over;
//   - the Great Library offer is listed only for the seat deciding it;
//   - the legal choices go only to the seat to move;
//   - the seed and the scenario-only OutOfPlay cards never appear.
//
// The view never aliases the state: it holds slugs, numbers and freshly
// allocated slices only. Lists are never nil, so JSON has no null arrays.

// Viewer is who a view or an event list is for: seat 0, seat 1 or a
// spectator.
type Viewer int8

// Spectator is the viewer without a seat. Any value other than 0 and 1 is
// treated as a spectator.
const Spectator Viewer = -1

// SeatViewer returns the viewer who holds seat s.
func SeatViewer(s Seat) Viewer { return Viewer(s) }

// Seat returns the viewer's seat, and false for a spectator.
func (v Viewer) Seat() (Seat, bool) {
	if v == 0 || v == 1 {
		return Seat(v), true
	}
	return 0, false
}

// ViewFor builds the view of s for viewer v, without hints. In a hot-seat
// game the room asks for the view of the seat to move. It is
// ViewWithHints(s, v, nil): the simulator, the tests and replays use it,
// and only the room, which knows the game's hint settings, adds hints.
func ViewFor(s *State, v Viewer) view.View { return ViewWithHints(s, v, nil) }

// ViewWithHints builds the view of s for viewer v, like ViewFor, and adds
// the hints h (ComputeHints(s, settings), computed once per state and
// shared by every viewer). A nil h, or a game that is over, adds none.
func ViewWithHints(s *State, v Viewer, h *Hints) view.View {
	seat, seated := v.Seat()
	decider := seated && !s.Over() && seat == s.ToMove
	out := view.View{
		Schema:        view.SchemaVersion,
		CatalogHash:   CatalogHash(),
		Ply:           int(s.Ply),
		Phase:         view.Phase(s.Phase.String()),
		Age:           int(s.Age),
		Active:        int(s.Active),
		FirstPlayer:   int(s.FirstSeat),
		ExtraTurn:     s.ExtraTurn,
		PlayingAgain:  s.PlayingAgain && !s.Over(),
		Structure:     viewStructure(s),
		Military:      viewMilitary(s),
		BoardTokens:   dtoTokenSet(s.BoardTokens),
		BoxTokenCount: dtoCountTokens(s.BoxTokens[:]),
		Discard:       dtoCardSet(s.Discard),
		WondersBuilt:  int(s.WondersBuilt),
		Pending:       viewPending(s, decider),
		Choices:       []view.Choice{},
		Disabled:      []view.DisabledChoice{},
		ScorePreview:  dtoScores(Scores(s)),
	}
	if seated {
		out.Viewer = dtoInt(int(seat))
	}
	if !s.Over() {
		out.ToMove = dtoInt(int(s.ToMove))
	}
	if s.Age >= 1 && s.Age <= NumAges {
		out.RemovedCount = dtoCountCards(s.Removed[s.Age-1][:])
	}
	if s.Phase == PhaseWonderDraft {
		out.Draft = viewDraft(s)
	}
	for i := range out.Players {
		out.Players[i] = viewPlayer(s, Seat(i))
	}
	if decider {
		out.Choices = viewChoices(s)
		if s.Phase == PhasePlay {
			out.Disabled = viewDisabled(s)
		}
	}
	if s.Over() {
		out.Result = viewResult(&s.Result)
		if !s.Opts.HideAtEnd {
			out.Revealed = viewRevealed(s)
		}
	}
	if h != nil && !s.Over() {
		addHints(s, v, h, &out)
	}
	out.CardTooltips = viewCardTooltips(s, &out)
	return out
}

// viewDraft returns the current draft round: its four wonders and who took
// each. The other round's wonders are not shown: those of round 2 are still
// hidden during round 1, and those of round 1 are in the players' hands.
func viewDraft(s *State) *view.DraftView {
	n := draftPicks(s)
	if n >= NumDraftWonders {
		return nil
	}
	round := draftRound(n)
	d := &view.DraftView{Round: round + 1, Offer: make([]view.DraftOfferView, 0, 4), Order: make([]int, 0, 4)}
	for i := 4 * round; i < 4*round+4; i++ {
		d.Order = append(d.Order, int(draftPicker(s.FirstSeat, i)))
	}
	for i := 4 * round; i < 4*round+4; i++ {
		w := s.DraftOrder[i]
		slug, ok := dtoWonder(w)
		if !ok {
			continue
		}
		o := view.DraftOfferView{Wonder: slug}
		if s.DraftTaken&(1<<i) != 0 {
			for seat := range s.Players {
				if wonderSlot(&s.Players[seat], w) >= 0 {
					o.TakenBy = dtoInt(seat)
				}
			}
		}
		d.Offer = append(d.Offer, o)
	}
	return d
}

// viewStructure returns the 20 slots of the current age, or none during the
// draft. A face-down card shows only its back; every face-up card,
// accessible or still covered, carries both seats' costs.
func viewStructure(s *State) []view.SlotView {
	out := []view.SlotView{}
	if s.Age < 1 || s.Age > NumAges {
		return out
	}
	occupied := Occupied(s)
	out = make([]view.SlotView, 0, SlotsPerAge)
	for i := range s.Slots {
		sl := &s.Slots[i]
		pos := SlotPos(s.Age, i)
		sv := view.SlotView{Index: i, Row: int(pos.Row), Col: int(pos.Col), State: view.SlotStateEmpty, CoveredBy: []int{}}
		if sl.Card != 0 {
			cover := CoverMask(s.Age, i) & occupied
			for j := range SlotsPerAge {
				if cover&(1<<j) != 0 {
					sv.CoveredBy = append(sv.CoveredBy, j)
				}
			}
			sv.Accessible = cover == 0
			sv.Back = dtoBack(sl.Back)
			sv.State = view.SlotStateFaceDown
			if sl.FaceUp {
				sv.State = view.SlotStateFaceUp
				sv.Card = dtoCardPtr(sl.Card)
				// Every face-up card, covered or not, carries both seats'
				// costs as if it were accessible now: face-up cards and
				// both cities are public, and the card sheet says what each
				// player would pay before the card is free.
				if sv.Card != nil {
					sv.Costs = &[2]view.CostView{viewBuildingCost(s, 0, sl.Card), viewBuildingCost(s, 1, sl.Card)}
				}
			}
		}
		out = append(out, sv)
	}
	return out
}

// viewBuildingCost is seat's cost of building card c now.
func viewBuildingCost(s *State, seat Seat, c catalog.CardID) view.CostView {
	pay := BuildingPayment(s, seat, c)
	return viewCost(s, seat, &pay)
}

// viewWonderCost is seat's cost of building wonder w now.
func viewWonderCost(s *State, seat Seat, w catalog.WonderID) view.CostView {
	pay := WonderPayment(s, seat, w)
	return viewCost(s, seat, &pay)
}

// viewCost converts a payment of seat to its DTO.
func viewCost(s *State, seat Seat, p *Payment) view.CostView {
	cv := view.CostView{
		Affordable:    p.Affordable,
		ChainFree:     p.ChainFree,
		ChainFrom:     dtoCardPtr(p.ChainFrom),
		PrintedCoins:  int(p.PrintedCoins),
		Own:           dtoResources(p.Own),
		Producers:     []view.ProducerUse{},
		Trade:         []view.TradeView{},
		TradeTotal:    int(p.TradeTotal),
		Total:         int(p.Total),
		ToOpponent:    int(p.ToOpponent),
		UrbanismBonus: int(p.UrbanismBonus),
	}
	if p.Affordable {
		cv.CoinsAfter = dtoInt(int(s.Players[seat].Coins) - int(p.Total))
	} else {
		cv.Shortfall = max(0, int(p.Total)-int(s.Players[seat].Coins))
	}
	for k := range min(int(p.NChoice), MaxChoice) {
		u := &p.Choice[k]
		cv.Producers = append(cv.Producers, view.ProducerUse{
			Card: dtoCardPtr(u.Card), Wonder: dtoWonderPtr(u.Wonder), Resource: dtoResource(u.Res),
		})
	}
	if slug, ok := dtoToken(p.DiscountBy); ok {
		cv.Discount = &view.DiscountView{Token: slug, Units: dtoResources(p.Discount)}
	}
	for r := range catalog.NumResources {
		if n := int(p.TradeUnits[r]); n > 0 {
			cv.Trade = append(cv.Trade, view.TradeView{
				Resource: dtoResource(r), Units: n, UnitPrice: int(p.UnitPrice[r]),
				Fixed: p.FixedPrice.Has(r), Coins: n * int(p.UnitPrice[r]),
			})
		}
	}
	return cv
}

// viewPlayer returns one seat's city, purse, wonders, production and the
// numbers derived from them.
func viewPlayer(s *State, seat Seat) view.PlayerView {
	p := &s.Players[seat]
	pv := view.PlayerView{
		Seat:         int(seat),
		Coins:        int(p.Coins),
		City:         dtoCardSet(p.City),
		Tokens:       dtoTokenSet(p.Tokens),
		Wonders:      make([]view.WonderView, 0, WondersPerPlayer),
		Production:   viewProduction(p),
		TradePrices:  dtoResources(TradePrices(s, seat)),
		Science:      viewScience(p),
		Chains:       []view.Chain{},
		DiscardValue: int(DiscardValue(s, seat)),
		Shields:      int(s.Shields[seat]),
	}
	for ch := catalog.Chain(1); ch < catalog.NumChains; ch++ {
		if p.Chains&ch.Bit() != 0 {
			pv.Chains = append(pv.Chains, view.Chain(ch.String()))
		}
	}
	for k := range p.Wonders {
		ws := &p.Wonders[k]
		slug, ok := dtoWonder(ws.ID)
		if !ok {
			continue
		}
		wv := view.WonderView{Wonder: slug, Built: ws.Built, Removed: ws.Removed}
		if ws.Built {
			wv.Tucked = dtoCardPtr(ws.Tucked)
		}
		if wonderOpen(s, seat, k) {
			cv := viewWonderCost(s, seat, ws.ID)
			wv.Cost = &cv
		}
		pv.Wonders = append(pv.Wonders, wv)
	}
	return pv
}

// viewProduction returns a player's fixed production, flexible producers
// (in the canonical order of cost.go) and fixed prices.
func viewProduction(p *Player) view.ProductionView {
	pv := view.ProductionView{Fixed: dtoResources(p.Prod), Producers: []view.Producer{}, TradeFix: dtoResourceMask(p.TradeFix)}
	srcs, n := choiceSources(p)
	for i := range n {
		pv.Producers = append(pv.Producers, view.Producer{
			Card: dtoCardPtr(srcs[i].card), Wonder: dtoWonderPtr(srcs[i].wonder), Resources: dtoResourceMask(srcs[i].mask),
		})
	}
	return pv
}

// viewScience returns a player's science symbols.
func viewScience(p *Player) view.ScienceView {
	sv := view.ScienceView{Symbols: []view.SymbolCount{}, Distinct: DistinctSymbols(p)}
	for sym := catalog.Globe; sym < catalog.NumSymbols; sym++ {
		if n := int(p.Sci[sym]); n > 0 {
			sv.Symbols = append(sv.Symbols, view.SymbolCount{Symbol: view.ScienceSymbol(sym.String()), Count: n})
		}
	}
	return sv
}

// viewLootOrder lists the looting tokens (indices into lootTokens) in track
// order: -6, -3, 3, 6.
var viewLootOrder = [4]int{3, 2, 0, 1}

// viewMilitary returns the pawn and the four looting tokens.
func viewMilitary(s *State) view.MilitaryView {
	mv := view.MilitaryView{Position: int(s.Military), Looting: make([]view.LootingView, 0, len(lootTokens))}
	for _, i := range viewLootOrder {
		t := &lootTokens[i]
		mv.Looting = append(mv.Looting, view.LootingView{
			At: int(t.at), Seat: int(t.victim), Value: int(t.value), Present: s.Looting&t.bit != 0,
		})
	}
	return mv
}

// viewPending returns the open starter choice or pending choice, or nil.
// The Great Library offer is listed only when decider is set (the viewer is
// the seat deciding it); everyone else gets only its size.
func viewPending(s *State, decider bool) *view.PendingView {
	kind, ok := pendingKind(s.Phase)
	if !ok {
		return nil
	}
	pv := &view.PendingView{Kind: kind, Seat: int(s.ToMove), Cards: []string{}, Tokens: []string{}}
	switch s.Phase {
	case PhaseDestroyCard:
		pv.Color = dtoColor(s.Destroy)
		opp := &s.Players[s.ToMove.Opp()]
		for c, ok := opp.City.Next(1); ok; c, ok = opp.City.Next(c + 1) {
			if cardColor(c) == s.Destroy {
				if slug, ok := dtoCard(c); ok {
					pv.Cards = append(pv.Cards, slug)
				}
			}
		}
	case PhaseBuildFromDiscard:
		pv.Cards = dtoCardSet(s.Discard)
	case PhasePickBoardToken:
		pv.Tokens = dtoTokenSet(s.BoardTokens)
	case PhasePickLibraryToken:
		for _, t := range s.Library {
			if t == 0 {
				continue
			}
			if !decider {
				pv.HiddenTokens++
			} else if slug, ok := dtoToken(t); ok {
				pv.Tokens = append(pv.Tokens, slug)
			}
		}
	}
	return pv
}

// viewChoices returns the legal actions of the seat to move, in the order
// of LegalActions, with the cost of each construction.
func viewChoices(s *State) []view.Choice {
	legal := LegalActions(s, make([]Action, 0, MaxLegalActions))
	out := make([]view.Choice, 0, len(legal))
	seat := s.ToMove
	var wonderCost [catalog.NumWonders + 1]*view.CostView // computed once per wonder, shared by its choices
	for _, a := range legal {
		key := a.Key()
		if key == "" {
			continue
		}
		c := view.Choice{Key: key, Kind: view.ChoiceKind(a.Type.Verb())}
		switch a.Type {
		case ActDraftWonder:
			c.Wonder = dtoWonderPtr(a.Wonder)
		case ActConstructBuilding:
			c.Card = dtoCardPtr(a.Card)
			cv := viewBuildingCost(s, seat, a.Card)
			c.Cost = &cv
		case ActDiscardBuilding:
			c.Card = dtoCardPtr(a.Card)
			c.Gain = dtoInt(int(DiscardValue(s, seat)))
		case ActConstructWonder:
			c.Wonder = dtoWonderPtr(a.Wonder)
			c.Card = dtoCardPtr(a.Card)
			if int(a.Wonder) < len(wonderCost) {
				if wonderCost[a.Wonder] == nil {
					cv := viewWonderCost(s, seat, a.Wonder)
					wonderCost[a.Wonder] = &cv
				}
				c.Cost = wonderCost[a.Wonder]
			}
		case ActDestroyCard:
			c.Card = dtoCardPtr(a.Card)
		case ActBuildFromDiscard:
			c.Card = dtoCardPtr(a.Card)
			// The Mausoleum's actual action is free, independent of the
			// public hypothetical building quote in CardTooltips.
			pay := Payment{Affordable: true}
			cv := viewCost(s, seat, &pay)
			c.Cost = &cv
		case ActChooseProgressToken:
			c.Token = dtoTokenPtr(a.Token)
		case ActChooseStarter:
			c.Starter = dtoInt(int(a.Starter))
		}
		out = append(out, c)
	}
	return out
}

// viewDisabled returns the play actions the seat to move cannot take now,
// in the order LegalActions would list them: for each accessible face-up
// card, building it, then building each of the seat's open wonders with it.
// Discarding is always possible, so it never appears.
func viewDisabled(s *State) []view.DisabledChoice {
	out := []view.DisabledChoice{}
	seat := s.ToMove
	p := &s.Players[seat]
	acc := Accessible(s)
	var wonderCost [catalog.NumWonders + 1]*view.CostView
	for i := range s.Slots {
		sl := &s.Slots[i]
		if acc&(1<<i) == 0 || !sl.FaceUp {
			continue
		}
		c := sl.Card
		build := ConstructAction(c)
		if r := legality(s, build); r != ReasonNone {
			cv := viewBuildingCost(s, seat, c)
			out = append(out, view.DisabledChoice{
				Key: build.Key(), Kind: view.ChoiceKindConstruct, Card: dtoCardPtr(c),
				Reason: view.Reason(r.String()), Cost: &cv,
			})
		}
		for k := range p.Wonders {
			ws := &p.Wonders[k]
			if ws.ID == 0 || ws.Built || ws.Removed || int(ws.ID) >= len(wonderCost) {
				continue
			}
			a := WonderAction(ws.ID, c)
			r := legality(s, a)
			if r == ReasonNone {
				continue
			}
			if wonderCost[ws.ID] == nil {
				cv := viewWonderCost(s, seat, ws.ID)
				wonderCost[ws.ID] = &cv
			}
			out = append(out, view.DisabledChoice{
				Key: a.Key(), Kind: view.ChoiceKindWonder, Card: dtoCardPtr(c), Wonder: dtoWonderPtr(ws.ID),
				Reason: view.Reason(r.String()), Cost: wonderCost[ws.ID],
			})
		}
	}
	return out
}

// viewResult converts a finished game's result.
func viewResult(r *Result) *view.ResultView {
	rv := &view.ResultView{Victory: view.VictoryKind(r.Kind.String()), Scores: dtoScores(r.Scores)}
	if r.Winner == SharedVictory {
		rv.Shared = true
	} else {
		rv.Winner = dtoInt(int(r.Winner))
	}
	return rv
}

// viewRevealed lists what stayed hidden during the game, for review once it
// is over.
func viewRevealed(s *State) *view.RevealView {
	rv := &view.RevealView{
		Removed:    []view.AgeCards{},
		Undealt:    []view.AgeCards{},
		FaceDown:   []view.SlotCard{},
		BoxGuilds:  dtoCardList(s.BoxGuilds[:]),
		BoxWonders: dtoWonderList(s.BoxWonders[:]),
		BoxTokens:  dtoTokenList(s.BoxTokens[:]),
	}
	for a := range NumAges {
		if cards := dtoCardList(s.Removed[a][:]); len(cards) > 0 {
			rv.Removed = append(rv.Removed, view.AgeCards{Age: a + 1, Cards: cards})
		}
		if cards := dtoCardList(s.Deals[a][:]); len(cards) > 0 {
			rv.Undealt = append(rv.Undealt, view.AgeCards{Age: a + 1, Cards: cards})
		}
	}
	for i := range s.Slots {
		sl := &s.Slots[i]
		if sl.Card == 0 || sl.FaceUp {
			continue
		}
		if slug, ok := dtoCard(sl.Card); ok {
			rv.FaceDown = append(rv.FaceDown, view.SlotCard{Slot: i, Card: slug})
		}
	}
	return rv
}

// The DTO helpers below turn engine values into the view's slugs, names and
// numbers. An id that is not registered (0, or the Unknown sentinel of an
// observation) has no slug: the single helpers report false or return nil,
// and the list helpers skip it.

func dtoInt(n int) *int { return &n }

func dtoCard(c catalog.CardID) (string, bool) {
	if d := catalog.Card(c); d != nil {
		return d.Slug, true
	}
	return "", false
}

func dtoWonder(w catalog.WonderID) (string, bool) {
	if d := catalog.Wonder(w); d != nil {
		return d.Slug, true
	}
	return "", false
}

func dtoToken(t catalog.TokenID) (string, bool) {
	if d := catalog.Token(t); d != nil {
		return d.Slug, true
	}
	return "", false
}

func dtoCardPtr(c catalog.CardID) *string {
	if slug, ok := dtoCard(c); ok {
		return &slug
	}
	return nil
}

func dtoWonderPtr(w catalog.WonderID) *string {
	if slug, ok := dtoWonder(w); ok {
		return &slug
	}
	return nil
}

func dtoTokenPtr(t catalog.TokenID) *string {
	if slug, ok := dtoToken(t); ok {
		return &slug
	}
	return nil
}

// dtoCardSet lists a card set in ascending id order.
func dtoCardSet(set catalog.CardSet) []string {
	out := make([]string, 0, set.Count())
	for c, ok := set.Next(1); ok; c, ok = set.Next(c + 1) {
		if slug, ok := dtoCard(c); ok {
			out = append(out, slug)
		}
	}
	return out
}

// dtoTokenSet lists a token set in ascending id order.
func dtoTokenSet(set catalog.TokenSet) []string {
	out := make([]string, 0, set.Count())
	for t, ok := set.Next(1); ok; t, ok = set.Next(t + 1) {
		if slug, ok := dtoToken(t); ok {
			out = append(out, slug)
		}
	}
	return out
}

// dtoCardList lists the cards of an array in its order, skipping empty
// places.
func dtoCardList(ids []catalog.CardID) []string {
	out := make([]string, 0, len(ids))
	for _, c := range ids {
		if slug, ok := dtoCard(c); ok {
			out = append(out, slug)
		}
	}
	return out
}

func dtoWonderList(ids []catalog.WonderID) []string {
	out := make([]string, 0, len(ids))
	for _, w := range ids {
		if slug, ok := dtoWonder(w); ok {
			out = append(out, slug)
		}
	}
	return out
}

func dtoTokenList(ids []catalog.TokenID) []string {
	out := make([]string, 0, len(ids))
	for _, t := range ids {
		if slug, ok := dtoToken(t); ok {
			out = append(out, slug)
		}
	}
	return out
}

// dtoCountCards counts the occupied places of an array of cards.
func dtoCountCards(ids []catalog.CardID) int {
	n := 0
	for _, c := range ids {
		if c != 0 {
			n++
		}
	}
	return n
}

// dtoCountTokens counts the occupied places of an array of tokens.
func dtoCountTokens(ids []catalog.TokenID) int {
	n := 0
	for _, t := range ids {
		if t != 0 {
			n++
		}
	}
	return n
}

func dtoResource(r catalog.Resource) view.Resource { return view.Resource(r.String()) }

// dtoResourceMask lists the resources of a mask in resource order.
func dtoResourceMask(m catalog.ResMask) []view.Resource {
	out := []view.Resource{}
	for r := range catalog.NumResources {
		if m.Has(r) {
			out = append(out, dtoResource(r))
		}
	}
	return out
}

// dtoResources converts a per-resource array.
func dtoResources(a [catalog.NumResources]uint8) view.ResourceCounts {
	return view.ResourceCounts{
		Wood: int(a[catalog.Wood]), Clay: int(a[catalog.Clay]), Stone: int(a[catalog.Stone]),
		Glass: int(a[catalog.Glass]), Papyrus: int(a[catalog.Papyrus]),
	}
}

// dtoBack returns a card back, or nil for no back.
func dtoBack(b catalog.Back) *view.Back {
	if b == catalog.NoBack || b > catalog.BackGuild {
		return nil
	}
	v := view.Back(b.String())
	return &v
}

// dtoColor returns a building colour, or nil for none.
func dtoColor(c catalog.Color) *view.Color {
	if c == catalog.NoColor || c >= catalog.NumColors {
		return nil
	}
	v := view.Color(c.String())
	return &v
}

func dtoScore(sc Score) view.ScoreView {
	return view.ScoreView{
		Blue: int(sc.Blue), Green: int(sc.Green), Yellow: int(sc.Yellow), Purple: int(sc.Purple),
		Wonders: int(sc.Wonders), Tokens: int(sc.Tokens), Coins: int(sc.Coins), Military: int(sc.Military),
		Total: int(sc.Total),
	}
}

func dtoScores(sc [2]Score) [2]view.ScoreView {
	return [2]view.ScoreView{dtoScore(sc[0]), dtoScore(sc[1])}
}
