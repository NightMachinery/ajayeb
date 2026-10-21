package main

import "github.com/gzuidhof/tygo/tygo"

// A separate pass is necessary: tygo resolves a package to its first config.
// Restrict both passes by filename so existing Duel output stays byte-identical.
func generatePlatform(out string) error {
	return tygo.New(&tygo.Config{Packages: []*tygo.PackageConfig{{
		Path:         "github.com/NightMachinery/ajayeb/server/internal/wire",
		OutputPath:   out,
		IncludeFiles: []string{"platform.go", "table.go", "lobby.go"},
		Indent:       "  ", FallbackType: "never", EnumStyle: "union",
		Frontmatter: "export type PlatformClientMessage = TablePick | TableUndo | TableSeat | TableRole | TableResign | LobbyCreate | LobbyEdit | LobbyRole | LobbyStart | PlatformLink | PlatformRelink;\nexport type PlatformServerMessage = TableState | TableReject | LobbyState | LobbyReject | PlatformLinked;",
	}}}).Generate()
}
