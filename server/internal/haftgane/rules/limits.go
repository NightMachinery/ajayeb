// Package rules holds Haftgane's pure rules values and canonical vocabulary.
// No setup, legal-action generation or resolution is implemented in this slice.
package rules

import "github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"

const (
	RulesVersion       = 1
	MinSeats           = 3
	MaxSeats           = 7
	NumAges            = 3
	TurnsPerAge        = 6
	MaxTurns           = 7
	HandSize           = 7
	MaxStages          = catalog.MaxStages
	MaxBoxGuilds       = 5
	MaxChoice          = 10
	MaxForSaleChoice   = 6
	MaxPileLooks       = 3
	MaxPoints          = 25
	MaxKeyBytes        = 64
	MaxPaymentOutcomes = 15
	MaxTurnChoices     = 217
)
