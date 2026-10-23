package rules

import (
	"fmt"
	"math/rand/v2"
	"os"
	"sync/atomic"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
)

type oraclePairs [15][15]bool

// The oracle tries every physical supplier of each demanded unit in turn. It
// never takes fixed production first, splits by resource, or matches/prunes own
// coverage. Every affordable complete covering enters the dominance frontier,
// including those buying units own production could have covered.
func paymentOracle(in tradeInput) oraclePairs {
	return paymentOraclePhysical(in, true)
}

// Equal unused masks in one city are interchangeable physical suppliers.
// Explore one representative at each depth, but retain every producer bit so
// two equal symbols can still cover two units. This removes only permutations
// of identical symbols, not coverings, purchase pairs or duplicate capacity.
// The uncollapsed traversal remains available for the equivalence test.
func paymentOraclePhysical(in tradeInput, collapseIdentical bool) oraclePairs {
	var demand []catalog.Resource
	for r, n := range in.need {
		for k := uint8(0); k < n; k++ {
			demand = append(demand, catalog.Resource(r))
		}
	}
	var fixed [3][catalog.NumResources]uint8
	var used [3]uint16
	var pairs oraclePairs
	var visit func(int, int, int)
	visit = func(at, left, right int) {
		if at == len(demand) {
			if left+right+int(in.printed) <= int(in.purse) {
				pairs[left][right] = true
			}
			return
		}
		r := demand[at]
		for city := 0; city < 3; city++ {
			supply := in.own[r]
			choices := in.ownChoice[:in.nOwnChoice]
			dl, dr := 0, 0
			if city > 0 {
				supply = in.sell[city-1][r]
				choices = in.sellChoice[city-1][:in.nSellChoice[city-1]]
				if city == 1 {
					dl = int(in.price[0][r])
				} else {
					dr = int(in.price[1][r])
				}
			}
			if fixed[city][r] < supply {
				fixed[city][r]++
				visit(at+1, left+dl, right+dr)
				fixed[city][r]--
			}
			var seen [128]bool
			for p, mask := range choices {
				bit := uint16(1) << p
				if used[city]&bit == 0 && mask.Has(r) {
					if collapseIdentical && seen[mask] {
						continue
					}
					seen[mask] = true
					used[city] |= bit
					visit(at+1, left+dl, right+dr)
					used[city] &^= bit
				}
			}
		}
	}
	visit(0, 0, 0)
	var frontier oraclePairs
	for l := 0; l < 15; l++ {
		for r := 0; r < 15; r++ {
			if !pairs[l][r] {
				continue
			}
			beaten := false
			for a := 0; a <= l; a++ {
				for b := 0; b <= r; b++ {
					if (a != l || b != r) && pairs[a][b] {
						beaten = true
					}
				}
			}
			frontier[l][r] = !beaten
		}
	}
	return frontier
}

