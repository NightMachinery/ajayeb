// Package view holds seedless, whitelisted Haftgane screen DTOs. Rules owns
// every derived field; the client renders these facts and submits offered keys.
package view

type Table struct {
	Point       string   `json:"point"`
	Kind        string   `json:"kind"`
	Required    []int    `json:"required"`
	Phase       string   `json:"phase"`
	Age         int      `json:"age"`
	Turn        int      `json:"turn"`
	Pass        string   `json:"pass"`
	Pile        int      `json:"pile"`
	Seats       []Seat   `json:"seats"`
	You         *Private `json:"you,omitempty"`
	Events      []Event  `json:"events"`
	Result      *Result  `json:"result,omitempty"`
	Reveal      *Reveal  `json:"reveal,omitempty"`
	CanDownload bool     `json:"canDownload"`
}
type Seat struct {
	Seat      int      `json:"seat"`
	Board     string   `json:"board"`
	Side      string   `json:"side"`
	Stages    int      `json:"stages"`
	Built     []string `json:"built"`
	Coins     int      `json:"coins"`
	Victories []int    `json:"victories"`
	Defeats   int      `json:"defeats"`
	Shields   int      `json:"shields"`
	HandSize  int      `json:"handSize"`
	HasKept   bool     `json:"hasKept"`
	Relative  int      `json:"relative"`
	Relation  string   `json:"relation"`
	Last      *Play    `json:"last,omitempty"`
	Score     Score    `json:"score"`
	Science   Science  `json:"science"`
}
type Play struct {
	Kind    string `json:"kind"`
	Design  string `json:"design"`
	Stage   int    `json:"stage"`
	Left    int    `json:"left"`
	Right   int    `json:"right"`
	Printed int    `json:"printed"`
}
type Private struct {
	Seat       int          `json:"seat"`
	Hand       []string     `json:"hand"`
	Kept       string       `json:"kept"`
	Tucked     []string     `json:"tucked"`
	Choices    []Choice     `json:"choices"`
	Disabled   []Disabled   `json:"disabled"`
	Offer      []OfferEntry `json:"offer"`
	Notebook   Notebook     `json:"notebook"`
	Neighbours []Neighbour  `json:"neighbours"`
}
type Choice struct {
	Key     string        `json:"key"`
	Kind    string        `json:"kind"`
	Design  string        `json:"design"`
	Side    string        `json:"side"`
	Stage   int           `json:"stage"`
	Left    int           `json:"left"`
	Right   int           `json:"right"`
	Printed int           `json:"printed"`
	Total   int           `json:"total"`
	Gain    int           `json:"gain"`
	Free    string        `json:"free"`
	Witness []PaymentUnit `json:"witness"`
}
type PaymentUnit struct {
	Resource string `json:"resource"`
	Source   string `json:"source"`
	Producer int    `json:"producer"`
	Price    int    `json:"price"`
}
type Disabled struct {
	Kind     string `json:"kind"`
	Design   string `json:"design"`
	Stage    int    `json:"stage"`
	Reason   string `json:"reason"`
	Cheapest int    `json:"cheapest"`
	ShortBy  int    `json:"shortBy"`
}
type OfferEntry struct {
	Design string `json:"design"`
	Count  int    `json:"count"`
	Legal  bool   `json:"legal"`
}
type Notebook struct {
	Seen       []SeenEntry `json:"seen"`
	PileLooks  []PileLook  `json:"pileLooks"`
	Statistics Statistics  `json:"statistics"`
}
type SeenEntry struct {
	Point  string   `json:"point"`
	Hand   []string `json:"hand"`
	Played string   `json:"played"`
	Action string   `json:"action"`
}
type PileLook struct {
	Point string        `json:"point"`
	Cards []DesignCount `json:"cards"`
}
type DesignCount struct {
	Design string `json:"design"`
	Count  int    `json:"count"`
}
type Bounds struct {
	Certain int `json:"certain"`
	AtMost  int `json:"atMost"`
}
type ColourBound struct {
	Colour string `json:"colour"`
	Bounds Bounds `json:"bounds"`
}
type DesignBound struct {
	Design string `json:"design"`
	Bounds Bounds `json:"bounds"`
}
type KnownPick struct {
	Point  string `json:"point"`
	Seat   int    `json:"seat"`
	Design string `json:"design"`
}
type Statistics struct {
	Circulating    []ColourBound `json:"circulating"`
	Returning      []DesignBound `json:"returning"`
	NeighbourPicks []KnownPick   `json:"neighbourPicks"`
}
type Neighbour struct {
	Seat      int         `json:"seat"`
	Direction string      `json:"direction"`
	Units     []TradeUnit `json:"units"`
	Outlook   int         `json:"outlook"`
}

// A choice symbol sells one unit, selected from its options. Fixed symbols
// have one option and Count physical units. Producer uses the same canonical
// per-source index as a payment witness.
type TradeUnit struct {
	Choice   bool         `json:"choice"`
	Producer int          `json:"producer"`
	Count    int          `json:"count"`
	Options  []TradePrice `json:"options"`
}
type TradePrice struct {
	Resource string    `json:"resource"`
	Price    int       `json:"price"`
	Discount *Discount `json:"discount,omitempty"`
}
type Discount struct {
	Design string `json:"design"`
	Stage  int    `json:"stage"`
}
type Science struct {
	Counts  []int `json:"counts"`
	Choices []int `json:"choices"`
	Points  int   `json:"points"`
}
type Score struct {
	Wonder   int `json:"wonder"`
	Treasure int `json:"treasure"`
	Military int `json:"military"`
	Blue     int `json:"blue"`
	Yellow   int `json:"yellow"`
	Green    int `json:"green"`
	Purple   int `json:"purple"`
	Total    int `json:"total"`
}
type Result struct {
	Scores  []Score `json:"scores"`
	Rank    []int   `json:"rank"`
	Winners []int   `json:"winners"`
}
type Reveal struct {
	Tucked    [][]string `json:"tucked"`
	Leftovers []Leftover `json:"leftovers"`
	Pile      []string   `json:"pile"`
	Box       []string   `json:"box"`
}
type Leftover struct {
	Age    int    `json:"age"`
	Seat   int    `json:"seat"`
	Design string `json:"design"`
	Kept   bool   `json:"kept"`
}
