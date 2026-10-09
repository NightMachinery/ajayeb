package wire

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"unicode"

	"github.com/NightMachinery/ajayeb/server/internal/view"
)

// jsonMarshalerType is used to recognize a type with a custom MarshalJSON
// (SeatChoice): its JSON shape is whatever that method produces, not its Go
// struct fields, so the walk below treats it as a leaf instead of descending
// into (unexported) fields that are never marshaled as such.
var jsonMarshalerType = reflect.TypeFor[json.Marshaler]()

// TestFrameShape walks every type reachable from each client and server
// frame type and checks the rules doc.go and tygo.yaml both rely on:
//
//   - every exported field has an explicit, unique, camelCase json name;
//   - omitempty is set exactly on pointer fields, and no other json option
//     is used;
//   - no field is unexported (it would silently never reach the wire);
//   - a type with its own MarshalJSON is a leaf (its Go fields do not need
//     to follow the DTO field rules, since they are never marshaled as
//     such).
func TestFrameShape(t *testing.T) {
	seen := map[reflect.Type]bool{}
	var walk func(rt reflect.Type, where string)
	walk = func(rt reflect.Type, where string) {
		switch rt.Kind() {
		case reflect.Pointer, reflect.Slice, reflect.Array:
			walk(rt.Elem(), where)
			return
		case reflect.Struct:
			if reflect.PointerTo(rt).Implements(jsonMarshalerType) || rt.Implements(jsonMarshalerType) {
				return
			}
		case reflect.String, reflect.Bool, reflect.Int:
			return
		default:
			t.Errorf("%s: type %s (kind %s) is not expected in a wire frame", where, rt, rt.Kind())
			return
		}
		if seen[rt] {
			return
		}
		seen[rt] = true
		names := map[string]string{}
		for i := range rt.NumField() {
			f := rt.Field(i)
			at := rt.Name() + "." + f.Name
			if !f.IsExported() {
				t.Errorf("%s is unexported", at)
				continue
			}
			tag, ok := f.Tag.Lookup("json")
			if !ok {
				t.Errorf("%s has no json tag", at)
				continue
			}
			name, opts, _ := strings.Cut(tag, ",")
			if name == "" || name == "-" {
				t.Errorf("%s: json name %q", at, name)
			}
			if r := []rune(name); len(r) > 0 && !unicode.IsLower(r[0]) {
				t.Errorf("%s: json name %q is not camelCase", at, name)
			}
			if strings.ContainsAny(name, "-_ ") {
				t.Errorf("%s: json name %q is not camelCase", at, name)
			}
			if prev, dup := names[name]; dup {
				t.Errorf("%s: json name %q is also used by %s", at, name, prev)
			}
			names[name] = f.Name

			isPtr := f.Type.Kind() == reflect.Pointer
			switch {
			case opts == "omitempty" && !isPtr:
				t.Errorf("%s: omitempty on a non-pointer field", at)
			case opts == "" && isPtr:
				t.Errorf("%s: a pointer field needs omitempty (a nil pointer would marshal as null)", at)
			case opts != "" && opts != "omitempty":
				t.Errorf("%s: json option %q is not allowed", at, opts)
			}

			if f.Name == "T" && name == "t" {
				tstype, ok := f.Tag.Lookup("tstype")
				if !ok {
					t.Errorf("%s: the \"t\" discriminator needs a tstype literal tag", at)
				} else if lit, ok := discriminator(tstype); !ok {
					t.Errorf("%s: tstype %q is not a quoted literal like 'hello'", at, tstype)
				} else if want := strings.ToLower(rt.Name()); lit != want {
					t.Errorf("%s: tstype literal %q, want %q (the type's own name, lower-cased)", at, lit, want)
				}
			}

			walk(f.Type, at)
		}
	}

	for _, m := range clientMessageTypes {
		walk(reflect.TypeOf(m), reflect.TypeOf(m).Name())
	}
	for _, m := range serverMessageTypes {
		walk(reflect.TypeOf(m), reflect.TypeOf(m).Name())
	}
	if len(seen) < 10 {
		t.Fatalf("the walk reached only %d struct types", len(seen))
	}
}

// TestDiscriminatorValues checks that every frame actually built through its
// NewXxx constructor carries the exact "t" value its type name implies
// (lower-cased), independent of the tag-based check in TestFrameShape: this
// one catches a NewXxx that set T to the wrong literal by hand.
func TestDiscriminatorValues(t *testing.T) {
	check := func(v any) {
		rv := reflect.ValueOf(v)
		got := rv.FieldByName("T").String()
		want := strings.ToLower(rv.Type().Name())
		if got != want {
			t.Errorf("%s: T = %q, want %q", rv.Type().Name(), got, want)
		}
	}
	for _, m := range clientMessageTypes {
		check(m)
	}
	for _, m := range serverMessageTypes {
		check(m)
	}
}

// TestNoNilSlices checks that the slice fields wire itself is responsible
// for (as opposed to view.View, whose own "never nil" rule is the view and
// engine packages' concern, exercised there with a real ViewFor) marshal as
// [] rather than null when their NewXxx constructor is given no slice
// arguments.
func TestNoNilSlices(t *testing.T) {
	assertArray := func(name string, data []byte) {
		t.Helper()
		var raw json.RawMessage
		if err := json.Unmarshal(data, &raw); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if string(raw) == "null" {
			t.Errorf("%s marshaled as null, want []", name)
		}
	}

	create := NewCreate(ModeOnline, nil, SeatRandom, nil)
	data, err := json.Marshal(create)
	if err != nil {
		t.Fatalf("Marshal(Create): %v", err)
	}
	var decodedCreate struct {
		Names json.RawMessage `json:"names"`
	}
	if err := json.Unmarshal(data, &decodedCreate); err != nil {
		t.Fatalf("Unmarshal(Create): %v", err)
	}
	assertArray("Create.Names", decodedCreate.Names)

	state := NewState("K7QX2M", 1, NewRoomInfo(ModeHotseat, RoomStatusPlaying, nil, 0), NewYouInfo(nil, false), nil, nil, view.View{})
	data, err = json.Marshal(state)
	if err != nil {
		t.Fatalf("Marshal(State): %v", err)
	}
	var decodedState struct {
		Room struct {
			Seats json.RawMessage `json:"seats"`
		} `json:"room"`
		You struct {
			Seats json.RawMessage `json:"seats"`
		} `json:"you"`
		Events json.RawMessage `json:"events"`
	}
	if err := json.Unmarshal(data, &decodedState); err != nil {
		t.Fatalf("Unmarshal(State): %v", err)
	}
	assertArray("State.Room.Seats", decodedState.Room.Seats)
	assertArray("State.You.Seats", decodedState.You.Seats)
	assertArray("State.Events", decodedState.Events)
}
