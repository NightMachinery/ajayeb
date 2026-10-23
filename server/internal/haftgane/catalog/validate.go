package catalog

import (
	"fmt"
	"strings"
)

// Validate checks the tagged union, not when a player may use the effect.
func (e Effect) Validate() error {
	used := Effect{Kind: e.Kind}
	positive := false
	switch e.Kind {
	case EffProduce:
		used.N, used.Res, used.ForSale = e.N, e.Res, e.ForSale
		positive = true
		if e.Res >= NumResources {
			return fmt.Errorf("produce: invalid resource")
		}
	case EffProduceChoice:
		used.Mask, used.ForSale = e.Mask, e.ForSale
		if !e.Mask.Valid() || e.Mask.Count() < 2 {
			return fmt.Errorf("produce-choice: need at least two resources")
		}
	case EffTradeDiscount:
		used.N, used.Mask, used.Scope = e.N, e.Mask, e.Scope
		if e.N != 1 || (e.Mask != Raw && e.Mask != Manufactured) || e.Scope == 0 || e.Scope&^(Left|Right) != 0 {
			return fmt.Errorf("trade-discount: price 1, one family and neighbour scope required")
		}
	case EffShields, EffCoins, EffPoints, EffPointsIfWonderComplete:
		used.N = e.N
		positive = true
	case EffScience:
		used.Symbol = e.Symbol
		if e.Symbol <= NoSymbol || e.Symbol >= NumSymbols {
			return fmt.Errorf("science: invalid symbol")
		}
	case EffScienceChoice, EffPlayLastCard, EffBuildFromDiscard:
	case EffCoinsPerCard, EffPointsPerCard:
		used.N, used.Colors, used.Scope = e.N, e.Colors, e.Scope
		positive = true
		if !e.Colors.Valid() || !e.Scope.Valid() {
			return fmt.Errorf("%s: colors and base-game scope required", e.Kind)
		}
	case EffCoinsPerStage, EffPointsPerStage:
		used.N, used.Scope = e.N, e.Scope
		positive = true
		if !e.Scope.Valid() {
			return fmt.Errorf("%s: base-game scope required", e.Kind)
		}
	case EffFreeBuild:
		used.When = e.When
		if e.When <= NoFreeCondition || e.When >= NumFreeConditions {
			return fmt.Errorf("free-build: invalid condition")
		}
	default:
		return fmt.Errorf("unknown effect kind %d", e.Kind)
	}
	if positive && e.N == 0 {
		return fmt.Errorf("%s: N must be positive", e.Kind)
	}
	if used != e {
		return fmt.Errorf("%s: unused effect fields must be zero", e.Kind)
	}
	return nil
}

func (c Cost) Validate() error {
	if c.Units() > MaxCostUnits {
		return fmt.Errorf("cost exceeds %d resource units", MaxCostUnits)
	}
	return nil
}
func validSlug(s string) bool {
	if len(s) == 0 || len(s) > MaxSlugBytes || s[0] < 'a' || s[0] > 'z' || strings.HasPrefix(s, "the-") || s[0] == '-' || s[len(s)-1] == '-' || strings.Contains(s, "--") {
		return false
	}
	for _, ch := range []byte(s) {
		if (ch < 'a' || ch > 'z') && (ch < '0' || ch > '9') && ch != '-' {
			return false
		}
	}
	return true
}
func nonempty(s string) bool { return s != "" && strings.TrimSpace(s) == s }

