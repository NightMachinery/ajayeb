package rules

import (
	"github.com/NightMachinery/ajayeb/server/internal/enginekit/rng"
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
)

const (
	samplePlaces = NumAges*MaxSeats*HandSize + MaxBoxGuilds
	sampleGroups = NumAges * (HandSize + 1)
)

// World is a state and a private witness, never an authoritative Game. The
// witness indexes each packet by removal turn (slot seven is its leftover).
// Search can obtain only the state, by value; no seed or record is exposed.
type World struct {
	state State
	cards [NumAges][MaxSeats][HandSize]catalog.DesignID
}

func (w World) State() State { return w.state }

type samplePlace struct {
	age, packet, turn uint8
	group, bucket     uint8 // bucket zero has no private-look total
	class             uint8
	birth, death      uint8 // pile entry interval, zero means absent/unconsumed
	base, allowed     catalog.DesignSet
	fixed, value      catalog.DesignID
}
type sampleGroup struct{ supply [catalog.NumDesigns + 1]uint8 }
type sampleOpportunity struct {
	moment  uint8
	design  catalog.DesignID // zero means skipped, unless pending
	pending bool
	blocked catalog.DesignSet
}
type sampleModel struct {
	stop           func() bool
	stopped        bool
	floor          [samplePlaces]catalog.DesignID
	places         [samplePlaces]samplePlace
	groups         [sampleGroups]sampleGroup
	looks          [MaxPileLooks]PileLook
	bucket         [MaxPileLooks + 1]sampleGroup
	ops            [MaxPileLooks]sampleOpportunity
	np, ng, nl, no int
	checks, limit  int
}

// Determinize reads only the permitted observation and caller-owned random
// tape. There are no retries of whole samples, wall clocks or global caches.
// A false result carries no partial world. Callers must use their legal Greedy
// fallback, never a relaxed constraint or authoritative hidden state.
func Determinize(o Observation, r *rng.RNG) (World, bool) {
	w, _, ok := determinize(o, r, MaxSamplerChecks)
	return w, ok
}

// SamplerCost describes deterministic work and why no world was returned.
// Stopped means the caller withdrew the work; Exhausted means its check budget
// ran out. Neither outcome may expose a partial world.
type SamplerCost struct {
	Checks             int
	Stopped, Exhausted bool
}

// DeterminizeControlled is the worker-facing sampler. limit is clamped to
// [0, MaxSamplerChecks]. stop may be nil; otherwise it is queried before work,
// before every feasibility/source check, within matching traversals, and around
// final materialization.
// A false stop callback preserves Determinize's exact world and random tape.
// The callback is caller-supplied input, not a clock or context read by rules.
func DeterminizeControlled(o Observation, r *rng.RNG, limit int, stop func() bool) (World, SamplerCost, bool) {
	var m sampleModel
	m.limit, m.stop = min(max(limit, 0), MaxSamplerChecks), stop
	if stop != nil && stop() {
		return World{}, SamplerCost{Stopped: true}, false
	}
	if r == nil || !m.prepare(&o) || !m.sources(0, r) {
		return World{}, SamplerCost{Checks: m.checks, Stopped: m.stopped, Exhausted: m.checks >= m.limit && !m.stopped}, false
	}
	if stop != nil && stop() {
		return World{}, SamplerCost{Checks: m.checks, Stopped: true}, false
	}
	m.shuffleClasses(r)
	w, ok := m.world(&o)
	if stop != nil && stop() {
		return World{}, SamplerCost{Checks: m.checks, Stopped: true}, false
	}
	cost := SamplerCost{Checks: m.checks}
	if !ok {
		return World{}, cost, false
	}
	return w, cost, true
}

func determinize(o Observation, r *rng.RNG, limit int) (World, int, bool) {
	w, cost, ok := DeterminizeControlled(o, r, limit, nil)
	return w, cost.Checks, ok
}

func sampleMoment(age, turn uint8) uint8 { return (age-1)*8 + turn }

