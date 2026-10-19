package store

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
)

// CheckReport is the read-only candidate preflight contract. Errors always block
// deployment; only LiveWouldArchive may be explicitly overridden by the operator.
type CheckReport struct {
	Format            string `json:"format"`
	Rules             int    `json:"rules"`
	Rooms             int    `json:"rooms"`
	Lobby             int    `json:"lobby"`
	Playing           int    `json:"playing"`
	Finished          int    `json:"finished"`
	Archived          int    `json:"archived"`
	Playable          int    `json:"playable"`
	WouldArchive      int    `json:"wouldArchive"`
	LiveWouldArchive  int    `json:"liveWouldArchive"`
	RefreshableHashes int    `json:"refreshableHashes"`
	SchemaErrors      int    `json:"schemaErrors"`
	Errors            int    `json:"errors"`
}

// CheckDataDir reads without initializing a Store, identity or listener. Missing
// room directories are empty; every other read/format failure counts as an
// error. No file is refreshed, archived, created or removed.
func CheckDataDir(dataDir string) CheckReport {
	r := CheckReport{Format: "ajayeb-room-check/1", Rules: engine.RulesVersion}
	info, err := os.Stat(dataDir)
	if errors.Is(err, fs.ErrNotExist) {
		return r
	}
	if err != nil || !info.IsDir() {
		r.Errors++
		return r
	}
	for _, folder := range []string{"rooms", "rooms-archive"} {
		r.checkFolder(filepath.Join(dataDir, folder), folder == "rooms-archive")
	}
	return r
}

func (r *CheckReport) checkFolder(dir string, archive bool) {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return
	}
	if err != nil {
		r.Errors++
		return
	}
	for _, entry := range entries {
		code, ok := strings.CutSuffix(entry.Name(), ".json")
		if !ok {
			continue
		}
		// Match Codes(): unrelated JSON files and directories are not rooms.
		if entry.IsDir() || ValidateCode(code) != nil {
			continue
		}
		r.Rooms++
		f, err := readFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			r.Errors++
			continue
		}
		if checkSchema(f) != nil {
			r.SchemaErrors++
			r.Errors++
			continue
		}
		if f.Code != code {
			r.Errors++
			continue
		}
		if archive && f.Status != StatusArchived {
			r.Errors++
			continue
		}
		switch f.Status {
		case StatusArchived:
			r.Archived++
			continue
		case StatusLobby:
			r.Lobby++
		case StatusPlaying:
			r.Playing++
		case StatusFinished:
			r.Finished++
		default:
			r.Errors++
			continue
		}
		s, _, err := assess(f)
		if err != nil {
			r.WouldArchive++
			if f.Status == StatusLobby || f.Status == StatusPlaying {
				r.LiveWouldArchive++
			}
			continue
		}
		r.Playable++
		if engine.HashString(&s) != f.Hash {
			r.RefreshableHashes++
		}
	}
}
