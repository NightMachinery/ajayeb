package engine

import "github.com/NightMachinery/ajayeb/server/internal/engine/catalog"

// This file gives the bots their picture of a game. A bot never receives the
// real State. It receives an Observation, a copy of the state in which every
// part hidden from its seat is replaced by catalog.Unknown, and it samples
// the hidden parts itself with Determinize.
//
// # What a seat cannot see
//
// Observe hides the same things from both seats, in every phase (game over
// included, since a bot never acts then):
//
//   - the seed, which becomes 0;
//   - the card of every face-down slot (the slot keeps its back, so in Age
//     III the observer still sees which face-down cards are guilds);
//   - the deals of the ages not laid out yet;
//   - the removed cards of every age;
//   - the guilds left in the box;
//   - the four wonders of draft round 2 until round 1 is over, and the four
//     wonders left in the box;
//   - the order of the token box: which tokens it holds is known by
//     elimination (they are the tokens in no other place), but not in which
//     order the Great Library would draw them;
//   - the Great Library offer, except from the player choosing from it.
//
// A hidden place keeps the difference between empty (0) and occupied
// (Unknown), so the observation still counts the cards and tokens in each
// place. Everything else is public: the face-up cards and every slot's
// back, both cities and the cards tucked under wonders, coins, tokens,
// wonders, the discard pile, the military track, the phase and its data,
// and the scenario-only OutOfPlay set (cards a test scenario took out of
// play; real games never use it).
//
// # Determinization
//
// Every hidden part of a game was fixed by the setup shuffle, and neither
// player knows more about it than the other (the Great Library offer
// aside), so no move gives anything away. Given an observation, every
// complete state that agrees with it is therefore equally likely. The one
// exception, which the bots accept, is an opponent's Great Library pick: the
// token it kept says a little about the two it put back. Determinize draws
// one of the agreeing states uniformly at random, in this order:
//
//  1. The wonders no one can see (draft round 2 during round 1, and the box)
//     are shuffled into the hidden wonder places, the draft order first.
//  2. The tokens no one can see are shuffled into a hidden Great Library
//     offer first and then into the box, in pile order.
//  3. For each age, the unseen cards are its cards in no visible place (a
//     face-up slot, a city, under a wonder, the discard pile, OutOfPlay):
//     - an age already over keeps its unseen cards among its removed cards;
//     - the current age's unseen cards are shuffled into its face-down slots
//       of that age's back and its removed cards; in Age III the unseen
//       guilds are shuffled into the guild-backed face-down slots and the
//       box;
//     - an age not laid out yet is dealt as NewGame deals it (setup.go): for
//       Ages I and II, shuffle, deal and keep the rest as removed cards; for
//       Age III, set its removed cards aside, split the unseen guilds
//       between the deal and the box, and shuffle the deal.
//
// Each hidden place is filled only from the cards (or wonders, or tokens)
// that may be there, so the sample passes CheckInvariants, agrees with
// every public field, and gives back the same observation:
// Observe(Determinize(o), o.Seat()) == o.
//
// Determinize allocates nothing; the bots call it once per search
// iteration.

// Observation is what one seat may know about a game: the state with every
// hidden part replaced by catalog.Unknown. Its fields are unexported, so code
// outside the engine (the bots) cannot reach the real state through it.
// Observations are small comparable values: two observations are equal when
// they hold the same public information for the same seat.
type Observation struct {
	seat Seat
	st   State
}

// obsInconsistent is the panic message of Determinize for an observation
// whose counts do not add up.
const obsInconsistent = "engine: Determinize: the observation does not describe a reachable state (it was not made by Observe from a valid state)"

// Observe returns what seat may know about s. It panics if seat is not 0 or
// 1. s is not changed.
func Observe(s *State, seat Seat) Observation {
	if seat > 1 {
		panic("engine: Observe needs seat 0 or 1")
	}
	o := Observation{seat: seat, st: *s}
	st := &o.st
	st.Seed = 0
	for i := range st.Slots {
		if sl := &st.Slots[i]; sl.Card != 0 && !sl.FaceUp {
			sl.Card = catalog.Unknown
		}
	}
	for a := range st.Deals {
		obsHide(st.Deals[a][:])
	}
	for a := range st.Removed {
		obsHide(st.Removed[a][:])
	}
	obsHide(st.BoxGuilds[:])
	if draftPicks(st) < 4 {
		obsHide(st.DraftOrder[4:])
	}
	obsHide(st.BoxWonders[:])
	obsHide(st.BoxTokens[:])
	if st.Phase != PhasePickLibraryToken || st.ToMove != seat {
		obsHide(st.Library[:])
	}
	return o
}

