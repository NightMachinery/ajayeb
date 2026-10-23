package catalog

// Regression pins for docs/haftgane/edition-differences.md: every value the
// first edition had differently, and every error seen in sources that claim
// the second edition. Each pin reads a field of the record tables (the
// notation of the golden CSVs) and names the value we hold and the wrong
// values a source might carry over. data-rulings.md records how each was
// settled.

import (
	"strconv"
	"strings"
	"testing"
)

// editionRows indexes the record tables: designs by "age/name", stages by
// "board/side/stage", boards by slug.
func editionRows(t *testing.T) map[string]map[string]string {
	t.Helper()
	tables, err := RecordTables()
	if err != nil {
		t.Fatal(err)
	}
	rows := map[string]map[string]string{}
	for _, tb := range tables {
		for _, r := range tb.Rows {
			cells := map[string]string{}
			for i, h := range tb.Header {
				cells[h] = r[i]
			}
			var key string
			switch tb.Name {
			case "cards":
				key = cells["age"] + "/" + cells["name"]
			case "wonders":
				key = cells["board"] + "/" + cells["side"] + "/" + cells["stage"]
			case "boards":
				key = cells["slug"]
			}
			rows[key] = cells
		}
	}
	return rows
}

type editionPin struct {
	item, field, want string
	wrong             []string
	why               string
}

