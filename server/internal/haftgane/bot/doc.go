// Package bot implements observation-only Random, Greedy and Apprentice.
// Policies receive rules.Observation, the exact engine offer, independent
// policy entropy and a versioned per-bot work row. They never receive a Game,
// State, setup seed, other seat's hand or runtime scheduling metadata.
// Exact scores, placements and prices come from the rules observation API;
// forward-looking features are explicit fixed-point policy estimates.
// SampleRoot prepares a private root State only through the engine sampler.
// ChooseAge adds paired simultaneous Age sweeps; its simulated-seat policies
// retain their observation-only boundary. The admitted-worker entry points are
// cancellable and use deterministic Greedy fallback. Rooms select ChooseAge
// with the versioned ScholarWork row frozen by calibration.
package bot
