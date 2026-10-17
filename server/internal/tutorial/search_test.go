package tutorial

import (
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// This file finds seeds for a tutorial script (design section 5.3). It runs
// only with AJAYEB_TUTORIAL_SEARCH=1:
//
//	AJAYEB_TUTORIAL_SEARCH=1 make test-go GO_PKGS=./internal/tutorial/... GO_RUN=TestSeedSearch GOFLAGS='-count=1 -v'
//
// AJAYEB_TUTORIAL_PLAN names the order of the player's seven Age I lessons
// (searchPlans; default "chain-first"), AJAYEB_TUTORIAL_SEEDS the last seed
// to try (default 200000) and AJAYEB_TUTORIAL_FOUND how many scripts to
// print (default 3).
//
// For each seed it plays the draft so that the player takes a wonder that
// Age I can build with no extra turn and no pending choice, then searches
// Age I depth first: each of the player's turns takes a card that meets its
// lesson's precondition, each of the bot's replies is a build (preferred,
// cheapest first) or a discard that keeps every rule of the design: the
// first reply uncovers a card, no red card before the military lesson, no
// pending choice or extra turn anywhere, and at least four builds among the
// seven replies. The first scripts found are printed as Go literals for
// script.go, with a readable listing.

// kind is the lesson of one of the player's Age I turns.
type kind int

const (
	kBuildFree kind = iota
	kBuildOwn
	kTrade
	kDiscard
	kWonder
	kScience
	kChain
)

var kindLesson = [...]wire.TutorialLessonID{
	kBuildFree: wire.TutorialLessonBuildFree,
	kBuildOwn:  wire.TutorialLessonBuildOwn,
	kTrade:     wire.TutorialLessonTrade,
	kDiscard:   wire.TutorialLessonDiscard,
	kWonder:    wire.TutorialLessonBuildWonder,
	kScience:   wire.TutorialLessonScience,
	kChain:     wire.TutorialLessonChain,
}

// searchPlans are the orders of the seven Age I lessons the search knows.
// "design" is the design's order. It cannot be met on any seed: the player
// starts with 7 coins and earns only 2 more (the discard) before the science
// card, while the trade costs at least 2, a wonder at least 6 (every wonder
// needs four or more resource units, the player makes at most one of them,
// and each bought unit costs at least 2) and every Age I green card at least
// 2. "chain-first" builds the chain card before the science card, and the
// only chain card the player can then afford and still pay for a green card
// is the Tavern, which is free and gives 4 coins.
var searchPlans = map[string][]kind{
	"design":      {kBuildFree, kBuildOwn, kTrade, kDiscard, kWonder, kScience, kChain},
	"chain-first": {kBuildFree, kBuildOwn, kTrade, kDiscard, kWonder, kChain, kScience},
}

// searchWonders are the wonders the player may build in Age I: none gives
// an extra turn, and none opens a choice.
var searchWonders = []catalog.WonderID{catalog.WonderPyramids, catalog.WonderColossus, catalog.WonderGreatLighthouse}

func envInt(t *testing.T, name string, def int) int {
	v := os.Getenv(name)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		t.Fatalf("%s=%q: %v", name, v, err)
	}
	return n
}

func TestSeedSearch(t *testing.T) {
	if os.Getenv("AJAYEB_TUTORIAL_SEARCH") != "1" {
		t.Skip("set AJAYEB_TUTORIAL_SEARCH=1 to search for tutorial seeds")
	}
	planName := os.Getenv("AJAYEB_TUTORIAL_PLAN")
	if planName == "" {
		planName = "chain-first"
	}
	plan, ok := searchPlans[planName]
	if !ok {
		t.Fatalf("unknown plan %q", planName)
	}
	last := envInt(t, "AJAYEB_TUTORIAL_SEEDS", 200000)
	want := envInt(t, "AJAYEB_TUTORIAL_FOUND", 3)
	found := 0
	for seed := uint64(1); seed <= uint64(last) && found < want; seed++ {
		sc, notes, ok := searchSeed(seed, plan)
		if !ok {
			continue
		}
		found++
		if os.Getenv("AJAYEB_TUTORIAL_BRIEF") == "1" {
			t.Logf("%s", notes)
			continue
		}
		t.Logf("seed %d (plan %s):\n%s\n%s", seed, planName, goLiteral(sc), notes)
	}
	t.Logf("plan %s: %d scripts in seeds 1 to %d", planName, found, last)
}

