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
	ChainIn    *CardChainTooltipView  `json:"chainIn,omitempty"`
	ChainOut   *CardChainTooltipView  `json:"chainOut,omitempty"`
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

// CardChainTooltipView describes the printed link's counterpart. Its identity
// comes from the public catalog, never a hidden-state card id. Costs are only
// supplied for an outgoing link, quoted without the chain for either seat.
type CardChainTooltipView struct {
	Place      PublicPlace             `json:"place"`
	Seat       *int                    `json:"seat,omitempty"`
	Slot       *int                    `json:"slot,omitempty"`
	Accessible bool                    `json:"accessible"`
	Costs      *[2]CardTooltipCostView `json:"costs,omitempty" tstype:"Pair<CardTooltipCostView>"`
}
