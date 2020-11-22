package playlist

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"sync"
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
	// DO NOT COPY!!! The state relies on the pointers being the same.
	_ [0]sync.Mutex

	Name   string
	Path   string
	Tracks []*Track

	unsaved bool
}

func (pl *Playlist) SetUnsaved() {
	if !pl.unsaved {
		pl.unsaved = true
	}
}

func (pl *Playlist) IsUnsaved() bool {
	return pl.unsaved
}

// Add adds the given path to a track to the current playlist. It marks the
// playlist as unsaved. If before is false, then the track is appended after the
// index. If before is true, then the track is appended before the index. The
// returned integers are the positions of the inserted tracks. If len(paths) is
// 0, then ix is returned for both.
func (pl *Playlist) Add(ix int, before bool, paths ...string) (start, end int) {
	if len(paths) == 0 {
		return ix, ix
	}

	pl.unsaved = true

	if !before {
		ix++
	}

	// https://github.com/golang/go/wiki/SliceTricks
	pl.Tracks = append(pl.Tracks, make([]*Track, len(paths))...)
	copy(pl.Tracks[ix+len(paths):], pl.Tracks[ix:])

	for i, path := range paths {
		pl.Tracks[ix+i] = &Track{
			Title:    filepath.Base(path),
			Filepath: path,
		}
	}

	return ix, ix + len(paths)
}

// Save saves the playlist. The function might be called in another goroutine.
func (pl *Playlist) Save(done func(error)) {
	if !pl.unsaved {
		done(nil)
		return
	}

	fn, ok := playlistWriters[filepath.Ext(pl.Path)]
	if !ok {
		done(fmt.Errorf("unknown format for path %q", pl.Path))
		return
	}

	if err := fn(pl, done); err != nil {
		// Try and fix the playlist if possible.
		var fixable FixableError
		if errors.As(err, &fixable) {
			fixable.Fix(pl)
			pl.Save(done) // resave
		}
	}
}
