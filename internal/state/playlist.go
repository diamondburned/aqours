package state

import (
	"sort"
	"sync/atomic"

	"github.com/diamondburned/aqours/internal/muse/playlist"
)

type PlaylistName = string

type Playlist struct {
	Name   PlaylistName
	Path   string
	Tracks []*Track

	state   *State
	unsaved uint32 // atomic
}

func convertPlaylist(state *State, orig *playlist.Playlist) *Playlist {
	playlist := &Playlist{
		Name:    orig.Name,
		Path:    orig.Path,
		Tracks:  make([]*Track, len(orig.Tracks)),
		state:   state,
		unsaved: 0, // fresh state
	}

	for i, track := range orig.Tracks {
		playlist.Tracks[i] = &Track{
			Filepath: track.Filepath,
			playlist: playlist,
		}

		// Reincrement reference.
		md, ok := state.metadata[track.Filepath]
		if !ok {
			md = newMetadata(track)
			state.metadata[track.Filepath] = md
			state.intern.unsaved = true
		}

		md.reference++
	}

	return playlist
}

// SetUnsaved marks the playlist as unsaved.
func (pl *Playlist) SetUnsaved() {
	atomic.StoreUint32(&pl.unsaved, 1)
}

// IsUnsaved returns true if the playlist is unsaved. It is thread-safe.
func (pl *Playlist) IsUnsaved() bool {
	return atomic.LoadUint32(&pl.unsaved) == 1
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
			Filepath: path,
			playlist: pl,
		}

		pl.state.metadata.ref(path)
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
		track := pl.Tracks[ix]
		pl.state.metadata.unref(pl.state, track.Filepath)

		// https://github.com/golang/go/wiki/SliceTricks
		copy(pl.Tracks[ix:], pl.Tracks[ix+1:])   // shift backwards
		pl.Tracks[len(pl.Tracks)-1] = nil        // nil last
		pl.Tracks = pl.Tracks[:len(pl.Tracks)-1] // omit last
	}
}

// Save saves the playlist. The function must not be called in another
// goroutine. The done callback may be called in a goroutine.
//
// TODO: refactor this to Save() and WaitUntilSaved().
func (pl *Playlist) Save(done func(error)) {
	if !pl.IsUnsaved() {
		done(nil)
		return
	}

	playlistCopy := playlist.Playlist{
		Name:   pl.Name,
		Path:   pl.Path,
		Tracks: make([]playlist.Track, len(pl.Tracks)),
	}

	for i, track := range pl.Tracks {
		playlistCopy.Tracks[i] = track.Metadata()
	}

	go playlistCopy.Save(func(err error) {
		if err == nil {
			atomic.StoreUint32(&pl.unsaved, 0)
		}
		done(err)
	})
}
