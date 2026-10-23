package roles

import (
	"encoding/json"
	"errors"
	"slices"
	"testing"
	"time"
)

var t0 = time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC)

func present(users ...string) func(string) bool {
	return func(u string) bool { return slices.Contains(users, u) }
}

func roleOf(t *testing.T, r *Roles, u string) string {
	t.Helper()
	switch {
	case r.Real(u):
		return RoleAdmin
	case r.Temporary == u:
		return RoleTemporary
	}
	return RoleHuman
}

// TestPromoteDemote: any real admin promotes any human in the room; an
// admin demotes only those they promoted, the creator anyone, and nobody
// the creator; a demoted admin's own promotees stay; anyone real may step
// down, the creator too; a temporary admin promotes and demotes nothing.
func TestPromoteDemote(t *testing.T) {
	r := New("c")
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	want := func(err, target error) {
		t.Helper()
		if !errors.Is(err, target) {
			t.Fatalf("got %v, want %v", err, target)
		}
	}
	_, err := r.Promote("c", "a", true, 0)
	must(err)
	_, err = r.Promote("a", "b", true, 0)
	must(err)
	_, err = r.Promote("x", "y", true, 0)
	want(err, ErrNotAdmin)
	_, err = r.Promote("c", "gone", false, 0)
	want(err, ErrBadRequest)
	_, err = r.Promote("c", "a", true, 0)
	want(err, ErrBadRequest)
	if by := r.PromotedBy("b"); by == nil || *by != "a" {
		t.Fatalf("b promoted by %v", by)
	}
	_, err = r.Demote("b", "a", 0)
	want(err, ErrNotAdmin)
	_, err = r.Demote("a", "c", 0)
	want(err, ErrForbidden)
	_, err = r.Demote("c", "nobody", 0)
	want(err, ErrBadRequest)
	e, err := r.Demote("c", "a", 3)
	must(err)
	if e.Kind != "role" || e.At != 3 || e.By != "c" || *e.UserID != "a" || *e.Role != RoleHuman {
		t.Fatalf("entry %+v", e)
	}
	if !r.Real("b") {
		t.Fatal("demoting a removed the admin a promoted")
	}
	_, err = r.Demote("b", "b", 0)
	want(err, ErrNotAdmin) // b was promoted by a, not by itself
	_, err = r.Demote("c", "b", 0)
	must(err)
	_, err = r.StepDown("c", 0)
	must(err)
	if r.Real("c") || len(r.Admins) != 0 || r.Creator != "c" {
		t.Fatalf("after the creator stepped down: %+v", r)
	}
	_, err = r.StepDown("c", 0)
	want(err, ErrNotAdmin)

	// A temporary admin promotes, demotes and steps down nothing.
	r.Check(t0, present("t"), []string{"t"}, 0)
	if r.Temporary != "t" || !r.IsAdmin("t") || r.Real("t") {
		t.Fatalf("no temporary admin at once without real admins: %+v", r)
	}
	_, err = r.Promote("t", "u", true, 0)
	want(err, ErrNotAdmin)
	_, err = r.StepDown("t", 0)
	want(err, ErrForbidden)
	for i := range MaxAdmins {
		r.Admins = append(r.Admins, Admin{UserID: string(rune('A' + i)), PromotedBy: ptr("c")})
	}
	r.Admins[0].PromotedBy = nil
	r.Creator = r.Admins[0].UserID
	_, err = r.Promote(r.Creator, "one-more", true, 0)
	want(err, ErrForbidden)
}

func ptr(s string) *string { return &s }

