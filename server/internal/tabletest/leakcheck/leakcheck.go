// Package leakcheck is the hand-game leak oracle for Haftgane's seat views,
// events, history and logs (docs/haftgane/platform-architecture.md,
// "Capacity and acceptance"). It is independent of the code it checks: it
// judges every document against ground truth (a Truth, built from the
// setup, the deal and the batches, never from a view) and against its own
// whitelist of the view's fields, and it fails closed on anything it does
// not know.
//
// Each rule is its own detector:
//
//   - shuffle-id: a card value that is not a bare catalog slug, or any string
//     holding a slug with a copy or position suffix (amber#2, amber.17);
//   - deal-order: a card list that is not in catalog order, as a hand left
//     in deal order would be;
//   - raw-key: an action key anywhere but the viewer's own offer, in a view,
//     an event, the history or a log line;
//   - hidden-card: a card the viewer may not know there: another seat's
//     hand, kept, tucked or sold card, a leftover, the pile or the box
//     before the end reveal, or any card in a log line;
//   - hash: the seed, a game fingerprint, an observation or pending hash, or
//     any hex token of twelve digits or more;
//   - foreign-offer: another seat's offer, salvage offer or private section;
//   - order: an offer not in the engine's canonical order, or a history out
//     of point order;
//   - unknown-field: a field outside the whitelist;
//   - envelope: a table frame's private field (pending pick, seat,
//     acknowledgement) sent to a viewer it does not belong to, or differing
//     from the viewer's accepted state; the table's tests report it, since
//     the envelope is the platform's, not the engine view's;
//   - mismatch: a public value that differs from the truth (not a leak, but
//     a view the checker cannot vouch for).
//
// Separate whitelists and truth builders cover the real engine and the
// isolated fake test double. The actor's tests pass each frame's engine view as
// a View document and the whole frame as a Frame document, which gets the
// field-independent detectors (raw-key, hash, shuffle-id) with the frame's
// own key, text and hex paths allowed in Config. They check each frame's
// envelope against its viewer first, and report what fails as envelope.
package leakcheck

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/rules"
)

// Rule names a detector.
type Rule string

const (
	ShuffleID    Rule = "shuffle-id"
	DealOrder    Rule = "deal-order"
	RawKey       Rule = "raw-key"
	HiddenCard   Rule = "hidden-card"
	Hash         Rule = "hash"
	ForeignOffer Rule = "foreign-offer"
	Order        Rule = "order"
	UnknownField Rule = "unknown-field"
	Envelope     Rule = "envelope"
	Mismatch     Rule = "mismatch"
)

// Leak is one finding.
type Leak struct {
	Rule   Rule
	Doc    string
	Path   string
	Detail string
}

func (l Leak) String() string {
	return fmt.Sprintf("%s: %s at %s: %s", l.Rule, l.Doc, l.Path, l.Detail)
}

// Kind is what a document is.
type Kind int

const (
	View    Kind = iota // one viewer's engine view
	History             // one viewer's history entries
	Frame               // a whole wire frame: field-independent detectors only
	Log                 // captured log text
)

// Doc is one captured document.
type Doc struct {
	Label  string
	Kind   Kind
	Viewer int // the viewer's seat, or -1 for the spectator
	Moment int // the number of batches resolved when it was sent (unused for a Frame or a Log)
	// Span is, for a View, the number of resolutions whose events it
	// carries, the latest last: more than one in a frame an outbox merged
	// for a slow connection. Zero means one.
	Span int
	Data []byte
}

// Spectator is the Viewer of a spectator's documents.
const Spectator = -1

// Config allows paths (as Path prints them, such as "you.choices[].key")
// that the field-independent detectors would otherwise flag.
type Config struct {
	KeyPaths  []string // paths that may hold the viewer's own offered keys
	TextPaths []string // free text, such as player names: no shuffle-id check
	HexPaths  []string // hex identifiers that are not secrets, such as user ids
}

// DefaultConfig is the fake view's: only the viewer's own choices hold keys.
var DefaultConfig = Config{KeyPaths: []string{"you.choices[].key"}}

// Secret is a value no document may contain.
type Secret struct {
	Name  string
	Value string
}

