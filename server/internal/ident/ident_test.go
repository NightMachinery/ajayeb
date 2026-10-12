package ident_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/ident"
)

func mustToken(t *testing.T) string {
	t.Helper()
	tok, err := ident.NewToken()
	if err != nil {
		t.Fatalf("NewToken: %v", err)
	}
	return tok
}

func TestNewTokenValidates(t *testing.T) {
	for range 20 {
		tok := mustToken(t)
		if len(tok) != ident.TokenLen {
			t.Fatalf("NewToken: length %d, want %d (token %q)", len(tok), ident.TokenLen, tok)
		}
		if err := ident.ValidateToken(tok); err != nil {
			t.Fatalf("ValidateToken(%q): %v", tok, err)
		}
	}
}

func TestNewTokenUnique(t *testing.T) {
	seen := map[string]bool{}
	for range 50 {
		tok := mustToken(t)
		if seen[tok] {
			t.Fatalf("NewToken produced a repeat: %q", tok)
		}
		seen[tok] = true
	}
}

func TestValidateTokenRejects(t *testing.T) {
	cases := map[string]string{
		"empty":             "",
		"too short":         "abc",
		"too long":          strings.Repeat("a", ident.TokenLen+1),
		"padding":           strings.Repeat("A", ident.TokenLen-1) + "=",
		"standard base64":   strings.Repeat("A", ident.TokenLen-1) + "+",
		"whitespace":        strings.Repeat("A", ident.TokenLen-1) + " ",
		"slash (std b64)":   strings.Repeat("A", ident.TokenLen-1) + "/",
		"right length junk": "!!!!!!!!!!!!!!!!!!!!!!",
	}
	for name, tok := range cases {
		t.Run(name, func(t *testing.T) {
			if err := ident.ValidateToken(tok); err == nil {
				t.Fatalf("ValidateToken(%q): want error, got nil", tok)
			}
		})
	}
}

func TestUserIDDeterministic(t *testing.T) {
	secret, err := ident.LoadSecret(t.TempDir())
	if err != nil {
		t.Fatalf("LoadSecret: %v", err)
	}
	tok := mustToken(t)
	id1, err := secret.UserID(tok)
	if err != nil {
		t.Fatalf("UserID: %v", err)
	}
	id2, err := secret.UserID(tok)
	if err != nil {
		t.Fatalf("UserID: %v", err)
	}
	if id1 != id2 {
		t.Fatalf("UserID is not deterministic: %q != %q", id1, id2)
	}
	if len(id1) != 32 {
		t.Fatalf("UserID length %d, want 32", len(id1))
	}
	for _, r := range id1 {
		if !strings.ContainsRune("0123456789abcdef", r) {
			t.Fatalf("UserID %q is not lower-case hex", id1)
		}
	}
}

func TestUserIDDiffersByToken(t *testing.T) {
	secret, err := ident.LoadSecret(t.TempDir())
	if err != nil {
		t.Fatalf("LoadSecret: %v", err)
	}
	a, err := secret.UserID(mustToken(t))
	if err != nil {
		t.Fatalf("UserID: %v", err)
	}
	b, err := secret.UserID(mustToken(t))
	if err != nil {
		t.Fatalf("UserID: %v", err)
	}
	if a == b {
		t.Fatalf("two random tokens derived the same user id %q", a)
	}
}

func TestUserIDDiffersBySecret(t *testing.T) {
	s1, err := ident.LoadSecret(t.TempDir())
	if err != nil {
		t.Fatalf("LoadSecret: %v", err)
	}
	s2, err := ident.LoadSecret(t.TempDir())
	if err != nil {
		t.Fatalf("LoadSecret: %v", err)
	}
	tok := mustToken(t)
	id1, err := s1.UserID(tok)
	if err != nil {
		t.Fatalf("UserID: %v", err)
	}
	id2, err := s2.UserID(tok)
	if err != nil {
		t.Fatalf("UserID: %v", err)
	}
	if id1 == id2 {
		t.Fatalf("two independent secrets derived the same user id %q for the same token", id1)
	}
}

func TestUserIDRejectsBadToken(t *testing.T) {
	secret, err := ident.LoadSecret(t.TempDir())
	if err != nil {
		t.Fatalf("LoadSecret: %v", err)
	}
	if _, err := secret.UserID("not-a-token"); err == nil {
		t.Fatalf("UserID accepted a malformed token")
	}
}

func TestLoadSecretCreatesFile(t *testing.T) {
	dir := t.TempDir()
	if _, err := ident.LoadSecret(dir); err != nil {
		t.Fatalf("LoadSecret: %v", err)
	}
	path := filepath.Join(dir, "secret.key")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if runtime.GOOS != "windows" {
		if mode := info.Mode().Perm(); mode != 0o600 {
			t.Fatalf("secret.key mode %o, want 0600", mode)
		}
	}
	if info.Size() != 32 {
		t.Fatalf("secret.key size %d, want 32", info.Size())
	}
	// No temp files should be left behind.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "secret.key" {
		names := make([]string, len(entries))
		for i, e := range entries {
			names[i] = e.Name()
		}
		t.Fatalf("dir holds %v, want only secret.key", names)
	}
}

