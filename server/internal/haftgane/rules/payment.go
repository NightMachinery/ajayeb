package rules

import "github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"

// PaymentSource and Producer identify a physical production symbol. Fixed
// producer indices are local to (source, resource); choice indices are local to
// the source's choice array. Neither symbol is consumed after this construction.
type PaymentSource uint8

const (
	OwnFixed PaymentSource = iota + 1
	OwnChoice
	LeftFixed
	LeftChoice
	RightFixed
	RightChoice
)

type PaymentUnit struct {
	Resource catalog.Resource
	Source   PaymentSource
	Producer uint8
	Price    uint8 // zero for own production; otherwise the payer's price
}

// Payment contains one canonical covering, in resource then source order.
// The pair, not the covering, identifies an offered construction.
type Payment struct {
	Left, Right uint8
	Units       uint8
	Witness     [catalog.MaxCostUnits]PaymentUnit
}

type Payments struct {
	Printed  uint8
	Count    uint8
	Outcomes [MaxPaymentOutcomes]Payment
}

// Quote describes an ordinary cost, including printed coins. An unaffordable
// quote is zero-valued. Free-by-chain/ability checks belong to the action before
// this solver; a stage must always use its ordinary cost.
type Quote struct {
	Total      uint16
	Affordable bool
}

type tradeInput struct {
	purse       int16
	printed     uint8
	need        [catalog.NumResources]uint8
	own         [catalog.NumResources]uint8
	ownChoice   [MaxChoice]catalog.ResMask
	nOwnChoice  uint8
	sell        [2][catalog.NumResources]uint8
	sellChoice  [2][MaxForSaleChoice]catalog.ResMask
	nSellChoice [2]uint8
	price       [2][catalog.NumResources]uint8
}

// PaymentsState reads only the frozen public production caches and purse.
func PaymentsState(st *State, seat Seat, cost catalog.Cost) Payments {
	in, ok := tradeFromState(st, seat, cost)
	if !ok {
		return Payments{}
	}
	return solvePayment(in)
}

// QuoteState is the allocation-free bot-facing ordinary quote.
func QuoteState(st *State, seat Seat, cost catalog.Cost) Quote {
	in, ok := tradeFromState(st, seat, cost)
	if !ok {
		return Quote{}
	}
	return quotePayment(in)
}

func (g *Game) Quote(seat Seat, cost catalog.Cost) Quote {
	if g == nil {
		return Quote{}
	}
	return QuoteState(&g.State, seat, cost)
}

func tradeFromState(st *State, seat Seat, cost catalog.Cost) (tradeInput, bool) {
	if st == nil || !seat.Valid(st.Seats) {
		return tradeInput{}, false
	}
	own := &st.Cities[seat]
	in := tradeInput{purse: own.Coins, printed: cost.Coins, need: cost.Res, own: own.Produce, ownChoice: own.Choice, nOwnChoice: own.NChoice}
	for side, s := range [2]Seat{seat.Left(st.Seats), seat.Right(st.Seats)} {
		city := &st.Cities[s]
		in.sell[side] = city.ForSale
		in.sellChoice[side] = city.ChoiceForSale
		in.nSellChoice[side] = city.NChoiceForSale
		for r := catalog.Resource(0); r < catalog.NumResources; r++ {
			in.price[side][r] = 2
			if own.Discount[side].Has(r) {
				in.price[side][r] = 1
			}
		}
	}
	return in, true
}

func (in *tradeInput) valid() bool {
	if in.purse < int16(in.printed) || in.nOwnChoice > MaxChoice || in.nSellChoice[0] > MaxForSaleChoice || in.nSellChoice[1] > MaxForSaleChoice {
		return false
	}
	var total uint16
	for r, n := range in.need {
		total += uint16(n)
		for s := 0; s < 2; s++ {
			if in.price[s][r] < 1 || in.price[s][r] > 2 {
				return false
			}
		}
	}
	if total > catalog.MaxCostUnits {
		return false
	}
	for _, m := range in.ownChoice[:in.nOwnChoice] {
		if !m.Valid() {
			return false
		}
	}
	for s := 0; s < 2; s++ {
		for _, m := range in.sellChoice[s][:in.nSellChoice[s]] {
			if !m.Valid() {
				return false
			}
		}
	}
	return true
}

