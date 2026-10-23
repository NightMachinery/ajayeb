package rules

import (
	"fmt"
	"slices"

	"github.com/NightMachinery/ajayeb/server/internal/enginekit/rng"
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
)

// Setup is the replayable starting input. Scenario is for tests and development
// positions only; none of its references is retained by the fixed-value Game.
type Setup struct {
	Seed     uint64    `json:"seed"`
	Seats    uint8     `json:"seats"`
	Options  Options   `json:"options"`
	Scenario *Scenario `json:"scenario,omitempty"`
}

// ASCII domains "haft-sea", "haft-boa", "haft-sid", "haft-gui", "haft-agN".
// These tags and every input/draw order are rules-versioned and golden-pinned.
const (
	streamSeating uint64 = 0x686166742d736561
	streamBoards  uint64 = 0x686166742d626f61
	streamSides   uint64 = 0x686166742d736964
	streamGuilds  uint64 = 0x686166742d677569
	streamAge1    uint64 = 0x686166742d616731
	streamAge2    uint64 = 0x686166742d616732
	streamAge3    uint64 = 0x686166742d616733
)

// SeatOrder maps each engine seat to its lobby slot. The actor applies this
// permutation once. It never changes board/deck draws. Unused/invalid slots
// are NoSeat, including the whole result for an invalid count or rule.
func SeatOrder(seed uint64, n uint8, rule SeatingRule) [MaxSeats]uint8 {
	var order [MaxSeats]uint8
	for i := range order {
		order[i] = uint8(NoSeat)
	}
	if n < MinSeats || n > MaxSeats || rule >= NumSeatingRules {
		return order
	}
	for i := uint8(0); i < n; i++ {
		order[i] = i
	}
	if rule == SeatingRandom {
		r := rng.NewRNG2(seed, streamSeating)
		rng.ShuffleSlice(&r, order[:n])
	}
	return order
}

func NewGame(s Setup) (Game, error) {
	if s.Seats < MinSeats || s.Seats > MaxSeats {
		return Game{}, fmt.Errorf("rules: seats must be %d..%d", MinSeats, MaxSeats)
	}
	if err := s.Options.Validate(); err != nil {
		return Game{}, err
	}
	if s.Scenario != nil {
		if s.Scenario.State.Seats != s.Seats {
			return Game{}, fmt.Errorf("rules: scenario seat count disagrees with setup")
		}
		return NewGameFromScenario(s.Seed, s.Options, s.Scenario)
	}
	g := Game{State: State{Seed: s.Seed, Seats: s.Seats, Opts: s.Options, Decider: NoSeat}}
	var boards [catalog.NumBoards]catalog.BoardID
	for i := range boards {
		boards[i] = catalog.BoardID(i + 1)
	}
	br := rng.NewRNG2(s.Seed, streamBoards)
	rng.ShuffleSlice(&br, boards[:])
	sr := rng.NewRNG2(s.Seed, streamSides)
	for seat := uint8(0); seat < s.Seats; seat++ {
		side := catalog.NoSide
		switch s.Options.Sides {
		case SidesDay:
			side = catalog.Day
		case SidesRandom:
			side = catalog.Side(1 + sr.IntN(2))
		}
		city, err := cityCaches(City{Board: boards[seat], Side: side, Coins: 3})
		if err != nil {
			return Game{}, err
		}
		g.State.Cities[seat] = city
	}
	var guilds [catalog.NumGuilds]catalog.DesignID
	count := 0
	for id := catalog.DesignID(1); id <= catalog.NumDesigns; id++ {
		d, ok := catalog.DesignByID(id)
		if !ok {
			return Game{}, fmt.Errorf("rules: catalog design %d missing", id)
		}
		if d.Color == catalog.Purple {
			if count >= len(guilds) {
				return Game{}, fmt.Errorf("rules: too many guilds")
			}
			guilds[count] = id
			count++
		}
	}
	if count != len(guilds) {
		return Game{}, fmt.Errorf("rules: catalog needs %d guilds", catalog.NumGuilds)
	}
	gr := rng.NewRNG2(s.Seed, streamGuilds)
	rng.ShuffleSlice(&gr, guilds[:])
	drawn := int(s.Seats) + 2
	copy(g.State.BoxGuilds[:], guilds[drawn:])
	canonicalIDs(g.State.BoxGuilds[:])
	// Guild shuffle chooses membership only; the deck's input is always id order.
	slices.Sort(guilds[:drawn])
	for age := uint8(1); age <= NumAges; age++ {
		hand, err := setupAge(s.Seed, s.Seats, age, guilds[:drawn])
		if err != nil {
			return Game{}, err
		}
		g.State.Deals[age-1] = hand
	}
	if s.Options.Sides != SidesChoose {
		if err := dealAge(&g, 1); err != nil {
			return Game{}, err
		}
	}
	return g, nil
}

