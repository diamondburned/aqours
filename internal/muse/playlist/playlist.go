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

	pl.SetUnsaved()

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

// Remove removes the tracks with the given indices. The function guarantees
// that the delete will never touch tracks that didn't have the given indices
// before removal; it does this by sorting the internal array of ixs.
func (pl *Playlist) Remove(ixs ...int) {
	if len(ixs) == 0 {
		return
	}

	pl.SetUnsaved()

	// Sort indices from largest to smallest so we could pop the last track off
	// first to preserve order.
	sort.Sort(sort.Reverse(sort.IntSlice(ixs)))

	for _, ix := range ixs {
		// https://github.com/golang/go/wiki/SliceTricks
		copy(pl.Tracks[ix:], pl.Tracks[ix+1:])   // shift backwards
		pl.Tracks[len(pl.Tracks)-1] = nil        // nil last
		pl.Tracks = pl.Tracks[:len(pl.Tracks)-1] // omit last
	}
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