var editionPins = []editionPin{
	// Age I.
	{"1/Well", "effects", "points:3", nil, "Well replaces Pawnshop"},
	{"1/Well", "chain_to", "Statue", []string{""}, "Well provides the chain to Statue; Pawnshop had none"},
	{"1/Well", "player_counts", "4;7", []string{"3;6"}, "Well and Theater swapped in one source"},
	{"1/Altar", "effects", "points:3", []string{"points:2"}, "Altar gives 3 points (first edition 2)"},
	{"1/Altar", "chain_to", "Pantheon", []string{"Temple"}, "Altar chains to Pantheon (first edition Temple)"},
	{"1/Theater", "effects", "points:3", []string{"points:2"}, "Theater gives 3 points (first edition 2)"},
	{"1/Theater", "chain_to", "Gardens", []string{"Statue"}, "Theater chains to Gardens (first edition Statue)"},
	{"1/Theater", "player_counts", "3;6", []string{"4;7"}, "Well and Theater swapped in one source"},
	{"1/Baths", "chain_to", "Aqueduct", []string{""}, "Baths still chains to Aqueduct"},
	{"1/Baths", "cost", "stone:1", []string{"free"}, "Baths listed as free"},
	{"1/Stockade", "cost", "wood:1", []string{"free"}, "Age I red cards listed as free"},
	{"1/Barracks", "cost", "ore:1", []string{"free", "stone:1"}, "Barracks listed as free or costing stone"},
	{"1/Guard Tower", "cost", "clay:1", []string{"free"}, "Age I red cards listed as free"},
	{"1/Clay Pit", "effects", "produce:clay/ore", []string{"produce:stone/clay"}, "Clay Pit offering clay or stone"},
	{"1/Mine", "effects", "produce:stone/ore", nil, "Mine named Cave in one source"},

	// Age II.
	{"2/Temple", "effects", "points:4", []string{"points:3"}, "Temple gives 4 points (first edition 3)"},
	{"2/Temple", "chain_from", "", []string{"Altar"}, "Temple has no chain (first edition from Altar)"},
	{"2/Temple", "chain_to", "", []string{"Pantheon"}, "Temple has no chain (first edition to Pantheon)"},
	{"2/Statue", "chain_from", "Well", []string{"Theater", "Pawnshop"}, "Statue is chained from Well"},
	{"2/Statue", "chain_to", "", []string{"Gardens"}, "Statue leads nowhere (first edition to Gardens)"},
	{"2/Bazar", "effects", "per:grey:self+neighbours:coins2", []string{"per:yellow:self+neighbours:coins2"}, "Bazar paying for yellow cards"},
	{"2/Stables", "player_counts", "3;5", []string{"3;6"}, "Stables at 3 and 6 in one source"},
	{"2/Quarry", "player_counts", "3;4", []string{"3;5"}, "Quarry at 3 and 5 in one source"},
	{"2/Brickyard", "player_counts", "3;4", []string{"3;5"}, "Brickyard at 3 and 5 in one source"},
	{"2/Library", "player_counts", "3;6", []string{"3;5"}, "Library at 3 and 5 in one source"},
	{"2/Forum", "chain_from", "East Trading Post;West Trading Post", []string{"East Trading Post"}, "Forum is chained from either trading post"},
	{"2/Walls", "chain_to", "Fortifications", []string{"", "Academy"}, "Walls frees Fortifications"},
	{"2/Training Ground", "chain_to", "Circus", []string{""}, "Training Ground frees Circus"},
	{"2/Stables", "chain_to", "", []string{"Arena"}, "ruling 4: misread chain diagram"},
	{"2/Archery Range", "chain_to", "", []string{"Siege Workshop"}, "ruling 4: misread chain diagram"},
	{"2/Courthouse", "chain_to", "", []string{"Senate"}, "ruling 4: misread chain diagram"},
	{"2/Dispensary", "chain_to", "Arena;Lodge", []string{"Lodge"}, "ruling 4: Dispensary provides Arena's symbol"},
	{"2/Laboratory", "chain_to", "Observatory;Siege Workshop", []string{"Observatory"}, "ruling 4: Laboratory provides Siege Workshop's symbol"},
	{"2/Library", "chain_to", "Senate;University", []string{"University"}, "ruling 4: Library provides Senate's symbol"},

	// Age III.
	{"3/Ludus", "colour", "yellow", nil, "Ludus is new"},
	{"3/Ludus", "player_counts", "5;7", nil, "Ludus is new, copies at 5 and 7"},
	{"3/Ludus", "cost", "stone:1 ore:1", []string{"stone:2 ore:1"}, "a second Ludus copy priced differently"},
	{"3/Ludus", "effects", "per:red:self:coins3;per:red:self:points1", nil, "Ludus counts red cards in your own city"},
	{"3/Castrum", "colour", "red", nil, "Castrum is new"},
	{"3/Castrum", "effects", "shields:3", nil, "Castrum is new, 3 shields"},
	{"3/Castrum", "player_counts", "4;7", nil, "Castrum is new, copies at 4 and 7"},
	{"3/Castrum", "cost", "wood:1 clay:2 papyrus:1", []string{"wood:1 clay:3 papyrus:1"}, "Castrum costing WCCCP"},
	{"3/Town Hall", "player_counts", "3;6", []string{"3;5;6"}, "Town Hall loses its 5-player copy"},
	{"3/Arena", "player_counts", "3;5", []string{"3;5;7"}, "Arena loses its 7-player copy"},
	{"3/Circus", "player_counts", "4;6", []string{"4;5;6"}, "Circus loses its 5-player copy"},
	{"3/Arsenal", "player_counts", "3;5", []string{"3;4;7"}, "Arsenal loses a copy and moves"},
	{"3/Fortifications", "cost", "clay:1 ore:3", []string{"stone:1 ore:3", "stone:3 clay:1"}, "Fortifications costs COOO (first edition SOOO)"},
	{"3/Pantheon", "chain_from", "Altar", []string{"Temple"}, "Pantheon is chained from Altar"},
	{"3/Gardens", "chain_from", "Theater", []string{"Statue", "Well"}, "Gardens is chained from Theater"},
	{"3/Town Hall", "cost", "stone:2 ore:1 glass:1", []string{"stone:3 glass:1"}, "a cost that did not change (wiki SSSG)"},
	{"3/Arena", "cost", "stone:2 ore:1", []string{"clay:2 ore:1"}, "a cost that did not change (wiki CCO)"},
	{"3/Circus", "cost", "stone:3 ore:1", []string{"clay:3 ore:1"}, "a cost that did not change (wiki CCCO)"},
	{"3/Haven", "cost", "wood:1 ore:1 cloth:1", []string{"clay:2"}, "Haven costing CC"},
	{"3/Haven", "chain_from", "Forum", []string{""}, "Haven is chained from Forum"},
	{"3/Fortifications", "chain_from", "Walls", []string{""}, "Fortifications is chained from Walls"},
	{"3/Circus", "chain_from", "Training Ground", []string{""}, "Circus is chained from Training Ground"},
	{"3/Academy", "chain_from", "School", []string{"Walls"}, "the wiki's chain page names Academy for Walls"},
	{"3/Siege Workshop", "cost", "wood:1 clay:3", []string{"stone:1 clay:3"}, "Siege Workshop costing CCCS"},
	{"3/Pantheon", "cost", "clay:2 ore:1 glass:1 papyrus:1 cloth:1", []string{"stone:1 clay:2 glass:1 papyrus:1 cloth:1"}, "Pantheon costing SCCGPL"},
	{"3/Gardens", "cost", "wood:1 clay:2", []string{"wood:2 clay:1"}, "Gardens costing WWC"},
	{"3/Academy", "player_counts", "3;7", []string{"3"}, "Academy with one copy"},
	{"3/University", "player_counts", "3;4", []string{"3;4;7"}, "University with three copies"},
	{"3/Arena", "chain_from", "Dispensary", []string{"Stables"}, "ruling 4: misread chain diagram"},
	{"3/Siege Workshop", "chain_from", "Laboratory", []string{"Archery Range"}, "ruling 4: misread chain diagram"},
	{"3/Senate", "chain_from", "Library", []string{"Courthouse"}, "ruling 4: misread chain diagram"},

	// Guilds.
	{"3/Decorators Guild", "effects", "other:points7-if-all-own-wonder-stages-built", nil, "Decorators Guild replaces Strategists Guild"},
	{"3/Decorators Guild", "cost", "stone:1 ore:2 cloth:1", []string{"stone:2 ore:2 glass:2"}, "Decorators Guild costing SSOOGG"},
	{"3/Builders Guild", "cost", "stone:3 clay:2 glass:1", []string{"stone:2 clay:2 glass:1", "stone:3 clay:1 glass:1"}, "Builders Guild costs SSSCCG (first edition SSCCG)"},
	{"3/Spies Guild", "cost", "clay:2 glass:1", []string{"clay:3 glass:1"}, "Spies Guild costs CCG (first edition CCCG)"},
	{"3/Craftsmens Guild", "cost", "stone:2 ore:2", []string{"stone:2 ore:1"}, "a cost that did not change (wiki SSO)"},
	{"3/Traders Guild", "cost", "glass:1 papyrus:1 cloth:1", []string{"clay:1 ore:2 glass:1 papyrus:1"}, "Traders Guild costing COOGP"},

	// Boards.
	{"alexandria", "starting_resource", "glass", nil, "Alexandria starts with glass"},
	{"alexandria/day/2", "cost", "ore:2", []string{"ore:3"}, "Alexandria day stage 2 costing OOO"},
	{"alexandria/day/3", "cost", "papyrus:1 cloth:1", []string{"glass:2"}, "Alexandria day stage 3 costs PL (first edition GG)"},
	{"alexandria/night/1", "cost", "clay:2", nil, "Alexandria night stage 1 unchanged"},
	{"alexandria/night/1", "effects", "produce:wood/stone/clay/ore", nil, "an Alexandria night side that differs in every stage"},
	{"alexandria/night/2", "cost", "ore:3", []string{"wood:2"}, "Alexandria night stage 2 costs OOO (first edition WW)"},
	{"alexandria/night/2", "effects", "produce:glass/papyrus/cloth", nil, "an Alexandria night side that differs in every stage"},
	{"alexandria/night/3", "cost", "wood:4", []string{"stone:3"}, "Alexandria night stage 3 costs WWWW (first edition SSS)"},
	{"alexandria/night/3", "effects", "points:7", nil, "an Alexandria night side that differs in every stage"},
	{"babylon", "starting_resource", "wood", []string{"clay"}, "Babylon starts with wood (first edition clay)"},
	{"babylon", "night_stages", "2", []string{"3"}, "Babylon night has two stages (first edition three; Q1)"},
	{"babylon/day/2", "cost", "ore:2 cloth:1", []string{"wood:3"}, "Babylon day stage 2 costs OOL (first edition WWW)"},
	{"babylon/day/3", "cost", "wood:4", []string{"clay:4"}, "Babylon day stage 3 costs WWWW (first edition CCCC)"},
	{"babylon/night/1", "cost", "stone:2", []string{"clay:1 cloth:1", "clay:2 glass:1"}, "Babylon night stage 1 (Q1): first-edition CL, or CCG in one source"},
	{"babylon/night/1", "effects", "other:play-last-card-of-age", []string{"points:3", "other:build-from-discard"}, "Babylon night stage 1 (Q1): first-edition points, or the discard ability in one source"},
	{"babylon/night/2", "cost", "clay:3 glass:1", []string{"wood:2 glass:1"}, "Babylon night stage 2 (Q1): first edition WWG"},
	{"babylon/night/2", "effects", "science:any", []string{"other:play-last-card-of-age"}, "Babylon night stage 2 (Q1)"},
	{"ephesos", "starting_resource", "papyrus", nil, "Ephesos starts with papyrus"},
	{"ephesos/day/1", "cost", "clay:2", []string{"stone:2"}, "Ephesos day stage 1 costs CC (first edition SS)"},
	{"ephesos/day/3", "cost", "ore:2 glass:1", []string{"papyrus:2"}, "Ephesos day stage 3 costs OOG (first edition PP)"},
	{"ephesos/night/3", "cost", "ore:2 cloth:1", []string{"glass:1 papyrus:1 cloth:1"}, "Ephesos night stage 3 costs OOL (first edition GPL)"},
	{"ephesos/night/3", "effects", "coins:4;points:5", []string{"coins:4;points:4"}, "Ephesos night stage 3 worth 4 points"},
	{"gizah", "starting_resource", "stone", []string{"clay"}, "Gizah starting with clay"},
	{"gizah", "night_stages", "4", nil, "Gizah night has four stages"},
	{"gizah/day/1", "cost", "wood:2", []string{"stone:2"}, "Gizah day stage 1 costs WW (first edition SS)"},
	{"gizah/day/2", "cost", "clay:2 cloth:1", []string{"wood:3"}, "Gizah day stage 2 costs CCL (first edition WWW)"},
	{"gizah/night/1", "cost", "wood:2", nil, "Gizah night unchanged"},
	{"gizah/night/2", "cost", "stone:3", nil, "Gizah night unchanged"},
	{"gizah/night/3", "cost", "clay:3", nil, "Gizah night stage 3 (Q2, confirmed)"},
	{"gizah/night/4", "cost", "stone:4 papyrus:1", nil, "Gizah night stage 4 (Q3, confirmed)"},
	{"gizah/night/4", "effects", "points:7", nil, "Gizah night unchanged"},
	{"halikarnassos", "starting_resource", "cloth", nil, "Halikarnassos starts with cloth on both sides (Q5)"},
	{"halikarnassos/day/1", "cost", "ore:2", []string{"clay:2", "stone:2"}, "Halikarnassos day stage 1 costs OO (first edition CC; SS in one source)"},
	{"halikarnassos/day/2", "cost", "glass:1 papyrus:1", []string{"ore:3"}, "Halikarnassos day stage 2 costs GP (first edition OOO)"},
	{"halikarnassos/day/3", "cost", "stone:3", []string{"cloth:2"}, "Halikarnassos day stage 3 costs SSS (first edition LL)"},
	{"halikarnassos/night/1", "cost", "clay:2", []string{"ore:2"}, "Halikarnassos night stage 1 costs CC (first edition OO)"},
	{"halikarnassos/night/1", "effects", "other:build-from-discard;points:2", nil, "every Halikarnassos night stage has the discard ability"},
	{"halikarnassos/night/2", "cost", "glass:1 papyrus:1", []string{"clay:3"}, "Halikarnassos night stage 2 costs GP (first edition CCC)"},
	{"halikarnassos/night/2", "effects", "other:build-from-discard;points:1", nil, "every Halikarnassos night stage has the discard ability"},
	{"halikarnassos/night/3", "cost", "wood:3", []string{"glass:1 papyrus:1 cloth:1"}, "Halikarnassos night stage 3 costs WWW (first edition GPL; Q4)"},
	{"halikarnassos/night/3", "effects", "other:build-from-discard", nil, "every Halikarnassos night stage has the discard ability"},
	{"olympia", "starting_resource", "clay", []string{"wood"}, "Olympia starts with clay (first edition wood)"},
	{"olympia/day/1", "cost", "stone:2", []string{"wood:2"}, "Olympia day stage 1 costs SS (first edition WW)"},
	{"olympia/day/2", "cost", "wood:2", []string{"stone:2"}, "Olympia day stage 2 costs WW (first edition SS)"},
	{"olympia/day/2", "effects", "other:first-card-of-each-colour-free", nil, "the first building of each colour is free (first edition: one free building per Age)"},
	{"olympia/day/3", "cost", "clay:3", []string{"ore:2"}, "Olympia day stage 3 costs CCC (first edition OO)"},
	{"olympia/day/3", "effects", "points:7", []string{""}, "Olympia day stage 3 giving no points"},
	{"olympia/night/1", "cost", "ore:2", []string{"wood:2"}, "Olympia night stage 1 is new"},
	{"olympia/night/1", "effects", "other:first-card-of-each-age-free;points:2", []string{"trade:raw:both"}, "Olympia night stage 1 is new (first edition: raw resources at 1 coin)"},
	{"olympia/night/2", "cost", "clay:3", []string{"stone:2"}, "Olympia night stage 2 is new"},
	{"olympia/night/2", "effects", "other:last-card-of-each-age-free;points:3", []string{"points:5"}, "Olympia night stage 2 is new"},
	{"olympia/night/3", "cost", "glass:1 papyrus:1 cloth:1", []string{"ore:2 cloth:1"}, "Olympia night stage 3 is new (first edition: guild copying)"},
	{"olympia/night/3", "effects", "points:5", nil, "Olympia night stage 3 is new (first edition: guild copying)"},
	{"rhodos", "starting_resource", "ore", nil, "Rhodos unchanged"},
	{"rhodos/day/1", "effects", "points:3", nil, "Rhodos unchanged"},
	{"rhodos/day/2", "cost", "clay:3", nil, "Rhodos unchanged"},
	{"rhodos/day/2", "effects", "shields:2", []string{"shields:5"}, "Rhodos day stage 2 giving 5 shields"},
	{"rhodos/day/3", "cost", "ore:4", nil, "Rhodos unchanged"},
	{"rhodos/night/1", "cost", "stone:3", []string{"stone:2"}, "Rhodos night stage 1 read as SS from a covered photo (Q6)"},
	{"rhodos/night/1", "effects", "coins:3;points:3;shields:1", nil, "Rhodos unchanged"},
	{"rhodos/night/2", "cost", "ore:4", nil, "Rhodos unchanged"},
	{"rhodos/night/2", "effects", "coins:4;points:4;shields:1", nil, "Rhodos unchanged"},
}

