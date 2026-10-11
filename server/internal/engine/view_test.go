package engine_test

import (
	"bytes"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
	"github.com/NightMachinery/ajayeb/server/internal/engine/enginetest"
	"github.com/NightMachinery/ajayeb/server/internal/view"
)

// The view tests check what ViewFor builds, field by field, against the
// state it was built from; the redaction tests (redaction_test.go) check
// what it leaves out. This file also writes the two kinds of recorded
// views:
//
//   - testdata/protocol: one file per protocol fixture, a position chosen
//     to show one phase or one kind of event, with the view and the
//     filtered events of seat 0, seat 1 and a spectator. The client's
//     tests and dev pages read them, and every phase occurs in at least one
//     of them.
//   - testdata/views: the views of a plain seeded game at fixed points
//     (the start, each age, the end), as a regression record of the whole
//     view of ordinary positions.
//
// Both are golden files: the test fails when the engine's output differs
// from them, and -update rewrites them after a deliberate change:
//
//	go test ./internal/engine/ -run 'TestProtocolFixtures|TestViewGoldens' -update
//
// Helpers are prefixed vw (the view checks) and proto (the fixtures).

// TestViewEnumsMatchEngine checks that every small vocabulary of the view
// package lists exactly the wire names the engine and the catalog use, in
// the same order. The view constants are read from the source of
// internal/view/enums.go, so a value added on either side and not on the
// other fails here.
func TestViewEnumsMatchEngine(t *testing.T) {
	got := vwViewConsts(t)
	phases := vwEnum(func(i uint8) string { return engine.Phase(i).String() }, 0)
	var pending []string
	for _, p := range phases {
		switch p {
		case "wonderDraft", "play", "gameOver":
		default:
			pending = append(pending, p)
		}
	}
	want := map[string][]string{
		"Phase":         phases,
		"PendingKind":   pending,
		"ChoiceKind":    vwEnum(func(i uint8) string { return engine.ActionType(i).Verb() }, 1),
		"Reason":        vwEnum(func(i uint8) string { return engine.Reason(i).String() }, 1),
		"SlotState":     {"faceUp", "faceDown", "empty"},
		"Back":          vwEnum(func(i uint8) string { return catalog.Back(i).String() }, 1),
		"Resource":      vwEnum(func(i uint8) string { return catalog.Resource(i).String() }, 0),
		"Color":         vwEnum(func(i uint8) string { return catalog.Color(i).String() }, 1),
		"ScienceSymbol": vwEnum(func(i uint8) string { return catalog.Symbol(i).String() }, 1),
		"Chain":         vwEnum(func(i uint8) string { return catalog.Chain(i).String() }, 1),
		"VictoryKind":   vwEnum(func(i uint8) string { return engine.VictoryKind(i).String() }, 1),
		"EventKind":     vwEnum(func(i uint8) string { return engine.EventKind(i).String() }, 1),
	}
	for typ, w := range want {
		if len(w) < 2 {
			t.Errorf("%s: the engine side lists only %v; the enumeration is broken", typ, w)
		}
		if !slices.Equal(got[typ], w) {
			t.Errorf("view.%s is %v, the engine's names are %v", typ, got[typ], w)
		}
	}
	for typ := range got {
		if _, ok := want[typ]; !ok {
			t.Errorf("view.%s has constants this test does not check against the engine; add it", typ)
		}
	}
}

// vwEnum lists name(i) for i from `from` up, stopping at the first value
// that is not a valid one: an empty name, or a "kind#N" fallback.
func vwEnum(name func(uint8) string, from uint8) []string {
	var out []string
	for i := int(from); i < 256; i++ {
		s := name(uint8(i))
		if s == "" || strings.Contains(s, "#") {
			break
		}
		out = append(out, s)
	}
	return out
}

// vwViewConsts parses internal/view/enums.go and returns the values of its
// string constants by type name, in source order.
func vwViewConsts(t *testing.T) map[string][]string {
	t.Helper()
	path := filepath.Join("..", "view", "enums.go")
	f, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	out := map[string][]string{}
	for _, d := range f.Decls {
		gd, ok := d.(*ast.GenDecl)
		if !ok || gd.Tok != token.CONST {
			continue
		}
		for _, spec := range gd.Specs {
			vs := spec.(*ast.ValueSpec)
			typ, ok := vs.Type.(*ast.Ident)
			if !ok {
				t.Fatalf("%s: constant %s has no named type", path, vs.Names[0].Name)
			}
			for _, v := range vs.Values {
				lit, ok := v.(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					t.Fatalf("%s: constant %s is not a string literal", path, vs.Names[0].Name)
				}
				s, err := strconv.Unquote(lit.Value)
				if err != nil {
					t.Fatal(err)
				}
				out[typ.Name] = append(out[typ.Name], s)
			}
		}
	}
	return out
}

// TestViewSweep plays seeded games and, at every state, checks the views of
// seat 0, seat 1 and a spectator field by field against the state (vwCheck).
// It also checks that building a view leaves the state untouched.
func TestViewSweep(t *testing.T) {
	games := 40
	if testing.Short() {
		games = 12
	}
	buf := make([]engine.Action, 0, engine.MaxLegalActions)
	var phases [8]int
	for i := range games {
		seed := redactSeed(1000 + i)
		opts := engine.Options{FirstPlayer: engine.FirstPlayerRule(i % 3), HideAtEnd: i%4 == 1}
		var pols [2]enginetest.Policy
		if i%2 == 0 {
			p := enginetest.NewRandomPolicy(seed)
			pols = [2]enginetest.Policy{p, p}
		} else {
			pols = [2]enginetest.Policy{enginetest.NewSimplePolicy(seed), enginetest.NewSimplePolicy(seed + 1)}
		}
		s := engine.NewGame(seed, opts)
		for n := 0; ; n++ {
			phases[s.Phase]++
			for _, v := range redactViewers {
				before := s
				vw := engine.ViewFor(&s, v)
				if s != before {
					t.Fatalf("game %d, ply %d: ViewFor changed the state", i, s.Ply)
				}
				vwCheck(t, &s, v, &vw)
			}
			if t.Failed() {
				t.Fatalf("game %d (seed %d), ply %d, phase %s", i, seed, s.Ply, s.Phase)
			}
			if s.Over() {
				break
			}
			if n > 300 {
				t.Fatalf("game %d: not over after %d actions", i, n)
			}
			a := pols[s.ToMove].Choose(&s, engine.LegalActions(&s, buf[:0]))
			if err := engine.Apply(&s, a, nil); err != nil {
				t.Fatalf("game %d: %s: %v", i, a, err)
			}
		}
	}
	t.Logf("%d games; states per phase %v", games, phases)
}

// TestViewFixturesCheck runs the field-by-field checks on every protocol
// fixture, which reach the rarer positions on purpose.
func TestViewFixturesCheck(t *testing.T) {
	for _, fx := range protoFixtures(t) {
		for _, v := range redactViewers {
			vw := engine.ViewFor(&fx.state, v)
			vwCheck(t, &fx.state, v, &vw)
		}
		if t.Failed() {
			t.Fatalf("fixture %s", fx.name)
		}
	}
}

// vwCheck checks every field of vw, the view of s for v, against s.
func vwCheck(t *testing.T, s *engine.State, v engine.Viewer, vw *view.View) {
	t.Helper()
	seat, seated := v.Seat()
	decider := seated && !s.Over() && seat == s.ToMove
	if vw.Schema != view.SchemaVersion || vw.CatalogHash != engine.CatalogHash() || vw.Ply != int(s.Ply) ||
		string(vw.Phase) != s.Phase.String() || vw.Age != int(s.Age) || vw.Active != int(s.Active) ||
		vw.FirstPlayer != int(s.FirstSeat) || vw.ExtraTurn != s.ExtraTurn || vw.WondersBuilt != int(s.WondersBuilt) {
		t.Errorf("viewer %d: header %d %s %d %s age %d active %d first %d extra %v built %d does not match the state",
			v, vw.Schema, vw.CatalogHash, vw.Ply, vw.Phase, vw.Age, vw.Active, vw.FirstPlayer, vw.ExtraTurn, vw.WondersBuilt)
	}
	switch {
	case seated && (vw.Viewer == nil || *vw.Viewer != int(seat)):
		t.Errorf("viewer %d: viewer field %v", v, vw.Viewer)
	case !seated && vw.Viewer != nil:
		t.Errorf("spectator: viewer field %d", *vw.Viewer)
	}
	switch {
	case s.Over() && vw.ToMove != nil:
		t.Errorf("viewer %d: toMove %d after the game", v, *vw.ToMove)
	case !s.Over() && (vw.ToMove == nil || *vw.ToMove != int(s.ToMove)):
		t.Errorf("viewer %d: toMove %v, want %d", v, vw.ToMove, s.ToMove)
	}

	vwCheckDraft(t, s, vw)
	vwCheckStructure(t, s, vw)
	for i := range vw.Players {
		vwCheckPlayer(t, s, engine.Seat(i), &vw.Players[i])
	}
	vwCheckMilitary(t, s, &vw.Military)

	if got, want := vw.BoardTokens, vwTokenSlugs(s.BoardTokens); !slices.Equal(got, want) {
		t.Errorf("boardTokens %v, want %v", got, want)
	}
	box := 0
	for _, tk := range s.BoxTokens {
		if tk != 0 {
			box++
		}
	}
	if vw.BoxTokenCount != box {
		t.Errorf("boxTokenCount %d, want %d", vw.BoxTokenCount, box)
	}
	removed := 0
	if s.Age >= 1 && s.Age <= engine.NumAges {
		for _, c := range s.Removed[s.Age-1] {
			if c != 0 {
				removed++
			}
		}
	}
	if vw.RemovedCount != removed {
		t.Errorf("removedCount %d, want %d", vw.RemovedCount, removed)
	}
	if got, want := vw.Discard, vwCardSlugs(s.Discard); !slices.Equal(got, want) {
		t.Errorf("discard %v, want %v", got, want)
	}
	if want := vwScores(engine.Scores(s)); vw.ScorePreview != want {
		t.Errorf("scorePreview %+v, want %+v", vw.ScorePreview, want)
	}
	vwCheckResult(t, s, vw)
	vwCheckPending(t, s, vw, decider)
	vwCheckChoices(t, s, vw, decider)
}

