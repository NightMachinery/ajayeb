package wire

import hview "github.com/NightMachinery/ajayeb/server/internal/haftgane/view"

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

// TableView is the real engine's whitelisted screen facts beside the
// frame's point and events. All nested values are seedless view DTOs.
type TableView struct {
	Phase       string        `json:"phase"`
	Age         int           `json:"age"`
	Turn        int           `json:"turn"`
	Pass        string        `json:"pass"`
	Pile        int           `json:"pile"`
	Seats       []TableCity   `json:"seats"`
	You         *TablePrivate `json:"you,omitempty"`
	Result      *TableResult  `json:"result,omitempty"`
	Reveal      *TableReveal  `json:"reveal,omitempty"`
	CanDownload bool          `json:"canDownload"`
}

type TableCity = hview.Seat
type TablePlay = hview.Play
type TablePrivate = hview.Private
type TableChoice = hview.Choice
type TableOffer = hview.OfferEntry
type TableSeen = hview.SeenEntry
type TableResult = hview.Result
type TableScore = hview.Score
type TableReveal = hview.Reveal

// TableEvent preserves the real filtered event and the platform's explicit
// pass direction (derived from Age only for hands-passed).
type TableEvent struct {
	Kind    string `json:"kind"`
	Seat    int    `json:"seat"`
	Other   int    `json:"other"`
	Age     int    `json:"age"`
	Turn    int    `json:"turn"`
	Play    string `json:"play"`
	Design  string `json:"design"`
	Stage   int    `json:"stage"`
	Side    string `json:"side"`
	Left    int    `json:"left"`
	Right   int    `json:"right"`
	Printed int    `json:"printed"`
	Delta   int    `json:"delta"`
	Source  string `json:"source"`
	Outcome string `json:"outcome"`
	Pass    string `json:"pass"`
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
