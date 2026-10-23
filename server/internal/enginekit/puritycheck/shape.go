package puritycheck

import "reflect"

// FixedShapeViolations lists every part of t that is not allowed in a fixed
// value type: anything but bools, fixed-width integers, arrays and structs.
// Platform-sized ints, floats, strings, pointers, slices, maps, interfaces,
// functions and channels are all rejected. Each game supplies its checked root
// types and paths, and keeps size limits and comparability assertions local.
func FixedShapeViolations(t reflect.Type, path string) []string {
	switch t.Kind() {
	case reflect.Bool,
		reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return nil
	case reflect.Array:
		return FixedShapeViolations(t.Elem(), path+"[]")
	case reflect.Struct:
		var out []string
		for i := range t.NumField() {
			f := t.Field(i)
			out = append(out, FixedShapeViolations(f.Type, path+"."+f.Name)...)
		}
		return out
	default:
		return []string{path + " is a " + t.Kind().String()}
	}
}
