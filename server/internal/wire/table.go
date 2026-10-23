package wire

// PointRef identifies an opening of a logical point. Revision is the existing
// durable transport revision at opening, preserved across restart, replaced on
// advance or undo. Only ID belongs in canonical rules history or bot entropy.
type PointRef struct {
	ID       string `json:"id"`
	Revision int    `json:"revision"`
}

// TablePick replaces this connection's seat's pending pick and confirmation.
// False withdraws confirmation; the final required confirmation resolves once.
type TablePick struct {
	T       string   `json:"t" tstype:"'table-pick'"`
	Code    string   `json:"code"`
	Seq     int      `json:"seq"`
	Point   PointRef `json:"point"`
	Key     string   `json:"key"`
	Confirm bool     `json:"confirm"`
}

// TableUndo requests rollback before the last completed batch containing the
// original sole human's decision. Base is the last received transport revision.
type TableUndo struct {
	T    string `json:"t" tstype:"'table-undo'"`
	Code string `json:"code"`
	Seq  int    `json:"seq"`
	Base int    `json:"base"`
}

// TableSeat requests an admin's controller switch. Human requires UserID (a
// spectator); bot requires Bot. The actor preserves the seat's pending pick.
type TableSeat struct {
	T       string  `json:"t" tstype:"'table-seat'"`
	Code    string  `json:"code"`
	Seq     int     `json:"seq"`
	Seat    int     `json:"seat"`
	Control string  `json:"control" tstype:"'human' | 'bot'"`
	UserID  *string `json:"userId,omitempty"`
	Bot     *string `json:"bot,omitempty"`
}

// TableRole changes a real admin role. Step-down targets the sender, so UserID
// must be absent. Temporary admins cannot promote or demote.
type TableRole struct {
	T      string  `json:"t" tstype:"'table-role'"`
	Code   string  `json:"code"`
	Seq    int     `json:"seq"`
	Op     string  `json:"op" tstype:"'promote' | 'demote' | 'step-down'"`
	UserID *string `json:"userId,omitempty"`
}

// TableResign requires explicit confirmation and replaces the sender with a
// bot. It never declares a terminal victory.
type TableResign struct {
	T       string `json:"t" tstype:"'table-resign'"`
	Code    string `json:"code"`
	Seq     int    `json:"seq"`
	Confirm bool   `json:"confirm"`
}

// TableView is one viewer's whitelist projection of the game, beside the
// frame's Point and Events, which carry the open point and the latest
// resolution. Until M16.haftgane-engine-contract it mirrors the fake
// engine's view (haftgane/fakeengine/fakeview) field for field; that slice
// replaces these fields with the real engine's. Never serialize engine
// state or arbitrary JSON here. Every card list is catalog design ids in
// catalog order, never a copy or deal position.
type TableView struct {
	Phase  string        `json:"phase"`
	Age    int           `json:"age"`
	Turn   int           `json:"turn"`
	Pass   string        `json:"pass"`
	Pile   int           `json:"pile"`
	Seats  []TableCity   `json:"seats"`
	You    *TablePrivate `json:"you,omitempty"`
	Result *TableResult  `json:"result,omitempty"`
	Reveal *TableReveal  `json:"reveal,omitempty"`
}

// TableCity is one seat's public city.
type TableCity struct {
	Seat     int        `json:"seat"`
	Board    string     `json:"board"`
	Coins    int        `json:"coins"`
	Built    []string   `json:"built"`
	Stages   int        `json:"stages"`
	HandSize int        `json:"handSize"`
	HasKept  bool       `json:"hasKept"`
	Last     *TablePlay `json:"last,omitempty"`
}

// TablePlay is a seat's latest play as everyone saw it: the design only
// for a build or a salvage.
type TablePlay struct {
	Kind   string `json:"kind"`
	Design string `json:"design"`
	Stage  int    `json:"stage"`
	Left   int    `json:"left"`
	Right  int    `json:"right"`
}

// TablePrivate is what only the seat's controller sees: its hand, kept and
// tucked cards, its offered keys with their payments, its private salvage
// offer during its own salvage point, and the hands it has held.
type TablePrivate struct {
	Seat    int           `json:"seat"`
	Hand    []string      `json:"hand"`
	Kept    string        `json:"kept"`
	Tucked  []string      `json:"tucked"`
	Choices []TableChoice `json:"choices"`
	Offer   []TableOffer  `json:"offer"`
	Seen    []TableSeen   `json:"seen"`
}

