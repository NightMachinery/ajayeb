package rules

import (
	"reflect"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
)

func baseTrade() tradeInput {
	in := tradeInput{purse: 30}
	for side := range in.price {
		for r := range in.price[side] {
			in.price[side][r] = 2
		}
	}
	return in
}

// Cases follow rules.md Paying costs and rulings 9/10, not a matching algorithm.
func TestPaymentRules(t *testing.T) {
	tests := []struct {
		name string
		edit func(*tradeInput)
		want [][2]uint8
	}{
		{"free cost", func(*tradeInput) {}, [][2]uint8{{0, 0}}},
		{"printed coins frozen purse", func(i *tradeInput) { i.printed = 3; i.purse = 2 }, nil},
		{"own fixed is free", func(i *tradeInput) { i.need[0] = 2; i.own[0] = 2 }, [][2]uint8{{0, 0}}},
		{"missing resource", func(i *tradeInput) { i.need[0] = 1 }, nil},
		{"either neighbour", func(i *tradeInput) { i.need[0] = 2; i.sell[0][0] = 2; i.sell[1][0] = 2 }, [][2]uint8{{0, 4}, {2, 2}, {4, 0}}},
		{"discount removes beaten outcomes", func(i *tradeInput) {
			i.need[0] = 1
			i.need[1] = 1
			i.sell[0][0] = 1
			i.sell[0][1] = 1
			i.sell[1][0] = 1
			i.sell[1][1] = 1
			i.price[0][0] = 1
			i.price[1][1] = 1
		}, [][2]uint8{{1, 1}, {0, 3}, {3, 0}}},
		{"printed plus trade must fit", func(i *tradeInput) { i.need[0] = 1; i.sell[0][0] = 1; i.printed = 1; i.purse = 2 }, nil},
		{"one choice buys only once", func(i *tradeInput) { i.need[0] = 1; i.need[1] = 1; i.nSellChoice[0] = 1; i.sellChoice[0][0] = 3 }, nil},
		{"two identical choice symbols", func(i *tradeInput) {
			i.need[0] = 2
			i.nSellChoice[0] = 2
			i.sellChoice[0][0] = 3
			i.sellChoice[0][1] = 3
		}, [][2]uint8{{4, 0}}},
		{"own choice avoids trade", func(i *tradeInput) {
			i.need[0] = 1
			i.nOwnChoice = 1
			i.ownChoice[0] = 3
			i.sell[0][0] = 1
			i.sell[1][0] = 1
		}, [][2]uint8{{0, 0}}},
		{"flexible own cover keeps rival choices", func(i *tradeInput) {
			i.need[0] = 1
			i.need[1] = 1
			i.nOwnChoice = 1
			i.ownChoice[0] = 3
			i.sell[0][0] = 1
			i.sell[1][1] = 1
		}, [][2]uint8{{0, 2}, {2, 0}}},
		{"matching needs reassignment", func(i *tradeInput) {
			i.need[0] = 1
			i.need[1] = 1
			i.nOwnChoice = 2
			i.ownChoice[0] = 3
			i.ownChoice[1] = 1
		}, [][2]uint8{{0, 0}}},
		{"own fixed first preserves flexible", func(i *tradeInput) { i.need[0] = 1; i.need[1] = 1; i.own[0] = 1; i.nOwnChoice = 1; i.ownChoice[0] = 3 }, [][2]uint8{{0, 0}}},
		{"negative purse", func(i *tradeInput) { i.purse = -1 }, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := baseTrade()
			tt.edit(&in)
			got := solvePayment(in)
			var pairs [][2]uint8
			for j := uint8(0); j < got.Count; j++ {
				p := got.Outcomes[j]
				pairs = append(pairs, [2]uint8{p.Left, p.Right})
				validatePaymentWitness(t, in, p)
			}
			if !reflect.DeepEqual(pairs, tt.want) {
				t.Fatalf("pairs %v, want %v", pairs, tt.want)
			}
			if got != solvePayment(in) {
				t.Fatal("noncanonical witness")
			}
			q := quotePayment(in)
			if q.Affordable != (got.Count > 0) || (got.Count > 0 && q.Total != uint16(in.printed)+uint16(pairs[0][0])+uint16(pairs[0][1])) {
				t.Fatalf("quote %+v disagrees with %+v", q, got)
			}
		})
	}
}

func TestPaymentStateAdapter(t *testing.T) {
	st := State{Seats: 3}
	c := catalog.Cost{Coins: 1}
	c.Res[catalog.Wood] = 1
	c.Res[catalog.Glass] = 1
	st.Cities[0].Coins = 4
	// Own yellow/wonder production is usable, but never read as a neighbour's saleable output.
	st.Cities[1].Produce[catalog.Wood] = 9
	st.Cities[1].Choice[0] = catalog.Manufactured
	st.Cities[1].NChoice = 1
	if PaymentsState(&st, 0, c).Count != 0 {
		t.Fatal("bought non-saleable output")
	}
	st.Cities[1].ForSale[catalog.Wood] = 1
	st.Cities[2].ForSale[catalog.Glass] = 1
	st.Cities[0].Discount[0] = catalog.Raw
	ps := PaymentsState(&st, 0, c)
	if ps.Count != 1 || ps.Printed != 1 || ps.Outcomes[0].Left != 1 || ps.Outcomes[0].Right != 2 {
		t.Fatalf("adapter %+v", ps)
	}
	q := QuoteState(&st, 0, c)
	if q != (Quote{Affordable: true, Total: 4}) {
		t.Fatalf("quote %+v", q)
	}
	if st.Cities[0].Coins != 4 {
		t.Fatal("quote mutated frozen state")
	}
	st.Cities[0].Coins = 3
	if QuoteState(&st, 0, c).Affordable {
		t.Fatal("spent future income")
	}
	st.Cities[0].Produce[catalog.Wood] = 1
	st.Cities[0].Choice[0] = catalog.Manufactured
	st.Cities[0].NChoice = 1
	if q = QuoteState(&st, 0, c); q != (Quote{Affordable: true, Total: 1}) {
		t.Fatal(q)
	}
	if QuoteState(nil, 0, c).Affordable || QuoteState(&st, NoSeat, c).Affordable || PaymentsState(&st, 3, c).Count != 0 {
		t.Fatal("invalid state/seat")
	}
}

func TestPaymentBoundsAndAllocation(t *testing.T) {
	in := baseTrade()
	in.need[0] = 7
	in.sell[0][0] = 7
	in.sell[1][0] = 7
	// Seven units at uniform price two yield eight incomparable outcomes.
	if got := solvePayment(in); got.Count != 8 {
		t.Fatal(got.Count)
	}
	if n := testing.AllocsPerRun(1000, func() { _ = solvePayment(in); _ = quotePayment(in) }); n != 0 {
		t.Fatalf("allocations %v", n)
	}
	in.need[1] = 1
	if solvePayment(in).Count != 0 || quotePayment(in).Affordable {
		t.Fatal("accepted cost above seven")
	}
	for _, edit := range []func(*tradeInput){func(i *tradeInput) { i.nOwnChoice = MaxChoice + 1 }, func(i *tradeInput) { i.nSellChoice[0] = MaxForSaleChoice + 1 }, func(i *tradeInput) { i.price[0][0] = 0 }, func(i *tradeInput) { i.nOwnChoice = 1; i.ownChoice[0] = 128 }} {
		bad := baseTrade()
		edit(&bad)
		if solvePayment(bad).Count != 0 {
			t.Fatal("accepted invalid input")
		}
	}
}
