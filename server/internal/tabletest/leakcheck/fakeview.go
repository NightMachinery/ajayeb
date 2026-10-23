package leakcheck

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

// The fake view's whitelist: the checker's own copy of fakeview's shape,
// decoded with unknown fields refused. It is written from the fake's
// documented view, not generated from fakeview, so a field added there
// fails here until someone decides whether it may be shown.

type wTable struct {
	Point    string    `json:"point"`
	Kind     string    `json:"kind"`
	Required []int     `json:"required"`
	Phase    string    `json:"phase"`
	Age      int       `json:"age"`
	Turn     int       `json:"turn"`
	Pass     string    `json:"pass"`
	Pile     int       `json:"pile"`
	Seats    []wSeat   `json:"seats"`
	You      *wPrivate `json:"you"`
	Events   []wEvent  `json:"events"`
	Result   *wResult  `json:"result"`
	Reveal   *wReveal  `json:"reveal"`
}

type wSeat struct {
	Seat     int      `json:"seat"`
	Board    string   `json:"board"`
	Coins    int      `json:"coins"`
	Built    []string `json:"built"`
	Stages   int      `json:"stages"`
	HandSize int      `json:"handSize"`
	HasKept  bool     `json:"hasKept"`
	Last     *wPlay   `json:"last"`
}

type wPlay struct {
	Kind   string `json:"kind"`
	Design string `json:"design"`
	Stage  int    `json:"stage"`
	Left   int    `json:"left"`
	Right  int    `json:"right"`
}

type wPrivate struct {
	Seat    int       `json:"seat"`
	Hand    []string  `json:"hand"`
	Kept    string    `json:"kept"`
	Tucked  []string  `json:"tucked"`
	Choices []wChoice `json:"choices"`
	Offer   []wOffer  `json:"offer"`
	Seen    []wSeen   `json:"seen"`
}

type wChoice struct {
	Key    string `json:"key"`
	Kind   string `json:"kind"`
	Design string `json:"design"`
	Left   int    `json:"left"`
	Right  int    `json:"right"`
}

type wOffer struct {
	Design string `json:"design"`
	Count  int    `json:"count"`
	Legal  bool   `json:"legal"`
}

type wSeen struct {
	Point  string   `json:"point"`
	Hand   []string `json:"hand"`
	Played string   `json:"played"`
	Action string   `json:"action"`
}

type wEvent struct {
	Kind   string `json:"kind"`
	Seat   int    `json:"seat"`
	Play   string `json:"play"`
	Design string `json:"design"`
	Stage  int    `json:"stage"`
	Left   int    `json:"left"`
	Right  int    `json:"right"`
	Age    int    `json:"age"`
	Pass   string `json:"pass"`
}

type wHistory struct {
	Point  string   `json:"point"`
	Events []wEvent `json:"events"`
}

type wResult struct {
	Scores  []wScore `json:"scores"`
	Rank    []int    `json:"rank"`
	Winners []int    `json:"winners"`
}

type wScore struct {
	Seat     int `json:"seat"`
	Built    int `json:"built"`
	Stages   int `json:"stages"`
	Treasure int `json:"treasure"`
	Total    int `json:"total"`
}

type wReveal struct {
	Tucked [][]string `json:"tucked"`
	Pile   []string   `json:"pile"`
	Box    []string   `json:"box"`
}

