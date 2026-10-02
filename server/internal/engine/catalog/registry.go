package catalog

import (
	"fmt"
	"iter"
	"strconv"
	"strings"
)

// The registry holds every registered card, wonder and token in tables
// indexed by id. The data files fill it while the package initializes, and
// after that it is read-only, so any number of goroutines may read it.
//
// How a data file registers its items: declare the items in a package-level
// slice and register them from a package-level variable initializer, not from
// an init function:
//
//	var ageOneCards = []CardDef{
//		{ID: 1, Slug: "lumber-yard", Name: "Lumber Yard", Color: Brown, Effects: Fx(Produce(Wood, 1))},
//	}
//
//	var _ = registerCards(1, ageOneCards)
//
// Go initializes every package-level variable before it runs any init
// function, while init functions of different files run in file-name order.
// Registering from initializers therefore guarantees that every init function
// (compiled.go's derived tables, for instance) sees the complete catalog.
// A derived table that must follow ReplaceForTest registers its builder with
// afterReplace from its init function.
//
// Registration validates each item and panics on the first problem: an id out
// of range or taken twice, a slug that is not kebab-case, starts with "the-"
// or is already used by any card, wonder or token, a missing name, colour or
// age, a guild outside Age III, a wonder or token slug that does not match its
// id constant, or an invalid effect (Effect.Validate). A panic during package
// initialization fails every test of every package that imports the catalog,
// so a data mistake cannot go unnoticed.

type registry struct {
	cards   [NumCards + 1]CardDef // a zero ID marks a free entry
	wonders [NumWonders + 1]WonderDef
	tokens  [NumTokens + 1]TokenDef
}

var (
	reg          = new(registry)
	replaceHooks []func()
)

// registerCards adds the buildings of one age (1 to 3; guilds are age 3) to
// the catalog. It sets each card's Age and derives its Back. It returns true
// so that it can be called from a package-level variable initializer.
func registerCards(age uint8, defs []CardDef) bool {
	for i := range defs {
		if err := reg.addCard(age, defs[i]); err != nil {
			panic(err)
		}
	}
	return true
}

// registerWonders adds wonders to the catalog; see registerCards.
func registerWonders(defs []WonderDef) bool {
	for i := range defs {
		if err := reg.addWonder(defs[i]); err != nil {
			panic(err)
		}
	}
	return true
}

// registerTokens adds progress tokens to the catalog; see registerCards.
func registerTokens(defs []TokenDef) bool {
	for i := range defs {
		if err := reg.addToken(defs[i]); err != nil {
			panic(err)
		}
	}
	return true
}

// afterReplace registers f to run every time ReplaceForTest swaps the catalog
// in or out, so that tables derived from the catalog stay consistent with it.
// Call it from an init function, after building the table once.
func afterReplace(f func()) { replaceHooks = append(replaceHooks, f) }

// ReplaceForTest swaps in a catalog made of exactly the given items, until the
// returned restore function runs; pass restore to t.Cleanup. Cards must have
// their Age set (1 to 3); everything else is validated as for the real data,
// and an invalid item panics.
//
// It exists for tests that need a few real items while the full data is
// absent (or need a small, controlled catalog). It is not safe while other
// goroutines read the catalog, so never use it from a parallel test, and never
// call it outside tests.
func ReplaceForTest(cards []CardDef, wonders []WonderDef, tokens []TokenDef) (restore func()) {
	r := new(registry)
	for i := range cards {
		if err := r.addCard(cards[i].Age, cards[i]); err != nil {
			panic(err)
		}
	}
	for i := range wonders {
		if err := r.addWonder(wonders[i]); err != nil {
			panic(err)
		}
	}
	for i := range tokens {
		if err := r.addToken(tokens[i]); err != nil {
			panic(err)
		}
	}
	old := reg
	reg = r
	runReplaceHooks()
	return func() {
		reg = old
		runReplaceHooks()
	}
}

func runReplaceHooks() {
	for _, f := range replaceHooks {
		f()
	}
}