type paymentSearch struct {
	in                    *tradeInput
	remain, fixed         [catalog.NumResources]uint8
	own, left, right      [catalog.NumResources]uint8
	available             [2][catalog.NumResources]uint8
	suffix                [catalog.NumResources + 1]uint8
	maxOwn                uint8
	bestRight             [MaxPaymentOutcomes]uint8
	pairs                 [MaxPaymentOutcomes]Payment
	quoteOnly             bool
	bestTotal, lowerBound uint16
}

func newPaymentSearch(in *tradeInput, quoteOnly bool) paymentSearch {
	s := paymentSearch{in: in, quoteOnly: quoteOnly, bestTotal: 65535}
	for j := range s.bestRight {
		s.bestRight[j] = 255
	}
	// A fixed own symbol can cover only its resource. Taking it first cannot
	// remove any other covering; leave flexible symbols for the remaining units.
	for r, n := range in.need {
		s.fixed[r] = min(n, in.own[r])
		s.remain[r] = n - s.fixed[r]
		for side := 0; side < 2; side++ {
			a := min(n, in.sell[side][r])
			for _, m := range in.sellChoice[side][:in.nSellChoice[side]] {
				if m.Has(catalog.Resource(r)) {
					a++
				}
			}
			s.available[side][r] = a
		}
	}
	for r := int(catalog.NumResources) - 1; r >= 0; r-- {
		s.suffix[r] = s.suffix[r+1] + s.remain[r]
	}
	s.maxOwn, _ = matchProduction(s.remain, [catalog.NumResources]uint8{}, in.ownChoice[:in.nOwnChoice])
	s.lowerBound = uint16(in.printed) + uint16(s.suffix[0]-s.maxOwn)
	return s
}

func solvePayment(in tradeInput) Payments {
	out := Payments{Printed: in.printed}
	if !in.valid() {
		return out
	}
	s := newPaymentSearch(&in, false)
	s.enumerate(0, 0, 0, 0)
	// One minimal right amount per left amount. Scanning left in ascending order
	// drops precisely those pairs dominated by an earlier left amount.
	best := uint8(255)
	for l, r := range s.bestRight {
		if r >= best {
			continue
		}
		best = r
		p := s.pairs[l]
		j := out.Count
		for j > 0 {
			prior := out.Outcomes[j-1]
			if uint16(prior.Left)+uint16(prior.Right) <= uint16(p.Left)+uint16(p.Right) {
				break
			}
			out.Outcomes[j] = prior
			j--
		}
		out.Outcomes[j] = p
		out.Count++
	}
	return out
}

func quotePayment(in tradeInput) Quote {
	if !in.valid() {
		return Quote{}
	}
	s := newPaymentSearch(&in, true)
	if s.lowerBound > uint16(in.purse) {
		return Quote{}
	}
	s.enumerate(0, 0, 0, 0)
	if s.bestTotal == 65535 {
		return Quote{}
	}
	return Quote{Total: s.bestTotal, Affordable: true}
}

// Keep only maximum own coverage. A smaller own matching has an augmenting
// path that adds a demanded unit while retaining EVERY previously matched
// demanded unit (the symbols can change). That added unit was bought. Leave
// all other bought units with their original sellers: those sales remain
// feasible, neither payment increases, and one strictly decreases because all
// prices are positive. Repeat to maximum own coverage. Therefore every omitted
// covering is dominated by a maximum-own covering, including under a purse cap.
func (s *paymentSearch) enumerate(r catalog.Resource, covered, left, right uint8) {
	if covered > s.maxOwn || covered+s.suffix[r] < s.maxOwn {
		return
	}
	total := uint16(s.in.printed) + uint16(left) + uint16(right)
	if total > uint16(s.in.purse) || (s.quoteOnly && (total >= s.bestTotal || s.bestTotal == s.lowerBound)) {
		return
	}
	if r == catalog.NumResources {
		s.accept(left, right)
		return
	}
	k := s.remain[r]
	for own := uint8(0); own <= k; own++ {
		s.own[r] = own
		for l := uint8(0); l <= k-own; l++ {
			rr := k - own - l
			if l > s.available[0][r] || rr > s.available[1][r] {
				continue
			}
			s.left[r] = l
			s.right[r] = rr
			s.enumerate(r+1, covered+own, left+l*s.in.price[0][r], right+rr*s.in.price[1][r])
		}
	}
}

