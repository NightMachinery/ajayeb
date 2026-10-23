package leakcheck_test

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/tabletest/leakcheck"
)

func TestSecretScanEmbeddedAndExemptPaths(t *testing.T) {
	for _, secret := range []string{"0123456789abcdef", "123456789", "permitted-looking-text", "123"} {
		for _, value := range []any{secret, "prefix " + secret + " suffix", json.Number(secret)} {
			// A numeric token is valid JSON only for an all-decimal secret.
			data, err := json.Marshal(map[string]any{"x": value})
			if err != nil {
				continue
			}
			if secret == "123" && value == "prefix 123 suffix" {
				continue
			} // short secrets match whole values only.
			truth := &leakcheck.Truth{Secrets: []leakcheck.Secret{{Name: "fixture secret", Value: secret}}}
			checker := leakcheck.New(truth, leakcheck.Config{HexPaths: []string{"x"}})
			if got := checker.Check(leakcheck.Doc{Kind: leakcheck.Frame, Data: data}); !slices.ContainsFunc(got, func(l leakcheck.Leak) bool { return l.Rule == leakcheck.Hash }) {
				t.Fatalf("secret escaped at an otherwise hex-exempt path: %s", data)
			}
		}
	}
}

// Memoized string facts still apply exemptions at each occurrence's path.
func TestStringFactsKeepPathEntitlements(t *testing.T) {
	checker := leakcheck.New(&leakcheck.Truth{}, leakcheck.Config{KeyPaths: []string{"own"}})
	clean := []byte(`{"own":"build:lumber-yard:0:0"}`)
	if got := checker.Check(leakcheck.Doc{Kind: leakcheck.Frame, Data: clean}); len(got) != 0 {
		t.Fatal(got)
	}
	dirty := []byte(`{"other":"build:lumber-yard:0:0"}`)
	if got := checker.Check(leakcheck.Doc{Kind: leakcheck.Frame, Data: dirty}); !slices.ContainsFunc(got, func(l leakcheck.Leak) bool { return l.Rule == leakcheck.RawKey }) {
		t.Fatal("memoized string inherited another path's exemption", got)
	}
}
