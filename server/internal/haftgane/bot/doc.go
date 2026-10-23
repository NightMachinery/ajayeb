// Package bot implements observation-only Random, Greedy and Apprentice.
// Policies receive rules.Observation, the exact engine offer, independent
// policy entropy and a versioned per-bot work row. They never receive a Game,
// State, setup seed, other seat's hand or runtime scheduling metadata.
// Exact scores, placements and prices come from the rules observation API;
// forward-looking features are explicit fixed-point policy estimates.
package bot
