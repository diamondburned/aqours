package tracks

import (
	"sort"

	"github.com/diamondburned/aqours/internal/state"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

type trackSorter struct {
	store  *gtk.ListStore
	rows   map[*state.Track]*TrackRow
	tracks []*state.Track
}

type trackMetadata struct {
	number int
	album  string
}

func newTrackSorter(list *TrackList, start, end int) sort.Interface {
	return trackSorter{
		store:  list.Store,
		rows:   list.TrackRows,
		tracks: list.Playlist.Tracks[start:end],
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
	rowA := sorter.rows[sorter.tracks[i]]
	rowB := sorter.rows[sorter.tracks[j]]
	sorter.store.Swap(rowA.Iter, rowB.Iter)

	// This should work, since it shares the same backing array.
	sorter.tracks[i], sorter.tracks[j] = sorter.tracks[j], sorter.tracks[i]
}

func (sorter trackSorter) metadata(i int) (trackN int, album string) {
	metadata := sorter.tracks[i].Metadata()
	return metadata.Number, metadata.Album
}
