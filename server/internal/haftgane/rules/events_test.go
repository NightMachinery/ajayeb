package rules

import "testing"

// With payments represented by EvPlay, the design's 32-event budget covers a
// seven-seat sixth turn. The sweep also checks actual resolved event counts.
func TestResolutionEventBudget(t *testing.T) {
	const limit = MaxResolutionEvents
	tests := []struct {
		name                                                                        string
		plays, income, leftovers, pass, discardResult, conflicts, ageStart, gameEnd int
	}{
		{name: "sixth turn with next Age", plays: MaxSeats, income: MaxSeats, leftovers: MaxSeats, conflicts: MaxSeats, ageStart: 1},
		{name: "sixth turn with skipped discard look", plays: MaxSeats, income: MaxSeats, leftovers: MaxSeats, discardResult: 1, conflicts: MaxSeats, ageStart: 1},
		{name: "sixth turn at game end", plays: MaxSeats, income: MaxSeats, leftovers: MaxSeats, conflicts: MaxSeats, gameEnd: 1},
		{name: "ordinary turn with skipped discard look", plays: MaxSeats, income: MaxSeats, pass: 1, discardResult: 1},
		{name: "last card then game end", plays: 1, income: 1, conflicts: MaxSeats, gameEnd: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := tt.plays + tt.income + tt.leftovers + tt.pass + tt.discardResult + tt.conflicts + tt.ageStart + tt.gameEnd
			if n > limit {
				t.Fatalf("%d events exceeds %d", n, limit)
			}
			t.Logf("conservative budget %d/%d", n, limit)
		})
	}
}
