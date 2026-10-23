// Package fakeview holds the fake engine's view DTOs, the counterpart of
// haftgane/view for the real engine. Every slice is non-nil, every field has
// an explicit JSON name, and omitempty appears only on pointer fields, under
// the same rules as Duel's view package. Cards are catalog slugs, never copy
// or deal positions, and every card list is sorted in catalog order.
//
// Like fakeengine it is test infrastructure (see fakeengine's package
// documentation for who may import it).
package fakeview

// Table is one viewer's view of the game: public for everyone, plus You for
// a seated viewer and Reveal once the game is over with the end reveal on.
type Table struct {
	Point    string   `json:"point"`
	Kind     string   `json:"kind"`
	Required []int    `json:"required"`
	Phase    string   `json:"phase"`
	Age      int      `json:"age"`
	Turn     int      `json:"turn"`
	Pass     string   `json:"pass"`
	Pile     int      `json:"pile"`
	Seats    []Seat   `json:"seats"`
	You      *Private `json:"you,omitempty"`
	Events   []Event  `json:"events"`
	Result   *Result  `json:"result,omitempty"`
	Reveal   *Reveal  `json:"reveal,omitempty"`
}

// Seat is one seat's public city.
type Seat struct {
	Seat     int      `json:"seat"`
	Board    string   `json:"board"`
	Coins    int      `json:"coins"`
	Built    []string `json:"built"`
	Stages   int      `json:"stages"`
	HandSize int      `json:"handSize"`
	HasKept  bool     `json:"hasKept"`
	Last     *Play    `json:"last,omitempty"`
}

// Play is a seat's latest play as everyone saw it: the design only for a
// build or a salvage.
type Play struct {
	Kind   string `json:"kind"`
	Design string `json:"design"`
	Stage  int    `json:"stage"`
	Left   int    `json:"left"`
	Right  int    `json:"right"`
}

// Private is what only the seat itself sees.
type Private struct {
	Seat    int          `json:"seat"`
	Hand    []string     `json:"hand"`
	Kept    string       `json:"kept"`
	Tucked  []string     `json:"tucked"`
	Choices []Choice     `json:"choices"`
	Offer   []OfferEntry `json:"offer"`
	Seen    []SeenEntry  `json:"seen"`
}

// Choice is one offered key, in the engine's canonical order.
type Choice struct {
	Key    string `json:"key"`
	Kind   string `json:"kind"`
	Design string `json:"design"`
	Left   int    `json:"left"`
	Right  int    `json:"right"`
}

// OfferEntry is one design of the pile, shown only to the archive's holder
// during its own salvage point.
type OfferEntry struct {
	Design string `json:"design"`
	Count  int    `json:"count"`
	Legal  bool   `json:"legal"`
}

// SeenEntry is one turn the seat played: the hand it held and what it did
// with which card (its notebook, the platform's Seen hands).
type SeenEntry struct {
	Point  string   `json:"point"`
	Hand   []string `json:"hand"`
	Played string   `json:"played"`
	Action string   `json:"action"`
}

// Event is one filtered event. Design is empty where the viewer may not
// know the card.
type Event struct {
	Kind   string `json:"kind"`
	Seat   int    `json:"seat"`
	Play   string `json:"play"`
	Design string `json:"design"`
	Stage  int    `json:"stage"`
	Left   int    `json:"left"`
	Right  int    `json:"right"`
	Age    int    `json:"age"`
	Pass   string `json:"pass"`
}

// HistoryEntry is one resolved point and its filtered events.
type HistoryEntry struct {
	Point  string  `json:"point"`
	Events []Event `json:"events"`
}

// Result is the final score, public under either end policy.
type Result struct {
	Scores  []Score `json:"scores"`
	Rank    []int   `json:"rank"`
	Winners []int   `json:"winners"`
}

// Score is one seat's score.
type Score struct {
	Seat     int `json:"seat"`
	Built    int `json:"built"`
	Stages   int `json:"stages"`
	Treasure int `json:"treasure"`
	Total    int `json:"total"`
}

// Reveal is every hidden card, shown once the game is over unless the room
// turned the end reveal off.
type Reveal struct {
	Tucked [][]string `json:"tucked"`
	Pile   []string   `json:"pile"`
	Box    []string   `json:"box"`
}