// vwCheckDraft checks the draft section: present only during the draft,
// with the current round's four wonders in dealt order.
func vwCheckDraft(t *testing.T, s *engine.State, vw *view.View) {
	t.Helper()
	if s.Phase != engine.PhaseWonderDraft {
		if vw.Draft != nil {
			t.Errorf("a draft section in phase %s", s.Phase)
		}
		return
	}
	if vw.Draft == nil {
		t.Errorf("no draft section during the draft")
		return
	}
	picks := 0
	for i := range engine.NumDraftWonders {
		if s.DraftTaken&(1<<i) != 0 {
			picks++
		}
	}
	round := picks / 4
	if vw.Draft.Round != round+1 || len(vw.Draft.Offer) != 4 {
		t.Errorf("draft round %d with %d offers after %d picks", vw.Draft.Round, len(vw.Draft.Offer), picks)
		return
	}
	for k, o := range vw.Draft.Offer {
		i := 4*round + k
		w := s.DraftOrder[i]
		if o.Wonder != w.String() {
			t.Errorf("draft offer %d is %s, want %s", k, o.Wonder, w)
		}
		taken := s.DraftTaken&(1<<i) != 0
		switch {
		case !taken && o.TakenBy != nil:
			t.Errorf("draft offer %s is not taken, but takenBy %d", o.Wonder, *o.TakenBy)
		case taken && (o.TakenBy == nil || !vwHasWonder(&s.Players[*o.TakenBy], w)):
			t.Errorf("draft offer %s: takenBy %v does not hold it", o.Wonder, o.TakenBy)
		}
	}
}

func vwHasWonder(p *engine.Player, w catalog.WonderID) bool {
	for _, ws := range p.Wonders {
		if ws.ID == w {
			return true
		}
	}
	return false
}

// vwCheckStructure checks the 20 slots: geometry, state, back, the card only
// when face up, coverage, accessibility, and both seats' costs exactly for
// the accessible face-up cards.
func vwCheckStructure(t *testing.T, s *engine.State, vw *view.View) {
	t.Helper()
	if s.Age < 1 || s.Age > engine.NumAges {
		if len(vw.Structure) != 0 {
			t.Errorf("%d structure slots in age %d", len(vw.Structure), s.Age)
		}
		return
	}
	if len(vw.Structure) != engine.SlotsPerAge {
		t.Errorf("%d structure slots, want %d", len(vw.Structure), engine.SlotsPerAge)
		return
	}
	occupied, acc := engine.Occupied(s), engine.Accessible(s)
	for i, sv := range vw.Structure {
		sl := s.Slots[i]
		pos := engine.SlotPos(s.Age, i)
		where := "slot " + strconv.Itoa(i)
		if sv.Index != i || sv.Row != int(pos.Row) || sv.Col != int(pos.Col) {
			t.Errorf("%s: index %d at (%d,%d), want (%d,%d)", where, sv.Index, sv.Row, sv.Col, pos.Row, pos.Col)
		}
		if sl.Card == 0 {
			if sv.State != view.SlotStateEmpty || sv.Back != nil || sv.Card != nil || sv.Costs != nil || sv.Accessible || len(sv.CoveredBy) != 0 {
				t.Errorf("%s: an empty slot shows %+v", where, sv)
			}
			continue
		}
		if sv.Back == nil || string(*sv.Back) != sl.Back.String() {
			t.Errorf("%s: back %v, want %s", where, sv.Back, sl.Back)
		}
		var cover []int
		for j := range engine.SlotsPerAge {
			if engine.CoverMask(s.Age, i)&occupied&(1<<j) != 0 {
				cover = append(cover, j)
			}
		}
		if !slices.Equal(sv.CoveredBy, cover) {
			t.Errorf("%s: coveredBy %v, want %v", where, sv.CoveredBy, cover)
		}
		if sv.Accessible != (acc&(1<<i) != 0) {
			t.Errorf("%s: accessible %v", where, sv.Accessible)
		}
		if !sl.FaceUp {
			if sv.State != view.SlotStateFaceDown || sv.Card != nil || sv.Costs != nil {
				t.Errorf("%s: a face-down slot shows %+v", where, sv)
			}
			continue
		}
		if sv.State != view.SlotStateFaceUp || sv.Card == nil || *sv.Card != sl.Card.String() {
			t.Errorf("%s: a face-up slot of %s shows %+v", where, sl.Card, sv)
			continue
		}
		// Every face-up card carries both seats' costs, covered or not: the
		// cost of building it as if it were accessible now.
		if sv.Costs == nil {
			t.Errorf("%s: no costs on the face-up %s (accessible %v)", where, sl.Card, sv.Accessible)
			continue
		}
		for seat := range sv.Costs {
			vwCheckBuildingCost(t, s, engine.Seat(seat), sl.Card, &sv.Costs[seat], where)
		}
	}
}

// vwCheckBuildingCost checks seat's cost view of building card c.
func vwCheckBuildingCost(t *testing.T, s *engine.State, seat engine.Seat, c catalog.CardID, cv *view.CostView, where string) {
	t.Helper()
	d := catalog.Card(c)
	where += ": seat " + strconv.Itoa(int(seat)) + " building " + c.String()
	vwCheckCost(t, s, seat, cv, engine.BuildingQuote(s, seat, c), d.Cost,
		vwDiscountToken(s, seat, d.Color == catalog.Blue, catalog.ForBlue), d.ChainIn, where)
}

// vwCheckWonderCost checks seat's cost view of building wonder w.
func vwCheckWonderCost(t *testing.T, s *engine.State, seat engine.Seat, w catalog.WonderID, cv *view.CostView, where string) {
	t.Helper()
	d := catalog.Wonder(w)
	where += ": seat " + strconv.Itoa(int(seat)) + " wonder " + w.String()
	vwCheckCost(t, s, seat, cv, engine.WonderQuote(s, seat, w), d.Cost,
		vwDiscountToken(s, seat, true, catalog.ForWonders), catalog.NoChain, where)
}

// vwDiscountToken returns the slug of seat's token whose discount applies
// to an item of the given target, or "" (also when applies is false).
func vwDiscountToken(s *engine.State, seat engine.Seat, applies bool, target catalog.DiscountTarget) string {
	if !applies {
		return ""
	}
	for tk := range s.Players[seat].Tokens.All() {
		for _, e := range catalog.Token(tk).Effects {
			if e.Kind == catalog.EffDiscount && e.Target == target {
				return tk.String()
			}
		}
	}
	return ""
}

