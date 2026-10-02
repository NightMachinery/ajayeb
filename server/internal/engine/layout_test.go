package engine

import (
	"fmt"
	"math/bits"
	"strings"
	"testing"
)

// slotAt returns the slot at (row, col) of an age, or -1.
func slotAt(age uint8, row, col int8) int {
	for i := range SlotsPerAge {
		if SlotPos(age, i) == (Pos{Row: row, Col: col}) {
			return i
		}
	}
	return -1
}

func mustSlot(t *testing.T, age uint8, row, col int8) int {
	t.Helper()
	i := slotAt(age, row, col)
	if i < 0 {
		t.Fatalf("age %d has no slot at (%d,%d)", age, row, col)
	}
	return i
}

func TestLayoutRows(t *testing.T) {
	want := [NumAges]string{
		"[5,7] [4,6,8] [3,5,7,9] [2,4,6,8,10] [1,3,5,7,9,11]",
		"[1,3,5,7,9,11] [2,4,6,8,10] [3,5,7,9] [4,6,8] [5,7]",
		"[5,7] [4,6,8] [3,5,7,9] [4,8] [3,5,7,9] [4,6,8] [5,7]",
	}
	for age := uint8(1); age <= NumAges; age++ {
		var rows []string
		var row []string
		prev := Pos{Row: 1, Col: 0}
		for i := range SlotsPerAge {
			p := SlotPos(age, i)
			switch {
			case p.Row == prev.Row && p.Col > prev.Col:
			case p.Row == prev.Row+1:
				rows = append(rows, "["+strings.Join(row, ",")+"]")
				row = nil
			default:
				t.Fatalf("age %d: slot %d at %v is out of layout order after %v", age, i, p, prev)
			}
			row = append(row, fmt.Sprint(p.Col))
			prev = p
		}
		rows = append(rows, "["+strings.Join(row, ",")+"]")
		if got := strings.Join(rows, " "); got != want[age-1] {
			t.Errorf("age %d rows = %s\nwant           %s", age, got, want[age-1])
		}
		if Rows(age) != len(rows) {
			t.Errorf("age %d: Rows = %d, want %d", age, Rows(age), len(rows))
		}
	}
}

func TestLayoutCounts(t *testing.T) {
	wantAccessible := [NumAges]int{6, 2, 2}
	for age := uint8(1); age <= NumAges; age++ {
		up := FaceUpMask(age)
		if n := bits.OnesCount32(up); n != 12 {
			t.Errorf("age %d: %d face-up slots, want 12", age, n)
		}
		if n := bits.OnesCount32(allSlots &^ up); n != 8 {
			t.Errorf("age %d: %d face-down slots, want 8", age, n)
		}
		acc := AccessibleMask(age, allSlots)
		if n := bits.OnesCount32(acc); n != wantAccessible[age-1] {
			t.Errorf("age %d: %d accessible at the start, want %d", age, n, wantAccessible[age-1])
		}
		// The accessible cards at the start are the bottom row, dealt face up.
		for i := range SlotsPerAge {
			bottom := int(SlotPos(age, i).Row) == Rows(age)
			if (acc&(1<<i) != 0) != bottom {
				t.Errorf("age %d slot %d %v: accessible %v, bottom row %v", age, i, SlotPos(age, i), acc&(1<<i) != 0, bottom)
			}
			if bottom && up&(1<<i) == 0 {
				t.Errorf("age %d slot %d: a bottom-row card is dealt face down", age, i)
			}
			// Face-up parity: odd rows up, even rows down.
			if odd := SlotPos(age, i).Row%2 == 1; odd != (up&(1<<i) != 0) {
				t.Errorf("age %d slot %d: face up %v in row %d", age, i, !odd, SlotPos(age, i).Row)
			}
		}
	}
}

// TestCoverersFromPositions recomputes the cover tables by brute force.
func TestCoverersFromPositions(t *testing.T) {
	for age := uint8(1); age <= NumAges; age++ {
		for i := range SlotsPerAge {
			p := SlotPos(age, i)
			want := [2]int8{
				int8(slotAt(age, p.Row+1, p.Col-1)),
				int8(slotAt(age, p.Row+1, p.Col+1)),
			}
			if got := Coverers(age, i); got != want {
				t.Errorf("age %d slot %d %v: Coverers = %v, want %v", age, i, p, got, want)
			}
			var mask uint32
			for _, j := range want {
				if j >= 0 {
					mask |= 1 << j
				}
			}
			if got := CoverMask(age, i); got != mask {
				t.Errorf("age %d slot %d: CoverMask = %b, want %b", age, i, got, mask)
			}
		}
	}
}

