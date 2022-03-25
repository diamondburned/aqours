package tracks

import (
	"sort"

	"github.com/diamondburned/aqours/internal/state"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// noopSort does nothing.
type noopSort struct{}

func (noopSort) Len() int           { return 0 }
func (noopSort) Less(i, j int) bool { return false }
func (noopSort) Swap(i, j int)      {}

type trackSorter struct {
	store  *gtk.ListStore
	rows   map[*state.Track]*TrackRow
	tracks []*state.Track
	iters  []*gtk.TreeIter
}

type trackMetadata struct {
	number int
	album  string
}

func newTrackSorter(list *TrackList, start, end int) sort.Interface {
	tracks := list.Playlist.Tracks[start:end]
	iters := make([]*gtk.TreeIter, len(tracks))

	for i, track := range tracks {
		it, ok := list.TrackRows[track].Iter()
		if !ok {
			return noopSort{}
		}
		iters[i] = it
	}

	return trackSorter{
		store:  list.Store,
		rows:   list.TrackRows,
		tracks: tracks,
		iters:  iters,
	}
}

func (sorter trackSorter) Len() int {
	return len(sorter.tracks)
}

func (sorter trackSorter) Less(i, j int) bool {
	ixA, albumA := sorter.metadata(i)
	ixB, albumB := sorter.metadata(j)

	return (albumA == albumB) && (ixA < ixB)
}

func (sorter trackSorter) Swap(i, j int) {
	sorter.store.Swap(sorter.iters[i], sorter.iters[j])
	sorter.tracks[i], sorter.tracks[j] = sorter.tracks[j], sorter.tracks[i]
}

func (sorter trackSorter) metadata(i int) (trackN int, album string) {
	metadata := sorter.tracks[i].Metadata()
	return metadata.Number, metadata.Album
}