// Moment is the ground truth after some number of resolved points. Card
// lists are catalog slugs in catalog order; per-seat slices are by seat.
type Moment struct {
	Point    string // the open point's logical id, "" once over
	Kind     string
	Required []int
	Phase    string
	Age      int
	Turn     int
	Over     bool
	Reveal   bool // over, with the end reveal on
	Decider  int  // the seat of a salvage point, or -1
	Coins    []int
	Stages   []int
	Hands    [][]string
	Kept     []string
	Tucked   [][]string
	Built    [][]string
	Pile     []string
	Box      []string
	Legal    [][]string // each seat's offered keys, in canonical order
}

// Action is one seat's part of a resolved point.
type Action struct {
	Kind   string
	Design string
	Left   int
	Right  int
}

// Event is one unfiltered engine event.
type Event struct {
	Other   int
	Turn    int
	Side    string
	Printed int
	Delta   int
	Source  string
	Outcome string
	Kind    string
	Seat    int
	Play    string
	Design  string
	Stage   int
	Left    int
	Right   int
	Age     int
	Pass    string
}

// Resolution is one resolved point: who decided, what each did, the hand
// each held when it opened, and the unfiltered events.
type Resolution struct {
	Point    string
	Kind     string
	Required []int
	Held     [][]string // by seat; empty for seats the point did not require
	Actions  []Action   // by seat; zero for seats the point did not require
	Events   []Event
}

// Truth is a game's ground truth: Moments[k] is the state after the first
// k Resolutions.
type Truth struct {
	RealStatistics   [][]rules.Statistics
	realNeighbours   [][][]realNeighbour
	realScience      [][]realScience
	realDisabled     [][][]realDisabled
	realKept         [][][]string // per moment, seat and offered choice
	RealObservations [][]rules.Observation
	RealChoices      [][][]rules.Choice
	RealPublicFacts  [][]rules.PublicFacts
	Real             bool
	Initial          []Event
	RealGames        []rules.Game
	Seats            int
	Slugs            []string // the catalog, in order
	Secrets          []Secret
	Moments          []Moment
	Resolutions      []Resolution
}

// Checker runs every detector.
type stringFacts struct {
	key, hex          bool
	secrets, suffixes []string
}

type Checker struct {
	strings      map[string]stringFacts
	secretNames  map[string]string
	suffixCache  map[string]bool
	Truth        *Truth
	Config       Config
	index        map[string]int
	otherSecrets []Secret
}

// New returns a checker for a truth.
func New(t *Truth, cfg Config) *Checker {
	c := &Checker{Truth: t, Config: cfg, index: map[string]int{}}
	c.secretNames = map[string]string{}
	for _, s := range t.Secrets {
		c.secretNames[s.Value] = s.Name
		if !hexRE.MatchString(s.Value) && !decimalRE.MatchString(s.Value) {
			c.otherSecrets = append(c.otherSecrets, s)
		}
	}
	c.suffixCache = map[string]bool{}
	c.strings = map[string]stringFacts{}

	for i, s := range t.Slugs {
		c.index[s] = i
	}
	return c
}

// Check checks every document and returns every leak, in document order.
func (c *Checker) Check(docs ...Doc) []Leak {
	var out []Leak
	for _, d := range docs {
		out = append(out, c.check(d)...)
	}
	return out
}

func (c *Checker) check(d Doc) []Leak {
	r := &report{doc: d.Label}
	if d.Kind == Log {
		c.checkLog(r, string(d.Data))
		return r.leaks
	}
	dec := json.NewDecoder(bytes.NewReader(d.Data))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		r.add(UnknownField, "", "not JSON: "+err.Error())
		return r.leaks
	}
	keyPaths := c.Config.KeyPaths
	if d.Kind == History {
		keyPaths = nil
	}
	c.walk(r, v, "", keyPaths)
	if d.Kind == Frame {
		return r.leaks
	}
	if d.Moment < 0 || d.Moment >= len(c.Truth.Moments) || d.Viewer < Spectator || d.Viewer >= c.Truth.Seats {
		r.add(Mismatch, "", fmt.Sprintf("moment %d, viewer %d is not in the game", d.Moment, d.Viewer))
		return r.leaks
	}
	switch d.Kind {
	case View:
		c.checkView(r, d)
	case History:
		c.checkHistory(r, d)
	}
	return r.leaks
}

