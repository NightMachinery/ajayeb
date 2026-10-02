package engine

import "github.com/NightMachinery/ajayeb/server/internal/engine/catalog"

// This file computes what a construction costs: the cheapest payment for a
// building or a wonder, its full breakdown for the view, and the discard
// value. The player never chooses how to pay; the engine always applies the
// cheapest payment (see docs/rules/economy.md, "The cheapest payment").
//
// The work is split in two layers:
//
//   - cheapest, the pure core. It reads a costInput (the printed cost, the
//     payer's production, the opponent's production and the token rules that
//     apply) and needs no State and no catalog, so it is tested directly
//     against an exhaustive search (cost_oracle_test.go). It never allocates.
//   - the adapters (BuildingPayment, WonderPayment, BuildingQuote,
//     WonderQuote, TradePrices, DiscardValue). They fill a costInput from the
//     Player caches that city.go maintains and from the catalog, and name the
//     cards and wonders behind each part of the breakdown.
//
// The algorithm, in the order cheapest applies it:
//
//  1. A building whose chain symbol the payer owns is free (and Urbanism pays
//     its bonus). Nothing else is computed. Wonders never chain.
//  2. The printed coins are paid as they are.
//  3. The payer's own brown and grey production covers what it can.
//  4. Each unit still needed has a price: 1 coin when the payer fixes the
//     price of that resource (a reserve, the Customs House), otherwise 2 plus
//     the opponent's brown and grey production of it.
//  5. Each flexible producer covers the dearest remaining unit it can make.
//  6. A discount (Masonry for blue buildings, Architecture for wonders)
//     waives the dearest remaining units, up to its size. An unused part is
//     lost, never refunded.
//  7. The rest is bought. When the opponent owns Economy, the coins paid for
//     those units go to the opponent; the printed coins always go to the bank.
//
// Why the greedy steps 5 and 6 are optimal: in the base game every flexible
// producer makes exactly the raw goods or exactly the manufactured goods
// (catalog.RawGoods, catalog.Manufactured), so producers of one class are
// interchangeable, and the best use of k producers of a class is to cover the
// k dearest units of that class. The discount may take any resource, so after
// the producers it simply takes the dearest units left. Every price is at
// least 1, so leaving a producer or a discount unit idle while a unit it
// could cover is still bought never helps. The catalog's derived tables are
// meant to assert the class precondition when they load (slice S2a);
// TestCostChoiceMasksAreClasses checks it from the engine's side once the
// data exists, and TestCheapestOracle compares cheapest with an exhaustive
// search over every way to cover every unit.
//
// Ties between equally priced units go to the lower resource index (wood,
// clay, stone, glass, papyrus), so the breakdown is deterministic. The total
// does not depend on how ties are broken.

// Trade and discard constants of the rules.
const (
	baseTradePrice  = 2 // a bought unit costs this plus the opponent's brown and grey production of it
	fixedTradePrice = 1 // the price of a unit whose price the payer fixes
	discardBase     = 2 // coins for discarding a card, plus one per yellow card in the discarder's city
	maxInt16        = 1<<15 - 1
)

// ChoiceUse is one flexible producer used in a payment.
type ChoiceUse struct {
	Card   catalog.CardID   // the yellow card that made the unit, or 0
	Wonder catalog.WonderID // the built wonder that made it, or 0
	Mask   catalog.ResMask  // the resources the producer can make
	Res    catalog.Resource // the resource it covered
}