// searchOpts are the tutorial's options: the player is the first player.
var searchOpts = engine.Options{FirstPlayer: engine.FirstPlayerSeat0}

// searchSeed tries every wonder of searchWonders in the seed's draft.
func searchSeed(seed uint64, plan []kind) (*Script, string, bool) {
	s0 := engine.NewGame(seed, searchOpts)
	for _, w := range searchWonders {
		iw := slices.Index(s0.DraftOrder[:], w)
		if iw < 0 {
			continue
		}
		s := s0
		draft, ok := playDraft(&s, iw)
		if !ok {
			continue
		}
		srch := &ageSearch{plan: plan, wonder: w, budget: 20000}
		if !srch.dfs(&s, 0) {
			continue
		}
		sc := srch.script(seed, draft)
		return sc, srch.notes(seed), true
	}
	return nil, "", false
}

// botDraftRank orders the wonders the bot prefers in the draft: an extra
// turn first, a plausible taste for a strong opponent.
func botDraftRank(w catalog.WonderID) int {
	if _, ok := catalog.FindEffect(catalog.Wonder(w).Effects, catalog.EffPlayAgain); ok {
		return 0
	}
	return 1
}

// playDraft plays the draft so that the player picks DraftOrder[iw] actively
// and applies it to s. It returns the draft's entries (the automatic picks
// have none).
func playDraft(s *engine.State, iw int) ([]Entry, bool) {
	target := s.DraftOrder[iw]
	var out []Entry
	taken := func(w catalog.WonderID) bool {
		i := slices.Index(s.DraftOrder[:], w)
		return s.DraftTaken&(1<<i) != 0
	}
	offer := func(round int) []catalog.WonderID {
		var ws []catalog.WonderID
		for _, w := range s.DraftOrder[round*4 : round*4+4] {
			if !taken(w) {
				ws = append(ws, w)
			}
		}
		return ws
	}
	apply := func(e Entry) bool {
		a, ok := legal(s, e.Key)
		if !ok || engine.Apply(s, a, nil) != nil {
			return false
		}
		out = append(out, e)
		return true
	}
	humanPick := func(round int, lesson wire.TutorialLessonID, lessons ...wire.TutorialLesson) bool {
		ws := offer(round)
		pick := ws[0]
		if slices.Contains(ws, target) {
			pick = target
		}
		return apply(you("draft:"+catalog.Wonder(pick).Slug, lesson, lessons...))
	}
	botPick := func(round int) bool {
		ws := offer(round)
		ws = slices.DeleteFunc(ws, func(w catalog.WonderID) bool { return w == target })
		slices.SortStableFunc(ws, func(a, b catalog.WonderID) int { return botDraftRank(a) - botDraftRank(b) })
		return apply(bot("draft:" + catalog.Wonder(ws[0]).Slug))
	}
	ok := humanPick(0, wire.TutorialLessonDraftPick, info(wire.TutorialLessonDraftIntro)) &&
		botPick(0) && botPick(0) && // the last wonder of round 1 comes to the player
		botPick(1) &&
		humanPick(1, wire.TutorialLessonDraftPick, info(wire.TutorialLessonDraftRound2)) &&
		humanPick(1, wire.TutorialLessonDraftPickAgain) // the last goes to the bot
	if !ok || s.Age != 1 || s.Phase != engine.PhasePlay || s.ToMove != 0 || !ownsWonder(s, 0, target) {
		return nil, false
	}
	return out, true
}

