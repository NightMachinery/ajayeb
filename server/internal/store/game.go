package store

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
)

// Game selects only the bounded schema/game header, without game decoding or
// replay. Manager uses it to choose the matching registered room kind.
func (st *Store) Game(code string) (string, error) {
	path, err := st.path(code)
	if err != nil {
		return "", err
	}
	unlock := st.lockRoom(code)
	defer unlock()
	data, err := readCapped(path)
	if errors.Is(err, fs.ErrNotExist) {
		data, err = readCapped(filepath.Join(st.ArchiveDir(), code+".json"))
	}
	if err != nil {
		return "", err
	}
	schema, game, err := fileKind(data)
	if err != nil {
		return "", err
	}
	if schema == Schema && game == "duel" || schema == 2 && st.codecs.schema2[game] != nil {
		return game, nil
	}
	return "", fmt.Errorf("%w: game %q in schema %d", ErrGame, game, schema)
}