func (r *registry) addCard(age uint8, d CardDef) error {
	if d.ID == 0 || int(d.ID) > NumCards {
		return fmt.Errorf("catalog: card id %d (%q) is outside 1..%d", d.ID, d.Slug, NumCards)
	}
	where := "catalog: card " + strconv.Itoa(int(d.ID)) + " (" + strconv.Quote(d.Slug) + ")"
	if r.cards[d.ID].ID != 0 {
		return fmt.Errorf("%s: id already registered as %q", where, r.cards[d.ID].Slug)
	}
	if err := r.checkNewSlug(d.Slug); err != nil {
		return fmt.Errorf("%s: %w", where, err)
	}
	if d.Name == "" {
		return fmt.Errorf("%s: missing name", where)
	}
	if age < 1 || age > 3 {
		return fmt.Errorf("%s: age %d is outside 1..3", where, age)
	}
	if d.Age != 0 && d.Age != age {
		return fmt.Errorf("%s: Age %d disagrees with the registration age %d", where, d.Age, age)
	}
	if d.Color == NoColor || d.Color >= NumColors {
		return fmt.Errorf("%s: invalid colour %d", where, d.Color)
	}
	back := AgeBack(age)
	if d.Color == Purple {
		if age != 3 {
			return fmt.Errorf("%s: a guild must be registered as age 3", where)
		}
		back = GuildBack
	}
	if d.Back != NoBack && d.Back != back {
		return fmt.Errorf("%s: Back %s disagrees with the derived back %s", where, d.Back, back)
	}
	if d.ChainIn >= NumChains || d.ChainOut >= NumChains {
		return fmt.Errorf("%s: invalid chain symbol", where)
	}
	if d.ChainIn != NoChain && d.ChainIn == d.ChainOut {
		return fmt.Errorf("%s: a card cannot chain into itself (%s)", where, d.ChainIn)
	}
	for i, e := range d.Effects {
		if err := e.Validate(); err != nil {
			return fmt.Errorf("%s: effect %d: %w", where, i, err)
		}
	}
	d.Age = age
	d.Back = back
	d.AltNames = append([]string(nil), d.AltNames...)
	d.Effects = append([]Effect(nil), d.Effects...)
	r.cards[d.ID] = d
	return nil
}

func (r *registry) addWonder(d WonderDef) error {
	if d.ID == 0 || int(d.ID) > NumWonders {
		return fmt.Errorf("catalog: wonder id %d (%q) is outside 1..%d", d.ID, d.Slug, NumWonders)
	}
	where := "catalog: wonder " + strconv.Itoa(int(d.ID)) + " (" + strconv.Quote(d.Slug) + ")"
	if r.wonders[d.ID].ID != 0 {
		return fmt.Errorf("%s: id already registered as %q", where, r.wonders[d.ID].Slug)
	}
	if d.Slug != wonderSlugs[d.ID] {
		return fmt.Errorf("%s: wonder id %d is %q in the canonical order", where, d.ID, wonderSlugs[d.ID])
	}
	if err := r.checkNewSlug(d.Slug); err != nil {
		return fmt.Errorf("%s: %w", where, err)
	}
	if d.Name == "" {
		return fmt.Errorf("%s: missing name", where)
	}
	for i, e := range d.Effects {
		if err := e.Validate(); err != nil {
			return fmt.Errorf("%s: effect %d: %w", where, i, err)
		}
	}
	d.Effects = append([]Effect(nil), d.Effects...)
	r.wonders[d.ID] = d
	return nil
}

func (r *registry) addToken(d TokenDef) error {
	if d.ID == 0 || int(d.ID) > NumTokens {
		return fmt.Errorf("catalog: token id %d (%q) is outside 1..%d", d.ID, d.Slug, NumTokens)
	}
	where := "catalog: token " + strconv.Itoa(int(d.ID)) + " (" + strconv.Quote(d.Slug) + ")"
	if r.tokens[d.ID].ID != 0 {
		return fmt.Errorf("%s: id already registered as %q", where, r.tokens[d.ID].Slug)
	}
	if d.Slug != tokenSlugs[d.ID] {
		return fmt.Errorf("%s: token id %d is %q in the canonical order", where, d.ID, tokenSlugs[d.ID])
	}
	if err := r.checkNewSlug(d.Slug); err != nil {
		return fmt.Errorf("%s: %w", where, err)
	}
	if d.Name == "" {
		return fmt.Errorf("%s: missing name", where)
	}
	for i, e := range d.Effects {
		if err := e.Validate(); err != nil {
			return fmt.Errorf("%s: effect %d: %w", where, i, err)
		}
	}
	d.Effects = append([]Effect(nil), d.Effects...)
	r.tokens[d.ID] = d
	return nil
}