func TestAgeIIIGap(t *testing.T) {
	const age = 3
	c44, c48 := int8(mustSlot(t, age, 4, 4)), int8(mustSlot(t, age, 4, 8))
	if slotAt(age, 4, 6) >= 0 {
		t.Fatal("age III has a card at (4,6)")
	}
	cases := []struct {
		row, col int8
		want     [2]int8
	}{
		{3, 3, [2]int8{-1, c44}},
		{3, 5, [2]int8{c44, -1}},
		{3, 7, [2]int8{-1, c48}},
		{3, 9, [2]int8{c48, -1}},
	}
	for _, c := range cases {
		i := mustSlot(t, age, c.row, c.col)
		if got := Coverers(age, i); got != c.want {
			t.Errorf("(%d,%d): Coverers = %v, want %v", c.row, c.col, got, c.want)
		}
	}
	// Taking (5,3) and (5,5) frees (4,4), which frees (3,3) and (3,5) at once.
	occ := allSlots
	for _, i := range []int{mustSlot(t, age, 7, 5), mustSlot(t, age, 7, 7), mustSlot(t, age, 6, 4),
		mustSlot(t, age, 6, 6), mustSlot(t, age, 6, 8), mustSlot(t, age, 5, 3), mustSlot(t, age, 5, 5)} {
		occ &^= 1 << i
	}
	acc := AccessibleMask(age, occ)
	if acc&(1<<c44) == 0 {
		t.Error("(4,4) is not accessible once (5,3) and (5,5) are taken")
	}
	occ &^= 1 << c44
	acc = AccessibleMask(age, occ)
	for _, col := range []int8{3, 5} {
		if acc&(1<<mustSlot(t, age, 3, col)) == 0 {
			t.Errorf("(3,%d) is not accessible once (4,4) is taken", col)
		}
	}
	for _, col := range []int8{7, 9} {
		if acc&(1<<mustSlot(t, age, 3, col)) != 0 {
			t.Errorf("(3,%d) is accessible while (4,8) is still there", col)
		}
	}
}

func TestRevealSequences(t *testing.T) {
	// Age I: (4,2) is uncovered only once both (5,1) and (5,3) are taken.
	occ := allSlots
	s42, s51, s53 := mustSlot(t, 1, 4, 2), mustSlot(t, 1, 5, 1), mustSlot(t, 1, 5, 3)
	occ &^= 1 << s51
	if AccessibleMask(1, occ)&(1<<s42) != 0 {
		t.Error("age I: (4,2) is accessible after only (5,1) is taken")
	}
	occ &^= 1 << s53
	if AccessibleMask(1, occ)&(1<<s42) == 0 {
		t.Error("age I: (4,2) is not accessible after (5,1) and (5,3) are taken")
	}
	// Age II: taking (5,5) frees (4,4), which has no card at (5,3), but not
	// (4,6), still under (5,7).
	occ = allSlots &^ (1 << mustSlot(t, 2, 5, 5))
	acc := AccessibleMask(2, occ)
	if acc&(1<<mustSlot(t, 2, 4, 4)) == 0 {
		t.Error("age II: (4,4) is not accessible after (5,5) is taken")
	}
	if acc&(1<<mustSlot(t, 2, 4, 6)) != 0 {
		t.Error("age II: (4,6) is accessible while (5,7) is still there")
	}
}

// TestEveryStructureEmpties takes cards in several orders until each
// structure is empty: there is always an accessible card while one is left.
func TestEveryStructureEmpties(t *testing.T) {
	for age := uint8(1); age <= NumAges; age++ {
		for order := range 3 {
			occ := allSlots
			for taken := 0; occ != 0; taken++ {
				acc := AccessibleMask(age, occ)
				if acc == 0 {
					t.Fatalf("age %d: no accessible card with %d left", age, bits.OnesCount32(occ))
				}
				if acc&^occ != 0 {
					t.Fatalf("age %d: an empty slot is accessible", age)
				}
				var i int
				switch order {
				case 0:
					i = bits.TrailingZeros32(acc)
				case 1:
					i = 31 - bits.LeadingZeros32(acc)
				default: // the k-th accessible slot, k varying with the turn
					k := (taken * 7) % bits.OnesCount32(acc)
					rest := acc
					for range k {
						rest &= rest - 1 // drop the lowest set bit
					}
					i = bits.TrailingZeros32(rest)
				}
				occ &^= 1 << i
			}
		}
	}
	// Bits above the 20 slots are ignored.
	if AccessibleMask(1, ^uint32(0)) != AccessibleMask(1, allSlots) {
		t.Error("AccessibleMask looks at bits above slot 19")
	}
}

func TestLayoutRejectsBadAge(t *testing.T) {
	for _, age := range []uint8{0, 4, 255} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("SlotPos(%d, 0) did not panic", age)
				}
			}()
			SlotPos(age, 0)
		}()
	}
}
