package fakeengine

// Sizes of the fake game.
const (
	MinSeats, MaxSeats = 3, 7
	NumAges            = 2
	TurnsPerAge        = 2
	HandSize           = 3
	MaxStages          = 2 // tucked cards per city
	MaxBox             = 4 // relics are the only extra copies
	SellCoins          = 2
	StartCoins         = 3

	// MaxPoints bounds the points of a game: per Age its turns, a salvage
	// after each, and a last card.
	MaxPoints = NumAges * (2*TurnsPerAge + 1)
	// MaxEvents bounds the events of one resolution: a play and a leftover
	// or kept card per seat, a salvage result, a pass, an Age start and the
	// game's end.
	MaxEvents = 2*MaxSeats + 4
)

// Seat is a rules position, 0 to Seats-1. Left(s) is s+1, Right(s) is s-1,
// both modulo the seat count, as in Haftgane.
type Seat uint8

// SeatSet is a set of seats, one bit per seat.
type SeatSet uint8

// Has reports whether s is in the set.
func (ss SeatSet) Has(s Seat) bool { return s < MaxSeats && ss&(1<<s) != 0 }

// Count returns the number of seats in the set.
func (ss SeatSet) Count() int {
	n := 0
	for s := Seat(0); s < MaxSeats; s++ {
		if ss.Has(s) {
			n++
		}
	}
	return n
}

// Seats returns the seats in the set, ascending.
func (ss SeatSet) Seats() []Seat {
	out := []Seat{}
	for s := Seat(0); s < MaxSeats; s++ {
		if ss.Has(s) {
			out = append(out, s)
		}
	}
	return out
}

// Phase is what the game waits for.
type Phase uint8

const (
	PhaseTurn     Phase = iota + 1 // every seat plays a card
	PhaseSalvage                   // the archive's holder takes a design from the pile
	PhaseLastCard                  // the orchard's holder plays its kept card
	PhaseOver
)

// SeatingRule decides the initial assignment of lobby slots to seats.
type SeatingRule uint8

const (
	SeatingRandom SeatingRule = iota // the default: a seeded permutation
	SeatingChosen                    // lobby order
)

// Options are the room options the rules see.
type Options struct {
	Seating   SeatingRule
	HideAtEnd bool // keep hidden cards hidden after the end (no download)
}

// Setup is everything a game is created from. A game is its setup plus its
// batches.
type Setup struct {
	Seed    uint64
	Seats   uint8
	Options Options
}

// City is one seat's public tableau plus its private tucked cards and kept
// card.
type City struct {
	Board  BoardID
	Built  uint32 // bit d set when design d is built
	Tucked [MaxStages]DesignID
	Stages uint8
	Coins  int16
	Kept   DesignID // the orchard's leftover, until its last card is played
}

// Score is one seat's final score.
type Score struct {
	Built, Stages, Treasure, Total int16
}

// Result is set when the game is over.
type Result struct {
	Over    bool
	Scores  [MaxSeats]Score
	Rank    [MaxSeats]uint8 // 1 is first; tied seats share a rank
	Winners SeatSet
}

// State is the current position: fixed-size, comparable, cloned by copy.
type State struct {
	Seed    uint64 // read only by NewGame; never in a view or observation
	Opts    Options
	Seats   uint8
	Phase   Phase
	Age     uint8
	Turn    uint8
	Points  uint16 // points resolved so far
	Decider Seat   // the seat of a salvage or last-card point
	Cities  [MaxSeats]City
	Hands   [MaxSeats][HandSize]DesignID // by holder, sorted, zero-padded
	Deals   [MaxSeats][HandSize]DesignID // Age 2's hands until dealt, sorted
	Box     [MaxBox]DesignID             // Age 2's undealt cards, sorted
	Pile    [NumDesigns + 1]uint8        // a count per design
	Result  Result
}

// ActionKind is what a key does.
type ActionKind uint8

const (
	ActBuild ActionKind = iota + 1
	ActTuck
	ActSell
	ActSalvage
)

// Action is a parsed key: a comparable tagged union.
type Action struct {
	Kind   ActionKind
	Design DesignID
	Left   uint8 // coins to the left neighbour (build only)
	Right  uint8 // coins to the right neighbour (build only)
}

// PointKind is the kind of a decision point.
type PointKind uint8

const (
	PointNone PointKind = iota // the game is over
	PointTurn
	PointSalvage
	PointLastCard
)

// Resolved is one resolved point in the record: which point, who was
// required, and every required seat's action.
type Resolved struct {
	Kind     PointKind
	Age      uint8
	Turn     uint8
	Required SeatSet
	Actions  [MaxSeats]Action
}

// Record is what has happened. Resolve is its only writer.
type Record struct {
	// Held is each seat's hand at the opening of each turn, sorted.
	Held [NumAges][TurnsPerAge][MaxSeats][HandSize]DesignID
	Log  [MaxPoints]Resolved
	NLog uint8
}

// Game is the engine value: the state and the record.
type Game struct {
	State  State
	Record Record
}

// Point is the open decision point: its logical id, kind and required
// seats. Its kind is PointNone once the game is over.
type Point struct {
	ID       string
	Kind     PointKind
	Required SeatSet
}

// Choice is one offered key with what it does.
type Choice struct {
	Key    string
	Kind   ActionKind
	Design DesignID
	Left   uint8
	Right  uint8
}

// Batch is one complete decision: the point's logical id and one key per
// required seat, sorted by seat.
type Batch struct {
	Point string
	Picks []Pick
}

// Pick is one seat's key in a batch.
type Pick struct {
	Seat Seat
	Key  string
}

// Viewer is who a view or history is for: a seat, or the spectator role.
type Viewer struct {
	Spectator bool
	Seat      Seat
}

// SeatViewer returns the viewer for seat s.
func SeatViewer(s Seat) Viewer { return Viewer{Seat: s} }

// SpectatorViewer returns the spectator role, which sees public
// information only.
func SpectatorViewer() Viewer { return Viewer{Spectator: true} }

// left and right return a seat's neighbours.
func (st *State) left(s Seat) Seat  { return Seat((uint8(s) + 1) % st.Seats) }
func (st *State) right(s Seat) Seat { return Seat((uint8(s) + st.Seats - 1) % st.Seats) }

// boardHolder returns the seat holding board b, if any.
func (st *State) boardHolder(b BoardID) (Seat, bool) {
	for s := Seat(0); uint8(s) < st.Seats; s++ {
		if st.Cities[s].Board == b {
			return s, true
		}
	}
	return 0, false
}

// built reports whether design d is in the city.
func (c *City) built(d DesignID) bool { return c.Built&(1<<d) != 0 }