// checkNewSlug rejects a malformed slug and one already used by any item.
// Slugs are unique across cards, wonders and tokens, so a slug alone names
// one item; the redaction tests rely on that when they scan views for hidden
// slugs.
func (r *registry) checkNewSlug(slug string) error {
	if err := checkSlug(slug); err != nil {
		return err
	}
	if _, ok := r.cardBySlug(slug); ok {
		return fmt.Errorf("slug %q is already used by a card", slug)
	}
	if _, ok := r.wonderBySlug(slug); ok {
		return fmt.Errorf("slug %q is already used by a wonder", slug)
	}
	if _, ok := r.tokenBySlug(slug); ok {
		return fmt.Errorf("slug %q is already used by a token", slug)
	}
	return nil
}

// maxSlugLen bounds slugs; the longest real one is about 20 bytes.
const maxSlugLen = 32

// checkSlug accepts kebab-case: lower-case ASCII letters and digits in words
// joined by single hyphens, without a leading "the-".
func checkSlug(slug string) error {
	if slug == "" || len(slug) > maxSlugLen {
		return fmt.Errorf("slug %q must be 1 to %d bytes", slug, maxSlugLen)
	}
	if strings.HasPrefix(slug, "the-") {
		return fmt.Errorf("slug %q must not start with \"the-\"", slug)
	}
	prevHyphen := true // a slug may not start with a hyphen
	for i := 0; i < len(slug); i++ {
		switch ch := slug[i]; {
		case ch >= 'a' && ch <= 'z', ch >= '0' && ch <= '9':
			prevHyphen = false
		case ch == '-' && !prevHyphen:
			prevHyphen = true
		default:
			return fmt.Errorf("slug %q is not kebab-case", slug)
		}
	}
	if prevHyphen {
		return fmt.Errorf("slug %q must not end with a hyphen", slug)
	}
	return nil
}

func (r *registry) cardBySlug(slug string) (CardID, bool) {
	for i := 1; i < len(r.cards); i++ {
		if r.cards[i].ID != 0 && r.cards[i].Slug == slug {
			return CardID(i), true
		}
	}
	return 0, false
}

func (r *registry) wonderBySlug(slug string) (WonderID, bool) {
	for i := 1; i < len(r.wonders); i++ {
		if r.wonders[i].ID != 0 && r.wonders[i].Slug == slug {
			return WonderID(i), true
		}
	}
	return 0, false
}

func (r *registry) tokenBySlug(slug string) (TokenID, bool) {
	for i := 1; i < len(r.tokens); i++ {
		if r.tokens[i].ID != 0 && r.tokens[i].Slug == slug {
			return TokenID(i), true
		}
	}
	return 0, false
}

// Card returns the registered building with this id, or nil. The result
// points into the read-only catalog: never modify it.
func Card(id CardID) *CardDef {
	r := reg
	if int(id) < len(r.cards) && r.cards[id].ID != 0 {
		return &r.cards[id]
	}
	return nil
}

// Wonder returns the registered wonder with this id, or nil. Never modify the
// result.
func Wonder(id WonderID) *WonderDef {
	r := reg
	if int(id) < len(r.wonders) && r.wonders[id].ID != 0 {
		return &r.wonders[id]
	}
	return nil
}

// Token returns the registered progress token with this id, or nil. Never
// modify the result.
func Token(id TokenID) *TokenDef {
	r := reg
	if int(id) < len(r.tokens) && r.tokens[id].ID != 0 {
		return &r.tokens[id]
	}
	return nil
}

// CardBySlug returns the id of the registered building with exactly this
// slug. Matching is exact: no case folding, trimming or "the-" handling.
func CardBySlug(slug string) (CardID, bool) { return reg.cardBySlug(slug) }

// WonderBySlug returns the id of the registered wonder with exactly this slug.
func WonderBySlug(slug string) (WonderID, bool) { return reg.wonderBySlug(slug) }

