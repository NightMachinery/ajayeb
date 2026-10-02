package engine

import (
	"errors"
	"fmt"
	"math/bits"
	"strings"

	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
)

// CheckInvariants verifies that s is a state the rules can reach (or a
// scenario built from one), and returns an error listing every violation,
// or nil. Tests and fuzzers call it after every step; the engine itself
// never does. It checks:
//
//   - cards: every registered card is in exactly one place (the structure,
//     a future deal, the removed cards, the box guilds, a city, under a
//     wonder, the discard pile, or the scenario-only OutOfPlay set), and each
//     place holds only cards it may hold;
//   - the structure: empty slots are zero, occupied slots show their card's
//     back, no empty slot is covered by a card, the odd rows are face up, a
//     card turned face up off those rows is accessible, and between turns
//     every accessible card is face up;
//   - coins are never negative;
//   - the pawn is within -9 to 9 (a capital only once the game is over) and
//     the looting tokens agree with it;
//   - wonders: the 12 split into the draft and the box, the draft picks
//     agree with the pick order, WondersBuilt counts the built wonders, at
//     most seven, and exactly one wonder is removed once seven are built;
//   - tokens: the 10 split into the board, the box (an ordered pile with no
//     gaps), the players and the Great Library offer;
//   - the caches of each player equal a recompute from scratch;
//   - phase data: the age and the phase agree, the seats are 0 or 1, only
//     the starter choice has ToMove differ from Active, Destroy and Library
//     are set exactly in their phases and have an option, the result is set
//     exactly when the game is over.
func CheckInvariants(s *State) error {
	c := invChecker{s: s}
	c.cards()
	c.structure()
	c.players()
	c.military()
	c.wonders()
	c.tokens()
	c.phase()
	if len(c.errs) == 0 {
		return nil
	}
	return errors.New("engine: invariants broken: " + strings.Join(c.errs, "; "))
}

type invChecker struct {
	s    *State
	errs []string
}

func (c *invChecker) fail(format string, args ...any) {
	if len(c.errs) < 20 {
		c.errs = append(c.errs, fmt.Sprintf(format, args...))
	}
}

// cardKind is the back group a card belongs to; containers accept kinds.
func cardKind(id catalog.CardID) catalog.Back { return cardBack(id) }

// ageKinds reports whether a card of kind k may be in age a's deck.
func ageKinds(a uint8, k catalog.Back) bool {
	return k == catalog.AgeBack(a) || (a == 3 && k == catalog.GuildBack)
}

func (c *invChecker) cards() {
	s := c.s
	var seen [256]uint8
	var where [256]string
	put := func(id catalog.CardID, loc string) {
		if id == 0 {
			return
		}
		if catalog.Card(id) == nil {
			c.fail("%s holds unregistered card id %d", loc, id)
			return
		}
		seen[id]++
		if seen[id] > 1 {
			c.fail("card %s is in %s and in %s", id, where[id], loc)
		}
		where[id] = loc
	}

	for i, sl := range s.Slots {
		put(sl.Card, "the structure")
		if sl.Card != 0 && (s.Age < 1 || !ageKinds(s.Age, cardKind(sl.Card))) {
			c.fail("slot %d holds %s, which is not a card of age %d", i, sl.Card, s.Age)
		}
	}
	for a := range s.Deals {
		age := uint8(a + 1)
		n := 0
		for _, id := range s.Deals[a] {
			put(id, "a future deal")
			if id == 0 {
				continue
			}
			n++
			if !ageKinds(age, cardKind(id)) {
				c.fail("the age %d deal holds %s", age, id)
			}
		}
		switch {
		case age <= s.Age && n != 0:
			c.fail("the age %d deal still holds %d cards after it was laid out", age, n)
		case age > s.Age && n != SlotsPerAge:
			c.fail("the age %d deal holds %d cards, want %d", age, n, SlotsPerAge)
		}
	}
	for a := range s.Removed {
		for _, id := range s.Removed[a] {
			put(id, "the removed cards")
			if id != 0 && cardKind(id) != catalog.AgeBack(uint8(a+1)) {
				c.fail("the removed cards of age %d hold %s", a+1, id)
			}
		}
	}
	for _, id := range s.BoxGuilds {
		put(id, "the box guilds")
		if id != 0 && cardKind(id) != catalog.GuildBack {
			c.fail("the box guilds hold %s, which is not a guild", id)
		}
	}
	for seat := range s.Players {
		p := &s.Players[seat]
		for id, ok := p.City.Next(0); ok; id, ok = p.City.Next(id + 1) {
			put(id, fmt.Sprintf("seat %d's city", seat))
		}
		for _, ws := range p.Wonders {
			put(ws.Tucked, fmt.Sprintf("seat %d's wonder %s", seat, ws.ID))
		}
	}
	for id, ok := s.Discard.Next(0); ok; id, ok = s.Discard.Next(id + 1) {
		put(id, "the discard pile")
	}
	for id, ok := s.OutOfPlay.Next(0); ok; id, ok = s.OutOfPlay.Next(id + 1) {
		put(id, "out of play")
	}
	for d := range catalog.Cards() {
		if seen[d.ID] == 0 {
			c.fail("card %s is nowhere", d.ID)
		}
	}
}

