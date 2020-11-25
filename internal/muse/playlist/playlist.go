package playlist

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
)

type (
	PlaylistReader func(path string) (*Playlist, error)
	PlaylistWriter func(pl *Playlist, done func(error)) error
)

var (
	playlistReaders = map[string]PlaylistReader{}
	playlistWriters = map[string]PlaylistWriter{}
)

func SupportedExtensions() []string {
	var exts = make([]string, 0, len(playlistReaders))
	for ext := range playlistReaders {
		exts = append(exts, ext)
	}
	sort.Strings(exts)
	return exts
}

// FixableError is returned from PlaylistWriter if the error can be fixed
// automatically.
type FixableError interface {
	error
	Fix(playlist *Playlist)
}

func Register(fileExt string, r PlaylistReader, w PlaylistWriter) {
	playlistReaders[fileExt] = r
	playlistWriters[fileExt] = w
}

func ParseFile(path string) (*Playlist, error) {
	fn, ok := playlistReaders[filepath.Ext(path)]
	if !ok {
		return nil, fmt.Errorf("unknown format for path %q", path)
	}

	return fn(path)
}

type Playlist struct {
	Name   string
	Path   string
	Tracks []Track
}

// Save saves the playlist. The function must not be called in another
// goroutine. The done callback may be called in a goroutine.
func (pl *Playlist) Save(done func(error)) {
	fn, ok := playlistWriters[filepath.Ext(pl.Path)]
	if !ok {
		done(fmt.Errorf("unknown format for path %q", pl.Path))
		return
	}

	pl.save(fn, done)
}

func (pl *Playlist) save(wfn PlaylistWriter, done func(error)) {
	if err := wfn(pl, done); err != nil {
		// Try and fix the playlist if possible.
		var fixable FixableError
		if errors.As(err, &fixable) {
			fixable.Fix(pl)
			pl.save(wfn, done) // resave
		}
	}
}