// ownsWonder reports whether seat holds w, unbuilt.
func ownsWonder(s *engine.State, seat engine.Seat, w catalog.WonderID) bool {
	for _, ws := range s.Players[seat].Wonders {
		if ws.ID == w && !ws.Built && !ws.Removed {
			return true
		}
	}
	return false
}

// legal returns the legal action with this key.
func legal(s *engine.State, key string) (engine.Action, bool) {
	for _, a := range engine.LegalActions(s, nil) {
		if a.Key() == key {
			return a, true
		}
	}
	return engine.Action{}, false
}

// accessible returns the accessible cards of the structure, by card id.
func accessible(s *engine.State) []catalog.CardID {
	mask := engine.Accessible(s)
	var out []catalog.CardID
	for i := range engine.SlotsPerAge {
		if mask&(1<<i) != 0 && s.Slots[i].Card != 0 {
			out = append(out, s.Slots[i].Card)
		}
	}
	slices.Sort(out)
	return out
}

// cardsLeft counts the cards still in the structure.
func cardsLeft(s *engine.State) int {
	n := 0
	for _, sl := range s.Slots {
		if sl.Card != 0 {
			n++
		}
	}
	return n
}

// calm reports whether s, after seat's move, is a plain next turn: Age I
// in play with the other seat to move, no extra turn and no choice open.
func calm(s *engine.State, seat engine.Seat) bool {
	return s.Phase == engine.PhasePlay && s.Age == 1 && s.ToMove == seat.Opp() && !s.ExtraTurn && !s.PlayingAgain
}

// step is one action of the Age I search, with what the notes report.
type step struct {
	seat   engine.Seat
	key    string
	kind   kind // the player's lesson; unused for the bot
	paid   int16
	cost   string
	reveal []catalog.CardID // the cards the action turned face up
}

// ageSearch is the depth-first search of Age I.
type ageSearch struct {
	plan   []kind
	wonder catalog.WonderID
	budget int
	path   []step
	end    engine.State
}

// militaryTaught reports whether the military lesson has been shown by the
// player's turn i (it shows on the turn after buildOwn).
func (as *ageSearch) militaryTaught(i int) bool {
	j := slices.Index(as.plan, kBuildOwn)
	return j >= 0 && i > j+1
}

func (as *ageSearch) dfs(s *engine.State, n int) bool {
	if as.budget--; as.budget < 0 {
		return false
	}
	if n == 2*len(as.plan) {
		builds := 0
		for _, st := range as.path {
			if st.seat == 1 && strings.HasPrefix(st.key, "construct:") {
				builds++
			}
		}
		if s.ToMove != 0 || s.Age != 1 || s.Phase != engine.PhasePlay || cardsLeft(s) < 4 || builds < 4 {
			return false
		}
		as.end = *s
		return true
	}
	if n%2 == 0 {
		return as.human(s, n)
	}
	return as.bot(s, n)
}