type report struct {
	doc   string
	leaks []Leak
}

func (r *report) add(rule Rule, path, detail string) {
	r.leaks = append(r.leaks, Leak{Rule: rule, Doc: r.doc, Path: path, Detail: detail})
}

// --- field-independent detectors ------------------------------------------

var (
	keyRE     = regexp.MustCompile(`\b(build-discarded|build|stage|side|tuck|sell|salvage):[a-z]`)
	hexRE     = regexp.MustCompile(`[0-9a-f]{12,}`)
	decimalRE = regexp.MustCompile(`[0-9]{6,}`)
)

// walk visits every string, number and field name.
func (c *Checker) walk(r *report, v any, path string, keyPaths []string) {
	switch x := v.(type) {
	case map[string]any:
		names := make([]string, 0, len(x))
		for k := range x {
			names = append(names, k)
		}
		slices.Sort(names)
		for _, k := range names {
			p := k
			if path != "" {
				p = path + "." + k
			}
			c.scanString(r, k, p+" (field name)", nil)
			c.walk(r, x[k], p, keyPaths)
		}
	case []any:
		for _, e := range x {
			c.walk(r, e, path+"[]", keyPaths)
		}
	case string:
		c.scanString(r, x, path, keyPaths)
	case json.Number:
		if name, ok := c.secretNames[string(x)]; ok {
			r.add(Hash, path, "a number equal to the "+name)
		}
	}
}

func (c *Checker) scanString(r *report, s, path string, keyPaths []string) {
	f, ok := c.strings[s]
	if !ok {
		f.key = strings.Contains(s, ":") && keyRE.MatchString(s)
		f.hex = hexRE.MatchString(s)
		secrets := c.otherSecrets
		if f.hex || decimalRE.MatchString(s) {
			secrets = c.Truth.Secrets
		}
		for _, sec := range secrets {
			if s == sec.Value || (len(sec.Value) >= 6 && strings.Contains(s, sec.Value)) {
				f.secrets = append(f.secrets, sec.Name)
			}
		}
		for _, tok := range strings.FieldsFunc(s, func(ch rune) bool { return ch == ':' || ch == ' ' || ch == ',' }) {
			if c.suffixed(tok) {
				f.suffixes = append(f.suffixes, tok)
			}
		}
		c.strings[s] = f
	}
	if f.key && !slices.Contains(keyPaths, path) {
		r.add(RawKey, path, fmt.Sprintf("%q", s))
	}
	for _, name := range f.secrets {
		r.add(Hash, path, "the "+name)
	}
	if f.hex && !slices.Contains(c.Config.HexPaths, path) {
		r.add(Hash, path, fmt.Sprintf("a hex token in %q", s))
	}
	if !slices.Contains(c.Config.TextPaths, path) {
		for _, tok := range f.suffixes {
			r.add(ShuffleID, path, fmt.Sprintf("%q names a card with a suffix", tok))
		}
	}
}

// suffixed reports whether tok is a slug followed by a copy or position
// suffix, rather than a slug or another word.
func (c *Checker) suffixed(tok string) (out bool) {
	if v, ok := c.suffixCache[tok]; ok {
		return v
	}
	defer func() { c.suffixCache[tok] = out }()
	if _, ok := c.index[tok]; ok {
		return false
	}
	for _, slug := range c.Truth.Slugs {
		if rest, ok := strings.CutPrefix(tok, slug); ok && rest != "" && strings.ContainsRune("#._-0123456789/[(", rune(rest[0])) {
			return true
		}
	}
	return false
}

// checkLog: a log line may name no key, no card and no secret.
func (c *Checker) checkLog(r *report, text string) {
	for i, line := range strings.Split(text, "\n") {
		path := fmt.Sprintf("line %d", i+1)
		c.scanString(r, line, path, nil)
		for _, w := range strings.FieldsFunc(line, func(ch rune) bool { return (ch < 'a' || ch > 'z') && ch != '-' }) {
			if _, ok := c.index[w]; ok {
				r.add(HiddenCard, path, fmt.Sprintf("the card %q", w))
			}
		}
	}
}