func (c *invChecker) structure() {
	s := c.s
	if s.Age == 0 {
		for i, sl := range s.Slots {
			if sl != (Slot{}) {
				c.fail("slot %d is set during the wonder draft", i)
			}
		}
		return
	}
	if s.Age > NumAges {
		c.fail("age %d is out of range", s.Age)
		return
	}
	occ := Occupied(s)
	acc := AccessibleMask(s.Age, occ)
	parity := FaceUpMask(s.Age)
	betweenTurns := s.Phase == PhasePlay || s.Phase == PhaseChooseStarter
	for i, sl := range s.Slots {
		bit := uint32(1) << i
		if sl.Card == 0 {
			if sl != (Slot{}) {
				c.fail("empty slot %d is not zero: %+v", i, sl)
			}
			if CoverMask(s.Age, i)&occ != 0 {
				c.fail("slot %d is empty but still covered by a card", i)
			}
			continue
		}
		if sl.Back != cardBack(sl.Card) {
			c.fail("slot %d shows back %s for %s, want %s", i, sl.Back, sl.Card, cardBack(sl.Card))
		}
		if parity&bit != 0 && !sl.FaceUp {
			c.fail("slot %d is on a face-up row but face down", i)
		}
		if parity&bit == 0 && sl.FaceUp && acc&bit == 0 {
			c.fail("slot %d was turned face up but is not accessible", i)
		}
		if betweenTurns && acc&bit != 0 && !sl.FaceUp {
			c.fail("slot %d is accessible but still face down between turns", i)
		}
	}
	if s.Phase == PhasePlay && occ == 0 {
		c.fail("the play phase has an empty structure")
	}
	if s.Phase == PhaseChooseStarter && occ != allSlots {
		c.fail("the starter choice comes with a freshly dealt structure, but %d slots are filled", bits.OnesCount32(occ))
	}
}

func (c *invChecker) players() {
	for seat := range c.s.Players {
		p := &c.s.Players[seat]
		if p.Coins < 0 {
			c.fail("seat %d has %d coins", seat, p.Coins)
		}
		if p.Tokens&^allTokens() != 0 {
			c.fail("seat %d has unregistered tokens %#x", seat, uint16(p.Tokens&^allTokens()))
		}
		fresh := *p
		recomputeCaches(&fresh)
		if fresh != *p {
			c.fail("seat %d's caches are stale:\n  have %s\n  want %s", seat, cacheString(p), cacheString(&fresh))
		}
	}
}

func cacheString(p *Player) string {
	return fmt.Sprintf("chains=%#x prod=%v choice=%v n=%d fix=%#x colors=%v sci=%v",
		p.Chains, p.Prod, p.Choice, p.NChoice, p.TradeFix, p.ColorN, p.Sci)
}

// allTokens returns the set of registered tokens.
func allTokens() catalog.TokenSet {
	var ts catalog.TokenSet
	for d := range catalog.Tokens() {
		ts.Add(d.ID)
	}
	return ts
}

