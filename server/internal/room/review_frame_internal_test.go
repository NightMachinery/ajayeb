package room

import (
	"context"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/store"
	"github.com/NightMachinery/ajayeb/server/internal/view"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// A completed quick review belongs to the move's frame. A forced decision
// guarantees completion without ranking, independent of scheduler speed;
// an expired parent deadline deterministically exercises the allowed drop.
func TestReviewQuickInPlace(t *testing.T) {
	for _, expired := range []bool{false, true} {
		name := "completed"
		if expired {
			name = "expired"
		}
		t.Run(name, func(t *testing.T) {
			s := namedBefore(t, "civilian-end")
			s.Players[0].Coins = 0
			key := "discard:academy"
			if expired {
				s = namedBefore(t, "military-supremacy")
				key = "discard:walls"
			} else if legal := engine.LegalActions(&s, nil); len(legal) != 1 || legal[0].Key() != key {
				t.Fatalf("completion fixture must need no ranking: %v", legal)
			}
			ctx := context.Background()
			if expired {
				var cancel context.CancelFunc
				ctx, cancel = context.WithDeadline(ctx, time.Unix(1, 0))
				defer cancel()
				if ctx.Err() != context.DeadlineExceeded {
					t.Fatal("expected an already expired deadline")
				}
			}
			hints := view.HintSettings{Advisor: true, AdvisorStrength: view.AdvisorStrengthQuick}
			f := &store.File{Mode: store.ModeAI, Seed: 3, Hints: &hints}
			f.Seats[0].Kind = store.SeatKindHuman
			f.Seats[1].Kind = store.SeatKindBot
			r := &Room{m: &Manager{cfg: Config{}.withDefaults()}, ctx: ctx, file: f, state: s, version: 1, reviews: reviewState{loaded: true}}
			ply := int(s.Ply)
			recordReviewChoice(t, &r.reviews, &r.state, key)
			ack := &wire.Ack{Seq: 17}
			mem := member{role: rolePlayer, seat: 0}
			var cache frameCache
			st := r.frame(mem, ack, nil, [2]bool{}, &cache, false)
			if st.Ack == nil || st.Ack.Seq != 17 || st.Version != 1 {
				t.Fatalf("not the move's acknowledged frame: %+v", st)
			}
			rv := st.View.Review
			if expired {
				if rv != nil {
					t.Fatalf("deadline-cut ranking produced a review: %+v", rv)
				}
			} else if rv == nil || rv.Ply != ply || rv.Played != key || rv.Strength != view.AdvisorStrengthQuick || len(rv.Entries) != 1 {
				t.Fatalf("the move's frame: review %+v, want completed quick review of ply %d", rv, ply)
			}
			slot := &r.reviews.seats[0]
			if !slot.done || slot.pending || r.reviewGen != 0 {
				t.Fatal("quick review left work for a later frame")
			}
			var later frameCache
			if got := r.frame(mem, nil, nil, [2]bool{}, &later, false).View.Review; got != rv {
				t.Fatal("later frame changed the completed or dropped quick review")
			}
		})
	}
}
