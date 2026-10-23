package rules

import (
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
	"testing"
)

var paymentSink Payments
var quoteSink Quote

func typicalPayment() tradeInput {
	in := baseTrade()
	in.purse = 7
	in.need[catalog.Wood] = 2
	in.need[catalog.Glass] = 1
	in.own[catalog.Wood] = 1
	in.nOwnChoice = 1
	in.ownChoice[0] = catalog.Wood.Mask() | catalog.Clay.Mask()
	in.sell[0][catalog.Glass] = 1
	in.sell[1][catalog.Glass] = 1
	in.price[0][catalog.Glass] = 1
	return in
}
func worstPayment() tradeInput {
	in := baseTrade()
	for r := range in.need {
		in.need[r] = 1
		in.price[0][r] = uint8(1 + r%2)
		in.price[1][r] = uint8(2 - r%2)
	}
	in.nOwnChoice = 3
	for p := range in.nOwnChoice {
		in.ownChoice[p] = catalog.AllResources
	}
	for s := 0; s < 2; s++ {
		in.nSellChoice[s] = MaxForSaleChoice
		for p := range in.nSellChoice[s] {
			in.sellChoice[s][p] = catalog.AllResources
		}
	}
	return in
}
func BenchmarkPaymentTypical(b *testing.B) {
	in := typicalPayment()
	b.ReportAllocs()
	for b.Loop() {
		paymentSink = solvePayment(in)
	}
}
func BenchmarkPaymentWorst(b *testing.B) {
	in := worstPayment()
	b.ReportAllocs()
	for b.Loop() {
		paymentSink = solvePayment(in)
	}
}
func BenchmarkQuoteTypical(b *testing.B) {
	in := typicalPayment()
	b.ReportAllocs()
	for b.Loop() {
		quoteSink = quotePayment(in)
	}
}
func BenchmarkQuoteWorst(b *testing.B) {
	in := worstPayment()
	b.ReportAllocs()
	for b.Loop() {
		quoteSink = quotePayment(in)
	}
}
func BenchmarkQuoteState(b *testing.B) {
	st := State{Seats: 3}
	st.Cities[0].Coins = 7
	st.Cities[0].Produce[catalog.Wood] = 1
	st.Cities[1].ForSale[catalog.Glass] = 1
	c := catalog.Cost{}
	c.Res[catalog.Wood] = 1
	c.Res[catalog.Glass] = 1
	b.ReportAllocs()
	for b.Loop() {
		quoteSink = QuoteState(&st, 0, c)
	}
}

// Every resource appears available in isolation, but the six shared seller
// symbols cannot supply seven units. This exercises a failing full search.
func impossiblePayment() tradeInput {
	in := worstPayment()
	in.nOwnChoice = 0
	in.nSellChoice = [2]uint8{3, 3}
	return in
}
func BenchmarkPaymentImpossible(b *testing.B) {
	in := impossiblePayment()
	b.ReportAllocs()
	for b.Loop() {
		paymentSink = solvePayment(in)
	}
}
func BenchmarkQuoteImpossible(b *testing.B) {
	in := impossiblePayment()
	b.ReportAllocs()
	for b.Loop() {
		quoteSink = quotePayment(in)
	}
}
