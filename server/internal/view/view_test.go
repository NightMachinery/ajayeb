package view

import (
	"reflect"
	"strings"
	"testing"
	"unicode"
)

// TestDTOShape walks every type reachable from View and GameEvent and checks
// the package rules that tygo and the client rely on:
//
//   - every exported field has an explicit, unique, camelCase json name;
//   - omitempty is set exactly on pointer fields (optional in TypeScript),
//     and no other json option is used;
//   - no map, interface, function, channel, float or fixed-width integer
//     type appears (numbers are int, which JSON and JavaScript share), and
//     no slice of bytes (which encoding/json would turn into base64);
//   - every array is a pair by seat, tagged tstype:"Pair<T>" (a TypeScript
//     tuple);
//   - no field is unexported (it would silently never reach the client).
func TestDTOShape(t *testing.T) {
	seen := map[reflect.Type]bool{}
	var walk func(rt reflect.Type, where string)
	walk = func(rt reflect.Type, where string) {
		switch rt.Kind() {
		case reflect.Pointer, reflect.Slice, reflect.Array:
			if (rt.Kind() == reflect.Slice || rt.Kind() == reflect.Array) && rt.Elem().Kind() == reflect.Uint8 {
				t.Errorf("%s: %s marshals as base64; use []int or a string type", where, rt)
			}
			walk(rt.Elem(), where)
			return
		case reflect.Struct:
		case reflect.String, reflect.Bool, reflect.Int:
			return
		default:
			t.Errorf("%s: type %s (kind %s) is not allowed in a view DTO", where, rt, rt.Kind())
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
			arr := f.Type
			if isPtr {
				arr = arr.Elem()
			}
			if arr.Kind() == reflect.Array {
				want := "Pair<" + arr.Elem().Name() + ">"
				if got := f.Tag.Get("tstype"); arr.Len() != 2 || got != want {
					t.Errorf("%s: an array of %d; only pairs by seat are arrays, tagged tstype:%q (has %q)", at, arr.Len(), want, got)
				}
			}
			switch {
			case opts == "omitempty" && !isPtr:
				t.Errorf("%s: omitempty on a non-pointer field", at)
			case opts == "" && isPtr:
				t.Errorf("%s: a pointer field needs omitempty (a nil pointer would marshal as null)", at)
			case opts != "" && opts != "omitempty":
				t.Errorf("%s: json option %q is not allowed", at, opts)
			}
			walk(f.Type, at)
		}
	}
	walk(reflect.TypeFor[View](), "View")
	walk(reflect.TypeFor[GameEvent](), "GameEvent")
	if len(seen) < 20 {
		t.Fatalf("the walk reached only %d struct types", len(seen))
	}
}
