package rules

import (
	"strconv"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
)

// AppendKey appends Key's canonical bytes to caller-owned storage. A valid
// action fits MaxKeyBytes; an invalid action appends nothing. It does not
// retain dst. Search can reuse a bounded buffer without allocating strings.
func (a Action) AppendKey(dst []byte) []byte {
	if !a.Valid() {
		return dst
	}
	dst = append(dst, actionVerbs[a.Kind]...)
	dst = append(dst, ':')
	if a.Kind == ActSide {
		return append(dst, a.Side.String()...)
	}
	design, _ := catalog.DesignByID(a.Design)
	dst = append(dst, design.Slug...)
	if a.Kind == ActBuild || a.Kind == ActStage {
		dst = append(dst, ':')
		dst = strconv.AppendUint(dst, uint64(a.Left), 10)
		dst = append(dst, ':')
		dst = strconv.AppendUint(dst, uint64(a.Right), 10)
	}
	return dst
}
