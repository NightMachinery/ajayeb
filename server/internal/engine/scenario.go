package engine

import (
	"fmt"
	"strings"

	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
)

// This file builds test and end-to-end positions: a Scenario describes the
// parts of a game that matter to a test (the wonders, the age, the cities,
// coins, tokens, built wonders, the discard pile, which cards are
// accessible, the pawn), and NewGameFromScenario turns it into a state that
// passes CheckInvariants, starting from an ordinary NewGame deal so that
// everything the scenario leaves out stays random but reproducible.
//
// Cards are only ever moved, never created or dropped, so every card stays
// in exactly one place. A card the scenario names leaves its place in one of
// these ways:
//
//   - from a past age's deal or OutOfPlay: it is simply taken;
//   - from the removed cards or the box guilds: its place becomes a hole
//     (a zero entry, which only scenario games have);
//   - from the structure or a future deal: a spare card of the same back
//     (an unnamed removed card of that age, or an unnamed box guild) takes
//     its place. In the structure, when no spare is left, the card is
//     instead taken from an accessible slot, as if it had been played
//     earlier in the age (the card first swaps places with an unnamed card
//     in an accessible slot).
//
// The past ages' cards that the scenario does not place, and the cards that
// OnlyLeft clears from the structure, go to OutOfPlay, which real games
// never use. A scenario game is meant for tests and dev rooms only; a log
// records its scenario so that it replays.

// Scenario describes a starting position. Every field is optional; the zero
// Scenario is the start of Age I after a draft in which every player took
// the wonders in offer order. Names are slugs.
type Scenario struct {
	Name string `json:"name,omitempty"`

	// FirstPlayer overrides the first player (who started the draft and
	// Age I, and who is to move unless ToMove says otherwise).
	FirstPlayer *Seat `json:"firstPlayer,omitempty"`

	// Wonders gives each seat its four wonders, in slot order, and skips
	// the draft. Without it the draft is completed automatically: each
	// pick takes the first wonder still on offer.
	Wonders *[2][WondersPerPlayer]string `json:"wonders,omitempty"`

	// Age is the age (1 to 3) whose structure is laid out; 0 means 1. The
	// earlier ages' cards are out of play unless placed below.
	Age int `json:"age,omitempty"`

	Players [2]ScenarioPlayer `json:"players"`

	// BoardTokens, when not empty, is exactly the set of tokens on the
	// board. Otherwise the board keeps its dealt tokens, minus those given
	// to players or put in BoxOrder.
	BoardTokens []string `json:"boardTokens,omitempty"`

	// BoxOrder puts these tokens at the front of the box, in this order (the
	// Great Library draws the first three). The rest of the box follows in
	// its dealt order.
	BoxOrder []string `json:"boxTokenOrder,omitempty"`

	Discard []string `json:"discard,omitempty"`

	// Accessible moves these cards of the current age into accessible,
	// face-up slots of the structure.
	Accessible []string `json:"accessible,omitempty"`

	// OnlyLeft empties the structure except for these cards of the current
	// age, which take the first slots in layout order (the top rows); use it
	// to set up the end of an age.
	OnlyLeft []string `json:"onlyLeft,omitempty"`

	// Pawn places the conflict pawn (-8 to 8; positive is toward seat 1's
	// capital). The looting tokens the pawn has reached are gone.
	Pawn *int `json:"pawn,omitempty"`

	// ToMove sets the active seat, which is to move.
	ToMove *Seat `json:"toMove,omitempty"`

	// Trigger is a script of action keys that sets the scenario's flow
	// going (see RunTrigger), and Expect names what the trigger should lead
	// to. Both are for named scenarios and their tests; NewGameFromScenario
	// ignores them.
	Trigger []string `json:"trigger,omitempty"`
	Expect  string   `json:"expect,omitempty"`
}

// ScenarioPlayer is one seat's part of a scenario.
type ScenarioPlayer struct {
	Coins  *int             `json:"coins,omitempty"` // default StartCoins
	City   []string         `json:"city,omitempty"`
	Tokens []string         `json:"tokens,omitempty"`
	Built  []ScenarioWonder `json:"built,omitempty"`
}

