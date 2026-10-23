// Package view contains seed-free Haftgane wire DTOs. Fields are built by
// whitelist in rules; no DTO embeds an engine value or a private record.
package view

// Event contains only the documented fields of its kind. Design is empty
// in the public form of a private play, leftover or kept-card event.
type Event struct {
	Kind    string `json:"kind"`
	Seat    int    `json:"seat"`
	Other   int    `json:"other"`
	Age     int    `json:"age"`
	Turn    int    `json:"turn"`
	Play    string `json:"play"`
	Design  string `json:"design"`
	Stage   int    `json:"stage"`
	Side    string `json:"side"`
	Left    int    `json:"left"`
	Right   int    `json:"right"`
	Printed int    `json:"printed"`
	Delta   int    `json:"delta"`
	Source  string `json:"source"`
	Outcome string `json:"outcome"`
}

type HistoryEntry struct {
	Point  string  `json:"point"`
	Events []Event `json:"events"`
}