func validatePaymentWitness(t testing.TB, in tradeInput, p Payment) {
	t.Helper()
	var counts [catalog.NumResources]uint8
	var fixed [3][catalog.NumResources]uint8
	var used [3]uint16
	var left, right int
	if p.Units > catalog.MaxCostUnits {
		t.Fatal("witness overflow")
	}
	for _, u := range p.Witness[:p.Units] {
		if u.Resource >= catalog.NumResources {
			t.Fatal("invalid witness resource")
		}
		counts[u.Resource]++
		city := -1
		choice := false
		switch u.Source {
		case OwnFixed:
			city = 0
		case OwnChoice:
			city = 0
			choice = true
		case LeftFixed:
			city = 1
		case LeftChoice:
			city = 1
			choice = true
		case RightFixed:
			city = 2
		case RightChoice:
			city = 2
			choice = true
		default:
			t.Fatal("invalid witness source")
		}
		supply := in.own[u.Resource]
		choices := in.ownChoice[:in.nOwnChoice]
		price := uint8(0)
		if city > 0 {
			supply = in.sell[city-1][u.Resource]
			choices = in.sellChoice[city-1][:in.nSellChoice[city-1]]
			price = in.price[city-1][u.Resource]
		}
		if u.Price != price {
			t.Fatal("witness price")
		}
		if choice {
			if int(u.Producer) >= len(choices) || !choices[u.Producer].Has(u.Resource) || used[city]&(1<<u.Producer) != 0 {
				t.Fatalf("choice producer reused/missing: %+v", p)
			}
			used[city] |= 1 << u.Producer
		} else {
			if u.Producer >= supply {
				t.Fatal("missing fixed producer")
			}
			fixed[city][u.Resource]++
			if fixed[city][u.Resource] > supply {
				t.Fatal("fixed producer reused")
			}
			// Producer ids are unique per resource, even when a supplier has doubles.
		}
		if city == 1 {
			left += int(price)
		}
		if city == 2 {
			right += int(price)
		}
	}
	for j, u := range p.Witness[:p.Units] {
		if u.Source == OwnFixed || u.Source == LeftFixed || u.Source == RightFixed {
			for _, v := range p.Witness[:j] {
				if u.Resource == v.Resource && u.Source == v.Source && u.Producer == v.Producer {
					t.Fatal("fixed id reused")
				}
			}
		}
	}
	if counts != in.need || left != int(p.Left) || right != int(p.Right) || left+right+int(in.printed) > int(in.purse) {
		t.Fatalf("invalid witness %+v for %+v", p, in)
	}
	for _, u := range p.Witness[p.Units:] {
		if u != (PaymentUnit{}) {
			t.Fatal("nonzero witness padding")
		}
	}
}

func checkPaymentOracle(t testing.TB, in tradeInput) {
	t.Helper()
	expected := paymentOracle(in)
	got := solvePayment(in)
	var actual oraclePairs
	if got.Count > MaxPaymentOutcomes {
		t.Fatal("outcome bound")
	}
	priorTotal, priorLeft := -1, -1
	for _, p := range got.Outcomes[:got.Count] {
		total := int(p.Left) + int(p.Right)
		if total < priorTotal || (total == priorTotal && int(p.Left) <= priorLeft) {
			t.Fatal("noncanonical order")
		}
		priorTotal = total
		priorLeft = int(p.Left)
		if p.Left > 14 || p.Right > 14 || actual[p.Left][p.Right] {
			t.Fatal("duplicate/out of range")
		}
		actual[p.Left][p.Right] = true
		validatePaymentWitness(t, in, p)
	}
	if actual != expected {
		t.Fatalf("oracle mismatch input=%+v got=%v expected=%v", in, actual, expected)
	}
	q := quotePayment(in)
	if q.Affordable != (got.Count > 0) || (got.Count > 0 && q.Total != uint16(in.printed)+uint16(got.Outcomes[0].Left)+uint16(got.Outcomes[0].Right)) {
		t.Fatalf("quote mismatch %+v input %+v", q, in)
	}
}

func TestPaymentOracleRandom(t *testing.T) {
	rng := rand.New(rand.NewPCG(0x73a91, 0x908fe))
	for k := 0; k < 100000; k++ {
		checkPaymentOracle(t, randomTrade(rng))
	}
	t.Log("100000 catalog-shaped inputs agree, including every witness and quote")
}

func randomTrade(rng *rand.Rand) tradeInput {
	in := baseTrade()
	in.purse = int16(rng.IntN(22))
	in.printed = uint8(rng.IntN(4))
	for k := rng.IntN(8); k > 0; k-- {
		in.need[rng.IntN(int(catalog.NumResources))]++
	}
	// Fixed production models repeated brown/grey/board symbols, choices model
	// two-raw brown cards and raw/manufactured yellow/wonder production.
	rawPairs := [...]catalog.ResMask{3, 5, 9, 6, 10, 12}
	for r := range in.need {
		in.own[r] = uint8(rng.IntN(3))
		for s := 0; s < 2; s++ {
			in.sell[s][r] = uint8(rng.IntN(3))
			in.price[s][r] = uint8(1 + rng.IntN(2))
		}
	}
	if rng.IntN(2) == 0 {
		in.own = [catalog.NumResources]uint8{}
	}
	in.nOwnChoice = uint8(rng.IntN(MaxChoice + 1))
	if rng.IntN(2) == 0 {
		in.nOwnChoice = 0
	}
	for p := range in.nOwnChoice {
		in.ownChoice[p] = rawPairs[rng.IntN(len(rawPairs))]
		switch rng.IntN(4) {
		case 0:
			in.ownChoice[p] = catalog.Raw
		case 1:
			in.ownChoice[p] = catalog.Manufactured
		}
	}
	for s := 0; s < 2; s++ {
		in.nSellChoice[s] = uint8(rng.IntN(MaxForSaleChoice + 1))
		for p := range in.nSellChoice[s] {
			in.sellChoice[s][p] = rawPairs[rng.IntN(len(rawPairs))]
		}
	}
	return in
}