func (m *sampleModel) prepare(o *Observation) bool {
	c := &o.Now
	if !c.Seat.Valid(c.Seats) || c.Phase > PhaseOver || c.Age > NumAges || c.Points > MaxPoints || c.Opts.Validate() != nil {
		return false
	}
	if _, ok := c.publicState(); !ok {
		return false
	}
	for age := uint8(1); age <= NumAges; age++ {
		first := m.np
		rest := m.ng
		m.ng++
		var all catalog.DesignSet
		for id := catalog.DesignID(1); id <= catalog.NumDesigns; id++ {
			d, _ := catalog.DesignByID(id)
			if d.Age != age {
				continue
			}
			count := uint8(0)
			if d.Color == catalog.Purple {
				count = 1 // all ten guilds, including the box places below
			} else {
				for _, marker := range d.Copies {
					if marker != 0 && marker <= c.Seats {
						count++
					}
				}
			}
			m.groups[rest].supply[id] = count
			if count != 0 {
				all.Add(id)
			}
		}
		for packet := Seat(0); packet < Seat(c.Seats); packet++ {
			var seen [HandSize + 1][HandSize]catalog.DesignID
			var sight [HandSize + 1]bool
			for turn := uint8(1); turn <= TurnsPerAge; turn++ {
				if packetHolder(packet, c.Seats, age, turn) == c.Seat && o.Know.Picks[age-1][turn-1] != 0 {
					seen[turn-1], sight[turn-1] = o.Know.Seen[age-1][turn-1], true
				}
			}
			if age == c.Age {
				switch c.Phase {
				case PhaseTurn:
					if c.Turn < 1 || c.Turn > TurnsPerAge {
						return false
					}
					if packetHolder(packet, c.Seats, age, c.Turn) == c.Seat {
						seen[c.Turn-1], sight[c.Turn-1] = c.Hand, true
					}
				case PhaseDiscardBuild:
					if c.Turn < 1 || c.Turn > TurnsPerAge {
						return false
					}
					if c.Turn < TurnsPerAge && packetHolder(packet, c.Seats, age, c.Turn) == c.Seat {
						seen[c.Turn], sight[c.Turn] = c.Hand, true
					}
				}
				if c.Kept != 0 && packetHolder(packet, c.Seats, age, TurnsPerAge) == c.Seat {
					seen[6][0], sight[6] = c.Kept, true
				}
			}
			if o.Know.Picks[age-1][6] != 0 && packetHolder(packet, c.Seats, age, TurnsPerAge) == c.Seat {
				seen[6], sight[6] = o.Know.Seen[age-1][6], true
			}
			for turn := uint8(1); turn <= HandSize; turn++ {
				m.places[m.np] = samplePlace{age: age, packet: uint8(packet), turn: turn, group: uint8(rest), base: all}
				m.np++
			}
			for start := 0; start < HandSize; start++ {
				if !sight[start] {
					continue
				}
				end := start + 1
				for end < HandSize && !sight[end] {
					end++
				}
				if !m.addStretch(first+int(packet)*HandSize, rest, start, end, seen[start], seen[end]) {
					return false
				}
			}
		}
		if age == 3 {
			for i := 0; i < catalog.NumGuilds-int(c.Seats)-2; i++ {
				var guilds catalog.DesignSet
				for id := catalog.DesignID(1); id <= catalog.NumDesigns; id++ {
					d, _ := catalog.DesignByID(id)
					if d.Color == catalog.Purple {
						guilds.Add(id)
					}
				}
				m.places[m.np] = samplePlace{age: age, group: uint8(rest), base: guilds}
				m.np++
			}
		}
	}
	return m.chronology(o)
}

func (m *sampleModel) addStretch(first, rest, start, end int, before, after [HandSize]catalog.DesignID) bool {
	if m.ng >= len(m.groups) {
		return false
	}
	g := m.ng
	m.ng++
	var counts [catalog.NumDesigns + 1]int
	for _, id := range before {
		if id > catalog.NumDesigns {
			return false
		}
		counts[id]++
	}
	for _, id := range after {
		if id > catalog.NumDesigns {
			return false
		}
		counts[id]--
	}
	total := 0
	for id := 1; id <= catalog.NumDesigns; id++ {
		n := counts[id]
		if n < 0 || n > int(m.groups[rest].supply[id]) {
			return false
		}
		m.groups[g].supply[id] = uint8(n)
		m.groups[rest].supply[id] -= uint8(n)
		total += n
	}
	if total != end-start {
		return false
	}
	for i := start; i < end; i++ {
		m.places[first+i].group = uint8(g)
	}
	return true
}

