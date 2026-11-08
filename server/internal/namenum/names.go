// Package namenum allocates room-lifetime display-name numbers. It belongs
// outside the actors so lobbies and games can carry the same ledger at Start.
package namenum

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"slices"
	"strings"
	"unicode/utf8"
)

// Assignment is a user's current display name and suffix. Number 0 is the
// first, unnumbered use of a name; later uses start at 2.
type Assignment struct {
	UserID string `json:"userId"`
	Name   string `json:"name"`
	Number int    `json:"number"`
}

// Counter identifies a normalized name by SHA-256, with bounded storage even
// for JSON-escaped names. Next is the next unused ordinal (the first uses 1).
type Counter struct {
	Key  string `json:"key"`
	Next int    `json:"next"`
}

// Ledger stores retained authority holders and bounded name history. Actors keep
// spectator assignments separately, only while connected.
type Ledger struct {
	Users []Assignment `json:"users"`
	Names []Counter    `json:"names"`
}

// Key matches Duel's comparison, ignoring case and outer whitespace.
func Key(name string) string { return strings.ToLower(strings.TrimSpace(name)) }

// Clone returns an independent ledger with JSON-array slices.
func (l Ledger) Clone() Ledger {
	return Ledger{Users: append([]Assignment{}, l.Users...), Names: append([]Counter{}, l.Names...)}
}

// Get returns the user's current assignment, or its zero value.
func (l Ledger) Get(user string) Assignment {
	for _, a := range l.Users {
		if a.UserID == user {
			return a
		}
	}
	return Assignment{}
}

// Assign records a new name, changing only this user's assignment. A
// cosmetic change of case or outer spaces keeps the existing number.
func (l *Ledger) Assign(user, name string) bool {
	return l.AssignBounded(user, name, int(^uint(0)>>1))
}

// AssignBounded freezes an existing user's rename when history is full. New
// authority holders can always enter: Import evicts an unused historical key.
func (l *Ledger) AssignBounded(user, name string, capacity int, present ...Assignment) bool {
	i := slices.IndexFunc(l.Users, func(a Assignment) bool { return a.UserID == user })
	if i >= 0 && l.Users[i].Name == name {
		return false
	}
	key := counterKey(name)
	if i >= 0 && counterKey(l.Users[i].Name) == key {
		l.Users[i].Name = name
		return true
	}
	if i >= 0 && len(l.Names) >= capacity {
		return false
	}
	a := l.Preview(user, name, present)
	l.Import(a, capacity)
	return true
}

// Valid rejects malformed or ambiguous persisted assignments.
func (l Ledger) Valid() error {
	names := map[string]int{}
	for _, c := range l.Names {
		if !validCounterKey(c.Key) || c.Next < 2 || c.Next > maxOrdinal || names[c.Key] != 0 {
			return errors.New("invalid name counter")
		}
		names[c.Key] = c.Next
	}
	users := map[string]bool{}
	type used struct {
		key    string
		number int
	}
	numbers := map[used]bool{}
	for _, a := range l.Users {
		key := counterKey(a.Name)
		k := used{key, a.Number}
		if a.UserID == "" || len(a.UserID) > 64 || !utf8.ValidString(a.UserID) || users[a.UserID] || Key(a.Name) == "" || len(a.Name) > 64 || !utf8.ValidString(a.Name) || a.Number < 0 || a.Number == 1 || a.Number >= names[key] || numbers[k] {
			return errors.New("invalid name assignment")
		}
		users[a.UserID], numbers[k] = true, true
	}
	return nil
}

const maxOrdinal = 1<<53 - 1

// Preview allocates without consuming persistent history, avoiding every
// retained and currently connected holder of the same name.
func (l Ledger) Preview(user, name string, present []Assignment) Assignment {
	key := counterKey(name)
	next := 1
	for _, c := range l.Names {
		if c.Key == key {
			next = c.Next
		}
	}
	for _, a := range append(append([]Assignment{}, l.Users...), present...) {
		if counterKey(a.Name) == key {
			n := a.Number
			if n == 0 {
				n = 1
			}
			if n >= next {
				next = n + 1
			}
		}
	}
	// A valid imported record can already have exhausted the JSON-safe ordinal
	// range. Reuse the lowest unoccupied ordinal instead of blocking admission.
	if next >= maxOrdinal-1 {
		next = 1
		all := append(append([]Assignment{}, l.Users...), present...)
		for {
			occupied := slices.ContainsFunc(all, func(a Assignment) bool {
				n := a.Number
				if n == 0 {
					n = 1
				}
				return counterKey(a.Name) == key && n == next
			})
			if !occupied {
				break
			}
			next++
		}
	}
	if next == 1 {
		next = 0
	}
	return Assignment{UserID: user, Name: name, Number: next}
}

// Import preserves a live or rematch assignment. At capacity it forgets the
// oldest historical key with no retained holder, never a current holder's key.
// Capacity must exceed the containing actor's bounded authority population.
func (l *Ledger) Import(a Assignment, capacity int) {
	key := counterKey(a.Name)
	i := slices.IndexFunc(l.Users, func(x Assignment) bool { return x.UserID == a.UserID })
	if i < 0 {
		l.Users = append(l.Users, a)
	} else {
		l.Users[i] = a
	}
	j := slices.IndexFunc(l.Names, func(c Counter) bool { return c.Key == key })
	if j < 0 {
		if len(l.Names) >= capacity {
			j = slices.IndexFunc(l.Names, func(c Counter) bool {
				return !slices.ContainsFunc(l.Users, func(x Assignment) bool { return counterKey(x.Name) == c.Key })
			})
			l.Names = slices.Delete(l.Names, j, j+1)
		}
		l.Names = append(l.Names, Counter{Key: key, Next: 2})
		j = len(l.Names) - 1
	}
	n := a.Number
	if n == 0 {
		n = 1
	}
	if l.Names[j].Next <= n {
		l.Names[j].Next = n + 1
	}
}

// Retain removes assignments whose users have left the bounded authority set.
func (l *Ledger) Retain(keep func(string) bool) {
	l.Users = slices.DeleteFunc(l.Users, func(a Assignment) bool { return !keep(a.UserID) })
}

func counterKey(name string) string {
	sum := sha256.Sum256([]byte(Key(name)))
	return hex.EncodeToString(sum[:])
}

func validCounterKey(key string) bool {
	b, err := hex.DecodeString(key)
	return err == nil && len(b) == sha256.Size && strings.ToLower(key) == key
}
