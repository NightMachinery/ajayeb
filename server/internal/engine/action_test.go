package engine

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
)

// Real items used by the action tests, so they pass with or without the
// full catalog data.
const (
	lumberYard     catalog.CardID = 1
	quarry         catalog.CardID = 5
	stable         catalog.CardID = 9
	garrison       catalog.CardID = 10
	baths          catalog.CardID = 23
	merchantsGuild catalog.CardID = 67
)

// useSampleCatalog swaps in a small catalog for the rest of the test.
func useSampleCatalog(t testing.TB) {
	t.Helper()
	cards := []catalog.CardDef{
		{ID: lumberYard, Slug: "lumber-yard", Name: "Lumber Yard", Age: 1, Color: catalog.Brown,
			Effects: catalog.Fx(catalog.Produce(catalog.Wood, 1))},
		{ID: quarry, Slug: "quarry", Name: "Quarry", Age: 1, Color: catalog.Brown,
			Effects: catalog.Fx(catalog.Produce(catalog.Stone, 1))},
		{ID: stable, Slug: "stable", Name: "Stable", Age: 1, Color: catalog.Red, Cost: catalog.MustCost("W"),
			ChainOut: catalog.ChainHorseshoe, Effects: catalog.Fx(catalog.Shields(1))},
		{ID: garrison, Slug: "garrison", Name: "Garrison", Age: 1, Color: catalog.Red, Cost: catalog.MustCost("C"),
			ChainOut: catalog.ChainSword, Effects: catalog.Fx(catalog.Shields(1))},
		{ID: baths, Slug: "baths", Name: "Baths", Age: 1, Color: catalog.Blue, Cost: catalog.MustCost("S"),
			ChainOut: catalog.ChainDrop, Effects: catalog.Fx(catalog.Points(3))},
		{ID: merchantsGuild, Slug: "merchants-guild", Name: "Merchants Guild", Age: 3, Color: catalog.Purple,
			Cost: catalog.MustCost("C W G P"),
			Effects: catalog.Fx(catalog.CoinsPerCard(catalog.ScopeMaxCity, 1, catalog.Yellow),
				catalog.PointsPerCard(catalog.ScopeMaxCity, 1, catalog.Yellow))},
	}
	wonders := []catalog.WonderDef{
		{ID: catalog.WonderPyramids, Slug: "pyramids", Name: "Pyramids", Cost: catalog.MustCost("SSS P"),
			Effects: catalog.Fx(catalog.Points(9))},
		{ID: catalog.WonderGreatLibrary, Slug: "great-library", Name: "Great Library", Cost: catalog.MustCost("WWW G P"),
			Effects: catalog.Fx(catalog.DrawTokens(3), catalog.Points(4))},
		{ID: catalog.WonderStatueOfZeus, Slug: "statue-of-zeus", Name: "Statue of Zeus", Cost: catalog.MustCost("PP C W S"),
			Effects: catalog.Fx(catalog.Destroy(catalog.Brown), catalog.Shields(1), catalog.Points(3))},
	}
	tokens := []catalog.TokenDef{
		{ID: catalog.TokenAgriculture, Slug: "agriculture", Name: "Agriculture",
			Effects: catalog.Fx(catalog.Coins(6), catalog.Points(4))},
		{ID: catalog.TokenLaw, Slug: "law", Name: "Law", Effects: catalog.Fx(catalog.Science(catalog.Scales))},
	}
	t.Cleanup(catalog.ReplaceForTest(cards, wonders, tokens))
}

var sampleKeys = []struct {
	a   Action
	key string
}{
	{DraftAction(catalog.WonderPyramids), "draft:pyramids"},
	{ConstructAction(baths), "construct:baths"},
	{DiscardAction(baths), "discard:baths"},
	{WonderAction(catalog.WonderPyramids, baths), "wonder:pyramids:baths"},
	{WonderAction(catalog.WonderGreatLibrary, merchantsGuild), "wonder:great-library:merchants-guild"},
	{DestroyAction(quarry), "destroy:quarry"},
	{ReviveAction(garrison), "revive:garrison"},
	{TokenAction(catalog.TokenLaw), "token:law"},
	{StarterAction(0), "starter:0"},
	{StarterAction(1), "starter:1"},
	{ResignAction(0), "resign:0"},
	{ResignAction(1), "resign:1"},
}

