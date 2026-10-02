package engine

import (
	"math/bits"

	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
)

// This file creates a game (NewGame) and holds the wonder draft's pick
// order. All chance of a game is resolved here, from the seed, in a fixed
// order that a golden test pins (TestNewGameSetupOrder):
//
//  1. Shuffle the wonders 1 to 12 (in id order): the first 8 become
//     DraftOrder (round 1, then round 2), the last 4 BoxWonders.
//  2. Shuffle the tokens 1 to 10: the first 5 go on the board, the last 5
//     are the box, in draw order.
//  3. Age I: shuffle its 23 cards (in id order); the first 20 are Deals[0]
//     in layout order, the last 3 Removed[0].
//  4. Age II: the same, into Deals[1] and Removed[1].
//  5. Age III: shuffle its 20 cards; the last 3 are Removed[2]. Shuffle the
//     7 guilds; the first 3 join the 17 kept cards, the last 4 are
//     BoxGuilds. Shuffle those 20 (the 17 kept cards, then the 3 guilds)
//     into Deals[2].
//  6. The first player: by Options.FirstPlayer, else one draw IntN(2).
//
// Every shuffle is RNG.Shuffle (Fisher-Yates from the last element). The
// first-player draw comes last, so fixing the first player by option leaves
// the deal unchanged.

// StartCoins is what each player starts with.
const StartCoins = 7

// The draft's pick order, relative to the first player: picks 0 to 3 are
// round 1 (first player, second, second, first) and picks 4 to 7 round 2
// (second, first, first, second). The last pick of each round (3 and 7) is
// automatic: the engine gives the last wonder of the round to its picker.
var draftPickers = [NumDraftWonders]Seat{0, 1, 1, 0, 1, 0, 0, 1}

// draftPicker returns the seat that makes pick n (0 to 7) of the draft.
func draftPicker(first Seat, n int) Seat { return draftPickers[n] ^ first }

// draftPicks returns how many draft picks have been made.
func draftPicks(s *State) int { return bits.OnesCount8(s.DraftTaken) }

// NewGame creates a game from a seed and options. It returns the state at the
// start of the wonder draft: round 1 is on offer, the first player is to
// move, each player has StartCoins coins, the pawn is in the centre and the
// four looting tokens are on the track. The ages are dealt but not laid out.
//
// NewGame needs the full catalog; it panics if the registered catalog does
// not have 23, 23 and 20 age cards and 7 guilds (only a test catalog swapped
// in with catalog.ReplaceForTest can cause that).
func NewGame(seed uint64, opts Options) State {
	var s State
	s.Seed = seed
	s.Opts = opts
	r := NewRNG(seed)

	// 1. Wonders.
	var w [catalog.NumWonders]catalog.WonderID
	for i := range w {
		w[i] = catalog.WonderID(i + 1)
	}
	r.Shuffle(len(w), func(i, j int) { w[i], w[j] = w[j], w[i] })
	copy(s.DraftOrder[:], w[:NumDraftWonders])
	copy(s.BoxWonders[:], w[NumDraftWonders:])

	// 2. Progress tokens.
	var tk [catalog.NumTokens]catalog.TokenID
	for i := range tk {
		tk[i] = catalog.TokenID(i + 1)
	}
	r.Shuffle(len(tk), func(i, j int) { tk[i], tk[j] = tk[j], tk[i] })
	for _, t := range tk[:NumBoardTokens] {
		s.BoardTokens.Add(t)
	}
	copy(s.BoxTokens[:], tk[NumBoardTokens:])

	// 3 and 4. Ages I and II.
	for age := 1; age <= 2; age++ {
		cards := setCards(catalog.T.AgeCards[age], SlotsPerAge+RemovedPerAge, age)
		r.Shuffle(len(cards), func(i, j int) { cards[i], cards[j] = cards[j], cards[i] })
		copy(s.Deals[age-1][:], cards[:SlotsPerAge])
		copy(s.Removed[age-1][:], cards[SlotsPerAge:])
	}

	// 5. Age III and the guilds.
	third := setCards(catalog.T.AgeCards[3], SlotsPerAge, 3)
	r.Shuffle(len(third), func(i, j int) { third[i], third[j] = third[j], third[i] })
	keep := SlotsPerAge - GuildsInPlay
	copy(s.Removed[2][:], third[keep:])
	guilds := setCards(catalog.T.Guilds, GuildsInPlay+NumBoxGuilds, 0)
	r.Shuffle(len(guilds), func(i, j int) { guilds[i], guilds[j] = guilds[j], guilds[i] })
	copy(s.BoxGuilds[:], guilds[GuildsInPlay:])
	deck := make([]catalog.CardID, 0, SlotsPerAge)
	deck = append(deck, third[:keep]...)
	deck = append(deck, guilds[:GuildsInPlay]...)
	r.Shuffle(len(deck), func(i, j int) { deck[i], deck[j] = deck[j], deck[i] })
	copy(s.Deals[2][:], deck)

	// 6. The first player.
	switch opts.FirstPlayer {
	case FirstPlayerSeat0:
		s.FirstSeat = 0
	case FirstPlayerSeat1:
		s.FirstSeat = 1
	default:
		s.FirstSeat = Seat(r.IntN(2))
	}

	for i := range s.Players {
		s.Players[i].Coins = StartCoins
	}
	s.Looting = LootAll
	s.Phase = PhaseWonderDraft
	s.ToMove = s.FirstSeat
	s.Active = s.FirstSeat
	return s
}