func (m *sampleModel) at(age, turn uint8, seat Seat, n uint8) *samplePlace {
	packetTurn := min(turn, uint8(TurnsPerAge))
	origin := packetOrigin(seat, n, age, packetTurn)
	for i := 0; i < m.np; i++ {
		p := &m.places[i]
		if p.age == age && p.packet == uint8(origin) && p.turn == turn {
			return p
		}
	}
	return nil
}

// Public chronology supplies pile entry times, forced public constructions,
// the name sets at skipped stages and the owner-only looks. No hidden city
// cache, pick, deck or record is consulted.
func (m *sampleModel) chronology(o *Observation) bool {
	c := &o.Now
	var cities [MaxSeats]City
	for s := Seat(0); s < Seat(c.Seats); s++ {
		cities[s] = City{Board: c.Cities[s].Board, Side: c.Cities[s].Side}
	}
	for age := uint8(1); age <= NumAges; age++ {
		for turn := uint8(1); turn <= MaxTurns; turn++ {
			moment := sampleMoment(age, turn)
			for s := Seat(0); s < Seat(c.Seats); s++ {
				play := o.Know.Plays[age-1][turn-1][s]
				if play.Kind == ActNone {
					continue
				}
				p := m.at(age, turn, s, c.Seats)
				if p == nil {
					return false
				}
				if s == c.Seat {
					p.fixed = o.Know.Picks[age-1][turn-1]
				}
				switch play.Kind {
				case ActBuild:
					if p.fixed != 0 && p.fixed != play.Design {
						return false
					}
					p.fixed = play.Design
					cities[s].Buildings.Add(play.Design)
				case ActStage:
					cities[s].Stages++
				case ActSell:
					p.birth = moment
				default:
					return false
				}
				if turn == TurnsPerAge {
					keep := false
					for stage := uint8(1); stage <= cities[s].Stages; stage++ {
						keep = keep || stageHasEffect(&cities[s], stage, catalog.EffPlayLastCard)
					}
					if !keep {
						m.at(age, HandSize, s, c.Seats).birth = moment
					}
				}
			}
			if turn > TurnsPerAge {
				continue
			}
			for s := Seat(0); s < Seat(c.Seats); s++ {
				play := o.Know.Plays[age-1][turn-1][s]
				if play.Kind != ActStage || !stageHasEffect(&cities[s], cities[s].Stages, catalog.EffBuildFromDiscard) {
					continue
				}
				if m.no >= len(m.ops) {
					return false
				}
				op := sampleOpportunity{moment: moment, design: o.Know.Rebuilt[age-1][turn-1][s], pending: c.Phase == PhaseDiscardBuild && c.Age == age && c.Turn == turn}
				if op.design > catalog.NumDesigns {
					return false
				}
				for id := catalog.DesignID(1); id <= catalog.NumDesigns; id++ {
					d, _ := catalog.DesignByID(id)
					if !cities[s].Buildings.And(catalog.T.ByName[d.NameID]).Empty() {
						op.blocked.Add(id)
					}
				}
				m.ops[m.no] = op
				m.no++
				if s == c.Seat {
					if m.nl >= len(m.looks) {
						return false
					}
					look := o.Know.PileLooks[m.nl]
					if look.Age != age || look.Turn != turn {
						return false
					}
					m.looks[m.nl] = look
					m.nl++
				}
				if op.design != 0 {
					cities[s].Buildings.Add(op.design)
				}
			}
		}
	}
	return true
}

// poll also splits individual matching checks, without charging extra work
// or changing their search order. A stopped partial matching is never exposed.
func (m *sampleModel) poll() bool {
	if m.stopped {
		return false
	}
	if m.stop != nil && m.stop() {
		m.stopped = true
		return false
	}
	return true
}
func (m *sampleModel) charge() bool {
	if !m.poll() {
		return false
	}
	if m.checks >= m.limit {
		return false
	}
	m.checks++
	return true
}

