package room

import (
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/store"
	"github.com/NightMachinery/ajayeb/server/internal/view"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

func TestUndoUnavailableReasonsAndFramePaths(t *testing.T) {
	cases := []struct {
		name                                  string
		mode                                  store.Mode
		off, spectator, over, human, decision bool
		want                                  wire.UndoUnavailableReason
	}{
		{"online precedence", store.ModeOnline, true, true, true, false, false, wire.UndoUnavailableOnline},
		{"disabled precedence", store.ModeHotseat, true, true, true, false, false, wire.UndoUnavailableDisabled},
		{"spectator precedence", store.ModeAI, false, true, true, true, true, wire.UndoUnavailableSpectator},
		{"game over", store.ModeHotseat, false, false, true, true, true, wire.UndoUnavailableGameOver},
		{"no decision", store.ModeHotseat, false, false, false, true, false, wire.UndoUnavailableNoHumanDecision},
		{"bot first", store.ModeAI, false, false, false, false, true, wire.UndoUnavailableNoHumanDecision},
		{"available", store.ModeHotseat, false, false, false, true, true, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := view.HintSettings{}
			f := &store.File{Mode: tc.mode, NoUndo: tc.off, Hints: &h}
			f.Seats[0].Kind = store.SeatKindBot
			if tc.human {
				f.Seats[0].Kind = store.SeatKindHuman
			}
			f.Seats[1].Kind = store.SeatKindHuman
			r := &Room{file: f, state: engine.NewGame(7, engine.Options{})}
			if tc.over {
				r.state.Phase = engine.PhaseGameOver
			}
			if tc.decision {
				r.hist = []histEntry{{seat: 0, events: []engine.Event{}}}
			}
			mem := member{role: roleHotseat}
			if tc.spectator {
				mem.role = roleSpectator
			}
			for _, withHistory := range []bool{false, true} {
				var fc frameCache
				st := r.frame(mem, nil, nil, [2]bool{}, &fc, withHistory)
				if st.CanUndo != (tc.want == "") || st.CanUndo != r.canUndo(mem) || st.CanUndo != (st.UndoUnavailable == nil) {
					t.Fatalf("inconsistent availability: %+v", st)
				}
				if tc.want != "" && *st.UndoUnavailable != tc.want {
					t.Fatalf("reason %s, want %s", *st.UndoUnavailable, tc.want)
				}
			}
		})
	}
}