// try applies one candidate action and recurses.
func (as *ageSearch) try(s *engine.State, n int, st step, check func(pre, post *engine.State, ev []engine.Event) bool) bool {
	a, ok := legal(s, st.key)
	if !ok {
		return false
	}
	t := *s
	var ev []engine.Event
	if engine.Apply(&t, a, &ev) != nil || !calm(&t, st.seat) {
		return false
	}
	if check != nil && !check(s, &t, ev) {
		return false
	}
	for _, e := range ev {
		if e.Kind == engine.EvReveal {
			st.reveal = append(st.reveal, e.Card)
		}
	}
	as.path = append(as.path, st)
	if as.dfs(&t, n+1) {
		return true
	}
	as.path = as.path[:len(as.path)-1]
	return false
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func slug(c catalog.CardID) string { return catalog.Card(c).Slug }

func (as *ageSearch) human(s *engine.State, n int) bool {
	k := as.plan[n/2]
	cards := accessible(s)
	if k == kTrade {
		// A red card would move the pawn again before the military lesson
		// has sunk in: try the others first.
		slices.SortStableFunc(cards, func(a, b catalog.CardID) int {
			return boolInt(catalog.Card(a).Color == catalog.Red) - boolInt(catalog.Card(b).Color == catalog.Red)
		})
	}
	for _, c := range cards {
		d := catalog.Card(c)
		pay := engine.BuildingPayment(s, 0, c)
		build := step{seat: 0, key: "construct:" + slug(c), kind: k, paid: pay.Total, cost: d.Cost.String()}
		switch k {
		case kBuildFree:
			if d.Color == catalog.Brown && d.Cost.Free() && pay.Total == 0 && !pay.ChainFree && as.try(s, n, build, nil) {
				return true
			}
		case kBuildOwn:
			own := 0
			for _, u := range pay.Own {
				own += int(u)
			}
			ok := d.Color == catalog.Red && !pay.ChainFree && pay.Total == 0 && d.Cost.Units() > 0 && own == d.Cost.Units()
			moved := func(pre, post *engine.State, _ []engine.Event) bool { return post.Military > pre.Military }
			if ok && as.try(s, n, build, moved) {
				return true
			}
		case kTrade:
			if !pay.ChainFree && pay.TradeTotal > 0 && pay.Affordable && as.try(s, n, build, nil) {
				return true
			}
		case kDiscard:
			if as.try(s, n, step{seat: 0, key: "discard:" + slug(c), kind: k}, nil) {
				return true
			}
		case kWonder:
			w := catalog.Wonder(as.wonder)
			wp := engine.WonderPayment(s, 0, as.wonder)
			st := step{seat: 0, key: "wonder:" + w.Slug + ":" + slug(c), kind: k, paid: wp.Total, cost: w.Cost.String()}
			if ownsWonder(s, 0, as.wonder) && wp.Affordable && as.try(s, n, st, nil) {
				return true
			}
		case kScience:
			if d.Color == catalog.Green && !pay.ChainFree && pay.Affordable && as.try(s, n, build, nil) {
				return true
			}
		case kChain:
			if d.ChainOut != catalog.NoChain && !pay.ChainFree && pay.Affordable && as.try(s, n, build, nil) {
				return true
			}
		}
	}
	return false
}

func (as *ageSearch) bot(s *engine.State, n int) bool {
	turn := n / 2 // the player's turn this reply follows
	type cand struct {
		st   step
		cost int16
	}
	var builds, discards []cand
	for _, c := range accessible(s) {
		d := catalog.Card(c)
		q := engine.BuildingQuote(s, 1, c)
		if q.Affordable && (d.Color != catalog.Red || as.militaryTaught(turn+1)) {
			builds = append(builds, cand{step{seat: 1, key: "construct:" + slug(c), paid: q.Total, cost: d.Cost.String()}, q.Total})
		}
		discards = append(discards, cand{st: step{seat: 1, key: "discard:" + slug(c)}})
	}
	slices.SortStableFunc(builds, func(a, b cand) int { return int(a.cost - b.cost) })
	var check func(pre, post *engine.State, ev []engine.Event) bool
	if n == 1 {
		// The first reply uncovers a card for the reveal lesson.
		check = func(_, _ *engine.State, ev []engine.Event) bool {
			return slices.ContainsFunc(ev, func(e engine.Event) bool { return e.Kind == engine.EvReveal })
		}
	}
	for _, c := range append(builds, discards...) {
		if as.try(s, n, c.st, check) {
			return true
		}
	}
	return false
}

// revealCard is the card the reveal lesson points at: one the first reply
// turned face up, preferably one the player takes later.
func (as *ageSearch) revealCard() catalog.CardID {
	rs := as.path[1].reveal
	for _, st := range as.path[2:] {
		if st.seat != 0 {
			continue
		}
		for _, r := range rs {
			if strings.HasSuffix(st.key, ":"+slug(r)) {
				return r
			}
		}
	}
	return rs[0]
}

// script builds the Script of a successful search, with the lessons placed
// as the design places them: each info lesson on the turn that needs it, a
// follow-up lesson on the player's next turn.
func (as *ageSearch) script(seed uint64, draft []Entry) *Script {
	sc := &Script{Name: "first-game", Seed: seed, Options: searchOpts, Human: 0}
	sc.Entries = append(sc.Entries, draft...)
	var follow []wire.TutorialLesson
	for i, st := range as.path {
		if st.seat == 1 {
			sc.Entries = append(sc.Entries, bot(st.key))
			continue
		}
		turn := i / 2
		lessons := slices.Clone(follow)
		follow = nil
		if turn == 1 {
			lessons = append(lessons, infoCard(wire.TutorialLessonReveal, slug(as.revealCard())))
		}
		switch st.kind {
		case kBuildFree:
			lessons = append(lessons, info(wire.TutorialLessonAgeOne), info(wire.TutorialLessonTurn))
		case kDiscard:
			lessons = append(lessons, info(wire.TutorialLessonCoins))
		case kWonder:
			lessons = append(lessons, info(wire.TutorialLessonWonders))
		case kBuildOwn:
			follow = append(follow, info(wire.TutorialLessonMilitary))
		case kScience:
			follow = append(follow, info(wire.TutorialLessonTokens))
		}
		sc.Entries = append(sc.Entries, you(st.key, kindLesson[st.kind], lessons...))
	}
	follow = append(follow, info(wire.TutorialLessonPlayOn))
	sc.Outro = follow
	return sc
}

// notes is the readable listing of the Age I search's path.
func (as *ageSearch) notes(seed uint64) string {
	var b strings.Builder
	fmt.Fprintf(&b, "seed %d, wonder %s; reveal %s\n", seed, catalog.Wonder(as.wonder).Slug, slug(as.revealCard()))
	for _, st := range as.path {
		who := "you"
		if st.seat == 1 {
			who = "bot"
		}
		fmt.Fprintf(&b, "  %s %-36s cost %-8s paid %d reveals %v\n", who, st.key, st.cost, st.paid, slugs(st.reveal))
	}
	fmt.Fprintf(&b, "  end: coins %d/%d, pawn %d, cards left %d\n", as.end.Players[0].Coins, as.end.Players[1].Coins, as.end.Military, cardsLeft(&as.end))
	return b.String()
}

func slugs(cs []catalog.CardID) []string {
	out := make([]string, 0, len(cs))
	for _, c := range cs {
		out = append(out, slug(c))
	}
	return out
}

// goLiteral prints sc as the Go literal script.go holds.
func goLiteral(sc *Script) string {
	var b strings.Builder
	lesson := func(l wire.TutorialLesson) string {
		name := "wire.TutorialLesson" + strings.ToUpper(string(l.ID[:1])) + string(l.ID[1:])
		if l.Card != nil {
			return fmt.Sprintf("infoCard(%s, %q)", name, *l.Card)
		}
		return "info(" + name + ")"
	}
	fmt.Fprintf(&b, "var FirstGame = Script{\n\tName:    %q,\n\tSeed:    %d,\n\tOptions: engine.Options{FirstPlayer: engine.FirstPlayerSeat0},\n\tHuman:   0,\n\tEntries: []Entry{\n", sc.Name, sc.Seed)
	for _, e := range sc.Entries {
		if e.Action == nil {
			fmt.Fprintf(&b, "\t\tbot(%q),\n", e.Key)
			continue
		}
		args := []string{strconv.Quote(e.Key), "wire.TutorialLesson" + strings.ToUpper(string(e.Action.ID[:1])) + string(e.Action.ID[1:])}
		for _, l := range e.Lessons {
			args = append(args, lesson(l))
		}
		fmt.Fprintf(&b, "\t\tyou(%s),\n", strings.Join(args, ", "))
	}
	b.WriteString("\t},\n\tOutro: []wire.TutorialLesson{")
	for i, l := range sc.Outro {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(lesson(l))
	}
	b.WriteString("},\n}\n")
	return b.String()
}
