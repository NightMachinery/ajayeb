package tutorial

import (
	"strings"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// Script is a tutorial: the seed and options of its game, the player's
// seat, the opening's action keys with the lessons of the player's turns,
// and the lessons shown once the script is done. Its JSON form is what the
// development endpoint GET /api/dev/tutorial returns.
type Script struct {
	Name string `json:"name"`
	// Seed is a decimal string, like the room file's, so a JavaScript
	// reader keeps all 64 bits.
	Seed    uint64         `json:"seed,string"`
	Options engine.Options `json:"options"`
	Human   engine.Seat    `json:"human"`
	Entries []Entry        `json:"entries"`
	// Outro are the info lessons of the first position after the script,
	// with the player to move.
	Outro []wire.TutorialLesson `json:"outro"`
}

// Entry is one action of the script. Action is set exactly on the player's
// entries, with Action.Key equal to Key; Lessons are the info lessons shown
// before it (never nil, empty on the bot's entries).
type Entry struct {
	Seat    engine.Seat           `json:"seat"`
	Key     string                `json:"key"`
	Lessons []wire.TutorialLesson `json:"lessons"`
	Action  *wire.TutorialAction  `json:"action,omitempty"`
}

// Turns returns the number of the player's entries: the guided turns.
func (sc *Script) Turns() int { return sc.humanBefore(len(sc.Entries)) }

// humanBefore returns the number of the player's entries before entry p.
func (sc *Script) humanBefore(p int) int {
	k := 0
	for _, e := range sc.Entries[:p] {
		if e.Seat == sc.Human {
			k++
		}
	}
	return k
}

// ByName returns the script with this name, or false for an unknown name
// (a room created under a script this build no longer ships plays on as a
// normal game).
func ByName(name string) (*Script, bool) {
	if name == FirstGame.Name {
		return &FirstGame, true
	}
	return nil, false
}

// --- building literal scripts -------------------------------------------------

// info is an info lesson.
func info(id wire.TutorialLessonID) wire.TutorialLesson { return wire.TutorialLesson{ID: id} }

// infoCard is an info lesson that points at a card (a catalog slug).
func infoCard(id wire.TutorialLessonID, card string) wire.TutorialLesson {
	return wire.TutorialLesson{ID: id, Card: &card}
}

// you is one of the player's entries (seat 0): the move key, the action
// lesson that asks for it, and the info lessons shown first. The action
// names the key's card and wonder.
func you(key string, action wire.TutorialLessonID, lessons ...wire.TutorialLesson) Entry {
	if lessons == nil {
		lessons = []wire.TutorialLesson{}
	}
	a := &wire.TutorialAction{ID: action, Key: key}
	a.Card, a.Wonder = keySlugs(key)
	return Entry{Seat: 0, Key: key, Lessons: lessons, Action: a}
}

// bot is one of the bot's entries (seat 1).
func bot(key string) Entry {
	return Entry{Seat: 1, Key: key, Lessons: []wire.TutorialLesson{}}
}

// keySlugs returns the card and the wonder an action key names, from the
// key's text (see engine's action keys): draft:<wonder>, construct:<card>,
// discard:<card> and wonder:<wonder>:<card>.
func keySlugs(key string) (card, wonder *string) {
	parts := strings.Split(key, ":")
	switch {
	case len(parts) == 2 && parts[0] == "draft":
		wonder = &parts[1]
	case len(parts) == 2 && (parts[0] == "construct" || parts[0] == "discard"):
		card = &parts[1]
	case len(parts) == 3 && parts[0] == "wonder":
		wonder, card = &parts[1], &parts[2]
	}
	return card, wonder
}

// FirstGame is the tutorial "first-game": the player's three draft picks
// and seven Age I turns on seed 82, with the bot's replies in between. The
// search in search_test.go found it (plan "chain-first"); script_test.go
// checks every lesson's precondition on the real deal. Never edit it in
// place once shipped (see doc.go).
//
// The design ordered science before chain. On no seed can the player then
// pay for the science card (search_test.go's searchPlans explains the
// coins), so the chain lesson comes first, on the Tavern, whose 4 coins pay
// for the green card, and the tokens lesson moves to the outro.
var FirstGame = Script{
	Name:    "first-game",
	Seed:    82,
	Options: engine.Options{FirstPlayer: engine.FirstPlayerSeat0},
	Human:   0,
	Entries: []Entry{
		// The draft. Round 1 offers temple-of-artemis, mausoleum,
		// great-library and piraeus; round 2 hanging-gardens, appian-way,
		// great-lighthouse and statue-of-zeus. The player needs the Great
		// Lighthouse for the wonder lesson: it gives no extra turn and opens
		// no choice.
		you("draft:temple-of-artemis", wire.TutorialLessonDraftPick, info(wire.TutorialLessonDraftIntro)),
		bot("draft:piraeus"),
		bot("draft:mausoleum"),
		// great-library, the last of round 1, comes to the player.
		bot("draft:hanging-gardens"), // the bot opens round 2
		you("draft:great-lighthouse", wire.TutorialLessonDraftPick, info(wire.TutorialLessonDraftRound2)),
		you("draft:appian-way", wire.TutorialLessonDraftPickAgain),
		// statue-of-zeus, the last of round 2, goes to the bot. Age I
		// begins, the player first.

		// Lumber Yard: brown, free, makes wood.
		you("construct:lumber-yard", wire.TutorialLessonBuildFree, info(wire.TutorialLessonAgeOne), info(wire.TutorialLessonTurn)),
		// Clay Reserve (3 coins) uncovers the Baths, the reveal lesson's
		// card, which the trade lesson builds two turns later.
		bot("construct:clay-reserve"),
		// Stable: red, costs one wood, paid with the Lumber Yard's (0
		// coins). Its shield moves the pawn one space toward the bot.
		you("construct:stable", wire.TutorialLessonBuildOwn, infoCard(wire.TutorialLessonReveal, "baths")),
		bot("construct:theater"), // free; no red card before the military lesson
		// Baths: costs one stone, which the player lacks: bought for 2.
		you("construct:baths", wire.TutorialLessonTrade, info(wire.TutorialLessonMilitary)),
		bot("construct:clay-pool"), // free
		// Stone Pit, sold for 2 coins.
		you("discard:stone-pit", wire.TutorialLessonDiscard, info(wire.TutorialLessonCoins)),
		bot("construct:altar"), // free
		// Great Lighthouse (wood, stone and two papyrus): the Lumber Yard's
		// wood and 6 coins for the rest; the Glassworks goes under it.
		you("wonder:great-lighthouse:glassworks", wire.TutorialLessonBuildWonder, info(wire.TutorialLessonWonders)),
		bot("construct:garrison"), // one clay, the bot's Clay Pool
		// Tavern: free, shows the amphora chain symbol, and gives 4 coins.
		you("construct:tavern", wire.TutorialLessonChain),
		bot("construct:wood-reserve"), // 3 coins
		// Scriptorium: green, 2 coins, the quill science symbol.
		you("construct:scriptorium", wire.TutorialLessonScience),
		bot("construct:guard-tower"), // free
		// The player is to move, in Age I, with 6 cards left.
	},
	Outro: []wire.TutorialLesson{info(wire.TutorialLessonTokens), info(wire.TutorialLessonPlayOn)},
}

// Current is the script a new tutorial room plays.
var Current = &FirstGame
