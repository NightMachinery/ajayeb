package main

import "github.com/gzuidhof/tygo/tygo"

// A separate pass is necessary: tygo resolves a package to its first config.
// Restrict both passes by filename so existing Duel output stays byte-identical.
func generatePlatform(out string) error {
	return tygo.New(&tygo.Config{TypeMappings: map[string]string{
		"view.HintSettings": "view.HintSettings",
		"hview.Seat":        "haftgane.Seat", "hview.Play": "haftgane.Play",
		"hview.Private": "haftgane.Private", "hview.Choice": "haftgane.Choice",
		"hview.OfferEntry": "haftgane.OfferEntry", "hview.SeenEntry": "haftgane.SeenEntry",
		"hview.Result": "haftgane.Result", "hview.Score": "haftgane.Score", "hview.Reveal": "haftgane.Reveal",
	}, Packages: []*tygo.PackageConfig{{
		Path:         "github.com/NightMachinery/ajayeb/server/internal/wire",
		OutputPath:   out,
		IncludeFiles: []string{"platform.go", "table.go", "lobby.go"},
		Indent:       "  ", FallbackType: "never", EnumStyle: "union",
		Frontmatter: "import type * as view from \"./view.gen\";\nimport type * as haftgane from \"./haftgane.gen\";\nexport type PlatformClientMessage = TablePick | TableUndo | TableSeat | TableRole | TableResign | LobbyCreate | LobbyEdit | LobbyRole | LobbyStart | PlatformLink | PlatformRelink;\nexport type PlatformServerMessage = TableState | TableReject | LobbyState | LobbyReject | PlatformLinked;",
	}}}).Generate()
}
