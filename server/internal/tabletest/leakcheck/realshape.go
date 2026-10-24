package leakcheck

// Independent real-view whitelist. Unknown nested fields fail closed.
type realTable struct {
	Point       string       `json:"point"`
	Kind        string       `json:"kind"`
	Required    []int        `json:"required"`
	Phase       string       `json:"phase"`
	Age         int          `json:"age"`
	Turn        int          `json:"turn"`
	Pass        string       `json:"pass"`
	Pile        int          `json:"pile"`
	Seats       []realSeat   `json:"seats"`
	You         *realPrivate `json:"you,omitempty"`
	Events      []wEvent     `json:"events"`
	Result      *realResult  `json:"result,omitempty"`
	Reveal      *realReveal  `json:"reveal,omitempty"`
	CanDownload bool         `json:"canDownload"`
}
type realSeat struct {
	Seat      int         `json:"seat"`
	Board     string      `json:"board"`
	Side      string      `json:"side"`
	Stages    int         `json:"stages"`
	Built     []string    `json:"built"`
	Coins     int         `json:"coins"`
	Victories []int       `json:"victories"`
	Defeats   int         `json:"defeats"`
	Shields   int         `json:"shields"`
	HandSize  int         `json:"handSize"`
	HasKept   bool        `json:"hasKept"`
	Relative  int         `json:"relative"`
	Relation  string      `json:"relation"`
	Last      *realPlay   `json:"last,omitempty"`
	Score     realScore   `json:"score"`
	Science   realScience `json:"science"`
}
type realPlay struct {
	Kind    string `json:"kind"`
	Design  string `json:"design"`
	Stage   int    `json:"stage"`
	Left    int    `json:"left"`
	Right   int    `json:"right"`
	Printed int    `json:"printed"`
}
type realPrivate struct {
	Seat       int              `json:"seat"`
	Hand       []string         `json:"hand"`
	Kept       string           `json:"kept"`
	Tucked     []string         `json:"tucked"`
	Choices    []realChoice     `json:"choices"`
	Disabled   []realDisabled   `json:"disabled"`
	Offer      []realOfferEntry `json:"offer"`
	Notebook   realNotebook     `json:"notebook"`
	Neighbours []realNeighbour  `json:"neighbours"`
}
type realChoice struct {
	Key     string            `json:"key"`
	Kind    string            `json:"kind"`
	Design  string            `json:"design"`
	Side    string            `json:"side"`
	Stage   int               `json:"stage"`
	Left    int               `json:"left"`
	Right   int               `json:"right"`
	Printed int               `json:"printed"`
	Total   int               `json:"total"`
	Gain    int               `json:"gain"`
	Free    string            `json:"free"`
	Witness []realPaymentUnit `json:"witness"`
	Kept    *string           `json:"kept,omitempty"`
}
type realPaymentUnit struct {
	Resource string `json:"resource"`
	Source   string `json:"source"`
	Producer int    `json:"producer"`
	Price    int    `json:"price"`
}
type realDisabled struct {
	Kind     string `json:"kind"`
	Design   string `json:"design"`
	Stage    int    `json:"stage"`
	Reason   string `json:"reason"`
	Cheapest int    `json:"cheapest"`
	ShortBy  int    `json:"shortBy"`
}
type realOfferEntry struct {
	Design string `json:"design"`
	Count  int    `json:"count"`
	Legal  bool   `json:"legal"`
}
type realNotebook struct {
	Seen       []realSeenEntry `json:"seen"`
	PileLooks  []realPileLook  `json:"pileLooks"`
	Statistics realStatistics  `json:"statistics"`
}
type realSeenEntry struct {
	Point  string   `json:"point"`
	Hand   []string `json:"hand"`
	Played string   `json:"played"`
	Action string   `json:"action"`
}
type realPileLook struct {
	Point string            `json:"point"`
	Cards []realDesignCount `json:"cards"`
}
type realDesignCount struct {
	Design string `json:"design"`
	Count  int    `json:"count"`
}
type realBounds struct {
	Certain int `json:"certain"`
	AtMost  int `json:"atMost"`
}
type realColourBound struct {
	Colour string     `json:"colour"`
	Bounds realBounds `json:"bounds"`
}
type realDesignBound struct {
	Design string     `json:"design"`
	Bounds realBounds `json:"bounds"`
}
type realKnownPick struct {
	Point  string `json:"point"`
	Seat   int    `json:"seat"`
	Design string `json:"design"`
}
type realStatistics struct {
	Circulating    []realColourBound `json:"circulating"`
	Returning      []realDesignBound `json:"returning"`
	NeighbourPicks []realKnownPick   `json:"neighbourPicks"`
}
type realNeighbour struct {
	Seat      int             `json:"seat"`
	Direction string          `json:"direction"`
	Units     []realTradeUnit `json:"units"`
	Outlook   int             `json:"outlook"`
}

// A choice symbol sells one unit, selected from its options. Fixed symbols
// have one option and Count physical units. Producer uses the same canonical
// per-source index as a payment witness.
type realTradeUnit struct {
	Choice   bool             `json:"choice"`
	Producer int              `json:"producer"`
	Count    int              `json:"count"`
	Options  []realTradePrice `json:"options"`
}
type realTradePrice struct {
	Resource string        `json:"resource"`
	Price    int           `json:"price"`
	Discount *realDiscount `json:"discount,omitempty"`
}
type realDiscount struct {
	Design string `json:"design"`
	Stage  int    `json:"stage"`
}
type realScience struct {
	Counts  []int `json:"counts"`
	Choices []int `json:"choices"`
	Points  int   `json:"points"`
}
type realScore struct {
	Wonder   int `json:"wonder"`
	Treasure int `json:"treasure"`
	Military int `json:"military"`
	Blue     int `json:"blue"`
	Yellow   int `json:"yellow"`
	Green    int `json:"green"`
	Purple   int `json:"purple"`
	Total    int `json:"total"`
}
type realResult struct {
	Scores   []realScore `json:"scores"`
	Rank     []int       `json:"rank"`
	Winners  []int       `json:"winners"`
	TieBreak string      `json:"tieBreak"`
	Tied     []int       `json:"tied"`
}
type realReveal struct {
	Tucked    [][]string     `json:"tucked"`
	Leftovers []realLeftover `json:"leftovers"`
	Pile      []string       `json:"pile"`
	Box       []string       `json:"box"`
}
type realLeftover struct {
	Age    int    `json:"age"`
	Seat   int    `json:"seat"`
	Design string `json:"design"`
	Kept   bool   `json:"kept"`
}
