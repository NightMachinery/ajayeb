package fakeengine

import (
	"errors"
	"strconv"
)

// Errors returned by NewGame, Replay and Resolve. An error never changes
// the game.
var (
	ErrSetup      = errors.New("fakeengine: invalid setup")
	ErrOver       = errors.New("fakeengine: the game is over")
	ErrWrongPoint = errors.New("fakeengine: the batch is for another point")
	ErrBatch      = errors.New("fakeengine: the batch does not hold one pick per required seat, sorted by seat")
	ErrIllegal    = errors.New("fakeengine: a key is not offered to its seat")
)

// NewGame validates the setup and resolves every random element from its
// own stream: the boards, Age 1's deal, and Age 2's deal and box. Treasure
// starts at StartCoins per seat. Age 1 is dealt at once (the fake has no
// side point), so the game opens on its first turn.
func NewGame(s Setup) (Game, error) {
	if s.Seats < MinSeats || s.Seats > MaxSeats {
		return Game{}, ErrSetup
	}
	if s.Options.Seating != SeatingRandom && s.Options.Seating != SeatingChosen {
		return Game{}, ErrSetup
	}
	var g Game
	st := &g.State
	st.Seed = s.Seed
	st.Opts = s.Options
	st.Seats = s.Seats

	boards := [NumBoards]uint8{0, 1, 2, 3, 4, 5, 6}
	rb := newStream(s.Seed, tagBoards)
	rb.shuffleBytes(boards[:])
	for i := Seat(0); uint8(i) < s.Seats; i++ {
		st.Cities[i].Board = BoardID(boards[i])
		st.Cities[i].Coins = StartCoins
	}

	deck1 := deckFor(1, s.Seats)
	r1 := newStream(s.Seed, tagDeck1)
	r1.shuffleDesigns(deck1)
	for i := Seat(0); uint8(i) < s.Seats; i++ {
		copy(st.Hands[i][:], deck1[int(i)*HandSize:int(i+1)*HandSize])
		sortHand(&st.Hands[i])
	}

	deck2 := deckFor(2, s.Seats)
	r2 := newStream(s.Seed, tagDeck2)
	r2.shuffleDesigns(deck2)
	for i := Seat(0); uint8(i) < s.Seats; i++ {
		copy(st.Deals[i][:], deck2[int(i)*HandSize:int(i+1)*HandSize])
		sortHand(&st.Deals[i])
	}
	copy(st.Box[:], deck2[int(s.Seats)*HandSize:])
	sortBox(&st.Box)

	st.Age, st.Turn, st.Phase = 1, 1, PhaseTurn
	g.recordHeld()
	return g, nil
}

// deckFor lists an Age's copies for n seats, by design id: every design of
// the Age once per copy whose minimum seat count is at most n, plus the
// relics in Age 2.
func deckFor(age, n uint8) []DesignID {
	deck := []DesignID{}
	for d := DesignID(1); d <= NumDesigns; d++ {
		if designs[d].Age != age {
			continue
		}
		for c := copiesFor(d, n); c > 0; c-- {
			deck = append(deck, d)
		}
	}
	return deck
}

// SeatOrder returns the initial assignment of n lobby slots to seats:
// entry s is the lobby slot seated at seat s. With SeatingChosen it is lobby
// order; with SeatingRandom a permutation from the seed's own stream. The
// actor persists it at Start, and restore recomputes it. Entries from n on,
// and every entry for an n outside MinSeats to MaxSeats, are zero.
func SeatOrder(seed uint64, n uint8, rule SeatingRule) [MaxSeats]uint8 {
	var out [MaxSeats]uint8
	if n < MinSeats || n > MaxSeats {
		return out
	}
	for i := uint8(0); i < n; i++ {
		out[i] = i
	}
	if rule == SeatingRandom {
		r := newStream(seed, tagSeating)
		r.shuffleBytes(out[:n])
	}
	return out
}

// Replay creates the game from its setup and resolves each batch in order,
// checking each batch's point, required seats and key membership. Arrival
// order never reaches the engine; the caller compares fingerprints.
func Replay(s Setup, batches []Batch) (Game, error) {
	g, err := NewGame(s)
	if err != nil {
		return Game{}, err
	}
	for i, b := range batches {
		next, _, err := g.Resolve(b)
		if err != nil {
			return Game{}, errors.Join(errors.New("fakeengine: replaying batch "+strconv.Itoa(i)), err)
		}
		g = next
	}
	return g, nil
}

// setup returns the game's own setup, for History's replay.
func (g *Game) setup() Setup {
	return Setup{Seed: g.State.Seed, Seats: g.State.Seats, Options: g.State.Opts}
}

// recordHeld notes every seat's hand at the opening of the current turn.
func (g *Game) recordHeld() {
	st := &g.State
	for s := Seat(0); uint8(s) < st.Seats; s++ {
		g.Record.Held[st.Age-1][st.Turn-1][s] = st.Hands[s]
	}
}

// sortHand sorts a hand ascending, with the empty slots last.
func sortHand(h *[HandSize]DesignID) {
	for i := 1; i < HandSize; i++ {
		for j := i; j > 0 && less(h[j], h[j-1]); j-- {
			h[j], h[j-1] = h[j-1], h[j]
		}
	}
}

// sortBox sorts the box the same way.
func sortBox(b *[MaxBox]DesignID) {
	for i := 1; i < MaxBox; i++ {
		for j := i; j > 0 && less(b[j], b[j-1]); j-- {
			b[j], b[j-1] = b[j-1], b[j]
		}
	}
}

// less orders designs ascending with zero (no card) last.
func less(a, b DesignID) bool {
	if a == 0 {
		return false
	}
	return b == 0 || a < b
}

// removeOne takes one copy of d out of a sorted hand.
func removeOne(h *[HandSize]DesignID, d DesignID) bool {
	for i, x := range h {
		if x == d {
			h[i] = 0
			sortHand(h)
			return true
		}
	}
	return false
}

// handCount returns the number of cards in a hand.
func handCount(h *[HandSize]DesignID) uint8 {
	var n uint8
	for _, x := range h {
		if x != 0 {
			n++
		}
	}
	return n
}