// At most three sources must be selected. Entries are exchangeable within
// group/time classes; keeping them as individual unit-capacity vertices makes
// the matching proof simple and also handles repeated designs.
func (m *sampleModel) sources(index int, r *rng.RNG) bool {
	if !m.charge() {
		return false
	}
	if index == m.no {
		return m.constrain() && m.fill(r)
	}
	op := m.ops[index]
	if op.pending {
		return m.pendingSource(index, r)
	}
	if op.design == 0 {
		return m.sources(index+1, r)
	}
	var candidates [samplePlaces]int
	n := 0
	for i := 0; i < m.np; i++ {
		p := &m.places[i]
		if p.birth == 0 || p.birth > op.moment || p.death != 0 || !p.base.Has(op.design) || p.fixed != 0 && p.fixed != op.design {
			continue
		}
		possible := true
		for _, earlier := range m.ops[:index] {
			if earlier.design == 0 && !earlier.pending && p.birth <= earlier.moment && !earlier.blocked.Has(op.design) {
				possible = false
			}
		}
		for _, look := range m.looks[:m.nl] {
			moment := sampleMoment(look.Age, look.Turn)
			if p.birth <= moment && moment <= op.moment && look.Count[op.design] == 0 {
				possible = false
			}
		}
		if possible {
			candidates[n], n = i, n+1
		}
	}
	rng.ShuffleSlice(r, candidates[:n])
	// Select one random representative of each exchangeable source class.
	// A failed class is not retried through all of its identical entries;
	// random representatives preserve support for every individual witness.
	var representatives [samplePlaces]int
	nr := 0
	for _, i := range candidates[:n] {
		duplicate := false
		for _, j := range representatives[:nr] {
			a, b := m.places[i], m.places[j]
			same := a.group == b.group && a.base == b.base && a.fixed == b.fixed
			for _, opportunity := range m.ops[:m.no] {
				same = same && (a.birth <= opportunity.moment) == (b.birth <= opportunity.moment)
			}
			if same {
				duplicate = true
				break
			}
		}
		if duplicate {
			continue
		}
		representatives[nr], nr = i, nr+1
	}
	for _, i := range representatives[:nr] {
		fixed := m.places[i].fixed
		m.places[i].death, m.places[i].fixed = op.moment, op.design
		if m.sources(index+1, r) {
			return true
		}
		m.places[i].death, m.places[i].fixed = 0, fixed
		if m.stopped || m.checks >= m.limit {
			break
		}
	}
	return false
}

func (m *sampleModel) constrain() bool {
	m.bucket = [MaxPileLooks + 1]sampleGroup{}
	for i := 0; i < m.np; i++ {
		p := &m.places[i]
		p.allowed, p.value, p.bucket = p.base, 0, 0
		if p.fixed != 0 {
			p.allowed = catalog.DesignSet{}
			p.allowed.Add(p.fixed)
		}
		for _, op := range m.ops[:m.no] {
			if op.design == 0 && !op.pending && p.birth != 0 && p.birth <= op.moment && (p.death == 0 || p.death >= op.moment) {
				p.allowed = p.allowed.And(op.blocked)
			}
		}
		// Once all consumed entries are fixed and removed, successive looks
		// are nested sets. Their differences are disjoint exact-supply buckets.
		if p.birth != 0 && p.death == 0 {
			for j, l := range m.looks[:m.nl] {
				if p.birth <= sampleMoment(l.Age, l.Turn) {
					p.bucket = uint8(j + 1)
					break
				}
			}
		}
		if p.allowed.Empty() {
			return false
		}
	}
	m.floor = [samplePlaces]catalog.DesignID{}
	for i := 0; i < m.np; i++ {
		m.places[i].class = uint8(i)
		for j := 0; j < i; j++ {
			if sameSampleClass(m.places[i], m.places[j]) {
				m.places[i].class = m.places[j].class
				break
			}
		}
	}
	var previous [catalog.NumDesigns + 1]int
	for j, l := range m.looks[:m.nl] {
		var now [catalog.NumDesigns + 1]int
		for id, n := range l.Count {
			now[id] = int(n)
		}
		moment := sampleMoment(l.Age, l.Turn)
		for _, p := range m.places[:m.np] {
			if p.birth != 0 && p.birth <= moment && p.death >= moment {
				now[p.fixed]--
			}
		}
		for id := 0; id <= catalog.NumDesigns; id++ {
			n := now[id] - previous[id]
			if n < 0 || n > samplePlaces {
				return false
			}
			m.bucket[j+1].supply[id] = uint8(n)
		}
		previous = now
	}
	return true
}