// Payment is the cheapest way for a player to pay for a building or a wonder
// right now, with its full breakdown. The turn machine applies it and the
// view shows it; the client never computes a cost itself.
//
// Unless ChainFree, each resource unit of the printed cost is covered exactly
// once: Own[r] + (the uses in Choice with Res r) + Discount[r] + TradeUnits[r]
// equals the printed units of r. Total is always PrintedCoins + TradeTotal.
type Payment struct {
	Affordable bool           // the payer has at least Total coins (always true when ChainFree)
	ChainFree  bool           // free through a chain: no coins, no resources
	ChainFrom  catalog.CardID // when ChainFree, the card in the payer's city that provides the chain symbol

	PrintedCoins int16                       // the printed coins, paid to the bank (0 when ChainFree)
	Own          [catalog.NumResources]uint8 // units covered by the payer's own brown and grey production

	Choice  [MaxChoice]ChoiceUse // flexible producers used, Choice[:NChoice]
	NChoice uint8

	Discount   [catalog.NumResources]uint8 // units waived by Masonry or Architecture
	DiscountBy catalog.TokenID             // the token whose discount applies to this item, or 0 (set even when nothing was left to waive)

	TradeUnits [catalog.NumResources]uint8 // units bought
	UnitPrice  [catalog.NumResources]uint8 // the payer's price per unit of each resource (whether or not any is bought)
	FixedPrice catalog.ResMask             // resources whose UnitPrice the payer fixes at 1 coin

	TradeTotal    int16 // coins paid for the bought units
	Total         int16 // PrintedCoins + TradeTotal: every coin the payer spends
	ToOpponent    int16 // the part of Total that goes to the opponent: TradeTotal when the opponent owns Economy, else 0
	UrbanismBonus int16 // coins the payer will receive for a chain-free build (Urbanism); 0 otherwise
}

// Quote is the short form of a Payment: what the turn machine and the bots
// need to apply or rule out a construction. BuildingQuote and WonderQuote
// compute it without the breakdown, which makes them the cheaper calls for
// hot paths such as legality checks and playouts.
type Quote struct {
	Total      int16 // every coin the payer spends
	ToOpponent int16 // the part of Total that goes to the opponent (Economy)
	ChainFree  bool  // free through a chain (Urbanism applies)
	Affordable bool  // the payer has at least Total coins
}

// Quote returns the short form of the payment.
func (p *Payment) Quote() Quote {
	return Quote{Total: p.Total, ToOpponent: p.ToOpponent, ChainFree: p.ChainFree, Affordable: p.Affordable}
}

// costInput is everything cheapest needs, with no reference to a State or to
// the catalog.
type costInput struct {
	purse   int16                       // the payer's coins, which decide Affordable
	printed int16                       // the printed coins
	need    [catalog.NumResources]uint8 // the printed resource units
	own     [catalog.NumResources]uint8 // the payer's fixed (brown and grey) production
	oppProd [catalog.NumResources]uint8 // the opponent's fixed production, which sets the payer's prices
	fix     catalog.ResMask             // resources the payer buys at the fixed price

	choice  [MaxChoice]catalog.ResMask // the payer's flexible producers, choice[:nChoice]
	nChoice uint8

	discount    uint8 // units a discount may waive (2 with Masonry for a blue building or Architecture for a wonder)
	chainFree   bool  // the payer owns the item's chain symbol: the build is free
	chainBonus  uint8 // coins the payer receives for a chain-free build (Urbanism)
	tradeIncome bool  // the opponent owns Economy: the trading coins go to the opponent
}