// TestTemporaryAdmin walks the five-minute rules with explicit times.
func TestTemporaryAdmin(t *testing.T) {
	r := New("c")
	_, _ = r.Promote("c", "a", true, 0)
	cands := []string{"s1", "s2", "spec"}

	// Both real admins connected: nothing.
	if log, next := r.Check(t0, present("c", "a", "s1"), cands, 0); len(log) != 0 || !next.IsZero() || len(r.Away) != 0 {
		t.Fatalf("all here: %v %v %+v", log, next, r.Away)
	}
	// c leaves at t0, a at t0+2m: the role waits for a's five minutes.
	r.Check(t0, present("a", "s1"), cands, 0)
	log, next := r.Check(t0.Add(2*time.Minute), present("s1", "s2"), cands, 0)
	if len(log) != 0 || !next.Equal(t0.Add(7*time.Minute)) || r.Temporary != "" {
		t.Fatalf("both away: %v %v %q", log, next, r.Temporary)
	}
	if len(r.Away) != 2 || !r.Away[0].Since.Equal(t0) || !r.Away[1].Since.Equal(t0.Add(2*time.Minute)) {
		t.Fatalf("away times: %+v", r.Away)
	}
	// One second early: still nothing.
	if log, _ := r.Check(t0.Add(7*time.Minute-time.Second), present("s1", "s2"), cands, 0); len(log) != 0 {
		t.Fatalf("early: %v", log)
	}
	// At the deadline the first connected candidate takes the role.
	log, next = r.Check(t0.Add(7*time.Minute), present("s1", "s2"), cands, 4)
	if r.Temporary != "s1" || len(log) != 1 || *log[0].Role != RoleTemporary || *log[0].UserID != "s1" || log[0].At != 4 || !next.IsZero() {
		t.Fatalf("made: %q %v %v", r.Temporary, log, next)
	}
	// The restart keeps everything: the roles are a plain JSON value.
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var back Roles
	if err := json.Unmarshal(data, &back); err != nil || back.Temporary != "s1" || len(back.Away) != 2 || !back.Away[1].Since.Equal(t0.Add(2*time.Minute)) {
		t.Fatalf("round trip: %+v %v", back, err)
	}
	if err := back.Valid(); err != nil {
		t.Fatal(err)
	}
	r = back
	// The temporary admin leaves at 8m: the role passes on five minutes
	// later, to the next connected candidate.
	log, next = r.Check(t0.Add(8*time.Minute), present("s2", "spec"), cands, 0)
	if len(log) != 0 || !next.Equal(t0.Add(13*time.Minute)) {
		t.Fatalf("temporary away: %v %v", log, next)
	}
	log, _ = r.Check(t0.Add(13*time.Minute), present("s2", "spec"), cands, 0)
	if r.Temporary != "s2" || len(log) != 2 || *log[0].UserID != "s1" || *log[0].Role != RoleHuman || *log[1].UserID != "s2" {
		t.Fatalf("passed on: %q %v", r.Temporary, log)
	}
	// A real admin's return demotes the temporary admin at once.
	log, next = r.Check(t0.Add(14*time.Minute), present("a", "s2"), cands, 0)
	if r.Temporary != "" || len(log) != 1 || *log[0].UserID != "s2" || *log[0].Role != RoleHuman || !next.IsZero() {
		t.Fatalf("returned: %q %v %v", r.Temporary, log, next)
	}
	if len(r.Away) != 1 || r.Away[0].UserID != "c" {
		t.Fatalf("away after the return: %+v", r.Away)
	}
	// a leaves again: a fresh five minutes from its new departure.
	_, next = r.Check(t0.Add(15*time.Minute), present("s2"), cands, 0)
	if !next.Equal(t0.Add(20 * time.Minute)) {
		t.Fatalf("away again: %v", next)
	}
	// Nobody connected to take it: no role, and the next connection checks.
	if log, _ := r.Check(t0.Add(30*time.Minute), present(), cands, 0); len(log) != 0 || r.Temporary != "" {
		t.Fatalf("nobody: %v", log)
	}
	if log, _ := r.Check(t0.Add(31*time.Minute), present("spec"), cands, 0); r.Temporary != "spec" || len(log) != 1 {
		t.Fatalf("a spectator connects: %v", log)
	}
	// The last real admin stepping down... there are two; c is away. When
	// the creator alone remains and steps down, the rule applies at once.
	r2 := New("c")
	r2.Check(t0, present("c", "s1"), cands, 0)
	if _, err := r2.StepDown("c", 0); err != nil {
		t.Fatal(err)
	}
	if log, _ := r2.Check(t0, present("c", "s1"), []string{"c", "s1"}, 0); r2.Temporary != "c" || len(log) != 1 {
		t.Fatalf("after the last real admin stepped down: %q %v", r2.Temporary, log)
	}
	if roleOf(t, &r2, "c") != RoleTemporary || roleOf(t, &r2, "s1") != RoleHuman {
		t.Fatal("roles after step-down")
	}
}

