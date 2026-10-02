package engine

import "github.com/NightMachinery/ajayeb/server/internal/engine/catalog"

// This file scores the game and ends it: the eight civilian scoring
// categories, the tie-break, the supremacy check of the end of a turn, and
// winNow, which every early end of the game goes through.
//
// The eight categories of a player's score, as on the score sheet:
//
//   - Blue, Green, Yellow: the printed points of the player's buildings of
//     that colour;
//   - Purple: the guilds' points, each guild counting the cities as they
//     are now (most of them the city with more of what they count, which
//     may be the opponent's);
//   - Wonders: the points of the player's built wonders (a removed wonder
//     was never built and scores nothing);
//   - Tokens: the printed points of the player's tokens (Agriculture,
//     Philosophy) and the points per token owned (Mathematics, counting
//     itself);
//   - Coins: one point per full three coins;
//   - Military: the points of the pawn's zone, for the player it favours.
//
// The brown, grey and red buildings carry no points in the game (a catalog
// test checks it), so they have no category.

// coinsPerPoint is how many coins make one point at the end.
const coinsPerPoint = 3

// Scores returns both players' scores, by category and in total, as if the
// game ended now. It is the score preview of the view and the score sheet of
// a finished game (Result.Scores).
func Scores(s *State) [2]Score {
	return [2]Score{scoreOf(s, 0), scoreOf(s, 1)}
}

// scoreOf scores one seat.
func scoreOf(s *State, seat Seat) Score {
	p, o := &s.Players[seat], &s.Players[seat.Opp()]
	t := &catalog.T
	var sc Score
	for c, ok := p.City.Next(1); ok; c, ok = p.City.Next(c + 1) {
		if int(c) >= len(t.VP) {
			continue
		}
		vp := int16(t.VP[c])
		switch t.Color[c] {
		case catalog.Blue:
			sc.Blue += vp
		case catalog.Green:
			sc.Green += vp
		case catalog.Yellow:
			sc.Yellow += vp
		case catalog.Purple:
			sc.Purple += vp
		}
		sc.Purple += cardEndPoints(p, o, c)
	}
	for i := range p.Wonders {
		if ws := &p.Wonders[i]; ws.Built && int(ws.ID) < len(t.WVP) {
			sc.Wonders += int16(t.WVP[ws.ID])
		}
	}
	nTokens := int16(p.Tokens.Count())
	for tk, ok := p.Tokens.Next(1); ok; tk, ok = p.Tokens.Next(tk + 1) {
		if int(tk) < len(t.TokenVP) {
			sc.Tokens += int16(t.TokenVP[tk])
		}
		if d := catalog.Token(tk); d != nil {
			if e, ok := catalog.FindEffect(d.Effects, catalog.EffPointsPerToken); ok {
				sc.Tokens += int16(e.N) * nTokens
			}
		}
	}
	sc.Coins = p.Coins / coinsPerPoint
	sc.Military = militaryPoints(s.Military, seat)
	sc.Total = sc.Blue + sc.Green + sc.Yellow + sc.Purple + sc.Wonders + sc.Tokens + sc.Coins + sc.Military
	return sc
}

// cardEndPoints returns the end-of-game points of building c's counting
// effects (the guilds), for its owner p against the opponent o, from the
// cities as they are now.
func cardEndPoints(p, o *Player, c catalog.CardID) int16 {
	d := catalog.Card(c)
	if d == nil {
		return 0
	}
	n := 0
	for _, e := range d.Effects {
		switch e.Kind {
		case catalog.EffPointsPerCard:
			n += int(e.N) * scoped(e.Scope, colorCount(p, e.Colors), colorCount(o, e.Colors))
		case catalog.EffPointsPerWonder:
			n += int(e.N) * scoped(e.Scope, builtWonders(p), builtWonders(o))
		case catalog.EffPointsPerCoins:
			n += scoped(e.Scope, int(p.Coins), int(o.Coins)) / int(e.N)
		}
	}
	return int16(n)
}

// civilianResult scores the game at the end of Age III: the higher total
// wins; equal totals go to the higher blue-card points; equal blue points
// too make a shared victory.
func civilianResult(s *State) Result {
	sc := Scores(s)
	return Result{Kind: VictoryCivilian, Winner: civilianWinner(&sc), Scores: sc}
}

// civilianWinner applies the comparison and the tie-break to a score sheet.
func civilianWinner(sc *[2]Score) int8 {
	a, b := &sc[0], &sc[1]
	switch {
	case a.Total > b.Total:
		return 0
	case a.Total < b.Total:
		return 1
	case a.Blue > b.Blue:
		return 0
	case a.Blue < b.Blue:
		return 1
	}
	return SharedVictory
}

// winNow ends the game at once with winner's victory of the given kind (a
// supremacy): the result, with the score sheet of the moment for the
// record, PhaseGameOver, and EvGameOver (A = the kind, B = the winner). An
// extra turn still pending is dropped.
func winNow(s *State, kind VictoryKind, winner Seat, ev *[]Event) {
	s.Result = Result{Kind: kind, Winner: int8(winner), Scores: Scores(s)}
	s.Phase = PhaseGameOver
	s.ExtraTurn = false
	emit(ev, Event{Kind: EvGameOver, A: int16(kind), B: int16(winner)})
}

// checkSupremacy ends the game if the pawn stands on a capital or a player
// owns six different science symbols, and reports whether the game is over
// (true also when it had already ended). The effects end the game the
// moment a supremacy happens; this is the end-of-turn safety net.
func checkSupremacy(s *State, ev *[]Event) bool {
	if s.Over() {
		return true
	}
	switch {
	case s.Military >= MilitaryCapital:
		winNow(s, VictoryMilitary, 0, ev)
		return true
	case s.Military <= -MilitaryCapital:
		winNow(s, VictoryMilitary, 1, ev)
		return true
	}
	return scienceSupremacy(s, s.Active, ev) || scienceSupremacy(s, s.Active.Opp(), ev)
}