func (d Design) Validate() error {
	if !d.ID.Valid() || !d.NameID.Valid() || !validSlug(d.Slug) || !nonempty(d.Name) {
		return fmt.Errorf("design: invalid id, name or slug")
	}
	if d.Age < 1 || d.Age > 3 || d.Color <= NoColor || d.Color >= NumColors {
		return fmt.Errorf("%s: invalid age or color", d.Slug)
	}
	if err := d.Cost.Validate(); err != nil {
		return fmt.Errorf("%s: %w", d.Slug, err)
	}
	if d.Cost.Coins != 0 && d.Cost.Units() != 0 {
		return fmt.Errorf("%s: card cost mixes coins and resources", d.Slug)
	}
	if d.ChainIn > NumChains || d.ChainOut[0] > NumChains || d.ChainOut[1] > NumChains || d.ChainOut[0] == 0 && d.ChainOut[1] != 0 || d.ChainOut[1] != 0 && d.ChainOut[0] == d.ChainOut[1] {
		return fmt.Errorf("%s: invalid chain symbols", d.Slug)
	}
	if d.Color == Purple {
		if d.Age != 3 || d.Copies != [3]uint8{} {
			return fmt.Errorf("%s: guild needs Age III and no copy markers", d.Slug)
		}
	} else {
		if d.Copies[0] == 0 {
			return fmt.Errorf("%s: missing copy markers", d.Slug)
		}
		previous := uint8(0)
		padding := false
		for _, marker := range d.Copies {
			if marker == 0 {
				padding = true
				continue
			}
			if padding || marker < 3 || marker > 7 || marker <= previous {
				return fmt.Errorf("%s: copy markers must be ascending, 3 to 7, then zero-padded", d.Slug)
			}
			previous = marker
		}
	}
	return validateEffects(d.Effects, d.Color, false)
}

func (b Board) Validate() error {
	if !b.ID.Valid() || b.Slug != b.ID.String() || !nonempty(b.Name) || !nonempty(b.Wonder) || b.Start >= NumResources {
		return fmt.Errorf("board: invalid identity or starting resource")
	}
	for side, def := range b.Sides {
		if def.NumStages < 2 || def.NumStages > MaxStages || side == 0 && def.NumStages != 3 {
			return fmt.Errorf("%s: invalid stage count", b.Slug)
		}
		for i, stage := range def.Stages {
			if i >= int(def.NumStages) {
				if stage.Cost != (Cost{}) || len(stage.Effects) != 0 {
					return fmt.Errorf("%s: unused stage is nonempty", b.Slug)
				}
				continue
			}
			if err := stage.Cost.Validate(); err != nil {
				return fmt.Errorf("%s stage %d: %w", b.Slug, i+1, err)
			}
			if err := validateEffects(stage.Effects, NoColor, true); err != nil {
				return fmt.Errorf("%s stage %d: %w", b.Slug, i+1, err)
			}
		}
	}
	return nil
}

func validateEffects(effects []Effect, color Color, stage bool) error {
	if len(effects) == 0 {
		return fmt.Errorf("missing effects")
	}
	for _, e := range effects {
		if err := e.Validate(); err != nil {
			return err
		}
		switch e.Kind {
		case EffProduce:
			if stage {
				if e.ForSale {
					return fmt.Errorf("stage production is not for sale")
				}
				continue
			}
			if !e.ForSale || color == Brown && !Raw.Has(e.Res) || color == Grey && !Manufactured.Has(e.Res) || color != Brown && color != Grey {
				return fmt.Errorf("card production must be brown/raw or grey/manufactured and for sale")
			}
		case EffProduceChoice:
			switch {
			case color == Brown && !stage:
				if !e.ForSale || e.Mask&^Raw != 0 || e.Mask.Count() != 2 {
					return fmt.Errorf("brown choice must sell one of two raw resources")
				}
			case stage || color == Yellow:
				if e.ForSale || (e.Mask != Raw && e.Mask != Manufactured) {
					return fmt.Errorf("yellow/stage choice uses one family and is not for sale")
				}
			default:
				return fmt.Errorf("choice production needs brown, yellow or a stage")
			}
		case EffFreeBuild, EffPlayLastCard, EffBuildFromDiscard:
			if !stage {
				return fmt.Errorf("%s belongs to a wonder stage", e.Kind)
			}
		}
	}
	return nil
}