// vwCheckCost checks a cost view against the engine's quote and against the
// rules of a breakdown: every printed unit covered exactly once, each part
// named and priced as the payer's state says.
func vwCheckCost(t *testing.T, s *engine.State, seat engine.Seat, cv *view.CostView, q engine.Quote,
	printed catalog.Cost, discountBy string, chainIn catalog.Chain, where string) {
	t.Helper()
	p := &s.Players[seat]
	if cv.Total != int(q.Total) || cv.Affordable != q.Affordable || cv.ChainFree != q.ChainFree || cv.ToOpponent != int(q.ToOpponent) {
		t.Errorf("%s: total %d affordable %v chainFree %v toOpponent %d, the quote is %+v",
			where, cv.Total, cv.Affordable, cv.ChainFree, cv.ToOpponent, q)
	}
	if cv.ChainFree {
		bonus := 0
		if p.Tokens.Has(catalog.TokenUrbanism) {
			bonus = 4
		}
		ok := cv.ChainFrom != nil && cv.Total == 0 && cv.PrintedCoins == 0 && cv.Own == (view.ResourceCounts{}) &&
			len(cv.Producers) == 0 && cv.Discount == nil && len(cv.Trade) == 0 && cv.TradeTotal == 0 &&
			cv.UrbanismBonus == bonus && cv.Shortfall == 0
		if ok {
			from, found := catalog.CardBySlug(*cv.ChainFrom)
			ok = found && p.City.Has(from) && catalog.Card(from).ChainOut == chainIn && chainIn != catalog.NoChain
		}
		if !ok {
			t.Errorf("%s: a chain-free cost shows %+v", where, *cv)
		}
		return
	}
	if cv.ChainFrom != nil || cv.UrbanismBonus != 0 {
		t.Errorf("%s: chainFrom %v and urbanismBonus %d on a paid build", where, cv.ChainFrom, cv.UrbanismBonus)
	}
	if cv.PrintedCoins != int(printed.Coins) {
		t.Errorf("%s: printedCoins %d, want %d", where, cv.PrintedCoins, printed.Coins)
	}
	covered := vwRes(cv.Own)
	for r := range covered {
		if covered[r] > int(p.Prod[r]) {
			t.Errorf("%s: own %v exceeds the production %v", where, cv.Own, p.Prod)
		}
	}
	var used []string
	for _, u := range cv.Producers {
		r := vwResource(t, u.Resource)
		covered[r]++
		var mask catalog.ResMask
		var name string
		switch {
		case u.Card != nil && u.Wonder == nil:
			c, ok := catalog.CardBySlug(*u.Card)
			if !ok || !p.City.Has(c) {
				t.Errorf("%s: producer card %s is not in the city", where, *u.Card)
				continue
			}
			e, _ := catalog.FindEffect(catalog.Card(c).Effects, catalog.EffProduceChoice)
			mask, name = e.Mask, *u.Card
		case u.Wonder != nil && u.Card == nil:
			w, ok := catalog.WonderBySlug(*u.Wonder)
			if !ok || !vwBuilt(p, w) {
				t.Errorf("%s: producer wonder %s is not built", where, *u.Wonder)
				continue
			}
			e, _ := catalog.FindEffect(catalog.Wonder(w).Effects, catalog.EffProduceChoice)
			mask, name = e.Mask, *u.Wonder
		default:
			t.Errorf("%s: a producer names %v and %v", where, u.Card, u.Wonder)
			continue
		}
		if !mask.Has(r) {
			t.Errorf("%s: producer %s cannot make %s", where, name, u.Resource)
		}
		if slices.Contains(used, name) {
			t.Errorf("%s: producer %s used twice", where, name)
		}
		used = append(used, name)
	}
	switch {
	case discountBy == "" && cv.Discount != nil:
		t.Errorf("%s: a discount by %s that does not apply", where, cv.Discount.Token)
	case discountBy != "" && (cv.Discount == nil || cv.Discount.Token != discountBy):
		t.Errorf("%s: discount %+v, want one by %s", where, cv.Discount, discountBy)
	case cv.Discount != nil:
		n := 0
		for r, u := range vwRes(cv.Discount.Units) {
			covered[r] += u
			n += u
		}
		if n > 2 {
			t.Errorf("%s: a discount of %d units", where, n)
		}
	}
	prices := engine.TradePrices(s, seat)
	tradeTotal, last := 0, -1
	for _, tr := range cv.Trade {
		r := vwResource(t, tr.Resource)
		if int(r) <= last || tr.Units <= 0 {
			t.Errorf("%s: trade entries %+v are not one per resource bought, in order", where, cv.Trade)
		}
		last = int(r)
		covered[r] += tr.Units
		if tr.UnitPrice != int(prices[r]) || tr.Fixed != p.TradeFix.Has(r) || tr.Coins != tr.Units*tr.UnitPrice {
			t.Errorf("%s: trade %+v; the price of %s is %d (fixed %v)", where, tr, tr.Resource, prices[r], p.TradeFix.Has(r))
		}
		tradeTotal += tr.Coins
	}
	for r := range covered {
		if covered[r] != int(printed.Res[r]) {
			t.Errorf("%s: %d units of %s covered, the cost has %d: %+v", where, covered[r], catalog.Resource(r), printed.Res[r], *cv)
		}
	}
	if cv.TradeTotal != tradeTotal || cv.Total != cv.PrintedCoins+cv.TradeTotal {
		t.Errorf("%s: tradeTotal %d (entries %d), total %d", where, cv.TradeTotal, tradeTotal, cv.Total)
	}
	if cv.Affordable != (cv.Total <= int(p.Coins)) {
		t.Errorf("%s: affordable %v with total %d and %d coins", where, cv.Affordable, cv.Total, p.Coins)
	}
	switch {
	case cv.Affordable && cv.Shortfall != 0:
		t.Errorf("%s: shortfall %d on an affordable cost", where, cv.Shortfall)
	case !cv.Affordable && cv.Shortfall != cv.Total-int(p.Coins):
		t.Errorf("%s: shortfall %d, want %d", where, cv.Shortfall, cv.Total-int(p.Coins))
	}
}

func vwBuilt(p *engine.Player, w catalog.WonderID) bool {
	for _, ws := range p.Wonders {
		if ws.ID == w && ws.Built {
			return true
		}
	}
	return false
}

// vwCheckPlayer checks one seat's section.
func vwCheckPlayer(t *testing.T, s *engine.State, seat engine.Seat, pv *view.PlayerView) {
	t.Helper()
	p := &s.Players[seat]
	where := "player " + strconv.Itoa(int(seat))
	if pv.Seat != int(seat) || pv.Coins != int(p.Coins) || pv.DiscardValue != int(engine.DiscardValue(s, seat)) {
		t.Errorf("%s: seat %d coins %d discardValue %d", where, pv.Seat, pv.Coins, pv.DiscardValue)
	}
	if got, want := pv.City, vwCardSlugs(p.City); !slices.Equal(got, want) {
		t.Errorf("%s: city %v, want %v", where, got, want)
	}
	if got, want := pv.Tokens, vwTokenSlugs(p.Tokens); !slices.Equal(got, want) {
		t.Errorf("%s: tokens %v, want %v", where, got, want)
	}
	k := 0
	for i, ws := range p.Wonders {
		if ws.ID == 0 {
			continue
		}
		if k >= len(pv.Wonders) {
			t.Errorf("%s: %d wonders listed, the seat has more", where, len(pv.Wonders))
			break
		}
		wv := pv.Wonders[k]
		k++
		if wv.Wonder != ws.ID.String() || wv.Built != ws.Built || wv.Removed != ws.Removed {
			t.Errorf("%s: wonder %d is %+v, want %+v", where, i, wv, ws)
		}
		switch {
		case ws.Built && (wv.Tucked == nil || *wv.Tucked != ws.Tucked.String()):
			t.Errorf("%s: %s tucks %v, want %s", where, ws.ID, wv.Tucked, ws.Tucked)
		case !ws.Built && wv.Tucked != nil:
			t.Errorf("%s: the unbuilt %s tucks %s", where, ws.ID, *wv.Tucked)
		}
		open := !ws.Built && !ws.Removed && s.WondersBuilt < engine.MaxWondersBuilt
		switch {
		case open && wv.Cost == nil:
			t.Errorf("%s: no cost for the open %s", where, ws.ID)
		case !open && wv.Cost != nil:
			t.Errorf("%s: a cost for %s, which cannot be built", where, ws.ID)
		case open:
			vwCheckWonderCost(t, s, seat, ws.ID, wv.Cost, where)
		}
	}
	if k != len(pv.Wonders) {
		t.Errorf("%s: %d wonders listed, the seat has %d", where, len(pv.Wonders), k)
	}
	if got := vwRes(pv.Production.Fixed); got != vwArr(p.Prod) {
		t.Errorf("%s: fixed production %v, want %v", where, got, p.Prod)
	}
	if got, want := pv.Production.TradeFix, vwMaskList(p.TradeFix); !slices.Equal(got, want) {
		t.Errorf("%s: tradeFix %v, want %v", where, got, want)
	}
	if len(pv.Production.Producers) != int(p.NChoice) {
		t.Errorf("%s: %d producers, the cache has %d", where, len(pv.Production.Producers), p.NChoice)
	} else {
		for i, pr := range pv.Production.Producers {
			if want := vwMaskList(p.Choice[i]); !slices.Equal(pr.Resources, want) || (pr.Card == nil) == (pr.Wonder == nil) {
				t.Errorf("%s: producer %d is %+v, the cache has %v", where, i, pr, want)
			}
		}
	}
	if got := vwRes(pv.TradePrices); got != vwArr(engine.TradePrices(s, seat)) {
		t.Errorf("%s: tradePrices %v", where, got)
	}
	var syms []view.SymbolCount
	for sym := catalog.Globe; sym < catalog.NumSymbols; sym++ {
		if n := int(p.Sci[sym]); n > 0 {
			syms = append(syms, view.SymbolCount{Symbol: view.ScienceSymbol(sym.String()), Count: n})
		}
	}
	if !slices.Equal(pv.Science.Symbols, syms) || pv.Science.Distinct != engine.DistinctSymbols(p) {
		t.Errorf("%s: science %+v, want %v", where, pv.Science, syms)
	}
	var chains []view.Chain
	for ch := catalog.Chain(1); ch < catalog.NumChains; ch++ {
		if p.Chains&ch.Bit() != 0 {
			chains = append(chains, view.Chain(ch.String()))
		}
	}
	if !slices.Equal(pv.Chains, chains) {
		t.Errorf("%s: chains %v, want %v", where, pv.Chains, chains)
	}
}