// setCards lists the cards of a set in ascending id order and checks that
// there are exactly n of them (age is for the panic message; 0 means the
// guilds).
func setCards(set catalog.CardSet, n, age int) []catalog.CardID {
	out := make([]catalog.CardID, 0, n)
	for c, ok := set.Next(1); ok; c, ok = set.Next(c + 1) {
		out = append(out, c)
	}
	if len(out) != n {
		what := "guilds"
		switch age {
		case 1:
			what = "Age I cards"
		case 2:
			what = "Age II cards"
		case 3:
			what = "Age III cards"
		}
		panic("engine: NewGame needs the full catalog, but it has the wrong number of " + what)
	}
	return out
}

// draftRound returns the draft round in progress (0 or 1) after n picks.
func draftRound(n int) int { return n / 4 }

// takeDraftWonder gives seat the wonder at index i of DraftOrder, in its
// first free wonder slot, and emits EvDraft (A = 1 for an automatic pick).
func takeDraftWonder(s *State, i int, seat Seat, auto bool, ev *[]Event) {
	w := s.DraftOrder[i]
	s.DraftTaken |= 1 << i
	p := &s.Players[seat]
	for k := range p.Wonders {
		if p.Wonders[k].ID == 0 {
			p.Wonders[k] = WonderSlot{ID: w}
			break
		}
	}
	e := Event{Kind: EvDraft, Seat: seat, Wonder: w}
	if auto {
		e.A = 1
	}
	emit(ev, e)
}

// lastOfRound returns the index in DraftOrder of the only wonder of round
// (0 or 1) not taken yet, or -1.
func lastOfRound(s *State, round int) int {
	last := -1
	for i := 4 * round; i < 4*round+4; i++ {
		if s.DraftTaken&(1<<i) == 0 {
			if last >= 0 {
				return -1
			}
			last = i
		}
	}
	return last
}

// applyDraft takes wonder index i for the player to move, then makes the
// automatic last pick of a round when one is due, reveals round 2 after
// round 1, and lays out Age I after the last pick.
func applyDraft(s *State, i int, ev *[]Event) {
	takeDraftWonder(s, i, s.ToMove, false, ev)
	n := draftPicks(s)
	if n == 3 || n == 7 {
		round := draftRound(n)
		if j := lastOfRound(s, round); j >= 0 {
			takeDraftWonder(s, j, draftPicker(s.FirstSeat, n), true, ev)
		}
		n++
		if n == 4 {
			emit(ev, Event{Kind: EvDraftRound, A: 2})
		}
	}
	if n >= NumDraftWonders {
		startAge(s, 1, ev)
		s.Phase = PhasePlay
		s.Active = s.FirstSeat
		s.ToMove = s.FirstSeat
		return
	}
	s.ToMove = draftPicker(s.FirstSeat, n)
	s.Active = s.ToMove
}