func TestLoadSecretPersists(t *testing.T) {
	dir := t.TempDir()
	s1, err := ident.LoadSecret(dir)
	if err != nil {
		t.Fatalf("LoadSecret (first): %v", err)
	}
	s2, err := ident.LoadSecret(dir)
	if err != nil {
		t.Fatalf("LoadSecret (second): %v", err)
	}
	tok := mustToken(t)
	id1, err := s1.UserID(tok)
	if err != nil {
		t.Fatalf("UserID: %v", err)
	}
	id2, err := s2.UserID(tok)
	if err != nil {
		t.Fatalf("UserID: %v", err)
	}
	if id1 != id2 {
		t.Fatalf("a second LoadSecret on the same dir derived a different id: %q != %q", id1, id2)
	}
}

// TestLoadSecretConcurrentCreate drives many goroutines at an empty
// directory's first LoadSecret at once. Every one must agree on the same
// secret in the end (checked indirectly: they all derive the same user id
// for a fixed token), and none may see a torn or empty file.
func TestLoadSecretConcurrentCreate(t *testing.T) {
	dir := t.TempDir()
	tok := mustToken(t)
	const n = 16

	ids := make([]string, n)
	errs := make([]error, n)
	var wg sync.WaitGroup
	wg.Add(n)
	for i := range n {
		go func(i int) {
			defer wg.Done()
			secret, err := ident.LoadSecret(dir)
			if err != nil {
				errs[i] = err
				return
			}
			id, err := secret.UserID(tok)
			if err != nil {
				errs[i] = err
				return
			}
			ids[i] = id
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("goroutine %d: %v", i, err)
		}
	}
	for i := 1; i < n; i++ {
		if ids[i] != ids[0] {
			t.Fatalf("goroutine %d derived %q, goroutine 0 derived %q: the secret was not agreed on", i, ids[i], ids[0])
		}
	}
}

func mustSecret(t *testing.T) ident.Secret {
	t.Helper()
	s, err := ident.LoadSecret(t.TempDir())
	if err != nil {
		t.Fatalf("LoadSecret: %v", err)
	}
	return s
}

func TestClaimShapeAndDeterminism(t *testing.T) {
	s := mustSecret(t)
	user, err := s.UserID(mustToken(t))
	if err != nil {
		t.Fatal(err)
	}
	c1 := s.Claim("K7QX2M", user, 0)
	c2 := s.Claim("K7QX2M", user, 0)
	if c1 != c2 {
		t.Fatalf("Claim is not deterministic: %q != %q", c1, c2)
	}
	if len(c1) != ident.ClaimLen {
		t.Fatalf("Claim length %d, want %d", len(c1), ident.ClaimLen)
	}
	if err := ident.ValidateClaim(c1); err != nil {
		t.Fatalf("ValidateClaim(%q): %v", c1, err)
	}
	if !s.VerifyClaim(c1, "K7QX2M", user, 0) {
		t.Fatal("VerifyClaim refused the claim it derived")
	}
}

func TestClaimRevealsNothing(t *testing.T) {
	s := mustSecret(t)
	tok := mustToken(t)
	user, err := s.UserID(tok)
	if err != nil {
		t.Fatal(err)
	}
	c := s.Claim("K7QX2M", user, 0)
	for _, secret := range []string{tok, user, user[:ident.ClaimLen]} {
		if strings.Contains(c, secret[:8]) || strings.Contains(secret, c[:8]) {
			t.Fatalf("claim %q shares a run with %q", c, secret)
		}
	}
}

func TestClaimScopedByRoomUserEpochAndSecret(t *testing.T) {
	s := mustSecret(t)
	other := mustSecret(t)
	a, _ := s.UserID(mustToken(t))
	b, _ := s.UserID(mustToken(t))
	c := s.Claim("K7QX2M", a, 0)
	cases := map[string]bool{
		"same":         s.VerifyClaim(c, "K7QX2M", a, 0),
		"other room":   s.VerifyClaim(c, "K7QX2N", a, 0),
		"other user":   s.VerifyClaim(c, "K7QX2M", b, 0),
		"next epoch":   s.VerifyClaim(c, "K7QX2M", a, 1),
		"other secret": other.VerifyClaim(c, "K7QX2M", a, 0),
		"no user":      s.VerifyClaim(c, "K7QX2M", "", 0),
	}
	for name, ok := range cases {
		if ok != (name == "same") {
			t.Errorf("%s: VerifyClaim = %v", name, ok)
		}
	}
	// A reset (the next epoch) derives a new claim, which works, and the
	// old one stops working.
	next := s.Claim("K7QX2M", a, 1)
	if next == c {
		t.Fatal("a new epoch derived the same claim")
	}
	if !s.VerifyClaim(next, "K7QX2M", a, 1) || s.VerifyClaim(c, "K7QX2M", a, 1) {
		t.Fatal("the epoch change did not revoke the old claim")
	}
}

func TestValidateClaimRejects(t *testing.T) {
	cases := map[string]string{
		"empty":     "",
		"too short": strings.Repeat("A", ident.ClaimLen-1),
		"too long":  strings.Repeat("A", ident.ClaimLen+1),
		"plus":      strings.Repeat("A", ident.ClaimLen-1) + "+",
		"slash":     strings.Repeat("A", ident.ClaimLen-1) + "/",
		"padding":   strings.Repeat("A", ident.ClaimLen-1) + "=",
		"space":     strings.Repeat("A", ident.ClaimLen-1) + " ",
		"unicode":   strings.Repeat("A", ident.ClaimLen-2) + "é",
	}
	s := mustSecret(t)
	for name, c := range cases {
		if err := ident.ValidateClaim(c); err == nil {
			t.Errorf("%s: ValidateClaim(%q) accepted it", name, c)
		}
		if s.VerifyClaim(c, "K7QX2M", "u", 0) {
			t.Errorf("%s: VerifyClaim accepted %q", name, c)
		}
	}
}