func TestEditionDifferences(t *testing.T) {
	rows := editionRows(t)
	for _, p := range editionPins {
		row, ok := rows[p.item]
		if !ok {
			t.Errorf("%s: no such item (%s)", p.item, p.why)
			continue
		}
		got, ok := row[p.field]
		if !ok {
			t.Fatalf("%s: no field %s", p.item, p.field)
		}
		if got != p.want {
			t.Errorf("%s %s = %q, want %q (%s)", p.item, p.field, got, p.want, p.why)
		}
		for _, w := range p.wrong {
			if w == p.want {
				t.Errorf("%s %s: the pin lists its own value as wrong", p.item, p.field)
			}
		}
	}
	t.Logf("%d edition and source-error pins", len(editionPins))
}

// Names and shapes that must not appear: first-edition cards and boards,
// misspellings and aliases, a third Babylon night stage.
func TestEditionAbsences(t *testing.T) {
	rows := editionRows(t)
	names := map[string]bool{}
	for id := DesignID(1); id <= NumDesigns; id++ {
		d, _ := DesignByID(id)
		names[d.Name] = true
	}
	for _, name := range []string{"Pawnshop", "Strategists Guild", "Cave", "Palisade", "Bazaar", "Caravanserai", "Apothocary", "Univeristy", "Craftsmen Guild"} {
		if names[name] {
			t.Errorf("design named %s", name)
		}
	}
	for _, name := range []string{"Well", "Decorators Guild", "Mine", "Stockade", "Bazar", "Caravansery", "Apothecary", "University", "Craftsmens Guild"} {
		if !names[name] {
			t.Errorf("no design named %s", name)
		}
	}
	for _, slug := range []string{"giza", "ephesus", "halicarnassus", "rhodes"} {
		if _, ok := rows[slug]; ok {
			t.Errorf("board %s", slug)
		}
	}
	if _, ok := rows["babylon/night/3"]; ok {
		t.Error("a third Babylon night stage (first edition)")
	}
	for item, row := range rows {
		if strings.HasPrefix(item, "olympia/") && strings.Contains(row["effects"], "trade:") {
			t.Errorf("%s: a trade discount on Olympia (first edition)", item)
		}
	}
}

