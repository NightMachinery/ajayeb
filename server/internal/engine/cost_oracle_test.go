package engine

import (
	"fmt"
	"math"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
)

// The oracle for cheapest: an exhaustive search over every way to cover
// every unit of a cost, written without any of the greedy reasoning of
// cost.go, and a validator that checks a Payment is a real way to pay.

// costOracleCosts are the printed costs of the 73 buildings and the 12
// wonders of the base game, in the reference notation, so the random inputs
// have the shapes real costs have.
var costOracleCosts = [...]string{
	// Age I
	"", "1c", "", "1c", "", "1c", "1c", "1c", "W", "C", "2c", "", "2c", "P", "2c", "G", "", "3c", "3c", "3c",
	"", "", "S",
	// Age II
	"2c", "2c", "2c", "", "", "C W", "3c", "SS", "S W P", "CC G", "W PP", "W GG", "CC S", "S W G", "",
	"3c + C", "2c + G P", "4c", "W P", "CC", "WW G", "SSS", "S W",
	// Age III
	"CC SS", "CCC WW", "WWW G", "SS C P", "8c", "S W GG", "C G P", "WW G P", "S PP", "C S W", "PP", "W G P",
	"CC G", "SS G", "C S W GG", "CC WW", "C W PP", "SSS WW", "CC S P", "SS G",
	// Guilds
	"C W G P", "C S G P", "SS C W G", "WW C P", "CC WW", "SS WW", "SS C P",
	// Wonders
	"SSS P", "CCC G", "PP S W", "W S G P", "CC GG P", "WWW G P", "WW C S", "WW G P", "PP C W S", "S C GG",
	"SS CC P", "SS W G",
}

// oracleTrade returns the fewest coins in must spend on trading, found by
// trying, for every unit of the cost in turn, each way to cover it: own
// production of that resource, each unused flexible producer that can make
// it, a unit of the discount, or buying it. The search is exact; the only
// pruning drops partial payments that already cost as much as the best
// complete one.
func oracleTrade(in *costInput) int {
	if in.chainFree {
		return 0
	}
	var price [catalog.NumResources]int
	for r := range catalog.NumResources {
		if in.fix&(1<<r) != 0 {
			price[r] = 1
		} else {
			price[r] = 2 + int(in.oppProd[r])
		}
	}
	var units []catalog.Resource
	for r := range catalog.NumResources {
		for range in.need[r] {
			units = append(units, r)
		}
	}
	ownLeft := in.own
	var used [MaxChoice]bool
	nChoice := min(int(in.nChoice), MaxChoice)
	discountLeft := int(in.discount)
	best := math.MaxInt
	var cover func(i, spent int)
	cover = func(i, spent int) {
		if spent >= best {
			return
		}
		if i == len(units) {
			best = spent
			return
		}
		r := units[i]
		if ownLeft[r] > 0 {
			ownLeft[r]--
			cover(i+1, spent)
			ownLeft[r]++
		}
		for j := range nChoice {
			if !used[j] && in.choice[j]&(1<<r) != 0 {
				used[j] = true
				cover(i+1, spent)
				used[j] = false
			}
		}
		if discountLeft > 0 {
			discountLeft--
			cover(i+1, spent)
			discountLeft++
		}
		cover(i+1, spent+price[r])
	}
	cover(0, 0)
	return best
}

