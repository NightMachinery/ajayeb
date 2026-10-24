package tableroom

import (
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/roles"
	"github.com/NightMachinery/ajayeb/server/internal/store"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// worstClient is a client id the wire accepts that encodes to the most
// bytes: 128 characters, each escaped to six by encoding/json.
func worstClient(i int) string { return strings.Repeat("<", 120) + fmt.Sprintf("%08d", i) }

// hexUser is a user id of a real id's length (ident: 32 hex characters).
func hexUser(kind, i int) string { return fmt.Sprintf("%02x%030x", kind, i) }

// TestClientKey: a short id of the web client's alphabet is remembered as
// it is, anything else as a digest of fixed length that no plain id can
// equal.
func TestClientKey(t *testing.T) {
	for _, id := range []string{"0b5e3a7c-1f2d-4e8a-9b6c-5d4e3f2a1b0c", "AbCdEfGhIjKlMnOpQrStUv", strings.Repeat("a", maxPlainClient)} {
		if got := ClientKey(id); got != id {
			t.Errorf("ClientKey(%q) = %q, want it unchanged", id, got)
		}
	}
	seen := map[string]bool{}
	for _, id := range []string{worstClient(0), worstClient(1), strings.Repeat("a", maxPlainClient+1), "tab one", "تب", "a.b"} {
		k := ClientKey(id)
		if k == id || len(k) > maxClientKey || !strings.HasPrefix(k, "~") || seen[k] {
			t.Errorf("ClientKey(%q) = %q", id, k)
		}
		seen[k] = true
		if ClientKey(id) != k {
			t.Errorf("ClientKey(%q) is not stable", id)
		}
	}
}

// TestLongClientIDsAreRememberedByDigest: the table remembers a client
// with a long id by its digest, and that client's retry, or an older
// seq, is still a duplicate.
func TestLongClientIDsAreRememberedByDigest(t *testing.T) {
	h := newTH(t, Config{BotWork: blockBots}, humans(3, 211))
	long := tabletest.ClientID(worstClient(7))
	if err := h.p.join(h.ctx, long, userOf(1), "P1"); err != nil {
		t.Fatal(err)
	}
	sn := h.snap(long)
	h.expect(h.send(long, 5, sn.Ref, sn.Legal[0], false), tabletest.Accepted, "the long-id tab's pick")
	h.expect(h.send(long, 5, sn.Ref, sn.Legal[1], false), tabletest.Duplicate, "its retry")
	h.expect(h.send(long, 4, sn.Ref, sn.Legal[1], false), tabletest.Duplicate, "an older seq (acknowledged as remembered)")
	r := h.record()
	for _, x := range r.Seqs {
		if x.Client == string(long) {
			t.Fatal("the record holds the long id itself")
		}
	}
	if !seqHas(r.Seqs, userOf(1), ClientKey(string(long))) || !seqHas(pendingOf(&r, 1).Seqs, userOf(1), ClientKey(string(long))) {
		t.Fatalf("the digest is not remembered: %+v / %+v", r.Seqs, r.Pending)
	}
}

func seqHas(xs []SeqRecord, user, client string) bool {
	for _, x := range xs {
		if x.User == user && x.Client == client {
			return true
		}
	}
	return false
}

// TestLargestRecordFitsTheStore builds the largest record the bounds allow
// (every list full, every id and name at the length that encodes to the
// most bytes, every seq the largest) and checks that restore accepts it
// and that it fits under the store's read cap, the 25 largest possible
// batches and a work row for every seat added. Before, its seq
// acknowledgements and floors alone took several times the cap, so an
// admin with puppet users could grow a table until the store could no
// longer load it (M16.haftgane-security-review).
func TestLargestRecordFitsTheStore(t *testing.T) {
	seed := uint64(212)
	// The creator holds no seat, so every admin is a user of their own.
	sp := Spec{Seed: &seed, Creator: hexUser(2, 0)}
	for s := range MaxSeats {
		sp.Participants = append(sp.Participants, Participant{UserID: hexUser(1, s), Name: strings.Repeat("<", maxNameBytes-1) + "x"})
	}
	h := newTH(t, Config{BotWork: blockBots}, sp)
	a, err := h.p.actor()
	if err != nil {
		t.Fatal(err)
	}
	if err := a.call(h.ctx, func() {
		g := a.st.game
		p := g.Point()
		r := a.st.rec.clone()
		since := time.Date(2026, 10, 24, 12, 0, 0, 123456789, time.UTC)
		creator := r.Roles.Creator
		for i := 1; len(r.Roles.Admins) < roles.MaxAdmins; i++ {
			by := creator
			r.Roles.Admins = append(r.Roles.Admins, roles.Admin{UserID: hexUser(2, i), PromotedBy: &by})
		}
		r.Roles.Temporary = hexUser(3, 0)
		r.Roles.Away = nil
		for _, ad := range r.Roles.Admins {
			r.Roles.Away = append(r.Roles.Away, roles.Away{UserID: ad.UserID, Since: since})
		}
		r.Roles.Away = append(r.Roles.Away, roles.Away{UserID: r.Roles.Temporary, Since: since})
		for i := range maxRetired {
			r.Retired = append(r.Retired, hexUser(4, i))
		}
		for i := range maxPrevious {
			r.Previous = append(r.Previous, PreviousRecord{UserID: hexUser(5, i), Seat: MaxSeats - 1})
		}
		for range maxRoomLog {
			u, s, ctl, role := hexUser(6, 0), MaxSeats-1, SeatHuman, "temporary"
			r.RoomLog = append(r.RoomLog, wire.PlatformLogEntry{Kind: "resign", At: maxBatches, By: hexUser(6, 1), UserID: &u, Seat: &s, Control: &ctl, Role: &role})
		}
		for range maxRoomLog {
			r.BotFallbacks = append(r.BotFallbacks, BotFallback{At: len(r.Batches), Point: p.ID, Seat: MaxSeats - 1, Reason: "watchdog"})
		}
		users := map[string]bool{}
		for _, u := range append(authorities(&r), r.Retired...) {
			users[u] = true
		}
		if len(users) != maxSeqUsers {
			t.Fatalf("%d users with authority or retired, want maxSeqUsers %d", len(users), maxSeqUsers)
		}
		n := 0
		for u := range users {
			for range DefaultSeqClients {
				r.Seqs = append(r.Seqs, SeqRecord{User: u, Client: ClientKey(worstClient(n)), Seq: math.MaxInt64})
				n++
			}
		}
		r.Pending = nil
		for _, s := range p.Required.Seats() {
			slot := PendingSlot{Seat: int(s), Key: legalKeys(&g, int(s))[0]}
			for u := range maxSlotUsers {
				for range DefaultPickClients {
					slot.Seqs = append(slot.Seqs, SeqRecord{User: hexUser(7, u), Client: ClientKey(worstClient(n)), Seq: math.MaxInt64})
					n++
				}
			}
			r.Pending = append(r.Pending, slot)
		}
		if err := validate(&r, &g, defaultBounds); err != nil {
			t.Fatalf("the largest record is not one restore accepts: %v", err)
		}
		// The batches and work rows cannot coexist with this open point;
		// add their largest encoding on top.
		for b := range maxBatches {
			br := BatchRecord{Point: strings.Repeat("p", maxPointID) + fmt.Sprint(b)}
			for s := range MaxSeats {
				br.Picks = append(br.Picks, PickRecord{Seat: s, Key: strings.Repeat("k", maxStoredKey)})
			}
			r.Batches = append(r.Batches, br)
		}
		for s := range MaxSeats {
			r.Work = append(r.Work, WorkRow{Seat: s, Point: strings.Repeat("p", maxPointID), BotPreset: r.BotPresets[len(r.BotPresets)-1]})
		}
		data, err := r.encode()
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("the largest table record: %d bytes, %.2f of the store's cap of %d", len(data), float64(len(data))/store.MaxFileBytes, store.MaxFileBytes)
		if len(data) > store.MaxFileBytes*3/4 {
			t.Errorf("the largest table record takes %d bytes, more than three quarters of the store's cap of %d", len(data), store.MaxFileBytes)
		}
	}); err != nil {
		t.Fatal(err)
	}
}

// The wire's longest strings that a record copies: an action key and a
// point id (wire.maxActionKeyLen, both).
const (
	maxStoredKey = 96
	maxPointID   = 96
)