// Exhaust all costs <=4 over three resources, every nonempty mask, <=2
// choice producers per city, fixed capacities 0..need for EVERY resource in
// EVERY city, and all 64 price matrices. Resource permutations are redundant:
// use nonincreasing needs and enumerate ALL masks/prices in those coordinates.
// Producer order is immaterial to outcomes; duplicates remain distinct
// capacities. Random/fuzz
// cases exercise arbitrary producer order, all seven resources and full limits.
// Opt in because this sweep has millions of cases; verify-quick retains the
// 100000-case oracle and the small representative sweep.
func TestPaymentOracleExhaustive(t *testing.T) {
	if os.Getenv("HAFTGANE_PAYMENT_EXHAUSTIVE") != "1" {
		t.Skip("set HAFTGANE_PAYMENT_EXHAUSTIVE=1 for full small-universe sweep")
	}
	var total atomic.Uint64
	t.Cleanup(func() { t.Logf("exhaustive cases: %d", total.Load()) })
	// There are 11 nonincreasing vectors (including zero) of total at most four
	// over three resources. Preserve exactly the original exhaustive universe.
	for a := uint8(0); a <= 4; a++ {
		for b := uint8(0); b <= a && a+b <= 4; b++ {
			for c := uint8(0); c <= b && a+b+c <= 4; c++ {
				need := [catalog.NumResources]uint8{a, b, c}
				t.Run(fmt.Sprintf("need-%d-%d-%d", a, b, c), func(t *testing.T) {
					t.Parallel()
					var cases uint64
					profiles := expandedPaymentProfiles(need)
					for _, own := range profiles {
						if !own.relevant(need) {
							continue
						}
						for _, left := range profiles {
							if !left.relevant(need) {
								continue
							}
							for _, right := range profiles {
								if !right.relevant(need) {
									continue
								}
								in := baseTrade()
								in.need = need
								in.own = own.fixed
								in.ownChoice[0] = own.choice[0]
								in.ownChoice[1] = own.choice[1]
								in.nOwnChoice = own.n
								in.sell[0] = left.fixed
								in.sell[1] = right.fixed
								in.sellChoice[0][0] = left.choice[0]
								in.sellChoice[0][1] = left.choice[1]
								in.nSellChoice[0] = left.n
								in.sellChoice[1][0] = right.choice[0]
								in.sellChoice[1][1] = right.choice[1]
								in.nSellChoice[1] = right.n
								for prices := 0; prices < 64; prices++ {
									for s := 0; s < 2; s++ {
										for r := 0; r < 3; r++ {
											in.price[s][r] = uint8(1 + (prices>>(s*3+r))&1)
										}
									}
									checkPaymentOracle(t, in)
									cases++
								}
							}
						}
					}
					total.Add(cases)
					t.Logf("exhaustive cases: %d", cases)
				})
			}
		}
	}
}

type smallPaymentProfile struct {
	fixed  [catalog.NumResources]uint8
	choice [2]catalog.ResMask
	n      uint8
}