// costPaymentProblems lists every way pay fails to be a valid payment for
// in: units covered twice or not at all, producers or discount used beyond
// what in allows, or totals that do not add up. It says nothing about
// optimality; the oracle checks that.
func costPaymentProblems(in *costInput, pay *Payment) []string {
	var bad []string
	fail := func(format string, args ...any) { bad = append(bad, fmt.Sprintf(format, args...)) }
	for r := range catalog.NumResources {
		want := uint8(2 + min(int(in.oppProd[r]), 253))
		if in.fix&(1<<r) != 0 {
			want = 1
		}
		if pay.UnitPrice[r] != want {
			fail("unit price of %v is %d, want %d", r, pay.UnitPrice[r], want)
		}
	}
	if pay.FixedPrice != in.fix&catalog.AllResources {
		fail("fixed prices %b, want %b", pay.FixedPrice, in.fix&catalog.AllResources)
	}
	if in.chainFree {
		zero := Payment{ChainFree: true, Affordable: true, UrbanismBonus: int16(in.chainBonus),
			UnitPrice: pay.UnitPrice, FixedPrice: pay.FixedPrice, ChainFrom: pay.ChainFrom}
		if *pay != zero {
			fail("chain-free payment is not empty: %+v", *pay)
		}
		return bad
	}
	if pay.ChainFree || pay.ChainFrom != 0 || pay.UrbanismBonus != 0 {
		fail("chain fields set on a paid build: free %v, from %v, bonus %d", pay.ChainFree, pay.ChainFrom, pay.UrbanismBonus)
	}

	// Producers: each use matches a distinct producer of in, and covers a
	// resource that producer can make.
	nChoice := min(int(in.nChoice), MaxChoice)
	if int(pay.NChoice) > nChoice {
		fail("%d producers used, only %d available", pay.NChoice, nChoice)
	}
	var byProducer [catalog.NumResources]int
	var taken [MaxChoice]bool
	for _, u := range pay.Choice[:min(int(pay.NChoice), MaxChoice)] {
		if !u.Mask.Has(u.Res) {
			fail("producer %b covers %v, which it cannot make", u.Mask, u.Res)
		}
		found := false
		for j := range nChoice {
			if !taken[j] && in.choice[j] == u.Mask {
				taken[j], found = true, true
				break
			}
		}
		if !found {
			fail("producer %b used more often than in has it", u.Mask)
		}
		if u.Res < catalog.NumResources {
			byProducer[u.Res]++
		}
	}
	for k := int(pay.NChoice); k < MaxChoice; k++ {
		if pay.Choice[k] != (ChoiceUse{}) {
			fail("unused choice entry %d is set: %+v", k, pay.Choice[k])
		}
	}

	// Coverage: every unit of the need is covered exactly once.
	discount, trade := 0, 0
	for r := range catalog.NumResources {
		if pay.Own[r] > in.own[r] {
			fail("own %v: %d used, %d produced", r, pay.Own[r], in.own[r])
		}
		covered := int(pay.Own[r]) + byProducer[r] + int(pay.Discount[r]) + int(pay.TradeUnits[r])
		if covered != int(in.need[r]) {
			fail("%v: %d units covered, %d needed", r, covered, in.need[r])
		}
		discount += int(pay.Discount[r])
		trade += int(pay.TradeUnits[r]) * int(pay.UnitPrice[r])
	}
	if discount > int(in.discount) {
		fail("%d units waived, the discount allows %d", discount, in.discount)
	}
	if int(pay.TradeTotal) != trade {
		fail("trade total %d, bought units cost %d", pay.TradeTotal, trade)
	}
	if pay.PrintedCoins != in.printed || pay.Total != pay.PrintedCoins+pay.TradeTotal {
		fail("printed %d (want %d), total %d (want printed + trade = %d)",
			pay.PrintedCoins, in.printed, pay.Total, pay.PrintedCoins+pay.TradeTotal)
	}
	if pay.Affordable != (pay.Total <= in.purse) {
		fail("affordable %v with total %d and %d coins", pay.Affordable, pay.Total, in.purse)
	}
	wantOpp := int16(0)
	if in.tradeIncome {
		wantOpp = pay.TradeTotal
	}
	if pay.ToOpponent != wantOpp {
		fail("to opponent %d, want %d", pay.ToOpponent, wantOpp)
	}
	return bad
}

// costPick returns i with probability w[i] / sum(w).
func costPick(r *RNG, w ...int) int {
	sum := 0
	for _, x := range w {
		sum += x
	}
	v := r.IntN(sum)
	for i, x := range w {
		if v < x {
			return i
		}
		v -= x
	}
	return len(w) - 1
}

