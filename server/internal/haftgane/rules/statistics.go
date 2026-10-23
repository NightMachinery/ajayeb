package rules

import "github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"

// Bounds are certain survivors and an upper bound, never sampled estimates.
type Bounds struct{ Certain, AtMost uint8 }
type KnownPick struct {
	Age, Turn uint8
	Seat      Seat
	Design    catalog.DesignID
}
type Statistics struct {
	Circulating   [catalog.NumColors]Bounds
	Returning     [catalog.NumDesigns + 1]Bounds
	Deductions    [2 * MaxTurns]KnownPick
	NumDeductions uint8
}

type rememberedPacket struct {
	count     [catalog.NumDesigns + 1]uint8
	anonymous uint8
	remaining uint8
}

// Statistics uses only the immutable observation. For each packet, the latest
// sighting replaces earlier sightings. Public builds remove a named card;
// private picks remove an unknown card. A colour survives at least count minus
// unknown removals and at most min(count, packet size).
func (o *Observation) Statistics() Statistics {
	var out Statistics
	if o == nil || !o.Now.Seat.Valid(o.Now.Seats) || o.Now.Age < 1 || o.Now.Age > NumAges || o.Now.Turn < 1 || o.Now.Turn > MaxTurns {
		return out
	}
	c, k := &o.Now, &o.Know
	age := c.Age - 1
	resolved := min(c.Turn-1, uint8(TurnsPerAge))
	if c.Phase == PhaseDiscardBuild {
		resolved = c.Turn
	}
	if c.Phase == PhaseLastCard || c.Phase == PhaseOver {
		resolved = TurnsPerAge
	}
	o.neighbourDeductions(&out, resolved)
	if resolved >= TurnsPerAge {
		return out
	} // Sixth-turn leftovers no longer circulate.
	var latest [MaxSeats]uint8
	for t := uint8(1); t <= resolved && t < TurnsPerAge; t++ {
		if k.Picks[age][t-1] != 0 {
			latest[packetOrigin(c.Seat, c.Seats, c.Age, t)] = t
		}
	}
	for origin, seen := range latest[:c.Seats] {
		if seen == 0 {
			continue
		}
		p := rememberedPacket{}
		for _, id := range k.Seen[age][seen-1] {
			if id != 0 {
				p.count[id]++
				p.remaining++
			}
		}
		if !p.remove(k.Picks[age][seen-1]) {
			continue
		}
		for t := seen + 1; t <= resolved; t++ {
			holder := packetHolder(Seat(origin), c.Seats, c.Age, t)
			play := k.Plays[age][t-1][holder]
			if play.Kind == ActBuild {
				p.remove(play.Design)
			} else {
				p.anonymous++
				if p.remaining > 0 {
					p.remaining--
				}
			}
		}
		currentTurn := resolved + 1
		if c.Phase == PhaseDiscardBuild {
			currentTurn = resolved
		}
		holder := packetHolder(Seat(origin), c.Seats, c.Age, currentTurn)
		if c.Phase == PhaseTurn && holder == c.Seat {
			// This is an already seen packet coming back, not a new packet in hand.
			p = rememberedPacket{}
			for _, id := range c.Hand {
				if id != 0 {
					p.count[id]++
					p.remaining++
				}
			}
		}
		var colours [catalog.NumColors]uint8
		for id, count := range p.count {
			if count == 0 {
				continue
			}
			d, ok := catalog.DesignByID(catalog.DesignID(id))
			if ok {
				colours[d.Color] += count
			}
		}
		for colour, count := range colours {
			out.Circulating[colour].add(count, p.anonymous, p.remaining)
		}
		// From the current holder, count turns until the next arrival at this
		// seat. A hand already here must make another full circuit.
		distance := uint8(0)
		for distance < c.Seats {
			distance++
			holder = passTo(holder, c.Seats, c.Age)
			if holder == c.Seat {
				break
			}
		}
		if c.Seats <= 5 && currentTurn+distance <= TurnsPerAge {
			future := distance
			if c.Phase == PhaseDiscardBuild {
				future--
			} // This holder already picked.
			remain := uint8(0)
			if p.remaining > future {
				remain = p.remaining - future
			}
			for id, count := range p.count {
				out.Returning[id].add(count, p.anonymous+future, remain)
			}
		}
	}
	return out
}
func (b *Bounds) add(count, unknown, remaining uint8) {
	if count > unknown {
		b.Certain += count - unknown
	}
	b.AtMost += min(count, remaining)
}
func (p *rememberedPacket) remove(id catalog.DesignID) bool {
	if id == 0 || int(id) >= len(p.count) || p.count[id] == 0 {
		return false
	}
	p.count[id]--
	if p.remaining > 0 {
		p.remaining--
	}
	return true
}
func passTo(seat Seat, n, age uint8) Seat {
	if age == 2 {
		return seat.Right(n)
	}
	return seat.Left(n)
}
func packetHolder(origin Seat, n, age, turn uint8) Seat {
	for t := uint8(1); t < turn; t++ {
		origin = passTo(origin, n, age)
	}
	return origin
}

// A packet comparison can identify a private neighbour pick only if every
// unexplained missing copy has the same design. No identity is guessed from
// an ambiguous set. Current hands are legitimate new sightings too.
func (o *Observation) neighbourDeductions(out *Statistics, resolved uint8) {
	c, k := &o.Now, &o.Know
	age := c.Age - 1
	for first := uint8(1); first+c.Seats <= TurnsPerAge; first++ {
		last := first + c.Seats
		if first > resolved || k.Picks[age][first-1] == 0 {
			continue
		}
		end := k.Seen[age][last-1]
		if c.Phase == PhaseTurn && last == c.Turn {
			end = c.Hand
		} else if last > resolved || k.Picks[age][last-1] == 0 {
			continue
		}
		var missing [catalog.NumDesigns + 1]int8
		for _, id := range k.Seen[age][first-1] {
			if id != 0 {
				missing[id]++
			}
		}
		missing[k.Picks[age][first-1]]--
		for _, id := range end {
			if id != 0 {
				missing[id]--
			}
		}
		var anonymous [MaxSeats]KnownPick
		count := 0
		holder := c.Seat
		for t := first + 1; t < last; t++ {
			holder = passTo(holder, c.Seats, c.Age)
			play := k.Plays[age][t-1][holder]
			switch play.Kind {
			case ActBuild:
				missing[play.Design]--
			case ActStage, ActSell:
				anonymous[count] = KnownPick{Age: c.Age, Turn: t, Seat: holder}
				count++
			}
		}
		var only catalog.DesignID
		copies := 0
		valid := true
		for id, n := range missing {
			if n < 0 {
				valid = false
				break
			}
			if n > 0 {
				if only != 0 {
					valid = false
					break
				}
				only = catalog.DesignID(id)
				copies = int(n)
			}
		}
		if !valid || only == 0 || copies != count {
			continue
		}
		for _, pick := range anonymous[:count] {
			if pick.Seat != c.Seat.Left(c.Seats) && pick.Seat != c.Seat.Right(c.Seats) {
				continue
			}
			pick.Design = only
			if int(out.NumDeductions) < len(out.Deductions) {
				out.Deductions[out.NumDeductions] = pick
				out.NumDeductions++
			}
		}
	}
}