func setupAge(seed uint64, n, age uint8, guilds []catalog.DesignID) ([MaxSeats][HandSize]catalog.DesignID, error) {
	var packets [MaxSeats][HandSize]catalog.DesignID
	var deck [MaxSeats * HandSize]catalog.DesignID
	used := 0
	add := func(id catalog.DesignID) bool {
		if used == len(deck) {
			return false
		}
		deck[used] = id
		used++
		return true
	}
	for id := catalog.DesignID(1); id <= catalog.NumDesigns; id++ {
		d, ok := catalog.DesignByID(id)
		if !ok {
			return packets, fmt.Errorf("rules: missing design %d", id)
		}
		if d.Age != age || d.Color == catalog.Purple {
			continue
		}
		for _, marker := range d.Copies {
			if marker != 0 && marker <= n {
				if !add(id) {
					return packets, fmt.Errorf("rules: deck overflow")
				}
			}
		}
	}
	if age == 3 {
		for _, id := range guilds {
			if !add(id) {
				return packets, fmt.Errorf("rules: guild deck overflow")
			}
		}
	}
	if used != int(n)*HandSize {
		return packets, fmt.Errorf("rules: Age %d has %d cards, want %d", age, used, int(n)*HandSize)
	}
	tags := [NumAges]uint64{streamAge1, streamAge2, streamAge3}
	r := rng.NewRNG2(seed, tags[age-1])
	rng.ShuffleSlice(&r, deck[:used])
	for seat := uint8(0); seat < n; seat++ {
		copy(packets[seat][:], deck[int(seat)*HandSize:int(seat+1)*HandSize])
		slices.Sort(packets[seat][:])
	}
	return packets, nil
}

// dealAge transfers, never duplicates: undealt packets leave State.Deals when
// they enter current hands and the historical Record.Packets.
func dealAge(g *Game, age uint8) error {
	if age < 1 || age > NumAges || g.State.Hands != ([MaxSeats][HandSize]catalog.DesignID{}) || g.Record.Packets[age-1] != ([MaxSeats][HandSize]catalog.DesignID{}) {
		return fmt.Errorf("rules: invalid Age deal transition")
	}
	for s := uint8(0); s < g.State.Seats; s++ {
		if _, err := checkIDs(g.State.Deals[age-1][s][:], age, HandSize); err != nil {
			return err
		}
	}
	g.State.Hands = g.State.Deals[age-1]
	g.Record.Packets[age-1] = g.State.Hands
	g.State.Deals[age-1] = [MaxSeats][HandSize]catalog.DesignID{}
	g.State.Phase, g.State.Age, g.State.Turn, g.State.Decider = PhaseTurn, age, 1, NoSeat
	return nil
}

func (g *Game) Point() Point {
	if g == nil || AllSeats(g.State.Seats) == 0 {
		return Point{}
	}
	st := &g.State
	switch st.Phase {
	case PhaseSides:
		return Point{Kind: PointSides, Required: AllSeats(st.Seats)}
	case PhaseTurn:
		return Point{Kind: PointTurn, Age: st.Age, Turn: st.Turn, Required: AllSeats(st.Seats)}
	case PhaseDiscardBuild:
		return Point{Kind: PointDiscardBuild, Age: st.Age, Turn: st.Turn, Required: st.Decider.Mask()}
	case PhaseLastCard:
		return Point{Kind: PointLastCard, Age: st.Age, Turn: st.Turn, Required: st.Decider.Mask()}
	default:
		return Point{}
	}
}

func sideActions(st *State, seat Seat) ([2]Action, bool) {
	if st == nil || st.Phase != PhaseSides || st.Opts.Sides != SidesChoose || !seat.Valid(st.Seats) || st.Cities[seat].Side != catalog.NoSide {
		return [2]Action{}, false
	}
	return [2]Action{{Kind: ActSide, Side: catalog.Day}, {Kind: ActSide, Side: catalog.Night}}, true
}

// resolveSides is the side-point branch for the later batch resolver. There is
// no partial engine state: the room holds pending choices until all are ready.
func resolveSides(g Game, picks [MaxSeats]Action) (Game, []Event, error) {
	if AllSeats(g.State.Seats) == 0 || g.State.Phase != PhaseSides {
		return g, nil, fmt.Errorf("rules: no side point")
	}
	for seat, pick := range picks {
		if seat >= int(g.State.Seats) {
			if pick != (Action{}) {
				return g, nil, fmt.Errorf("rules: side pick outside table")
			}
			continue
		}
		offered, ok := sideActions(&g.State, Seat(seat))
		if !ok || (pick != offered[0] && pick != offered[1]) {
			return g, nil, fmt.Errorf("rules: invalid side pick for seat %d", seat)
		}
	}
	next := g
	events := make([]Event, 0, int(g.State.Seats)+1)
	for seat := uint8(0); seat < g.State.Seats; seat++ {
		next.State.Cities[seat].Side = picks[seat].Side
		events = append(events, Event{Kind: EvSideChosen, Seat: Seat(seat), Side: picks[seat].Side})
	}
	if err := dealAge(&next, 1); err != nil {
		return g, nil, err
	}
	next.State.Points++
	return next, append(events, Event{Kind: EvAgeStart, Age: 1}), nil
}