// vwCheckMilitary checks the pawn and the looting tokens.
func vwCheckMilitary(t *testing.T, s *engine.State, mv *view.MilitaryView) {
	t.Helper()
	want := []view.LootingView{
		{At: -6, Seat: 0, Value: 5, Present: s.Looting&engine.LootMinus6 != 0},
		{At: -3, Seat: 0, Value: 2, Present: s.Looting&engine.LootMinus3 != 0},
		{At: 3, Seat: 1, Value: 2, Present: s.Looting&engine.LootPlus3 != 0},
		{At: 6, Seat: 1, Value: 5, Present: s.Looting&engine.LootPlus6 != 0},
	}
	if mv.Position != int(s.Military) || !slices.Equal(mv.Looting, want) {
		t.Errorf("military %+v, want position %d and %+v", *mv, s.Military, want)
	}
}

// vwCheckResult checks the result: present exactly at game over.
func vwCheckResult(t *testing.T, s *engine.State, vw *view.View) {
	t.Helper()
	if !s.Over() {
		if vw.Result != nil || vw.Revealed != nil {
			t.Errorf("a result or revealed section before the end")
		}
		return
	}
	r := vw.Result
	if r == nil {
		t.Errorf("no result at game over")
		return
	}
	shared := s.Result.Winner == engine.SharedVictory
	ok := string(r.Victory) == s.Result.Kind.String() && r.Shared == shared && r.Scores == vwScores(s.Result.Scores)
	if shared {
		ok = ok && r.Winner == nil
	} else {
		ok = ok && r.Winner != nil && *r.Winner == int(s.Result.Winner)
	}
	if !ok {
		t.Errorf("result %+v, want %+v", *r, s.Result)
	}
	if (vw.Revealed == nil) != s.Opts.HideAtEnd {
		t.Errorf("revealed %v with HideAtEnd %v", vw.Revealed != nil, s.Opts.HideAtEnd)
	}
}

// vwCheckPending checks the pending section of each phase.
func vwCheckPending(t *testing.T, s *engine.State, vw *view.View, decider bool) {
	t.Helper()
	pv := vw.Pending
	switch s.Phase {
	case engine.PhaseWonderDraft, engine.PhasePlay, engine.PhaseGameOver:
		if pv != nil {
			t.Errorf("a pending section %+v in phase %s", *pv, s.Phase)
		}
		return
	}
	if pv == nil || string(pv.Kind) != s.Phase.String() || pv.Seat != int(s.ToMove) {
		t.Errorf("pending %+v in phase %s, to move %d", pv, s.Phase, s.ToMove)
		return
	}
	var cards, tokens []string
	hidden := 0
	var color *view.Color
	switch s.Phase {
	case engine.PhaseDestroyCard:
		c := view.Color(s.Destroy.String())
		color = &c
		for card := range s.Players[s.ToMove.Opp()].City.All() {
			if catalog.Card(card).Color == s.Destroy {
				cards = append(cards, card.String())
			}
		}
	case engine.PhaseBuildFromDiscard:
		cards = vwCardSlugs(s.Discard)
	case engine.PhasePickBoardToken:
		tokens = vwTokenSlugs(s.BoardTokens)
	case engine.PhasePickLibraryToken:
		for _, tk := range s.Library {
			switch {
			case tk == 0:
			case decider:
				tokens = append(tokens, tk.String())
			default:
				hidden++
			}
		}
	}
	if !slices.Equal(pv.Cards, cards) || !slices.Equal(pv.Tokens, tokens) || pv.HiddenTokens != hidden ||
		(color == nil) != (pv.Color == nil) || (color != nil && *color != *pv.Color) {
		t.Errorf("pending %+v, want cards %v tokens %v hidden %d colour %v", *pv, cards, tokens, hidden, color)
	}
}

// vwCheckChoices checks choices and disabled: empty except for the seat to
// move, whose choices are exactly the legal actions in order, and whose
// disabled list is exactly the play actions that are not legal.
func vwCheckChoices(t *testing.T, s *engine.State, vw *view.View, decider bool) {
	t.Helper()
	if !decider {
		if len(vw.Choices) != 0 || len(vw.Disabled) != 0 {
			t.Errorf("%d choices and %d disabled for a viewer who is not to move", len(vw.Choices), len(vw.Disabled))
		}
		return
	}
	seat := s.ToMove
	legal := engine.LegalActions(s, nil)
	if len(vw.Choices) != len(legal) {
		t.Errorf("%d choices for %d legal actions", len(vw.Choices), len(legal))
		return
	}
	for i, a := range legal {
		c := vw.Choices[i]
		where := "choice " + a.Key()
		if c.Key != a.Key() || string(c.Kind) != a.Type.Verb() {
			t.Errorf("%s: key %s kind %s", where, c.Key, c.Kind)
		}
		want := view.Choice{Key: c.Key, Kind: c.Kind}
		switch a.Type {
		case engine.ActDraftWonder:
			want.Wonder = vwStr(a.Wonder.String())
		case engine.ActConstructBuilding:
			want.Card, want.Cost = vwStr(a.Card.String()), c.Cost
			if c.Cost == nil || !c.Cost.Affordable {
				t.Errorf("%s: cost %+v", where, c.Cost)
			} else {
				vwCheckBuildingCost(t, s, seat, a.Card, c.Cost, where)
			}
		case engine.ActDiscardBuilding:
			want.Card = vwStr(a.Card.String())
			want.Gain = vwInt(int(engine.DiscardValue(s, seat)))
		case engine.ActConstructWonder:
			want.Card, want.Wonder, want.Cost = vwStr(a.Card.String()), vwStr(a.Wonder.String()), c.Cost
			if c.Cost == nil || !c.Cost.Affordable {
				t.Errorf("%s: cost %+v", where, c.Cost)
			} else {
				vwCheckWonderCost(t, s, seat, a.Wonder, c.Cost, where)
			}
		case engine.ActDestroyCard, engine.ActBuildFromDiscard:
			want.Card = vwStr(a.Card.String())
		case engine.ActChooseProgressToken:
			want.Token = vwStr(a.Token.String())
		case engine.ActChooseStarter:
			want.Starter = vwInt(int(a.Starter))
		}
		if !vwSameChoice(c, want) {
			t.Errorf("%s: %+v, want %+v", where, c, want)
		}
	}

	var dis []view.DisabledChoice
	if s.Phase == engine.PhasePlay {
		acc := engine.Accessible(s)
		for i, sl := range s.Slots {
			if acc&(1<<i) == 0 || !sl.FaceUp {
				continue
			}
			if a := engine.ConstructAction(sl.Card); engine.IsLegal(s, a) != nil {
				dis = append(dis, view.DisabledChoice{Key: a.Key(), Kind: view.ChoiceKindConstruct, Card: vwStr(sl.Card.String()),
					Reason: view.ReasonUnaffordable})
			}
			for _, ws := range s.Players[seat].Wonders {
				if ws.ID == 0 || ws.Built || ws.Removed {
					continue
				}
				a := engine.WonderAction(ws.ID, sl.Card)
				err := engine.IsLegal(s, a)
				if err == nil {
					continue
				}
				dis = append(dis, view.DisabledChoice{Key: a.Key(), Kind: view.ChoiceKindWonder, Card: vwStr(sl.Card.String()),
					Wonder: vwStr(ws.ID.String()), Reason: view.Reason(err.(*engine.IllegalError).Reason.String())})
			}
		}
	}
	if len(vw.Disabled) != len(dis) {
		t.Errorf("disabled %+v, want %+v", vw.Disabled, dis)
		return
	}
	for i, d := range vw.Disabled {
		w := dis[i]
		if d.Key != w.Key || d.Kind != w.Kind || d.Reason != w.Reason || !vwSameStr(d.Card, w.Card) || !vwSameStr(d.Wonder, w.Wonder) || d.Cost == nil {
			t.Errorf("disabled %d is %+v, want %+v", i, d, w)
			continue
		}
		a, err := engine.ParseKey(d.Key)
		if err != nil {
			t.Fatal(err)
		}
		if a.Type == engine.ActConstructBuilding {
			vwCheckBuildingCost(t, s, seat, a.Card, d.Cost, "disabled "+d.Key)
		} else {
			vwCheckWonderCost(t, s, seat, a.Wonder, d.Cost, "disabled "+d.Key)
		}
		if d.Reason == view.ReasonUnaffordable && d.Cost.Affordable {
			t.Errorf("disabled %s: unaffordable, but the cost is affordable", d.Key)
		}
	}
}