// TestRefusedWrite: when a room's store refuses a role change, the room
// keeps the saved roles and the unsaved ones (ahead). Holds fails closed
// for the temporary role in each of the rule's three changes: a real
// admin's return, the role's grant and its passing on; and Carry starts
// the next check from the unsaved change, so it is made once, with its
// log entries, when a write succeeds.
func TestRefusedWrite(t *testing.T) {
	cands := []string{"s1", "s2"}
	saved := New("c")
	saved.Check(t0, present("s1", "s2"), cands, 0)

	// The grant at five minutes is refused: nobody holds the role yet.
	grant := saved.Clone()
	grant.Check(t0.Add(5*time.Minute), present("s1", "s2"), cands, 0)
	if grant.Temporary != "s1" {
		t.Fatalf("the grant: %q", grant.Temporary)
	}
	if saved.Holds("s1", &grant, t0.Add(5*time.Minute), present("s1", "s2"), cands) {
		t.Fatal("an unsaved grant gave the powers")
	}
	// Retried from the saved roles and the unsaved ones: the same grant.
	retry := saved.Clone()
	log := retry.Carry(&grant, 3)
	more, _ := retry.Check(t0.Add(5*time.Minute+Retry), present("s1", "s2"), cands, 3)
	if retry.Temporary != "s1" || len(log) != 1 || *log[0].UserID != "s1" || *log[0].Role != RoleTemporary || log[0].At != 3 || len(more) != 0 {
		t.Fatalf("the retried grant: %q %v %v", retry.Temporary, log, more)
	}
	if err := retry.Valid(); err != nil {
		t.Fatal(err)
	}
	saved = retry
	if !saved.Holds("s1", nil, t0.Add(6*time.Minute), present("s1", "s2"), cands) {
		t.Fatal("the saved grant")
	}

	// The creator's return is refused: s1 loses the powers at once.
	back := saved.Clone()
	back.Check(t0.Add(6*time.Minute), present("c", "s1", "s2"), cands, 0)
	if back.Temporary != "" {
		t.Fatalf("the return: %q", back.Temporary)
	}
	if saved.Holds("s1", &back, t0.Add(6*time.Minute), present("c", "s1", "s2"), cands) ||
		!saved.Holds("c", &back, t0.Add(6*time.Minute), present("c", "s1", "s2"), cands) {
		t.Fatal("the powers after a refused return")
	}

	// The passing on is refused, and s1 comes back before the retry: the
	// rule had given the role to s2, so neither holds it until it is
	// saved, and the retry passes it on.
	saved.Check(t0.Add(8*time.Minute), present("s2"), cands, 0)
	pass := saved.Clone()
	pass.Check(t0.Add(13*time.Minute), present("s2"), cands, 0)
	if pass.Temporary != "s2" {
		t.Fatalf("the passing on: %q", pass.Temporary)
	}
	now := t0.Add(13*time.Minute + time.Second)
	if saved.Holds("s1", &pass, now, present("s1", "s2"), cands) || saved.Holds("s2", &pass, now, present("s1", "s2"), cands) {
		t.Fatal("the powers after a refused passing on")
	}
	if !saved.Holds("s1", nil, now, present("s1", "s2"), cands) {
		t.Fatal("without the unsaved roles s1 would keep the role: the case Carry exists for")
	}
	retry = saved.Clone()
	log = retry.Carry(&pass, 0)
	more, _ = retry.Check(now, present("s1", "s2"), cands, 0)
	if retry.Temporary != "s2" || len(log) != 2 || *log[0].UserID != "s1" || *log[1].UserID != "s2" || len(more) != 0 {
		t.Fatalf("the retried passing on: %q %v %v", retry.Temporary, log, more)
	}
	if err := retry.Valid(); err != nil || len(retry.Away) != 1 || retry.Away[0].UserID != "c" {
		t.Fatalf("after the retry: %+v %v", retry.Away, err)
	}

	// An unsaved temporary admin promoted since is not carried back.
	promoted := saved.Clone()
	if _, err := promoted.Promote("c", "s2", true, 0); err != nil {
		t.Fatal(err)
	}
	if log := promoted.Carry(&pass, 0); promoted.Temporary != "" || len(log) != 1 || *log[0].UserID != "s1" {
		t.Fatalf("carried onto a promotion: %q %v", promoted.Temporary, log)
	}
}

// TestValid refuses records that break the invariants.
func TestValid(t *testing.T) {
	ok := New("c")
	_, _ = ok.Promote("c", "a", true, 0)
	ok.Check(t0, present("a"), nil, 0)
	if err := ok.Valid(); err != nil {
		t.Fatal(err)
	}
	for name, mut := range map[string]func(r *Roles){
		"duplicate admin":        func(r *Roles) { r.Admins = append(r.Admins, r.Admins[0]) },
		"empty admin":            func(r *Roles) { r.Admins[1].UserID = "" },
		"unpromoted non-creator": func(r *Roles) { r.Admins[1].PromotedBy = nil },
		"empty promoter":         func(r *Roles) { r.Admins[1].PromotedBy = ptr("") },
		"real temporary":         func(r *Roles) { r.Temporary = "a" },
		"away stranger":          func(r *Roles) { r.Away = append(r.Away, Away{UserID: "x", Since: t0}) },
		"away twice":             func(r *Roles) { r.Away = append(r.Away, r.Away[0]) },
		"away at zero":           func(r *Roles) { r.Away[0].Since = time.Time{} },
	} {
		r := ok.Clone()
		mut(&r)
		if r.Valid() == nil {
			t.Errorf("%s: valid", name)
		}
	}
}
