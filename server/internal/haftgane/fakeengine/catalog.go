package fakeengine

// DesignID identifies a card design: 1 to NumDesigns, in catalog order. Zero
// means "no card".
type DesignID uint8

// BoardID identifies a board: 0 to NumBoards-1.
type BoardID uint8

// NumDesigns is the number of designs: seven per Age and four relics.
const NumDesigns = 18

// NumBoards is the number of boards, one per seat at seven seats.
const NumBoards = 7

// The two boards with an ability. The other five have none.
const (
	BoardArchive BoardID = 0 // salvage: build a design from the pile after tucking
	BoardOrchard BoardID = 1 // last card: keep each Age's leftover and play it
)

// Design is one card as printed. Copies lists the minimum seat count of each
// copy (a copy is in the deck when the game has at least that many seats); a
// relic has one copy, always in Age 2's deck.
type Design struct {
	ID     DesignID
	Slug   string
	Age    uint8
	Relic  bool
	Cost   uint8 // coins paid to the neighbours, in any split
	Points uint8
	Copies [3]uint8
}

// copyPattern gives each Age's seven designs, in id order, their copies: at
// every seat count n the Age has exactly 3n copies, nine of them at three
// seats and three more per extra seat.
var copyPattern = [7][3]uint8{
	{3, 3, 3}, {3, 3, 4}, {3, 3, 5}, {3, 4, 6}, {3, 4, 5}, {5, 6, 7}, {6, 7, 7},
}

// designs is the catalog, indexed by DesignID (entry 0 is unused). Slugs are
// distinct, lower-case, digit-free words, and none is a prefix of another
// followed by a hyphen, so a slug with a suffix is never another slug.
var designs = [NumDesigns + 1]Design{
	{},
	{ID: 1, Slug: "amber", Age: 1, Cost: 0, Points: 1, Copies: copyPattern[0]},
	{ID: 2, Slug: "basalt", Age: 1, Cost: 0, Points: 1, Copies: copyPattern[1]},
	{ID: 3, Slug: "cedar", Age: 1, Cost: 1, Points: 2, Copies: copyPattern[2]},
	{ID: 4, Slug: "dune", Age: 1, Cost: 1, Points: 2, Copies: copyPattern[3]},
	{ID: 5, Slug: "ember", Age: 1, Cost: 2, Points: 3, Copies: copyPattern[4]},
	{ID: 6, Slug: "flint", Age: 1, Cost: 1, Points: 1, Copies: copyPattern[5]},
	{ID: 7, Slug: "garnet", Age: 1, Cost: 2, Points: 2, Copies: copyPattern[6]},
	{ID: 8, Slug: "harbor", Age: 2, Cost: 0, Points: 1, Copies: copyPattern[0]},
	{ID: 9, Slug: "iris", Age: 2, Cost: 1, Points: 2, Copies: copyPattern[1]},
	{ID: 10, Slug: "jade", Age: 2, Cost: 1, Points: 2, Copies: copyPattern[2]},
	{ID: 11, Slug: "kelp", Age: 2, Cost: 2, Points: 3, Copies: copyPattern[3]},
	{ID: 12, Slug: "lotus", Age: 2, Cost: 0, Points: 1, Copies: copyPattern[4]},
	{ID: 13, Slug: "marble", Age: 2, Cost: 2, Points: 4, Copies: copyPattern[5]},
	{ID: 14, Slug: "nacre", Age: 2, Cost: 1, Points: 3, Copies: copyPattern[6]},
	{ID: 15, Slug: "obelisk", Age: 2, Relic: true, Cost: 2, Points: 5, Copies: [3]uint8{3}},
	{ID: 16, Slug: "prism", Age: 2, Relic: true, Cost: 1, Points: 4, Copies: [3]uint8{3}},
	{ID: 17, Slug: "quartz", Age: 2, Relic: true, Cost: 2, Points: 5, Copies: [3]uint8{3}},
	{ID: 18, Slug: "rune", Age: 2, Relic: true, Cost: 0, Points: 3, Copies: [3]uint8{3}},
}

// boardSlugs names the boards by BoardID.
var boardSlugs = [NumBoards]string{"archive", "orchard", "bastion", "citadel", "dock", "spire", "vault"}

// Designs returns the catalog in id order (a copy; entry i has ID i+1).
func Designs() []Design {
	out := make([]Design, 0, NumDesigns)
	for d := DesignID(1); d <= NumDesigns; d++ {
		out = append(out, designs[d])
	}
	return out
}

// Slug returns a design's wire id, or "" for an id outside the catalog.
func (d DesignID) Slug() string {
	if d == 0 || d > NumDesigns {
		return ""
	}
	return designs[d].Slug
}

// DesignBySlug returns the design with that wire id.
func DesignBySlug(slug string) (DesignID, bool) {
	for d := DesignID(1); d <= NumDesigns; d++ {
		if designs[d].Slug == slug {
			return d, true
		}
	}
	return 0, false
}

// Slug returns a board's wire id, or "" for an id outside the catalog.
func (b BoardID) Slug() string {
	if int(b) >= NumBoards {
		return ""
	}
	return boardSlugs[b]
}

// BoardSlugs returns every board's wire id, in BoardID order.
func BoardSlugs() []string {
	out := make([]string, 0, NumBoards)
	for _, s := range boardSlugs {
		out = append(out, s)
	}
	return out
}

// copiesFor returns how many copies of design d a game of n seats holds.
func copiesFor(d DesignID, n uint8) uint8 {
	var c uint8
	for _, m := range designs[d].Copies {
		if m != 0 && m <= n {
			c++
		}
	}
	return c
}
