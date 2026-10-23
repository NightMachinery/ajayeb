package catalog

import (
	"fmt"
	"slices"
)

type registry struct {
	designs [NumDesigns + 1]Design
	boards  [NumBoards + 1]Board
	twins   [NumDesigns + 1]DesignID
}

var reg = &registry{}

// Twin maps a design to the other Age's same-name design, or zero. The data
// slice will populate it; production readers never mutate the catalog.
var Twin [NumDesigns + 1]DesignID

// ValidateCatalog accepts a small synthetic subset or the later full catalog.
// Complete edition counts and registration order are pinned by the data slice.
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
	old, oldTwin := reg, Twin
	reg, Twin = next, next.twins
	return func() { reg, Twin = old, oldTwin }
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