func (m *sampleModel) permits(p *samplePlace, id catalog.DesignID) bool {
	if !p.allowed.Has(id) || m.groups[p.group].supply[id] == 0 || p.bucket != 0 && m.bucket[p.bucket].supply[id] == 0 {
		return false
	}
	// During constrained search, fill exchangeable places in sorted order.
	// Otherwise a failed multiset can be retried through factorially many
	// permutations. Physical places are shuffled again after success.
	if id < m.floor[p.class] {
		return false
	}
	return true
}

// feasible is a unit-capacity max-flow (augmenting paths in the bipartite
// places/copies graph). The two margins are checked separately; any remaining
// cross-margin dependency is handled by bounded backtracking, never relaxed.
func (m *sampleModel) feasible(index int, bucket bool) bool {
	if !m.charge() {
		return false
	}
	var places [samplePlaces]int
	var copies [samplePlaces]catalog.DesignID
	var matched [samplePlaces]int
	np, nc := 0, 0
	for i := 0; i < m.np; i++ {
		p := &m.places[i]
		if p.value != 0 {
			continue
		}
		member := int(p.group) == index
		if bucket {
			member = int(p.bucket) == index
		}
		if member {
			places[np], np = i, np+1
		}
	}
	supply := &m.groups[index].supply
	if bucket {
		supply = &m.bucket[index].supply
	}
	for id := catalog.DesignID(1); id <= catalog.NumDesigns; id++ {
		for k := uint8(0); k < supply[id]; k++ {
			if nc >= len(copies) {
				return false
			}
			copies[nc], matched[nc], nc = id, -1, nc+1
		}
	}
	if np != nc {
		return false
	}
	for _, i := range places[:np] {
		var seen [samplePlaces]bool
		if !m.augment(i, copies[:nc], matched[:nc], &seen) {
			return false
		}
	}
	return true
}

func (m *sampleModel) augment(place int, copies []catalog.DesignID, matched []int, seen *[samplePlaces]bool) bool {
	if !m.poll() {
		return false
	}
	for j, id := range copies {
		if seen[j] || !m.permits(&m.places[place], id) {
			continue
		}
		seen[j] = true
		if matched[j] < 0 || m.augment(matched[j], copies, matched, seen) {
			matched[j] = place
			return true
		}
		if m.stopped {
			return false
		}
	}
	return false
}

func (m *sampleModel) fill(r *rng.RNG) bool {
	for g := 0; g < m.ng; g++ {
		if !m.feasible(g, false) {
			return false
		}
	}
	for b := 1; b <= m.nl; b++ {
		if !m.feasible(b, true) {
			return false
		}
	}
	return m.margins() && m.draw(r)
}

func (m *sampleModel) draw(r *rng.RNG) bool {
	chosen, fewest := -1, catalog.NumDesigns+1
	for i := 0; i < m.np; i++ {
		p := &m.places[i]
		if p.value != 0 {
			continue
		}
		n, restricted := 0, p.bucket != 0
		for id := catalog.DesignID(1); id <= catalog.NumDesigns; id++ {
			if m.groups[p.group].supply[id] == 0 {
				continue
			}
			if m.permits(p, id) {
				n++
			} else {
				restricted = true
			}
		}
		if n == 0 {
			return false
		}
		if restricted && n < fewest {
			chosen, fewest = i, n
		}
	}
	if chosen < 0 {
		// Every remaining place accepts its whole group supply. Fisher-Yates
		// gives uniform copy arrangements here, including the pre-look case.
		for g := 0; g < m.ng; g++ {
			var deck [samplePlaces]catalog.DesignID
			n := 0
			for id := catalog.DesignID(1); id <= catalog.NumDesigns; id++ {
				for k := uint8(0); k < m.groups[g].supply[id]; k++ {
					deck[n], n = id, n+1
				}
			}
			rng.ShuffleSlice(r, deck[:n])
			at := 0
			for i := 0; i < m.np; i++ {
				p := &m.places[i]
				if int(p.group) == g && p.value == 0 {
					if at >= n {
						return false
					}
					p.value = deck[at]
					at++
				}
			}
			if at != n {
				return false
			}
		}
		return true
	}
	p := &m.places[chosen]
	var copies [samplePlaces]catalog.DesignID
	n := 0
	for id := catalog.DesignID(1); id <= catalog.NumDesigns; id++ {
		if m.permits(p, id) {
			for k := uint8(0); k < m.groups[p.group].supply[id]; k++ {
				copies[n], n = id, n+1
			}
		}
	}
	rng.ShuffleSlice(r, copies[:n])
	var tried catalog.DesignSet
	for _, id := range copies[:n] {
		if tried.Has(id) {
			continue
		}
		tried.Add(id)
		p.value = id
		floor := m.floor[p.class]
		if m.no != 0 && p.turn != 0 {
			m.floor[p.class] = id
		}
		m.groups[p.group].supply[id]--
		if p.bucket != 0 {
			m.bucket[p.bucket].supply[id]--
		}
		ok := m.feasible(int(p.group), false)
		for b := 1; ok && b <= m.nl; b++ {
			ok = m.feasible(b, true)
		}
		if ok && m.margins() && m.draw(r) {
			return true
		}
		m.groups[p.group].supply[id]++
		if p.bucket != 0 {
			m.bucket[p.bucket].supply[id]++
		}
		p.value = 0
		m.floor[p.class] = floor
		if m.stopped || m.checks >= m.limit {
			break
		}
	}
	return false
}