// ScenarioWonder is a built wonder. Tucked names the card under it; when it
// is empty, a card out of play (or else a removed card) is used.
type ScenarioWonder struct {
	Wonder string `json:"wonder"`
	Tucked string `json:"tucked,omitempty"`
}

// NewGameFromScenario creates the game NewGame(seed, opts) would, then
// rearranges it as sc describes. The result passes CheckInvariants. A nil
// scenario gives the plain NewGame state.
func NewGameFromScenario(seed uint64, opts Options, sc *Scenario) (State, error) {
	s := NewGame(seed, opts)
	if sc == nil {
		return s, nil
	}
	b := scenarioBuilder{s: &s, sc: sc}
	if err := b.build(); err != nil {
		return State{}, fmt.Errorf("engine: scenario %s: %w", truncateString(sc.Name), err)
	}
	if err := CheckInvariants(&s); err != nil {
		return State{}, fmt.Errorf("engine: scenario %s: %w", truncateString(sc.Name), err)
	}
	return s, nil
}

// RunTrigger applies the scenario's trigger keys to s in order, appending
// their events to *ev unless ev is nil. It stops at the first key that does
// not parse or is not legal.
func RunTrigger(s *State, sc *Scenario, ev *[]Event) error {
	for i, k := range sc.Trigger {
		a, err := ParseKey(k)
		if err != nil {
			return fmt.Errorf("engine: trigger step %d: %w", i, err)
		}
		if err := Apply(s, a, ev); err != nil {
			return fmt.Errorf("engine: trigger step %d: %w", i, err)
		}
	}
	return nil
}

// scenarioBuilder carries the resolved scenario while it is applied.
type scenarioBuilder struct {
	s  *State
	sc *Scenario

	city       [2][]catalog.CardID
	discard    []catalog.CardID
	accessible []catalog.CardID
	onlyLeft   []catalog.CardID
	built      [2][]builtWonder
	tokens     [2][]catalog.TokenID
	board      []catalog.TokenID
	boxOrder   []catalog.TokenID

	pinned catalog.CardSet // every card the scenario names
	named  [256]string     // where each named card was named, for error messages
}

type builtWonder struct {
	wonder catalog.WonderID
	tucked catalog.CardID // 0: pick one
}

func (b *scenarioBuilder) build() error {
	if err := b.resolve(); err != nil {
		return err
	}
	s, sc := b.s, b.sc
	if sc.FirstPlayer != nil {
		if *sc.FirstPlayer > 1 {
			return fmt.Errorf("firstPlayer %d is not 0 or 1", *sc.FirstPlayer)
		}
		s.FirstSeat = *sc.FirstPlayer
	}
	if err := b.draft(); err != nil {
		return err
	}
	age := sc.Age
	if age == 0 {
		age = 1
	}
	if age < 1 || age > NumAges {
		return fmt.Errorf("age %d is not 1 to 3", sc.Age)
	}
	for a := 1; a < age; a++ {
		for i, c := range s.Deals[a-1] {
			if c != 0 {
				s.OutOfPlay.Add(c)
				s.Deals[a-1][i] = 0
			}
		}
	}
	startAge(s, uint8(age), nil)
	s.Phase = PhasePlay
	s.Active, s.ToMove = s.FirstSeat, s.FirstSeat

	if err := b.placeOnlyLeft(); err != nil {
		return err
	}
	for seat := range b.city {
		for _, c := range b.city[seat] {
			if err := b.takeOut(c); err != nil {
				return err
			}
		}
	}
	for _, c := range b.discard {
		if err := b.takeOut(c); err != nil {
			return err
		}
	}
	for seat := range b.built {
		for _, bw := range b.built[seat] {
			if bw.tucked != 0 {
				if err := b.takeOut(bw.tucked); err != nil {
					return err
				}
			}
		}
	}
	if err := b.placeAccessible(); err != nil {
		return err
	}
	b.refreshFaces()

	for seat := range sc.Players {
		if p := sc.Players[seat].Coins; p != nil {
			if *p < 0 || *p > maxInt16 {
				return fmt.Errorf("seat %d: coins %d out of range", seat, *p)
			}
			s.Players[seat].Coins = int16(*p)
		}
		for _, c := range b.city[seat] {
			addCard(s, Seat(seat), c)
		}
	}
	for _, c := range b.discard {
		s.Discard.Add(c)
	}
	if err := b.buildWonders(); err != nil {
		return err
	}
	if err := b.placeTokens(); err != nil {
		return err
	}
	if sc.Pawn != nil {
		p := *sc.Pawn
		if p <= -9 || p >= 9 {
			return fmt.Errorf("pawn %d is not -8 to 8", p)
		}
		s.Military = int8(p)
		s.Looting = LootAll
		if p >= 3 {
			s.Looting &^= LootPlus3
		}
		if p >= 6 {
			s.Looting &^= LootPlus6
		}
		if p <= -3 {
			s.Looting &^= LootMinus3
		}
		if p <= -6 {
			s.Looting &^= LootMinus6
		}
	}
	if sc.ToMove != nil {
		if *sc.ToMove > 1 {
			return fmt.Errorf("toMove %d is not 0 or 1", *sc.ToMove)
		}
		s.Active, s.ToMove = *sc.ToMove, *sc.ToMove
	}
	return nil
}

