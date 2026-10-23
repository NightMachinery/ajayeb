package catalog

// The unconfirmed list: every catalog field that no clear photo of the
// printed component or publisher's document has confirmed. Each entry names
// its open question in docs/haftgane/data-rulings.md ("The night-side
// questions"), the value the catalog holds provisionally, and the date the
// owner accepted that value, if they did. A clear photo that confirms a
// value removes its entry; one that contradicts it changes the data and the
// entry together.

import "testing"

type unconfirmedField struct {
	item, field, value string
	question           string
	ownerAccepted      string // the date of the owner's acceptance, or ""
}

var unconfirmed = []unconfirmedField{
	{"babylon", "night_stages", "2", "Q1", "2026-10-22"},
	{"babylon/night/1", "cost", "stone:2", "Q1", "2026-10-22"},
	{"babylon/night/1", "effects", "other:play-last-card-of-age", "Q1", "2026-10-22"},
	{"babylon/night/2", "cost", "clay:3 glass:1", "Q1", "2026-10-22"},
	{"babylon/night/2", "effects", "science:any", "Q1", "2026-10-22"},
}

// TestUnconfirmedList checks that each entry names a real field holding the
// listed provisional value, so the data cannot change without the list.
func TestUnconfirmedList(t *testing.T) {
	rows := editionRows(t)
	for _, u := range unconfirmed {
		row, ok := rows[u.item]
		if !ok {
			t.Errorf("%s: no such item", u.item)
			continue
		}
		if got := row[u.field]; got != u.value {
			t.Errorf("%s %s = %q, the list says %q (%s)", u.item, u.field, got, u.value, u.question)
		}
		t.Logf("unconfirmed: %s %s = %q (%s, owner-accepted %s)", u.item, u.field, u.value, u.question, u.ownerAccepted)
	}
	if _, ok := rows["babylon/night/3"]; ok {
		t.Error("Babylon night has a third stage that the list does not cover")
	}
}

// TestReleaseGate fails while any unconfirmed field is neither confirmed nor
// accepted by the owner. A release stays blocked until it passes.
func TestReleaseGate(t *testing.T) {
	for _, u := range unconfirmed {
		if u.ownerAccepted == "" {
			t.Errorf("%s %s (%s) awaits a photo or the owner's acceptance", u.item, u.field, u.question)
		}
	}
}
