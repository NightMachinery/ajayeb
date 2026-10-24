package wire

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

func (TablePick) isClientMessage()      {}
func (TableUndo) isClientMessage()      {}
func (TableSeat) isClientMessage()      {}
func (TableRole) isClientMessage()      {}
func (TableResign) isClientMessage()    {}
func (LobbyCreate) isClientMessage()    {}
func (LobbyEdit) isClientMessage()      {}
func (LobbyRole) isClientMessage()      {}
func (LobbyStart) isClientMessage()     {}
func (PlatformLink) isClientMessage()   {}
func (PlatformRelink) isClientMessage() {}
func (TableState) isServerMessage()     {}
func (TableReject) isServerMessage()    {}
func (LobbyState) isServerMessage()     {}
func (LobbyReject) isServerMessage()    {}
func (PlatformLinked) isServerMessage() {}

// SupportsProtocol keeps the Duel build constant independent of negotiation.
func SupportsProtocol(v int) bool { return v == ProtocolVersion || v == PlatformProtocolVersion }

// PlatformFamily selects only a fixed family, never a client-supplied package.
// It is also used to answer a v1 platform request with the existing error protocol.
func PlatformFamily(raw []byte) string {
	t, _ := peekType(raw)
	if strings.HasPrefix(t, "table-") {
		return "haftgane"
	}
	if strings.HasPrefix(t, "lobby-") {
		return "lobby"
	}
	return ""
}

// PlatformTarget is the closed dispatcher table for decoded platform requests.
// The empty family on common link operations means dispatch by actual room kind.
func PlatformTarget(m ClientMessage) (family, code string, seq int, ok bool) {
	switch m := m.(type) {
	case TablePick:
		return "haftgane", m.Code, m.Seq, true
	case TableUndo:
		return "haftgane", m.Code, m.Seq, true
	case TableSeat:
		return "haftgane", m.Code, m.Seq, true
	case TableRole:
		return "haftgane", m.Code, m.Seq, true
	case TableResign:
		return "haftgane", m.Code, m.Seq, true
	case LobbyEdit:
		return "lobby", m.Code, m.Seq, true
	case LobbyRole:
		return "lobby", m.Code, m.Seq, true
	case LobbyStart:
		return "lobby", m.Code, m.Seq, true
	case PlatformLink:
		return "", m.Code, 0, true
	case PlatformRelink:
		return "", m.Code, 0, true
	default:
		return "", "", 0, false
	}
}