// TableChoice is one offered key, in the engine's canonical order.
type TableChoice struct {
	Key    string `json:"key"`
	Kind   string `json:"kind"`
	Design string `json:"design"`
	Left   int    `json:"left"`
	Right  int    `json:"right"`
}

// TableOffer is one design of the pile during the seat's own salvage point.
type TableOffer struct {
	Design string `json:"design"`
	Count  int    `json:"count"`
	Legal  bool   `json:"legal"`
}

// TableSeen is one point the seat played: the hand it held and its play.
type TableSeen struct {
	Point  string   `json:"point"`
	Hand   []string `json:"hand"`
	Played string   `json:"played"`
	Action string   `json:"action"`
}

// TableResult is the final score, public under either end policy.
type TableResult struct {
	Scores  []TableScore `json:"scores"`
	Rank    []int        `json:"rank"`
	Winners []int        `json:"winners"`
}

// TableScore is one seat's score.
type TableScore struct {
	Seat     int `json:"seat"`
	Built    int `json:"built"`
	Stages   int `json:"stages"`
	Treasure int `json:"treasure"`
	Total    int `json:"total"`
}

// TableReveal is every hidden card, sent once the game is over unless the
// table turned the final reveal off.
type TableReveal struct {
	Tucked [][]string `json:"tucked"`
	Pile   []string   `json:"pile"`
	Box    []string   `json:"box"`
}

// TableEvent is one public outcome of a resolution, filtered for the
// viewer: Design is empty where the viewer may not know the card (another
// seat's tuck, sale, leftover or kept card). Events and history describe
// outcomes, never raw submitted keys. Like TableView it mirrors the fake
// engine's event until M16.haftgane-engine-contract.
type TableEvent struct {
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

// TableHistoryEntry describes one completed batch's public outcomes.
type TableHistoryEntry struct {
	Batch  int          `json:"batch"`
	Events []TableEvent `json:"events"`
}

// TablePoint exposes only the current point and confirmation flags. A private
// change must not advance another viewer's revision or reveal edit frequency.
type TablePoint struct {
	Ref       PointRef `json:"ref"`
	Kind      string   `json:"kind"`
	Required  []int    `json:"required"`
	Confirmed []int    `json:"confirmed"`
}

// TablePending appears only in the controlling seat's own frame.
type TablePending struct {
	Key     string `json:"key"`
	Confirm bool   `json:"confirm"`
}

// TableState is a full per-viewer whitelist snapshot. Revision orders changes
// visible to this viewer (private edits advance only their owner's frame).
// Batch counts complete canonical batches; undo reduces it. StartingHumanCount
// and AllowUndo are fixed at Start; UndoOff can only become true thereafter.
type TableState struct {
	T                  string               `json:"t" tstype:"'table-state'"`
	Game               string               `json:"game" tstype:"'haftgane'"`
	Code               string               `json:"code"`
	Revision           int                  `json:"revision"`
	Batch              int                  `json:"batch"`
	Status             string               `json:"status" tstype:"'playing' | 'finished'"`
	Seats              []PlatformSeat       `json:"seats"`
	Humans             []PlatformHuman      `json:"humans"`
	You                PlatformYou          `json:"you"`
	StartingHumanCount int                  `json:"startingHumanCount"`
	AllowUndo          bool                 `json:"allowUndo"`
	UndoOff            bool                 `json:"undoOff"`
	FinalReveal        bool                 `json:"finalReveal"`
	Point              *TablePoint          `json:"point,omitempty"`
	Pending            *TablePending        `json:"pending,omitempty"`
	Ack                *PlatformAck         `json:"ack,omitempty"`
	Events             []TableEvent         `json:"events"`
	History            *[]TableHistoryEntry `json:"history,omitempty"`
	RoomLog            []PlatformLogEntry   `json:"roomLog"`
	View               TableView            `json:"view"`
}

// TableReject is private to the submitting connection. Decision-closed means
// the point reference has closed, illegal means the key was not offered, and
// stale means an older sequence would replace a newer accepted pick.
type TableReject struct {
	T    string `json:"t" tstype:"'table-reject'"`
	Seq  int    `json:"seq"`
	Code string `json:"code" tstype:"'decision-closed' | 'illegal' | 'stale' | 'bad-request' | 'not-admin' | 'forbidden' | 'rate' | 'storage'"`
}