// vwSameChoice compares two choices, pointers by value.
func vwSameChoice(a, b view.Choice) bool {
	return a.Key == b.Key && a.Kind == b.Kind && vwSameStr(a.Card, b.Card) && vwSameStr(a.Wonder, b.Wonder) &&
		vwSameStr(a.Token, b.Token) && vwSameInt(a.Starter, b.Starter) && vwSameInt(a.Gain, b.Gain) && a.Cost == b.Cost
}

func vwSameStr(a, b *string) bool { return (a == nil) == (b == nil) && (a == nil || *a == *b) }
func vwSameInt(a, b *int) bool    { return (a == nil) == (b == nil) && (a == nil || *a == *b) }
func vwStr(s string) *string      { return &s }
func vwInt(n int) *int            { return &n }

// vwRes converts resource counts to an array in resource order.
func vwRes(rc view.ResourceCounts) [catalog.NumResources]int {
	return [catalog.NumResources]int{rc.Wood, rc.Clay, rc.Stone, rc.Glass, rc.Papyrus}
}

func vwArr(a [catalog.NumResources]uint8) [catalog.NumResources]int {
	var out [catalog.NumResources]int
	for i, v := range a {
		out[i] = int(v)
	}
	return out
}

// vwResource returns the engine resource of a view resource name.
func vwResource(t *testing.T, r view.Resource) catalog.Resource {
	t.Helper()
	for i := range catalog.NumResources {
		if i.String() == string(r) {
			return i
		}
	}
	t.Fatalf("unknown resource %q", r)
	return 0
}

func vwMaskList(m catalog.ResMask) []view.Resource {
	out := []view.Resource{}
	for r := range catalog.NumResources {
		if m.Has(r) {
			out = append(out, view.Resource(r.String()))
		}
	}
	return out
}

func vwCardSlugs(set catalog.CardSet) []string {
	out := []string{}
	for c := range set.All() {
		out = append(out, c.String())
	}
	return out
}

func vwTokenSlugs(set catalog.TokenSet) []string {
	out := []string{}
	for tk := range set.All() {
		out = append(out, tk.String())
	}
	return out
}

func vwScores(sc [2]engine.Score) [2]view.ScoreView {
	var out [2]view.ScoreView
	for i, s := range sc {
		out[i] = view.ScoreView{
			Blue: int(s.Blue), Green: int(s.Green), Yellow: int(s.Yellow), Purple: int(s.Purple),
			Wonders: int(s.Wonders), Tokens: int(s.Tokens), Coins: int(s.Coins), Military: int(s.Military), Total: int(s.Total),
		}
	}
	return out
}

// TestViewCostBreakdown checks the costs of a position built to show every
// part of a breakdown, for both seats, against values worked out by hand
// from the rules: own production, flexible producers named by card and by
// wonder, Masonry's and Architecture's discounts, a fixed trade price, the
// opponent's Economy, a chain with Urbanism's bonus, and a shortfall.
func TestViewCostBreakdown(t *testing.T) {
	s := protoCostsState(t, protoSeed(100))
	vw := engine.ViewFor(&s, 0)
	slot := map[string]*view.SlotView{}
	for i := range vw.Structure {
		if c := vw.Structure[i].Card; c != nil {
			slot[*c] = &vw.Structure[i]
		}
	}
	type res = view.ResourceCounts
	type want struct {
		printed, total, toOpp, shortfall, bonus int
		chainFrom                               string
		own                                     res
		producers                               []view.ProducerUse
		discount                                *view.DiscountView
		trade                                   []view.TradeView
	}
	masonry := func(u res) *view.DiscountView { return &view.DiscountView{Token: "masonry", Units: u} }
	architecture := func(u res) *view.DiscountView { return &view.DiscountView{Token: "architecture", Units: u} }
	forum := func(r view.Resource) view.ProducerUse { return view.ProducerUse{Card: vwStr("forum"), Resource: r} }
	lighthouse := func(r view.Resource) view.ProducerUse {
		return view.ProducerUse{Wonder: vwStr("great-lighthouse"), Resource: r}
	}
	trade := func(r view.Resource, units, price int, fixed bool) view.TradeView {
		return view.TradeView{Resource: r, Units: units, UnitPrice: price, Fixed: fixed, Coins: units * price}
	}
	cases := []struct {
		what string
		cv   *view.CostView
		want want
	}{
		// Seat 0 owns Lumber Yard, Glassworks, Forum (glass or papyrus),
		// Stone Reserve, Theater, Masonry and Urbanism, and 9 coins; seat 1
		// owns 3 clay and Economy, so seat 0 pays 5 per clay, 1 per stone
		// and 2 for the rest, and seat 1 receives the trading coins.
		{"seat 0 courthouse", &slot["courthouse"].Costs[0], want{own: res{Wood: 1, Glass: 1}, discount: masonry(res{Wood: 1})}},
		{"seat 0 statue", &slot["statue"].Costs[0], want{chainFrom: "theater", bonus: 4}},
		{"seat 0 aqueduct", &slot["aqueduct"].Costs[0], want{total: 1, toOpp: 1, discount: masonry(res{Stone: 2}),
			trade: []view.TradeView{trade(view.ResourceStone, 1, 1, true)}}},
		{"seat 0 caravansery", &slot["caravansery"].Costs[0], want{printed: 2, total: 2, own: res{Glass: 1},
			producers: []view.ProducerUse{forum(view.ResourcePapyrus)}}},
		{"seat 0 walls", &slot["walls"].Costs[0], want{total: 2, toOpp: 2, trade: []view.TradeView{trade(view.ResourceStone, 2, 1, true)}}},
		{"seat 0 parade ground", &slot["parade-ground"].Costs[0], want{total: 10, toOpp: 10, shortfall: 1, own: res{Glass: 1},
			trade: []view.TradeView{trade(view.ResourceClay, 2, 5, false)}}},
		{"seat 0 pyramids", vw.Players[0].Wonders[0].Cost, want{total: 3, toOpp: 3,
			producers: []view.ProducerUse{forum(view.ResourcePapyrus)}, trade: []view.TradeView{trade(view.ResourceStone, 3, 1, true)}}},

		// Seat 1 owns Clay Pool, Brickyard, the Great Lighthouse (wood,
		// clay or stone), Economy and Architecture, and 7 coins; it pays 3
		// per wood and glass (seat 0 makes one of each) and 2 for the rest.
		{"seat 1 courthouse", &slot["courthouse"].Costs[1], want{total: 6, producers: []view.ProducerUse{lighthouse(view.ResourceWood)},
			trade: []view.TradeView{trade(view.ResourceWood, 1, 3, false), trade(view.ResourceGlass, 1, 3, false)}}},
		{"seat 1 statue", &slot["statue"].Costs[1], want{own: res{Clay: 2}}},
		{"seat 1 aqueduct", &slot["aqueduct"].Costs[1], want{total: 4, producers: []view.ProducerUse{lighthouse(view.ResourceStone)},
			trade: []view.TradeView{trade(view.ResourceStone, 2, 2, false)}}},
		{"seat 1 caravansery", &slot["caravansery"].Costs[1], want{printed: 2, total: 7,
			trade: []view.TradeView{trade(view.ResourceGlass, 1, 3, false), trade(view.ResourcePapyrus, 1, 2, false)}}},
		{"seat 1 great library", vw.Players[1].Wonders[1].Cost, want{total: 5, producers: []view.ProducerUse{lighthouse(view.ResourceWood)},
			discount: architecture(res{Wood: 2}),
			trade:    []view.TradeView{trade(view.ResourceGlass, 1, 3, false), trade(view.ResourcePapyrus, 1, 2, false)}}},
	}
	for _, c := range cases {
		cv, w := c.cv, c.want
		if cv == nil {
			t.Errorf("%s: no cost", c.what)
			continue
		}
		if w.producers == nil {
			w.producers = []view.ProducerUse{}
		}
		if w.trade == nil {
			w.trade = []view.TradeView{}
		}
		tradeTotal := 0
		for _, tr := range w.trade {
			tradeTotal += tr.Coins
		}
		wantCV := view.CostView{
			Affordable: w.shortfall == 0, Shortfall: w.shortfall, ChainFree: w.chainFrom != "", PrintedCoins: w.printed,
			Own: w.own, Producers: w.producers, Discount: w.discount, Trade: w.trade, TradeTotal: tradeTotal,
			Total: w.total, ToOpponent: w.toOpp, UrbanismBonus: w.bonus,
		}
		if w.chainFrom != "" {
			wantCV.ChainFrom = vwStr(w.chainFrom)
		}
		if g, e := vwJSON(t, cv), vwJSON(t, wantCV); !bytes.Equal(g, e) {
			t.Errorf("%s:\n got %s\nwant %s", c.what, g, e)
		}
	}
	// Seat 0 to move: the parade ground is the one card it cannot build,
	// and the colossus and the mausoleum (5 per clay) the wonders it cannot
	// afford, whichever card it would tuck.
	var dis []string
	for _, d := range vw.Disabled {
		if d.Kind == view.ChoiceKindConstruct || strings.HasSuffix(d.Key, ":walls") {
			dis = append(dis, d.Key+" "+string(d.Reason))
		}
	}
	wantDis := []string{"wonder:colossus:walls unaffordable", "wonder:mausoleum:walls unaffordable", "construct:parade-ground unaffordable"}
	if !slices.Equal(dis, wantDis) {
		t.Errorf("disabled %v, want %v", dis, wantDis)
	}
}

