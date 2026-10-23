package tableroom

import (
	"encoding/json"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/tabletest"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest/leakcheck"
)

// Rich fields must not become escape hatches when the platform begins
// carrying the full real-engine view. Every canary changes a received frame.
func TestRealViewExtraCanaries(t *testing.T) {
	h := newTH(t, Config{}, humans(7, 2029))
	fr := h.latest(tabletest.SeatClient(0))
	truth, err := leakcheck.FromReal(makeSetup(2029, 7, false, false), nil)
	if err != nil {
		t.Fatal(err)
	}
	checker := leakcheck.New(truth, frameConfig)
	base := engineViewJSON(t, fr, nil)
	if got := checker.Check(leakcheck.Doc{Kind: leakcheck.View, Viewer: 0, Moment: 0, Data: base}); len(got) > 0 {
		t.Fatal(got)
	}
	foreign := truth.Moments[0].Hands[1][0]
	for name, plant := range map[string]func(map[string]any){
		"remembered opponent hand": func(v map[string]any) {
			v["you"].(map[string]any)["notebook"].(map[string]any)["seen"] = []any{map[string]any{"point": "age-1-turn-1", "hand": []string{foreign}, "played": foreign, "action": "sell"}}
		},
		"foreign pile look": func(v map[string]any) {
			v["you"].(map[string]any)["notebook"].(map[string]any)["pileLooks"] = []any{map[string]any{"point": "age-1-turn-1-discard-build", "cards": []any{map[string]any{"design": foreign, "count": 1}}}}
		},
		"foreign statistical deduction": func(v map[string]any) {
			v["you"].(map[string]any)["notebook"].(map[string]any)["statistics"].(map[string]any)["neighbourPicks"] = []any{map[string]any{"point": "age-1-turn-1", "seat": 1, "design": foreign}}
		},
		"foreign witness value": func(v map[string]any) {
			v["you"].(map[string]any)["choices"].([]any)[0].(map[string]any)["witness"] = []any{map[string]any{"resource": foreign, "source": "own-fixed", "producer": 0, "price": 0}}
		},
		"forged science counts": func(v map[string]any) {
			v["seats"].([]any)[0].(map[string]any)["science"].(map[string]any)["counts"] = []int{9, 9, 9}
		},
		"unknown nested hash": func(v map[string]any) {
			v["you"].(map[string]any)["notebook"].(map[string]any)["observationHash"] = truth.Secrets[1].Value
		},
	} {
		t.Run(name, func(t *testing.T) {
			var v map[string]any
			if err := json.Unmarshal(base, &v); err != nil {
				t.Fatal(err)
			}
			plant(v)
			data, _ := json.Marshal(v)
			if got := checker.Check(leakcheck.Doc{Kind: leakcheck.View, Viewer: 0, Moment: 0, Data: data}); len(got) == 0 {
				t.Fatal("contaminated rich frame accepted")
			}
		})
	}
}
