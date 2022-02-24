package state

import (
	"sync"

	"github.com/diamondburned/aqours/internal/muse/playlist"
)

type metadataMap map[string]*metadata

func (mm metadataMap) ref(path string) {
	md, ok := mm[path]
	if !ok {
		return
	}
	md.reference++
}

func (mm metadataMap) unref(s *State, path string) {
	md, ok := mm[path]
	if !ok {
		return
	}
	md.reference--
	if md.reference == 0 {
		delete(mm, path)
		s.intern.unsaved = true
	}
}

// metadata is a metadata state value that is shared across tracks.
type metadata struct {
	// DON'T COPY!!
	_ [0]sync.Mutex

	playlist.Track
	reference int32
}

func newMetadata(t playlist.Track) *metadata {
	t.Filepath = ""

	return &metadata{
		Track: t,
	}
}

// Track is a track value that keeps track of a Metadata pointer and a filepath.
type Track struct {
	// DON'T COPY!!
	_ [0]sync.Mutex

	Filepath string
	playlist *Playlist
}

// UpdateMetadata updates the track's metadata in the global metadata store. If
// the metadata does not yet exist, it will create a new one and automatically
// reference it. Else, no references are taken.
func (t *Track) UpdateMetadata(i playlist.Track) {
	md, ok := t.playlist.state.metadata[t.Filepath]
	if !ok {
		md = newMetadata(i)
		md.reference = 1
		t.playlist.state.metadata[t.Filepath] = md
	}

	i.Filepath = ""
	md.Track = i

	// Mark as unsaved.
	t.playlist.state.MarkChanged()
}

// Metadata returns a copy of the current track's metadata with the filepath
// filled in. If the metadata is not found, then a placeholder one is returned.
func (t *Track) Metadata() (track playlist.Track) {
	if md, ok := t.playlist.state.metadata[t.Filepath]; ok {
		track = md.Track
	} else {
		track.Title = playlist.TitleFromPath(t.Filepath)
	}

	track.Filepath = t.Filepath

	return
}
