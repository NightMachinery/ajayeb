package ai_test

import (
	"context"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/ai"
	"github.com/NightMachinery/ajayeb/server/internal/engine"
)

// TestBotsNeverResign: a forfeit (resign:<seat>) is legal at any time but
// never among the legal actions a bot picks from, so no level ever plays
// one. Each level plays whole games against itself on a tiny budget.
func TestBotsNeverResign(t *testing.T) {
	for _, level := range []ai.Level{ai.Random, ai.Easy, ai.Medium, ai.Hard} {
		for g := range 2 {
			seed := uint64(0x5e51_6000 + g)
			s := engine.NewGame(seed, engine.Options{})
			bots := [2]ai.Bot{ai.New(level, seed), ai.New(level, seed+1)}
			for !s.Over() {
				a, err := bots[s.ToMove].Choose(context.Background(), engine.Observe(&s, s.ToMove), ai.Budget{MaxIterations: 8, Workers: 1}, nil)
				if err != nil {
					t.Fatalf("%s seed %d ply %d: %v", level, seed, s.Ply, err)
				}
				if a.Type == engine.ActResign {
					t.Fatalf("%s seed %d ply %d: the bot resigned", level, seed, s.Ply)
				}
				if err := engine.Apply(&s, a, nil); err != nil {
					t.Fatalf("%s seed %d ply %d: %v", level, seed, s.Ply, err)
				}
			}
			if s.Result.Kind == engine.VictoryForfeit {
				t.Fatalf("%s seed %d: a bot game ended by forfeit", level, seed)
			}
		}
	}
}
