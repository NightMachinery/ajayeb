package tabletest

import (
	"errors"
	"math/rand/v2"
	"strconv"
)

// ScriptOptions shape each seat's submissions at a point.
type ScriptOptions struct {
	// Drafts is the percent chance that a seat first picks another key
	// unconfirmed, then its final key confirmed (a changed pick).
	Drafts int
	// Resends is the percent chance, per submission, that the client sends
	// it again later with the same seq: a duplicate while the table
	// remembers the client's seq, stale or decision-closed once it forgot.
	Resends int
	// Race makes every seat confirm first and then change its pick or
	// withdraw its confirmation and confirm again, so whether the change
	// lands depends on arrival order. Without it a seat confirms only in
	// its last submission, and every arrival order must give one batch.
	Race bool
	// Tabs sends a seat's last submission from a second tab of its user
	// (TabClient) and the others from its client. The seq memory is per
	// user, so only a user's own tabs push each other's seqs out of a small
	// one; a resend of an earlier submission then meets the pick's floor (a
	// duplicate, or stale after a race's two submissions from the client)
	// or a resolved point.
	Tabs bool
}

// Op is one submission as scheduled.
type Op struct {
	Client ClientID   `json:"client"`
	Seat   int        `json:"seat"`
	Sub    Submission `json:"sub"`
	Resend bool       `json:"resend,omitempty"`
}

// Seqs hands out each client's increasing seqs across a whole game.
type Seqs map[ClientID]uint64

func (s Seqs) next(c ClientID) uint64 {
	s[c]++
	return s[c]
}

// Script is the submissions of one required seat at one point, in the
// order its client sends them, and the key it ends on when nothing races.
type Script struct {
	Seat  int
	Ops   []Op
	Final string
}

// Scripts builds every required seat's submissions from the common
// snapshot (snaps[i] is clients[i]'s). A seat's keys come from a stream
// seeded by the pick seed, the logical point id and the seat, never by
// arrival order, so schedules that differ only in their arrival seed submit
// the same keys.
func Scripts(snaps []Snapshot, clients []ClientID, pickSeed uint64, opts ScriptOptions, seqs Seqs) ([]Script, error) {
	if len(snaps) != len(clients) {
		return nil, errors.New("tabletest: one snapshot per client")
	}
	var out []Script
	for i, sn := range snaps {
		if !contains(sn.Required, sn.Seat) {
			continue
		}
		if len(sn.Legal) == 0 {
			return nil, errors.New("tabletest: seat " + strconv.Itoa(sn.Seat) + " is required but offered nothing")
		}
		c := clients[i]
		rng := rand.New(rand.NewPCG(mix(pickSeed, hashString(sn.Point)), uint64(sn.Seat)))
		idx := rng.IntN(len(sn.Legal))
		final := sn.Legal[idx]
		other := func() string {
			if len(sn.Legal) == 1 {
				return final
			}
			return sn.Legal[(idx+1+rng.IntN(len(sn.Legal)-1))%len(sn.Legal)]
		}
		op := func(key string, confirm bool) Op {
			return Op{Client: c, Seat: sn.Seat, Sub: Submission{Seq: seqs.next(c), Ref: sn.Ref, Key: key, Confirm: confirm}}
		}
		var ops []Op
		switch {
		case opts.Race:
			first := other()
			ops = append(ops, op(first, true))
			if rng.IntN(2) == 0 {
				ops = append(ops, op(final, false), op(final, true)) // change
			} else {
				ops = append(ops, op(first, false), op(first, true)) // withdraw, confirm again
				final = first
			}
		case len(sn.Legal) > 1 && rng.IntN(100) < opts.Drafts:
			ops = append(ops, op(other(), false), op(final, true))
		default:
			ops = append(ops, op(final, true))
		}
		if opts.Tabs {
			// The last op took the client's last seq; give it back.
			tab, last := TabClient(sn.Seat), &ops[len(ops)-1]
			seqs[c]--
			last.Client, last.Sub.Seq = tab, seqs.next(tab)
		}
		for j := 0; j < len(ops); j++ {
			if ops[j].Resend || rng.IntN(100) >= opts.Resends {
				continue
			}
			again := ops[j]
			again.Resend = true
			at := j + 1 + rng.IntN(len(ops)-j)
			ops = append(ops[:at], append([]Op{again}, ops[at:]...)...)
		}
		out = append(out, Script{Seat: sn.Seat, Ops: ops, Final: final})
	}
	return out, nil
}

// Interleave merges the scripts in a seeded random order that keeps each
// client's own order; every merge is equally likely.
func Interleave(scripts []Script, seed uint64) []Op {
	rng := rand.New(rand.NewPCG(seed, 0x6a09e667f3bcc909))
	next := make([]int, len(scripts))
	left := 0
	for _, s := range scripts {
		left += len(s.Ops)
	}
	out := make([]Op, 0, left)
	for left > 0 {
		r := rng.IntN(left)
		for i, s := range scripts {
			rest := len(s.Ops) - next[i]
			if r < rest {
				out = append(out, s.Ops[next[i]])
				next[i]++
				break
			}
			r -= rest
		}
		left--
	}
	return out
}

// Interleavings calls yield with every merge of the scripts that keeps each
// client's own order, until yield returns false.
func Interleavings(scripts []Script, yield func([]Op) bool) {
	next := make([]int, len(scripts))
	total := 0
	for _, s := range scripts {
		total += len(s.Ops)
	}
	cur := make([]Op, 0, total)
	var walk func() bool
	walk = func() bool {
		if len(cur) == total {
			return yield(append([]Op(nil), cur...))
		}
		for i, s := range scripts {
			if next[i] == len(s.Ops) {
				continue
			}
			cur = append(cur, s.Ops[next[i]])
			next[i]++
			ok := walk()
			next[i]--
			cur = cur[:len(cur)-1]
			if !ok {
				return false
			}
		}
		return true
	}
	walk()
}

// CountInterleavings returns how many merges Interleavings visits, or
// limit+1 when there are more than limit.
func CountInterleavings(scripts []Script, limit uint64) uint64 {
	// The multinomial coefficient, built up one script at a time.
	n, count := uint64(0), uint64(1)
	for _, s := range scripts {
		for k := uint64(1); k <= uint64(len(s.Ops)); k++ {
			n++
			count = count * n / k
			if count > limit {
				return limit + 1
			}
		}
	}
	return count
}

// mix is the splitmix64 finalizer of a and b.
func mix(a, b uint64) uint64 {
	z := a ^ (b * 0x9e3779b97f4a7c15)
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb
	return z ^ (z >> 31)
}

// hashString is FNV-1a.
func hashString(s string) uint64 {
	h := uint64(14695981039346656037)
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= 1099511628211
	}
	return h
}

func contains(xs []int, x int) bool {
	for _, y := range xs {
		if y == x {
			return true
		}
	}
	return false
}