// resolve turns every slug into an id and pins the named cards.
func (b *scenarioBuilder) resolve() error {
	sc := b.sc
	var errs []string
	card := func(slug, where string) catalog.CardID {
		c, ok := catalog.CardBySlug(slug)
		if !ok {
			errs = append(errs, fmt.Sprintf("%s: unknown card %s", where, truncateString(slug)))
			return 0
		}
		return c
	}
	pin := func(c catalog.CardID, where string) {
		if c == 0 {
			return
		}
		if b.pinned.Has(c) {
			errs = append(errs, fmt.Sprintf("card %s is named in %s and in %s", c, b.named[c], where))
			return
		}
		b.pinned.Add(c)
		b.named[c] = where
	}
	token := func(slug, where string) catalog.TokenID {
		t, ok := catalog.TokenBySlug(slug)
		if !ok {
			errs = append(errs, fmt.Sprintf("%s: unknown token %s", where, truncateString(slug)))
			return 0
		}
		return t
	}
	for seat := range sc.Players {
		p := &sc.Players[seat]
		where := fmt.Sprintf("seat %d's city", seat)
		for _, slug := range p.City {
			if c := card(slug, where); c != 0 {
				pin(c, where)
				b.city[seat] = append(b.city[seat], c)
			}
		}
		for _, slug := range p.Tokens {
			if t := token(slug, fmt.Sprintf("seat %d's tokens", seat)); t != 0 {
				b.tokens[seat] = append(b.tokens[seat], t)
			}
		}
		for _, bw := range p.Built {
			w, ok := catalog.WonderBySlug(bw.Wonder)
			if !ok {
				errs = append(errs, fmt.Sprintf("seat %d's built wonders: unknown wonder %s", seat, truncateString(bw.Wonder)))
				continue
			}
			x := builtWonder{wonder: w}
			if bw.Tucked != "" {
				where := fmt.Sprintf("seat %d's %s", seat, w)
				if c := card(bw.Tucked, where); c != 0 {
					pin(c, where)
					x.tucked = c
				}
			}
			b.built[seat] = append(b.built[seat], x)
		}
	}
	for _, slug := range sc.Discard {
		if c := card(slug, "the discard pile"); c != 0 {
			pin(c, "the discard pile")
			b.discard = append(b.discard, c)
		}
	}
	var left catalog.CardSet
	for _, slug := range sc.OnlyLeft {
		if c := card(slug, "onlyLeft"); c != 0 {
			pin(c, "onlyLeft")
			left.Add(c)
			b.onlyLeft = append(b.onlyLeft, c)
		}
	}
	for _, slug := range sc.Accessible {
		c := card(slug, "accessible")
		if c == 0 {
			continue
		}
		if !left.Has(c) { // a card may be both left and accessible
			pin(c, "accessible")
		}
		for _, x := range b.accessible {
			if x == c {
				errs = append(errs, fmt.Sprintf("card %s is named twice in accessible", c))
			}
		}
		b.accessible = append(b.accessible, c)
	}
	for _, slug := range sc.BoardTokens {
		if t := token(slug, "boardTokens"); t != 0 {
			b.board = append(b.board, t)
		}
	}
	for _, slug := range sc.BoxOrder {
		if t := token(slug, "boxTokenOrder"); t != 0 {
			b.boxOrder = append(b.boxOrder, t)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}

// draft gives the players their wonders and marks the draft complete.
func (b *scenarioBuilder) draft() error {
	s := b.s
	for i := range s.Players {
		s.Players[i].Wonders = [WondersPerPlayer]WonderSlot{}
	}
	if b.sc.Wonders == nil {
		var n [2]int
		for k := range NumDraftWonders {
			seat := draftPicker(s.FirstSeat, k)
			s.Players[seat].Wonders[n[seat]] = WonderSlot{ID: s.DraftOrder[k]}
			n[seat]++
		}
		s.DraftTaken = 1<<NumDraftWonders - 1
		return nil
	}
	var ids [2][WondersPerPlayer]catalog.WonderID
	var used [catalog.NumWonders + 1]bool
	for seat := range ids {
		for k, slug := range b.sc.Wonders[seat] {
			w, ok := catalog.WonderBySlug(slug)
			if !ok {
				return fmt.Errorf("seat %d: unknown wonder %s", seat, truncateString(slug))
			}
			if used[w] {
				return fmt.Errorf("wonder %s is given twice", w)
			}
			used[w] = true
			ids[seat][k] = w
		}
	}
	// The other four wonders go to the box in their dealt order.
	var dealt [catalog.NumWonders]catalog.WonderID
	copy(dealt[:], s.DraftOrder[:])
	copy(dealt[NumDraftWonders:], s.BoxWonders[:])
	box := 0
	for _, w := range dealt {
		if !used[w] {
			s.BoxWonders[box] = w
			box++
		}
	}
	// The draft order is laid out so that the pick order gives each seat
	// its wonders in slot order.
	var n [2]int
	for k := range NumDraftWonders {
		seat := draftPicker(s.FirstSeat, k)
		w := ids[seat][n[seat]]
		n[seat]++
		s.DraftOrder[k] = w
		s.Players[seat].Wonders[n[seat]-1] = WonderSlot{ID: w}
	}
	s.DraftTaken = 1<<NumDraftWonders - 1
	return nil
}

// Card places, for moving named cards.
type locKind uint8

const (
	locNone locKind = iota
	locSlot
	locDeal
	locRemoved
	locBoxGuild
	locOutOfPlay
	locPlaced // a city, the discard pile or under a wonder
)

type cardLoc struct {
	kind locKind
	a, i int // the age index (deals, removed cards) and the place within
}

func (b *scenarioBuilder) locate(c catalog.CardID) cardLoc {
	s := b.s
	for i := range s.Slots {
		if s.Slots[i].Card == c {
			return cardLoc{kind: locSlot, i: i}
		}
	}
	for a := range s.Deals {
		for i, x := range s.Deals[a] {
			if x == c {
				return cardLoc{kind: locDeal, a: a, i: i}
			}
		}
	}
	for a := range s.Removed {
		for i, x := range s.Removed[a] {
			if x == c {
				return cardLoc{kind: locRemoved, a: a, i: i}
			}
		}
	}
	for i, x := range s.BoxGuilds {
		if x == c {
			return cardLoc{kind: locBoxGuild, i: i}
		}
	}
	if s.OutOfPlay.Has(c) {
		return cardLoc{kind: locOutOfPlay}
	}
	return cardLoc{kind: locPlaced}
}

func (b *scenarioBuilder) get(l cardLoc) catalog.CardID {
	s := b.s
	switch l.kind {
	case locSlot:
		return s.Slots[l.i].Card
	case locDeal:
		return s.Deals[l.a][l.i]
	case locRemoved:
		return s.Removed[l.a][l.i]
	case locBoxGuild:
		return s.BoxGuilds[l.i]
	}
	return 0
}

// set puts card c (0 to empty it) at place l. Slot faces are refreshed at
// the end of the build (refreshFaces).
func (b *scenarioBuilder) set(l cardLoc, c catalog.CardID) {
	s := b.s
	switch l.kind {
	case locSlot:
		if c == 0 {
			s.Slots[l.i] = Slot{}
		} else {
			s.Slots[l.i] = Slot{Card: c, FaceUp: s.Slots[l.i].FaceUp, Back: cardBack(c)}
		}
	case locDeal:
		s.Deals[l.a][l.i] = c
	case locRemoved:
		s.Removed[l.a][l.i] = c
	case locBoxGuild:
		s.BoxGuilds[l.i] = c
	case locOutOfPlay:
		if c != 0 {
			s.OutOfPlay.Add(c)
		}
	}
}

// swap exchanges the cards at two places (neither OutOfPlay nor placed).
func (b *scenarioBuilder) swap(x, y cardLoc) {
	cx, cy := b.get(x), b.get(y)
	b.set(x, cy)
	b.set(y, cx)
}

// spare finds an unnamed card with the same back as c among the removed
// cards of its age or the box guilds.
func (b *scenarioBuilder) spare(c catalog.CardID) (cardLoc, bool) {
	s := b.s
	k := cardBack(c)
	if k == catalog.GuildBack {
		for i, x := range s.BoxGuilds {
			if x != 0 && !b.pinned.Has(x) {
				return cardLoc{kind: locBoxGuild, i: i}, true
			}
		}
		return cardLoc{}, false
	}
	for a := range s.Removed {
		if catalog.AgeBack(uint8(a+1)) != k {
			continue
		}
		for i, x := range s.Removed[a] {
			if x != 0 && !b.pinned.Has(x) {
				return cardLoc{kind: locRemoved, a: a, i: i}, true
			}
		}
	}
	return cardLoc{}, false
}

// takeOut removes named card c from wherever it is, so that it can be put in
// a city, the discard pile or under a wonder.
func (b *scenarioBuilder) takeOut(c catalog.CardID) error {
	s := b.s
	l := b.locate(c)
	switch l.kind {
	case locOutOfPlay:
		s.OutOfPlay.Remove(c)
		return nil
	case locRemoved, locBoxGuild:
		b.set(l, 0)
		return nil
	case locSlot, locDeal:
		if sp, ok := b.spare(c); ok {
			b.swap(l, sp)
			b.set(sp, 0)
			return nil
		}
		if l.kind == locDeal {
			return fmt.Errorf("card %s (%s) is in the age %d deal and no spare card of its back is left to replace it", c, b.named[c], l.a+1)
		}
		return b.vacate(c, l.i)
	}
	return fmt.Errorf("card %s cannot be placed twice", c)
}

// vacate empties the slot of card c, first moving c to an accessible slot so
// the structure stays one that play can reach.
func (b *scenarioBuilder) vacate(c catalog.CardID, i int) error {
	s := b.s
	acc := Accessible(s)
	if acc&(1<<i) == 0 {
		j := -1
		for k := range s.Slots {
			if acc&(1<<k) != 0 && !b.pinned.Has(s.Slots[k].Card) {
				j = k
				break
			}
		}
		if j < 0 {
			return fmt.Errorf("card %s (%s) is in the structure and no accessible slot is free to take it from", c, b.named[c])
		}
		b.swap(cardLoc{kind: locSlot, i: i}, cardLoc{kind: locSlot, i: j})
		i = j
	}
	s.Slots[i] = Slot{}
	return nil
}

// intoStructure brings named card c of the current age into the structure,
// swapping it with an unnamed card of the same back.
func (b *scenarioBuilder) intoStructure(c catalog.CardID, why string) error {
	s := b.s
	l := b.locate(c)
	if l.kind == locSlot {
		return nil
	}
	if l.kind == locPlaced {
		return fmt.Errorf("%s: card %s is already placed elsewhere", why, c)
	}
	if !ageKinds(s.Age, cardBack(c)) {
		return fmt.Errorf("%s: card %s is not a card of age %d", why, c, s.Age)
	}
	for j := range s.Slots {
		y := s.Slots[j].Card
		if y != 0 && !b.pinned.Has(y) && cardBack(y) == cardBack(c) {
			slot := cardLoc{kind: locSlot, i: j}
			if l.kind == locOutOfPlay {
				s.OutOfPlay.Remove(c)
				s.OutOfPlay.Add(y)
				b.set(slot, c)
			} else {
				b.swap(l, slot)
			}
			return nil
		}
	}
	return fmt.Errorf("%s: no unnamed card with the back of %s is left in the structure to swap it with", why, c)
}

// placeOnlyLeft keeps exactly the OnlyLeft cards in the structure, in the
// first slots of layout order, and puts every other card out of play.
func (b *scenarioBuilder) placeOnlyLeft() error {
	if len(b.onlyLeft) == 0 {
		return nil
	}
	s := b.s
	if len(b.onlyLeft) > SlotsPerAge {
		return fmt.Errorf("onlyLeft names %d cards", len(b.onlyLeft))
	}
	for _, c := range b.onlyLeft {
		if err := b.intoStructure(c, "onlyLeft"); err != nil {
			return err
		}
	}
	for t, c := range b.onlyLeft {
		if j := SlotOf(s, c); j != t {
			b.swap(cardLoc{kind: locSlot, i: j}, cardLoc{kind: locSlot, i: t})
		}
	}
	for i := len(b.onlyLeft); i < SlotsPerAge; i++ {
		if c := s.Slots[i].Card; c != 0 {
			s.OutOfPlay.Add(c)
			s.Slots[i] = Slot{}
		}
	}
	return nil
}

// placeAccessible moves each Accessible card into an accessible slot.
func (b *scenarioBuilder) placeAccessible() error {
	s := b.s
	var placed uint32 // slots holding an Accessible card already placed
	for _, c := range b.accessible {
		if len(b.onlyLeft) > 0 && SlotOf(s, c) < 0 {
			return fmt.Errorf("accessible: card %s is not among the onlyLeft cards", c)
		}
		if err := b.intoStructure(c, "accessible"); err != nil {
			return err
		}
		acc := Accessible(s)
		i := SlotOf(s, c)
		if acc&(1<<i) == 0 {
			j := -1
			for pass := 0; pass < 2 && j < 0; pass++ {
				for k := range s.Slots {
					y := s.Slots[k].Card
					// First pass: an unnamed card. Second: an onlyLeft card.
					if acc&(1<<k) != 0 && placed&(1<<k) == 0 && (pass == 1 || !b.pinned.Has(y)) {
						j = k
						break
					}
				}
			}
			if j < 0 {
				return fmt.Errorf("accessible: no accessible slot is left for card %s", c)
			}
			b.swap(cardLoc{kind: locSlot, i: i}, cardLoc{kind: locSlot, i: j})
			i = j
		}
		placed |= 1 << i
	}
	return nil
}

// refreshFaces sets every occupied slot face up on the odd rows or when
// accessible, as it would be between turns, and shows each card's back.
func (b *scenarioBuilder) refreshFaces() {
	s := b.s
	acc := Accessible(s)
	parity := FaceUpMask(s.Age)
	for i := range s.Slots {
		sl := &s.Slots[i]
		if sl.Card == 0 {
			*sl = Slot{}
			continue
		}
		sl.FaceUp = (parity|acc)&(1<<i) != 0
		sl.Back = cardBack(sl.Card)
	}
}

// buildWonders builds the scenario's wonders and applies the seven-wonder
// rule.
func (b *scenarioBuilder) buildWonders() error {
	s := b.s
	for seat := range b.built {
		p := &s.Players[seat]
		for _, bw := range b.built[seat] {
			k := wonderSlot(p, bw.wonder)
			if k < 0 {
				return fmt.Errorf("seat %d does not have %s (its wonders: %s)", seat, bw.wonder, wonderList(p))
			}
			if p.Wonders[k].Built {
				return fmt.Errorf("seat %d builds %s twice", seat, bw.wonder)
			}
			tucked := bw.tucked
			if tucked == 0 {
				var err error
				if tucked, err = b.anyCard(); err != nil {
					return err
				}
			}
			buildWonder(s, Seat(seat), k, tucked)
		}
	}
	if s.WondersBuilt > MaxWondersBuilt {
		return fmt.Errorf("%d wonders are built; at most %d can be", s.WondersBuilt, MaxWondersBuilt)
	}
	sevenWonderRule(s, nil)
	return nil
}

// anyCard takes an unnamed card to tuck under a wonder: the lowest card out
// of play, else a removed card, else a box guild.
func (b *scenarioBuilder) anyCard() (catalog.CardID, error) {
	s := b.s
	if c, ok := s.OutOfPlay.Next(1); ok {
		s.OutOfPlay.Remove(c)
		return c, nil
	}
	for a := range s.Removed {
		for i, c := range s.Removed[a] {
			if c != 0 && !b.pinned.Has(c) {
				s.Removed[a][i] = 0
				return c, nil
			}
		}
	}
	for i, c := range s.BoxGuilds {
		if c != 0 && !b.pinned.Has(c) {
			s.BoxGuilds[i] = 0
			return c, nil
		}
	}
	return 0, fmt.Errorf("no card is left to tuck under a wonder")
}

func wonderList(p *Player) string {
	var names []string
	for _, ws := range p.Wonders {
		if ws.ID != 0 {
			names = append(names, ws.ID.String())
		}
	}
	return strings.Join(names, ", ")
}

// placeTokens deals the tokens to the players, the board and the box.
func (b *scenarioBuilder) placeTokens() error {
	s := b.s
	var owned, onBoard, inOrder catalog.TokenSet
	for seat := range b.tokens {
		for _, t := range b.tokens[seat] {
			if owned.Has(t) {
				return fmt.Errorf("token %s is given twice", t)
			}
			owned.Add(t)
		}
	}
	for _, t := range b.boxOrder {
		if owned.Has(t) || inOrder.Has(t) {
			return fmt.Errorf("boxTokenOrder: token %s is named twice", t)
		}
		inOrder.Add(t)
	}
	if len(b.board) > 0 {
		for _, t := range b.board {
			if owned.Has(t) || inOrder.Has(t) || onBoard.Has(t) {
				return fmt.Errorf("boardTokens: token %s is named twice", t)
			}
			onBoard.Add(t)
		}
	} else {
		onBoard = s.BoardTokens &^ owned &^ inOrder
	}
	// The box: the given order, then the dealt box order, then the dealt
	// board tokens that ended up nowhere, by id.
	var box []catalog.TokenID
	var inBox catalog.TokenSet
	add := func(t catalog.TokenID) {
		if t != 0 && !inBox.Has(t) && !owned.Has(t) && !onBoard.Has(t) {
			inBox.Add(t)
			box = append(box, t)
		}
	}
	for _, t := range b.boxOrder {
		add(t)
	}
	for _, t := range s.BoxTokens {
		add(t)
	}
	for t, ok := s.BoardTokens.Next(1); ok; t, ok = s.BoardTokens.Next(t + 1) {
		add(t)
	}
	if len(box) > NumBoxTokens {
		return fmt.Errorf("%d tokens would be in the box, which holds %d; put more on the board", len(box), NumBoxTokens)
	}
	s.BoardTokens = onBoard
	s.BoxTokens = [NumBoxTokens]catalog.TokenID{}
	copy(s.BoxTokens[:], box)
	for seat := range b.tokens {
		for _, t := range b.tokens[seat] {
			addToken(s, Seat(seat), t)
		}
	}
	return nil
}