// vwJSON marshals v or fails the test.
func vwJSON(t *testing.T, v any) []byte {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal %T: %v", v, err)
	}
	return data
}

// TestViewDraft follows the draft section through a draft: the first
// round's wonders, who took each, the second round revealed only after the
// first, and no draft section once Age I is laid out.
func TestViewDraft(t *testing.T) {
	s := engine.NewGame(protoSeed(101), engine.Options{FirstPlayer: engine.FirstPlayerSeat1})
	offer := func(vw *view.View) []string {
		var out []string
		for _, o := range vw.Draft.Offer {
			taker := "-"
			if o.TakenBy != nil {
				taker = strconv.Itoa(*o.TakenBy)
			}
			out = append(out, o.Wonder+"/"+taker)
		}
		return out
	}
	w := func(i int) string { return s.DraftOrder[i].String() }
	vw := engine.ViewFor(&s, engine.Spectator)
	if vw.Draft == nil || vw.Draft.Round != 1 || len(vw.Structure) != 0 || vw.Age != 0 {
		t.Fatalf("start: draft %+v, %d slots, age %d", vw.Draft, len(vw.Structure), vw.Age)
	}
	if got, want := offer(&vw), []string{w(0) + "/-", w(1) + "/-", w(2) + "/-", w(3) + "/-"}; !slices.Equal(got, want) {
		t.Fatalf("round 1 offer %v, want %v", got, want)
	}
	enginetest.Do(t, &s, "draft:"+w(2))
	vw = engine.ViewFor(&s, engine.Spectator)
	if got, want := offer(&vw), []string{w(0) + "/-", w(1) + "/-", w(2) + "/1", w(3) + "/-"}; !slices.Equal(got, want) {
		t.Fatalf("after one pick: %v, want %v", got, want)
	}
	enginetest.Do(t, &s, "draft:"+w(0), "draft:"+w(1))
	vw = engine.ViewFor(&s, engine.Spectator)
	if vw.Draft == nil || vw.Draft.Round != 2 {
		t.Fatalf("after round 1: draft %+v", vw.Draft)
	}
	if got, want := offer(&vw), []string{w(4) + "/-", w(5) + "/-", w(6) + "/-", w(7) + "/-"}; !slices.Equal(got, want) {
		t.Fatalf("round 2 offer %v, want %v", got, want)
	}
	if got := []string{vw.Players[0].Wonders[0].Wonder, vw.Players[0].Wonders[1].Wonder}; !slices.Equal(got, []string{w(0), w(1)}) {
		t.Fatalf("seat 0 drafted %v", got)
	}
	enginetest.Do(t, &s, "draft:"+w(4), "draft:"+w(5), "draft:"+w(6))
	vw = engine.ViewFor(&s, engine.Spectator)
	if vw.Draft != nil || vw.Phase != view.PhasePlay || vw.Age != 1 || len(vw.Structure) != engine.SlotsPerAge {
		t.Fatalf("after the draft: draft %+v, phase %s, age %d, %d slots", vw.Draft, vw.Phase, vw.Age, len(vw.Structure))
	}
}

// TestViewLibraryOffer checks the Great Library offer from both sides: the
// drawer sees the three tokens and can take each; the opponent and a
// spectator see only that three were drawn, and the box count drops by
// three for everyone.
func TestViewLibraryOffer(t *testing.T) {
	s, _ := protoNamed(t, protoSeed(102), "great-library", engine.Options{}, 1)
	if s.Phase != engine.PhasePickLibraryToken || s.ToMove != 0 {
		t.Fatalf("phase %s, to move %d", s.Phase, s.ToMove)
	}
	for _, v := range redactViewers {
		vw := engine.ViewFor(&s, v)
		p := vw.Pending
		var keys []string
		for _, c := range vw.Choices {
			keys = append(keys, c.Key)
		}
		if v == 0 {
			if !slices.Equal(p.Tokens, []string{"law", "philosophy", "strategy"}) || p.HiddenTokens != 0 ||
				!slices.Equal(keys, []string{"token:law", "token:philosophy", "token:strategy"}) {
				t.Errorf("the drawer sees %+v and choices %v", *p, keys)
			}
		} else if len(p.Tokens) != 0 || p.HiddenTokens != 3 || len(keys) != 0 {
			t.Errorf("viewer %d sees %+v and choices %v", v, *p, keys)
		}
		if vw.BoxTokenCount != 2 {
			t.Errorf("viewer %d: box count %d, want 2", v, vw.BoxTokenCount)
		}
	}
}

// TestViewRevealedAtEnd checks the revealed section of a finished game
// against the state, and that HideAtEnd leaves it out.
func TestViewRevealedAtEnd(t *testing.T) {
	for _, hide := range []bool{false, true} {
		s, _ := protoNamed(t, protoSeed(103), "civilian-end", engine.Options{HideAtEnd: hide}, -1)
		if !s.Over() {
			t.Fatalf("the game is not over")
		}
		vw := engine.ViewFor(&s, 1)
		if hide {
			if vw.Revealed != nil {
				t.Errorf("HideAtEnd, but revealed %+v", *vw.Revealed)
			}
			continue
		}
		r := vw.Revealed
		if r == nil {
			t.Fatalf("nothing revealed")
		}
		var removed []view.AgeCards
		for a := range engine.NumAges {
			var cards []string
			for _, c := range s.Removed[a] {
				if c != 0 {
					cards = append(cards, c.String())
				}
			}
			if len(cards) > 0 {
				removed = append(removed, view.AgeCards{Age: a + 1, Cards: cards})
			}
		}
		var guilds, wonders, tokens []string
		for _, c := range s.BoxGuilds {
			if c != 0 {
				guilds = append(guilds, c.String())
			}
		}
		for _, w := range s.BoxWonders {
			wonders = append(wonders, w.String())
		}
		for _, tk := range s.BoxTokens {
			if tk != 0 {
				tokens = append(tokens, tk.String())
			}
		}
		if !bytes.Equal(vwJSON(t, r.Removed), vwJSON(t, removed)) || !slices.Equal(r.BoxGuilds, guilds) ||
			!slices.Equal(r.BoxWonders, wonders) || !slices.Equal(r.BoxTokens, tokens) || len(r.Undealt) != 0 || len(r.FaceDown) != 0 {
			t.Errorf("revealed %+v; want removed %v, guilds %v, wonders %v, tokens %v", *r, removed, guilds, wonders, tokens)
		}
		if len(wonders) != engine.NumBoxWonders || len(tokens) != engine.NumBoxTokens {
			t.Errorf("box wonders %v and tokens %v", wonders, tokens)
		}
	}
}

// TestViewDoesNotAlias checks that a view keeps no reference into the
// state: its JSON stays the same after the state moves on.
func TestViewDoesNotAlias(t *testing.T) {
	s := protoCostsState(t, protoSeed(104))
	vw := engine.ViewFor(&s, 0)
	before := vwJSON(t, vw)
	enginetest.Do(t, &s, "construct:walls")
	if after := vwJSON(t, vw); !bytes.Equal(before, after) {
		t.Fatalf("the view changed with the state:\n%s\n%s", before, after)
	}
}

// TestViewerSeat checks the viewer helpers.
func TestViewerSeat(t *testing.T) {
	for _, c := range []struct {
		v      engine.Viewer
		seat   engine.Seat
		seated bool
	}{{0, 0, true}, {1, 1, true}, {engine.Spectator, 0, false}, {2, 0, false}, {-7, 0, false}} {
		seat, seated := c.v.Seat()
		if seat != c.seat || seated != c.seated {
			t.Errorf("Viewer(%d).Seat() = %d, %v", c.v, seat, seated)
		}
	}
	if engine.SeatViewer(1) != 1 || engine.SeatViewer(0) != 0 {
		t.Errorf("SeatViewer")
	}
	// Any value other than 0 and 1 is a spectator in the view too.
	s := engine.NewGame(protoSeed(105), engine.Options{})
	if vw := engine.ViewFor(&s, 5); vw.Viewer != nil || len(vw.Choices) != 0 {
		t.Errorf("viewer 5 gets viewer %v and %d choices", vw.Viewer, len(vw.Choices))
	}
}