// Copies at a given player count, against the wiki's per-player-count pages
// (ruling 8): one Haven at 3 players, one Vineyard at 4 and at 5, two at 6,
// three Caravanseries at 6, and at 5 players one copy of each Age I grey
// design and two of each Age II grey design.
func TestCopiesAtPlayerCounts(t *testing.T) {
	cases := []struct {
		slug    string
		players uint8
		want    uint8
	}{
		{"haven", 3, 1}, {"vineyard", 4, 1}, {"vineyard", 5, 1}, {"vineyard", 6, 2}, {"caravansery", 6, 3},
		{"glassworks-age-1", 5, 1}, {"press-age-1", 5, 1}, {"loom-age-1", 5, 1},
		{"glassworks-age-2", 5, 2}, {"press-age-2", 5, 2}, {"loom-age-2", 5, 2},
		{"well", 3, 0}, {"theater", 3, 1}, {"castrum", 3, 0}, {"castrum", 4, 1}, {"tavern", 7, 3},
	}
	for _, c := range cases {
		id, ok := DesignBySlug(c.slug)
		if !ok {
			t.Fatalf("no design %s", c.slug)
		}
		if got := CopiesAt(id, c.players); got != c.want {
			t.Errorf("%s at %d players: %s copies, want %d", c.slug, c.players, strconv.Itoa(int(got)), c.want)
		}
	}
}