// TokenBySlug returns the id of the registered token with exactly this slug.
func TokenBySlug(slug string) (TokenID, bool) { return reg.tokenBySlug(slug) }

// Cards yields every registered building in ascending id order.
func Cards() iter.Seq[*CardDef] {
	r := reg
	return func(yield func(*CardDef) bool) {
		for i := 1; i < len(r.cards); i++ {
			if r.cards[i].ID != 0 && !yield(&r.cards[i]) {
				return
			}
		}
	}
}

// Wonders yields every registered wonder in ascending id order.
func Wonders() iter.Seq[*WonderDef] {
	r := reg
	return func(yield func(*WonderDef) bool) {
		for i := 1; i < len(r.wonders); i++ {
			if r.wonders[i].ID != 0 && !yield(&r.wonders[i]) {
				return
			}
		}
	}
}

// Tokens yields every registered token in ascending id order.
func Tokens() iter.Seq[*TokenDef] {
	r := reg
	return func(yield func(*TokenDef) bool) {
		for i := 1; i < len(r.tokens); i++ {
			if r.tokens[i].ID != 0 && !yield(&r.tokens[i]) {
				return
			}
		}
	}
}

// String returns the card's slug, or "card#N" for an id that is not
// registered ("unknown" for the Unknown sentinel).
func (id CardID) String() string {
	if d := Card(id); d != nil {
		return d.Slug
	}
	if id == Unknown {
		return "unknown"
	}
	return "card#" + strconv.Itoa(int(id))
}

// MarshalText encodes the card as its slug. Ids that are not registered,
// including 0 and Unknown, are an error.
func (id CardID) MarshalText() ([]byte, error) {
	if d := Card(id); d != nil {
		return []byte(d.Slug), nil
	}
	return nil, fmt.Errorf("catalog: card id %d is not registered", id)
}

// UnmarshalText decodes a card slug, exactly as CardBySlug matches it.
func (id *CardID) UnmarshalText(b []byte) error {
	v, ok := CardBySlug(string(b))
	if !ok {
		return fmt.Errorf("catalog: unknown card slug %q", truncate(b))
	}
	*id = v
	return nil
}

// String returns the wonder's slug, or "wonder#N" for an id that is not
// registered ("unknown" for the Unknown sentinel).
func (id WonderID) String() string {
	if d := Wonder(id); d != nil {
		return d.Slug
	}
	if id == Unknown {
		return "unknown"
	}
	return "wonder#" + strconv.Itoa(int(id))
}

// MarshalText encodes the wonder as its slug. Ids that are not registered
// are an error.
func (id WonderID) MarshalText() ([]byte, error) {
	if d := Wonder(id); d != nil {
		return []byte(d.Slug), nil
	}
	return nil, fmt.Errorf("catalog: wonder id %d is not registered", id)
}

// UnmarshalText decodes a wonder slug, exactly as WonderBySlug matches it.
func (id *WonderID) UnmarshalText(b []byte) error {
	v, ok := WonderBySlug(string(b))
	if !ok {
		return fmt.Errorf("catalog: unknown wonder slug %q", truncate(b))
	}
	*id = v
	return nil
}

// String returns the token's slug, or "token#N" for an id that is not
// registered ("unknown" for the Unknown sentinel).
func (id TokenID) String() string {
	if d := Token(id); d != nil {
		return d.Slug
	}
	if id == Unknown {
		return "unknown"
	}
	return "token#" + strconv.Itoa(int(id))
}

// MarshalText encodes the token as its slug. Ids that are not registered are
// an error.
func (id TokenID) MarshalText() ([]byte, error) {
	if d := Token(id); d != nil {
		return []byte(d.Slug), nil
	}
	return nil, fmt.Errorf("catalog: token id %d is not registered", id)
}

// UnmarshalText decodes a token slug, exactly as TokenBySlug matches it.
func (id *TokenID) UnmarshalText(b []byte) error {
	v, ok := TokenBySlug(string(b))
	if !ok {
		return fmt.Errorf("catalog: unknown token slug %q", truncate(b))
	}
	*id = v
	return nil
}

// truncate bounds untrusted input quoted in an error message.
func truncate(b []byte) string {
	if len(b) > 2*maxSlugLen {
		return string(b[:2*maxSlugLen]) + "..."
	}
	return string(b)
}