func (m *sampleModel) world(o *Observation) (World, bool) {
	c := &o.Now
	st, ok := c.publicState()
	if !ok {
		return World{}, false
	}
	w := World{state: st}
	box := 0
	var tucked [MaxSeats]uint8
	// Places are ordered by age, packet, turn, so tucked stage order is
	// recovered below from the public chronology instead of this loop.
	for _, p := range m.places[:m.np] {
		if p.value == 0 {
			return World{}, false
		}
		if p.turn == 0 {
			w.state.BoxGuilds[box] = p.value
			box++
			continue
		}
		w.cards[p.age-1][p.packet][p.turn-1] = p.value
		if p.birth != 0 && p.death == 0 {
			w.state.Discard[p.value]++
		}
		if c.Phase == PhaseSides || p.age > c.Age {
			w.state.Deals[p.age-1][p.packet][p.turn-1] = p.value
		}
	}
	for age := uint8(1); age <= NumAges; age++ {
		for turn := uint8(1); turn <= MaxTurns; turn++ {
			for s := Seat(0); s < Seat(c.Seats); s++ {
				play := o.Know.Plays[age-1][turn-1][s]
				if play.Kind == ActStage {
					if tucked[s] >= MaxStages {
						return World{}, false
					}
					w.state.Cities[s].Tucked[tucked[s]] = m.at(age, turn, s, c.Seats).value
					tucked[s]++
				}
			}
		}
	}
	if c.Age != 0 {
		for s := Seat(0); s < Seat(c.Seats); s++ {
			switch c.Phase {
			case PhaseTurn, PhaseDiscardBuild:
				start := c.Turn
				if c.Phase == PhaseDiscardBuild {
					start++
				}
				if c.Turn <= TurnsPerAge && c.Cities[s].HandSize != 0 {
					packet := packetOrigin(s, c.Seats, c.Age, c.Turn)
					copy(w.state.Hands[s][:], w.cards[c.Age-1][packet][start-1:])
				}
			}
			if c.Cities[s].HasKept {
				w.state.Cities[s].Kept = m.at(c.Age, HandSize, s, c.Seats).value
			}
		}
	}
	for age := range w.state.Deals {
		for s := range w.state.Deals[age] {
			canonicalIDs(w.state.Deals[age][s][:])
		}
	}
	for s := range w.state.Hands {
		canonicalIDs(w.state.Hands[s][:])
	}
	canonicalIDs(w.state.BoxGuilds[:])
	return w, Project(&w.state, c.Seat) == o.Now
}