// DecodeClientVersion leaves DecodeClient and every Duel request untouched.
// V2 adds strict table/lobby requests and the common seven-seat device links.
func DecodeClientVersion(raw []byte, version int) (ClientMessage, error) {
	if !SupportsProtocol(version) {
		return nil, frameErr("version", fmt.Errorf("unsupported version %d", version))
	}
	if version == ProtocolVersion {
		return DecodeClient(raw)
	}
	if len(raw) > MaxClientFrameBytes {
		return nil, frameErr("platform", fmt.Errorf("frame exceeds %d bytes", MaxClientFrameBytes))
	}
	t, err := peekType(raw)
	if err != nil {
		return nil, frameErr("platform", err)
	}
	// Keep all historical Duel link defaults and validation on v2 too. Only
	// a link outside that two-seat shape needs the platform decoder.
	if t == "link" || t == "relink" {
		if m, err := DecodeClient(raw); err == nil {
			return m, nil
		}
	}
	switch t {
	case "table-pick":
		return decodePlatform[TablePick](raw, t, func(m TablePick) error {
			if m.Point.Revision < 0 {
				return fmt.Errorf("negative point reference")
			}
			if err := checkNonEmptyBounded("point.id", m.Point.ID, maxActionKeyLen); err != nil {
				return err
			}
			return checkNonEmptyBounded("key", m.Key, maxActionKeyLen)
		})
	case "table-undo":
		return decodePlatform[TableUndo](raw, t, func(m TableUndo) error {
			if m.Base < 0 {
				return fmt.Errorf("negative base")
			}
			return nil
		})
	case "table-seat":
		return decodePlatform[TableSeat](raw, t, func(m TableSeat) error {
			if !platformSeat(m.Seat) {
				return fmt.Errorf("seat outside 0..6")
			}
			switch m.Control {
			case "human":
				if m.UserID == nil || m.Bot != nil {
					return fmt.Errorf("human requires only userId")
				}
				return platformID(*m.UserID)
			case "bot":
				if m.Bot == nil || m.UserID != nil {
					return fmt.Errorf("bot requires only bot")
				}
				return platformBot(*m.Bot)
			default:
				return fmt.Errorf("unknown control")
			}
		})
	case "table-role":
		return decodePlatform[TableRole](raw, t, func(m TableRole) error { return platformRole(m.Op, m.UserID) })
	case "table-resign":
		return decodePlatform[TableResign](raw, t, func(m TableResign) error {
			if !m.Confirm {
				return fmt.Errorf("resignation requires confirm true")
			}
			return nil
		})
	case "lobby-create":
		return decodePlatform[LobbyCreate](raw, t, func(m LobbyCreate) error {
			if m.SeatCount < 2 || m.SeatCount > 7 {
				return fmt.Errorf("seatCount outside 2..7")
			}
			return platformOptions(m.Options)
		})
	case "lobby-edit":
		return decodePlatform[LobbyEdit](raw, t, validateLobbyEdit)
	case "lobby-role":
		return decodePlatform[LobbyRole](raw, t, func(m LobbyRole) error { return platformRole(m.Op, m.UserID) })
	case "lobby-start":
		return decodePlatform[LobbyStart](raw, t, nil)
	case "link":
		return decodePlatform[PlatformLink](raw, t, func(m PlatformLink) error {
			if !platformSeat(m.Seat) {
				return fmt.Errorf("seat outside 0..6")
			}
			return nil
		})
	case "relink":
		return decodePlatform[PlatformRelink](raw, t, func(m PlatformRelink) error {
			if !platformSeat(m.Seat) {
				return fmt.Errorf("seat outside 0..6")
			}
			return nil
		})
	default:
		return DecodeClient(raw)
	}
}

func decodePlatform[T ClientMessage](raw []byte, t string, validate func(T) error) (ClientMessage, error) {
	var m T
	if err := strictUnmarshal(raw, &m); err != nil {
		return nil, frameErr(t, err)
	}
	// Unlike the historical Duel decoder, all new non-pointer fields must be
	// present, including zero seats, zero counters and false confirmations.
	if err := platformShape(raw, reflect.TypeOf(m)); err != nil {
		return nil, frameErr(t, err)
	}
	_, code, seq, targeted := PlatformTarget(m)
	if targeted {
		if err := checkNonEmptyBounded("code", code, maxCodeLen); err != nil {
			return nil, frameErr(t, err)
		}
		if seq < 0 {
			return nil, frameErr(t, fmt.Errorf("negative seq"))
		}
	}
	if validate != nil {
		if err := validate(m); err != nil {
			return nil, frameErr(t, err)
		}
	}
	return m, nil
}