// cheapest computes the cheapest payment for in. It returns the coins the
// payer spends in total and the trading part of them. When out is not nil it
// also writes the full breakdown there; the card and wonder names of the
// flexible producers (ChoiceUse.Card and .Wonder), ChainFrom and DiscountBy
// are left zero for the adapters to fill. It never allocates.
func cheapest(in *costInput, out *Payment) (total, trade int16) {
	price := unitPrices(in.fix, &in.oppProd)
	if out != nil {
		*out = Payment{UnitPrice: price, FixedPrice: in.fix & catalog.AllResources}
	}
	if in.chainFree {
		if out != nil {
			out.ChainFree = true
			out.Affordable = true
			out.UrbanismBonus = int16(in.chainBonus)
		}
		return 0, 0
	}

	// Own production first: it is free and good only for its own resource.
	var rem [catalog.NumResources]uint8
	for r := range catalog.NumResources {
		own := min(in.need[r], in.own[r])
		rem[r] = in.need[r] - own
		if out != nil {
			out.Own[r] = own
		}
	}

	// Each flexible producer covers the dearest unit it can make.
	for i := range min(int(in.nChoice), MaxChoice) {
		m := in.choice[i]
		r, ok := dearest(&rem, &price, m)
		if !ok {
			continue
		}
		rem[r]--
		if out != nil {
			out.Choice[out.NChoice] = ChoiceUse{Mask: m, Res: r}
			out.NChoice++
		}
	}

	// The discount waives the dearest units left, of any resource.
	for range in.discount {
		r, ok := dearest(&rem, &price, catalog.AllResources)
		if !ok {
			break
		}
		rem[r]--
		if out != nil {
			out.Discount[r]++
		}
	}

	// Buy the rest.
	t := 0
	for r := range catalog.NumResources {
		t += int(rem[r]) * int(price[r])
	}
	trade = sat16(t)
	total = sat16(int(in.printed) + t)
	if out != nil {
		out.PrintedCoins = in.printed
		out.TradeUnits = rem
		out.TradeTotal = trade
		out.Total = total
		out.Affordable = total <= in.purse
		if in.tradeIncome {
			out.ToOpponent = trade
		}
	}
	return total, trade
}

// quote runs cheapest without a breakdown and summarizes the result.
func quote(in *costInput) Quote {
	total, trade := cheapest(in, nil)
	q := Quote{Total: total, ChainFree: in.chainFree, Affordable: in.chainFree || total <= in.purse}
	if in.tradeIncome {
		q.ToOpponent = trade
	}
	return q
}

// dearest returns the resource in mask with at least one unit left in rem and
// the highest price, ties going to the lower resource index. It reports false
// when mask has no unit left.
func dearest(rem, price *[catalog.NumResources]uint8, mask catalog.ResMask) (catalog.Resource, bool) {
	best, bestPrice := catalog.Resource(0), uint8(0)
	for r := range catalog.NumResources {
		if rem[r] > 0 && mask.Has(r) && price[r] > bestPrice {
			best, bestPrice = r, price[r]
		}
	}
	return best, bestPrice > 0
}

// unitPrices returns the payer's price per unit of each resource: the fixed
// price for the resources in fix, otherwise the base price plus the
// opponent's fixed production of that resource. Every price is at least 1.
func unitPrices(fix catalog.ResMask, oppProd *[catalog.NumResources]uint8) [catalog.NumResources]uint8 {
	var price [catalog.NumResources]uint8
	for r := range catalog.NumResources {
		if fix.Has(r) {
			price[r] = fixedTradePrice
		} else {
			price[r] = uint8(min(baseTradePrice+int(oppProd[r]), 255))
		}
	}
	return price
}

// sat16 converts a non-negative coin amount to int16, saturating. Real costs
// stay far below the limit; this only keeps a malformed scenario from
// wrapping around.
func sat16(v int) int16 {
	if v > maxInt16 {
		return maxInt16
	}
	return int16(v)
}

// payRules are the progress-token rules that change a payment, gathered from
// the effects of the tokens a player owns.
type payRules struct {
	discount    [catalog.ForWonders + 1]uint8           // units waived, by discount target
	discountBy  [catalog.ForWonders + 1]catalog.TokenID // the first token giving each discount
	tradeIncome bool                                    // the owner receives the opponent's trading coins (Economy)
	chainBonus  uint8                                   // coins for each chain-free build (Urbanism)
}