func (s *paymentSearch) accept(left, right uint8) {
	if !s.quoteOnly && right >= s.bestRight[left] {
		return
	}
	ownCount, ownMatch := matchProduction(s.own, [catalog.NumResources]uint8{}, s.in.ownChoice[:s.in.nOwnChoice])
	if ownCount != s.maxOwn {
		return
	}
	leftCount, leftMatch := matchProduction(s.left, s.in.sell[0], s.in.sellChoice[0][:s.in.nSellChoice[0]])
	rightCount, rightMatch := matchProduction(s.right, s.in.sell[1], s.in.sellChoice[1][:s.in.nSellChoice[1]])
	var nl, nr uint8
	for r := range s.left {
		nl += s.left[r]
		nr += s.right[r]
	}
	if leftCount != nl || rightCount != nr {
		return
	}
	total := uint16(s.in.printed) + uint16(left) + uint16(right)
	s.bestTotal = min(s.bestTotal, total)
	if s.quoteOnly {
		return
	}
	s.bestRight[left] = right
	p := Payment{Left: left, Right: right}
	var oi, li, ri uint8
	for r := catalog.Resource(0); r < catalog.NumResources; r++ {
		for k := uint8(0); k < s.fixed[r]; k++ {
			p.add(r, OwnFixed, k, 0)
		}
		for k := uint8(0); k < s.own[r]; k++ {
			p.add(r, OwnChoice, ownMatch[oi], 0)
			oi++
		}
		for k := uint8(0); k < s.left[r]; k++ {
			if k < s.in.sell[0][r] {
				p.add(r, LeftFixed, k, s.in.price[0][r])
			} else {
				p.add(r, LeftChoice, leftMatch[li], s.in.price[0][r])
			}
			li++
		}
		for k := uint8(0); k < s.right[r]; k++ {
			if k < s.in.sell[1][r] {
				p.add(r, RightFixed, k, s.in.price[1][r])
			} else {
				p.add(r, RightChoice, rightMatch[ri], s.in.price[1][r])
			}
			ri++
		}
	}
	s.pairs[left] = p
}

func (p *Payment) add(r catalog.Resource, source PaymentSource, producer, price uint8) {
	p.Witness[p.Units] = PaymentUnit{Resource: r, Source: source, Producer: producer, Price: price}
	p.Units++
}

// matchProduction uses fixed symbols first, then deterministic augmenting paths
// in unit/resource order and producer-index order. It returns maximum coverage
// and a producer index for each demanded unit (255 for fixed or unmatched).
func matchProduction(need, fixed [catalog.NumResources]uint8, choices []catalog.ResMask) (uint8, [catalog.MaxCostUnits]uint8) {
	m := productionMatch{choices: choices}
	for p := range m.owner {
		m.owner[p] = -1
	}
	var count uint8
	for r, n := range need {
		for k := uint8(0); k < n; k++ {
			m.resource[m.units] = catalog.Resource(r)
			m.units++
			if k < fixed[r] {
				count++
				continue
			}
			m.seen = 0
			if m.augment(m.units - 1) {
				count++
			}
		}
	}
	var assigned [catalog.MaxCostUnits]uint8
	for u := range assigned {
		assigned[u] = 255
	}
	for p, u := range m.owner[:len(choices)] {
		if u >= 0 {
			assigned[u] = uint8(p)
		}
	}
	return count, assigned
}

type productionMatch struct {
	choices  []catalog.ResMask
	resource [catalog.MaxCostUnits]catalog.Resource
	owner    [MaxChoice]int8
	units    uint8
	seen     uint16
}

func (m *productionMatch) augment(unit uint8) bool {
	for p, mask := range m.choices {
		bit := uint16(1) << p
		if m.seen&bit != 0 || !mask.Has(m.resource[unit]) {
			continue
		}
		m.seen |= bit
		if m.owner[p] < 0 || m.augment(uint8(m.owner[p])) {
			m.owner[p] = int8(unit)
			return true
		}
	}
	return false
}