// TestCatalogHash checks the catalog fingerprint: 16 lower-case hex digits,
// stable across calls, and different as soon as any field of any item
// changes.
func TestCatalogHash(t *testing.T) {
	h := engine.CatalogHash()
	if len(h) != 16 || strings.Trim(h, "0123456789abcdef") != "" {
		t.Fatalf("CatalogHash() = %q, want 16 lower-case hex digits", h)
	}
	if engine.CatalogHash() != h {
		t.Fatalf("CatalogHash is not stable")
	}
	var cards []catalog.CardDef
	for d := range catalog.Cards() {
		cards = append(cards, *d)
	}
	var wonders []catalog.WonderDef
	for d := range catalog.Wonders() {
		wonders = append(wonders, *d)
	}
	var tokens []catalog.TokenDef
	for d := range catalog.Tokens() {
		tokens = append(tokens, *d)
	}
	hashOf := func(c []catalog.CardDef, w []catalog.WonderDef, k []catalog.TokenDef) string {
		restore := catalog.ReplaceForTest(c, w, k)
		defer restore()
		return engine.CatalogHash()
	}
	if got := hashOf(cards, wonders, tokens); got != h {
		t.Fatalf("the same data hashes to %s, not %s", got, h)
	}
	if cards[22].Slug != "baths" {
		t.Fatalf("card 23 is %s, not baths", cards[22].Slug)
	}
	changes := []struct {
		what string
		edit func(c []catalog.CardDef, w []catalog.WonderDef, k []catalog.TokenDef)
	}{
		{"a card's cost", func(c []catalog.CardDef, _ []catalog.WonderDef, _ []catalog.TokenDef) {
			c[22].Cost.Res[catalog.Stone] = 2
		}},
		{"a card's name", func(c []catalog.CardDef, _ []catalog.WonderDef, _ []catalog.TokenDef) { c[22].Name = "Bath" }},
		{"a card's effect", func(c []catalog.CardDef, _ []catalog.WonderDef, _ []catalog.TokenDef) {
			c[22].Effects = slices.Clone(c[22].Effects)
			c[22].Effects[0].N++
		}},
		{"a wonder's cost", func(_ []catalog.CardDef, w []catalog.WonderDef, _ []catalog.TokenDef) { w[0].Cost.Coins = 1 }},
		{"a token's name", func(_ []catalog.CardDef, _ []catalog.WonderDef, k []catalog.TokenDef) { k[0].Name = "Farming" }},
	}
	for _, ch := range changes {
		c, w, k := slices.Clone(cards), slices.Clone(wonders), slices.Clone(tokens)
		ch.edit(c, w, k)
		if got := hashOf(c, w, k); got == h {
			t.Errorf("changing %s leaves the hash at %s", ch.what, h)
		}
	}
	if engine.CatalogHash() != h {
		t.Fatalf("the hash did not come back after the catalog was restored")
	}
}

// ---------------------------------------------------------------------------
// Protocol fixtures and golden views.

// protoFixture is one recorded position: the state and the events of the
// action that led to it (none for a starting position).
type protoFixture struct {
	name, about string
	state       engine.State
	events      []engine.Event
}

// protoFile is the JSON of one fixture or golden view file: each viewer's
// view and filtered events.
type protoFile struct {
	Name    string        `json:"name"`
	About   string        `json:"about"`
	Viewers []protoViewer `json:"viewers"`
}

// protoViewer is what one viewer receives. Viewer is "seat0", "seat1" or
// "spectator".
type protoViewer struct {
	Viewer string           `json:"viewer"`
	View   view.View        `json:"view"`
	Events []view.GameEvent `json:"events"`
}

// protoSeed returns the seed of fixture i: a large number, so that the
// redaction test's seed scan cannot match an unrelated number by chance.
func protoSeed(i int) uint64 { return 0xa7a4eb5eed000000 ^ (uint64(i)+1)*0x9e3779b97f4a7c15 }

// protoCases lists the fixtures. Between them they reach every phase (the
// test checks it) and every kind of event.
var protoCases = []struct {
	name, about string
	build       func(t *testing.T, seed uint64) (engine.State, []engine.Event)
}{
	{"draft-round-1", "the wonder draft after the first pick; the second round is still hidden",
		func(t *testing.T, seed uint64) (engine.State, []engine.Event) { return protoDraft(t, seed, 1) }},
	{"draft-round-2", "the second draft round, revealed after the first round's automatic last pick",
		func(t *testing.T, seed uint64) (engine.State, []engine.Event) { return protoDraft(t, seed, 3) }},
	{"age-1-start", "Age I laid out after the draft: face-down rows show only their backs",
		func(t *testing.T, seed uint64) (engine.State, []engine.Event) { return protoDraft(t, seed, 6) }},
	{"age-1-reveal", "two cards discarded from the bottom row free a face-down card, which is revealed", protoReveal},
	{"costs", "Age II with every part of a cost breakdown, for both seats, and an unaffordable card",
		func(t *testing.T, seed uint64) (engine.State, []engine.Event) { return protoCostsState(t, seed), nil }},
	{"choose-starter", "the Sphinx's extra turn on Age I's last card is lost, and the last active player chooses who starts Age II",
		protoNamedCase("extra-turn-lost", engine.Options{}, -1)},
	{"starter-chosen", "the trailing seat chose who starts Age II",
		protoNamedCase("age-transition-chooser", engine.Options{}, -1)},
	{"destroy-card", "the Statue of Zeus opened the choice of a brown card to destroy",
		protoNamedCase("zeus-destroy", engine.Options{}, 1)},
	{"card-destroyed", "the opponent's Quarry destroyed",
		protoNamedCase("zeus-destroy", engine.Options{}, -1)},
	{"build-from-discard", "the Mausoleum opened the choice of a discarded card to build",
		protoNamedCase("mausoleum", engine.Options{}, 1)},
	{"card-revived", "Baths built from the discard pile",
		protoNamedCase("mausoleum", engine.Options{}, -1)},
	{"pick-board-token", "a science pair opened the choice of a board token",
		protoNamedCase("science-pair", engine.Options{}, -1)},
	{"pick-library-token", "the Great Library drew three tokens that only the drawer sees",
		protoNamedCase("great-library", engine.Options{}, 1)},
	{"library-token-taken", "the drawer kept Law; the other two went back to the box unseen",
		protoNamedCase("great-library", engine.Options{}, -1)},
	{"looting", "four shields take both of seat 1's looting tokens",
		protoNamedCase("looting-double", engine.Options{}, -1)},
	{"appian-economy", "the Appian Way bought by trade: the opponent's Economy takes the coins, then loses 3",
		protoNamedCase("appian-economy", engine.Options{}, -1)},
	{"seventh-wonder", "the seventh wonder built: seat 1's last unbuilt wonder leaves the game", protoSeventh},
	{"effect-skipped", "the Statue of Zeus with no brown card to destroy skips its choice",
		protoNamedCase("zeus-no-target", engine.Options{}, -1)},
	{"game-over-civilian", "the last card of Age III ends the game on points; the hidden cards are revealed",
		protoNamedCase("civilian-end", engine.Options{}, -1)},
	{"game-over-shared", "equal points and equal blue points: a shared victory",
		protoNamedCase("draw", engine.Options{}, -1)},
	{"game-over-science", "Law from the board completes six science symbols",
		protoNamedCase("law-supremacy", engine.Options{}, -1)},
	{"game-over-military-hidden", "the pawn reaches the capital in a game that keeps its secrets after the end",
		protoNamedCase("military-supremacy", engine.Options{HideAtEnd: true}, -1)},
}

// protoFixtures builds every fixture.
func protoFixtures(t *testing.T) []protoFixture {
	t.Helper()
	out := make([]protoFixture, 0, len(protoCases))
	for i, c := range protoCases {
		s, ev := c.build(t, protoSeed(i))
		if err := engine.CheckInvariants(&s); err != nil {
			t.Fatalf("fixture %s: %v", c.name, err)
		}
		out = append(out, protoFixture{name: c.name, about: c.about, state: s, events: ev})
	}
	return out
}

// protoDraft plays the first picks of a draft from a plain deal with seat 0
// first, each pick taking the first wonder on offer, and returns the state
// and the last pick's events. picks counts the players' picks, not the
// automatic last pick of each round.
func protoDraft(t *testing.T, seed uint64, picks int) (engine.State, []engine.Event) {
	t.Helper()
	s := engine.NewGame(seed, engine.Options{FirstPlayer: engine.FirstPlayerSeat0})
	var ev []engine.Event
	for range picks {
		ev = enginetest.Do(t, &s, enginetest.Keys(&s)[0])
	}
	return s, ev
}

// protoReveal discards the two leftmost cards of Age I's bottom row, which
// frees the face-down card above them.
func protoReveal(t *testing.T, seed uint64) (engine.State, []engine.Event) {
	t.Helper()
	s, _ := protoDraft(t, seed, 6)
	enginetest.Do(t, &s, "discard:"+s.Slots[14].Card.String())
	ev := enginetest.Do(t, &s, "discard:"+s.Slots[15].Card.String())
	return s, ev
}

