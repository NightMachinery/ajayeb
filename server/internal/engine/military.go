package engine

// This file is the military track: moving the conflict pawn, the looting
// tokens, military supremacy, and the military points of the score.
//
// The pawn is one signed number, State.Military, from -9 to 9. Seat 0's
// shields push it up, toward seat 1's capital at +9; seat 1's shields push
// it down, toward seat 0's capital at -9. A move is walked one space at a
// time. A looting token sits on space 3 and space 6 of each side; the first
// time the pawn stands on or beyond a token's space on that side, the
// player of that side loses up to the token's value in coins to the bank
// (never going below zero), and the token leaves the game. One move can take
// both tokens of a side, the nearer one first. Reaching a capital ends the
// game at once with a military victory for the player who pushed; the rest
// of the move, and of the action, is dropped.

// MilitaryCapital is the distance from the centre of each capital: a pawn
// that reaches it wins the game.
const MilitaryCapital = 9

// lootToken is one looting token of the track.
type lootToken struct {
	bit    uint8 // its Loot* bit in State.Looting
	at     int8  // its space: 3 or 6 on seat 1's side (positive), -3 or -6 on seat 0's side
	victim Seat  // the player who loses coins when the pawn reaches it
	value  int16 // coins lost
}

// lootTokens lists the tokens, nearer ones first on each side, so that a
// long move takes them in the order the pawn reaches them.
var lootTokens = [4]lootToken{
	{bit: LootPlus3, at: 3, victim: 1, value: 2},
	{bit: LootPlus6, at: 6, victim: 1, value: 5},
	{bit: LootMinus3, at: -3, victim: 0, value: 2},
	{bit: LootMinus6, at: -6, victim: 0, value: 5},
}

// reached reports whether a pawn at pos stands on or beyond the token's
// space, on the token's side.
func (t *lootToken) reached(pos int8) bool {
	if t.at > 0 {
		return pos >= t.at
	}
	return pos <= t.at
}

// movePawn moves the pawn shields spaces toward seat's opponent, walking one
// space at a time and taking each looting token it reaches. It reports
// EvPawn (A = from, B = to, Seat = seat) for the move, then EvLoot for each
// token taken (Seat = the player who lost coins, A = the coins lost, B = the
// token's value). If the pawn reaches the opponent's capital, the move stops
// there and the game ends with seat's military victory (outcomeOver).
// Moving zero spaces does and reports nothing. Every shield counts in
// State.Shields, even those beyond a capital.
func movePawn(s *State, seat Seat, shields uint8, ev *[]Event) outcome {
	if shields == 0 {
		return outcomeFinish
	}
	s.Shields[seat] = int8(min(int(s.Shields[seat])+int(shields), 127))
	dir := 1
	if seat == 1 {
		dir = -1
	}
	from := int(s.Military)
	to := min(max(from+dir*int(shields), -MilitaryCapital), MilitaryCapital)
	emit(ev, Event{Kind: EvPawn, Seat: seat, A: int16(from), B: int16(to)})
	for pos := from; pos != to; {
		pos += dir
		s.Military = int8(pos)
		for i := range lootTokens {
			t := &lootTokens[i]
			if s.Looting&t.bit != 0 && t.reached(s.Military) {
				s.Looting &^= t.bit
				lost := takeCoins(s, t.victim, int(t.value))
				emit(ev, Event{Kind: EvLoot, Seat: t.victim, A: lost, B: t.value})
			}
		}
	}
	if to == MilitaryCapital || to == -MilitaryCapital {
		winNow(s, VictoryMilitary, seat, ev)
		return outcomeOver
	}
	return outcomeFinish
}

// militaryLeader returns the seat the pawn favours (the one whose opponent
// it stands nearer to), and false when the pawn is on the centre space.
func militaryLeader(pawn int8) (Seat, bool) {
	switch {
	case pawn > 0:
		return 0, true
	case pawn < 0:
		return 1, true
	}
	return 0, false
}

// militaryZonePoints are the points of the player ahead, by the pawn's
// distance from the centre: 2 for spaces 1 and 2, 5 for 3 to 5, 10 for 6 to
// 8. The capital (9) ends the game by supremacy; the score sheet then shows
// 10.
var militaryZonePoints = [MilitaryCapital + 1]int16{0, 2, 2, 5, 5, 5, 10, 10, 10, 10}

// militaryPoints returns the military points of seat with the pawn at pawn:
// the zone's points for the player ahead, nothing for the other.
func militaryPoints(pawn int8, seat Seat) int16 {
	lead, ok := militaryLeader(pawn)
	if !ok || lead != seat {
		return 0
	}
	d := int(pawn)
	if d < 0 {
		d = -d
	}
	return militaryZonePoints[min(d, MilitaryCapital)]
}
