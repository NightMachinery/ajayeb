package view

// CardTooltipView describes an already visible building. The engine builds
// this index from the redacted view, never from hidden deals or removals.
// Costs are public quotes for both seats, independent of turn or legality.
// Rows follow catalog id order. The catalog supplies printed effects/links.
type CardTooltipView struct {
	Card       string                 `json:"card"`
	Costs      [2]CardTooltipCostView `json:"costs" tstype:"Pair<CardTooltipCostView>"`
	Place      PublicPlace            `json:"place"`
	Seat       *int                   `json:"seat,omitempty"`
	Slot       *int                   `json:"slot,omitempty"`
	Accessible bool                   `json:"accessible"`
}

// CardTooltipCostView is the server solver's compact payment summary. It
// omits production source ids and breakdowns, keeping the shared index small.
type CardTooltipCostView struct {
	Total      int  `json:"total"`
	Affordable bool `json:"affordable"`
	ChainFree  bool `json:"chainFree"`
	NeedsTrade bool `json:"needsTrade"`
	Shortfall  int  `json:"shortfall"`
}
