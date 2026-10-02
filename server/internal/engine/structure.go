package engine

import "github.com/NightMachinery/ajayeb/server/internal/engine/catalog"

// This file handles the current age's structure in State.Slots: dealing an
// age, which slots are accessible, taking a card, and turning face up the
// face-down cards a turn has freed. The geometry itself is in layout.go.
//
// An empty slot is the zero Slot. An occupied slot keeps its card's back, so
// a redacted copy of a face-down slot still shows it. A face-down card is
// turned face up only at the end of the turn that freed it (finishTurn), so
// between turns every accessible card is face up.

// Occupied returns the slots of the structure that still hold a card, as a
// slot mask (bit i for slot i).
func Occupied(s *State) uint32 {
	var m uint32
	for i := range s.Slots {
		if s.Slots[i].Card != 0 {
			m |= 1 << i
		}
	}
	return m
}

// Accessible returns the slots whose card may be taken: occupied slots that
// no occupied slot covers. It is empty during the wonder draft.
func Accessible(s *State) uint32 {
	if s.Age < 1 || s.Age > NumAges {
		return 0
	}
	return AccessibleMask(s.Age, Occupied(s))
}

// SlotOf returns the index of the structure slot that holds card c, or -1.
func SlotOf(s *State, c catalog.CardID) int {
	if c == 0 {
		return -1
	}
	for i := range s.Slots {
		if s.Slots[i].Card == c {
			return i
		}
	}
	return -1
}

// takeSlot empties slot i and returns the card it held.
func takeSlot(s *State, i int) catalog.CardID {
	c := s.Slots[i].Card
	s.Slots[i] = Slot{}
	return c
}

// startAge deals the given age (1 to 3): its row of Deals moves into Slots,
// face up on the odd rows and face down on the even rows, each slot showing
// its card's back, and the row is zeroed. It sets Age and emits EvAgeStart.
// It does not touch the phase or the seats; the caller does.
func startAge(s *State, age uint8, ev *[]Event) {
	l := layoutOf(age)
	row := &s.Deals[age-1]
	for i := range SlotsPerAge {
		c := row[i]
		row[i] = 0
		if c == 0 {
			s.Slots[i] = Slot{}
			continue
		}
		s.Slots[i] = Slot{Card: c, FaceUp: l.faceUp&(1<<i) != 0, Back: cardBack(c)}
	}
	s.Age = age
	emit(ev, Event{Kind: EvAgeStart, A: int16(age)})
}

// revealAccessible turns face up every accessible card that is still face
// down, in slot order, emitting EvReveal for each (A = the slot, Card = the
// card).
func revealAccessible(s *State, ev *[]Event) {
	acc := Accessible(s)
	for i := range s.Slots {
		sl := &s.Slots[i]
		if acc&(1<<i) != 0 && !sl.FaceUp {
			sl.FaceUp = true
			emit(ev, Event{Kind: EvReveal, Card: sl.Card, A: int16(i)})
		}
	}
}

// cardBack returns the printed back of card c, or NoBack for an id that is
// not registered.
func cardBack(c catalog.CardID) catalog.Back {
	if int(c) < len(catalog.T.Back) {
		return catalog.T.Back[c]
	}
	return catalog.NoBack
}