// costRealisticInput draws an input the base game can produce: a real
// printed cost; production of 0 to 4 raw units and 0 to 2 manufactured units
// of each resource on both sides; reserves and the Customs House; at most two
// flexible producers of each class, in random order; the 2-unit discount;
// Economy; and the occasional chain.
func costRealisticInput(r *RNG, costs []catalog.Cost) costInput {
	var in costInput
	c := costs[r.IntN(len(costs))]
	in.printed, in.need = int16(c.Coins), c.Res
	for res := range catalog.NumResources {
		if catalog.RawGoods.Has(res) {
			in.own[res] = uint8(costPick(r, 50, 25, 15, 6, 4))
			in.oppProd[res] = uint8(costPick(r, 50, 25, 15, 6, 4))
			if r.IntN(100) < 15 {
				in.fix |= res.Mask()
			}
		} else {
			in.own[res] = uint8(costPick(r, 60, 30, 10))
			in.oppProd[res] = uint8(costPick(r, 60, 30, 10))
		}
	}
	if r.IntN(100) < 15 {
		in.fix |= catalog.Manufactured
	}
	for range costPick(r, 55, 30, 15) {
		in.choice[in.nChoice] = catalog.RawGoods
		in.nChoice++
	}
	for range costPick(r, 55, 30, 15) {
		in.choice[in.nChoice] = catalog.Manufactured
		in.nChoice++
	}
	r.Shuffle(int(in.nChoice), func(i, j int) { in.choice[i], in.choice[j] = in.choice[j], in.choice[i] })
	if r.IntN(100) < 35 {
		in.discount = 2
	}
	in.purse = int16(r.IntN(21))
	in.tradeIncome = r.IntN(4) == 0
	if r.IntN(20) == 0 {
		in.chainFree = true
		if r.IntN(2) == 0 {
			in.chainBonus = 4
		}
	}
	return in
}

// costStressInput draws inputs beyond the base game, to test the greedy
// argument rather than the data: up to 6 units of any resources, up to 4
// production of anything, any fixed prices, up to four producers of either
// class in any mix, and discounts of 0 to 3 units.
func costStressInput(r *RNG) costInput {
	var in costInput
	for range r.IntN(7) {
		in.need[r.IntN(int(catalog.NumResources))]++
	}
	in.printed = int16(r.IntN(9))
	for res := range catalog.NumResources {
		in.own[res] = uint8(costPick(r, 60, 20, 10, 5, 5))
		in.oppProd[res] = uint8(r.IntN(5))
	}
	in.fix = catalog.ResMask(r.IntN(32))
	in.nChoice = uint8(r.IntN(MaxChoice + 1))
	for i := range in.nChoice {
		in.choice[i] = catalog.RawGoods
		if r.IntN(2) == 0 {
			in.choice[i] = catalog.Manufactured
		}
	}
	in.discount = uint8(r.IntN(4))
	in.purse = int16(r.IntN(31))
	in.tradeIncome = r.IntN(2) == 0
	if r.IntN(50) == 0 {
		in.chainFree = true
		in.chainBonus = uint8(r.IntN(6))
	}
	return in
}

// TestCheapestOracle draws 20,000 realistic inputs and 5,000 stress inputs.
// For each, cheapest must give a valid payment whose trading cost equals the
// exhaustive search, with and without a breakdown.
func TestCheapestOracle(t *testing.T) {
	costs := make([]catalog.Cost, len(costOracleCosts))
	for i, s := range costOracleCosts {
		costs[i] = catalog.MustCost(s)
	}
	const realistic, stress = 20000, 5000
	rng := NewRNG(20261002)
	var usedChoice, usedDiscount, usedBoth, bought, fixed, failures int
	for i := range realistic + stress {
		var in costInput
		if i < realistic {
			in = costRealisticInput(&rng, costs)
		} else {
			in = costStressInput(&rng)
		}
		var pay Payment
		total, trade := cheapest(&in, &pay)
		wantTrade := oracleTrade(&in)
		wantTotal := 0
		if !in.chainFree {
			wantTotal = int(in.printed) + wantTrade
		}
		problems := costPaymentProblems(&in, &pay)
		if int(trade) != wantTrade || int(total) != wantTotal {
			problems = append(problems, fmt.Sprintf("cheapest (%d, %d), exhaustive search (%d, %d)",
				total, trade, wantTotal, wantTrade))
		}
		if t2, tr2 := cheapest(&in, nil); t2 != total || tr2 != trade {
			problems = append(problems, fmt.Sprintf("without a breakdown (%d, %d), with one (%d, %d)",
				t2, tr2, total, trade))
		}
		if q := quote(&in); q != pay.Quote() {
			problems = append(problems, fmt.Sprintf("quote %+v, payment %+v", q, pay.Quote()))
		}
		if len(problems) > 0 {
			failures++
			t.Errorf("case %d:\ninput   %+v\npayment %+v\n%v", i, in, pay, problems)
			if failures >= 10 {
				t.Fatal("too many failures")
			}
		}
		if pay.NChoice > 0 {
			usedChoice++
		}
		if pay.Discount != [5]uint8{} {
			usedDiscount++
			if pay.NChoice > 0 {
				usedBoth++
			}
		}
		if pay.TradeTotal > 0 {
			bought++
		}
		if in.fix != 0 && pay.TradeTotal > 0 {
			fixed++
		}
	}
	// The draw must exercise every part of the algorithm, or the comparison
	// proves little. (About a third of the real costs are coins only, so
	// fewer than half of the cases buy anything.)
	t.Logf("%d cases: %d use producers, %d a discount, %d both, %d buy, %d with fixed prices",
		realistic+stress, usedChoice, usedDiscount, usedBoth, bought, fixed)
	if usedChoice < 3000 || usedDiscount < 3000 || usedBoth < 1000 || bought < 5000 || fixed < 1000 {
		t.Errorf("the random inputs miss parts of the algorithm")
	}
}

