package wire

// This file defines the tutorial object of a state frame (State.Tutorial):
// where a tutorial game stands for its player and what to teach now. The
// room derives it from the tutorial's script and the action log (package
// tutorial), so it carries ids, slugs and keys only; the client words every
// lesson.

// TutorialStatus is where a tutorial game stands for its player.
type TutorialStatus string

// The tutorial statuses.
const (
	TutorialStatusGuided   TutorialStatus = "guided"   // on the script, a guided turn ahead
	TutorialStatusFree     TutorialStatus = "free"     // the guided part is over: finished, or the player left the path
	TutorialStatusComplete TutorialStatus = "complete" // Age II is in play, or the game is over
)

// TutorialLessonID names one lesson; the client words it.
type TutorialLessonID string

// The lessons.
const (
	TutorialLessonDraftIntro     TutorialLessonID = "draftIntro"     // info: the wonder draft
	TutorialLessonDraftRound2    TutorialLessonID = "draftRound2"    // info: the second draft round
	TutorialLessonDraftPick      TutorialLessonID = "draftPick"      // action: take a wonder in the draft
	TutorialLessonDraftPickAgain TutorialLessonID = "draftPickAgain" // action: take another wonder
	TutorialLessonAgeOne         TutorialLessonID = "ageOne"         // info: the card structure
	TutorialLessonTurn           TutorialLessonID = "turn"           // info: what a turn is
	TutorialLessonBuildFree      TutorialLessonID = "buildFree"      // action: build a free card
	TutorialLessonReveal         TutorialLessonID = "reveal"         // info: an uncovered card turns face up
	TutorialLessonBuildOwn       TutorialLessonID = "buildOwn"       // action: build a card paid with your own production
	TutorialLessonMilitary       TutorialLessonID = "military"       // info: the conflict pawn
	TutorialLessonTrade          TutorialLessonID = "trade"          // action: build a card buying a resource from the bank
	TutorialLessonCoins          TutorialLessonID = "coins"          // info: coins
	TutorialLessonDiscard        TutorialLessonID = "discard"        // action: sell a card for coins
	TutorialLessonWonders        TutorialLessonID = "wonders"        // info: building a wonder
	TutorialLessonBuildWonder    TutorialLessonID = "buildWonder"    // action: build a wonder
	TutorialLessonScience        TutorialLessonID = "science"        // action: build a green card
	TutorialLessonTokens         TutorialLessonID = "tokens"         // info: the progress tokens
	TutorialLessonChain          TutorialLessonID = "chain"          // action: build a card with a chain symbol
	TutorialLessonPlayOn         TutorialLessonID = "playOn"         // info: the guided part is over, play on
	TutorialLessonLeftPath       TutorialLessonID = "leftPath"       // info: the player left the script
)

// TutorialLesson is one info lesson: its id and the card or wonder it names
// or points at (catalog slugs).
type TutorialLesson struct {
	ID     TutorialLessonID `json:"id"`
	Card   *string          `json:"card,omitempty"`
	Wonder *string          `json:"wonder,omitempty"`
}

// TutorialAction is the guided turn's move: the lesson that explains it and
// the one action key the turn asks for (a key of the view's choices), with
// the card and wonder the key names (catalog slugs).
type TutorialAction struct {
	ID     TutorialLessonID `json:"id"`
	Key    string           `json:"key"`
	Card   *string          `json:"card,omitempty"`
	Wonder *string          `json:"wonder,omitempty"`
}

// TutorialState is the tutorial as the player's frame carries it, derived
// from the script and the room's action log (package tutorial).
type TutorialState struct {
	Script string         `json:"script"` // the script's name, such as "first-game"
	Status TutorialStatus `json:"status"`
	Left   bool           `json:"left"`  // the log left the script before its end
	Turn   int            `json:"turn"`  // the player's guided turn, 1-based; 0 outside the guided part
	Turns  int            `json:"turns"` // the player's guided turns in the script; 0 outside the guided part
	// Lessons are the info lessons to show now, in order; never nil.
	Lessons []TutorialLesson `json:"lessons"`
	// Action is the guided move, while guided and the player is to move.
	Action *TutorialAction `json:"action,omitempty"`
}
