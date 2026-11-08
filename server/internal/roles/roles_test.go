package roles

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"
)

var t0 = time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC)

func present(users ...string) func(string) bool {
	return func(u string) bool { return slices.Contains(users, u) }
}
func mustRole(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
func wantRole(t *testing.T, err, want error) {
	t.Helper()
	if !errors.Is(err, want) {
		t.Fatalf("got %v want %v", err, want)
	}
}
func roundRole(t *testing.T, r Roles) Roles {
	t.Helper()
	data, e := json.Marshal(r)
	mustRole(t, e)
	var back Roles
	mustRole(t, json.Unmarshal(data, &back))
	mustRole(t, back.Valid())
	return back
}

func TestRoleMatrix(t *testing.T) {
	r := New("creator")
	_, e := r.StepDown("creator", 0)
	wantRole(t, e, ErrForbidden)
	_, e = r.Promote("creator", "real", true, 1)
	mustRole(t, e)
	_, e = r.Promote("real", "child", true, 2)
	mustRole(t, e)
	_, e = r.Demote("child", "creator", 3)
	wantRole(t, e, ErrForbidden)
	_, e = r.Demote("child", "real", 3)
	wantRole(t, e, ErrNotAdmin)
	_, e = r.Demote("creator", "real", 3)
	mustRole(t, e)
	if !r.Real("child") {
		t.Fatal("demoted ancestry cascaded")
	}
	_, e = r.StepDown("child", 4)
	mustRole(t, e)
	_, e = r.Promote("stranger", "t", true, 4)
	wantRole(t, e, ErrNotAdmin)
	r.Check(t0, present("t", "d", "e", "other"), []string{"t", "d", "e", "other"}, 0)
	log, next := r.Check(t0.Add(AwayLimit), present("t", "d", "e", "other"), []string{"t", "d", "e", "other"}, 5)
	if r.Temporary != "t" || len(log) != 1 || !next.IsZero() {
		t.Fatalf("grant: %+v %v", r, log)
	}
	_, e = r.Promote("t", "d", true, 6)
	mustRole(t, e)
	_, e = r.Promote("d", "e", true, 7)
	mustRole(t, e)
	_, e = r.Promote("t", "other", true, 8)
	mustRole(t, e)
	if r.Real("d") || !r.Temp("d") || *r.PromotedBy("e") != "d" {
		t.Fatal("temporary promotion became real")
	}
	_, e = r.Demote("d", "other", 9)
	wantRole(t, e, ErrNotAdmin)
	_, e = r.Demote("t", "creator", 9)
	wantRole(t, e, ErrForbidden)
	_, e = r.Demote("t", "d", 9)
	mustRole(t, e)
	if !r.Temp("e") {
		t.Fatal("temporary demotion cascaded")
	}
	r = roundRole(t, r)
	// All surviving temporaries lose authority, including descendants whose
	// promoter was demoted. Every change is publicly logged.
	log, _ = r.Check(t0.Add(6*time.Minute), present("creator", "t", "e", "other"), []string{"t", "e", "other"}, 10)
	if len(log) != 3 || r.Temp("e") || r.Temporary != "" {
		t.Fatalf("return: %+v %v", r, log)
	}
	for _, e := range log {
		if e.Role == nil || *e.Role != RoleHuman {
			t.Fatal("missing demotion log")
		}
	}
}

func TestTemporaryHistoryAndSelection(t *testing.T) {
	r := New("c")
	seated, spec := []string{"a", "b", "c2"}, []string{"watcher", "old"}
	cands := Candidates("ROOMAA", r.Elections, seated, spec)
	if !slices.Equal(cands, Candidates("ROOMAA", 0, []string{"c2", "a", "b"}, []string{"old", "watcher"})) {
		t.Fatal("arrival order changes lottery")
	}
	for _, u := range cands[:3] {
		if !slices.Contains(seated, u) {
			t.Fatal("spectator ahead of seated human")
		}
	}
	r.Check(t0, present(cands...), cands, 0)
	if log, next := r.Check(t0.Add(AwayLimit-time.Second), present(cands...), cands, 0); len(log) != 0 || !next.Equal(t0.Add(AwayLimit)) {
		t.Fatal("early grant")
	}
	back := roundRole(t, r)
	for _, v := range []*Roles{&r, &back} {
		v.Check(t0.Add(AwayLimit), present(cands...), Candidates("ROOMAA", v.Elections, seated, spec), 1)
	}
	if r.Temporary != back.Temporary || r.Elections != 1 || !r.Equal(&back) {
		t.Fatal("restart changes election")
	}
	first := r.Temporary
	r.Check(t0.Add(6*time.Minute), present("c", cands[0]), cands, 2)
	// A former spectator takes priority over never-temporary seated humans.
	r.remember("old")
	r.Check(t0.Add(7*time.Minute), present(cands...), cands, 0)
	r = roundRole(t, r)
	r.Check(t0.Add(12*time.Minute), present(cands...), cands, 3)
	if r.Temporary != "old" {
		t.Fatalf("former spectator not preferred to seats, first=%s: %+v", first, r)
	}
	// A connected promoted temporary keeps the team alive when its root leaves.
	_, e := r.Promote("old", "watcher", true, 3)
	mustRole(t, e)
	r.Check(t0.Add(13*time.Minute), present("watcher"), cands, 3)
	r.Check(t0.Add(18*time.Minute), present("watcher"), cands, 3)
	if r.Temporary != "old" || !r.Temp("watcher") {
		t.Fatal("connected deputy lost authority")
	}
	// Only after all temporaries have been away five minutes is a new team selected.
	r.Check(t0.Add(19*time.Minute), present("a"), cands, 3)
	log, _ := r.Check(t0.Add(24*time.Minute), present("a"), cands, 4)
	if r.Temporary != "a" || len(log) != 3 {
		t.Fatalf("replacement: %+v %v", r, log)
	}
}

func TestRefusedWrite(t *testing.T) {
	saved := New("c")
	cands := []string{"t", "d", "x"}
	here := present(cands...)
	saved.Check(t0, here, cands, 0)
	ahead := saved.Clone()
	ahead.Check(t0.Add(AwayLimit), here, cands, 0)
	if saved.Holds("t", &ahead, t0.Add(AwayLimit), here, cands) {
		t.Fatal("unsaved grant has powers")
	}
	retry := saved.Clone()
	log := retry.Carry(&ahead, 1)
	retry.Check(t0.Add(AwayLimit+Retry), here, cands, 1)
	if len(log) != 1 || !retry.Equal(&ahead) {
		t.Fatalf("retry: %+v %+v", retry, ahead)
	}
	saved = retry
	_, e := saved.Promote("t", "d", true, 2)
	mustRole(t, e)
	// Refused away-time writes do not overwrite a subsequent promotion.
	away := saved.Clone()
	away.Check(t0.Add(6*time.Minute), present("t", "x"), cands, 2)
	_, e = saved.Promote("d", "x", true, 3)
	mustRole(t, e)
	saved.Carry(&away, 3)
	if !saved.Temp("x") {
		t.Fatal("away-time retry erased a promotion")
	}
	gone := saved.Clone()
	gone.Check(t0.Add(7*time.Minute), present("c", "t", "d", "x"), cands, 4)
	for _, u := range []string{"t", "d", "x"} {
		if saved.Holds(u, &gone, t0.Add(7*time.Minute), present("c", "t", "d", "x"), cands) {
			t.Fatalf("%s retained powers after refused return", u)
		}
	}
	// Even if the returning real admin disconnects before the retry, its
	// new five-minute absence starts now; the stale interval cannot regrant.
	if saved.Holds("x", &gone, t0.Add(7*time.Minute), here, cands) {
		t.Fatal("stale absence regranted a temporary role")
	}
	retry = saved.Clone()
	log = retry.Carry(&gone, 4)
	if len(log) != 3 || retry.Temporary != "" || len(retry.Deputies) != 0 {
		t.Fatalf("team demotion: %+v %v", retry, log)
	}
}

func TestStepDown(t *testing.T) {
	r := New("")
	r.Check(t0, present("a", "b"), []string{"a", "b"}, 0)
	_, e := r.Promote("a", "b", true, 0)
	mustRole(t, e)
	_, e = r.StepDown("a", 1)
	mustRole(t, e)
	if r.Temporary != "b" || r.Temp("a") {
		t.Fatal("temporary root step-down")
	}
	_, e = r.StepDown("b", 2)
	mustRole(t, e)
	if log, _ := r.Check(t0, present("a", "b"), []string{"a", "b"}, 2); len(log) != 0 {
		t.Fatal("step-down immediately regranted")
	}
	r.Check(t0, present(), nil, 0)
	r.Check(t0, present("a"), []string{"a"}, 3)
	if !r.Temp("a") {
		t.Fatal("reconnection did not allow election")
	}
}

func TestBoundsAndValid(t *testing.T) {
	r := New("c")
	for i := 0; i < MaxAdmins-1; i++ {
		_, e := r.Promote("c", fmt.Sprintf("real-%d", i), true, 0)
		mustRole(t, e)
	}
	_, e := r.Promote("c", "overflow", true, 0)
	wantRole(t, e, ErrForbidden)
	r = New("c")
	r.Check(t0, present("t"), []string{"t"}, 0)
	r.Check(t0.Add(AwayLimit), present("t"), []string{"t"}, 0)
	for i := 0; i < MaxAdmins-1; i++ {
		_, e := r.Promote("t", fmt.Sprintf("temp-%d", i), true, 0)
		mustRole(t, e)
	}
	_, e = r.Promote("t", "overflow", true, 0)
	wantRole(t, e, ErrForbidden)
	for i := 0; i < MaxHistory*2; i++ {
		r.remember(fmt.Sprintf("old-%d", i))
	}
	if len(r.History) != MaxHistory {
		t.Fatal("history not capped")
	}
	mustRole(t, r.Valid())
	r = roundRole(t, r)
	for name, mut := range map[string]func(*Roles){
		"creator missing":   func(r *Roles) { r.Admins = r.Admins[1:] },
		"duplicate real":    func(r *Roles) { r.Admins = append(r.Admins, r.Admins[0]) },
		"duplicate temp":    func(r *Roles) { r.Deputies[0].UserID = r.Temporary },
		"real temp":         func(r *Roles) { r.Temporary = "c" },
		"duplicate history": func(r *Roles) { r.History[1] = r.History[0] },
		"oversize history":  func(r *Roles) { r.History = append(r.History, "extra") },
		"away stranger":     func(r *Roles) { r.Away = append(r.Away, Away{UserID: "stranger", Since: t0}) },
	} {
		v := r.Clone()
		mut(&v)
		if v.Valid() == nil {
			t.Errorf("%s accepted", name)
		}
	}
}

func TestUpgradePermanentCreatorAtCapacity(t *testing.T) {
	old := New("creator")
	old.Admins = nil
	for i := range MaxAdmins {
		old.Admins = append(old.Admins, Admin{UserID: fmt.Sprintf("real-%d", i), PromotedBy: &old.Creator})
	}
	old.Temporary = "t"
	old.Upgrade()
	if len(old.Admins) != MaxAuthorities || !old.Real("creator") || old.Temporary != "" {
		t.Fatalf("upgrade removed real grants: %+v", old)
	}
	mustRole(t, old.Valid())
	old.Check(t0, present("t"), []string{"t"}, 0)
	if log, _ := old.Check(t0.Add(AwayLimit), present("t"), []string{"t"}, 0); len(log) != 0 {
		t.Fatal("automatic grant exceeded authority cap")
	}
	mustRole(t, old.Valid())
	_, err := old.Demote("creator", "real-0", 1)
	mustRole(t, err)
	old.Check(t0.Add(AwayLimit), present("t"), []string{"t"}, 1)
	if old.Temporary != "t" {
		t.Fatal("no election after freeing the reserved slot")
	}
	mustRole(t, old.Valid())
}
