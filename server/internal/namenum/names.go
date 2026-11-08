// Package namenum allocates room-lifetime display-name numbers. It belongs
// outside the actors so lobbies and games can carry the same ledger at Start.
package namenum

import (
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

// Counter remembers every name ever used, including names no current user
// has. Next is the next unused ordinal (the unnumbered name uses ordinal 1).
type Counter struct {
	Key  string `json:"key"`
	Next int    `json:"next"`
}

// Ledger retains users after departure and counters after renaming. Never
// prune either: that would reuse a suffix or renumber a returning user.
// The containing record's byte cap bounds growth; a failed write must discard
// its candidate ledger together with the rest of the transaction.
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
	i := slices.IndexFunc(l.Users, func(a Assignment) bool { return a.UserID == user })
	if i >= 0 && l.Users[i].Name == name {
		return false
	}
	key := Key(name)
	if i >= 0 && Key(l.Users[i].Name) == key {
		l.Users[i].Name = name
		return true
	}
	j := slices.IndexFunc(l.Names, func(c Counter) bool { return c.Key == key })
	number := 0
	if j < 0 {
		l.Names = append(l.Names, Counter{Key: key, Next: 2})
	} else {
		number = l.Names[j].Next
		l.Names[j].Next++
	}
	a := Assignment{UserID: user, Name: name, Number: number}
	if i < 0 {
		l.Users = append(l.Users, a)
	} else {
		l.Users[i] = a
	}
	return true
}

// Valid rejects malformed or ambiguous persisted assignments. Names are at
// most the hello protocol's 64 bytes; lowercasing can expand their keys.
func (l Ledger) Valid() error {
	const maxOrdinal = 1<<53 - 1
	names := map[string]int{}
	for _, c := range l.Names {
		if c.Key == "" || len(c.Key) > 3*64 || !utf8.ValidString(c.Key) || Key(c.Key) != c.Key || c.Next < 2 || c.Next > maxOrdinal || names[c.Key] != 0 {
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
		key := Key(a.Name)
		k := used{key, a.Number}
		if a.UserID == "" || len(a.UserID) > 64 || !utf8.ValidString(a.UserID) || users[a.UserID] || key == "" || len(a.Name) > 64 || !utf8.ValidString(a.Name) || a.Number < 0 || a.Number == 1 || a.Number >= names[key] || numbers[k] {
			return errors.New("invalid name assignment")
		}
		users[a.UserID], numbers[k] = true, true
	}
	return nil
}
