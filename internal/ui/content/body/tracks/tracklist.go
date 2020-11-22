package tracks

import (
	"log"
	"net/url"
	"strings"

	"github.com/diamondburned/aqours/internal/muse/playlist"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

type TrackPath = string

type TrackList struct {
	gtk.ScrolledWindow
	Tree   *gtk.TreeView
	Store  *gtk.ListStore
	Select *gtk.TreeSelection

	Playlist  *playlist.Playlist
	TrackRows map[*playlist.Track]*TrackRow

	playing *playlist.Track
}

var trackListDragTargets = []gtk.TargetEntry{
	targetEntry("text/uri-list", gtk.TARGET_OTHER_APP, 1),
}

func targetEntry(target string, f gtk.TargetFlags, info uint) gtk.TargetEntry {
	e, _ := gtk.TargetEntryNew(target, f, info)
	return *e
}

func NewTrackList(parent ParentController, pl *playlist.Playlist) *TrackList {
	store, _ := gtk.ListStoreNew(
		glib.TYPE_STRING, // columnTitle
		glib.TYPE_STRING, // columnArtist
		glib.TYPE_STRING, // columnAlbum
		glib.TYPE_STRING, // columnTime
		glib.TYPE_INT,    // columnSelected - pango.Weight
	)

	tree, _ := gtk.TreeViewNewWithModel(store)
	tree.SetActivateOnSingleClick(false)
	tree.AppendColumn(newColumn("Title", columnTitle))
	tree.AppendColumn(newColumn("Artist", columnArtist))
	tree.AppendColumn(newColumn("Album", columnAlbum))
	tree.AppendColumn(newColumn("", columnTime))
	tree.AppendColumn(newColumn("", columnSelected))
	tree.Show()

	// tree.SetReorderable(true)
	// tree.Connect("")

	s, _ := tree.GetSelection()
	s.SetMode(gtk.SELECTION_MULTIPLE)

	scroll, _ := gtk.ScrolledWindowNew(nil, nil)
	scroll.SetPolicy(gtk.POLICY_NEVER, gtk.POLICY_AUTOMATIC)
	scroll.SetVExpand(true)
	scroll.Add(tree)
	scroll.Show()

	list := TrackList{
		ScrolledWindow: *scroll,

		Tree:   tree,
		Store:  store,
		Select: s,

		Playlist:  pl,
		TrackRows: make(map[*playlist.Track]*TrackRow, len(pl.Tracks)),
	}

	tree.Connect("row-activated", func(_ *gtk.TextView, path *gtk.TreePath) {
		parent.PlayTrack(pl, path.GetIndices()[0])
	})

	// TODO: Implement TreeView drag-and-drop for reordering. Effectively, the
	// application should use the standardized file URI as the data for
	// reordering, which should make it work with a range of applications.
	//
	// To deal with the lack of data when we move the tracks out and in the
	// list, we could have a remove track and add path functions. The add
	// function would simply treat the track as an unprocessed one.

	tree.EnableModelDragDest(trackListDragTargets, gdk.ACTION_LINK)
	tree.Connect("drag-data-received",
		func(_ gtk.IWidget, ctx *gdk.DragContext, x, y int, data *gtk.SelectionData) {
			path, pos, ok := tree.GetDestRowAtPos(x, y)
			if !ok {
				log.Println("No path found at dragged pos.")
				return
			}

			// Get the files in form of line-delimited URIs
			var uris = strings.Fields(string(data.GetData()))

			// Create a path slice that we decode URIs into.
			var paths = uris[:0]

			// Decode the URIs.
			for _, uri := range uris {
				u, err := url.Parse(uri)
				if err != nil {
					log.Printf("Failed parsing URI %q: %v\n", uri, err)
					continue
				}
				if u.Scheme != "file" {
					log.Println("Unknown file scheme (only locals):", u.Scheme)
					continue
				}
				paths = append(paths, u.Path)
			}

			if len(paths) == 0 {
				return
			}

			// log.Println("Dragged into", list.Tracks[path.GetIndices()[0]].Filepath)
			// log.Printf("%q\n", paths)

			ix := path.GetIndices()[0]
			before := false ||
				pos == gtk.TREE_VIEW_DROP_BEFORE ||
				pos == gtk.TREE_VIEW_DROP_INTO_OR_BEFORE

			start, end := list.Playlist.Add(ix, before, paths...)
			probeQueue := make([]probeJob, 0, end-start)

			for i := start; i < end; i++ {
				row := &TrackRow{
					Iter: list.Store.Insert(i),
					Bold: false,
				}
				track := list.Playlist.Tracks[i]

				row.setListStore(track, list.Store)
				list.TrackRows[track] = row

				probeQueue = append(probeQueue, newProbeJob(&list, track, row))
			}

			parent.UpdateTracks(list.Playlist)
			queueProbeJobs(probeQueue...)
		},
	)

	// Create a nil slice and leave the allocation for later.
	var probeQueue []probeJob

	for _, track := range pl.Tracks {
		row := &TrackRow{
			Iter: list.Store.Append(),
			Bold: false,
		}

		row.setListStore(track, list.Store)
		list.TrackRows[track] = row

		if !track.IsProbed() {
			// Allocate a slice only if we have at least 1.
			if probeQueue == nil {
				probeQueue = make([]probeJob, 0, len(pl.Tracks))
			}
			probeQueue = append(probeQueue, newProbeJob(&list, track, row))
		}
	}

	// TODO: this has a cache stampede problem. We need to have a context to
	// cancel this.
	if len(probeQueue) > 0 {
		queueProbeJobs(probeQueue...)
	}

	return &list
}

func (list *TrackList) SetPlaying(playing *playlist.Track) {
	rw, ok := list.TrackRows[playing]
	if !ok {
		log.Printf("Track not found on (*Tracklist).SetPlaying: %q\n", playing.Filepath)
		return
	}

	if list.playing != nil {
		playingRow := list.TrackRows[list.playing]
		playingRow.SetBold(list.Store, false)

		// Decide if we should move the selection.
		reselect := true &&
			list.Select.CountSelectedRows() == 1 &&
			list.Select.IterIsSelected(playingRow.Iter)

		if reselect {
			list.Select.UnselectIter(playingRow.Iter)
			list.Select.SelectIter(rw.Iter)

			path, _ := list.Store.GetPath(rw.Iter)
			list.Tree.ScrollToCell(path, nil, false, 0, 0)
		}
	}

	list.playing = playing
	rw.SetBold(list.Store, true)
}