// obsHide replaces every id in ids that is not 0 with catalog.Unknown.
func obsHide[T ~uint8](ids []T) {
	for i := range ids {
		if ids[i] != 0 {
			ids[i] = catalog.Unknown
		}
	}
}

// Seat returns the observer.
func (o *Observation) Seat() Seat { return o.seat }

// Public returns the observation's state, in which every hidden id is
// catalog.Unknown (and the seed is 0). It points into the observation and is
// read-only by convention. The engine functions that read only public
// information give the real answers on it: LegalActions lists exactly the
// real legal actions when the observer is to move, and the quotes of
// face-up cards and wonders, DiscardValue and Scores are the real ones.
// Never Apply an action to it; Determinize first.
func (o *Observation) Public() *State { return &o.st }

// Determinize returns a complete state drawn uniformly from those that agree
// with the observation (see the top of this file), using r. The observation
// is not changed, and the same observation and generator state always give
// the same sample. It panics if the observation was not made by Observe from
// a valid state.
func (o *Observation) Determinize(r *RNG) State {
	d := o.st
	obsDealWonders(&d, r)
	obsDealTokens(&d, r)

	unseen := catalog.T.All.AndNot(obsVisibleCards(&d))
	var pool, guilds obsPool
	for age := uint8(1); age <= NumAges; age++ {
		pool.load(unseen.And(catalog.T.AgeCards[age]), r)
		switch {
		case age < d.Age:
			pool.fillCards(d.Removed[age-1][:])
		case age == d.Age:
			pool.fillSlots(&d, catalog.AgeBack(age))
			pool.fillCards(d.Removed[age-1][:])
			if age == NumAges {
				guilds.load(unseen.And(catalog.T.Guilds), r)
				guilds.fillSlots(&d, catalog.GuildBack)
				guilds.fillCards(d.BoxGuilds[:])
				guilds.done()
			}
		case age < NumAges:
			pool.fillCards(d.Deals[age-1][:])
			pool.fillCards(d.Removed[age-1][:])
		default:
			obsDealLastAge(&d, &pool, &guilds, unseen, r)
		}
		pool.done()
	}
	return d
}

// obsDealLastAge deals the Age III cards and the guilds while Age III is
// still to come, as NewGame does: the last of the shuffled Age III cards are
// the removed ones, the last of the shuffled guilds stay in the box, and the
// rest of both are shuffled together into the deal. pool holds the shuffled
// unseen Age III cards on entry; afterwards it holds the shuffled deal.
func obsDealLastAge(d *State, pool, guilds *obsPool, unseen catalog.CardSet, r *RNG) {
	keep := pool.n - obsCountHidden(d.Removed[NumAges-1][:])
	guilds.load(unseen.And(catalog.T.Guilds), r)
	inPlay := guilds.n - obsCountHidden(d.BoxGuilds[:])
	if keep < 0 || inPlay < 0 {
		panic(obsInconsistent)
	}
	pool.k = keep
	pool.fillCards(d.Removed[NumAges-1][:])
	pool.done()
	guilds.k = inPlay
	guilds.fillCards(d.BoxGuilds[:])
	guilds.done()

	// The deal: the kept Age III cards, then the guilds in play, shuffled.
	copy(pool.ids[keep:], guilds.ids[:inPlay])
	pool.n, pool.k = keep+inPlay, 0
	ShuffleSlice(r, pool.ids[:pool.n])
	pool.fillCards(d.Deals[NumAges-1][:])
}

// obsVisibleCards returns the cards an observation shows: those in face-up
// slots, in the cities, under wonders, in the discard pile and out of play.
func obsVisibleCards(d *State) catalog.CardSet {
	seen := d.Discard.Or(d.OutOfPlay)
	for i := range d.Slots {
		if c := d.Slots[i].Card; c != 0 && c != catalog.Unknown {
			seen.Add(c)
		}
	}
	for seat := range d.Players {
		p := &d.Players[seat]
		seen = seen.Or(p.City)
		for _, w := range p.Wonders {
			if w.Tucked != 0 {
				seen.Add(w.Tucked)
			}
		}
	}
	return seen
}

