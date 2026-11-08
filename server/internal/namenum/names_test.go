package namenum

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestLifetimeNumbers(t *testing.T) {
	var l Ledger
	l.Assign("a", " River ")
	l.Assign("b", "river")
	l.Assign("c", "RIVER")
	if l.Get("a").Number != 0 || l.Get("b").Number != 2 || l.Get("c").Number != 3 {
		t.Fatal(l)
	}
	copy := l.Clone()
	copy.Assign("b", "Brook")
	if l.Get("b").Name != "river" {
		t.Fatal("clone shared storage")
	}
	l.Assign("a", "Brook")
	l.Assign("d", "River")
	l.Assign("a", "River")
	if l.Get("b").Number != 2 || l.Get("c").Number != 3 || l.Get("d").Number != 4 || l.Get("a").Number != 5 {
		t.Fatal(l)
	}
	l.Assign("b", " RIVER ")
	if l.Get("b").Number != 2 {
		t.Fatal("cosmetic rename changed suffix")
	}
	data, err := json.Marshal(l)
	if err != nil {
		t.Fatal(err)
	}
	var restored Ledger
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	if err := restored.Valid(); err != nil {
		t.Fatal(err)
	}
	if restored.Assign("c", "RIVER") {
		t.Fatal("reconnect allocated again")
	}
	restored.Assign("e", "river")
	if restored.Get("e").Number != 6 {
		t.Fatal(restored)
	}
}

func TestRejectInvalidLedger(t *testing.T) {
	var l Ledger
	l.Assign("a", "River")
	for _, corrupt := range []func(*Ledger){
		func(l *Ledger) { l.Users = append(l.Users, l.Users[0]) },
		func(l *Ledger) { l.Users[0].Number = 1 },
		func(l *Ledger) { l.Users[0].Number = 2 },
		func(l *Ledger) { l.Names[0].Next = 1 },
		func(l *Ledger) { l.Names[0].Key = "River" },
		func(l *Ledger) { l.Names = nil },
	} {
		c := l.Clone()
		corrupt(&c)
		if c.Valid() == nil {
			t.Fatal("invalid ledger accepted", c)
		}
	}
}

func TestBoundedNameHistory(t *testing.T) {
	var l Ledger
	l.AssignBounded("a", "River", 4)
	visitor := l.Preview("watch", "Brook", nil)
	l.AssignBounded("a", "Brook", 4, visitor)
	if l.Get("a").Number != 2 {
		t.Fatal("rename collided with visitor")
	}
	l.AssignBounded("a", "Third", 4)
	l.AssignBounded("a", "Fourth", 4)
	held := l.Get("a")
	l.Import(Assignment{UserID: "holder", Name: "Fourth", Number: 2}, 4)
	visitor = l.Preview("watch", "Fifth", nil)
	if !l.AssignBounded("a", "Fifth", 4, visitor) || l.Get("a").Name != "Fifth" || l.Get("a").Number != 2 {
		t.Fatal("full ledger blocked rename or collided with visitor")
	}
	if l.Get("holder").Number != 2 || l.Get("holder").Name != held.Name || l.Names[0].Key != counterKey("Brook") {
		t.Fatal("rename changed another holder or did not evict the oldest unused key", l)
	}
	l.AssignBounded("b", "New authority", 4)
	if len(l.Names) != 4 || l.Get("b").Name != "New authority" {
		t.Fatal("full history blocked new authority")
	}
	if l.Valid() != nil {
		t.Fatal("invalid bounded ledger", l)
	}
}

func TestExhaustedOrdinalStillAdmitsAuthority(t *testing.T) {
	var l Ledger
	l.AssignBounded("a", "River", 4)
	l.Names[0].Next = maxOrdinal
	l.AssignBounded("b", "River", 4)
	if l.Get("b").Number != 2 || l.Valid() != nil {
		t.Fatal("exhausted ordinal blocked admission", l)
	}
}

func TestNameRule(t *testing.T) {
	for _, name := range []string{strings.Repeat("م", 40), strings.Repeat("😀", 17), strings.Repeat("😀", MaxNameRunes)} {
		if !ValidName(name) || CleanName(" "+name+" ") != name {
			t.Fatal("valid Unicode name refused", name)
		}
		var l Ledger
		l.Assign("a", " "+name+" ")
		if err := l.Valid(); err != nil || l.Get("a").Name != name {
			t.Fatal("ledger disagrees with name rule", err)
		}
	}
	for _, name := range []string{"", " ", " River ", strings.Repeat("😀", MaxNameRunes+1), string([]byte{0xff})} {
		if ValidName(name) {
			t.Fatal("invalid name accepted", name)
		}
		var l Ledger
		l.Assign("a", "River")
		l.Users[0].Name = name
		if l.Valid() == nil {
			t.Fatal("ledger accepted invalid name", name)
		}
	}
}