func TestActionKeys(t *testing.T) {
	useSampleCatalog(t)
	for _, c := range sampleKeys {
		if err := c.a.Validate(); err != nil {
			t.Errorf("%+v: Validate = %v", c.a, err)
		}
		if got := c.a.Key(); got != c.key {
			t.Errorf("%+v: Key = %q, want %q", c.a, got, c.key)
		}
		if got := c.a.String(); got != c.key {
			t.Errorf("%+v: String = %q, want %q", c.a, got, c.key)
		}
		b, err := c.a.MarshalText()
		if err != nil || string(b) != c.key {
			t.Errorf("%+v: MarshalText = %q, %v", c.a, b, err)
		}
		got, err := ParseKey(c.key)
		if err != nil || got != c.a {
			t.Errorf("ParseKey(%q) = %+v, %v; want %+v", c.key, got, err, c.a)
		}
		var u Action
		if err := u.UnmarshalText([]byte(c.key)); err != nil || u != c.a {
			t.Errorf("UnmarshalText(%q) = %+v, %v", c.key, u, err)
		}
		if c.a.Type.Verb() == "" || !strings.HasPrefix(c.key, c.a.Type.Verb()+":") || c.a.Type.String() != c.a.Type.Verb() {
			t.Errorf("%q: verb %q does not prefix the key", c.key, c.a.Type.Verb())
		}
	}
}

func TestActionJSON(t *testing.T) {
	useSampleCatalog(t)
	type logLine struct {
		Ply    int      `json:"ply"`
		Action Action   `json:"action"`
		Legal  []Action `json:"legal"`
	}
	in := logLine{Ply: 3, Action: WonderAction(catalog.WonderPyramids, baths),
		Legal: []Action{ConstructAction(baths), StarterAction(1)}}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	const want = `{"ply":3,"action":"wonder:pyramids:baths","legal":["construct:baths","starter:1"]}`
	if string(b) != want {
		t.Errorf("json = %s\nwant   %s", b, want)
	}
	var out logLine
	if err := json.Unmarshal(b, &out); err != nil || out.Action != in.Action || len(out.Legal) != 2 ||
		out.Legal[1] != in.Legal[1] {
		t.Errorf("round trip = %+v, %v", out, err)
	}
	for _, bad := range []string{`{"action":"construct:Baths"}`, `{"action":{"type":"construct"}}`, `{"action":3}`,
		`{"action":null,"legal":["starter:2"]}`} {
		if err := json.Unmarshal([]byte(bad), &out); err == nil {
			t.Errorf("json.Unmarshal(%s) succeeded", bad)
		}
	}
	if _, err := json.Marshal(logLine{}); err == nil {
		t.Error("json.Marshal of the zero action succeeded")
	}
}

func TestParseKeyRejects(t *testing.T) {
	useSampleCatalog(t)
	bad := []string{
		"", ":", "construct", "construct:", ":baths", "construct:baths:", "construct::baths",
		"Construct:baths", "CONSTRUCT:baths", "construct:Baths", "construct:BATHS",
		" construct:baths", "construct:baths ", "construct: baths", "construct :baths",
		"construct:baths\n", "construct:baths\x00", "construct:the-baths", "construct:lumber_yard",
		"construct:lumber yard", "construct:lumberyard", "build:baths", "play:baths", "take:baths",
		"construct:pyramids", "construct:law", "discard:pyramids", "destroy:law", "revive:great-library",
		"draft:baths", "draft:the-pyramids", "draft:Pyramids", "draft:law", "draft:", "draft:sphinx",
		"wonder:pyramids", "wonder:pyramids:", "wonder::baths", "wonder:baths:pyramids",
		"wonder:pyramids:baths:baths", "wonder:pyramids:law", "wonder:law:baths", "wonder:the-pyramids:baths",
		"token:", "token:baths", "token:pyramids", "token:Law", "token:economy", "token:law:law",
		"starter:", "starter:2", "starter:-1", "starter:01", "starter:+1", "starter:1.0", "starter: 1",
		"starter:0:0", "starter:one", "starter:\u0661",
		"c\u043enstruct:baths", // Cyrillic o
		"construct:baths" + strings.Repeat("x", 200),
		strings.Repeat("construct:baths", 100),
		"draft:pyramids\r", "token:law;", "token:law#", "starter:1 ",
	}
	for _, k := range bad {
		a, err := ParseKey(k)
		if err == nil {
			t.Errorf("ParseKey(%q) = %+v, want an error", k, a)
			continue
		}
		if !errors.Is(err, ErrBadKey) {
			t.Errorf("ParseKey(%q) error %v does not wrap ErrBadKey", k, err)
		}
		if len(err.Error()) > 300 {
			t.Errorf("ParseKey of a %d-byte key: the error is %d bytes long", len(k), len(err.Error()))
		}
		keep := StarterAction(1)
		if err := keep.UnmarshalText([]byte(k)); err == nil || keep != StarterAction(1) {
			t.Errorf("UnmarshalText(%q) changed the action or succeeded", k)
		}
	}
}