// platformShape checks exact key spelling, duplicate keys, required fields and
// nulls recursively. Optional means absent, never null. It is independent of
// the semantic validators, so a false or zero value cannot disguise omission.
func platformShape(raw []byte, typ reflect.Type) error {
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return fmt.Errorf("null is not a field value")
	}
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		return nil
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	token, err := dec.Token()
	if err != nil || token != json.Delim('{') {
		return fmt.Errorf("expected object")
	}
	fields := make(map[string]reflect.StructField, typ.NumField())
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		fields[strings.Split(f.Tag.Get("json"), ",")[0]] = f
	}
	seen := make(map[string]bool, len(fields))
	for dec.More() {
		keyToken, err := dec.Token()
		if err != nil {
			return err
		}
		key, ok := keyToken.(string)
		if !ok {
			return fmt.Errorf("expected field name")
		}
		f, ok := fields[key]
		if !ok || seen[key] {
			return fmt.Errorf("unknown or duplicate field %q", key)
		}
		seen[key] = true
		var value json.RawMessage
		if err := dec.Decode(&value); err != nil {
			return err
		}
		if err := platformShape(value, f.Type); err != nil {
			return fmt.Errorf("%s: %w", key, err)
		}
	}
	for key, f := range fields {
		if f.Type.Kind() != reflect.Pointer && !seen[key] {
			return fmt.Errorf("missing field %q", key)
		}
	}
	return nil
}
func platformSeat(seat int) bool { return seat >= 0 && seat < 7 }
func platformID(id string) error { return checkNonEmptyBounded("userId", id, maxClientIDLen) }
func platformBot(bot string) error {
	switch bot {
	case "easy", "normal", "hard":
		return nil
	default:
		return fmt.Errorf("unknown bot difficulty")
	}
}
func platformRole(op string, user *string) error {
	switch op {
	case "step-down":
		if user != nil {
			return fmt.Errorf("step-down targets sender")
		}
		return nil
	case "promote", "demote":
		if user == nil {
			return fmt.Errorf("role operation requires userId")
		}
		return platformID(*user)
	default:
		return fmt.Errorf("unknown role operation")
	}
}
func platformOptions(o LobbyOptions) error {
	if o.Hints != nil {
		if err := checkAdvisorStrength("options.hints", o.Hints.AdvisorStrength); err != nil {
			return err
		}
	}
	if o.Sides != nil && *o.Sides != "choose" && *o.Sides != "day" && *o.Sides != "random" {
		return fmt.Errorf("unknown side rule")
	}
	if o.Seating != nil && *o.Seating != "random" && *o.Seating != "lobby" {
		return fmt.Errorf("unknown seating")
	}
	if o.FirstPlayer != nil && (*o.FirstPlayer < 0 || *o.FirstPlayer > 1) {
		return fmt.Errorf("firstPlayer outside 0..1")
	}
	return nil
}
func validateLobbyEdit(m LobbyEdit) error {
	// Match the operation's payload exactly. Extra but known fields are errors.
	expected := map[string]bool{}
	switch m.Op {
	case "seats":
		expected["seatCount"] = true
	case "options":
		expected["options"] = true
	case "add-bot":
		expected["seat"] = true
		expected["bot"] = true
	case "remove-bot", "open-reservation":
		expected["seat"] = true
	case "seat":
		expected["seat"] = true
		expected["userId"] = true
	case "remove-human":
		expected["userId"] = true
	case "release":
	default:
		return fmt.Errorf("unknown lobby edit")
	}
	actual := map[string]bool{"seatCount": m.SeatCount != nil, "options": m.Options != nil, "seat": m.Seat != nil, "userId": m.UserID != nil, "bot": m.Bot != nil}
	for key, present := range actual {
		if present != expected[key] {
			return fmt.Errorf("%s: wrong payload for %s", key, m.Op)
		}
	}
	if m.SeatCount != nil && (*m.SeatCount < 2 || *m.SeatCount > 7) {
		return fmt.Errorf("seatCount outside 2..7")
	}
	if m.Seat != nil && !platformSeat(*m.Seat) {
		return fmt.Errorf("seat outside 0..6")
	}
	if m.UserID != nil {
		if err := platformID(*m.UserID); err != nil {
			return err
		}
	}
	if m.Bot != nil {
		if err := platformBot(*m.Bot); err != nil {
			return err
		}
	}
	if m.Options != nil {
		return platformOptions(*m.Options)
	}
	return nil
}

// NewTableState initializes every required collection to a JSON array.
func NewTableState(code string) TableState {
	return TableState{T: "table-state", Game: "haftgane", Code: code, Status: "playing", Seats: []PlatformSeat{}, Humans: []PlatformHuman{}, Events: []TableEvent{}, RoomLog: []PlatformLogEntry{}, View: TableView{Seats: []TableCity{}}}
}

// NewLobbyState initializes every required collection to a JSON array.
func NewLobbyState(code string) LobbyState {
	return LobbyState{T: "lobby-state", Game: "lobby", Code: code, Seats: []PlatformSeat{}, Humans: []PlatformHuman{}, RoomLog: []PlatformLogEntry{}}
}

// NewTableReject builds a private table rejection.
func NewTableReject(seq int, code string) TableReject {
	return TableReject{T: "table-reject", Seq: seq, Code: code}
}

// NewLobbyReject builds a private lobby rejection.
func NewLobbyReject(seq int, code string) LobbyReject {
	return LobbyReject{T: "lobby-reject", Seq: seq, Code: code}
}
