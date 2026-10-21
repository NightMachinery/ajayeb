package main

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/NightMachinery/ajayeb/server/internal/store"
)

// checkRooms exits before all identity, directory, signal and listener setup.
// 1 is an operational/schema error; 3 is live-room loss requiring explicit consent.
func checkRooms(dataDir string, stdout, stderr io.Writer) int {
	kinds, err := roomKinds()
	if err != nil {
		fmt.Fprintf(stderr, "ajayeb: room kinds: %v\n", err)
		return 1
	}
	report := store.CheckDataDirWithCodecs(dataDir, kinds.Codecs())
	if err := json.NewEncoder(stdout).Encode(report); err != nil {
		fmt.Fprintf(stderr, "ajayeb: writing room check report: %v\n", err)
		return 1
	}
	if report.Errors > 0 {
		return 1
	}
	if report.LiveWouldArchive > 0 {
		return 3
	}
	return 0
}