func strict(data []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

// visible is the oracle's statement of who may know an event's card: anyone
// for a build or a salvage; only the seat itself for its tuck, its sale,
// its leftover and its kept card; nobody for the rest.
func visible(e Event, viewer int) bool {
	switch {
	case e.Kind == "salvage", e.Kind == "play" && e.Play == "build":
		return true
	case e.Kind == "play", e.Kind == "leftover", e.Kind == "kept":
		return viewer >= 0 && viewer == e.Seat
	}
	return false
}

func (c *Checker) checkView(r *report, d Doc) {
	var v wTable
	if err := strict(d.Data, &v); err != nil {
		r.add(UnknownField, "", err.Error())
		return
	}
	t, m := c.Truth, &c.Truth.Moments[d.Moment]
	same := func(path string, got, want any) {
		if fmt.Sprint(got) != fmt.Sprint(want) {
			r.add(Mismatch, path, fmt.Sprintf("%v, want %v", got, want))
		}
	}
	same("point", v.Point, m.Point)
	same("kind", v.Kind, m.Kind)
	same("phase", v.Phase, m.Phase)
	same("age", v.Age, m.Age)
	same("turn", v.Turn, m.Turn)
	same("pile", v.Pile, len(m.Pile))
	c.seats(r, "required", v.Required, m.Required)
	if pass := map[int]string{1: "left", 2: "right"}[m.Age]; v.Pass != pass {
		r.add(Mismatch, "pass", fmt.Sprintf("%q, want %q", v.Pass, pass))
	}

	if len(v.Seats) != t.Seats {
		r.add(Mismatch, "seats", fmt.Sprintf("%d seats, want %d", len(v.Seats), t.Seats))
	}
	for i, s := range v.Seats {
		p := fmt.Sprintf("seats[%d]", i)
		if s.Seat != i {
			r.add(Order, p+".seat", fmt.Sprintf("seat %d at position %d", s.Seat, i))
			continue
		}
		if i >= t.Seats {
			continue
		}
		same(p+".coins", s.Coins, m.Coins[i])
		same(p+".stages", s.Stages, m.Stages[i])
		same(p+".handSize", s.HandSize, len(m.Hands[i]))
		same(p+".hasKept", s.HasKept, m.Kept[i] != "")
		c.cards(r, p+".built", s.Built, m.Built[i], Mismatch)
		c.last(r, p+".last", s.Last, i, d)
	}

	c.private(r, d, v.You)

	want := []Event{}
	if d.Moment > 0 {
		want = t.Resolutions[d.Moment-1].Events
	}
	c.events(r, "events", v.Events, want, d.Viewer)

	if (v.Result != nil) != m.Over {
		r.add(Mismatch, "result", fmt.Sprintf("present %v, over %v", v.Result != nil, m.Over))
	}
	if v.Result != nil {
		for i, s := range v.Result.Scores {
			if s.Seat != i {
				r.add(Order, fmt.Sprintf("result.scores[%d]", i), fmt.Sprintf("seat %d", s.Seat))
			}
		}
		if !slices.IsSorted(v.Result.Winners) {
			r.add(Order, "result.winners", fmt.Sprint(v.Result.Winners))
		}
	}

	switch {
	case v.Reveal != nil && !m.Reveal:
		r.add(HiddenCard, "reveal", "the end reveal while the game is on, or with the reveal off")
	case v.Reveal == nil && m.Reveal:
		r.add(Mismatch, "reveal", "missing at the end")
	case v.Reveal != nil:
		if len(v.Reveal.Tucked) != t.Seats {
			r.add(Mismatch, "reveal.tucked", fmt.Sprintf("%d seats", len(v.Reveal.Tucked)))
		}
		for i, tk := range v.Reveal.Tucked {
			if i < t.Seats {
				c.cards(r, fmt.Sprintf("reveal.tucked[%d]", i), tk, m.Tucked[i], Mismatch)
			}
		}
		c.cards(r, "reveal.pile", v.Reveal.Pile, m.Pile, Mismatch)
		c.cards(r, "reveal.box", v.Reveal.Box, m.Box, Mismatch)
	}
}

// private checks the viewer's own section.
func (c *Checker) private(r *report, d Doc, y *wPrivate) {
	m := &c.Truth.Moments[d.Moment]
	if d.Viewer < 0 {
		if y != nil {
			r.add(ForeignOffer, "you", fmt.Sprintf("seat %d's private section, shown to a spectator", y.Seat))
		}
		return
	}
	if y == nil {
		r.add(Mismatch, "you", "missing for a seat")
		return
	}
	if y.Seat != d.Viewer {
		r.add(ForeignOffer, "you", fmt.Sprintf("seat %d's private section, shown to seat %d", y.Seat, d.Viewer))
		return
	}
	v := d.Viewer
	c.cards(r, "you.hand", y.Hand, m.Hands[v], HiddenCard)
	c.cards(r, "you.tucked", y.Tucked, m.Tucked[v], HiddenCard)
	if y.Kept != m.Kept[v] {
		c.card(r, "you.kept", y.Kept)
		rule := Mismatch
		if y.Kept != "" {
			rule = HiddenCard
		}
		r.add(rule, "you.kept", fmt.Sprintf("%q, but the seat keeps %q", y.Kept, m.Kept[v]))
	}

	keys := make([]string, 0, len(y.Choices))
	for i, ch := range y.Choices {
		p := fmt.Sprintf("you.choices[%d]", i)
		keys = append(keys, ch.Key)
		if !slices.Contains(m.Legal[v], ch.Key) {
			r.add(ForeignOffer, p+".key", c.offeredTo(ch.Key, m, v))
			continue
		}
		parts := strings.Split(ch.Key, ":")
		if ch.Kind != parts[0] || ch.Design != parts[1] {
			r.add(Mismatch, p, fmt.Sprintf("%s %s for the key %q", ch.Kind, ch.Design, ch.Key))
		}
	}
	if !slices.Equal(keys, m.Legal[v]) {
		if a, b := slices.Sorted(slices.Values(keys)), slices.Sorted(slices.Values(m.Legal[v])); slices.Equal(a, b) {
			r.add(Order, "you.choices", "the offered keys are not in the engine's order")
		} else {
			r.add(Mismatch, "you.choices", fmt.Sprintf("%d keys, want %d", len(keys), len(m.Legal[v])))
		}
	}

	if v != m.Decider && len(y.Offer) > 0 {
		r.add(ForeignOffer, "you.offer", fmt.Sprintf("a salvage offer shown to seat %d, which is not deciding one", v))
	}
	if v == m.Decider {
		var designs, pile []string
		for i, o := range y.Offer {
			p := fmt.Sprintf("you.offer[%d]", i)
			c.card(r, p+".design", o.Design)
			designs = append(designs, o.Design)
			if n := count(m.Pile, o.Design); o.Count != n {
				r.add(Mismatch, p+".count", fmt.Sprintf("%d, want %d", o.Count, n))
			}
			if legal := !slices.Contains(m.Built[v], o.Design); o.Legal != legal {
				r.add(Mismatch, p+".legal", fmt.Sprint(o.Legal))
			}
		}
		pile = slices.Compact(slices.Clone(m.Pile))
		c.order(r, "you.offer", designs)
		if !sameSet(designs, pile) {
			r.add(Mismatch, "you.offer", fmt.Sprintf("%v, want %v", designs, pile))
		}
	}

	var seen []wSeen
	for _, res := range c.Truth.Resolutions[:d.Moment] {
		if slices.Contains(res.Required, v) {
			seen = append(seen, wSeen{Point: res.Point, Hand: res.Held[v], Played: res.Actions[v].Design, Action: res.Actions[v].Kind})
		}
	}
	if len(y.Seen) != len(seen) {
		r.add(Mismatch, "you.seen", fmt.Sprintf("%d entries, want %d", len(y.Seen), len(seen)))
		return
	}
	for i, s := range y.Seen {
		p := fmt.Sprintf("you.seen[%d]", i)
		if s.Point != seen[i].Point {
			r.add(Order, p+".point", fmt.Sprintf("%q, want %q", s.Point, seen[i].Point))
		}
		c.cards(r, p+".hand", s.Hand, seen[i].Hand, HiddenCard)
		if s.Played != seen[i].Played {
			c.card(r, p+".played", s.Played)
			r.add(HiddenCard, p+".played", fmt.Sprintf("%q, but the seat played %q", s.Played, seen[i].Played))
		}
		if s.Action != seen[i].Action {
			r.add(Mismatch, p+".action", fmt.Sprintf("%q, want %q", s.Action, seen[i].Action))
		}
	}
}

// last checks a seat's latest public play: the card only for a build or a
// salvage.
func (c *Checker) last(r *report, path string, got *wPlay, seat int, d Doc) {
	var want *wPlay
	stage := 0
	for _, res := range c.Truth.Resolutions[:d.Moment] {
		if !slices.Contains(res.Required, seat) {
			continue
		}
		a := res.Actions[seat]
		want = &wPlay{Kind: a.Kind, Left: a.Left, Right: a.Right}
		switch a.Kind {
		case "build", "salvage":
			want.Design = a.Design
		case "tuck":
			stage++
			want.Stage = stage
		}
	}
	switch {
	case got == nil && want == nil:
	case got == nil || want == nil:
		r.add(Mismatch, path, fmt.Sprintf("present %v, want %v", got != nil, want != nil))
	default:
		if got.Design != "" && want.Design == "" {
			c.card(r, path+".design", got.Design)
			r.add(HiddenCard, path+".design", fmt.Sprintf("the card of seat %d's %s", seat, want.Kind))
		} else if *got != *want {
			r.add(Mismatch, path, fmt.Sprintf("%+v, want %+v", *got, *want))
		}
	}
}

// events compares filtered events with the truth's, filtered by visible.
func (c *Checker) events(r *report, path string, got []wEvent, want []Event, viewer int) {
	if len(got) != len(want) {
		r.add(Mismatch, path, fmt.Sprintf("%d events, want %d", len(got), len(want)))
		return
	}
	for i, g := range got {
		p := fmt.Sprintf("%s[%d]", path, i)
		w := want[i]
		design := ""
		if visible(w, viewer) {
			design = w.Design
		}
		if g.Design != design {
			c.card(r, p+".design", g.Design)
			if g.Design != "" && design == "" {
				r.add(HiddenCard, p+".design", fmt.Sprintf("the card of seat %d's %s %s", w.Seat, w.Kind, w.Play))
			} else {
				r.add(Mismatch, p+".design", fmt.Sprintf("%q, want %q", g.Design, design))
			}
		}
		g.Design = design
		if g != (wEvent{w.Kind, w.Seat, w.Play, design, w.Stage, w.Left, w.Right, w.Age, w.Pass}) {
			r.add(Mismatch, p, fmt.Sprintf("%+v, want %+v", g, w))
		}
	}
}

func (c *Checker) checkHistory(r *report, d Doc) {
	var h []wHistory
	if err := strict(d.Data, &h); err != nil {
		r.add(UnknownField, "", err.Error())
		return
	}
	res := c.Truth.Resolutions[:d.Moment]
	if len(h) != len(res) {
		r.add(Mismatch, "", fmt.Sprintf("%d entries, want %d", len(h), len(res)))
		return
	}
	for i, e := range h {
		if e.Point != res[i].Point {
			r.add(Order, fmt.Sprintf("[%d].point", i), fmt.Sprintf("%q, want %q: the history is not in point order", e.Point, res[i].Point))
			return
		}
	}
	for i, e := range h {
		c.events(r, fmt.Sprintf("[%d].events", i), e.Events, res[i].Events, d.Viewer)
	}
}

// card checks that a card value is a bare catalog slug.
func (c *Checker) card(r *report, path, s string) {
	if _, ok := c.index[s]; !ok && s != "" {
		r.add(ShuffleID, path, fmt.Sprintf("%q is not a catalog design", s))
	}
}

// cards checks a card list: bare slugs, in catalog order, and the same
// cards as want. Extra cards are reported under extra; missing ones as a
// mismatch.
func (c *Checker) cards(r *report, path string, got, want []string, extra Rule) {
	for i, s := range got {
		c.card(r, fmt.Sprintf("%s[%d]", path, i), s)
	}
	c.order(r, path, got)
	if more := minus(got, want); len(more) > 0 {
		r.add(extra, path, fmt.Sprintf("%v that the viewer may not know here", more))
	} else if less := minus(want, got); len(less) > 0 {
		r.add(Mismatch, path, fmt.Sprintf("missing %v", less))
	}
}

// order reports a card list that is not in catalog order.
func (c *Checker) order(r *report, path string, got []string) {
	for i := 1; i < len(got); i++ {
		a, okA := c.index[got[i-1]]
		b, okB := c.index[got[i]]
		if okA && okB && a > b {
			r.add(DealOrder, path, fmt.Sprintf("%v is not in catalog order", got))
			return
		}
	}
}

// seats checks a seat list: ascending, and equal to want.
func (c *Checker) seats(r *report, path string, got, want []int) {
	switch {
	case slices.Equal(got, want):
	case sameSet(got, want):
		r.add(Order, path, fmt.Sprintf("%v is not in seat order", got))
	default:
		r.add(Mismatch, path, fmt.Sprintf("%v, want %v", got, want))
	}
}

// offeredTo names who a key belongs to.
func (c *Checker) offeredTo(key string, m *Moment, viewer int) string {
	for s, keys := range m.Legal {
		if s != viewer && slices.Contains(keys, key) {
			return fmt.Sprintf("%q is offered to seat %d, not to seat %d", key, s, viewer)
		}
	}
	return fmt.Sprintf("%q is offered to nobody", key)
}

func count(xs []string, x string) int {
	n := 0
	for _, y := range xs {
		if y == x {
			n++
		}
	}
	return n
}

// minus returns the multiset a minus b.
func minus[T comparable](a, b []T) []T {
	left := slices.Clone(b)
	var out []T
	for _, x := range a {
		if i := slices.Index(left, x); i >= 0 {
			left = slices.Delete(left, i, i+1)
		} else {
			out = append(out, x)
		}
	}
	return out
}

func sameSet[T comparable](a, b []T) bool {
	return len(a) == len(b) && len(minus(a, b)) == 0
}
