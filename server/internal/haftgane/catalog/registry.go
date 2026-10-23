package catalog

import (
	"fmt"
	"slices"
)

type registry struct {
	designs [NumDesigns + 1]Design
	boards  [NumBoards + 1]Board
	twins   [NumDesigns + 1]DesignID
	tables  Tables
}

// The base game's catalog, registered from package-level initializers (not
// init) so that every reader sees it. A data error panics at initialization,
// which fails every test of every package that imports the catalog.
var reg = mustRegistry(BaseDesigns(), boardDefs)

// Twin maps a design to the other Age's same-name design, or zero. Production
// readers never mutate the catalog.
var Twin = reg.twins

// T holds the lookup tables compiled from the catalog (compiled.go). Never
// write to it.
var T = reg.tables

// BaseDesigns returns the base game's 80 designs in id order, with their Age
// set from the list that holds them and their name ids assigned: names are
// numbered from 1 in order of their first design, so a twin shares the id of
// its Age I design.
func BaseDesigns() []Design {
	var all []Design
	for _, list := range []struct {
		age     uint8
		designs []Design
	}{{1, ageOneDesigns}, {2, ageTwoDesigns}, {3, ageThreeDesigns}, {3, guildDesigns}} {
		for _, d := range list.designs {
			d.Age = list.age
			all = append(all, d)
		}
	}
	next := NameID(1)
	for i := range all {
		for j := range i {
			if all[j].Name == all[i].Name {
				all[i].NameID = all[j].NameID
				break
			}
		}
		if all[i].NameID == 0 {
			all[i].NameID = next
			next++
		}
	}
	return all
}

// BaseBoards returns the seven boards in id order.
func BaseBoards() []Board { return slices.Clone(boardDefs) }

func mustRegistry(designs []Design, boards []Board) *registry {
	r, err := makeRegistry(designs, boards)
	if err == nil {
		err = r.checkComplete()
	}
	if err != nil {
		panic("catalog: " + err.Error())
	}
	return r
}

// checkComplete asserts what the engine relies on beyond ValidateCatalog:
// every design and board id is present. The edition's counts, order and
// values are pinned by the tests.
func (r *registry) checkComplete() error {
	for id := DesignID(1); id <= NumDesigns; id++ {
		if r.designs[id].ID != id {
			return fmt.Errorf("design %d missing", id)
		}
	}
	for id := BoardID(1); id <= NumBoards; id++ {
		if r.boards[id].ID != id {
			return fmt.Errorf("board %d missing", id)
		}
	}
	return nil
}

// ValidateCatalog accepts a small synthetic subset or the full catalog. The
// complete edition's counts and registration order are pinned by the tests.
func ValidateCatalog(designs []Design, boards []Board) error {
	_, err := makeRegistry(designs, boards)
	return err
}
func makeRegistry(designs []Design, boards []Board) (*registry, error) {
	r := new(registry)
	for _, d := range designs {
		if err := d.Validate(); err != nil {
			return nil, err
		}
		if r.designs[d.ID].ID != 0 {
			return nil, fmt.Errorf("duplicate design id %d", d.ID)
		}
		for _, other := range r.designs {
			if other.ID == 0 {
				continue
			}
			if other.Slug == d.Slug {
				return nil, fmt.Errorf("duplicate design slug %s", d.Slug)
			}
			if (other.NameID == d.NameID) != (other.Name == d.Name) {
				return nil, fmt.Errorf("inconsistent name identity %s", d.Name)
			}
			if other.NameID == d.NameID {
				if r.twins[other.ID] != 0 || other.Age == d.Age || other.Age > 2 || d.Age > 2 || !sameTwin(other, d) {
					return nil, fmt.Errorf("invalid twins %s", d.Name)
				}
				r.twins[other.ID], r.twins[d.ID] = d.ID, other.ID
			}
		}
		d.Effects = slices.Clone(d.Effects)
		r.designs[d.ID] = d
	}
	for _, b := range boards {
		if err := b.Validate(); err != nil {
			return nil, err
		}
		if r.boards[b.ID].ID != 0 {
			return nil, fmt.Errorf("duplicate board id %d", b.ID)
		}
		for side := range b.Sides {
			for stage := range b.Sides[side].Stages {
				b.Sides[side].Stages[stage].Effects = slices.Clone(b.Sides[side].Stages[stage].Effects)
			}
		}
		r.boards[b.ID] = b
	}
	if err := r.checkChains(); err != nil {
		return nil, err
	}
	r.tables = compile(r)
	return r, nil
}
func sameTwin(a, b Design) bool {
	return a.Name == b.Name && a.NameID == b.NameID && a.Color == b.Color && a.Cost == b.Cost && a.ChainIn == b.ChainIn && a.ChainOut == b.ChainOut && slices.Equal(a.Effects, b.Effects)
}
func (r *registry) checkChains() error {
	var target [NumChains + 1]DesignID
	var providers [NumChains + 1]uint8
	for _, d := range r.designs {
		if d.ID == 0 {
			continue
		}
		if d.ChainIn != 0 {
			if target[d.ChainIn] != 0 {
				return fmt.Errorf("chain %d frees multiple designs", d.ChainIn)
			}
			target[d.ChainIn] = d.ID
		}
		for _, chain := range d.ChainOut {
			if chain != 0 {
				providers[chain]++
			}
		}
	}
	doubles := 0
	for chain := Chain(1); chain <= NumChains; chain++ {
		if target[chain] == 0 && providers[chain] == 0 {
			continue
		}
		if target[chain] == 0 || providers[chain] == 0 || providers[chain] > 2 {
			return fmt.Errorf("chain %d missing endpoint or too many providers", chain)
		}
		if providers[chain] == 2 {
			doubles++
		}
		for _, d := range r.designs {
			for _, out := range d.ChainOut {
				if out == chain && d.Age >= r.designs[target[chain]].Age {
					return fmt.Errorf("chain %d provider is not earlier", chain)
				}
			}
		}
	}
	if doubles > 1 {
		return fmt.Errorf("more than one chain has two providers")
	}
	return nil
}

// ReplaceForTest installs a validated synthetic catalog and returns its restore
// closure. Never use it from production or parallel tests, like Duel's helper.
func ReplaceForTest(designs []Design, boards []Board) func() {
	next, err := makeRegistry(designs, boards)
	if err != nil {
		panic(err)
	}
	old, oldTwin, oldT := reg, Twin, T
	reg, Twin, T = next, next.twins, next.tables
	return func() { reg, Twin, T = old, oldTwin, oldT }
}

// DesignByID returns read-only definition data. Effects must not be mutated.
func DesignByID(id DesignID) (Design, bool) {
	if !id.Valid() || reg.designs[id].ID == 0 {
		return Design{}, false
	}
	return reg.designs[id], true
}
func DesignBySlug(slug string) (DesignID, bool) {
	for _, d := range reg.designs {
		if d.ID != 0 && d.Slug == slug {
			return d.ID, true
		}
	}
	return 0, false
}
func BoardByID(id BoardID) (Board, bool) {
	if !id.Valid() || reg.boards[id].ID == 0 {
		return Board{}, false
	}
	return reg.boards[id], true
}
