// Package rules implements Haftgane's deterministic base-game engine: setup,
// legal simultaneous choices, resolution, scoring, replay, whitelisted views
// and permitted observations. Rules-gate tests prove the capacities below.
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

// Reachable-game bounds. TestRulesGateDimensions derives these from the
// catalog, unique boards and point structure; TestRulesGateSweep checks every
// natural point. Explicit partial-history scenarios may have larger cities.
const (
	MaxCityBuildings     = 20 // Babylon: 18 ordinary + 3 kept cards - its ability stage.
	MaxPileCards         = NumAges * HandSize * MaxSeats
	MaxResolutionEvents  = 32 // Four per-seat event groups plus three transition events.
	MaxKnownHands        = NumAges * MaxTurns
	MaxSeenCards         = NumAges * HandSize * (HandSize + 1) / 2
	MaxHistoryEntries    = MaxPoints + 1 // Optional initial Age-I entry and resolved points.
	MaxCanonicalKeyBytes = len("build-discarded:") + catalog.MaxSlugBytes
)

// Value-size and canonical field-encoding bounds. Memory ceilings tolerate ABI
// padding; encoded sizes contain no padding and are versioned by field order.
// TestRulesGateDimensions and the fixed-value guards check them independently.
const (
	MaxStateBytes            = 1536
	MaxRecordBytes           = 1536
	MaxGameBytes             = 3072
	MaxObservationBytes      = 2048
	GameEncodingBytes        = 2157
	ObservationEncodingBytes = 1693
)

// TestRulesGateViewByteBound materializes the largest structural DTO grammar:
// every optional section, bounded list, 64-byte safe-ASCII string and widest
// emitted integer at once. Partial-history scenarios are covered too. This
// conservative ceiling is not a transport cap or a reachable-view claim.
const (
	MaxViewStringBytes = MaxKeyBytes
	MaxViewJSONBytes   = 570406
)