// margins couples the two matchings: a bucket cannot obtain two copies from
// a group that has only one, even when both unit slots accept that design.
// For each design, max-flow sends its remaining group supplies to the exact
// look-bucket demands (bucket zero takes the rest). Slot capacities are an
// upper bound; the ordinary group matching checks their competition by design.
func (m *sampleModel) margins() bool {
	if m.nl == 0 {
		return true
	}
	const nodes = sampleGroups + MaxPileLooks + 3
	for id := catalog.DesignID(1); id <= catalog.NumDesigns; id++ {
		restricted := 0
		for b := 1; b <= m.nl; b++ {
			restricted += int(m.bucket[b].supply[id])
		}
		if restricted == 0 {
			continue
		}
		if !m.charge() {
			return false
		}
		var capacity [nodes][nodes]int
		source, sink := m.ng+m.nl+1, m.ng+m.nl+2
		total := 0
		for g := 0; g < m.ng; g++ {
			n := int(m.groups[g].supply[id])
			capacity[source][g] = n
			total += n
		}
		if restricted > total {
			return false
		}
		capacity[m.ng][sink] = total - restricted
		for b := 1; b <= m.nl; b++ {
			capacity[m.ng+b][sink] = int(m.bucket[b].supply[id])
		}
		for _, p := range m.places[:m.np] {
			if p.value == 0 && p.allowed.Has(id) && id >= m.floor[p.class] {
				capacity[p.group][m.ng+int(p.bucket)]++
			}
		}
		sent := 0
		for {
			var parent [nodes]int
			for i := range parent {
				parent[i] = -1
			}
			parent[source] = source
			var queue [nodes]int
			queue[0] = source
			tail := 1
			for head := 0; head < tail && parent[sink] < 0; head++ {
				if !m.poll() {
					return false
				}
				from := queue[head]
				for to := 0; to <= sink; to++ {
					if parent[to] < 0 && capacity[from][to] > 0 {
						parent[to] = from
						queue[tail] = to
						tail++
					}
				}
			}
			if parent[sink] < 0 {
				break
			}
			amount := samplePlaces
			for to := sink; to != source; to = parent[to] {
				amount = min(amount, capacity[parent[to]][to])
			}
			for to := sink; to != source; to = parent[to] {
				from := parent[to]
				capacity[from][to] -= amount
				capacity[to][from] += amount
			}
			sent += amount
		}
		if sent != total {
			return false
		}
	}
	return true
}

func sameSampleClass(a, b samplePlace) bool {
	return a.group == b.group && a.bucket == b.bucket && a.allowed == b.allowed
}

func (m *sampleModel) shuffleClasses(r *rng.RNG) {
	var seen [samplePlaces]bool
	for i := 0; i < m.np; i++ {
		if seen[i] {
			continue
		}
		var indices [samplePlaces]int
		n := 0
		for j := i; j < m.np; j++ {
			if sameSampleClass(m.places[i], m.places[j]) {
				seen[j] = true
				indices[n], n = j, n+1
			}
		}
		r.Shuffle(n, func(a, b int) {
			x, y := indices[a], indices[b]
			m.places[x].value, m.places[y].value = m.places[y].value, m.places[x].value
		})
	}
}

// An open discard point is itself public information: unlike a skip it must
// have at least one buildable entry. Select an unconsumed witness entry and
// restrict its names; its identity remains private and it is not consumed.
func (m *sampleModel) pendingSource(index int, r *rng.RNG) bool {
	op := m.ops[index]
	var candidates [samplePlaces]int
	n := 0
	for i, p := range m.places[:m.np] {
		allowed := p.base.AndNot(op.blocked)
		if p.birth != 0 && p.birth <= op.moment && p.death == 0 && !allowed.Empty() && (p.fixed == 0 || allowed.Has(p.fixed)) {
			candidates[n], n = i, n+1
		}
	}
	rng.ShuffleSlice(r, candidates[:n])
	var representatives [samplePlaces]int
	nr := 0
	for _, i := range candidates[:n] {
		duplicate := false
		for _, j := range representatives[:nr] {
			a, b := m.places[i], m.places[j]
			same := a.group == b.group && a.base == b.base && a.fixed == b.fixed
			for _, event := range m.ops[:m.no] {
				same = same && (a.birth <= event.moment) == (b.birth <= event.moment)
			}
			if same {
				duplicate = true
				break
			}
		}
		if duplicate {
			continue
		}
		representatives[nr], nr = i, nr+1
	}
	for _, i := range representatives[:nr] {
		base := m.places[i].base
		m.places[i].base = base.AndNot(op.blocked)
		if m.sources(index+1, r) {
			return true
		}
		m.places[i].base = base
		if m.stopped || m.checks >= m.limit {
			break
		}
	}
	return false
}