// payRulesOf reads the payment rules of the tokens in ts from the catalog.
// Tokens that are not registered contribute nothing.
func payRulesOf(ts catalog.TokenSet) payRules {
	var pr payRules
	for id, ok := ts.Next(1); ok; id, ok = ts.Next(id + 1) {
		d := catalog.Token(id)
		if d == nil {
			continue
		}
		for _, e := range d.Effects {
			switch e.Kind {
			case catalog.EffDiscount:
				if e.Target == catalog.ForBlue || e.Target == catalog.ForWonders {
					pr.discount[e.Target] = uint8(min(int(pr.discount[e.Target])+int(e.N), 255))
					if pr.discountBy[e.Target] == 0 {
						pr.discountBy[e.Target] = id
					}
				}
			case catalog.EffTradeIncome:
				pr.tradeIncome = true
			case catalog.EffChainBonus:
				pr.chainBonus = uint8(min(int(pr.chainBonus)+int(e.N), 255))
			}
		}
	}
	return pr
}

// setPayers fills the parts of in that depend on the players but not on the
// item: the payer's purse, production and fixed prices, the opponent's
// production and Economy. It returns the payer's own token rules.
func (in *costInput) setPayers(s *State, seat Seat) payRules {
	p, o := &s.Players[seat], &s.Players[seat.Opp()]
	in.purse = p.Coins
	in.own = p.Prod
	in.oppProd = o.Prod
	in.fix = p.TradeFix
	in.choice = p.Choice
	in.nChoice = min(p.NChoice, MaxChoice)
	in.tradeIncome = payRulesOf(o.Tokens).tradeIncome
	return payRulesOf(p.Tokens)
}

// buildingInput fills in for seat constructing the building d. It returns the
// token whose discount applies, or 0.
func buildingInput(s *State, seat Seat, d *catalog.CardDef, in *costInput) catalog.TokenID {
	pr := in.setPayers(s, seat)
	if d.ChainIn != catalog.NoChain && s.Players[seat].Chains&d.ChainIn.Bit() != 0 {
		in.chainFree = true
		in.chainBonus = pr.chainBonus
		return 0
	}
	in.printed = int16(d.Cost.Coins)
	in.need = d.Cost.Res
	if d.Color == catalog.Blue {
		in.discount = pr.discount[catalog.ForBlue]
		return pr.discountBy[catalog.ForBlue]
	}
	return 0
}

// wonderInput fills in for seat building the wonder d. Wonders never chain.
// It returns the token whose discount applies, or 0.
func wonderInput(s *State, seat Seat, d *catalog.WonderDef, in *costInput) catalog.TokenID {
	pr := in.setPayers(s, seat)
	in.printed = int16(d.Cost.Coins)
	in.need = d.Cost.Res
	in.discount = pr.discount[catalog.ForWonders]
	return pr.discountBy[catalog.ForWonders]
}

// BuildingPayment returns the cheapest payment for seat to construct the
// building c now, with its full breakdown. It does not check that the card is
// accessible or that seat is to move; that is legality. A card that is not
// registered (including catalog.Unknown) gives the zero Payment, which is not
// affordable.
func BuildingPayment(s *State, seat Seat, c catalog.CardID) Payment {
	var pay Payment
	d := catalog.Card(c)
	if d == nil {
		return pay
	}
	var in costInput
	by := buildingInput(s, seat, d, &in)
	cheapest(&in, &pay)
	p := &s.Players[seat]
	if pay.ChainFree {
		pay.ChainFrom = chainSource(p, d.ChainIn)
		return pay
	}
	if in.discount > 0 {
		pay.DiscountBy = by
	}
	nameChoiceSources(p, &pay)
	return pay
}

// WonderPayment returns the cheapest payment for seat to build the wonder w
// now, with its full breakdown. It does not check that w is one of seat's
// unbuilt wonders; that is legality. A wonder that is not registered gives
// the zero Payment, which is not affordable.
func WonderPayment(s *State, seat Seat, w catalog.WonderID) Payment {
	var pay Payment
	d := catalog.Wonder(w)
	if d == nil {
		return pay
	}
	var in costInput
	by := wonderInput(s, seat, d, &in)
	cheapest(&in, &pay)
	if in.discount > 0 {
		pay.DiscountBy = by
	}
	nameChoiceSources(&s.Players[seat], &pay)
	return pay
}

