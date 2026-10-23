package engine

import (
	"reflect"
	"strings"
	"testing"
)

// Extra sentinels run against the original guards before extraction and the
// shared guards afterwards. Keep every existing checker fixture as well.
func TestPurityNewNegativeFixtures(t *testing.T) {
	cases := []struct{ name, rel, src, want string }{
		{"clock-subpackage", "sentinel.go", `package engine; import "time/tzdata"`, "imports time/tzdata"},
		{"os-subpackage", "sentinel.go", `package engine; import "os/signal"`, "imports os/signal"},
		{"network", "sentinel.go", `package engine; import "net/url"`, "imports net/url"},
		{"logging", "sentinel.go", `package engine; import "log"`, "imports log"},
		{"synchronization", "sentinel.go", `package engine; import "sync"`, "imports sync"},
		{"unsafe-alias", "sentinel.go", `package engine; import memory "unsafe"`, "imports unsafe"},
		{"syscall", "sentinel.go", `package engine; import "syscall"`, "imports syscall"},
		{"unseeded-random", "sentinel.go", `package engine; import entropy "crypto/rand"`, "imports crypto/rand"},
		{"deprecated-io", "sentinel.go", `package engine; import "io/ioutil"`, "imports io/ioutil"},
		{"v1-random-alias", "rng.go", `package engine; import random "math/rand"`, "imports math/rand (v1)"},
		{"v2-wrong-file", "catalog/random.go", `package catalog; import random "math/rand/v2"`, "imports math/rand/v2; use the engine RNG"},
		{"map-alias", "sentinel.go", `package engine; type lookup = map[uint16]bool`, "uses a map"},
		{"map-exception-is-exact", "nested/eventfilter.go", `package nested; type lookup map[uint16]bool`, "uses a map"},
		{"hex-float", "sentinel.go", `package engine; const ratio = 0x1p2`, "floating-point literal 0x1p2"},
		{"imaginary-literal", "sentinel.go", `package engine; const imaginary = 3.25i`, "floating-point literal 3.25i"},
		{"float32-alias", "sentinel.go", `package engine; type value = float32`, "uses float32"},
		{"float64-conversion", "sentinel.go", `package engine; var value = float64(3)`, "uses float64"},
		{"complex64-alias", "sentinel.go", `package engine; type value = complex64`, "uses complex64"},
		{"complex128-alias", "sentinel.go", `package engine; type value = complex128`, "uses complex128"},
		{"parse-error", "sentinel.go", `package engine; func broken(`, "sentinel.go:"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := purityViolations(c.rel, []byte(c.src))
			if len(got) != 1 || !strings.Contains(got[0], c.want) {
				t.Fatalf("guard accepted or misreported negative fixture: got %v, want one containing %q", got, c.want)
			}
			t.Log(got[0])
		})
	}
}

func TestFixedShapeNewNegativeFixtures(t *testing.T) {
	cases := []struct {
		name string
		typ  reflect.Type
		want string
	}{
		{"platform-int", reflect.TypeFor[int](), "root is a int"},
		{"platform-uint", reflect.TypeFor[uint](), "root is a uint"},
		{"uintptr", reflect.TypeFor[uintptr](), "root is a uintptr"},
		{"float32", reflect.TypeFor[float32](), "root is a float32"},
		{"float64", reflect.TypeFor[float64](), "root is a float64"},
		{"complex64", reflect.TypeFor[complex64](), "root is a complex64"},
		{"complex128", reflect.TypeFor[complex128](), "root is a complex128"},
		{"string", reflect.TypeFor[string](), "root is a string"},
		{"slice", reflect.TypeFor[[]uint16](), "root is a slice"},
		{"map", reflect.TypeFor[map[uint16]uint8](), "root is a map"},
		{"pointer", reflect.TypeFor[*uint16](), "root is a ptr"},
		{"interface", reflect.TypeFor[any](), "root is a interface"},
		{"function", reflect.TypeFor[func(uint8)](), "root is a func"},
		{"channel", reflect.TypeFor[chan uint16](), "root is a chan"},
		{"nested-array-field", reflect.TypeFor[[3]struct{ Value *uint8 }](), "root[].Value is a ptr"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := fixedShapeViolations(c.typ, "root")
			if len(got) != 1 || got[0] != c.want {
				t.Fatalf("guard accepted or misreported negative fixture: got %v, want [%s]", got, c.want)
			}
			t.Log(got[0])
		})
	}
}