// obsDealWonders shuffles the wonders that appear in neither the draft
// order nor the box into their hidden places: draft round 2 (while it is
// hidden), then the box.
func obsDealWonders(d *State, r *RNG) {
	var seen [catalog.NumWonders + 1]bool
	for _, w := range d.DraftOrder {
		if w != catalog.Unknown && int(w) < len(seen) {
			seen[w] = true
		}
	}
	for _, w := range d.BoxWonders {
		if w != catalog.Unknown && int(w) < len(seen) {
			seen[w] = true
		}
	}
	var pool [catalog.NumWonders]catalog.WonderID
	n := 0
	for w := catalog.WonderID(1); int(w) <= catalog.NumWonders; w++ {
		if !seen[w] && catalog.Wonder(w) != nil {
			pool[n] = w
			n++
		}
	}
	ShuffleSlice(r, pool[:n])
	k := obsFill(d.DraftOrder[:], pool[:n], 0)
	k = obsFill(d.BoxWonders[:], pool[:n], k)
	if k != n {
		panic(obsInconsistent)
	}
}

// obsDealTokens shuffles the tokens that are in no visible place (the
// board, the players, a visible Great Library offer) into the hidden places:
// a hidden offer first, then the box in pile order.
func obsDealTokens(d *State, r *RNG) {
	seen := d.BoardTokens | d.Players[0].Tokens | d.Players[1].Tokens
	for _, t := range d.Library {
		if t != 0 && t != catalog.Unknown {
			seen.Add(t)
		}
	}
	var pool [catalog.NumTokens]catalog.TokenID
	n := 0
	for t := catalog.TokenID(1); int(t) <= catalog.NumTokens; t++ {
		if !seen.Has(t) && catalog.Token(t) != nil {
			pool[n] = t
			n++
		}
	}
	ShuffleSlice(r, pool[:n])
	k := obsFill(d.Library[:], pool[:n], 0)
	k = obsFill(d.BoxTokens[:], pool[:n], k)
	if k != n {
		panic(obsInconsistent)
	}
}

// obsFill replaces, in order, each Unknown entry of places with pool[k],
// pool[k+1] and so on, and returns the next unused index of pool. It panics
// if pool runs out.
func obsFill[T ~uint8](places, pool []T, k int) int {
	for i := range places {
		if places[i] == catalog.Unknown {
			if k >= len(pool) {
				panic(obsInconsistent)
			}
			places[i] = pool[k]
			k++
		}
	}
	return k
}

// obsCountHidden returns the number of Unknown entries of places.
func obsCountHidden(places []catalog.CardID) int {
	n := 0
	for _, c := range places {
		if c == catalog.Unknown {
			n++
		}
	}
	return n
}

// obsPool is a shuffled pool of unseen cards, dealt out in order: ids[:n]
// is the pool, ids[:k] has been dealt.
type obsPool struct {
	ids  [catalog.NumCards]catalog.CardID
	n, k int
}

// load fills the pool with the cards of set, in id order, and shuffles it.
func (p *obsPool) load(set catalog.CardSet, r *RNG) {
	p.n, p.k = 0, 0
	for c, ok := set.Next(1); ok; c, ok = set.Next(c + 1) {
		p.ids[p.n] = c
		p.n++
	}
	ShuffleSlice(r, p.ids[:p.n])
}

// fillCards deals the next cards of the pool into the Unknown entries of
// places.
func (p *obsPool) fillCards(places []catalog.CardID) {
	p.k = obsFill(places, p.ids[:p.n], p.k)
}

// fillSlots deals the next cards of the pool into the hidden face-down
// slots of d that show back b, in slot order.
func (p *obsPool) fillSlots(d *State, b catalog.Back) {
	for i := range d.Slots {
		if sl := &d.Slots[i]; sl.Card == catalog.Unknown && sl.Back == b {
			if p.k >= p.n {
				panic(obsInconsistent)
			}
			sl.Card = p.ids[p.k]
			p.k++
		}
	}
}

// done panics unless every card of the pool has been dealt.
func (p *obsPool) done() {
	if p.k != p.n {
		panic(obsInconsistent)
	}
}