func (c *invChecker) military() {
	s := c.s
	m := s.Military
	if m < -9 || m > 9 {
		c.fail("the pawn is at %d", m)
	}
	if (m == 9 || m == -9) && s.Phase != PhaseGameOver {
		c.fail("the pawn is at a capital (%d) but the game goes on", m)
	}
	if s.Looting&^LootAll != 0 {
		c.fail("unknown looting bits %#x", s.Looting&^LootAll)
	}
	has := func(b uint8) bool { return s.Looting&b != 0 }
	if has(LootPlus3) && m >= 3 {
		c.fail("the +3 looting token is still there with the pawn at %d", m)
	}
	if has(LootPlus6) && m >= 6 {
		c.fail("the +6 looting token is still there with the pawn at %d", m)
	}
	if has(LootMinus3) && m <= -3 {
		c.fail("the -3 looting token is still there with the pawn at %d", m)
	}
	if has(LootMinus6) && m <= -6 {
		c.fail("the -6 looting token is still there with the pawn at %d", m)
	}
	if !has(LootPlus6) && has(LootPlus3) {
		c.fail("the +6 looting token is gone but the +3 one is not")
	}
	if !has(LootMinus6) && has(LootMinus3) {
		c.fail("the -6 looting token is gone but the -3 one is not")
	}
}

func (c *invChecker) wonders() {
	s := c.s
	var seen [256]uint8
	for _, w := range s.DraftOrder {
		seen[w]++
	}
	for _, w := range s.BoxWonders {
		seen[w]++
	}
	if seen[0] != 0 {
		c.fail("the draft order or the box has an empty wonder place")
	}
	for d := range catalog.Wonders() {
		if seen[d.ID] != 1 {
			c.fail("wonder %s appears %d times in the draft order and the box", d.ID, seen[d.ID])
		}
	}

	// Who owns which drafted wonder.
	var owner [NumDraftWonders]int8
	for i := range owner {
		owner[i] = -1
	}
	built, removed := 0, 0
	for seat := range s.Players {
		p := &s.Players[seat]
		ended := false
		for k, ws := range p.Wonders {
			if ws.ID == 0 {
				ended = true
				if ws != (WonderSlot{}) {
					c.fail("seat %d's empty wonder slot %d is not zero", seat, k)
				}
				continue
			}
			if ended {
				c.fail("seat %d's wonder slots have a gap before slot %d", seat, k)
			}
			i := -1
			for j, w := range s.DraftOrder {
				if w == ws.ID {
					i = j
				}
			}
			switch {
			case i < 0:
				c.fail("seat %d owns %s, which was not drafted", seat, ws.ID)
			case s.DraftTaken&(1<<i) == 0:
				c.fail("seat %d owns %s, which is still on offer", seat, ws.ID)
			case owner[i] >= 0:
				c.fail("both seats own %s", ws.ID)
			default:
				owner[i] = int8(seat)
			}
			if ws.Built {
				built++
				if ws.Tucked == 0 {
					c.fail("seat %d's %s is built with no card tucked under it", seat, ws.ID)
				}
			} else if ws.Tucked != 0 {
				c.fail("seat %d's unbuilt %s has a tucked card", seat, ws.ID)
			}
			if ws.Removed {
				removed++
				if ws.Built {
					c.fail("seat %d's %s is both built and removed", seat, ws.ID)
				}
			}
		}
	}
	for i := range owner {
		if s.DraftTaken&(1<<i) != 0 && owner[i] < 0 {
			c.fail("drafted wonder %s has no owner", s.DraftOrder[i])
		}
	}

	// The picks agree with the pick order, round by round.
	n := draftPicks(s)
	var want, have [2][2]int // [round][seat]
	for k := range n {
		want[draftRound(k)][draftPicker(s.FirstSeat, k)]++
	}
	for i := range owner {
		if owner[i] >= 0 {
			have[i/4][owner[i]]++
		}
	}
	if want != have {
		c.fail("after %d draft picks the seats hold %v wonders per round, want %v", n, have, want)
	}
	if s.Phase == PhaseWonderDraft {
		if n >= NumDraftWonders || n == 3 || n == 7 {
			c.fail("the draft is waiting after %d picks", n)
		} else if s.ToMove != draftPicker(s.FirstSeat, n) {
			c.fail("draft pick %d is seat %d's, but seat %d is to move", n, draftPicker(s.FirstSeat, n), s.ToMove)
		}
	} else if n != NumDraftWonders {
		c.fail("phase %s after only %d draft picks", s.Phase, n)
	}

	if int(s.WondersBuilt) != built {
		c.fail("WondersBuilt is %d but %d wonders are built", s.WondersBuilt, built)
	}
	if built > MaxWondersBuilt {
		c.fail("%d wonders are built", built)
	}
	switch {
	case built == MaxWondersBuilt && removed != 1:
		c.fail("seven wonders are built but %d are removed", removed)
	case built < MaxWondersBuilt && removed != 0:
		c.fail("%d wonders are removed with only %d built", removed, built)
	}
}