func TestActionValidateRejects(t *testing.T) {
	useSampleCatalog(t)
	bad := []Action{
		{},
		{Type: numActionTypes},
		{Type: 200, Card: baths},
		{Type: ActConstructBuilding},          // no card
		{Type: ActConstructBuilding, Card: 2}, // not registered
		{Type: ActConstructBuilding, Card: catalog.Unknown},                   // observation sentinel
		{Type: ActConstructBuilding, Card: baths, Wonder: 1},                  // extra field
		{Type: ActDiscardBuilding, Card: baths, Token: 4},                     // extra field
		{Type: ActDiscardBuilding, Card: baths, Starter: 1},                   // extra field
		{Type: ActConstructWonder, Wonder: catalog.WonderPyramids},            // no card
		{Type: ActConstructWonder, Card: baths},                               // no wonder
		{Type: ActConstructWonder, Wonder: catalog.WonderSphinx, Card: baths}, // wonder not registered
		{Type: ActDraftWonder},
		{Type: ActDraftWonder, Wonder: catalog.WonderPyramids, Card: baths},
		{Type: ActDestroyCard, Card: 0},
		{Type: ActBuildFromDiscard, Card: 74},
		{Type: ActChooseProgressToken},
		{Type: ActChooseProgressToken, Token: catalog.TokenEconomy}, // not registered
		{Type: ActChooseProgressToken, Token: catalog.TokenLaw, Card: baths},
		{Type: ActChooseStarter, Starter: 2},
		{Type: ActChooseStarter, Starter: 255},
		{Type: ActChooseStarter, Starter: 1, Token: catalog.TokenLaw},
	}
	for _, a := range bad {
		err := a.Validate()
		if err == nil {
			t.Errorf("Validate(%+v) = nil", a)
			continue
		}
		if !errors.Is(err, ErrMalformedAction) {
			t.Errorf("Validate(%+v) error %v does not wrap ErrMalformedAction", a, err)
		}
		if k := a.Key(); k != "" {
			t.Errorf("Key(%+v) = %q, want \"\"", a, k)
		}
		if _, err := a.MarshalText(); !errors.Is(err, ErrMalformedAction) {
			t.Errorf("MarshalText(%+v) error = %v", a, err)
		}
		if s := a.String(); !strings.HasPrefix(s, "Action{") {
			t.Errorf("String(%+v) = %q", a, s)
		}
	}
	if ActNone.String() != "none" || ActionType(99).String() != "action#99" || ActionType(99).Verb() != "" {
		t.Error("ActionType names of invalid types are wrong")
	}
}

// TestActionKeysFullCatalog checks every key of the real catalog: each one
// round-trips, fits the length limit and names a single action. It covers
// only the starter keys until the catalog data is registered.
func TestActionKeysFullCatalog(t *testing.T) {
	var all []Action
	for c := range catalog.Cards() {
		all = append(all, ConstructAction(c.ID), DiscardAction(c.ID), DestroyAction(c.ID), ReviveAction(c.ID))
		for w := range catalog.Wonders() {
			all = append(all, WonderAction(w.ID, c.ID))
		}
	}
	for w := range catalog.Wonders() {
		all = append(all, DraftAction(w.ID))
	}
	for tk := range catalog.Tokens() {
		all = append(all, TokenAction(tk.ID))
	}
	all = append(all, StarterAction(0), StarterAction(1))
	seen := make(map[string]Action, len(all))
	for _, a := range all {
		k := a.Key()
		if k == "" || len(k) > maxKeyLen {
			t.Errorf("%+v: key %q is empty or longer than %d", a, k, maxKeyLen)
			continue
		}
		if prev, dup := seen[k]; dup {
			t.Errorf("key %q names both %+v and %+v", k, prev, a)
		}
		seen[k] = a
		if back, err := ParseKey(k); err != nil || back != a {
			t.Errorf("ParseKey(%q) = %+v, %v; want %+v", k, back, err, a)
		}
	}
	t.Logf("checked %d keys", len(all))
}

// FuzzParseKey checks that ParseKey never panics and is canonical: whatever it
// accepts is exactly the key of the action it returns.
func FuzzParseKey(f *testing.F) {
	useSampleCatalog(f)
	for _, c := range sampleKeys {
		f.Add(c.key)
	}
	for _, s := range []string{"", ":", "wonder:pyramids:", "starter:01", "construct:Baths", "token:law:law"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		a, err := ParseKey(s)
		if err != nil {
			if !errors.Is(err, ErrBadKey) {
				t.Fatalf("ParseKey(%q): error %v does not wrap ErrBadKey", s, err)
			}
			return
		}
		if err := a.Validate(); err != nil {
			t.Fatalf("ParseKey(%q) returned a malformed action: %v", s, err)
		}
		if k := a.Key(); k != s {
			t.Fatalf("ParseKey(%q) is not canonical: its action's key is %q", s, k)
		}
	})
}

func TestIllegalError(t *testing.T) {
	useSampleCatalog(t)
	s := State{Phase: PhasePlay}
	var err error = illegal(&s, ConstructAction(baths), ReasonUnaffordable)
	const want = "engine: illegal action construct:baths in phase play: unaffordable"
	if err.Error() != want {
		t.Errorf("Error = %q, want %q", err.Error(), want)
	}
	var ie *IllegalError
	if !errors.As(err, &ie) || ie.Reason != ReasonUnaffordable || ie.Phase != PhasePlay || ie.Action != ConstructAction(baths) {
		t.Errorf("errors.As = %+v", ie)
	}
}