// BuildingQuote is BuildingPayment(s, seat, c).Quote() without the
// breakdown. An unregistered card gives the zero Quote, which is not
// affordable.
func BuildingQuote(s *State, seat Seat, c catalog.CardID) Quote {
	d := catalog.Card(c)
	if d == nil {
		return Quote{}
	}
	var in costInput
	buildingInput(s, seat, d, &in)
	return quote(&in)
}

// WonderQuote is WonderPayment(s, seat, w).Quote() without the breakdown. An
// unregistered wonder gives the zero Quote, which is not affordable.
func WonderQuote(s *State, seat Seat, w catalog.WonderID) Quote {
	d := catalog.Wonder(w)
	if d == nil {
		return Quote{}
	}
	var in costInput
	wonderInput(s, seat, d, &in)
	return quote(&in)
}

// TradePrices returns what seat pays per unit of each resource bought now:
// 1 coin for a resource whose price seat fixes, otherwise 2 plus the
// opponent's brown and grey production of it. The opponent's flexible
// producers never count.
func TradePrices(s *State, seat Seat) [catalog.NumResources]uint8 {
	return unitPrices(s.Players[seat].TradeFix, &s.Players[seat.Opp()].Prod)
}

// DiscardValue returns the coins seat receives for discarding a card now: 2
// plus one per yellow card in seat's city.
func DiscardValue(s *State, seat Seat) int16 {
	return discardBase + int16(s.Players[seat].ColorN[catalog.Yellow])
}

// chainSource returns the card in p's city that provides the chain symbol ch,
// or 0 if the catalog does not show one.
func chainSource(p *Player, ch catalog.Chain) catalog.CardID {
	for id, ok := p.City.Next(1); ok; id, ok = p.City.Next(id + 1) {
		if d := catalog.Card(id); d != nil && d.ChainOut == ch {
			return id
		}
	}
	return 0
}

// choiceSource is one flexible producer of a player, named.
type choiceSource struct {
	card   catalog.CardID
	wonder catalog.WonderID
	mask   catalog.ResMask
}

// choiceSources lists p's flexible producers in a canonical order: the cards
// of the city by ascending id, then the built wonders in slot order.
func choiceSources(p *Player) (srcs [MaxChoice]choiceSource, n int) {
	for id, ok := p.City.Next(1); ok && n < MaxChoice; id, ok = p.City.Next(id + 1) {
		d := catalog.Card(id)
		if d == nil {
			continue
		}
		if e, found := catalog.FindEffect(d.Effects, catalog.EffProduceChoice); found {
			srcs[n] = choiceSource{card: id, mask: e.Mask}
			n++
		}
	}
	for i := range p.Wonders {
		ws := &p.Wonders[i]
		if !ws.Built || n >= MaxChoice {
			continue
		}
		d := catalog.Wonder(ws.ID)
		if d == nil {
			continue
		}
		if e, found := catalog.FindEffect(d.Effects, catalog.EffProduceChoice); found {
			srcs[n] = choiceSource{wonder: ws.ID, mask: e.Mask}
			n++
		}
	}
	return srcs, n
}

// nameChoiceSources fills ChoiceUse.Card and .Wonder for the flexible
// producers pay uses. Producers with equal masks are interchangeable, so each
// use takes the first source with its mask that no earlier use has taken. A
// use with no matching source (the catalog does not show one) stays unnamed.
func nameChoiceSources(p *Player, pay *Payment) {
	if pay.NChoice == 0 {
		return
	}
	srcs, n := choiceSources(p)
	var taken uint8
	for k := range min(int(pay.NChoice), MaxChoice) {
		u := &pay.Choice[k]
		for j := range n {
			if taken&(1<<j) == 0 && srcs[j].mask == u.Mask {
				taken |= 1 << j
				u.Card, u.Wonder = srcs[j].card, srcs[j].wonder
				break
			}
		}
	}
}
