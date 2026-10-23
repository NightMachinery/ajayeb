package rules

import (
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
)

func FuzzPayment(f *testing.F) {
	for _, seed := range [][]byte{{}, {1, 2, 3}, {255, 128, 64, 32, 16, 8, 4, 2, 1}, make([]byte, 128), {207, 207, 207}, {207, 207, 207, 207, 56}} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		at := 0
		next := func() byte {
			if len(data) == 0 {
				return 0
			}
			b := data[at%len(data)]
			at++
			return b
		}
		in := baseTrade()
		in.purse = int16(next() % 30)
		in.printed = next() % 4
		for k := next() % 8; k > 0; k-- {
			in.need[next()%7]++
		}
		for r := range in.need {
			in.own[r] = next() % 4
			for s := 0; s < 2; s++ {
				in.sell[s][r] = next() % 4
				in.price[s][r] = 1 + next()%2
			}
		}
		in.nOwnChoice = next() % (MaxChoice + 1)
		for p := range in.nOwnChoice {
			in.ownChoice[p] = 1 + catalog.ResMask(next()%127)
		}
		for s := 0; s < 2; s++ {
			in.nSellChoice[s] = next() % (MaxForSaleChoice + 1)
			for p := range in.nSellChoice[s] {
				in.sellChoice[s][p] = 1 + catalog.ResMask(next()%127)
			}
		}
		checkPaymentOracle(t, in)
		first, second := solvePayment(in), solvePayment(in)
		if first != second {
			t.Fatal("noncanonical witness")
		}
	})
}