// The oracle itself, on payments worked out by hand.
func TestOracleKnownCases(t *testing.T) {
	man, raw := catalog.Manufactured, catalog.RawGoods
	cases := []struct {
		name string
		in   costInput
		want int
	}{
		{"nothing to pay", costInput{}, 0},
		{"one unit bought", costInput{need: costRes("C"), oppProd: costRes("CC")}, 4},
		{"own production first", costInput{need: costRes("WW"), own: costRes("W")}, 2},
		{"docs Palace", costInput{need: costRes("C S W GG"), oppProd: costRes("S G")}, 13},
		{"docs Palace with Masonry", costInput{need: costRes("C S W GG"), oppProd: costRes("S G"), discount: 2}, 7},
		{"docs Palace with Masonry and Forum", costInput{need: costRes("C S W GG"), oppProd: costRes("S G"),
			discount: 2, choice: [MaxChoice]catalog.ResMask{man}, nChoice: 1}, 4},
		{"fixed price", costInput{need: costRes("CC"), oppProd: costRes("CCC"), fix: catalog.Clay.Mask()}, 2},
		{"two raw producers", costInput{need: costRes("SS C"), oppProd: costRes("S"),
			choice: [MaxChoice]catalog.ResMask{raw, raw}, nChoice: 2}, 2},
		{"chain", costInput{need: costRes("SSS"), chainFree: true}, 0},
		// With masks that are not one class each, covering the dearest unit
		// first is wrong: here the mixed producer must take the wood so the
		// glass-only producer can take the glass. The oracle finds 0; this
		// is why cost.go needs class masks.
		{"mixed masks", costInput{need: costRes("W G"), oppProd: costRes("G"),
			choice:  [MaxChoice]catalog.ResMask{catalog.MaskOf(catalog.Wood, catalog.Glass), catalog.Glass.Mask()},
			nChoice: 2}, 0},
	}
	for _, c := range cases {
		if got := oracleTrade(&c.in); got != c.want {
			t.Errorf("%s: oracle %d, want %d", c.name, got, c.want)
		}
	}
}

// Producers whose masks are not one class each never occur in the base game,
// but cheapest must still return a valid payment for them, never cheaper
// than the true optimum.
func TestCheapestArbitraryMasksStayValid(t *testing.T) {
	rng := NewRNG(77)
	for i := range 3000 {
		in := costStressInput(&rng)
		for j := range in.nChoice {
			in.choice[j] = catalog.ResMask(1 + rng.IntN(31))
		}
		var pay Payment
		cheapest(&in, &pay)
		if problems := costPaymentProblems(&in, &pay); len(problems) > 0 {
			t.Fatalf("case %d:\ninput   %+v\npayment %+v\n%v", i, in, pay, problems)
		}
		if !in.chainFree && int(pay.TradeTotal) < oracleTrade(&in) {
			t.Fatalf("case %d: trade %d below the optimum %d", i, pay.TradeTotal, oracleTrade(&in))
		}
	}
}