// protoNamedCase returns a fixture builder for a named scenario.
func protoNamedCase(name string, opts engine.Options, steps int) func(*testing.T, uint64) (engine.State, []engine.Event) {
	return func(t *testing.T, seed uint64) (engine.State, []engine.Event) {
		return protoNamed(t, seed, name, opts, steps)
	}
}

// protoNamed builds the named scenario and applies the first steps keys of
// its trigger (all of them when steps is negative), returning the state and
// the last key's events.
func protoNamed(t *testing.T, seed uint64, name string, opts engine.Options, steps int) (engine.State, []engine.Event) {
	t.Helper()
	sc := engine.NamedScenario(name)
	if sc == nil {
		t.Fatalf("no named scenario %q", name)
	}
	s, err := engine.NewGameFromScenario(seed, opts, sc)
	if err != nil {
		t.Fatalf("scenario %s: %v", name, err)
	}
	if steps < 0 || steps > len(sc.Trigger) {
		steps = len(sc.Trigger)
	}
	var ev []engine.Event
	if steps > 0 {
		enginetest.Do(t, &s, sc.Trigger[:steps-1]...)
		ev = enginetest.Do(t, &s, sc.Trigger[steps-1])
	}
	return s, ev
}

// protoCostsState is an Age II position built to show every part of a cost
// breakdown (TestViewCostBreakdown works its costs out by hand).
func protoCostsState(t *testing.T, seed uint64) engine.State {
	t.Helper()
	return enginetest.New(t, seed).Name("cost breakdowns").Age(2).
		Wonders(1, "great-lighthouse").Built(1, "great-lighthouse", "").
		City(0, "lumber-yard", "glassworks", "forum", "stone-reserve", "theater").Tokens(0, "masonry", "urbanism").Coins(0, 9).
		City(1, "clay-pool", "brickyard").Tokens(1, "economy", "architecture").Coins(1, 7).
		OnlyLeft("courthouse", "statue", "aqueduct", "caravansery", "walls", "parade-ground").ToMove(0).State()
}

// protoSeventh builds the seventh wonder of the game: seat 0 has three
// wonders built and seat 1 three, and seat 0 builds the Sphinx.
func protoSeventh(t *testing.T, seed uint64) (engine.State, []engine.Event) {
	t.Helper()
	s := enginetest.New(t, seed).Name("the seventh wonder").Age(2).
		Wonders(0, "pyramids", "colossus", "great-lighthouse", "sphinx").
		Wonders(1, "temple-of-artemis", "mausoleum", "great-library", "piraeus").
		Built(0, "pyramids", "").Built(0, "colossus", "").Built(0, "great-lighthouse", "").
		Built(1, "temple-of-artemis", "").Built(1, "mausoleum", "").Built(1, "great-library", "").
		Coins(0, 40).Accessible("rostrum").ToMove(0).State()
	ev := enginetest.Do(t, &s, "wonder:sphinx:rostrum")
	return s, ev
}

// protoViewerNames names the viewers of redactViewers in the files.
var protoViewerNames = []string{"seat0", "seat1", "spectator"}

// protoFileOf records what each viewer receives in s after events ev.
func protoFileOf(name, about string, s *engine.State, ev []engine.Event) protoFile {
	f := protoFile{Name: name, About: about, Viewers: make([]protoViewer, 0, len(redactViewers))}
	for i, v := range redactViewers {
		f.Viewers = append(f.Viewers, protoViewer{Viewer: protoViewerNames[i], View: engine.ViewFor(s, v), Events: engine.FilterEvents(ev, v)})
	}
	return f
}

// TestProtocolFixtures writes or checks testdata/protocol and checks that
// the fixtures cover every phase, every kind of choice and every kind of
// event, so the client's tests see each shape at least once.
func TestProtocolFixtures(t *testing.T) {
	fixtures := protoFixtures(t)
	phases := map[string]bool{}
	choices := map[view.ChoiceKind]bool{}
	events := map[view.EventKind]bool{}
	var names []string
	for _, fx := range fixtures {
		f := protoFileOf(fx.name, fx.about, &fx.state, fx.events)
		for _, pv := range f.Viewers {
			phases[string(pv.View.Phase)] = true
			for _, c := range pv.View.Choices {
				choices[c.Kind] = true
			}
			for _, e := range pv.Events {
				events[e.Kind] = true
			}
		}
		names = append(names, fx.name+".json")
		vwGolden(t, filepath.Join("testdata", "protocol", fx.name+".json"), f)
	}
	consts := vwViewConsts(t)
	for _, p := range consts["Phase"] {
		if !phases[p] {
			t.Errorf("no protocol fixture is in phase %s", p)
		}
	}
	for _, k := range consts["ChoiceKind"] {
		if !choices[view.ChoiceKind(k)] {
			t.Errorf("no protocol fixture offers a %s choice", k)
		}
	}
	for _, k := range consts["EventKind"] {
		if !events[view.EventKind(k)] {
			t.Errorf("no protocol fixture has a %s event", k)
		}
	}
	vwNoStrayFiles(t, filepath.Join("testdata", "protocol"), names)
}

// TestViewGoldens plays a plain seeded game and records the views of every
// viewer at fixed points in testdata/views: the start of the draft, the
// first turn of each age, and the end.
func TestViewGoldens(t *testing.T) {
	const seed uint64 = 0x5eed_a1a7_eb00_0007
	s := engine.NewGame(seed, engine.Options{})
	pols := [2]enginetest.Policy{enginetest.NewSimplePolicy(seed), enginetest.NewSimplePolicy(seed + 1)}
	points := []struct {
		name, about string
		at          func(s *engine.State) bool
	}{
		{"start", "the start of the wonder draft", func(s *engine.State) bool { return true }},
		{"age-1", "the first turn of Age I", func(s *engine.State) bool { return s.Age == 1 }},
		{"age-2", "the first decision of Age II", func(s *engine.State) bool { return s.Age == 2 }},
		{"age-3", "the first decision of Age III", func(s *engine.State) bool { return s.Age == 3 }},
		{"end", "the end of the game", func(s *engine.State) bool { return s.Over() }},
	}
	var names []string
	var ev []engine.Event
	buf := make([]engine.Action, 0, engine.MaxLegalActions)
	next := 0
	for n := 0; next < len(points); n++ {
		if points[next].at(&s) {
			p := points[next]
			vwGolden(t, filepath.Join("testdata", "views", p.name+".json"), protoFileOf(p.name, p.about, &s, ev))
			names = append(names, p.name+".json")
			next++
			continue
		}
		if s.Over() || n > 300 {
			t.Fatalf("the game ended (or ran past %d actions) before %s; pick a seed that reaches every point", n, points[next].name)
		}
		a := pols[s.ToMove].Choose(&s, engine.LegalActions(&s, buf[:0]))
		ev = nil
		if err := engine.Apply(&s, a, &ev); err != nil {
			t.Fatalf("%s: %v", a, err)
		}
	}
	vwNoStrayFiles(t, filepath.Join("testdata", "views"), names)
}

// vwGolden compares v, marshaled with indentation, with the file at path,
// or writes it there with -update.
func vwGolden(t *testing.T, path string, v any) {
	t.Helper()
	data, err := json.MarshalIndent(v, "", " ")
	if err != nil {
		t.Fatalf("marshal %s: %v", path, err)
	}
	data = append(data, '\n')
	if *updateGolden {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v (run go test ./internal/engine/ -run '%s' -update to create it)", path, err, t.Name())
	}
	if !bytes.Equal(data, want) {
		t.Errorf("%s is stale (after a deliberate change, run go test ./internal/engine/ -run '%s' -update):\n%s",
			path, t.Name(), vwFirstDiff(data, want))
	}
}

// vwFirstDiff describes the first differing line of two files.
func vwFirstDiff(got, want []byte) string {
	g, w := strings.Split(string(got), "\n"), strings.Split(string(want), "\n")
	for i := range max(len(g), len(w)) {
		var a, b string
		if i < len(g) {
			a = g[i]
		}
		if i < len(w) {
			b = w[i]
		}
		if a != b {
			return "line " + strconv.Itoa(i+1) + ":\n  got  " + a + "\n  file " + b
		}
	}
	return "(no line differs)"
}

// vwNoStrayFiles fails when dir holds a file the test did not write, such
// as the fixture of a case that was renamed or removed.
func vwNoStrayFiles(t *testing.T, dir string, names []string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if !slices.Contains(names, e.Name()) {
			if *updateGolden {
				if err := os.Remove(filepath.Join(dir, e.Name())); err != nil {
					t.Fatal(err)
				}
				continue
			}
			t.Errorf("%s/%s is not written by any case; remove it or run with -update", dir, e.Name())
		}
	}
}
