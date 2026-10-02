package engine

import "strconv"

// The three age structures. Rows are numbered from 1 at the top; columns are
// counted in half-card widths from 1 at the left, so a card at (r, c) is
// partly covered by the cards at (r+1, c-1) and (r+1, c+1). Slots are
// numbered in layout order: row by row from the top, left to right within a
// row. State.Slots and State.Deals use this order.
//
//	Age I    rows 1-5  [5,7] [4,6,8] [3,5,7,9] [2,4,6,8,10] [1,3,5,7,9,11]
//	Age II   rows 1-5  [1,3,5,7,9,11] [2,4,6,8,10] [3,5,7,9] [4,6,8] [5,7]
//	Age III  rows 1-7  [5,7] [4,6,8] [3,5,7,9] [4,8] [3,5,7,9] [4,6,8] [5,7]
//
// Odd rows are dealt face up and even rows face down, which gives every age
// 12 face-up and 8 face-down cards. A card is accessible when no card is left
// in either covering place; at the start 6, 2 and 2 cards are accessible in
// Ages I, II and III. Row 4 of Age III holds only (4,4) and (4,8), so every
// card of row 3 is covered by one card: (3,3) and (3,5) by (4,4), (3,7) and
// (3,9) by (4,8).
//
// Slot masks are uint32 bitsets with bit i for slot i.

// Pos is a slot's place in its structure.
type Pos struct {
	Row int8 // 1 at the top
	Col int8 // half-card widths, 1 at the left
}

// layoutRows lists, for each age and each row from the top, the columns of
// that row's cards.
var layoutRows = [NumAges][][]int8{
	{{5, 7}, {4, 6, 8}, {3, 5, 7, 9}, {2, 4, 6, 8, 10}, {1, 3, 5, 7, 9, 11}},
	{{1, 3, 5, 7, 9, 11}, {2, 4, 6, 8, 10}, {3, 5, 7, 9}, {4, 6, 8}, {5, 7}},
	{{5, 7}, {4, 6, 8}, {3, 5, 7, 9}, {4, 8}, {3, 5, 7, 9}, {4, 6, 8}, {5, 7}},
}

// ageLayout is the derived geometry of one age.
type ageLayout struct {
	pos       [SlotsPerAge]Pos
	cover     [SlotsPerAge][2]int8 // the slots at (r+1, c-1) and (r+1, c+1), or -1 where there is none
	coverMask [SlotsPerAge]uint32  // the same as a slot mask
	faceUp    uint32               // the slots dealt face up
	rows      int8
}

// allSlots is the mask of every slot of a structure.
const allSlots uint32 = 1<<SlotsPerAge - 1

var layouts = buildLayouts()

func buildLayouts() [NumAges]ageLayout {
	var out [NumAges]ageLayout
	for a, rows := range layoutRows {
		l := &out[a]
		l.rows = int8(len(rows))
		i := 0
		for r, cols := range rows {
			for _, c := range cols {
				if i == SlotsPerAge {
					panic("engine: an age layout has more than 20 slots")
				}
				l.pos[i] = Pos{Row: int8(r + 1), Col: c}
				if (r+1)%2 == 1 {
					l.faceUp |= 1 << i
				}
				i++
			}
		}
		if i != SlotsPerAge {
			panic("engine: an age layout has fewer than 20 slots")
		}
		for i := range l.pos {
			l.cover[i] = [2]int8{-1, -1}
			for j := range l.pos {
				if l.pos[j].Row != l.pos[i].Row+1 {
					continue
				}
				switch l.pos[j].Col {
				case l.pos[i].Col - 1:
					l.cover[i][0] = int8(j)
				case l.pos[i].Col + 1:
					l.cover[i][1] = int8(j)
				}
			}
			for _, j := range l.cover[i] {
				if j >= 0 {
					l.coverMask[i] |= 1 << j
				}
			}
		}
	}
	return out
}

// layoutOf returns the geometry of an age. It panics unless 1 <= age <= 3.
func layoutOf(age uint8) *ageLayout {
	if age < 1 || age > NumAges {
		panic("engine: no structure for age " + strconv.Itoa(int(age)))
	}
	return &layouts[age-1]
}

// SlotPos returns the place of slot i (0 to 19) in the structure of an age
// (1 to 3).
func SlotPos(age uint8, i int) Pos { return layoutOf(age).pos[i] }

// Coverers returns the slots that partly cover slot i: the one at (r+1, c-1)
// and the one at (r+1, c+1), each -1 where the structure has no card.
func Coverers(age uint8, i int) [2]int8 { return layoutOf(age).cover[i] }

// CoverMask returns the slots that partly cover slot i, as a slot mask.
func CoverMask(age uint8, i int) uint32 { return layoutOf(age).coverMask[i] }

// FaceUpMask returns the slots that an age deals face up (its odd rows).
func FaceUpMask(age uint8) uint32 { return layoutOf(age).faceUp }

// Rows returns the number of rows of an age's structure (5, 5 or 7).
func Rows(age uint8) int { return int(layoutOf(age).rows) }

// AccessibleMask returns the slots of occupied that no other occupied slot
// covers: the cards a player may take, given the slots that still hold a
// card.
func AccessibleMask(age uint8, occupied uint32) uint32 {
	l := layoutOf(age)
	occupied &= allSlots
	var acc uint32
	for i := range SlotsPerAge {
		if occupied&(1<<i) != 0 && occupied&l.coverMask[i] == 0 {
			acc |= 1 << i
		}
	}
	return acc
}
