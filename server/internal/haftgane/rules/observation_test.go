package rules

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
)

// Engine design section 11: the bot receives a whitelist with no setup root,
// hidden places or derived caches, and its ordinary quotes use public facts.
func TestObservedWhitelistAndQuotes(t *testing.T) {
	g, err := NewGame(Setup{Seats: 7, Seed: 41, Options: Options{Sides: SidesDay}})
	if err != nil {
		t.Fatal(err)
	}
	for seat := Seat(0); seat < 7; seat++ {
		o := g.Observe(seat)
		if o.Now.Hand != g.State.Hands[seat] || o.Now.Seat != seat || o.Now.Seats != 7 {
			t.Fatal("own current cards missing")
		}
		for id := catalog.DesignID(1); id <= catalog.NumDesigns; id++ {
			d, _ := catalog.DesignByID(id)
			for s := Seat(0); s < 7; s++ {
				if o.Quote(s, d.Cost) != QuoteState(&g.State, s, d.Cost) {
					t.Fatal("observation quote differs", seat, s, id)
				}
			}
		}
	}
	spectator := g.Observe(NoSeat)
	if spectator.Now.Seat != NoSeat || spectator.Now.Hand != ([HandSize]catalog.DesignID{}) || spectator.Now.Offer != ([catalog.NumDesigns + 1]uint8{}) || spectator.Know != (Knowledge{}) {
		t.Fatal("spectator got private facts")
	}
	for _, typ := range []reflect.Type{reflect.TypeFor[Current](), reflect.TypeFor[PublicCity](), reflect.TypeFor[Knowledge](), reflect.TypeFor[Observation]()} {
		for _, f := range reflect.VisibleFields(typ) {
			switch f.Name {
			case "Seed", "Record", "Deals", "BoxGuilds", "Produce", "ForSale", "Choice", "Discount", "Chains", "Abilities":
				t.Fatal("non-whitelisted observation field", typ.Name(), f.Name)
			}
		}
	}
}
func TestObservedHiddenSentinels(t *testing.T) {
	g, err := NewGame(Setup{Seats: 7, Seed: 41, Options: Options{Sides: SidesDay}})
	if err != nil {
		t.Fatal(err)
	}
	before := g.Observe(0)
	// Intentionally invalid private sentinels prove that Project copies no raw
	// private object. Reachable paired histories are tested separately.
	g.State.Seed = ^g.State.Seed
	g.State.Hands[2][0] = 80
	g.State.Cities[3].Tucked[0] = 79
	g.State.Deals[2][0][0] = 78
	g.State.BoxGuilds[0] = 77
	after := g.Observe(0)
	if before != after || !bytes.Equal(before.AppendEncoding(nil), after.AppendEncoding(nil)) || before.Fingerprint() != after.Fingerprint() {
		t.Fatal("hidden sentinel changed permitted observation")
	}
}
func TestObservedEncoding(t *testing.T) {
	g, err := NewGame(Setup{Seats: 3, Seed: 29, Options: Options{Sides: SidesDay}})
	if err != nil {
		t.Fatal(err)
	}
	o := g.Observe(0)
	var independent bytes.Buffer
	if err := binary.Write(&independent, binary.LittleEndian, o); err != nil {
		t.Fatal(err)
	}
	encoded := o.AppendEncoding(nil)
	if len(encoded) != 1693 || !bytes.Equal(encoded, independent.Bytes()) {
		t.Fatal("canonical observation encoding differs", len(encoded), independent.Len())
	}
	prefix := []byte{255, 19}
	with := o.AppendEncoding(append([]byte{}, prefix...))
	if !bytes.Equal(with[:2], prefix) || !bytes.Equal(with[2:], encoded) {
		t.Fatal("encoding append contract")
	}
	changed := o
	changed.Now.Hand[0] = 80
	if o.Fingerprint() == changed.Fingerprint() {
		t.Fatal("own observation change not fingerprinted")
	}
}
func TestObservedKnowledgeOwnTurn(t *testing.T) {
	g, err := NewGame(Setup{Seats: 7, Seed: 41, Options: Options{Sides: SidesDay}})
	if err != nil {
		t.Fatal(err)
	}
	before := g
	next, _, err := g.Resolve(salesBatch(g))
	if err != nil {
		t.Fatal(err)
	}
	for seat := Seat(0); seat < 7; seat++ {
		o := next.Observe(seat)
		if o.Know.Seen[0][0] != before.State.Hands[seat] || o.Know.Picks[0][0] != before.State.Hands[seat][0] {
			t.Fatal("own held hand or pick missing")
		}
		for s := Seat(0); s < 7; s++ {
			p := o.Know.Plays[0][0][s]
			if p.Kind != ActSell || p.Design != 0 {
				t.Fatal("private sale became public knowledge")
			}
		}
	}
}

// Section 10: privacy is selected by kind/action, never by the raw assertion.
func TestFilteredEventWhitelist(t *testing.T) {
	raw := []Event{{Kind: EvPlay, Seat: 1, Action: ActStage, Design: 17, Stage: 1, Left: 2, Private: false, Other: 6, Delta: 123, Side: catalog.Night}, {Kind: EvCoins, Seat: 1, Source: CoinSale, Design: 18, Delta: 3, Private: false}, {Kind: EventKind(255), Seat: 1, Design: 80}}
	public := FilterEvents(raw, 0)
	owner := FilterEvents(raw, 1)
	if len(public) != 2 || len(owner) != 2 || public[0].Design != "" || owner[0].Design != "altar" || public[1].Design != "" {
		t.Fatal("private event identity leaked or unknown event emitted", public, owner)
	}
	if public[0].Other != 0 || public[0].Delta != 0 || public[0].Side != "" || public[0].Stage != 1 || public[0].Left != 2 {
		t.Fatal("unrelated raw fields copied into event", public[0])
	}
	if len(FilterEvents(nil, 0)) != 0 || FilterEvents(nil, 0) == nil {
		t.Fatal("empty event list must be nonnil")
	}
}