func (p smallPaymentProfile) relevant(need [catalog.NumResources]uint8) bool {
	var mask catalog.ResMask
	for r, n := range need {
		if n > 0 {
			mask |= catalog.Resource(r).Mask()
		}
		if p.fixed[r] > n {
			return false
		}
	}
	for _, m := range p.choice[:p.n] {
		if m&mask != m {
			return false
		}
	}
	return true
}
func smallPaymentProfiles() []smallPaymentProfile {
	// Seven choice masks plus three fixed-symbol types; multisets of size <=2.
	add := func(p smallPaymentProfile, k int) smallPaymentProfile {
		if k < 3 {
			p.fixed[k]++
		} else {
			p.choice[p.n] = catalog.ResMask(k - 2)
			p.n++
		}
		return p
	}
	out := []smallPaymentProfile{{}}
	for a := 0; a < 10; a++ {
		p := add(smallPaymentProfile{}, a)
		out = append(out, p)
		for b := a; b < 10; b++ {
			out = append(out, add(p, b))
		}
	}
	return out
}

func TestPaymentOracleSmall(t *testing.T) {
	for _, p := range smallPaymentProfiles() {
		for _, q := range smallPaymentProfiles() {
			in := baseTrade()
			in.need[0] = 1
			in.need[1] = 2
			in.own = p.fixed
			in.ownChoice[0] = p.choice[0]
			in.ownChoice[1] = p.choice[1]
			in.nOwnChoice = p.n
			in.sell[0] = q.fixed
			in.nSellChoice[0] = q.n
			in.sellChoice[0][0] = q.choice[0]
			in.sellChoice[0][1] = q.choice[1]
			in.sell[1][0] = 1
			in.sell[1][1] = 1
			in.price[0][0] = 1
			checkPaymentOracle(t, in)
		}
	}
}

// Capacities beyond need are indistinguishable for a single construction.
// A resource with fixed >= need no longer needs a choice symbol, so erase that
// bit from masks and omit empty symbols. This is a quotient of input universes,
// not an oracle shortcut: paymentOracle still tries each physical unit without
// taking fixed production first. The initial small sweep additionally covers
// redundant fixed/choice overlap; random/fuzz cover full-size original inputs.
func expandedPaymentProfiles(need [catalog.NumResources]uint8) []smallPaymentProfile {
	var out []smallPaymentProfile
	for a := uint8(0); a <= need[0]; a++ {
		for b := uint8(0); b <= need[1]; b++ {
			for c := uint8(0); c <= need[2]; c++ {
				p := smallPaymentProfile{fixed: [catalog.NumResources]uint8{a, b, c}}
				var active catalog.ResMask
				for r, n := range need {
					if p.fixed[r] < n {
						active |= catalog.Resource(r).Mask()
					}
				}
				out = append(out, p)
				for x := catalog.ResMask(1); x < 8; x++ {
					if x&active != x {
						continue
					}
					p.choice = [2]catalog.ResMask{x, 0}
					p.n = 1
					out = append(out, p)
					for y := x; y < 8; y++ {
						if y&active != y {
							continue
						}
						p.choice[1] = y
						p.n = 2
						out = append(out, p)
					}
				}
			}
		}
	}
	return out
}

// Compare the original physical enumeration with its symmetry reduction on
// duplicated and permuted suppliers. The full exhaustive small universe and
// fuzz tests separately compare the reduced oracle with production.
func TestPaymentOraclePhysicalSymmetry(t *testing.T) {
	r := rand.New(rand.NewPCG(0x73796d6d65747279, 0x706879736963616c))
	for k := 0; k < 1000; k++ {
		in := randomTrade(r)
		in.need = [catalog.NumResources]uint8{}
		for j := r.IntN(5); j > 0; j-- {
			in.need[r.IntN(7)]++
		}
		in.nOwnChoice = uint8(r.IntN(5))
		for p := range in.nOwnChoice {
			in.ownChoice[p] = catalog.Raw
		}
		for city := 0; city < 2; city++ {
			in.nSellChoice[city] = uint8(r.IntN(4))
			for p := range in.nSellChoice[city] {
				in.sellChoice[city][p] = catalog.ResMask(3 + city*12)
			}
		}
		if before, after := paymentOraclePhysical(in, false), paymentOraclePhysical(in, true); before != after {
			t.Fatalf("symmetry changed outcomes input=%+v before=%v after=%v", in, before, after)
		}
	}
	t.Log("1000 duplicated/permuted physical-supplier worlds preserve exact frontiers")
}
