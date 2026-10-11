package view

// HintSettings are a game's hint switches (docs/dev/decisions.md D23): one
// bool per hint family, true when the family is on. They belong to the
// room, not to the engine's options, so they never change the rules or a
// replay: package room keeps them with the room file and sends the same
// settings to every connection in RoomInfo.hints, and the game's admin
// may change them at any time while the game runs (the wire's "hints"
// frame). The families are those of the hints design; the defaults (every
// family on, except advisor outside games against the bot) are
// room.DefaultHints.
//
// A struct, not a map, so the client gets a fixed shape and the order of
// the families never depends on map iteration. A new family is a new
// field: a room file written before it existed reads it as false, so the
// field's arrival must also teach package room's loader its default.
type HintSettings struct {
	Prices   bool `json:"prices"`
	Economy  bool `json:"economy"`
	Science  bool `json:"science"`
	Military bool `json:"military"`
	Threats  bool `json:"threats"`
	Tempo    bool `json:"tempo"`
	Wonders  bool `json:"wonders"`
	Tokens   bool `json:"tokens"`
	Scoring  bool `json:"scoring"`
	Counting bool `json:"counting"`
	Recap    bool `json:"recap"`
	// Advisor is the post-move review, shown only to the player who moved.
	Advisor bool `json:"advisor"`
}
