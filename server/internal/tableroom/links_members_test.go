package tableroom

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest/faults"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

func (h *th) userLink(id tabletest.ClientID, user string, reset bool) (string, string) {
	h.t.Helper()
	var out string
	h.on(func(a *actor) { out = a.linkUser(h.p.clients[id], user, reset, true) })
	h.p.drain()
	if out != outAccepted {
		return out, ""
	}
	messages := h.p.other[id]
	f, ok := messages[len(messages)-1].(wire.PlatformLinked)
	if !ok || f.UserID == nil || *f.UserID != user || f.Seat != -1 {
		h.t.Fatalf("wrong linked: %+v", messages)
	}
	return out, f.Claim
}

func TestRememberedHumanLinks(t *testing.T) {
	h := newTH(t, Config{BotWork: blockBots}, humans(3, 106))
	admin, player, spectator := tabletest.SeatClient(0), tabletest.SeatClient(1), tabletest.SpectatorClient
	if out, _ := h.userLink(spectator, "watcher", false); out != outBad {
		t.Fatal("plain spectator has a link")
	}
	if out, _ := h.userLink(player, userOf(0), false); out != outNotAdmin {
		t.Fatal("non-admin read another link")
	}
	h.want(admin, roleOp(1, "promote", "watcher"), outAccepted, "promote spectator")
	_, claim := h.userLink(spectator, "watcher", false)
	if err := h.joinClaim("watcher-phone", "phone", claim); err != nil {
		t.Fatal(err)
	}
	fr := h.latest("watcher-phone")
	if fr.You.UserID != "watcher" || fr.You.Seat != nil || !human(t, fr, "watcher").Admin {
		t.Fatalf("watcher's identity: %+v", fr.You)
	}
	_, copied := h.userLink(admin, "watcher", false)
	if claim != copied {
		t.Fatal("copy changed claim")
	}
	e := h.sameLogEverywhere(2)[1]
	if e.Kind != "link-copy" || e.By != userOf(0) || e.UserID == nil || *e.UserID != "watcher" {
		t.Fatalf("log: %+v", e)
	}
	for id, f := range h.p.frames {
		for _, frame := range f {
			raw, err := json.Marshal(frame)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(raw), claim) {
				t.Fatalf("claim leaked to %s's state", id)
			}
		}
	}
	// A user-targeted reset by the claimed device preserves only its own session.
	_, next := h.userLink("watcher-phone", "watcher", true)
	if next == claim || h.revoked("watcher-phone") {
		t.Fatal("reset lost requester")
	}
	if err := h.joinClaim("old-watcher-phone", "old-phone", claim); !errors.Is(err, room.ErrBadClaim) {
		t.Fatal("old claim still works")
	}
	_, final := h.userLink(admin, "watcher", true)
	if final == next || !h.revoked("watcher-phone") {
		t.Fatal("admin reset did not detach prior session")
	}
	// Epochs and roles survive reload, and the claim remains room-scoped.
	h.f.restart()
	if err := h.joinClaim("reloaded-watcher", "reloaded", final); err != nil {
		t.Fatal(err)
	}
	other, err := h.f.tables.CreateTable(humans(3, 107))
	if err != nil {
		t.Fatal(err)
	}
	if err := h.f.m.JoinClaim(h.ctx, newClient("phone", "wrong-room", "Phone"), other, &final); !errors.Is(err, room.ErrBadClaim) {
		t.Fatal("claim crossed rooms")
	}
}

func TestFormat5TableLinksLoad(t *testing.T) {
	h := newTH(t, Config{BotWork: blockBots}, humans(3, 109))
	r := h.record()
	r.Format, r.Links = 5, nil
	before := r.clone()
	data, err := r.encode()
	if err != nil {
		t.Fatal(err)
	}
	next, err := decodeRecord(data)
	if err != nil {
		t.Fatal(err)
	}
	if next.Format != 6 || !reflect.DeepEqual(next.Seats, before.Seats) || !reflect.DeepEqual(next.Names, before.Names) {
		t.Fatal("format 5 metadata changed")
	}
}

func TestLinkDisclosureRequiresDurableLog(t *testing.T) {
	h := newTH(t, Config{BotWork: blockBots}, humans(3, 110))
	admin := tabletest.SeatClient(0)
	before := h.record()
	h.f.store.inject(faults.FailWrite)
	if out, claim := h.userLink(admin, userOf(1), false); out != outStorage || claim != "" {
		t.Fatal("failed write disclosed a link")
	}
	after := h.record()
	if after.Public != before.Public || !reflect.DeepEqual(after.RoomLog, before.RoomLog) {
		t.Fatal("failed write changed public state")
	}
}