func (c *invChecker) tokens() {
	s := c.s
	var seen [256]uint8
	for t, ok := s.BoardTokens.Next(0); ok; t, ok = s.BoardTokens.Next(t + 1) {
		seen[t]++
	}
	gap := false
	for i, t := range s.BoxTokens {
		if t == 0 {
			gap = true
			continue
		}
		if gap {
			c.fail("the box has a gap before place %d", i)
		}
		seen[t]++
	}
	for seat := range s.Players {
		ts := s.Players[seat].Tokens
		for t, ok := ts.Next(0); ok; t, ok = ts.Next(t + 1) {
			seen[t]++
		}
	}
	for _, t := range s.Library {
		if t != 0 {
			seen[t]++
		}
	}
	if seen[0] != 0 {
		c.fail("token id 0 is in a token set")
	}
	for d := range catalog.Tokens() {
		if seen[d.ID] != 1 {
			c.fail("token %s appears %d times among the board, the box, the players and the offer", d.ID, seen[d.ID])
		}
	}
	for t := 1; t < len(seen); t++ {
		if seen[t] != 0 && catalog.Token(catalog.TokenID(t)) == nil {
			c.fail("unregistered token id %d", t)
		}
	}
}

func (c *invChecker) phase() {
	s := c.s
	if s.Phase >= numPhases {
		c.fail("phase %d is out of range", s.Phase)
		return
	}
	if s.ToMove > 1 || s.Active > 1 || s.FirstSeat > 1 {
		c.fail("a seat is out of range: toMove %d, active %d, first %d", s.ToMove, s.Active, s.FirstSeat)
	}
	if (s.Phase == PhaseWonderDraft) != (s.Age == 0) {
		c.fail("phase %s in age %d", s.Phase, s.Age)
	}
	if s.Phase != PhaseChooseStarter && s.Phase != PhaseGameOver && s.ToMove != s.Active {
		c.fail("seat %d is to move in phase %s, but seat %d is active", s.ToMove, s.Phase, s.Active)
	}
	if s.ExtraTurn && (s.Phase == PhaseWonderDraft || s.Phase == PhaseChooseStarter) {
		c.fail("an extra turn is pending in phase %s", s.Phase)
	}

	if (s.Destroy != catalog.NoColor) != (s.Phase == PhaseDestroyCard) {
		c.fail("Destroy is %s in phase %s", s.Destroy, s.Phase)
	}
	if s.Phase == PhaseDestroyCard {
		opp := &s.Players[s.ToMove.Opp()]
		if s.Destroy >= catalog.NumColors || opp.ColorN[s.Destroy] == 0 {
			c.fail("the destroy choice of a %s card has no target", s.Destroy)
		}
	}
	nLib := 0
	for _, t := range s.Library {
		if t != 0 {
			nLib++
		}
	}
	if (nLib != 0) != (s.Phase == PhasePickLibraryToken) {
		c.fail("the Great Library offer holds %d tokens in phase %s", nLib, s.Phase)
	}
	if s.Phase == PhaseBuildFromDiscard && s.Discard.Empty() {
		c.fail("the discard choice has an empty discard pile")
	}
	if s.Phase == PhasePickBoardToken && s.BoardTokens.Empty() {
		c.fail("the board token choice has an empty board")
	}

	over := s.Phase == PhaseGameOver
	if over != (s.Result.Kind != VictoryNone) {
		c.fail("result kind %s in phase %s", s.Result.Kind, s.Phase)
	}
	if !over && s.Result != (Result{}) {
		c.fail("a result is set while the game goes on")
	}
	if over && s.Result.Kind >= numVictoryKinds {
		c.fail("result kind %d is out of range", s.Result.Kind)
	}
	if over && s.Result.Winner != 0 && s.Result.Winner != 1 && s.Result.Winner != SharedVictory {
		c.fail("result winner %d", s.Result.Winner)
	}
}
